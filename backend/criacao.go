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

// Criação com IA (fase 4, DECISOES.md §3 e T25). A trilha nasce como rascunho logo no começo, e
// cada passo da IA é uma proposta dela (alvo = a trilha, sem etapa):
//
//	A. "A IA monta":       entrevista (1 a 4 rodadas) → mapa (o Lucas marca o recorte) → plano
//	B. "Eu tenho a lista": validação (o Lucas aceita sugestões) → plano
//
// Aprovar o plano importa a trilha (o mesmo caminho do POST /api/importar). Comentários, ajuste,
// descarte e a fila são os mesmos da lição.

const (
	tipoEntrevista = "entrevista"
	tipoMapa       = "mapa"
	tipoValidacao  = "validacao"
	tipoPlano      = "plano"
	passoPronta    = "pronta"
)

var tiposCriacao = []string{tipoEntrevista, tipoMapa, tipoValidacao, tipoPlano}

var titulosCriacao = map[string]string{
	tipoEntrevista: "Entrevista", tipoMapa: "Mapa do assunto", tipoValidacao: "Validação da lista", tipoPlano: "Plano da trilha",
}

// Tamanho da entrevista (§3): rodadas e perguntas no total.
type limiteEntrevista struct {
	Rodadas, Perguntas int
	Nome               string
}

var tamanhosEntrevista = map[string]limiteEntrevista{
	"rapida":   {1, 3, "rápida (2 a 3 perguntas, uma rodada)"},
	"padrao":   {2, 8, "padrão (5 a 8 perguntas, em 1 ou 2 rodadas)"},
	"profunda": {4, 15, "profunda (até ~15 perguntas, em até 4 rodadas)"},
}

type criacao struct {
	TrilhaID   int64          `json:"trilha_id"`
	Entrada    string         `json:"entrada"` // ia · lista
	Tema       string         `json:"tema"`
	Pedido     string         `json:"pedido"`
	Entrevista string         `json:"entrevista"`
	Lista      string         `json:"lista"`
	Passo      string         `json:"passo"`
	Rodadas    []rodada       `json:"rodadas"`
	Escolhas   escolhasCriado `json:"escolhas"`
	CriadoEm   string         `json:"criado_em"`
}

// rodada: as perguntas de uma proposta de entrevista e o que o Lucas respondeu.
type rodada struct {
	PropostaID int64      `json:"proposta_id"`
	Perguntas  []pergunta `json:"perguntas"`
	Respostas  []resposta `json:"respostas"`
	Comentario string     `json:"comentario"`
}

type pergunta struct {
	ID       string   `json:"id"`
	Texto    string   `json:"texto"`
	PorQue   string   `json:"por_que"`
	Opcoes   []string `json:"opcoes"`
	Multipla bool     `json:"multipla"`
}

type resposta struct {
	Pergunta string   `json:"pergunta"`
	Escolhas []string `json:"escolhas"`
	Outro    string   `json:"outro"`
	Sugira   bool     `json:"sugira"`
}

// escolhasCriado: o que o Lucas decidiu ao aprovar o mapa ou a validação.
type escolhasCriado struct {
	MapaID      int64    `json:"mapa_id,omitempty"`
	Recorte     []string `json:"recorte,omitempty"`
	ValidacaoID int64    `json:"validacao_id,omitempty"`
	Aceitas     []string `json:"aceitas,omitempty"`
}

// ---------- o que a IA devolve ----------

type conteudoEntrevista struct {
	Perfil     string               `json:"perfil"`
	Eixos      []string             `json:"eixos"`
	Panorama   string               `json:"panorama"`
	Perguntas  []pergunta           `json:"perguntas"`
	Suficiente bool                 `json:"suficiente"`
	Notas      string               `json:"notas"`
	Respostas  []respostaComentario `json:"respostas"`
}

type etapaDoMapa struct {
	Chave  string `json:"chave"`
	Titulo string `json:"titulo"`
	Resumo string `json:"resumo"`
	Nivel  string `json:"nivel"`
}

type conteudoMapa struct {
	Objetivo string `json:"objetivo"`
	Mapa     []struct {
		Titulo    string        `json:"titulo"`
		Descricao string        `json:"descricao"`
		Etapas    []etapaDoMapa `json:"etapas"`
	} `json:"mapa"`
	RecorteSugerido []string             `json:"recorte_sugerido"`
	PorQueRecorte   string               `json:"por_que_recorte"`
	NivelPartida    string               `json:"nivel_partida"`
	NivelAlvo       string               `json:"nivel_alvo"`
	Notas           string               `json:"notas"`
	Respostas       []respostaComentario `json:"respostas"`
}

type sugestao struct {
	ID      string `json:"id"`
	Acao    string `json:"acao"` // inserir · dividir · observar
	AntesDe string `json:"antes_de,omitempty"`
	Item    string `json:"item,omitempty"`
	Titulo  string `json:"titulo"`
	Motivo  string `json:"motivo"`
}

type conteudoValidacao struct {
	Itens []struct {
		Chave  string `json:"chave"`
		Titulo string `json:"titulo"`
		Nota   string `json:"nota"`
	} `json:"itens"`
	Sugestoes    []sugestao           `json:"sugestoes"`
	NivelPartida string               `json:"nivel_partida"`
	NivelAlvo    string               `json:"nivel_alvo"`
	Notas        string               `json:"notas"`
	Respostas    []respostaComentario `json:"respostas"`
}

