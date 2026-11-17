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
	"time"
)

// Encaixar material (T32): o Lucas manda um material à trilha inteira (um livro ou PDF da
// Biblioteca, uma Referência ou Tema do Acervo, que já transcreve vídeos do YouTube e Reels, um
// link ou um documento) e a IA diz onde ele entra: em quais etapas (com o capítulo ou o trecho) e
// quais tópicos ele traz que a trilha ainda não tem. É uma proposta da trilha (chave própria, uma
// por material, para vários poderem estar na fila): o Lucas marca o que aceita, pede ajuste ou
// descarta; aprovar põe o material nas etapas marcadas e cria as etapas novas marcadas.

const tipoEncaixe = "encaixe"

var tiposEncaixe = []string{"livro", "acervo", "web", "documento"}

type encaixeEtapa struct {
	EtapaChave  string `json:"etapa_chave"`
	Localizador string `json:"localizador"`
	Papel       string `json:"papel"`
	Nota        string `json:"nota"`
	Motivo      string `json:"motivo"`
	// Preenchidos pelo servidor, para a tela.
	Numero string `json:"numero,omitempty"`
	Titulo string `json:"titulo,omitempty"`
}

type encaixeNovo struct {
	Titulo      string `json:"titulo"`
	Objetivo    string `json:"objetivo"`
	Bloom       string `json:"bloom"`
	Minutos     int    `json:"minutos"`
	Modulo      string `json:"modulo"`    // título de um módulo existente ou de um novo
	DepoisDe    string `json:"depois_de"` // chave da etapa depois da qual entra ("" = no fim do módulo)
	Localizador string `json:"localizador"`
	Nota        string `json:"nota"`
	Motivo      string `json:"motivo"`
	ModuloNovo  bool   `json:"modulo_novo,omitempty"`
}

type conteudoEncaixe struct {
	Material  material             `json:"material"`
	Encaixes  []encaixeEtapa       `json:"encaixes"`
	Novos     []encaixeNovo        `json:"novos"`
	Notas     string               `json:"notas,omitempty"`
	Respostas []respostaComentario `json:"respostas,omitempty"`
}

const esquemaEncaixe = `{"type":"object","properties":{
"encaixes":{"type":"array","items":{"type":"object","properties":{
  "etapa_chave":{"type":"string","description":"a chave da etapa (coluna chave do pedido)"},
  "localizador":{"type":"string","description":"livro: cap. N, caps. 2–4 ou p. A–B; vídeo ou texto: o trecho ou minutos; vazio = o material todo"},
  "papel":{"type":"string","enum":["principal","apoio","aprofundar"]},
  "nota":{"type":"string","description":"o que essa parte do material traz para a etapa, para o Lucas (vai junto com o material)"},
  "motivo":{"type":"string","description":"por que encaixa aqui (só na proposta)"}},
  "required":["etapa_chave","localizador","papel","nota","motivo"]}},
"novos":{"type":"array","items":{"type":"object","properties":{
  "titulo":{"type":"string"},
  "objetivo":{"type":"string","description":"com um verbo de Bloom, dizendo o que aprender, bem específico"},
  "bloom":{"type":"string","enum":["lembrar","entender","aplicar","analisar","avaliar","criar"]},
  "minutos":{"type":"integer"},
  "modulo":{"type":"string","description":"o título exato de um módulo existente, ou o de um módulo novo"},
  "depois_de":{"type":"string","description":"a chave da etapa depois da qual o tópico entra, ou vazio para o fim do módulo"},
  "localizador":{"type":"string","description":"a parte do material que ensina o tópico"},
  "nota":{"type":"string"},
  "motivo":{"type":"string","description":"por que a trilha precisa deste tópico"}},
  "required":["titulo","objetivo","bloom","minutos","modulo","depois_de","localizador","nota","motivo"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["encaixes","novos","notas","respostas"]}`

// ---------- pedir ----------

