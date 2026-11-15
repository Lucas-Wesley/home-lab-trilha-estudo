package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Propostas (DECISOES.md §5): tudo o que a IA produz espera o Lucas. Ele escreve um pedido antes
// e comenta depois (geral, numa seção ou num trecho); "Pedir ajuste" gera a versão seguinte com a
// resposta da IA a cada comentário; "Aprovar" grava o resultado. Na fase 3, o tipo é a lição.

type comentario struct {
	ID         int64  `json:"id"`
	PropostaID int64  `json:"proposta_id"`
	Ancora     string `json:"ancora"`
	Citacao    string `json:"citacao"`
	Autor      string `json:"autor"`
	Texto      string `json:"texto"`
	RespondeA  int64  `json:"responde_a,omitempty"`
	CriadoEm   string `json:"criado_em"`
}

type proposta struct {
	ID           int64           `json:"id"`
	TrilhaID     int64           `json:"trilha_id"`
	EtapaChave   string          `json:"etapa_chave"`
	Tipo         string          `json:"tipo"`
	Versao       int             `json:"versao"`
	AnteriorID   int64           `json:"anterior_id,omitempty"`
	Estado       string          `json:"estado"`
	Pedido       string          `json:"pedido"`
	Conteudo     json.RawMessage `json:"conteudo,omitempty"`
	Erro         string          `json:"erro"`
	CriadoEm     string          `json:"criado_em"`
	AtualizadoEm string          `json:"atualizado_em"`
	Comentarios  []comentario    `json:"comentarios"`
	Tarefa       *tarefa         `json:"tarefa,omitempty"`
}

// conteudoLicao é o que a IA devolve para uma lição (e o que fica guardado na proposta).
type conteudoLicao struct {
	Licao     string               `json:"licao"`
	Notas     string               `json:"notas"`
	Respostas []respostaComentario `json:"respostas"`
	Links     []linkConferido      `json:"links,omitempty"` // conferidos pelo servidor (risco 3)
}

type respostaComentario struct {
	Comentario int64  `json:"comentario"`
	Resposta   string `json:"resposta"`
}

const esquemaLicao = `{"type":"object","properties":{
"licao":{"type":"string","description":"a lição inteira, em Markdown"},
"notas":{"type":"string","description":"o que o Lucas precisa saber sobre esta versão: decisões, dúvidas, o que ficou de fora"},
"respostas":{"type":"array","items":{"type":"object","properties":{
  "comentario":{"type":"integer"},"resposta":{"type":"string"}},"required":["comentario","resposta"]}}},
"required":["licao","notas","respostas"]}`

const (
	propGerando     = "gerando"
	propAguardando  = "aguardando"
	propAprovada    = "aprovada"
	propDescartada  = "descartada"
	propSubstituida = "substituida"
	propFalhou      = "falhou"
)

const colunasProposta = `id, trilha_id, etapa_chave, tipo, versao, COALESCE(anterior_id, 0), estado, pedido, conteudo, erro, criado_em, atualizado_em`

func lerPropostaLinha(row interface{ Scan(...any) error }) (proposta, error) {
	var p proposta
	var conteudo string
	err := row.Scan(&p.ID, &p.TrilhaID, &p.EtapaChave, &p.Tipo, &p.Versao, &p.AnteriorID, &p.Estado, &p.Pedido,
		&conteudo, &p.Erro, &p.CriadoEm, &p.AtualizadoEm)
	if conteudo != "" {
		p.Conteudo = json.RawMessage(conteudo)
	}
	return p, err
}

