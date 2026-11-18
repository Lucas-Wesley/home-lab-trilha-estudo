package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Revisar o roteiro (T34): a IA relê os tópicos da trilha contra o objetivo, o recorte, o nível de
// partida, o alvo e a profundidade (que o Lucas pode ter mudado depois de criar a trilha) e propõe
// mudanças: editar um tópico, criar um, mover ou remover. Vale para a trilha inteira (chave
// "@roteiro") ou para um tópico só ("@roteiro:<chave da etapa>"). O Lucas marca o que aceita.
// As mesmas mudanças "editar" vêm também no encaixe de material (T32), como revisões dos tópicos.

const tipoRoteiro = "roteiro"

const chaveRoteiro = "@roteiro"

var acoesRoteiro = []string{"editar", "novo", "mover", "remover"}

// atualEtapa: como a etapa está hoje, para a tela mostrar o antes e o depois.
type atualEtapa struct {
	Titulo   string `json:"titulo"`
	Objetivo string `json:"objetivo"`
	Criterio string `json:"criterio"`
	Bloom    string `json:"bloom"`
	Minutos  int    `json:"minutos"`
	Modulo   string `json:"modulo"`
	Status   string `json:"status"`
	TemLicao bool   `json:"tem_licao"`
	ID       int64  `json:"id"`
}

type mudancaRoteiro struct {
	Acao       string `json:"acao"`
	EtapaChave string `json:"etapa_chave"`
	Titulo     string `json:"titulo"`
	Objetivo   string `json:"objetivo"`
	Criterio   string `json:"criterio"`
	Bloom      string `json:"bloom"`
	Minutos    int    `json:"minutos"`
	Modulo     string `json:"modulo"`
	DepoisDe   string `json:"depois_de"`
	Motivo     string `json:"motivo"`
	// Preenchidos pelo servidor, para a tela.
	Numero     string      `json:"numero,omitempty"`
	Atual      *atualEtapa `json:"atual,omitempty"`
	ModuloNovo bool        `json:"modulo_novo,omitempty"`
}

type conteudoRoteiro struct {
	Foco      string               `json:"foco,omitempty"` // chave da etapa, quando é a revisão de um tópico
	Mudancas  []mudancaRoteiro     `json:"mudancas"`
	Notas     string               `json:"notas,omitempty"`
	Respostas []respostaComentario `json:"respostas,omitempty"`
}

// Campos de uma mudança, no esquema (o encaixe usa os de "editar").
const camposMudanca = `
  "etapa_chave":{"type":"string","description":"editar, mover, remover: a chave da etapa; novo: vazio"},
  "titulo":{"type":"string","description":"editar: o título novo, ou vazio para manter; novo: o título"},
  "objetivo":{"type":"string","description":"com verbo de Bloom e o que aprender, bem específico; editar: vazio para manter"},
  "criterio":{"type":"string","description":"critério de conclusão verificável (com a prática no Mac, se a trilha pede); editar: vazio para manter"},
  "bloom":{"type":"string","enum":["","lembrar","entender","aplicar","analisar","avaliar","criar"]},
  "minutos":{"type":"integer","description":"0 para manter"},
  "motivo":{"type":"string","description":"por que mudar, em uma ou duas frases"}`

const esquemaRoteiro = `{"type":"object","properties":{
"mudancas":{"type":"array","items":{"type":"object","properties":{
  "acao":{"type":"string","enum":["editar","novo","mover","remover"]},` + camposMudanca + `,
  "modulo":{"type":"string","description":"novo, mover: o título exato de um módulo existente, ou o de um módulo novo; senão vazio"},
  "depois_de":{"type":"string","description":"novo, mover: a chave da etapa depois da qual entra (vazio = no fim do módulo)"}},
  "required":["acao","etapa_chave","titulo","objetivo","criterio","bloom","minutos","modulo","depois_de","motivo"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["mudancas","notas","respostas"]}`

func chaveDoRoteiro(foco string) string {
	if foco == "" {
		return chaveRoteiro
	}
	return chaveRoteiro + ":" + foco
}

func focoDoRoteiro(chave string) string {
	return strings.TrimPrefix(strings.TrimPrefix(chave, chaveRoteiro), ":")
}