func (a *app) handleEncaixar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Material material `json:"material"`
		Pedido   string   `json:"pedido"`
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
	m := corpo.Material
	m.Titulo, m.Referencia, m.Localizador = strings.TrimSpace(m.Titulo), strings.TrimSpace(m.Referencia), strings.TrimSpace(m.Localizador)
	switch {
	case !contem(tiposEncaixe, m.Tipo):
		a.responderErro(w, erroDeEntrada{errors.New("escolha um livro, um item do Acervo, um documento ou um link")})
		return
	case m.Referencia == "":
		a.responderErro(w, erroDeEntrada{errors.New("o material não tem referência")})
		return
	case m.Tipo == "web" && !strings.HasPrefix(m.Referencia, "http://") && !strings.HasPrefix(m.Referencia, "https://"):
		a.responderErro(w, erroDeEntrada{errors.New("o link precisa começar com http:// ou https://")})
		return
	}
	if m.Titulo == "" {
		m.Titulo = m.Referencia
	}
	if len(tr.Modulos) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a trilha ainda não tem módulos para encaixar o material")})
		return
	}
	bruto, _ := json.Marshal(m)
	chave := fmt.Sprintf("@encaixe-%d", time.Now().UnixNano())
	p, err := a.novaProposta(ctx, id, chave, tipoEncaixe, strings.TrimSpace(corpo.Pedido), 0, "Encaixar: "+m.Titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(ctx, `INSERT INTO encaixes (proposta_id, material) VALUES (?, ?)`, p.ID, string(bruto)); err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

// materialDoEncaixe: o material guardado na 1ª versão (os ajustes apontam para a anterior).
func (a *app) materialDoEncaixe(ctx context.Context, p proposta) (material, error) {
	var m material
	id, anterior := p.ID, p.AnteriorID
	for i := 0; i < 100 && id > 0; i++ {
		var bruto string
		err := a.db.QueryRowContext(ctx, `SELECT material FROM encaixes WHERE proposta_id = ?`, id).Scan(&bruto)
		if err == nil {
			return m, json.Unmarshal([]byte(bruto), &m)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return m, err
		}
		id = anterior
		anterior = 0
		if id > 0 {
			ant, err := a.propostaPorID(ctx, id)
			if err != nil {
				return m, err
			}
			anterior = ant.AnteriorID
		}
	}
	return m, errors.New("o material deste encaixe não foi achado")
}

// handleEncaixes: as propostas de encaixe da trilha (uma por material), da mais nova à mais antiga.
func (a *app) handleEncaixes(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	rows, err := a.db.QueryContext(ctx, `SELECT id FROM propostas WHERE trilha_id = ? AND tipo = ? AND estado != ?
		ORDER BY id DESC LIMIT 60`, id, tipoEncaixe, propDescartada)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var ids []int64
	for rows.Next() {
		var x int64
		if rows.Scan(&x) == nil {
			ids = append(ids, x)
		}
	}
	rows.Close()
	lista := []map[string]any{}
	vistas := map[string]bool{} // uma por material: a versão mais nova (que não foi descartada)
	for _, x := range ids {
		p, err := a.propostaPorID(ctx, x)
		if err != nil || vistas[p.EtapaChave] {
			continue
		}
		vistas[p.EtapaChave] = true
		if err := a.completar(ctx, &p); err != nil {
			continue
		}
		m, _ := a.materialDoEncaixe(ctx, p)
		lista = append(lista, map[string]any{"proposta": p, "material": m})
	}
	responderJSON(w, http.StatusOK, lista)
}

// ---------- gerar ----------

// numeracao: "3.2" de cada etapa, pela ordem da trilha.
func numeracao(tr trilha) map[string]string {
	n := map[string]string{}
	for i, m := range tr.Modulos {
		for j, e := range m.Etapas {
			n[e.Chave] = fmt.Sprintf("%d.%d", i+1, j+1)
		}
	}
	return n
}

func (a *app) gerarEncaixe(ctx context.Context, t *tarefa) error {
	p, err := a.propostaPorID(ctx, t.PropostaID)
	if err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, p.TrilhaID)
	if err != nil {
		return err
	}
	m, err := a.materialDoEncaixe(ctx, p)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	arquivos, comentarios, err := a.montarPedidoEncaixe(ctx, tr, p, m)
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
	rg, err := a.ia.lerRegra("encaixe.md")
	if err != nil {
		return err
	}
	ferramentas := "Read,Glob,Grep"
	if m.Tipo == "web" {
		ferramentas += ",WebFetch"
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: rg.Texto, esquema: esquemaEncaixe, modelo: t.Modelo, ferramentas: ferramentas,
		instrucao: "Diga onde um material se encaixa numa trilha de estudo e que tópicos ele traz que a trilha não tem. " +
			"Tudo está em `pedido.md`, nesta pasta: leia-o inteiro primeiro, e depois o material.",
	})
	if err != nil {
		return err
	}
	var c conteudoEncaixe
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	c.Material = m
	var tirados []string
	c.Encaixes, c.Novos, tirados = limparEncaixe(tr, m, c.Encaixes, c.Novos)
	if len(tirados) > 0 {
		c.Notas = strings.TrimSpace(c.Notas + "\n\n(O app tirou: " + strings.Join(tirados, "; ") + ".)")
	}
	if len(c.Encaixes) == 0 && len(c.Novos) == 0 {
		return errors.New("a IA não achou onde o material se encaixa" + entreParenteses(strings.TrimSpace(c.Notas)))
	}
	bruto, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return a.guardarProposta(ctx, p, string(bruto), c.Respostas, comentarios)
}

