package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// "Sugerir material" (fase 6, T27): a IA procura material para uma etapa na web, nos livros da
// Biblioteca, nos Temas do Acervo e nos documentos da trilha. O servidor confere cada link e tira o
// que não existe (livro ou Tema fora da lista, documento que a trilha não tem). É uma proposta da
// etapa como a lição: comentário (geral ou em material:N), ajuste, e aprovar com os que o Lucas
// marcou, que entram no material da etapa.

const tipoMateriais = "materiais"

type materialSugerido struct {
	material
	Motivo   string `json:"motivo"`
	Situacao string `json:"situacao,omitempty"` // web: ok · quebrado · nao_confirmado
}

type conteudoMateriais struct {
	Materiais []materialSugerido   `json:"materiais"`
	Notas     string               `json:"notas,omitempty"`
	Respostas []respostaComentario `json:"respostas,omitempty"`
}

const esquemaMateriais = `{"type":"object","properties":{
"materiais":{"type":"array","items":{"type":"object","properties":{
  "tipo":{"type":"string","enum":["web","livro","acervo","documento"]},
  "titulo":{"type":"string"},
  "referencia":{"type":"string","description":"web: a URL; livro: o id do livro na Biblioteca; acervo: a pasta do Tema; documento: o slug do documento"},
  "localizador":{"type":"string","description":"livro: cap. N ou p. A–B, se souber; documento: a âncora da seção; senão vazio"},
  "papel":{"type":"string","enum":["principal","apoio","aprofundar"]},
  "nota":{"type":"string","description":"o que ler e por quê, para o Lucas (vai junto com o material)"},
  "motivo":{"type":"string","description":"por que este material serve a esta etapa (só na proposta)"}},
  "required":["tipo","titulo","referencia","localizador","papel","nota","motivo"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["materiais","notas","respostas"]}`