func (a *app) propostaPorID(ctx context.Context, id int64) (proposta, error) {
	p, err := lerPropostaLinha(a.db.QueryRowContext(ctx, `SELECT `+colunasProposta+` FROM propostas WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, errNaoEncontrou
	}
	return p, err
}

// completar junta os comentários e a tarefa da proposta.
func (a *app) completar(ctx context.Context, p *proposta) error {
	rows, err := a.db.QueryContext(ctx, `SELECT id, proposta_id, ancora, citacao, autor, texto, COALESCE(responde_a, 0), criado_em
		FROM comentarios WHERE proposta_id = ? ORDER BY id`, p.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	p.Comentarios = []comentario{}
	for rows.Next() {
		var c comentario
		if err := rows.Scan(&c.ID, &c.PropostaID, &c.Ancora, &c.Citacao, &c.Autor, &c.Texto, &c.RespondeA, &c.CriadoEm); err != nil {
			return err
		}
		p.Comentarios = append(p.Comentarios, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.proposta_id = ? ORDER BY t.id DESC LIMIT 1`, p.ID))
	if err == nil {
		p.Tarefa = t
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

// ---------- pedir uma lição ----------

func (a *app) handleGerarLicao(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Pedido string `json:"pedido"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	p, err := a.novaProposta(r.Context(), e.TrilhaID, e.Chave, "licao", strings.TrimSpace(corpo.Pedido), 0, "Lição: "+e.Titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

var errPropostaAberta = erroDeEntrada{errors.New("já há uma proposta sendo gerada para esta etapa; espere terminar ou cancele a tarefa")}

// novaProposta cria a versão seguinte (estado "gerando") e põe a tarefa na fila.
func (a *app) novaProposta(ctx context.Context, trilhaID int64, chave, tipo, pedido string, anterior int64, titulo string) (proposta, error) {
	// O banco tem uma conexão só: o que for lido fora da transação, antes dela.
	modelo := a.modeloPara(ctx, tipo)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return proposta{}, err
	}
	defer tx.Rollback()
	var gerando, versao int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE estado = ?), COALESCE(MAX(versao), 0)
		FROM propostas WHERE trilha_id = ? AND etapa_chave = ? AND tipo = ?`, propGerando, trilhaID, chave, tipo).Scan(&gerando, &versao); err != nil {
		return proposta{}, err
	}
	if gerando > 0 {
		return proposta{}, errPropostaAberta
	}
	var anteriorID any
	if anterior > 0 {
		anteriorID = anterior
	}
	momento := agora()
	res, err := tx.ExecContext(ctx, `INSERT INTO propostas (trilha_id, etapa_chave, tipo, versao, anterior_id, estado, pedido, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, trilhaID, chave, tipo, versao+1, anteriorID, propGerando, pedido, momento, momento)
	if err != nil {
		return proposta{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return proposta{}, err
	}
	if _, err := a.fila.enfileirar(ctx, tx, tipo, id, fmt.Sprintf("%s (v%d)", titulo, versao+1), modelo); err != nil {
		return proposta{}, err
	}
	if err := tx.Commit(); err != nil {
		return proposta{}, err
	}
	a.fila.acordar()
	p, err := a.propostaPorID(ctx, id)
	if err != nil {
		return p, err
	}
	return p, a.completar(ctx, &p)
}

func (a *app) handlePropostasDaEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	tipo := r.URL.Query().Get("tipo")
	if tipo == "" {
		tipo = "licao"
	}
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+colunasProposta+` FROM propostas
		WHERE trilha_id = ? AND etapa_chave = ? AND tipo = ? ORDER BY versao`, e.TrilhaID, e.Chave, tipo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	lista := []proposta{}
	for rows.Next() {
		p, err := lerPropostaLinha(rows)
		if err != nil {
			rows.Close()
			a.responderErro(w, err)
			return
		}
		lista = append(lista, p)
	}
	rows.Close()
	for i := range lista {
		if err := a.completar(r.Context(), &lista[i]); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	responderJSON(w, http.StatusOK, lista)
}

// ---------- comentários ----------

func (a *app) responderProposta(w http.ResponseWriter, r *http.Request, id int64, status int) {
	p, err := a.propostaPorID(r.Context(), id)
	if err == nil {
		err = a.completar(r.Context(), &p)
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, status, p)
}

func (a *app) handleComentar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var c comentario
	if err := lerCorpo(r, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	c.Texto, c.Ancora, c.Citacao = strings.TrimSpace(c.Texto), strings.TrimSpace(c.Ancora), strings.TrimSpace(c.Citacao)
	if c.Texto == "" {
		a.responderErro(w, erroDeEntrada{errors.New("o comentário está vazio")})
		return
	}
	if len([]rune(c.Citacao)) > 2000 {
		a.responderErro(w, erroDeEntrada{errors.New("o trecho citado é longo demais (até 2000 caracteres)")})
		return
	}
	p, err := a.propostaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if p.Estado != propAguardando {
		a.responderErro(w, erroDeEntrada{errors.New("só dá para comentar uma proposta que está aguardando")})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `INSERT INTO comentarios (proposta_id, ancora, citacao, autor, texto, criado_em)
		VALUES (?, ?, ?, 'lucas', ?, ?)`, id, c.Ancora, c.Citacao, c.Texto, agora()); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, id, http.StatusCreated)
}

func (a *app) handleExcluirComentario(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var propostaID int64
	var autor, estado string
	err = a.db.QueryRowContext(r.Context(), `SELECT c.proposta_id, c.autor, p.estado FROM comentarios c
		JOIN propostas p ON p.id = c.proposta_id WHERE c.id = ?`, id).Scan(&propostaID, &autor, &estado)
	if errors.Is(err, sql.ErrNoRows) {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if autor != "lucas" || estado != propAguardando {
		a.responderErro(w, erroDeEntrada{errors.New("só dá para tirar um comentário seu numa proposta que está aguardando")})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM comentarios WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, propostaID, http.StatusOK)
}

// ---------- decidir ----------

func (a *app) propostaAguardando(r *http.Request) (proposta, error) {
	id, err := idDaRota(r)
	if err != nil {
		return proposta{}, err
	}
	p, err := a.propostaPorID(r.Context(), id)
	if err != nil {
		return p, err
	}
	if p.Estado != propAguardando {
		return p, erroDeEntrada{fmt.Errorf("a proposta v%d não está aguardando (está %s)", p.Versao, p.Estado)}
	}
	return p, nil
}

func (a *app) handleAjustar(w http.ResponseWriter, r *http.Request) {
	p, err := a.propostaAguardando(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Pedido string `json:"pedido"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	corpo.Pedido = strings.TrimSpace(corpo.Pedido)
	var comentarios int
	if err := a.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM comentarios WHERE proposta_id = ? AND autor = 'lucas'`, p.ID).Scan(&comentarios); err != nil {
		a.responderErro(w, err)
		return
	}
	if comentarios == 0 && corpo.Pedido == "" {
		a.responderErro(w, erroDeEntrada{errors.New("diga o que ajustar: comente a proposta ou escreva um pedido")})
		return
	}
	titulo := "Ajuste da lição"
	if ehCriacao(p.Tipo) {
		titulo = titulosCriacao[p.Tipo] + ": " + a.tituloDaTrilha(r.Context(), p.TrilhaID)
	} else if e, err := a.etapaPorChave(r.Context(), p.TrilhaID, p.EtapaChave); err == nil {
		titulo = titulosEstudo[p.Tipo] + ": " + e.Titulo
	}
	nova, err := a.novaProposta(r.Context(), p.TrilhaID, p.EtapaChave, p.Tipo, corpo.Pedido, p.ID, titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, nova)
}

func (a *app) handleAprovar(w http.ResponseWriter, r *http.Request) {
	p, err := a.propostaAguardando(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if ehCriacao(p.Tipo) {
		a.aprovarCriacao(w, r, p)
		return
	}
	if ehEstudo(p.Tipo) {
		a.aprovarEstudo(w, r, p)
		return
	}
	if p.Tipo == tipoMateriais {
		a.aprovarMateriais(w, r, p)
		return
	}
	if p.Tipo == tipoReforco {
		a.aprovarReforco(w, r, p)
		return
	}
	if err := a.semVersaoGerando(r.Context(), p); err != nil {
		a.responderErro(w, err)
		return
	}
	var c conteudoLicao
	if err := json.Unmarshal(p.Conteudo, &c); err != nil || strings.TrimSpace(c.Licao) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("a proposta não tem lição para gravar")})
		return
	}
	slug := a.slugDaTrilha(r.Context(), p.TrilhaID)
	if _, err := a.etapaPorChave(r.Context(), p.TrilhaID, p.EtapaChave); err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa desta proposta não existe mais na trilha")})
		return
	}
	if err := a.conteudo.gravar(fmt.Sprintf("%s: lição %s (proposta v%d aprovada)", slug, p.EtapaChave, p.Versao),
		map[string]*string{caminhoLicao(slug, p.EtapaChave): &c.Licao}); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(r.Context(), p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}

func (a *app) handleDescartar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	p, err := a.propostaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if p.Estado != propAguardando && p.Estado != propFalhou {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("a proposta v%d não pode ser descartada (está %s)", p.Versao, p.Estado)})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE propostas SET estado = ?, atualizado_em = ? WHERE id = ?`, propDescartada, agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, id, http.StatusOK)
}

// ---------- a tarefa: gerar a lição ----------

func (a *app) etapaPorChave(ctx context.Context, trilhaID int64, chave string) (etapa, error) {
	var id int64
	err := a.db.QueryRowContext(ctx, `SELECT id FROM etapas WHERE trilha_id = ? AND chave = ?`, trilhaID, chave).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return etapa{}, errNaoEncontrou
	}
	if err != nil {
		return etapa{}, err
	}
	return a.etapaPorID(ctx, id)
}