// planoTrilha é a trilha do pacote de importação (importar.go), mais o que o servidor conferiu.
type conteudoPlano struct {
	Trilha    json.RawMessage      `json:"trilha"`
	Notas     string               `json:"notas"`
	Respostas []respostaComentario `json:"respostas"`
	Links     []linkConferido      `json:"links,omitempty"`
	Revisao   []aviso              `json:"revisao,omitempty"`
}

// aviso da revisão automática do plano (os critérios A de regras/criar-trilha.md). grave impede
// aprovar: o plano não entraria no app.
type aviso struct {
	Criterio string `json:"criterio"`
	Onde     string `json:"onde,omitempty"` // etapa:<chave> · modulo:<n>
	Texto    string `json:"texto"`
	Grave    bool   `json:"grave,omitempty"`
}

const esquemaRespostas = `"respostas":{"type":"array","items":{"type":"object","properties":{
  "comentario":{"type":"integer"},"resposta":{"type":"string"}},"required":["comentario","resposta"]}}`

var esquemasCriacao = map[string]string{
	tipoEntrevista: `{"type":"object","properties":{
"perfil":{"type":"string","enum":["historia","tecnologia","lingua","ciencias","generico"]},
"eixos":{"type":"array","items":{"type":"string"}},
"panorama":{"type":"string"},
"perguntas":{"type":"array","items":{"type":"object","properties":{
  "id":{"type":"string"},"texto":{"type":"string"},"por_que":{"type":"string"},
  "opcoes":{"type":"array","items":{"type":"string"}},"multipla":{"type":"boolean"}},
  "required":["id","texto","por_que","opcoes","multipla"]}},
"suficiente":{"type":"boolean"},"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["perfil","eixos","panorama","perguntas","suficiente","notas","respostas"]}`,

	tipoMapa: `{"type":"object","properties":{
"objetivo":{"type":"string"},
"mapa":{"type":"array","items":{"type":"object","properties":{
  "titulo":{"type":"string"},"descricao":{"type":"string"},
  "etapas":{"type":"array","items":{"type":"object","properties":{
    "chave":{"type":"string"},"titulo":{"type":"string"},"resumo":{"type":"string"},
    "nivel":{"type":"string","enum":["basico","intermediario","avancado"]}},
    "required":["chave","titulo","resumo","nivel"]}}},
  "required":["titulo","descricao","etapas"]}},
"recorte_sugerido":{"type":"array","items":{"type":"string"}},
"por_que_recorte":{"type":"string"},"nivel_partida":{"type":"string"},"nivel_alvo":{"type":"string"},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["objetivo","mapa","recorte_sugerido","por_que_recorte","nivel_partida","nivel_alvo","notas","respostas"]}`,

	tipoValidacao: `{"type":"object","properties":{
"itens":{"type":"array","items":{"type":"object","properties":{
  "chave":{"type":"string"},"titulo":{"type":"string"},"nota":{"type":"string"}},"required":["chave","titulo","nota"]}},
"sugestoes":{"type":"array","items":{"type":"object","properties":{
  "id":{"type":"string"},"acao":{"type":"string","enum":["inserir","dividir","observar"]},
  "antes_de":{"type":"string"},"item":{"type":"string"},"titulo":{"type":"string"},"motivo":{"type":"string"}},
  "required":["id","acao","titulo","motivo"]}},
"nivel_partida":{"type":"string"},"nivel_alvo":{"type":"string"},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["itens","sugestoes","nivel_partida","nivel_alvo","notas","respostas"]}`,

	tipoPlano: `{"type":"object","properties":{
"trilha":{"type":"object","properties":{
  "titulo":{"type":"string"},"objetivo":{"type":"string"},"descricao":{"type":"string"},"recorte":{"type":"string"},
  "nivel_partida":{"type":"string"},"nivel_alvo":{"type":"string"},
  "modulos":{"type":"array","items":{"type":"object","properties":{
    "titulo":{"type":"string"},"descricao":{"type":"string"},
    "etapas":{"type":"array","items":{"type":"object","properties":{
      "chave":{"type":"string"},"titulo":{"type":"string"},"objetivo":{"type":"string"},
      "bloom":{"type":"string","enum":["lembrar","entender","aplicar","analisar","avaliar","criar"]},
      "tipo":{"type":"string","enum":["normal","base"]},"minutos":{"type":"integer"},"criterio":{"type":"string"},
      "requer":{"type":"array","items":{"type":"string"}},
      "materiais":{"type":"array","items":{"type":"object","properties":{
        "tipo":{"type":"string","enum":["web","documento"]},"titulo":{"type":"string"},"referencia":{"type":"string"},
        "localizador":{"type":"string"},"papel":{"type":"string","enum":["principal","apoio","aprofundar"]},"nota":{"type":"string"}},
        "required":["tipo","titulo","referencia","papel"]}}},
      "required":["chave","titulo","objetivo","bloom","minutos","criterio","requer","materiais"]}}},
    "required":["titulo","etapas"]}}},
  "required":["titulo","objetivo","descricao","recorte","modulos"]},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["trilha","notas","respostas"]}`,
}

// ---------- banco ----------

