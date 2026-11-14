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
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Fase 5 (T26): estudar a etapa. Exercícios (escolha e completar) e simulado (perguntas abertas)
// são propostas da IA como a lição: o Lucas comenta, pede ajuste e aprova, com o gabarito
// escondido na revisão. Aprovados, ficam no conteúdo (trilhas/<slug>/exercicios/<chave>.json e
// .../simulado/<chave>.json). Os exercícios são corrigidos na hora pelo servidor, sem a IA; o
// simulado vai para a fila e a IA corrige. O critério automático é a nota dos exercícios
// (Ajustes, padrão 80%); concluir continua sendo um toque do Lucas.

const (
	tipoExercicios = "exercicios"
	tipoSimulado   = "simulado"
	tipoCorrecao   = "correcao"
)

// tiposEstudo: as propostas da etapa além da lição.
var tiposEstudo = []string{tipoExercicios, tipoSimulado}

var titulosEstudo = map[string]string{
	"licao":        "Lição",
	tipoExercicios: "Exercícios",
	tipoSimulado:   "Simulado",
	tipoMateriais:  "Material sugerido",
}

type exercicio struct {
	Tipo         string   `json:"tipo"` // escolha · completar
	Enunciado    string   `json:"enunciado"`
	Alternativas []string `json:"alternativas,omitempty"`
	Correta      int      `json:"correta"`           // escolha: índice da alternativa certa (0 = a primeira)
	Aceitas      []string `json:"aceitas,omitempty"` // completar: as respostas aceitas
	Explicacao   string   `json:"explicacao"`
	Conceito     string   `json:"conceito"`
}

type conteudoExercicios struct {
	Exercicios []exercicio          `json:"exercicios"`
	Notas      string               `json:"notas,omitempty"`
	Respostas  []respostaComentario `json:"respostas,omitempty"`
}

type perguntaSimulado struct {
	Pergunta string `json:"pergunta"`
	Esperado string `json:"esperado"` // os pontos que uma boa resposta traz (o gabarito da correção)
	Conceito string `json:"conceito"`
}

type conteudoSimulado struct {
	Perguntas []perguntaSimulado   `json:"perguntas"`
	Notas     string               `json:"notas,omitempty"`
	Respostas []respostaComentario `json:"respostas,omitempty"`
}

var esquemasEstudo = map[string]string{
	tipoExercicios: `{"type":"object","properties":{
"exercicios":{"type":"array","items":{"type":"object","properties":{
  "tipo":{"type":"string","enum":["escolha","completar"]},
  "enunciado":{"type":"string"},
  "alternativas":{"type":"array","items":{"type":"string"},"description":"só em escolha: de 3 a 5"},
  "correta":{"type":"integer","description":"só em escolha: o índice da alternativa certa, a partir de 0"},
  "aceitas":{"type":"array","items":{"type":"string"},"description":"só em completar: as respostas aceitas (variações de escrita)"},
  "explicacao":{"type":"string","description":"por que a certa é certa (e as outras não), mostrada depois de responder"},
  "conceito":{"type":"string","description":"o conceito da lição que o exercício testa"}},
  "required":["tipo","enunciado","explicacao","conceito"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["exercicios","notas","respostas"]}`,
	tipoSimulado: `{"type":"object","properties":{
"perguntas":{"type":"array","items":{"type":"object","properties":{
  "pergunta":{"type":"string"},
  "esperado":{"type":"string","description":"os pontos que uma boa resposta traz; é o gabarito da correção"},
  "conceito":{"type":"string"}},
  "required":["pergunta","esperado","conceito"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["perguntas","notas","respostas"]}`,
}

const esquemaCorrecao = `{"type":"object","properties":{
"correcoes":{"type":"array","items":{"type":"object","properties":{
  "pergunta":{"type":"integer","description":"o número da pergunta no pedido"},
  "nota":{"type":"integer","description":"de 0 a 100"},
  "veredito":{"type":"string","enum":["certo","parcial","errado"]},
  "comentario":{"type":"string","description":"o que está certo e o que está errado na resposta"},
  "faltou":{"type":"string","description":"o que uma resposta completa traria a mais; vazio se nada"}},
  "required":["pergunta","nota","veredito","comentario","faltou"]}},
"resumo":{"type":"string","description":"o que o Lucas domina e o que revisar, em duas ou três frases"}},
"required":["correcoes","resumo"]}`

func ehEstudo(tipo string) bool { return contem(tiposEstudo, tipo) }

func caminhoEstudo(slug, tipo, chave string) string {
	return "trilhas/" + slug + "/" + tipo + "/" + chave + ".json"
}

// ---------- o que a etapa tem para estudar ----------

func (a *app) lerExercicios(slug, chave string) []exercicio {
	var c conteudoExercicios
	if texto, err := a.conteudo.ler(caminhoEstudo(slug, tipoExercicios, chave)); err == nil {
		json.Unmarshal([]byte(texto), &c)
	}
	return vazioSeNulo(c.Exercicios)
}