// limparEncaixe tira o que aponta para etapa que não existe ou repete o que a etapa já tem, e
// completa o número e o título de cada etapa para a tela.
func limparEncaixe(tr trilha, m material, encaixes []encaixeEtapa, novos []encaixeNovo) ([]encaixeEtapa, []encaixeNovo, []string) {
	num := numeracao(tr)
	etapas := map[string]etapa{}
	modulos := map[string]bool{}
	for _, md := range tr.Modulos {
		modulos[strings.ToLower(strings.TrimSpace(md.Titulo))] = true
		for _, e := range md.Etapas {
			etapas[e.Chave] = e
		}
	}
	var tirados []string
	var bons []encaixeEtapa
	vistos := map[string]bool{}
	for _, x := range encaixes {
		x.EtapaChave, x.Localizador = strings.TrimSpace(x.EtapaChave), strings.TrimSpace(x.Localizador)
		e, ok := etapas[x.EtapaChave]
		if !ok {
			tirados = append(tirados, fmt.Sprintf("a etapa %q (não existe)", x.EtapaChave))
			continue
		}
		if !contem(papeisMaterial, x.Papel) {
			x.Papel = "apoio"
		}
		k := x.EtapaChave + "|" + x.Localizador
		ja := false
		for _, mm := range e.Materiais {
			if mm.Tipo == m.Tipo && mm.Referencia == m.Referencia && mm.Localizador == x.Localizador {
				ja = true
			}
		}
		if ja || vistos[k] {
			tirados = append(tirados, fmt.Sprintf("%s %s (o material já está nela)", num[e.Chave], e.Titulo))
			continue
		}
		vistos[k] = true
		x.Numero, x.Titulo = num[e.Chave], e.Titulo
		bons = append(bons, x)
	}
	var novosBons []encaixeNovo
	for _, x := range novos {
		x.Titulo, x.Modulo, x.DepoisDe = strings.TrimSpace(x.Titulo), strings.TrimSpace(x.Modulo), strings.TrimSpace(x.DepoisDe)
		if x.Titulo == "" || x.Modulo == "" {
			tirados = append(tirados, "um tópico novo sem título ou módulo")
			continue
		}
		if !contem(niveisBloom, x.Bloom) {
			x.Bloom = "entender"
		}
		if x.Minutos <= 0 {
			x.Minutos = 60
		}
		if _, ok := etapas[x.DepoisDe]; !ok {
			x.DepoisDe = ""
		}
		x.ModuloNovo = !modulos[strings.ToLower(x.Modulo)]
		novosBons = append(novosBons, x)
	}
	return vazioSeNulo(bons), vazioSeNulo(novosBons), tirados
}