func (a *app) criacaoDa(ctx context.Context, trilhaID int64) (criacao, error) {
	var c criacao
	var rodadas, escolhas string
	err := a.db.QueryRowContext(ctx, `SELECT trilha_id, entrada, tema, pedido, entrevista, lista, passo, rodadas, escolhas, criado_em
		FROM criacoes WHERE trilha_id = ?`, trilhaID).Scan(&c.TrilhaID, &c.Entrada, &c.Tema, &c.Pedido, &c.Entrevista, &c.Lista,
		&c.Passo, &rodadas, &escolhas, &c.CriadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return c, errNaoEncontrou
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal([]byte(rodadas), &c.Rodadas); err != nil {
		return c, err
	}
	if c.Rodadas == nil {
		c.Rodadas = []rodada{}
	}
	return c, json.Unmarshal([]byte(escolhas), &c.Escolhas)
}

func salvarCriacao(ctx context.Context, tx *sql.Tx, c criacao) error {
	rodadas, err := json.Marshal(c.Rodadas)
	if err != nil {
		return err
	}
	escolhas, err := json.Marshal(c.Escolhas)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE criacoes SET passo = ?, rodadas = ?, escolhas = ?, atualizado_em = ? WHERE trilha_id = ?`,
		c.Passo, string(rodadas), string(escolhas), agora(), c.TrilhaID)
	return err
}

func (a *app) propostasDaCriacao(ctx context.Context, trilhaID int64) ([]proposta, error) {
	marcas := strings.Repeat("?,", len(tiposCriacao))
	args := []any{trilhaID}
	for _, t := range tiposCriacao {
		args = append(args, t)
	}
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasProposta+` FROM propostas
		WHERE trilha_id = ? AND etapa_chave = '' AND tipo IN (`+marcas[:len(marcas)-1]+`) ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	lista := []proposta{}
	for rows.Next() {
		p, err := lerPropostaLinha(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		lista = append(lista, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range lista {
		if err := a.completar(ctx, &lista[i]); err != nil {
			return nil, err
		}
	}
	return lista, nil
}

// ---------- começar ----------

type pedidoCriacao struct {
	Entrada      string `json:"entrada"`
	Tema         string `json:"tema"`
	Pedido       string `json:"pedido"`
	Entrevista   string `json:"entrevista"`
	Lista        string `json:"lista"`
	Profundidade int    `json:"profundidade"`
	Instrucoes   string `json:"instrucoes"`
}

func (a *app) handleCriarComIA(w http.ResponseWriter, r *http.Request) {
	var c pedidoCriacao
	if err := lerCorpo(r, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	c.Tema, c.Pedido, c.Lista = strings.TrimSpace(c.Tema), strings.TrimSpace(c.Pedido), strings.TrimSpace(c.Lista)
	var passo string
	switch c.Entrada {
	case "ia":
		passo = tipoEntrevista
		if c.Entrevista == "" {
			c.Entrevista = "padrao"
		}
		if _, ok := tamanhosEntrevista[c.Entrevista]; !ok {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("tamanho de entrevista desconhecido: %q", c.Entrevista)})
			return
		}
		c.Lista = ""
	case "lista":
		passo = tipoValidacao
		c.Entrevista = ""
		if len(itensDaLista(c.Lista)) < 2 {
			a.responderErro(w, erroDeEntrada{errors.New("escreva a lista: um assunto por linha, pelo menos dois")})
			return
		}
	default:
		a.responderErro(w, erroDeEntrada{fmt.Errorf("forma de criar desconhecida: %q", c.Entrada)})
		return
	}
	if c.Tema == "" {
		a.responderErro(w, erroDeEntrada{errors.New("diga o que você quer aprender")})
		return
	}
	id, err := a.criarTrilha(r.Context(), trilha{Titulo: c.Tema, Profundidade: c.Profundidade, Instrucoes: strings.TrimSpace(c.Instrucoes)})
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `INSERT INTO criacoes (trilha_id, entrada, tema, pedido, entrevista, lista, passo, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, c.Entrada, c.Tema, c.Pedido, c.Entrevista, c.Lista, passo, agora(), agora()); err != nil {
		a.responderErro(w, err)
		return
	}
	// Os documentos próprios entram depois (a tela manda logo em seguida) e a IA só começa
	// quando a tela pedir: POST /api/trilhas/{id}/criacao/gerar.
	a.responderCriacao(w, r, id, http.StatusCreated)
}

// itensDaLista: uma linha, um item; marcadores ("-", "1.", "*") saem.
func itensDaLista(lista string) []string {
	var itens []string
	for _, linha := range strings.Split(lista, "\n") {
		linha = strings.TrimSpace(linha)
		linha = strings.TrimLeft(linha, "-*•→ \t")
		if i := strings.IndexAny(linha, ".)"); i > 0 && i <= 3 {
			if _, err := strconv.Atoi(linha[:i]); err == nil {
				linha = strings.TrimSpace(linha[i+1:])
			}
		}
		if linha != "" {
			itens = append(itens, linha)
		}
	}
	return itens
}

type respostaCriacao struct {
	Criacao   criacao    `json:"criacao"`
	Propostas []proposta `json:"propostas"`
	Limite    struct {
		Rodadas   int `json:"rodadas"`
		Perguntas int `json:"perguntas"`
	} `json:"limite"`
}

func (a *app) responderCriacao(w http.ResponseWriter, r *http.Request, trilhaID int64, status int) {
	c, err := a.criacaoDa(r.Context(), trilhaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ps, err := a.propostasDaCriacao(r.Context(), trilhaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	res := respostaCriacao{Criacao: c, Propostas: ps}
	if l, ok := tamanhosEntrevista[c.Entrevista]; ok {
		res.Limite.Rodadas, res.Limite.Perguntas = l.Rodadas, l.Perguntas
	}
	responderJSON(w, status, res)
}

func (a *app) handleCriacao(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderCriacao(w, r, id, http.StatusOK)
}

var errCriacaoOcupada = erroDeEntrada{errors.New("a IA já está trabalhando nesta trilha, ou há uma proposta esperando a sua decisão")}

// gerarPasso pede à IA a proposta do passo atual (o começo, ou de novo depois de descartar ou de
// uma falha). Uma proposta da criação por vez: gerando ou aguardando, recusa.
func (a *app) gerarPasso(ctx context.Context, c criacao, pedido string) (proposta, error) {
	if c.Passo == passoPronta {
		return proposta{}, erroDeEntrada{errors.New("a trilha já foi criada; mudanças agora são na página dela")}
	}
	var abertas int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM propostas WHERE trilha_id = ? AND etapa_chave = '' AND estado IN (?, ?)`,
		c.TrilhaID, propGerando, propAguardando).Scan(&abertas); err != nil {
		return proposta{}, err
	}
	if abertas > 0 {
		return proposta{}, errCriacaoOcupada
	}
	return a.novaProposta(ctx, c.TrilhaID, "", c.Passo, pedido, 0, titulosCriacao[c.Passo]+": "+c.Tema)
}

func (a *app) handleGerarPasso(w http.ResponseWriter, r *http.Request) {
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
	c, err := a.criacaoDa(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.gerarPasso(r.Context(), c, strings.TrimSpace(corpo.Pedido)); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderCriacao(w, r, id, http.StatusCreated)
}

// ---------- aprovar cada passo ----------

type decisaoCriacao struct {
	// entrevista
	Respostas  []resposta `json:"respostas"`
	Comentario string     `json:"comentario"`
	Proximo    string     `json:"proximo"` // rodada · mapa
	// mapa e validação
	Recorte      []string `json:"recorte"`
	Aceitas      []string `json:"aceitas"`
	NivelPartida string   `json:"nivel_partida"`
	NivelAlvo    string   `json:"nivel_alvo"`
	Profundidade int      `json:"profundidade"`
	Objetivo     string   `json:"objetivo"`
	// pedido que vai junto do próximo passo
	Pedido string `json:"pedido"`
}

func (a *app) aprovarCriacao(w http.ResponseWriter, r *http.Request, p proposta) {
	var d decisaoCriacao
	if r.ContentLength != 0 {
		if err := lerCorpo(r, &d); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	ctx := r.Context()
	c, err := a.criacaoDa(ctx, p.TrilhaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if c.Passo != p.Tipo {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("a criação está no passo %q, não em %q", c.Passo, p.Tipo)})
		return
	}
	switch p.Tipo {
	case tipoEntrevista:
		err = a.aprovarEntrevista(ctx, &c, p, d)
	case tipoMapa:
		err = a.aprovarMapa(ctx, &c, p, d)
	case tipoValidacao:
		err = a.aprovarValidacao(ctx, &c, p, d)
	case tipoPlano:
		err = a.aprovarPlano(ctx, &c, p)
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if c.Passo != passoPronta {
		if _, err := a.gerarPasso(ctx, c, strings.TrimSpace(d.Pedido)); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	a.responderCriacao(w, r, p.TrilhaID, http.StatusOK)
}

// fecharProposta marca a proposta aprovada e grava a criação, numa transação.
func (a *app) fecharProposta(ctx context.Context, c criacao, p proposta, depois func(tx *sql.Tx) error) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE propostas SET estado = ?, atualizado_em = ? WHERE id = ?`, propAprovada, agora(), p.ID); err != nil {
		return err
	}
	if err := salvarCriacao(ctx, tx, c); err != nil {
		return err
	}
	if depois != nil {
		if err := depois(tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *app) aprovarEntrevista(ctx context.Context, c *criacao, p proposta, d decisaoCriacao) error {
	var ce conteudoEntrevista
	if err := json.Unmarshal(p.Conteudo, &ce); err != nil {
		return err
	}
	limite := tamanhosEntrevista[c.Entrevista]
	switch d.Proximo {
	case "", tipoMapa:
		c.Passo = tipoMapa
	case "rodada":
		if len(c.Rodadas)+1 >= limite.Rodadas {
			return erroDeEntrada{fmt.Errorf("a entrevista %s tem no máximo %d rodada(s); siga para o mapa", limite.Nome, limite.Rodadas)}
		}
	default:
		return erroDeEntrada{fmt.Errorf("próximo passo desconhecido: %q", d.Proximo)}
	}
	ids := map[string]bool{}
	for _, q := range ce.Perguntas {
		ids[q.ID] = true
	}
	respostas := []resposta{}
	for _, x := range d.Respostas {
		x.Outro = strings.TrimSpace(x.Outro)
		if !ids[x.Pergunta] {
			return erroDeEntrada{fmt.Errorf("pergunta desconhecida: %q", x.Pergunta)}
		}
		if len(x.Escolhas) > 0 || x.Outro != "" || x.Sugira {
			respostas = append(respostas, x)
		}
	}
	c.Rodadas = append(c.Rodadas, rodada{PropostaID: p.ID, Perguntas: ce.Perguntas, Respostas: respostas, Comentario: strings.TrimSpace(d.Comentario)})
	return a.fecharProposta(ctx, *c, p, nil)
}

func (a *app) aprovarMapa(ctx context.Context, c *criacao, p proposta, d decisaoCriacao) error {
	var cm conteudoMapa
	if err := json.Unmarshal(p.Conteudo, &cm); err != nil {
		return err
	}
	existe := map[string]bool{}
	for _, m := range cm.Mapa {
		for _, e := range m.Etapas {
			existe[e.Chave] = true
		}
	}
	var recorte []string
	for _, k := range d.Recorte {
		if !existe[k] {
			return erroDeEntrada{fmt.Errorf("a etapa %q não está no mapa", k)}
		}
		recorte = append(recorte, k)
	}
	if len(recorte) == 0 {
		return erroDeEntrada{errors.New("marque no mapa as etapas que entram na trilha")}
	}
	c.Escolhas.MapaID, c.Escolhas.Recorte = p.ID, recorte
	c.Passo = tipoPlano
	return a.fecharProposta(ctx, *c, p, func(tx *sql.Tx) error {
		return atualizarNiveis(ctx, tx, c.TrilhaID, d, primeiro(strings.TrimSpace(d.Objetivo), cm.Objetivo))
	})
}

func (a *app) aprovarValidacao(ctx context.Context, c *criacao, p proposta, d decisaoCriacao) error {
	var cv conteudoValidacao
	if err := json.Unmarshal(p.Conteudo, &cv); err != nil {
		return err
	}
	existe := map[string]bool{}
	for _, s := range cv.Sugestoes {
		existe[s.ID] = true
	}
	aceitas := []string{}
	for _, id := range d.Aceitas {
		if !existe[id] {
			return erroDeEntrada{fmt.Errorf("sugestão desconhecida: %q", id)}
		}
		aceitas = append(aceitas, id)
	}
	c.Escolhas.ValidacaoID, c.Escolhas.Aceitas = p.ID, aceitas
	c.Passo = tipoPlano
	return a.fecharProposta(ctx, *c, p, func(tx *sql.Tx) error {
		return atualizarNiveis(ctx, tx, c.TrilhaID, d, "")
	})
}

// atualizarNiveis grava na trilha o que o Lucas confirmou (vazio = fica como está).
func atualizarNiveis(ctx context.Context, tx *sql.Tx, trilhaID int64, d decisaoCriacao, objetivo string) error {
	if d.Profundidade != 0 && (d.Profundidade < 1 || d.Profundidade > 5) {
		return erroDeEntrada{errors.New("profundidade vai de 1 (panorama) a 5 (deep dive)")}
	}
	_, err := tx.ExecContext(ctx, `UPDATE trilhas SET
		nivel_partida = CASE WHEN ? = '' THEN nivel_partida ELSE ? END,
		nivel_alvo = CASE WHEN ? = '' THEN nivel_alvo ELSE ? END,
		profundidade = CASE WHEN ? = 0 THEN profundidade ELSE ? END,
		objetivo = CASE WHEN ? = '' THEN objetivo ELSE ? END,
		atualizado_em = ? WHERE id = ?`,
		strings.TrimSpace(d.NivelPartida), strings.TrimSpace(d.NivelPartida), strings.TrimSpace(d.NivelAlvo), strings.TrimSpace(d.NivelAlvo),
		d.Profundidade, d.Profundidade, objetivo, objetivo, agora(), trilhaID)
	return err
}

// planoComoPacote monta o pacote de importação do plano, preso à trilha em criação: mesmo slug,
// e o que o Lucas já decidiu (instruções, profundidade) não muda.
func (a *app) planoComoPacote(ctx context.Context, trilhaID int64, cp conteudoPlano) (pacote, error) {
	var pk pacote
	pk.Substituir = true
	if err := json.Unmarshal([]byte(`{"trilha":`+string(cp.Trilha)+`}`), &pk); err != nil {
		return pk, erroDeEntrada{fmt.Errorf("o plano não está no formato de trilha: %w", err)}
	}
	if pk.Trilha == nil {
		return pk, erroDeEntrada{errors.New("o plano veio sem trilha")}
	}
	atual, err := lerTrilhaLinha(a.db.QueryRowContext(ctx, `SELECT `+colunasTrilha+` FROM trilhas WHERE id = ?`, trilhaID))
	if err != nil {
		return pk, err
	}
	t := &pk.Trilha.trilha
	t.Slug, t.Instrucoes, t.Profundidade = atual.Slug, atual.Instrucoes, atual.Profundidade
	t.Titulo = primeiro(strings.TrimSpace(t.Titulo), atual.Titulo)
	t.NivelPartida = primeiro(strings.TrimSpace(t.NivelPartida), atual.NivelPartida)
	t.NivelAlvo = primeiro(strings.TrimSpace(t.NivelAlvo), atual.NivelAlvo)
	return pk, nil
}

func (a *app) aprovarPlano(ctx context.Context, c *criacao, p proposta) error {
	var cp conteudoPlano
	if err := json.Unmarshal(p.Conteudo, &cp); err != nil {
		return err
	}
	for _, x := range cp.Revisao {
		if x.Grave {
			return erroDeEntrada{fmt.Errorf("o plano tem um problema que impede criar a trilha (%s): peça um ajuste", x.Texto)}
		}
	}
	pk, err := a.planoComoPacote(ctx, c.TrilhaID, cp)
	if err != nil {
		return err
	}
	if _, err := a.importar(ctx, pk); err != nil {
		return err
	}
	c.Passo = passoPronta
	return a.fecharProposta(ctx, *c, p, nil)
}

// ---------- a tarefa ----------

func (a *app) gerarCriacao(ctx context.Context, t *tarefa) error {
	p, err := a.propostaPorID(ctx, t.PropostaID)
	if err != nil {
		return err
	}
	c, err := a.criacaoDa(ctx, p.TrilhaID)
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

	arquivos, comentarios, err := a.montarPedidoCriacao(ctx, tr, c, p)
	if err != nil {
		return err
	}
	for nome, texto := range arquivos {
		if err := os.WriteFile(filepath.Join(pasta, nome), []byte(texto), 0o644); err != nil {
			return err
		}
	}
	var extras []string
	if caminho, err := a.conteudo.caminho("trilhas/" + tr.Slug + "/documentos"); err == nil {
		if _, err := os.Stat(caminho); err == nil {
			extras = append(extras, caminho)
		}
	}
	regra, err := a.regraDaCriacao(p.Tipo)
	if err != nil {
		return err
	}
	instrucao := "Você está ajudando a criar uma trilha de estudo (" + strings.ToLower(titulosCriacao[p.Tipo]) + "). " +
		"Tudo o que você precisa saber está em `pedido.md`, nesta pasta: leia-o inteiro primeiro, e depois os arquivos que ele indicar."
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: regra, instrucao: instrucao,
		ferramentas: "Read,Glob,Grep,WebSearch,WebFetch", esquema: esquemasCriacao[p.Tipo], modelo: t.Modelo,
	})
	if err != nil {
		return err
	}
	bruto, respostas, err := a.lerRespostaCriacao(ctx, tr, p.Tipo, resp)
	if err != nil {
		return err
	}
	return a.guardarResultado(ctx, p, bruto, respostas, comentarios)
}

// regraDaCriacao: o plano leva também o método de criar trilhas.
func (a *app) regraDaCriacao(tipo string) (string, error) {
	arquivo := map[string]string{tipoEntrevista: "entrevista.md", tipoMapa: "mapa.md", tipoValidacao: "validar-lista.md", tipoPlano: "plano.md"}[tipo]
	rg, err := a.ia.lerRegra(arquivo)
	if err != nil {
		return "", err
	}
	if tipo != tipoPlano {
		return rg.Texto, nil
	}
	metodo, err := a.ia.lerRegra("criar-trilha.md")
	if err != nil {
		return "", err
	}
	return rg.Texto + "\n\n---\n\n" + metodo.Texto, nil
}

// lerRespostaCriacao confere o que a IA devolveu e, no plano, roda a revisão automática.
func (a *app) lerRespostaCriacao(ctx context.Context, tr trilha, tipo string, resp respostaClaude) ([]byte, []respostaComentario, error) {
	switch tipo {
	case tipoEntrevista:
		var x conteudoEntrevista
		if err := dadosDaResposta(resp, &x); err != nil {
			return nil, nil, err
		}
		if len(x.Perguntas) == 0 {
			x.Suficiente = true
		}
		x.Perguntas, x.Eixos = vazioSeNulo(x.Perguntas), vazioSeNulo(x.Eixos)
		b, err := json.Marshal(x)
		return b, x.Respostas, err
	case tipoMapa:
		var x conteudoMapa
		if err := dadosDaResposta(resp, &x); err != nil {
			return nil, nil, err
		}
		if len(x.Mapa) == 0 {
			return nil, nil, errors.New("o Claude devolveu um mapa vazio")
		}
		x.RecorteSugerido = vazioSeNulo(x.RecorteSugerido)
		b, err := json.Marshal(x)
		return b, x.Respostas, err
	case tipoValidacao:
		var x conteudoValidacao
		if err := dadosDaResposta(resp, &x); err != nil {
			return nil, nil, err
		}
		if len(x.Itens) == 0 {
			return nil, nil, errors.New("o Claude devolveu a lista vazia")
		}
		x.Sugestoes = vazioSeNulo(x.Sugestoes)
		b, err := json.Marshal(x)
		return b, x.Respostas, err
	default:
		var x conteudoPlano
		if err := dadosDaResposta(resp, &x); err != nil {
			return nil, nil, err
		}
		if len(x.Trilha) == 0 || string(x.Trilha) == "null" {
			return nil, nil, errors.New("o Claude devolveu um plano vazio")
		}
		x.Revisao, x.Links = a.revisarPlano(ctx, tr, x)
		b, err := json.Marshal(x)
		return b, x.Respostas, err
	}
}

// guardarResultado: a mesma passagem da lição (gerando → aguardando, a versão anterior fica
// substituída, a resposta da IA a cada comentário vira comentário dela).
func (a *app) guardarResultado(ctx context.Context, p proposta, bruto []byte, respostas []respostaComentario, comentarios map[int64]bool) error {
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
	for _, rc := range respostas {
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

// ---------- revisão automática do plano (critérios A de regras/criar-trilha.md) ----------

func (a *app) revisarPlano(ctx context.Context, tr trilha, cp conteudoPlano) ([]aviso, []linkConferido) {
	var avisos []aviso
	pk, err := a.planoComoPacote(ctx, tr.ID, cp)
	if err == nil {
		err = validarPacote(&pk)
	}
	if err != nil {
		return []aviso{{Criterio: "formato", Texto: err.Error(), Grave: true}}, nil
	}
	docs := map[string]bool{}
	for _, d := range tr.Documentos {
		docs[d.Slug] = true
	}
	var urls []string
	for mi, m := range pk.Trilha.Modulos {
		if n := len(m.Etapas); n < 2 || n > 6 {
			avisos = append(avisos, aviso{Criterio: "C4", Onde: fmt.Sprintf("modulo:%d", mi+1),
				Texto: fmt.Sprintf("o módulo \"%s\" tem %d etapa(s); o método pede de 2 a 6", m.Titulo, n)})
		}
		for _, e := range m.Etapas {
			onde := "etapa:" + e.Chave
			if strings.TrimSpace(e.Objetivo) == "" || e.Bloom == "" {
				avisos = append(avisos, aviso{Criterio: "C2", Onde: onde, Texto: "etapa sem objetivo ou sem nível de Bloom"})
			}
			if e.Minutos < 20 || e.Minutos > 90 {
				avisos = append(avisos, aviso{Criterio: "C3", Onde: onde, Texto: fmt.Sprintf("%d min; o método pede de 20 a 90", e.Minutos)})
			}
			if strings.TrimSpace(e.Criterio) == "" {
				avisos = append(avisos, aviso{Criterio: "C15", Onde: onde, Texto: "etapa sem critério de conclusão"})
			}
			principal := false
			for _, mt := range e.Materiais {
				if mt.Papel == "principal" {
					principal = true
				}
				switch mt.Tipo {
				case "documento":
					if !docs[mt.Referencia] {
						avisos = append(avisos, aviso{Criterio: "C10", Onde: onde, Texto: fmt.Sprintf("o documento \"%s\" não existe nesta trilha", mt.Referencia)})
					}
				case "web":
					urls = append(urls, mt.Referencia)
				}
			}
			if !principal {
				avisos = append(avisos, aviso{Criterio: "C9", Onde: onde, Texto: "etapa sem material principal"})
			}
		}
	}
	var links []linkConferido
	if a.ia.conferirLinks != nil && len(urls) > 0 {
		links = a.ia.conferirLinks(ctx, strings.Join(urls, "\n"))
		for _, l := range links {
			if l.Situacao == "quebrado" {
				avisos = append(avisos, aviso{Criterio: "C11", Texto: "link quebrado: " + l.URL + entreParenteses(l.Detalhe)})
			}
		}
	}
	return avisos, links
}

func entreParenteses(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return " (" + strings.TrimSpace(s) + ")"
}

// ---------- o pedido.md da criação ----------

func (a *app) montarPedidoCriacao(ctx context.Context, tr trilha, c criacao, p proposta) (map[string]string, map[int64]bool, error) {
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, err
	}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: %s\n\n", strings.ToLower(titulosCriacao[p.Tipo]))

	escreva("## O que o Lucas quer aprender\n\n- Tema: %s\n", c.Tema)
	escreva("- O que ele quer com isto: %s\n", padrao(c.Pedido, "(não disse)"))
	if c.Entrada == "lista" {
		escreva("- Forma de criar: ele deu a lista, na ordem dele (a IA só aponta o que falta)\n")
	} else {
		escreva("- Forma de criar: a IA monta (entrevista → mapa completo → ele escolhe o recorte → plano)\n")
	}
	escreva("- Profundidade escolhida: %s\n", nomesProfundidade[tr.Profundidade])
	if tr.NivelPartida != "" {
		escreva("- Nível de partida: %s\n", tr.NivelPartida)
	}
	if tr.NivelAlvo != "" {
		escreva("- Nível alvo: %s\n", tr.NivelAlvo)
	}
	if tr.Objetivo != "" {
		escreva("- Objetivo da trilha: %s\n", tr.Objetivo)
	}

	if c.Entrada == "lista" {
		escreva("\n## A lista dele\n\n")
		for i, item := range itensDaLista(c.Lista) {
			escreva("%d. %s\n", i+1, item)
		}
	}

	if len(c.Rodadas) > 0 {
		escreva("\n## A entrevista até aqui\n\n")
		for i, rd := range c.Rodadas {
			escreva("### Rodada %d\n\n", i+1)
			resp := map[string]resposta{}
			for _, x := range rd.Respostas {
				resp[x.Pergunta] = x
			}
			for _, q := range rd.Perguntas {
				escreva("- **%s** (%s) %s\n", q.ID, q.Texto, descreverResposta(resp[q.ID]))
			}
			if rd.Comentario != "" {
				escreva("- Comentário dele: %s\n", rd.Comentario)
			}
			escreva("\n")
		}
	}

	if len(tr.Documentos) > 0 {
		escreva("\n## Documentos da trilha\n\nMaterial próprio dele, na pasta `documentos` (adicionada ao seu acesso; procure-a com Glob se o caminho relativo não abrir). Leia antes de planejar:\n\n")
		for _, d := range tr.Documentos {
			escreva("- `documentos/%s.md` — %s\n", d.Slug, d.Titulo)
		}
	}

	arquivos := map[string]string{}
	switch p.Tipo {
	case tipoEntrevista:
		limite := tamanhosEntrevista[c.Entrevista]
		feitas := 0
		for _, rd := range c.Rodadas {
			feitas += len(rd.Perguntas)
		}
		n := len(c.Rodadas) + 1
		escreva("\n## Esta rodada\n\nEntrevista %s. Esta é a rodada %d de no máximo %d; já foram feitas %d pergunta(s), e restam até %d.",
			limite.Nome, n, limite.Rodadas, feitas, max(limite.Perguntas-feitas, 0))
		if n >= limite.Rodadas {
			escreva(" É a **última rodada**: depois dela vem o mapa.")
		}
		if n == 1 {
			escreva(" Faça a pré-pesquisa antes de perguntar (primeira rodada).")
		}
		escreva("\n")
	case tipoMapa:
		escreva("\n## O que fazer agora\n\nMonte o mapa completo do assunto e sugira o recorte, pela entrevista.\n")
	case tipoValidacao:
		escreva("\n## O que fazer agora\n\nConfira a lista e aponte o que falta. A ordem é dele.\n")
	case tipoPlano:
		if err := a.escreverBaseDoPlano(ctx, &s, c); err != nil {
			return nil, nil, err
		}
		if len(tr.Modulos) > 0 {
			escreva("\nA trilha já tem estrutura (em `trilha-atual.json`): este plano vai substituí-la; mantenha as chaves das etapas que continuarem.\n")
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
			return nil, nil, err
		}
		if err := a.completar(ctx, &ant); err != nil {
			return nil, nil, err
		}
		arquivos["versao-anterior.json"] = string(ant.Conteudo)
		escreva("\n## É um ajuste da versão %d\n\n", ant.Versao)
		escreva("A versão anterior está em `versao-anterior.json`. Parta dela: mude o que os comentários pedem e mantenha o resto (o Lucas vai ver a diferença). Responda **cada** comentário em `respostas`, pelo número: o que mudou e onde, ou por que não mudou.\n\n")
		for _, x := range ant.Comentarios {
			if x.Autor != "lucas" {
				continue
			}
			comentarios[x.ID] = true
			escreva("- Comentário %d", x.ID)
			if x.Ancora != "" {
				escreva(" (em %s)", x.Ancora)
			}
			if x.Citacao != "" {
				escreva(", sobre o trecho: «%s»", x.Citacao)
			}
			escreva(": %s\n", x.Texto)
		}
		if len(comentarios) == 0 {
			escreva("(Sem comentários: o pedido acima diz o que ajustar.)\n")
		}
	}
	if p.Tipo == tipoPlano && len(tr.Modulos) > 0 {
		b, err := json.MarshalIndent(tr.Modulos, "", "  ")
		if err != nil {
			return nil, nil, err
		}
		arquivos["trilha-atual.json"] = string(b)
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, nil
}

func descreverResposta(x resposta) string {
	var partes []string
	if len(x.Escolhas) > 0 {
		partes = append(partes, "escolheu: "+strings.Join(x.Escolhas, "; "))
	}
	if x.Outro != "" {
		partes = append(partes, "escreveu: "+x.Outro)
	}
	if x.Sugira {
		partes = append(partes, "pediu: \"não sei, sugira\" (escolha você e diga a escolha)")
	}
	if len(partes) == 0 {
		return "→ não respondeu"
	}
	return "→ " + strings.Join(partes, " · ")
}

// escreverBaseDoPlano: o recorte do mapa (A) ou a lista com as sugestões aceitas (B).
func (a *app) escreverBaseDoPlano(ctx context.Context, s *strings.Builder, c criacao) error {
	escreva := func(format string, args ...any) { fmt.Fprintf(s, format, args...) }
	if c.Entrada == "lista" {
		var cv conteudoValidacao
		if c.Escolhas.ValidacaoID > 0 {
			v, err := a.propostaPorID(ctx, c.Escolhas.ValidacaoID)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(v.Conteudo, &cv); err != nil {
				return err
			}
		}
		escreva("\n## A lista, como a IA entendeu (use estas chaves, nesta ordem)\n\n")
		for _, it := range cv.Itens {
			escreva("- `%s` — %s%s\n", it.Chave, it.Titulo, entreParenteses(it.Nota))
		}
		aceitas := map[string]bool{}
		for _, id := range c.Escolhas.Aceitas {
			aceitas[id] = true
		}
		escreva("\n## Sugestões que ele aceitou (aplique)\n\n")
		n := 0
		for _, sg := range cv.Sugestoes {
			if aceitas[sg.ID] {
				escreva("- %s\n", descreverSugestao(sg))
				n++
			}
		}
		if n == 0 {
			escreva("(nenhuma)\n")
		}
		escreva("\n## Sugestões que ele recusou (não aplique)\n\n")
		n = 0
		for _, sg := range cv.Sugestoes {
			if !aceitas[sg.ID] {
				escreva("- %s\n", descreverSugestao(sg))
				n++
			}
		}
		if n == 0 {
			escreva("(nenhuma)\n")
		}
		return nil
	}
	var cm conteudoMapa
	if c.Escolhas.MapaID > 0 {
		m, err := a.propostaPorID(ctx, c.Escolhas.MapaID)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(m.Conteudo, &cm); err != nil {
			return err
		}
	}
	no := map[string]bool{}
	for _, k := range c.Escolhas.Recorte {
		no[k] = true
	}
	escreva("\n## O mapa aprovado e o recorte que ele escolheu\n\nAs etapas marcadas com ✔ entram na trilha (mantenha as chaves); as outras ficam de fora e servem de contexto (o que vem antes e depois).\n\n")
	for _, m := range cm.Mapa {
		escreva("- %s\n", m.Titulo)
		for _, e := range m.Etapas {
			marca := "  "
			if no[e.Chave] {
				marca = "✔ "
			}
			escreva("  - %s`%s` %s — %s (%s)\n", marca, e.Chave, e.Titulo, e.Resumo, e.Nivel)
		}
	}
	return nil
}

func descreverSugestao(sg sugestao) string {
	switch sg.Acao {
	case "inserir":
		return fmt.Sprintf("inserir \"%s\" antes de `%s` — %s", sg.Titulo, sg.AntesDe, sg.Motivo)
	case "dividir":
		return fmt.Sprintf("dividir `%s`: %s — %s", sg.Item, sg.Titulo, sg.Motivo)
	default:
		return fmt.Sprintf("observação sobre `%s`: %s — %s", sg.Item, sg.Titulo, sg.Motivo)
	}
}

func vazioSeNulo[T any](lista []T) []T {
	if lista == nil {
		return []T{}
	}
	return lista
}

// ehCriacao: os tipos de proposta da criação (alvo = a trilha).
func ehCriacao(tipo string) bool {
	return contem(tiposCriacao, tipo)
}