func (a *app) lerSimulado(slug, chave string) []perguntaSimulado {
	var c conteudoSimulado
	if texto, err := a.conteudo.ler(caminhoEstudo(slug, tipoSimulado, chave)); err == nil {
		json.Unmarshal([]byte(texto), &c)
	}
	return vazioSeNulo(c.Perguntas)
}

type tentativa struct {
	ID           int64           `json:"id"`
	Tipo         string          `json:"tipo"`   // exercicios · simulado
	Estado       string          `json:"estado"` // corrigida · corrigindo · falhou
	Nota         int             `json:"nota"`   // 0 a 100
	Acertos      int             `json:"acertos"`
	Total        int             `json:"total"`
	Respostas    json.RawMessage `json:"respostas"`
	Resultado    json.RawMessage `json:"resultado,omitempty"`
	Erro         string          `json:"erro"`
	TarefaID     int64           `json:"tarefa_id,omitempty"`
	CriadoEm     string          `json:"criado_em"`
	AtualizadoEm string          `json:"atualizado_em"`
}

const (
	tentCorrigida  = "corrigida"
	tentCorrigindo = "corrigindo"
	tentFalhou     = "falhou"
)

const colunasTentativa = `id, tipo, estado, nota, acertos, total, respostas, resultado, erro, COALESCE(tarefa_id, 0), criado_em, atualizado_em`

func lerTentativa(row interface{ Scan(...any) error }) (tentativa, error) {
	var t tentativa
	var respostas, resultado string
	err := row.Scan(&t.ID, &t.Tipo, &t.Estado, &t.Nota, &t.Acertos, &t.Total, &respostas, &resultado, &t.Erro, &t.TarefaID, &t.CriadoEm, &t.AtualizadoEm)
	t.Respostas = json.RawMessage(padrao(respostas, "[]"))
	if resultado != "" {
		t.Resultado = json.RawMessage(resultado)
	}
	return t, err
}

func (a *app) tentativas(ctx context.Context, trilhaID int64, chave string) ([]tentativa, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasTentativa+` FROM tentativas
		WHERE trilha_id = ? AND etapa_chave = ? ORDER BY id DESC LIMIT 30`, trilhaID, chave)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []tentativa{}
	for rows.Next() {
		t, err := lerTentativa(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}

// notaMinima: o critério automático (Ajustes); padrão 80%.
func (a *app) notaMinima(ctx context.Context) int {
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return 80
	}
	if n, err := strconv.Atoi(cfg.NotaMinima); err == nil && n > 0 {
		return n
	}
	return 80
}

// Para a tela: os exercícios e as perguntas sem o gabarito (ele volta na correção).
type exercicioSemGabarito struct {
	Tipo         string   `json:"tipo"`
	Enunciado    string   `json:"enunciado"`
	Alternativas []string `json:"alternativas,omitempty"`
	Conceito     string   `json:"conceito"`
}

type perguntaSemGabarito struct {
	Pergunta string `json:"pergunta"`
	Conceito string `json:"conceito"`
}

func (a *app) handleEstudo(w http.ResponseWriter, r *http.Request) {
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	slug := a.slugDaTrilha(r.Context(), e.TrilhaID)
	exs := []exercicioSemGabarito{}
	for _, x := range a.lerExercicios(slug, e.Chave) {
		exs = append(exs, exercicioSemGabarito{x.Tipo, x.Enunciado, x.Alternativas, x.Conceito})
	}
	pergs := []perguntaSemGabarito{}
	for _, p := range a.lerSimulado(slug, e.Chave) {
		pergs = append(pergs, perguntaSemGabarito{p.Pergunta, p.Conceito})
	}
	lista, err := a.tentativas(r.Context(), e.TrilhaID, e.Chave)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	minima := a.notaMinima(r.Context())
	atingido := false
	for _, t := range lista {
		if t.Tipo == tipoExercicios {
			atingido = t.Nota >= minima
			break
		}
	}
	envios, err := a.enviosFlashcards(r.Context(), e.TrilhaID, e.Chave)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"exercicios": exs, "simulado": pergs, "tentativas": lista,
		"nota_minima": minima, "criterio_atingido": atingido,
		"flashcards": map[string]any{"disponivel": a.integra.flashcards != "", "link": a.integra.flashcardsLink, "envios": envios},
	})
}

func (a *app) etapaDaRota(w http.ResponseWriter, r *http.Request) (etapa, bool) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return etapa{}, false
	}
	e, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return e, false
	}
	return e, true
}

// ---------- pedir exercícios ou simulado ----------