func (a *app) montarPedidoEncaixe(ctx context.Context, tr trilha, p proposta, m material) (map[string]string, map[int64]bool, error) {
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, err
	}
	arquivos := map[string]string{}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: encaixar um material na trilha \"%s\"\n\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Recorte", tr.Recorte}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
		if strings.TrimSpace(campo[1]) != "" {
			escreva("- %s: %s\n", campo[0], campo[1])
		}
	}

	escreva("\n## O material\n\n- Tipo: %s\n- Título: %s\n- Referência: %s\n", m.Tipo, m.Titulo, m.Referencia)
	if m.Localizador != "" {
		escreva("- Só este trecho: %s\n", m.Localizador)
	}
	switch m.Tipo {
	case "livro", "acervo":
		lidos := a.arquivosDoMaterial(ctx, []material{m}, arquivos)
		if l := lidos[0]; l.Arquivo != "" {
			escreva("- Texto em `%s`", l.Arquivo)
			if m.Tipo == "livro" && m.Localizador == "" {
				escreva(" (o **sumário** do livro: encaixe por capítulo, com as páginas)")
			}
			escreva(".\n")
		} else {
			return nil, nil, fmt.Errorf("não deu para ler o material: %s", padrao(l.Erro, "sem texto"))
		}
	case "documento":
		escreva("- Arquivo `documentos/%s.md` (pasta `documentos`, adicionada ao seu acesso; procure-a com Glob se o caminho relativo não abrir).\n", m.Referencia)
	case "web":
		escreva("- Abra o link com WebFetch e leia o conteúdo inteiro. Se for vídeo (YouTube, Instagram) e a página não trouxer o conteúdo, use o título e a descrição, e diga nas notas que o vídeo deve ir ao Acervo (que transcreve) para um encaixe fino.\n")
	}

	num := numeracao(tr)
	escreva("\n## A trilha, na ordem (número, chave, título, objetivo, material que já tem)\n\n")
	for i, md := range tr.Modulos {
		escreva("\n### Módulo %d: %s\n", i+1, md.Titulo)
		if md.Descricao != "" {
			escreva("%s\n", md.Descricao)
		}
		escreva("\n")
		for _, e := range md.Etapas {
			escreva("- **%s** `%s` — %s: %s", num[e.Chave], e.Chave, e.Titulo, e.Objetivo)
			if len(e.Materiais) > 0 {
				var ts []string
				for _, mm := range e.Materiais {
					ts = append(ts, mm.Titulo+sufixo(" ", entreParenteses(mm.Localizador)))
				}
				escreva(" · já tem: %s", strings.Join(ts, "; "))
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
			return nil, nil, err
		}
		if err := a.completar(ctx, &ant); err != nil {
			return nil, nil, err
		}
		arquivos["versao-anterior.json"] = string(ant.Conteudo)
		escreva("\n## É um ajuste da versão %d\n\nA versão anterior está em `versao-anterior.json`. Parta dela: mude o que o pedido e os comentários dizem e mantenha o resto. Responda **cada** comentário em `respostas`, pelo número.\n\n", ant.Versao)
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
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, nil
}

// ---------- aprovar ----------

func (a *app) aprovarEncaixe(w http.ResponseWriter, r *http.Request, p proposta) {
	var corpo struct {
		Encaixes []int `json:"encaixes"`
		Novos    []int `json:"novos"`
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
	var c conteudoEncaixe
	if err := json.Unmarshal(p.Conteudo, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(corpo.Encaixes) == 0 && len(corpo.Novos) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("marque onde o material entra (ou descarte a proposta)")})
		return
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
	mudadas := map[string]etapa{}
	var ordemMudadas []string
	vistos := map[int]bool{}
	for _, i := range corpo.Encaixes {
		if i < 0 || i >= len(c.Encaixes) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("encaixe %d não está na proposta", i+1)})
			return
		}
		if vistos[i] {
			continue
		}
		vistos[i] = true
		x := c.Encaixes[i]
		e, ok := mudadas[x.EtapaChave]
		if !ok {
			if e, ok = etapas[x.EtapaChave]; !ok {
				a.responderErro(w, erroDeEntrada{fmt.Errorf("a etapa %q não existe mais na trilha", x.EtapaChave)})
				return
			}
			ordemMudadas = append(ordemMudadas, x.EtapaChave)
		}
		mm := c.Material
		mm.Localizador, mm.Papel, mm.Nota = x.Localizador, x.Papel, x.Nota
		e.Materiais = append(e.Materiais, mm)
		if err := validarEtapa(&e); err != nil {
			a.responderErro(w, erroDeEntrada{err})
			return
		}
		mudadas[x.EtapaChave] = e
	}
	var novos []encaixeNovo
	vistos = map[int]bool{}
	for _, i := range corpo.Novos {
		if i < 0 || i >= len(c.Novos) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("tópico novo %d não está na proposta", i+1)})
			return
		}
		if !vistos[i] {
			vistos[i] = true
			novos = append(novos, c.Novos[i])
		}
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	for _, k := range ordemMudadas {
		if err := a.salvarEtapa(ctx, tx, mudadas[k]); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	var criadas []string
	for _, x := range novos {
		mid, ok := modulos[strings.ToLower(x.Modulo)]
		if !ok {
			res, err := tx.ExecContext(ctx, `INSERT INTO modulos (trilha_id, ordem, titulo, descricao)
				VALUES (?, (SELECT COALESCE(MAX(ordem), -1) + 1 FROM modulos WHERE trilha_id = ?), ?, '')`, p.TrilhaID, p.TrilhaID, x.Modulo)
			if err != nil {
				a.responderErro(w, err)
				return
			}
			if mid, err = res.LastInsertId(); err != nil {
				a.responderErro(w, err)
				return
			}
			modulos[strings.ToLower(x.Modulo)] = mid
		}
		// Posição: logo depois de "depois_de", se ela está nesse módulo; senão, no fim do módulo.
		var ordem int
		if d, ok := etapas[x.DepoisDe]; ok && d.ModuloID == mid {
			var atual int
			if err := tx.QueryRowContext(ctx, `SELECT ordem FROM etapas WHERE id = ?`, d.ID).Scan(&atual); err != nil {
				a.responderErro(w, err)
				return
			}
			ordem = atual + 1
		} else if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordem), -1) + 1 FROM etapas WHERE modulo_id = ?`, mid).Scan(&ordem); err != nil {
			a.responderErro(w, err)
			return
		}
		mm := c.Material
		mm.Localizador, mm.Papel, mm.Nota = x.Localizador, "principal", x.Nota
		nova := etapa{TrilhaID: p.TrilhaID, ModuloID: mid, Titulo: x.Titulo, Objetivo: x.Objetivo, Bloom: x.Bloom, Tipo: "normal",
			Minutos: x.Minutos, Requer: []int64{}, Materiais: []material{mm}}
		if err := validarEtapa(&nova); err != nil {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("%s: %w", x.Titulo, err)})
			return
		}
		chave, err := slugLivre(ctx, tx, slugificar(x.Titulo), `SELECT COUNT(*) FROM etapas WHERE trilha_id = ? AND chave = ?`, p.TrilhaID)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		if _, err := tx.ExecContext(ctx, `UPDATE etapas SET ordem = ordem + 1 WHERE modulo_id = ? AND ordem >= ?`, mid, ordem); err != nil {
			a.responderErro(w, err)
			return
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo) VALUES (?, ?, ?, ?, ?)`,
			p.TrilhaID, mid, ordem, chave, nova.Titulo)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		if nova.ID, err = res.LastInsertId(); err != nil {
			a.responderErro(w, err)
			return
		}
		if err := a.salvarEtapa(ctx, tx, nova); err != nil {
			a.responderErro(w, err)
			return
		}
		etapas[chave] = nova
		criadas = append(criadas, chave)
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(ctx, p.TrilhaID)
	msg := fmt.Sprintf("material \"%s\" encaixado em %d etapa(s)", c.Material.Titulo, len(ordemMudadas))
	if len(criadas) > 0 {
		msg += fmt.Sprintf(" e %d tópico(s) novo(s): %s", len(criadas), strings.Join(criadas, ", "))
	}
	if err := a.exportar(ctx, p.TrilhaID, msg+fmt.Sprintf(" (proposta v%d aprovada)", p.Versao), nil); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}