func (a *app) handleSugerirMaterial(w http.ResponseWriter, r *http.Request) {
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
	p, err := a.novaProposta(r.Context(), e.TrilhaID, e.Chave, tipoMateriais, strings.TrimSpace(corpo.Pedido), 0, titulosEstudo[tipoMateriais]+": "+e.Titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

// catalogo: o que existe nos outros apps (vazio quando o app não está configurado ou não respondeu).
type catalogo struct {
	livros    []livroBiblioteca
	temas     []temaAcervo
	documento map[string]bool
	avisos    []string
}

func (a *app) lerCatalogo(ctx context.Context, tr trilha) catalogo {
	c := catalogo{documento: map[string]bool{}}
	for _, d := range tr.Documentos {
		c.documento[d.Slug] = true
	}
	if a.integra.biblioteca != "" {
		if _, err := chamarApp(ctx, a.integra.biblioteca, "Biblioteca", http.MethodGet, "/api/books", nil, &c.livros); err != nil {
			c.avisos = append(c.avisos, "Biblioteca: "+err.Error())
		}
	}
	if a.integra.acervo != "" {
		if _, err := chamarApp(ctx, a.integra.acervo, "Acervo", http.MethodGet, "/api/temas", nil, &c.temas); err != nil {
			c.avisos = append(c.avisos, "Acervo: "+err.Error())
		}
	}
	return c
}

func (a *app) gerarMateriais(ctx context.Context, t *tarefa) error {
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
	cat := a.lerCatalogo(ctx, tr)
	arquivos, comentarios, err := a.montarPedidoMateriais(ctx, tr, p, cat)
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
	rg, err := a.ia.lerRegra("materiais.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: rg.Texto, esquema: esquemaMateriais, modelo: t.Modelo,
		ferramentas: "Read,Glob,Grep,WebSearch,WebFetch",
		instrucao: "Procure material de estudo para uma etapa de uma trilha. Tudo o que você precisa saber está em " +
			"`pedido.md`, nesta pasta: leia-o inteiro primeiro.",
	})
	if err != nil {
		return err
	}
	var c conteudoMateriais
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	e, _ := a.etapaPorChave(ctx, p.TrilhaID, p.EtapaChave)
	var tirados []string
	c.Materiais, tirados = a.limparMateriais(ctx, c.Materiais, cat, e.Materiais)
	if len(tirados) > 0 {
		c.Notas = strings.TrimSpace(c.Notas + "\n\n(O app tirou: " + strings.Join(tirados, "; ") + ".)")
	}
	if len(c.Materiais) == 0 {
		return errors.New("a IA não achou material que exista de verdade" + entreParenteses(strings.Join(tirados, "; ")))
	}
	bruto, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return a.guardarProposta(ctx, p, string(bruto), c.Respostas, comentarios)
}

// limparMateriais tira o que não existe ou já está na etapa, e confere os links da web (o
// quebrado fica, marcado, para o Lucas ver por que não marcar).
func (a *app) limparMateriais(ctx context.Context, lista []materialSugerido, cat catalogo, atuais []material) ([]materialSugerido, []string) {
	livros, temas := map[string]bool{}, map[string]bool{}
	for _, l := range cat.livros {
		livros[l.ID] = true
	}
	for _, t := range cat.temas {
		temas[t.Caminho] = true
	}
	ja := map[string]bool{}
	chave := func(m material) string { return m.Tipo + "|" + m.Referencia + "|" + m.Localizador }
	for _, m := range atuais {
		ja[chave(m)] = true
	}
	var bons []materialSugerido
	var tirados []string
	var urls []string
	for _, x := range lista {
		x.Titulo, x.Referencia, x.Localizador = strings.TrimSpace(x.Titulo), strings.TrimSpace(x.Referencia), strings.TrimSpace(x.Localizador)
		if x.Papel == "" {
			x.Papel = "apoio"
		}
		motivo := ""
		switch {
		case x.Titulo == "" || x.Referencia == "":
			motivo = "um material sem título ou referência"
		case !contem(papeisMaterial, x.Papel):
			motivo = fmt.Sprintf("\"%s\" (papel desconhecido)", x.Titulo)
		case x.Tipo == "livro" && !livros[x.Referencia]:
			motivo = fmt.Sprintf("o livro \"%s\" (não está na Biblioteca)", x.Titulo)
		case x.Tipo == "acervo" && !temas[x.Referencia]:
			motivo = fmt.Sprintf("\"%s\" (não é um Tema do Acervo)", x.Titulo)
		case x.Tipo == "documento" && !cat.documento[x.Referencia]:
			motivo = fmt.Sprintf("o documento \"%s\" (a trilha não tem)", x.Referencia)
		case x.Tipo == "web" && !strings.HasPrefix(x.Referencia, "https://") && !strings.HasPrefix(x.Referencia, "http://"):
			motivo = fmt.Sprintf("\"%s\" (endereço inválido)", x.Titulo)
		case x.Tipo != "web" && x.Tipo != "livro" && x.Tipo != "acervo" && x.Tipo != "documento":
			motivo = fmt.Sprintf("\"%s\" (tipo %q)", x.Titulo, x.Tipo)
		case ja[chave(x.material)]:
			motivo = fmt.Sprintf("\"%s\" (já está na etapa)", x.Titulo)
		}
		if motivo != "" {
			tirados = append(tirados, motivo)
			continue
		}
		ja[chave(x.material)] = true
		if x.Tipo == "web" {
			urls = append(urls, x.Referencia)
		}
		bons = append(bons, x)
	}
	if a.ia.conferirLinks != nil && len(urls) > 0 {
		situacao := map[string]string{}
		for _, l := range a.ia.conferirLinks(ctx, strings.Join(urls, "\n")) {
			situacao[l.URL] = l.Situacao
		}
		for i := range bons {
			if bons[i].Tipo == "web" {
				bons[i].Situacao = situacao[bons[i].Referencia]
			}
		}
	}
	return vazioSeNulo(bons), tirados
}

func (a *app) montarPedidoMateriais(ctx context.Context, tr trilha, p proposta, cat catalogo) (map[string]string, map[int64]bool, error) {
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
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, err
	}
	profundidade := e.Profundidade
	if profundidade == 0 {
		profundidade = tr.Profundidade
	}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: material da etapa \"%s\"\n\n", e.Titulo)
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
	escreva("- Profundidade: %s\n", nomesProfundidade[profundidade])
	if e.Minutos > 0 {
		escreva("- Tempo previsto: %d min\n", e.Minutos)
	}
	escreva("\n## O material que a etapa já tem (não repita)\n\n")
	if len(e.Materiais) == 0 {
		escreva("(nenhum)\n")
	}
	for _, m := range e.Materiais {
		escreva("%s", descreverMaterial(m, lidoDoMaterial{}))
	}

	arquivos := map[string]string{}
	escreva("\n## Onde procurar\n\n")
	escreva("- **Web**: busque e abra cada página antes de sugerir (o servidor confere os links de novo).\n")
	if len(cat.livros) > 0 {
		var b strings.Builder
		b.WriteString("# Livros da Biblioteca do Lucas\n\n| id | Título | Autor | Formato |\n|---|---|---|---|\n")
		for _, l := range cat.livros {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", l.ID, strings.ReplaceAll(l.Title, "|", "/"), strings.ReplaceAll(l.Author, "|", "/"), l.Format)
		}
		arquivos["biblioteca.md"] = b.String()
		escreva("- **Livros da Biblioteca** dele: a lista está em `biblioteca.md` (use o id como referência; o capítulo, se não souber, deixe vazio e diga na nota o que procurar).\n")
	}
	if len(cat.temas) > 0 {
		var b strings.Builder
		b.WriteString("# Temas do Acervo do Lucas (as notas dele, revisadas)\n\n")
		for _, t := range cat.temas {
			fmt.Fprintf(&b, "- `%s` — %s%s\n", t.Caminho, t.Titulo, sufixo(": ", t.Resumo))
		}
		arquivos["acervo.md"] = b.String()
		escreva("- **Temas do Acervo** dele: a lista está em `acervo.md` (use a pasta como referência).\n")
	}
	if len(tr.Documentos) > 0 {
		escreva("- **Documentos da trilha**, na pasta `documentos` (adicionada ao seu acesso; procure-a com Glob se o caminho relativo não abrir):\n")
		for _, d := range tr.Documentos {
			escreva("  - `%s` — %s\n", d.Slug, d.Titulo)
		}
	}
	for _, x := range cat.avisos {
		escreva("- (Não deu para ler a lista do %s.)\n", x)
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
		escreva("A versão anterior está em `versao-anterior.json`. Parta dela: mude o que os comentários pedem e mantenha o resto. Os comentários num material dizem `material:N` (N a partir de 1, na ordem da versão anterior). Responda **cada** comentário em `respostas`, pelo número.\n\n")
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

// aprovarMateriais: os marcados entram no fim do material da etapa.
func (a *app) aprovarMateriais(w http.ResponseWriter, r *http.Request, p proposta) {
	var corpo struct {
		Aceitos []int `json:"aceitos"`
	}
	if r.ContentLength != 0 {
		if err := lerCorpo(r, &corpo); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	ctx := r.Context()
	e, err := a.etapaPorChave(ctx, p.TrilhaID, p.EtapaChave)
	if err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa desta proposta não existe mais na trilha")})
		return
	}
	if err := a.semVersaoGerando(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	var c conteudoMateriais
	if err := json.Unmarshal(p.Conteudo, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(corpo.Aceitos) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("marque os materiais que entram na etapa (ou descarte a proposta)")})
		return
	}
	vistos := map[int]bool{}
	for _, i := range corpo.Aceitos {
		if i < 0 || i >= len(c.Materiais) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("material %d não está na proposta", i+1)})
			return
		}
		if !vistos[i] {
			vistos[i] = true
			e.Materiais = append(e.Materiais, c.Materiais[i].material)
		}
	}
	if err := validarEtapa(&e); err != nil {
		a.responderErro(w, erroDeEntrada{err})
		return
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	if err := a.salvarEtapa(ctx, tx, e); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.exportar(ctx, e.TrilhaID, fmt.Sprintf("material da etapa %s (proposta v%d aprovada)", e.Chave, p.Versao), nil); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}