func (a *app) handleGerarEstudo(tipo string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var corpo struct {
			Pedido string `json:"pedido"`
		}
		if err := lerCorpo(r, &corpo); err != nil {
			a.responderErro(w, err)
			return
		}
		e, ok := a.etapaDaRota(w, r)
		if !ok {
			return
		}
		if !e.TemLicao {
			a.responderErro(w, erroDeEntrada{errors.New("primeiro a lição: os exercícios e o simulado saem dela")})
			return
		}
		p, err := a.novaProposta(r.Context(), e.TrilhaID, e.Chave, tipo, strings.TrimSpace(corpo.Pedido), 0, titulosEstudo[tipo]+": "+e.Titulo)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		responderJSON(w, http.StatusCreated, p)
	}
}

// gerarEstudo é a tarefa: monta o pedido, chama a IA e guarda a proposta.
func (a *app) gerarEstudo(ctx context.Context, t *tarefa) error {
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
	arquivos, comentarios, err := a.montarPedidoEstudo(ctx, tr, p)
	if err != nil {
		return err
	}
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra(p.Tipo + ".md")
	if err != nil {
		return err
	}
	o := map[string]string{tipoExercicios: "os exercícios", tipoSimulado: "as perguntas do simulado"}[p.Tipo]
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read,Glob,Grep", esquema: esquemasEstudo[p.Tipo], modelo: t.Modelo,
		instrucao: "Escreva " + o + " de uma etapa de uma trilha de estudo. Tudo o que você precisa saber está em " +
			"`pedido.md`, nesta pasta, e a lição da etapa em `licao.md`: leia os dois inteiros primeiro.",
	})
	if err != nil {
		return err
	}
	var bruto []byte
	var respostas []respostaComentario
	switch p.Tipo {
	case tipoExercicios:
		var c conteudoExercicios
		if err := dadosDaResposta(resp, &c); err != nil {
			return err
		}
		var descartados int
		c.Exercicios, descartados = limparExercicios(c.Exercicios)
		if len(c.Exercicios) == 0 {
			return errors.New("o Claude não devolveu nenhum exercício bem formado")
		}
		if descartados > 0 {
			c.Notas = strings.TrimSpace(c.Notas + fmt.Sprintf("\n\n(O app tirou %d exercício(s) mal formado(s): sem alternativas, com a certa fora da lista ou sem resposta aceita.)", descartados))
		}
		respostas = c.Respostas
		bruto, err = json.Marshal(c)
	case tipoSimulado:
		var c conteudoSimulado
		if err := dadosDaResposta(resp, &c); err != nil {
			return err
		}
		var limpas []perguntaSimulado
		for _, q := range c.Perguntas {
			q.Pergunta, q.Esperado, q.Conceito = strings.TrimSpace(q.Pergunta), strings.TrimSpace(q.Esperado), strings.TrimSpace(q.Conceito)
			if q.Pergunta != "" && q.Esperado != "" {
				limpas = append(limpas, q)
			}
		}
		if len(limpas) == 0 {
			return errors.New("o Claude não devolveu nenhuma pergunta")
		}
		c.Perguntas = limpas
		respostas = c.Respostas
		bruto, err = json.Marshal(c)
	}
	if err != nil {
		return err
	}
	return a.guardarProposta(ctx, p, string(bruto), respostas, comentarios)
}