// aoTerminarTarefa: tarefa que falhou ou foi cancelada leva a proposta junto (a versão anterior,
// se houver, continua aguardando).
func (a *app) aoTerminarTarefa(t *tarefa) {
	if t.Estado != falhou && t.Estado != cancelada {
		return
	}
	if t.Tipo == tipoNivelamento {
		a.aoTerminarNivelamento(t)
		return
	}
	if t.PropostaID == 0 {
		// A correção do simulado: a tentativa fica com o erro (e pode ser feita de novo).
		if _, err := a.db.Exec(`UPDATE tentativas SET estado = ?, erro = ?, atualizado_em = ? WHERE tarefa_id = ? AND estado = ?`,
			tentFalhou, t.Erro, agora(), t.ID, tentCorrigindo); err != nil {
			log.Printf("tentativa da tarefa %d: %v", t.ID, err)
		}
		return
	}
	if _, err := a.db.Exec(`UPDATE propostas SET estado = ?, erro = ?, atualizado_em = ? WHERE id = ? AND estado = ?`,
		propFalhou, t.Erro, agora(), t.PropostaID, propGerando); err != nil {
		log.Printf("proposta %d: %v", t.PropostaID, err)
	}
}

func (a *app) gerarLicao(ctx context.Context, t *tarefa) error {
	p, err := a.propostaPorID(ctx, t.PropostaID)
	if err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, p.TrilhaID)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)

	arquivos, comentarios, err := a.montarPedidoLicao(ctx, tr, p)
	if err != nil {
		return err
	}
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	var extras []string
	for _, sub := range []string{"documentos", "licoes"} {
		if caminho, err := a.conteudo.caminho("trilhas/" + tr.Slug + "/" + sub); err == nil {
			if _, err := os.Stat(caminho); err == nil {
				extras = append(extras, caminho)
			}
		}
	}
	rg, err := a.ia.lerRegra("licao.md")
	if err != nil {
		return err
	}
	instrucao := "Escreva a lição de uma etapa de uma trilha de estudo. Tudo o que você precisa saber está em " +
		"`pedido.md`, nesta pasta: leia-o inteiro primeiro, e depois os arquivos que ele indicar."
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: rg.Texto, instrucao: instrucao,
		ferramentas: "Read,Glob,Grep,WebSearch,WebFetch", esquema: esquemaLicao, modelo: t.Modelo,
	})
	if err != nil {
		return err
	}
	var c conteudoLicao
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	c.Licao = strings.TrimSpace(c.Licao)
	if c.Licao == "" {
		return errors.New("o Claude devolveu uma lição vazia")
	}
	if a.ia.conferirLinks != nil {
		c.Links = a.ia.conferirLinks(ctx, c.Licao)
	}
	bruto, err := json.Marshal(c)
	if err != nil {
		return err
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	momento := agora()
	if _, err := tx.ExecContext(ctx, `UPDATE propostas SET estado = ?, conteudo = ?, atualizado_em = ? WHERE id = ? AND estado = ?`,
		propAguardando, string(bruto), momento, p.ID, propGerando); err != nil {
		return err
	}
	if p.AnteriorID > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE propostas SET estado = ?, atualizado_em = ? WHERE id = ? AND estado = ?`,
			propSubstituida, momento, p.AnteriorID, propAguardando); err != nil {
			return err
		}
	}
	// A resposta da IA a cada comentário fica na versão nova, apontando para o comentário.
	for _, rc := range c.Respostas {
		if !comentarios[rc.Comentario] || strings.TrimSpace(rc.Resposta) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO comentarios (proposta_id, autor, texto, responde_a, criado_em)
			VALUES (?, 'ia', ?, ?, ?)`, p.ID, strings.TrimSpace(rc.Resposta), rc.Comentario, momento); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var nomesBloom = map[string]string{
	"lembrar": "lembrar", "entender": "entender", "aplicar": "aplicar",
	"analisar": "analisar", "avaliar": "avaliar", "criar": "criar",
}