// ---------- pedir e listar ----------

func (a *app) handleRevisarRoteiro(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pedido     string `json:"pedido"`
		EtapaChave string `json:"etapa_chave"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	tr, err := a.trilhaCompleta(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if len(tr.Modulos) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a trilha ainda não tem tópicos para revisar")})
		return
	}
	titulo := "Revisar o roteiro: " + tr.Titulo
	if corpo.EtapaChave != "" {
		e, err := a.etapaPorChave(ctx, id, corpo.EtapaChave)
		if err != nil {
			a.responderErro(w, erroDeEntrada{errors.New("essa etapa não é desta trilha")})
			return
		}
		titulo = "Revisar o tópico: " + e.Titulo
	}
	p, err := a.novaProposta(ctx, id, chaveDoRoteiro(corpo.EtapaChave), tipoRoteiro, strings.TrimSpace(corpo.Pedido), 0, titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

// handleRoteiro: a última versão (não descartada) da revisão da trilha ou do tópico (?etapa=chave).
func (a *app) handleRoteiro(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	var pid int64
	err = a.db.QueryRowContext(ctx, `SELECT id FROM propostas WHERE trilha_id = ? AND tipo = ? AND etapa_chave = ? AND estado != ?
		ORDER BY versao DESC LIMIT 1`, id, tipoRoteiro, chaveDoRoteiro(r.URL.Query().Get("etapa")), propDescartada).Scan(&pid)
	if errors.Is(err, sql.ErrNoRows) {
		responderJSON(w, http.StatusOK, map[string]any{"proposta": nil})
		return
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	p, err := a.propostaPorID(ctx, pid)
	if err == nil {
		err = a.completar(ctx, &p)
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"proposta": p})
}

// ---------- gerar ----------

func (a *app) gerarRoteiro(ctx context.Context, t *tarefa) error {
	p, err := a.propostaPorID(ctx, t.PropostaID)
	if err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, p.TrilhaID)
	if err != nil {
		return err
	}
	foco := focoDoRoteiro(p.EtapaChave)
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	arquivos, comentarios, web, err := a.montarPedidoRoteiro(ctx, tr, p, foco)
	if err != nil {
		return err
	}
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	var extras []string
	if caminho, err := a.conteudo.caminho("trilhas/" + tr.Slug + "/documentos"); err == nil {
		if _, err := os.Stat(caminho); err == nil {
			extras = append(extras, caminho)
		}
	}
	rg, err := a.ia.lerRegra("roteiro.md")
	if err != nil {
		return err
	}
	ferramentas := "Read,Glob,Grep"
	if web {
		ferramentas += ",WebFetch"
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: rg.Texto, esquema: esquemaRoteiro, modelo: t.Modelo, ferramentas: ferramentas,
		instrucao: "Revise o roteiro (os tópicos) de uma trilha de estudo e proponha mudanças. " +
			"Tudo está em `pedido.md`, nesta pasta: leia-o inteiro primeiro.",
	})
	if err != nil {
		return err
	}
	var c conteudoRoteiro
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	c.Foco = foco
	var tirados []string
	c.Mudancas, tirados = limparMudancas(tr, c.Mudancas, acoesRoteiro)
	if len(tirados) > 0 {
		c.Notas = strings.TrimSpace(c.Notas + "\n\n(O app tirou: " + strings.Join(tirados, "; ") + ".)")
	}
	bruto, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return a.guardarProposta(ctx, p, string(bruto), c.Respostas, comentarios)
}

// limparMudancas tira o que aponta para etapa que não existe, a edição que não muda nada e a
// ação que não cabe; completa número, módulo novo e como a etapa está hoje.
func limparMudancas(tr trilha, ms []mudancaRoteiro, acoes []string) ([]mudancaRoteiro, []string) {
	num := numeracao(tr)
	etapas := map[string]atualEtapa{}
	modulos := map[string]bool{}
	for _, md := range tr.Modulos {
		modulos[strings.ToLower(strings.TrimSpace(md.Titulo))] = true
		for _, e := range md.Etapas {
			etapas[e.Chave] = atualEtapa{e.Titulo, e.Objetivo, e.Criterio, e.Bloom, e.Minutos, md.Titulo, e.Status, e.TemLicao, e.ID}
		}
	}
	var bons []mudancaRoteiro
	var tirados []string
	for _, x := range ms {
		x.Acao = strings.TrimSpace(x.Acao)
		x.EtapaChave, x.Titulo, x.Objetivo, x.Criterio = strings.TrimSpace(x.EtapaChave), strings.TrimSpace(x.Titulo), strings.TrimSpace(x.Objetivo), strings.TrimSpace(x.Criterio)
		x.Modulo, x.DepoisDe = strings.TrimSpace(x.Modulo), strings.TrimSpace(x.DepoisDe)
		if !contem(acoes, x.Acao) {
			tirados = append(tirados, fmt.Sprintf("uma mudança do tipo %q", x.Acao))
			continue
		}
		if x.Bloom != "" && !contem(niveisBloom, x.Bloom) {
			x.Bloom = ""
		}
		if x.Minutos < 0 {
			x.Minutos = 0
		}
		if _, ok := etapas[x.DepoisDe]; !ok {
			x.DepoisDe = ""
		}
		if x.Acao != "novo" {
			at, ok := etapas[x.EtapaChave]
			if !ok {
				tirados = append(tirados, fmt.Sprintf("a etapa %q (não existe)", x.EtapaChave))
				continue
			}
			x.Numero, x.Atual = num[x.EtapaChave], &at
		}
		switch x.Acao {
		case "editar":
			at := x.Atual
			// Campo igual ao de hoje = manter.
			if x.Titulo == at.Titulo {
				x.Titulo = ""
			}
			if x.Objetivo == at.Objetivo {
				x.Objetivo = ""
			}
			if x.Criterio == at.Criterio {
				x.Criterio = ""
			}
			if x.Bloom == at.Bloom {
				x.Bloom = ""
			}
			if x.Minutos == at.Minutos {
				x.Minutos = 0
			}
			if x.Titulo == "" && x.Objetivo == "" && x.Criterio == "" && x.Bloom == "" && x.Minutos == 0 {
				tirados = append(tirados, fmt.Sprintf("a edição de %s (não mudava nada)", x.Numero))
				continue
			}
			x.Modulo, x.DepoisDe = "", ""
		case "novo":
			if x.Titulo == "" || x.Modulo == "" {
				tirados = append(tirados, "um tópico novo sem título ou módulo")
				continue
			}
			x.EtapaChave = ""
			if x.Bloom == "" {
				x.Bloom = "entender"
			}
			if x.Minutos <= 0 {
				x.Minutos = 60
			}
			x.ModuloNovo = !modulos[strings.ToLower(x.Modulo)]
		case "mover":
			if x.Modulo == "" {
				x.Modulo = x.Atual.Modulo
			}
			if x.DepoisDe == x.EtapaChave {
				x.DepoisDe = ""
			}
			x.ModuloNovo = !modulos[strings.ToLower(x.Modulo)]
		case "remover":
			x.Modulo, x.DepoisDe = "", ""
		}
		bons = append(bons, x)
	}
	return vazioSeNulo(bons), tirados
}

func (a *app) montarPedidoRoteiro(ctx context.Context, tr trilha, p proposta, foco string) (map[string]string, map[int64]bool, bool, error) {
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	arquivos := map[string]string{}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: revisar o roteiro da trilha \"%s\"\n\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Descrição", tr.Descricao}, {"Recorte", tr.Recorte}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
		if strings.TrimSpace(campo[1]) != "" {
			escreva("- %s: %s\n", campo[0], strings.TrimSpace(campo[1]))
		}
	}
	escreva("- Profundidade das lições: %s\n", nomesProfundidade[tr.Profundidade])

	web := false
	if foco != "" {
		e, err := a.etapaPorChave(ctx, tr.ID, foco)
		if err != nil {
			return nil, nil, false, errors.New("a etapa desta revisão não existe mais na trilha")
		}
		escreva("\n## Foco: só o tópico `%s` (%s %s)\n\n", e.Chave, numeracao(tr)[e.Chave], e.Titulo)
		escreva("Revise **este tópico**: título, objetivo, critério, nível e tempo; se ele for grande demais, proponha dividi-lo (editar este + novos logo depois); se estiver no lugar errado, mover; se sobrar, remover. Mude outros tópicos só como consequência direta (ex.: o vizinho que passa a repetir este).\n")
		if len(e.Materiais) > 0 {
			escreva("\nO material que o tópico já tem (use-o para ver o que de fato dá para ensinar aqui):\n\n")
			lidos := a.arquivosDoMaterial(ctx, e.Materiais, arquivos)
			for i, m := range e.Materiais {
				escreva("- %s (%s%s)", m.Titulo, m.Tipo, sufixo(", ", m.Localizador))
				switch {
				case lidos[i].Arquivo != "":
					escreva(": texto em `%s`", lidos[i].Arquivo)
				case m.Tipo == "documento":
					escreva(": `documentos/%s.md`", m.Referencia)
				case m.Tipo == "web":
					escreva(": %s (abra com WebFetch se precisar)", m.Referencia)
					web = true
				}
				escreva("\n")
			}
		}
	}

	num := numeracao(tr)
	escreva("\n## A trilha, na ordem\n\nCada tópico: número, `chave`, [tipo], título — objetivo · critério · Bloom, minutos · estado.\n")
	for i, md := range tr.Modulos {
		escreva("\n### Módulo %d: %s\n", i+1, md.Titulo)
		if md.Descricao != "" {
			escreva("%s\n", md.Descricao)
		}
		escreva("\n")
		for _, e := range md.Etapas {
			escreva("- **%s** `%s`", num[e.Chave], e.Chave)
			if e.Tipo != "normal" {
				escreva(" [%s]", e.Tipo)
			}
			escreva(" %s — %s", e.Titulo, e.Objetivo)
			if e.Criterio != "" {
				escreva(" · critério: %s", e.Criterio)
			}
			escreva(" · %s, %d min", padrao(e.Bloom, "sem Bloom"), e.Minutos)
			var est []string
			if e.TemLicao {
				est = append(est, "tem lição")
			}
			if e.Status == "concluida" || e.Status == "em_andamento" {
				est = append(est, map[string]string{"concluida": "concluída", "em_andamento": "em andamento"}[e.Status])
			}
			if len(e.Materiais) > 0 {
				est = append(est, fmt.Sprintf("%d material(is)", len(e.Materiais)))
			}
			if len(est) > 0 {
				escreva(" · %s", strings.Join(est, ", "))
			}
			escreva("\n")
		}
	}

	escreva("\n## Instruções do Lucas\n\nVão da mais permanente à mais pontual; em conflito, vale a mais específica.\n\n")
	escreva("- Globais: %s\n", padrao(cfg.InstrucoesGlobais, "(nenhuma)"))
	escreva("- Da trilha: %s\n", padrao(tr.Instrucoes, "(nenhuma)"))
	escreva("- Deste pedido: %s\n", padrao(p.Pedido, "(nenhuma)"))

	comentarios := map[int64]bool{}
	if p.AnteriorID > 0 {
		ant, err := a.propostaPorID(ctx, p.AnteriorID)
		if err != nil {
			return nil, nil, false, err
		}
		if err := a.completar(ctx, &ant); err != nil {
			return nil, nil, false, err
		}
		arquivos["versao-anterior.json"] = string(ant.Conteudo)
		escreva("\n## É um ajuste da versão %d\n\nA versão anterior está em `versao-anterior.json`. Parta dela: mude o que o pedido e os comentários dizem e mantenha o resto. Responda **cada** comentário em `respostas`, pelo número.\n\n", ant.Versao)
		for _, c := range ant.Comentarios {
			if c.Autor != "lucas" {
				continue
			}
			comentarios[c.ID] = true
			escreva("- Comentário %d: %s\n", c.ID, c.Texto)
		}
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, web, nil
}

// ---------- aplicar ----------

// editarEtapa aplica os campos de uma mudança "editar" (vazio = manter).
func editarEtapa(e etapa, x mudancaRoteiro) etapa {
	if x.Titulo != "" {
		e.Titulo = x.Titulo
	}
	if x.Objetivo != "" {
		e.Objetivo = x.Objetivo
	}
	if x.Criterio != "" {
		e.Criterio = x.Criterio
	}
	if x.Bloom != "" {
		e.Bloom = x.Bloom
	}
	if x.Minutos > 0 {
		e.Minutos = x.Minutos
	}
	return e
}

// moduloOuNovo: o id do módulo pelo título; se não existe, cria no fim da trilha.
func moduloOuNovo(ctx context.Context, tx *sql.Tx, trilhaID int64, modulos map[string]int64, titulo string) (int64, error) {
	if mid, ok := modulos[strings.ToLower(strings.TrimSpace(titulo))]; ok {
		return mid, nil
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO modulos (trilha_id, ordem, titulo, descricao)
		VALUES (?, (SELECT COALESCE(MAX(ordem), -1) + 1 FROM modulos WHERE trilha_id = ?), ?, '')`, trilhaID, trilhaID, strings.TrimSpace(titulo))
	if err != nil {
		return 0, err
	}
	mid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	modulos[strings.ToLower(strings.TrimSpace(titulo))] = mid
	return mid, nil
}