// guardarProposta: a versão nova fica aguardando, a anterior vira substituída, e a resposta da IA
// a cada comentário fica na versão nova.
func (a *app) guardarProposta(ctx context.Context, p proposta, conteudo string, respostas []respostaComentario, comentarios map[int64]bool) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	momento := agora()
	if _, err := tx.ExecContext(ctx, `UPDATE propostas SET estado = ?, conteudo = ?, atualizado_em = ? WHERE id = ? AND estado = ?`,
		propAguardando, conteudo, momento, p.ID, propGerando); err != nil {
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

// limparExercicios tira os mal formados (que não dariam para corrigir) e devolve quantos saíram.
func limparExercicios(lista []exercicio) ([]exercicio, int) {
	var bons []exercicio
	for _, x := range lista {
		x.Enunciado, x.Explicacao, x.Conceito = strings.TrimSpace(x.Enunciado), strings.TrimSpace(x.Explicacao), strings.TrimSpace(x.Conceito)
		switch x.Tipo {
		case "escolha":
			x.Aceitas = nil
			if len(x.Alternativas) < 2 || x.Correta < 0 || x.Correta >= len(x.Alternativas) {
				continue
			}
		case "completar":
			x.Alternativas, x.Correta = nil, 0
			var aceitas []string
			for _, s := range x.Aceitas {
				if normalizarResposta(s) != "" {
					aceitas = append(aceitas, strings.TrimSpace(s))
				}
			}
			if len(aceitas) == 0 {
				continue
			}
			x.Aceitas = aceitas
		default:
			continue
		}
		if x.Enunciado == "" {
			continue
		}
		bons = append(bons, x)
	}
	return bons, len(lista) - len(bons)
}

// montarPedidoEstudo escreve o pedido.md, a lição e o que a etapa já tem (para não repetir).
func (a *app) montarPedidoEstudo(ctx context.Context, tr trilha, p proposta) (map[string]string, map[int64]bool, error) {
	var e etapa
	var modulo string
	for _, m := range tr.Modulos {
		for _, x := range m.Etapas {
			if x.Chave == p.EtapaChave {
				e, modulo = x, m.Titulo
			}
		}
	}
	if e.ID == 0 {
		return nil, nil, errors.New("a etapa desta proposta não existe mais na trilha")
	}
	licao, err := a.conteudo.ler(caminhoLicao(tr.Slug, e.Chave))
	if err != nil || strings.TrimSpace(licao) == "" {
		return nil, nil, errors.New("a etapa não tem lição; os exercícios e o simulado saem dela")
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
	nome := map[string]string{tipoExercicios: "exercícios", tipoSimulado: "simulado"}[p.Tipo]
	escreva("# Pedido: %s da etapa \"%s\"\n\n", nome, e.Titulo)
	escreva("## A trilha\n\n- Título: %s\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
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
	if e.Criterio != "" {
		escreva("- Critério de conclusão: %s\n", e.Criterio)
	}
	escreva("\nA lição aprovada da etapa está em `licao.md`. Tudo o que você perguntar tem de estar nela (ou ser consequência direta dela).\n")

	arquivos := map[string]string{"licao.md": licao}
	switch p.Tipo {
	case tipoExercicios:
		if atuais := a.lerExercicios(tr.Slug, e.Chave); len(atuais) > 0 {
			b, _ := json.MarshalIndent(atuais, "", "  ")
			arquivos["exercicios-atuais.json"] = string(b)
			escreva("\n## A etapa já tem exercícios\n\nEstão em `exercicios-atuais.json`. Este pedido gera um **conjunto novo, que substitui** o atual: mantenha os bons, troque os que o pedido manda trocar e não repita a mesma pergunta com outras palavras.\n")
		}
	case tipoSimulado:
		if atuais := a.lerSimulado(tr.Slug, e.Chave); len(atuais) > 0 {
			escreva("\n## O banco do simulado já tem estas perguntas\n\nAs novas **entram no banco junto com elas**: não repita nenhuma, nem com outras palavras; cubra o que elas não cobrem.\n\n")
			for i, q := range atuais {
				escreva("%d. %s\n", i+1, q.Pergunta)
			}
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
		escreva("A versão anterior está em `versao-anterior.json`. Parta dela: mude o que os comentários pedem e mantenha o resto. Os comentários numa questão dizem `questao:N` (N a partir de 1, na ordem da versão anterior). Responda **cada** comentário em `respostas`, pelo número: o que mudou, ou por que não mudou.\n\n")
		for _, c := range ant.Comentarios {
			if c.Autor != "lucas" {
				continue
			}
			comentarios[c.ID] = true
			escreva("- Comentário %d", c.ID)
			if c.Ancora != "" {
				escreva(" (em %s)", c.Ancora)
			}
			escreva(": %s\n", c.Texto)
		}
		if len(comentarios) == 0 {
			escreva("(Sem comentários: o pedido acima diz o que ajustar.)\n")
		}
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, nil
}

// aprovarEstudo grava os exercícios (substituem) ou as perguntas (entram no banco do simulado).
func (a *app) aprovarEstudo(w http.ResponseWriter, r *http.Request, p proposta) {
	ctx := r.Context()
	if _, err := a.etapaPorChave(ctx, p.TrilhaID, p.EtapaChave); err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa desta proposta não existe mais na trilha")})
		return
	}
	if err := a.semVersaoGerando(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	slug := a.slugDaTrilha(ctx, p.TrilhaID)
	var texto, mensagem string
	switch p.Tipo {
	case tipoExercicios:
		var c conteudoExercicios
		if err := json.Unmarshal(p.Conteudo, &c); err != nil || len(c.Exercicios) == 0 {
			a.responderErro(w, erroDeEntrada{errors.New("a proposta não tem exercícios para gravar")})
			return
		}
		b, _ := json.MarshalIndent(conteudoExercicios{Exercicios: c.Exercicios}, "", "  ")
		texto, mensagem = string(b), fmt.Sprintf("%s: exercícios %s (proposta v%d aprovada)", slug, p.EtapaChave, p.Versao)
	case tipoSimulado:
		var c conteudoSimulado
		if err := json.Unmarshal(p.Conteudo, &c); err != nil || len(c.Perguntas) == 0 {
			a.responderErro(w, erroDeEntrada{errors.New("a proposta não tem perguntas para gravar")})
			return
		}
		banco := a.lerSimulado(slug, p.EtapaChave)
		vistas := map[string]bool{}
		for _, q := range banco {
			vistas[normalizarResposta(q.Pergunta)] = true
		}
		for _, q := range c.Perguntas {
			if !vistas[normalizarResposta(q.Pergunta)] {
				banco = append(banco, q)
				vistas[normalizarResposta(q.Pergunta)] = true
			}
		}
		b, _ := json.MarshalIndent(conteudoSimulado{Perguntas: banco}, "", "  ")
		texto, mensagem = string(b), fmt.Sprintf("%s: simulado %s, %d pergunta(s) no banco (proposta v%d aprovada)", slug, p.EtapaChave, len(banco), p.Versao)
	}
	if err := a.conteudo.gravar(mensagem, map[string]*string{caminhoEstudo(slug, p.Tipo, p.EtapaChave): &texto}); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}

func (a *app) semVersaoGerando(ctx context.Context, p proposta) error {
	var gerando int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM propostas WHERE trilha_id = ? AND etapa_chave = ? AND tipo = ? AND estado = ?`,
		p.TrilhaID, p.EtapaChave, p.Tipo, propGerando).Scan(&gerando); err != nil {
		return err
	}
	if gerando > 0 {
		return erroDeEntrada{errors.New("uma versão nova está sendo gerada; espere ela chegar ou cancele a tarefa antes de aprovar esta")}
	}
	return nil
}

// marcarAprovada: esta fica aprovada e as outras que aguardavam, do mesmo alvo e tipo, substituídas.
func (a *app) marcarAprovada(ctx context.Context, p proposta) error {
	if _, err := a.db.ExecContext(ctx, `UPDATE propostas SET estado = CASE WHEN id = ? THEN ? ELSE ? END, atualizado_em = ?
		WHERE trilha_id = ? AND etapa_chave = ? AND tipo = ? AND (id = ? OR estado = ?)`,
		p.ID, propAprovada, propSubstituida, agora(), p.TrilhaID, p.EtapaChave, p.Tipo, p.ID, propAguardando); err != nil {
		return err
	}
	a.tocarTrilha(ctx, p.TrilhaID)
	return nil
}

// ---------- exercícios: corrigidos na hora ----------

type respostaExercicio struct {
	Escolha *int   `json:"escolha,omitempty"`
	Texto   string `json:"texto,omitempty"`
}

type correcaoExercicio struct {
	Certa      bool   `json:"certa"`
	Enunciado  string `json:"enunciado"`
	Resposta   string `json:"resposta"` // o que o Lucas respondeu, como texto
	Correta    string `json:"correta"`  // a certa, como texto
	Explicacao string `json:"explicacao"`
	Conceito   string `json:"conceito"`
}

func (a *app) handleTentativaExercicios(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Respostas []respostaExercicio `json:"respostas"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	exs := a.lerExercicios(a.slugDaTrilha(r.Context(), e.TrilhaID), e.Chave)
	if len(exs) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa ainda não tem exercícios")})
		return
	}
	if len(corpo.Respostas) != len(exs) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("vieram %d respostas para %d exercícios; recarregue a página", len(corpo.Respostas), len(exs))})
		return
	}
	correcoes, acertos := corrigirExercicios(exs, corpo.Respostas)
	nota := (acertos*100 + len(exs)/2) / len(exs) // arredondada: 2 de 3 = 67%
	respostas, _ := json.Marshal(corpo.Respostas)
	resultado, _ := json.Marshal(correcoes)
	id, err := a.novaTentativa(r.Context(), e, tipoExercicios, tentCorrigida, nota, acertos, len(exs), string(respostas), string(resultado), 0)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderTentativa(w, r, id, http.StatusCreated)
}

func corrigirExercicios(exs []exercicio, respostas []respostaExercicio) ([]correcaoExercicio, int) {
	var lista []correcaoExercicio
	acertos := 0
	for i, x := range exs {
		c := correcaoExercicio{Enunciado: x.Enunciado, Explicacao: x.Explicacao, Conceito: x.Conceito}
		resp := respostas[i]
		switch x.Tipo {
		case "escolha":
			c.Correta = x.Alternativas[x.Correta]
			if resp.Escolha != nil && *resp.Escolha >= 0 && *resp.Escolha < len(x.Alternativas) {
				c.Resposta = x.Alternativas[*resp.Escolha]
				c.Certa = *resp.Escolha == x.Correta
			}
		case "completar":
			c.Correta = x.Aceitas[0]
			c.Resposta = strings.TrimSpace(resp.Texto)
			dada := normalizarResposta(resp.Texto)
			for _, aceita := range x.Aceitas {
				if dada != "" && dada == normalizarResposta(aceita) {
					c.Certa = true
				}
			}
		}
		if c.Certa {
			acertos++
		}
		lista = append(lista, c)
	}
	return lista, acertos
}

// normalizarResposta: sem acento, caixa, pontuação nas pontas e espaços repetidos ("  NAT. " = "nat").
func normalizarResposta(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	campos := strings.Fields(b.String())
	return strings.TrimFunc(strings.Join(campos, " "), func(r rune) bool { return unicode.IsPunct(r) && r != '%' })
}

func (a *app) novaTentativa(ctx context.Context, e etapa, tipo, estado string, nota, acertos, total int, respostas, resultado string, tarefaID int64) (int64, error) {
	var tarefa any
	if tarefaID > 0 {
		tarefa = tarefaID
	}
	momento := agora()
	res, err := a.db.ExecContext(ctx, `INSERT INTO tentativas (trilha_id, etapa_chave, tipo, estado, nota, acertos, total, respostas, resultado, tarefa_id, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.TrilhaID, e.Chave, tipo, estado, nota, acertos, total, respostas, resultado, tarefa, momento, momento)
	if err != nil {
		return 0, err
	}
	// Estudar a etapa é começá-la.
	a.db.ExecContext(ctx, `UPDATE etapas SET status = 'em_andamento', iniciado_em = CASE WHEN iniciado_em = '' THEN ? ELSE iniciado_em END
		WHERE id = ? AND status = 'disponivel'`, momento, e.ID)
	return res.LastInsertId()
}

func (a *app) responderTentativa(w http.ResponseWriter, r *http.Request, id int64, status int) {
	t, err := lerTentativa(a.db.QueryRowContext(r.Context(), `SELECT `+colunasTentativa+` FROM tentativas WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = errNaoEncontrou
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, status, t)
}

// ---------- simulado: a IA corrige ----------

type respostaSimulado struct {
	Pergunta int    `json:"pergunta"` // índice no banco (0 = a primeira)
	Resposta string `json:"resposta"`
}

// O que fica guardado em cada tentativa do simulado: a pergunta e o gabarito daquele momento
// (o banco cresce e pode mudar depois).
type respondidaSimulado struct {
	Pergunta string `json:"pergunta"`
	Esperado string `json:"esperado"`
	Conceito string `json:"conceito"`
	Resposta string `json:"resposta"`
}

type correcaoSimulado struct {
	Correcoes []struct {
		Pergunta   int    `json:"pergunta"`
		Nota       int    `json:"nota"`
		Veredito   string `json:"veredito"`
		Comentario string `json:"comentario"`
		Faltou     string `json:"faltou"`
	} `json:"correcoes"`
	Resumo string `json:"resumo"`
}

func (a *app) handleTentativaSimulado(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Respostas []respostaSimulado `json:"respostas"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	banco := a.lerSimulado(a.slugDaTrilha(ctx, e.TrilhaID), e.Chave)
	var respondidas []respondidaSimulado
	usadas := map[int]bool{}
	for _, x := range corpo.Respostas {
		x.Resposta = strings.TrimSpace(x.Resposta)
		if x.Resposta == "" || usadas[x.Pergunta] {
			continue
		}
		if x.Pergunta < 0 || x.Pergunta >= len(banco) {
			a.responderErro(w, erroDeEntrada{errors.New("pergunta fora do banco do simulado; recarregue a página")})
			return
		}
		if len([]rune(x.Resposta)) > 8000 {
			a.responderErro(w, erroDeEntrada{errors.New("uma resposta passou de 8000 caracteres")})
			return
		}
		usadas[x.Pergunta] = true
		q := banco[x.Pergunta]
		respondidas = append(respondidas, respondidaSimulado{q.Pergunta, q.Esperado, q.Conceito, x.Resposta})
	}
	if len(respondidas) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("responda ao menos uma pergunta")})
		return
	}
	var corrigindo int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tentativas WHERE trilha_id = ? AND etapa_chave = ? AND estado = ?`,
		e.TrilhaID, e.Chave, tentCorrigindo).Scan(&corrigindo); err != nil {
		a.responderErro(w, err)
		return
	}
	if corrigindo > 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a IA ainda está corrigindo o simulado anterior desta etapa")})
		return
	}
	modelo := a.modeloPara(ctx, tipoCorrecao)
	respostas, _ := json.Marshal(respondidas)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoCorrecao, 0, "Correção do simulado: "+e.Titulo, modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	id, err := a.novaTentativa(ctx, e, tipoSimulado, tentCorrigindo, 0, 0, len(respondidas), string(respostas), "", tarefaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	a.responderTentativa(w, r, id, http.StatusCreated)
}