var nomesProfundidade = map[int]string{
	1: "1. Panorama: o que é, para que serve, onde se encaixa",
	2: "2. Essencial: conceitos principais e exemplos; o necessário para usar e conversar sobre o assunto",
	3: "3. Sólido: fundamentos, principal, ao redor, aplicação e teste (o molde completo de camadas)",
	4: "4. Aprofundado: o sólido + casos-limite, trade-offs, erros comuns, alternativas comparadas, fontes primárias",
	5: "5. Deep dive: o aprofundado + origens e evolução, a base completa embaixo, especificações e debates em aberto",
}

// montarPedidoLicao escreve o pedido.md (e a lição atual ou a versão anterior, quando houver).
// Devolve também os comentários que a IA deve responder.
func (a *app) montarPedidoLicao(ctx context.Context, tr trilha, p proposta) (map[string]string, map[int64]bool, error) {
	var e etapa
	var modulo string
	var anterior, proxima *etapa
	var todas []etapa
	for _, m := range tr.Modulos {
		for i := range m.Etapas {
			todas = append(todas, m.Etapas[i])
			if m.Etapas[i].Chave == p.EtapaChave {
				e, modulo = m.Etapas[i], m.Titulo
			}
		}
	}
	if e.ID == 0 {
		return nil, nil, errors.New("a etapa desta proposta não existe mais na trilha")
	}
	porID := map[int64]etapa{}
	for i, x := range todas {
		porID[x.ID] = x
		if x.ID == e.ID {
			if i > 0 {
				anterior = &todas[i-1]
			}
			if i+1 < len(todas) {
				proxima = &todas[i+1]
			}
		}
	}
	profundidade := e.Profundidade
	if profundidade == 0 {
		profundidade = tr.Profundidade
	}
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, err
	}

	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: lição da etapa \"%s\"\n\n", e.Titulo)

	escreva("## A trilha\n\n- Título: %s\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Recorte", tr.Recorte}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
		if strings.TrimSpace(campo[1]) != "" {
			escreva("- %s: %s\n", campo[0], campo[1])
		}
	}
	escreva("\n## A etapa\n\n- Título: %s\n- Módulo: %s\n", e.Titulo, modulo)
	if e.Objetivo != "" {
		escreva("- Objetivo: %s\n", e.Objetivo)
	}
	if b, ok := nomesBloom[e.Bloom]; ok {
		escreva("- Nível de Bloom do objetivo: %s\n", b)
	}
	escreva("- Profundidade: %s\n", nomesProfundidade[profundidade])
	if e.Tipo == "base" {
		escreva("- Tipo: etapa de base (fundamento de que outras etapas dependem)\n")
	}
	if e.Tipo == "reforco" {
		escreva("- Tipo: etapa de **reforço** (o Lucas errou estes pontos nos exercícios, no simulado ou na revisão; explique de outro jeito, com outros exemplos, e não repita a lição da etapa reforçada)\n")
	}
	if e.Foco != "" {
		escreva("- Foco pedido para esta lição: %s\n", e.Foco)
	}
	if e.Minutos > 0 {
		escreva("- Tempo previsto para a etapa: %d min (leitura da lição + material principal + teste)\n", e.Minutos)
	}
	if e.Criterio != "" {
		escreva("- Critério de conclusão: %s\n", e.Criterio)
	}

	escreva("\n## Onde ela está na trilha\n\n")
	if len(e.Requer) > 0 {
		escreva("Pré-requisitos (já estudados antes desta; não reexplique, só retome em uma frase se precisar):\n\n")
		for _, rid := range e.Requer {
			if r, ok := porID[rid]; ok {
				escreva("- %s%s%s\n", r.Titulo, sufixo(" — ", r.Objetivo), licaoEm(r))
			}
		}
		escreva("\n")
	}
	if anterior != nil {
		escreva("Etapa anterior: %s%s%s\n\n", anterior.Titulo, sufixo(" — ", anterior.Objetivo), licaoEm(*anterior))
	}
	if proxima != nil {
		escreva("Etapa seguinte (o que ela cobre fica para ela): %s%s\n\n", proxima.Titulo, sufixo(" — ", proxima.Objetivo))
	}
	escreva("A trilha inteira, na ordem:\n\n")
	for _, m := range tr.Modulos {
		escreva("- %s\n", m.Titulo)
		for _, x := range m.Etapas {
			marca := ""
			if x.ID == e.ID {
				marca = "  ← esta"
			}
			escreva("  - %s%s\n", x.Titulo, marca)
		}
	}

	// Livros da Biblioteca e Temas do Acervo (fase 6): o texto vai junto, em material/.
	doMaterial := map[string]string{}
	if len(e.Materiais) > 0 {
		lidos := a.arquivosDoMaterial(ctx, e.Materiais, doMaterial)
		escreva("\n## Material da etapa\n\nA lição se apoia nele (é a fonte principal); cite-o onde usar.\n\n")
		for i, m := range e.Materiais {
			escreva("%s", descreverMaterial(m, lidos[i]))
		}
		escreva("\nOs documentos da trilha estão na pasta `documentos` (adicionada ao seu acesso; procure-a com Glob se o caminho relativo não abrir).")
		if len(doMaterial) > 0 {
			escreva(" O texto dos livros e do Acervo está em `material/`, nesta pasta: num livro, cite o capítulo e a página (o número entre colchetes); ele é protegido, então explique com as suas palavras e mande ler o trecho, sem copiar.")
		}
		escreva("\n")
	}

	escreva("\n## Instruções do Lucas\n\nVão da mais permanente à mais pontual; em conflito, vale a mais específica.\n\n")
	escreva("- Globais: %s\n", padrao(cfg.InstrucoesGlobais, "(nenhuma)"))
	escreva("- Da trilha: %s\n", padrao(tr.Instrucoes, "(nenhuma)"))
	escreva("- Deste pedido: %s\n", padrao(p.Pedido, "(nenhuma)"))

	arquivos := doMaterial
	comentarios := map[int64]bool{}
	slug := tr.Slug
	if p.AnteriorID > 0 {
		ant, err := a.propostaPorID(ctx, p.AnteriorID)
		if err != nil {
			return nil, nil, err
		}
		if err := a.completar(ctx, &ant); err != nil {
			return nil, nil, err
		}
		var c conteudoLicao
		if err := json.Unmarshal(ant.Conteudo, &c); err != nil {
			return nil, nil, fmt.Errorf("versão anterior ilegível: %w", err)
		}
		arquivos["versao-anterior.md"] = c.Licao
		escreva("\n## É um ajuste da versão %d\n\n", ant.Versao)
		escreva("A versão anterior está em `versao-anterior.md`. Parta dela: mude o que os comentários pedem e mantenha o resto (o Lucas vai ver a diferença entre as versões). Responda **cada** comentário em `respostas`, pelo número: o que mudou e onde, ou por que não mudou.\n\n")
		for _, c := range ant.Comentarios {
			if c.Autor != "lucas" {
				continue
			}
			comentarios[c.ID] = true
			escreva("- Comentário %d", c.ID)
			if c.Ancora != "" {
				escreva(" (em %s)", c.Ancora)
			}
			if c.Citacao != "" {
				escreva(", sobre o trecho: «%s»", c.Citacao)
			}
			escreva(": %s\n", c.Texto)
		}
		if len(comentarios) == 0 {
			escreva("(Sem comentários: o pedido acima diz o que ajustar.)\n")
		}
	} else if texto, err := a.conteudo.ler(caminhoLicao(slug, e.Chave)); err == nil && strings.TrimSpace(texto) != "" {
		arquivos["licao-atual.md"] = texto
		escreva("\n## A etapa já tem lição\n\nEla está em `licao-atual.md`. Este pedido é uma **mudança** nela: parta dela e mude o que o pedido diz; o que ele não pede, mantenha.\n")
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, nil
}

func (a *app) tituloDaTrilha(ctx context.Context, id int64) string {
	var titulo string
	a.db.QueryRowContext(ctx, `SELECT titulo FROM trilhas WHERE id = ?`, id).Scan(&titulo)
	return titulo
}

func sufixo(sep, texto string) string {
	if strings.TrimSpace(texto) == "" {
		return ""
	}
	return sep + texto
}

func padrao(texto, alternativa string) string {
	if strings.TrimSpace(texto) == "" {
		return alternativa
	}
	return strings.TrimSpace(texto)
}

func licaoEm(e etapa) string {
	if e.TemLicao {
		return " (lição em `licoes/" + e.Chave + ".md`)"
	}
	return ""
}