// lugarNoModulo abre espaço no módulo e devolve a ordem: logo depois de "depoisDe" (se ela está
// nesse módulo) ou no fim. ignorar = a própria etapa, quando ela está sendo movida.
func lugarNoModulo(ctx context.Context, tx *sql.Tx, mid int64, depoisDe int64, ignorar int64) (int, error) {
	var ordem int
	var moduloDela int64
	err := tx.QueryRowContext(ctx, `SELECT ordem, modulo_id FROM etapas WHERE id = ?`, depoisDe).Scan(&ordem, &moduloDela)
	if err == nil && moduloDela == mid && depoisDe != ignorar {
		ordem++
	} else if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordem), -1) + 1 FROM etapas WHERE modulo_id = ? AND id != ?`, mid, ignorar).Scan(&ordem); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE etapas SET ordem = ordem + 1 WHERE modulo_id = ? AND ordem >= ? AND id != ?`, mid, ordem, ignorar); err != nil {
		return 0, err
	}
	return ordem, nil
}

// inserirEtapa cria a etapa nova no módulo (pelo título, criando-o se preciso) depois de "depoisDe".
func (a *app) inserirEtapa(ctx context.Context, tx *sql.Tx, modulos map[string]int64, etapas map[string]etapa, modulo, depoisDe string, nova etapa) (string, error) {
	mid, err := moduloOuNovo(ctx, tx, nova.TrilhaID, modulos, modulo)
	if err != nil {
		return "", err
	}
	nova.ModuloID = mid
	if nova.Requer == nil {
		nova.Requer = []int64{}
	}
	if err := validarEtapa(&nova); err != nil {
		return "", erroDeEntrada{fmt.Errorf("%s: %w", nova.Titulo, err)}
	}
	chave, err := slugLivre(ctx, tx, slugificar(nova.Titulo), `SELECT COUNT(*) FROM etapas WHERE trilha_id = ? AND chave = ?`, nova.TrilhaID)
	if err != nil {
		return "", err
	}
	ordem, err := lugarNoModulo(ctx, tx, mid, etapas[depoisDe].ID, 0)
	if err != nil {
		return "", err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo) VALUES (?, ?, ?, ?, ?)`,
		nova.TrilhaID, mid, ordem, chave, nova.Titulo)
	if err != nil {
		return "", err
	}
	if nova.ID, err = res.LastInsertId(); err != nil {
		return "", err
	}
	if err := a.salvarEtapa(ctx, tx, nova); err != nil {
		return "", err
	}
	nova.Chave = chave
	etapas[chave] = nova
	return chave, nil
}

func (a *app) aprovarRoteiro(w http.ResponseWriter, r *http.Request, p proposta) {
	var corpo struct {
		Mudancas []int `json:"mudancas"`
	}
	if r.ContentLength != 0 {
		if err := lerCorpo(r, &corpo); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	ctx := r.Context()
	if err := a.semVersaoGerando(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	var c conteudoRoteiro
	if err := json.Unmarshal(p.Conteudo, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(corpo.Mudancas) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("marque as mudanças que entram (ou descarte a proposta)")})
		return
	}
	var marcadas []mudancaRoteiro
	vistos := map[int]bool{}
	for _, i := range corpo.Mudancas {
		if i < 0 || i >= len(c.Mudancas) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("mudança %d não está na proposta", i+1)})
			return
		}
		if !vistos[i] {
			vistos[i] = true
			marcadas = append(marcadas, c.Mudancas[i])
		}
	}
	// O banco tem uma conexão só: tudo o que for lido, antes da transação.
	tr, err := a.trilhaCompleta(ctx, p.TrilhaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	etapas := map[string]etapa{}
	modulos := map[string]int64{}
	for _, md := range tr.Modulos {
		modulos[strings.ToLower(strings.TrimSpace(md.Titulo))] = md.ID
		for _, e := range md.Etapas {
			etapas[e.Chave] = e
		}
	}
	for _, x := range marcadas {
		if _, ok := etapas[x.EtapaChave]; x.Acao != "novo" && !ok {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("a etapa %q não existe mais na trilha", x.EtapaChave)})
			return
		}
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	// Ordem: editar, remover, mover, criar (o tópico novo pode entrar depois de um movido).
	var resumo []string
	removidos := map[string]*string{}
	for _, fase := range []string{"editar", "remover", "mover", "novo"} {
		for _, x := range marcadas {
			if x.Acao != fase {
				continue
			}
			switch x.Acao {
			case "editar":
				e := editarEtapa(etapas[x.EtapaChave], x)
				if err := validarEtapa(&e); err != nil {
					a.responderErro(w, erroDeEntrada{err})
					return
				}
				if err := a.salvarEtapa(ctx, tx, e); err != nil {
					a.responderErro(w, err)
					return
				}
				etapas[x.EtapaChave] = e
				resumo = append(resumo, "editou "+x.EtapaChave)
			case "remover":
				if _, err := tx.ExecContext(ctx, `DELETE FROM etapas WHERE id = ?`, etapas[x.EtapaChave].ID); err != nil {
					a.responderErro(w, err)
					return
				}
				removidos[caminhoLicao(tr.Slug, x.EtapaChave)] = nil
				removidos[caminhoEstudo(tr.Slug, tipoExercicios, x.EtapaChave)] = nil
				removidos[caminhoEstudo(tr.Slug, tipoSimulado, x.EtapaChave)] = nil
				delete(etapas, x.EtapaChave)
				resumo = append(resumo, "removeu "+x.EtapaChave)
			case "mover":
				e := etapas[x.EtapaChave]
				mid, err := moduloOuNovo(ctx, tx, p.TrilhaID, modulos, x.Modulo)
				if err != nil {
					a.responderErro(w, err)
					return
				}
				ordem, err := lugarNoModulo(ctx, tx, mid, etapas[x.DepoisDe].ID, e.ID)
				if err != nil {
					a.responderErro(w, err)
					return
				}
				if _, err := tx.ExecContext(ctx, `UPDATE etapas SET modulo_id = ?, ordem = ? WHERE id = ?`, mid, ordem, e.ID); err != nil {
					a.responderErro(w, err)
					return
				}
				e.ModuloID = mid
				etapas[x.EtapaChave] = e
				resumo = append(resumo, "moveu "+x.EtapaChave)
			case "novo":
				nova := etapa{TrilhaID: p.TrilhaID, Titulo: x.Titulo, Objetivo: x.Objetivo, Criterio: x.Criterio, Bloom: x.Bloom, Tipo: "normal",
					Minutos: x.Minutos, Materiais: []material{}}
				chave, err := a.inserirEtapa(ctx, tx, modulos, etapas, x.Modulo, x.DepoisDe, nova)
				if err != nil {
					a.responderErro(w, err)
					return
				}
				resumo = append(resumo, "criou "+chave)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(ctx, p.TrilhaID)
	if err := a.exportar(ctx, p.TrilhaID, fmt.Sprintf("roteiro revisado: %s (proposta v%d aprovada)", strings.Join(resumo, ", "), p.Versao), removidos); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}