// corrigirSimulado é a tarefa: a IA lê a lição, o gabarito e as respostas e dá a nota de cada uma.
func (a *app) corrigirSimulado(ctx context.Context, t *tarefa) error {
	var tid, trilhaID int64
	var chave, respostas string
	err := a.db.QueryRowContext(ctx, `SELECT id, trilha_id, etapa_chave, respostas FROM tentativas WHERE tarefa_id = ?`, t.ID).
		Scan(&tid, &trilhaID, &chave, &respostas)
	if err != nil {
		// A tentativa pode ter saído junto com a etapa ou a trilha.
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("a tentativa desta correção não existe mais")
		}
		return err
	}
	var respondidas []respondidaSimulado
	if err := json.Unmarshal([]byte(respostas), &respondidas); err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, trilhaID)
	if err != nil {
		return err
	}
	e, err := a.etapaPorChave(ctx, trilhaID, chave)
	if err != nil {
		return errors.New("a etapa desta correção não existe mais")
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	licao, _ := a.conteudo.ler(caminhoLicao(tr.Slug, chave))
	var s strings.Builder
	fmt.Fprintf(&s, "# Pedido: correção do simulado da etapa \"%s\"\n\n- Trilha: %s\n", e.Titulo, tr.Titulo)
	if e.Objetivo != "" {
		fmt.Fprintf(&s, "- Objetivo da etapa: %s\n", e.Objetivo)
	}
	fmt.Fprintf(&s, "\nA lição da etapa está em `licao.md`. Corrija cada resposta abaixo pela pergunta e pelo gabarito (o que uma boa resposta traz), e use a lição para conferir o que o Lucas escreveu além do gabarito.\n")
	for i, x := range respondidas {
		fmt.Fprintf(&s, "\n## Pergunta %d\n\n%s\n\n**Gabarito:** %s\n\n**Resposta do Lucas:**\n\n%s\n", i+1, x.Pergunta, x.Esperado, x.Resposta)
	}
	if err := os.WriteFile(filepath.Join(pasta, "pedido.md"), []byte(s.String()), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pasta, "licao.md"), []byte(licao), 0o644); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("corrigir-simulado.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read", esquema: esquemaCorrecao, modelo: t.Modelo,
		instrucao: "Corrija as respostas do Lucas a um simulado. Tudo está em `pedido.md` e `licao.md`, nesta pasta: leia os dois inteiros primeiro.",
	})
	if err != nil {
		return err
	}
	var c correcaoSimulado
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	// Uma correção por pergunta, na ordem do pedido; a que faltar conta zero.
	porPergunta := map[int]int{}
	for i, x := range c.Correcoes {
		if x.Pergunta >= 1 && x.Pergunta <= len(respondidas) {
			porPergunta[x.Pergunta] = i
		}
	}
	type itemCorrigido struct {
		respondidaSimulado
		Nota       int    `json:"nota"`
		Veredito   string `json:"veredito"`
		Comentario string `json:"comentario"`
		Faltou     string `json:"faltou"`
	}
	itens := []itemCorrigido{}
	soma, certas := 0, 0
	for i, x := range respondidas {
		it := itemCorrigido{respondidaSimulado: x, Veredito: "errado", Comentario: "A IA não corrigiu esta resposta."}
		if j, ok := porPergunta[i+1]; ok {
			cc := c.Correcoes[j]
			it.Nota, it.Veredito, it.Comentario, it.Faltou = min(100, max(0, cc.Nota)), cc.Veredito, textoDaIA(cc.Comentario), textoDaIA(cc.Faltou)
		}
		soma += it.Nota
		if it.Veredito == "certo" {
			certas++
		}
		itens = append(itens, it)
	}
	resultado, _ := json.Marshal(map[string]any{"itens": itens, "resumo": textoDaIA(c.Resumo)})
	_, err = a.db.ExecContext(ctx, `UPDATE tentativas SET estado = ?, nota = ?, acertos = ?, resultado = ?, atualizado_em = ? WHERE id = ?`,
		tentCorrigida, soma/len(respondidas), certas, string(resultado), agora(), tid)
	return err
}

// ---------- Flashcards (POST /api/textos de lá, F44) ----------

type integracoes struct {
	flashcards     string // FLASHCARDS_URL (rede interna); vazio = não configurado
	flashcardsLink string // FLASHCARDS_LINK, para os links da tela
	trilhaLink     string // TRILHA_LINK, o endereço deste app (o link de volta de cada cartão)
	biblioteca     string // BIBLIOTECA_URL (fase 6, material.go)
	bibliotecaLink string // BIBLIOTECA_LINK
	acervo         string // ACERVO_URL
	acervoLink     string // ACERVO_LINK
}

type envioFlashcards struct {
	ID        int64  `json:"id"`
	GeracaoID int64  `json:"geracao_id"`
	Erro      string `json:"erro"`
	CriadoEm  string `json:"criado_em"`
}

func (a *app) enviosFlashcards(ctx context.Context, trilhaID int64, chave string) ([]envioFlashcards, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, geracao_id, erro, criado_em FROM envios_flashcards
		WHERE trilha_id = ? AND etapa_chave = ? ORDER BY id DESC LIMIT 10`, trilhaID, chave)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []envioFlashcards{}
	for rows.Next() {
		var x envioFlashcards
		if err := rows.Scan(&x.ID, &x.GeracaoID, &x.Erro, &x.CriadoEm); err != nil {
			return nil, err
		}
		lista = append(lista, x)
	}
	return lista, rows.Err()
}

func (a *app) handleBaralhos(w http.ResponseWriter, r *http.Request) {
	var baralhos []struct {
		ID   string `json:"id"`
		Nome string `json:"nome"`
	}
	if status, err := chamarApp(r.Context(), a.integra.flashcards, "Flashcards", "GET", "/api/baralhos", nil, &baralhos); err != nil {
		erroJSON(w, statusOu(status, http.StatusBadGateway), err.Error())
		return
	}
	if baralhos == nil {
		baralhos = baralhos[:0]
	}
	responderJSON(w, http.StatusOK, baralhos)
}

func (a *app) handleEnviarFlashcards(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		BaralhoID string `json:"baralho_id"`
		Baralho   string `json:"baralho"`
		Tipo      string `json:"tipo"` // vazio (perguntas) ou "leitura"
		Instrucao string `json:"instrucao"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tr, err := a.trilhaCompleta(ctx, e.TrilhaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	licao, err := a.conteudo.ler(caminhoLicao(tr.Slug, e.Chave))
	if err != nil || strings.TrimSpace(licao) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa ainda não tem lição para virar cartões")})
		return
	}
	if corpo.Tipo != "" && corpo.Tipo != "leitura" {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("tipo de cartão desconhecido: %q", corpo.Tipo)})
		return
	}
	link := fmt.Sprintf("%s/#/trilhas/%d/etapas/%d", a.integra.trilhaLink, e.TrilhaID, e.ID)
	modulo := ""
	for _, m := range tr.Modulos {
		for _, x := range m.Etapas {
			if x.ID == e.ID {
				modulo = m.Titulo
			}
		}
	}
	contexto := fmt.Sprintf("Lição da etapa \"%s\" (módulo \"%s\") da trilha de estudo \"%s\".", e.Titulo, modulo, tr.Titulo)
	if e.Objetivo != "" {
		contexto += " Objetivo da etapa: " + e.Objetivo
	}
	envio := map[string]any{
		"titulo": tr.Titulo + " › " + e.Titulo, "contexto": contexto, "texto": licao, "link": link,
		"secoes": secoesDaLicao(licao, link), "origem": "trilha",
		"baralho_id": strings.TrimSpace(corpo.BaralhoID), "baralho": strings.TrimSpace(corpo.Baralho),
		"tipo": corpo.Tipo, "instrucao": strings.TrimSpace(corpo.Instrucao),
	}
	var g struct {
		ID int64 `json:"id"`
	}
	status, err := chamarApp(ctx, a.integra.flashcards, "Flashcards", "POST", "/api/textos", envio, &g)
	if err != nil && (status == http.StatusBadRequest || status == http.StatusRequestEntityTooLarge || a.integra.flashcards == "") {
		a.responderErro(w, erroDeEntrada{err}) // o pedido é que não serve: nada a anotar
		return
	}
	erro := ""
	if err != nil {
		erro = err.Error()
	}
	if _, err := a.db.ExecContext(ctx, `INSERT INTO envios_flashcards (trilha_id, etapa_chave, geracao_id, erro, criado_em) VALUES (?, ?, ?, ?, ?)`,
		e.TrilhaID, e.Chave, g.ID, erro, agora()); err != nil {
		a.responderErro(w, err)
		return
	}
	if erro != "" {
		erroJSON(w, statusOu(status, http.StatusBadGateway), erro)
		return
	}
	envios, _ := a.enviosFlashcards(ctx, e.TrilhaID, e.Chave)
	responderJSON(w, http.StatusCreated, map[string]any{"geracao_id": g.ID, "envios": envios})
}

// secoesDaLicao: os capítulos (##) da lição, com o link de cada um na etapa. A âncora é a mesma
// que o Leitor dá ao título (markdown.js, ancora()). Referências não viram cartão.
func secoesDaLicao(licao, link string) []map[string]string {
	lista := []map[string]string{}
	vistas := map[string]int{}
	for _, linha := range strings.Split(licao, "\n") {
		if !strings.HasPrefix(linha, "## ") {
			continue
		}
		titulo := strings.TrimSpace(strings.TrimPrefix(linha, "## "))
		id := ancoraDoTitulo(titulo)
		n := vistas[id]
		vistas[id]++
		if n > 0 {
			id = fmt.Sprintf("%s-%d", id, n)
		}
		lista = append(lista, map[string]string{"titulo": titulo, "link": link + "#" + id})
	}
	return lista
}

// ancoraDoTitulo: o mesmo que ancora() do markdown.js (formato do GitHub).
func ancoraDoTitulo(texto string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(texto)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('-')
		}
	}
	return b.String()
}

// textoDaIA: às vezes a IA escapa as aspas dentro do texto do JSON (\"seção\"); na tela
// apareceria a barra.
func textoDaIA(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, `\"`, `"`))
}
