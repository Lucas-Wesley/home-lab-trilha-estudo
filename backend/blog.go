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
	"sort"
	"strconv"
	"strings"
)

// Blog (T62): no item do Projeto, a IA orienta o Lucas sobre o que escrever no blog a partir do
// item. Quem escreve é ele, tudo: a IA só sugere temas (modo "sugerir") ou os tópicos que um título
// dele não pode deixar de ter (modo "topicos"). Os artigos são independentes do projeto (nada de
// "no meu projeto"), não são tutorial, e são atemporais quando der. Não muda nada em lugar nenhum:
// o resultado fica em `blog_pedidos`, com as marcas do Lucas (guardada, descartada, escrita).

const tipoBlog = "blog" // a tarefa da fila

const (
	blogGerando = "gerando"
	blogPronta  = "pronta"
	blogFalhou  = "falhou"
)

// As engrenagens: cada opção com o texto que vai para a IA. A primeira é o padrão.
var engrenagensBlog = map[string][][2]string{
	"horizonte": {
		{"atemporal", "atemporal: o que continua valendo daqui a dez anos; versões e ferramentas só como exemplo"},
		{"atual", "atual: o estado de agora (versões, ferramentas, práticas recentes), com data"},
		{"misto", "núcleo atemporal + uma seção datada, separada, com o que é de agora"},
	},
	"nivel": {
		{"aprofundamento", "aprofundamento: o leitor já programa; o assunto, mesmo básico, é tratado a fundo (o porquê, o mecanismo, os limites)"},
		{"intermediario", "intermediário: o leitor conhece o assunto de uso e quer entender melhor"},
		{"iniciante", "iniciante: o leitor está chegando ao assunto; parte do zero, sem ser raso"},
	},
	"formato": {
		{"livre", "a IA escolhe o formato que melhor serve ao conteúdo (e diz qual)"},
		{"fundamentos", "ensaio de fundamentos: o porquê de uma base, como as peças sustentam o resto"},
		{"por_dentro", "como funciona por dentro: a anatomia de um mecanismo"},
		{"decisao", "decisão e trade-offs: as opções, o critério e o que se perde em cada uma"},
		{"mapa", "mapa de um assunto: como as peças se encaixam, o panorama"},
		{"armadilhas", "erros e armadilhas: o que quebra, por que quebra e como perceber"},
		{"principio", "princípio aplicado: uma ideia e vários casos em que ela decide"},
		{"opiniao", "opinião fundamentada: uma tese defendida com argumentos e contra-argumentos"},
		{"historia", "a história de uma ideia: por que surgiu, o que resolvia, o que mudou"},
	},
	"abrangencia": {
		{"reunir", "reunir os assuntos do item num artigo com substância (não um assunto isolado)"},
		{"um", "um assunto por artigo, tratado inteiro"},
		{"ambos", "as duas coisas: um artigo que reúne e outros de um assunto só"},
	},
	"tamanho": {
		{"medio", "médio: cerca de 2.000 palavras"},
		{"curto", "curto: cerca de 1.000 palavras"},
		{"longo", "longo: 3.000 palavras ou mais"},
	},
	"codigo": {
		{"pouco", "pouco código: só o que prova o ponto"},
		{"nenhum", "nenhum código: o texto se sustenta sem ele"},
		{"exemplo", "com um exemplo trabalhado, do começo ao fim"},
	},
}

type opcoesBlog struct {
	Horizonte   string `json:"horizonte"`
	Nivel       string `json:"nivel"`
	Formato     string `json:"formato"`
	Abrangencia string `json:"abrangencia"`
	Tamanho     string `json:"tamanho"`
	Codigo      string `json:"codigo"`
	Quantas     int    `json:"quantas"`
}

func (o *opcoesBlog) campos() map[string]*string {
	return map[string]*string{"horizonte": &o.Horizonte, "nivel": &o.Nivel, "formato": &o.Formato,
		"abrangencia": &o.Abrangencia, "tamanho": &o.Tamanho, "codigo": &o.Codigo}
}

// normalizar: opção vazia vira o padrão; desconhecida é erro (a tela só manda as conhecidas).
func (o *opcoesBlog) normalizar() error {
	for nome, v := range o.campos() {
		opcoes := engrenagensBlog[nome]
		if *v == "" {
			*v = opcoes[0][0]
			continue
		}
		achou := false
		for _, x := range opcoes {
			achou = achou || x[0] == *v
		}
		if !achou {
			return fmt.Errorf("opção desconhecida em %s: %q", nome, *v)
		}
	}
	if o.Quantas == 0 {
		o.Quantas = 3
	}
	o.Quantas = min(5, max(1, o.Quantas))
	return nil
}

func textoDaOpcao(nome, valor string) string {
	for _, x := range engrenagensBlog[nome] {
		if x[0] == valor {
			return x[1]
		}
	}
	return valor
}

type marcaBlog struct {
	Estado string `json:"estado"` // guardada · descartada · escrita
	Link   string `json:"link,omitempty"`
	Em     string `json:"em"`
}

type pedidoBlog struct {
	ID           int64                `json:"id"`
	ItemID       string               `json:"item_id"`
	Modo         string               `json:"modo"` // sugerir · topicos
	Titulo       string               `json:"titulo"`
	Nota         string               `json:"nota"`
	Opcoes       opcoesBlog           `json:"opcoes"`
	Pedido       string               `json:"pedido"`
	Estado       string               `json:"estado"`
	Resultado    json.RawMessage      `json:"resultado,omitempty"`
	Marcas       map[string]marcaBlog `json:"marcas"`
	Erro         string               `json:"erro"`
	TarefaID     int64                `json:"tarefa_id"`
	Tarefa       *tarefa              `json:"tarefa,omitempty"`
	CriadoEm     string               `json:"criado_em"`
	AtualizadoEm string               `json:"atualizado_em"`
}

type arcoBlog struct {
	Comeco string `json:"comeco"`
	Meio   string `json:"meio"`
	Fim    string `json:"fim"`
}

type topicoBlog struct {
	Topico      string `json:"topico"`
	Porque      string `json:"porque"`
	Obrigatorio bool   `json:"obrigatorio"`
	Situacao    string `json:"situacao,omitempty"` // estudado · a_estudar · fora_do_item
	Onde        string `json:"onde,omitempty"`
}

type baseBlog struct {
	Fonte    string `json:"fonte"`
	Situacao string `json:"situacao"`
}

type sugestaoBlog struct {
	Titulo         string       `json:"titulo"`
	Alternativas   []string     `json:"alternativas"`
	Tese           string       `json:"tese"`
	Leitor         string       `json:"leitor"`
	Formato        string       `json:"formato"`
	Horizonte      string       `json:"horizonte"`
	Arco           arcoBlog     `json:"arco"`
	Topicos        []topicoBlog `json:"topicos"`
	ExemploCentral string       `json:"exemplo_central"`
	Fora           string       `json:"fora"`
	Fontes         []string     `json:"fontes"`
	Base           []baseBlog   `json:"base"`
}

// resultadoBlog: o que a IA devolve. No modo sugerir, só Sugestoes e Notas; no modo topicos, o resto.
type resultadoBlog struct {
	Sugestoes    []sugestaoBlog `json:"sugestoes,omitempty"`
	Tese         string         `json:"tese,omitempty"`
	TituloMelhor string         `json:"titulo_melhor,omitempty"`
	Topicos      []topicoBlog   `json:"topicos,omitempty"`
	Arco         *arcoBlog      `json:"arco,omitempty"`
	Fora         string         `json:"fora,omitempty"`
	Armadilhas   []string       `json:"armadilhas,omitempty"`
	Perguntas    []string       `json:"perguntas,omitempty"`
	Fontes       []string       `json:"fontes,omitempty"`
	Notas        string         `json:"notas"`
}

const esquemaArcoBlog = `{"type":"object","description":"começo, meio e fim, uma ou duas frases cada","properties":{
  "comeco":{"type":"string","description":"de onde o texto parte: a pergunta, o problema ou a tensão que prende"},
  "meio":{"type":"string","description":"o que ele desenvolve e em que ordem"},
  "fim":{"type":"string","description":"aonde chega: a conclusão, a regra prática, o que o leitor leva"}},
  "required":["comeco","meio","fim"]}`

const esquemaFontesBlog = `{"type":"array","description":"fontes primárias para conferir: documentação oficial, especificação, RFC, paper, livro e capítulo","items":{"type":"string"}}`

func esquemaTopicosBlog(comSituacao bool) string {
	extra, req := "", `"topico","porque","obrigatorio"`
	if comSituacao {
		extra = `,
  "situacao":{"type":"string","enum":["estudado","a_estudar","fora_do_item"],"description":"estudado: o Lucas já estudou (etapa concluída ou lição aprovada no pedido); a_estudar: está no estudo do item e ainda não foi; fora_do_item: o item não cobre"},
  "onde":{"type":"string","description":"a etapa (Tn › módulo › etapa) que cobre, ou onde estudar se fora do item"}`
		req += `,"situacao","onde"`
	}
	return `{"type":"array","items":{"type":"object","properties":{
  "topico":{"type":"string"},
  "porque":{"type":"string","description":"uma frase: por que o artigo precisa dele"},
  "obrigatorio":{"type":"boolean","description":"true: sem ele o artigo não cumpre o título; false: enriquece"}` + extra + `},
  "required":[` + req + `]}}`
}

var esquemaSugerirBlog = `{"type":"object","properties":{
"sugestoes":{"type":"array","items":{"type":"object","properties":{
  "titulo":{"type":"string"},
  "alternativas":{"type":"array","description":"dois outros títulos para a mesma ideia","items":{"type":"string"}},
  "tese":{"type":"string","description":"uma frase: o que o artigo afirma"},
  "leitor":{"type":"string","description":"para quem é e o que leva ao terminar"},
  "formato":{"type":"string","description":"o formato e por que serve a este conteúdo"},
  "horizonte":{"type":"string","description":"por que é atemporal, ou o que nele envelhece"},
  "arco":` + esquemaArcoBlog + `,
  "topicos":` + esquemaTopicosBlog(false) + `,
  "exemplo_central":{"type":"string","description":"o caso concreto que sustenta o texto inteiro"},
  "fora":{"type":"string","description":"o que deixar de fora para não virar apostila"},
  "fontes":` + esquemaFontesBlog + `,
  "base":{"type":"array","description":"o que do item alimenta o artigo","items":{"type":"object","properties":{
    "fonte":{"type":"string","description":"Tn › módulo › etapa, o livro e capítulo, ou a parte do caderno"},
    "situacao":{"type":"string","enum":["estudado","a_estudar","fora_do_item"]}},"required":["fonte","situacao"]}}},
  "required":["titulo","alternativas","tese","leitor","formato","horizonte","arco","topicos","exemplo_central","fora","fontes","base"]}},
"notas":{"type":"string","description":"uma ou duas frases para o Lucas: qual escrever primeiro e por quê"}},
"required":["sugestoes","notas"]}`

var esquemaTopicosDoTitulo = `{"type":"object","properties":{
"tese":{"type":"string","description":"a tese que o título sugere, em uma frase"},
"titulo_melhor":{"type":"string","description":"um título mais preciso, se o do Lucas for vago; vazio se o dele já serve"},
"topicos":` + esquemaTopicosBlog(true) + `,
"arco":` + esquemaArcoBlog + `,
"fora":{"type":"string","description":"o que fica de fora deste artigo"},
"armadilhas":{"type":"array","description":"o que os artigos sobre o tema costumam errar ou repetir","items":{"type":"string"}},
"perguntas":{"type":"array","description":"as perguntas que o leitor vai ter e o artigo deve responder","items":{"type":"string"}},
"fontes":` + esquemaFontesBlog + `,
"notas":{"type":"string","description":"uma ou duas frases para o Lucas"}},
"required":["tese","titulo_melhor","topicos","arco","fora","armadilhas","perguntas","fontes","notas"]}`

const colunasBlog = `id, item_id, modo, titulo, nota, opcoes, pedido, estado, resultado, marcas, erro,
	COALESCE(tarefa_id, 0), criado_em, atualizado_em FROM blog_pedidos`

func lerPedidoBlog(row interface{ Scan(...any) error }) (pedidoBlog, error) {
	var p pedidoBlog
	var opcoes, resultado, marcas string
	err := row.Scan(&p.ID, &p.ItemID, &p.Modo, &p.Titulo, &p.Nota, &opcoes, &p.Pedido, &p.Estado, &resultado, &marcas,
		&p.Erro, &p.TarefaID, &p.CriadoEm, &p.AtualizadoEm)
	if err == nil {
		json.Unmarshal([]byte(opcoes), &p.Opcoes)
		p.Marcas = map[string]marcaBlog{}
		json.Unmarshal([]byte(marcas), &p.Marcas)
		if resultado != "" {
			p.Resultado = json.RawMessage(resultado)
		}
	}
	return p, err
}

// itemDoProjeto: o item do roadmap já ligado às trilhas (a mesma montagem do GET /api/projeto).
func (a *app) itemDoProjeto(ctx context.Context, id string) (*itemProjeto, error) {
	if a.projeto == nil {
		return nil, errors.New("o projeto de estudo não está configurado")
	}
	arquivos, _, _, err := a.projeto.obter(ctx, false)
	if err != nil {
		return nil, err
	}
	resp, err := a.montarProjeto(ctx, arquivos)
	if err != nil {
		return nil, err
	}
	for _, f := range resp.Fases {
		for _, it := range f.Itens {
			if it.ID == id {
				return it, nil
			}
		}
	}
	return nil, errNaoEncontrou
}

// handlePedirBlog: POST /api/projeto/itens/{id}/blog
// {"modo":"sugerir|topicos","titulo":"…","nota":"…","opcoes":{…},"pedido":"…"}
func (a *app) handlePedirBlog(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Modo   string     `json:"modo"`
		Titulo string     `json:"titulo"`
		Nota   string     `json:"nota"`
		Opcoes opcoesBlog `json:"opcoes"`
		Pedido string     `json:"pedido"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	corpo.Titulo, corpo.Nota, corpo.Pedido = strings.TrimSpace(corpo.Titulo), strings.TrimSpace(corpo.Nota), strings.TrimSpace(corpo.Pedido)
	switch {
	case corpo.Modo != "sugerir" && corpo.Modo != "topicos":
		a.responderErro(w, erroDeEntrada{errors.New(`modo deve ser "sugerir" ou "topicos"`)})
		return
	case corpo.Modo == "topicos" && corpo.Titulo == "":
		a.responderErro(w, erroDeEntrada{errors.New("escreva o título do artigo para pedir os tópicos")})
		return
	}
	if corpo.Modo == "sugerir" {
		corpo.Titulo, corpo.Nota = "", ""
	}
	if err := corpo.Opcoes.normalizar(); err != nil {
		a.responderErro(w, erroDeEntrada{err})
		return
	}
	ctx := r.Context()
	it, err := a.itemDoProjeto(ctx, r.PathValue("id"))
	if errors.Is(err, errNaoEncontrou) {
		erroJSON(w, http.StatusNotFound, "item não encontrado no roadmap")
		return
	}
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	var ativo int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM blog_pedidos WHERE item_id = ? AND estado = ?`, it.ID, blogGerando).Scan(&ativo); err != nil {
		a.responderErro(w, err)
		return
	}
	if ativo > 0 {
		a.responderErro(w, erroDeEntrada{errors.New("já há um pedido do blog em andamento neste item")})
		return
	}
	titulo := "Blog: sugestões de artigo para o " + it.ID
	if corpo.Modo == "topicos" {
		titulo = "Blog: tópicos para \"" + corpo.Titulo + "\""
	}
	opcoes, _ := json.Marshal(corpo.Opcoes)
	modelo := a.modeloPara(ctx, tipoBlog) // antes da transação: o banco tem uma conexão só
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoBlog, 0, titulo, modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	momento := agora()
	res, err := tx.ExecContext(ctx, `INSERT INTO blog_pedidos (item_id, modo, titulo, nota, opcoes, pedido, estado, tarefa_id, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, it.ID, corpo.Modo, corpo.Titulo, corpo.Nota, string(opcoes), corpo.Pedido, blogGerando, tarefaID, momento, momento)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	id, _ := res.LastInsertId()
	p, err := a.pedidoBlogPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

// handleBlogDoItem: GET /api/projeto/itens/{id}/blog — os 5 últimos pedidos do item e, além
// deles, os mais antigos que têm alguma marca (a ideia guardada não some do quadro).
func (a *app) handleBlogDoItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasBlog+` WHERE item_id = ? ORDER BY id DESC`, r.PathValue("id"))
	if err != nil {
		a.responderErro(w, err)
		return
	}
	lista := []pedidoBlog{}
	for n := 0; rows.Next(); n++ {
		p, err := lerPedidoBlog(rows)
		if err != nil {
			rows.Close()
			a.responderErro(w, err)
			return
		}
		if n < 5 || len(p.Marcas) > 0 {
			lista = append(lista, p)
		}
	}
	rows.Close()
	for i := range lista {
		a.completarPedidoBlog(ctx, &lista[i])
	}
	responderJSON(w, http.StatusOK, lista)
}

// handleMarcarBlog: PUT /api/blog/{id}/marcas/{n} {"estado":"guardada|descartada|escrita|","link":"…"}
// — n é a sugestão (0 no modo topicos); estado vazio tira a marca.
func (a *app) handleMarcarBlog(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Estado string `json:"estado"`
		Link   string `json:"link"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if corpo.Estado != "" && !contem([]string{"guardada", "descartada", "escrita"}, corpo.Estado) {
		a.responderErro(w, erroDeEntrada{errors.New("estado deve ser guardada, descartada ou escrita")})
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("número da sugestão inválido")})
		return
	}
	ctx := r.Context()
	p, err := a.pedidoBlogPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if p.Estado != blogPronta {
		a.responderErro(w, erroDeEntrada{errors.New("o pedido ainda não tem resultado")})
		return
	}
	var res resultadoBlog
	json.Unmarshal(p.Resultado, &res)
	total := len(res.Sugestoes)
	if p.Modo == "topicos" {
		total = 1
	}
	if n < 0 || n >= total {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("a sugestão %d não existe neste pedido", n+1)})
		return
	}
	chave := strconv.Itoa(n)
	if corpo.Estado == "" {
		delete(p.Marcas, chave)
	} else {
		p.Marcas[chave] = marcaBlog{Estado: corpo.Estado, Link: strings.TrimSpace(corpo.Link), Em: agora()}
	}
	marcas, _ := json.Marshal(p.Marcas)
	if _, err := a.db.ExecContext(ctx, `UPDATE blog_pedidos SET marcas = ?, atualizado_em = ? WHERE id = ?`, string(marcas), agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	p, err = a.pedidoBlogPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, p)
}

func (a *app) pedidoBlogPorID(ctx context.Context, id int64) (pedidoBlog, error) {
	p, err := lerPedidoBlog(a.db.QueryRowContext(ctx, `SELECT `+colunasBlog+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, errNaoEncontrou
	}
	if err == nil {
		a.completarPedidoBlog(ctx, &p)
	}
	return p, err
}

// completarPedidoBlog: a tarefa (para a tela dizer "na fila" ou "gerando").
func (a *app) completarPedidoBlog(ctx context.Context, p *pedidoBlog) {
	if p.TarefaID == 0 || p.Estado != blogGerando {
		return
	}
	if t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, p.TarefaID)); err == nil {
		p.Tarefa = t
	}
}

// ---------- a tarefa ----------

func (a *app) gerarBlog(ctx context.Context, t *tarefa) error {
	p, err := lerPedidoBlog(a.db.QueryRowContext(ctx, `SELECT `+colunasBlog+` WHERE tarefa_id = ?`, t.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("o pedido do blog desta tarefa não existe mais")
	}
	if err != nil {
		return err
	}
	it, err := a.itemDoProjeto(ctx, p.ItemID)
	if err != nil {
		return fmt.Errorf("o item %s: %w", p.ItemID, err)
	}
	arquivos, err := a.montarPedidoBlog(ctx, p, it)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("blog.md")
	if err != nil {
		return err
	}
	esquema, instrucao := esquemaSugerirBlog, "Sugira artigos para o blog do Lucas a partir do item. "
	if p.Modo == "topicos" {
		esquema, instrucao = esquemaTopicosDoTitulo, "Diga os tópicos que o artigo com o título do Lucas não pode deixar de ter. "
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, esquema: esquema, modelo: t.Modelo, ferramentas: "Read,Glob,Grep",
		instrucao: instrucao + "Tudo está em `pedido.md`, nesta pasta: leia o pedido, o caderno do item e as lições antes de responder.",
	})
	if err != nil {
		return err
	}
	var res resultadoBlog
	if err := dadosDaResposta(resp, &res); err != nil {
		return err
	}
	if err := limparResultadoBlog(&res, p.Modo); err != nil {
		return err
	}
	bruto, _ := json.Marshal(res)
	_, err = a.db.ExecContext(ctx, `UPDATE blog_pedidos SET estado = ?, resultado = ?, erro = '', atualizado_em = ? WHERE id = ?`,
		blogPronta, string(bruto), agora(), p.ID)
	return err
}

func limparLista(l []string) []string {
	saida := []string{}
	for _, x := range l {
		if x = textoDaIA(x); x != "" {
			saida = append(saida, x)
		}
	}
	return saida
}

func limparTopicosBlog(l []topicoBlog, comSituacao bool) []topicoBlog {
	saida := []topicoBlog{}
	for _, x := range l {
		x.Topico, x.Porque, x.Onde = textoDaIA(x.Topico), textoDaIA(x.Porque), textoDaIA(x.Onde)
		if x.Topico == "" {
			continue
		}
		if !comSituacao {
			x.Situacao, x.Onde = "", ""
		} else if !contem([]string{"estudado", "a_estudar", "fora_do_item"}, x.Situacao) {
			x.Situacao = "fora_do_item"
		}
		saida = append(saida, x)
	}
	return saida
}

func limparArco(a *arcoBlog) {
	a.Comeco, a.Meio, a.Fim = textoDaIA(a.Comeco), textoDaIA(a.Meio), textoDaIA(a.Fim)
}

// limparResultadoBlog: textos aparados, enums conhecidos, listas nunca nulas, e só os campos do
// modo. Sem nenhuma sugestão (ou sem tópicos) é falha: a IA não respondeu o que se pediu.
func limparResultadoBlog(r *resultadoBlog, modo string) error {
	r.Notas = textoDaIA(r.Notas)
	if modo == "sugerir" {
		sugestoes := []sugestaoBlog{}
		for _, s := range r.Sugestoes {
			s.Titulo = textoDaIA(s.Titulo)
			if s.Titulo == "" {
				continue
			}
			s.Alternativas, s.Fontes = limparLista(s.Alternativas), limparLista(s.Fontes)
			s.Tese, s.Leitor, s.Formato, s.Horizonte = textoDaIA(s.Tese), textoDaIA(s.Leitor), textoDaIA(s.Formato), textoDaIA(s.Horizonte)
			s.ExemploCentral, s.Fora = textoDaIA(s.ExemploCentral), textoDaIA(s.Fora)
			limparArco(&s.Arco)
			s.Topicos = limparTopicosBlog(s.Topicos, false)
			base := []baseBlog{}
			for _, b := range s.Base {
				if b.Fonte = textoDaIA(b.Fonte); b.Fonte == "" {
					continue
				}
				if !contem([]string{"estudado", "a_estudar", "fora_do_item"}, b.Situacao) {
					b.Situacao = "fora_do_item"
				}
				base = append(base, b)
			}
			s.Base = base
			sugestoes = append(sugestoes, s)
		}
		*r = resultadoBlog{Sugestoes: sugestoes, Notas: r.Notas}
		if len(sugestoes) == 0 {
			return errors.New("a IA não devolveu nenhuma sugestão de artigo")
		}
		return nil
	}
	arco := arcoBlog{}
	if r.Arco != nil {
		arco = *r.Arco
	}
	limparArco(&arco)
	*r = resultadoBlog{Tese: textoDaIA(r.Tese), TituloMelhor: textoDaIA(r.TituloMelhor), Topicos: limparTopicosBlog(r.Topicos, true),
		Arco: &arco, Fora: textoDaIA(r.Fora), Armadilhas: limparLista(r.Armadilhas), Perguntas: limparLista(r.Perguntas),
		Fontes: limparLista(r.Fontes), Notas: r.Notas}
	if len(r.Topicos) == 0 {
		return errors.New("a IA não devolveu nenhum tópico")
	}
	return nil
}

// montarPedidoBlog: o item (o trecho do roadmap e o caderno), o que já foi estudado (com as lições
// aprovadas em licoes/), as engrenagens, as ideias já guardadas ou escritas e o pedido do Lucas.
func (a *app) montarPedidoBlog(ctx context.Context, p pedidoBlog, it *itemProjeto) (map[string]string, error) {
	arquivos := map[string]string{}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	if p.Modo == "topicos" {
		escreva("# Pedido: tópicos para um título do blog\n\n")
		escreva("O Lucas vai escrever um artigo com este título:\n\n> %s\n\n", p.Titulo)
		if p.Nota != "" {
			escreva("O que ele quer dizer, nas palavras dele:\n\n> %s\n\n", p.Nota)
		}
		escreva("Diga os tópicos que esse artigo não pode deixar de ter, a partir do título e do que o item já tem.\n\n")
	} else {
		escreva("# Pedido: artigos do blog a partir do item %s\n\n", it.ID)
		escreva("Sugira %d artigo(s) que o Lucas pode escrever com o que este item ensina.\n\n", p.Opcoes.Quantas)
	}

	escreva("## O item (a matéria-prima, não o assunto do artigo)\n\n")
	escreva("- %s · %s (%s)\n- Situação: %s", it.ID, it.Titulo, it.Fase, it.Situacao)
	if it.Estado != "" {
		escreva(", fase do ciclo: %s", it.Estado)
	}
	escreva("\n\nO trecho do item no roadmap, como está:\n\n%s\n\n", strings.TrimSpace(it.Markdown))
	if strings.TrimSpace(it.Caderno) != "" {
		arquivos["caderno.md"] = it.Caderno
		escreva("O caderno do item (problema, estudo, decisões, execução e o que o Lucas aprendeu, até onde chegou) está em `caderno.md`.\n\n")
	}

	escreva("## O que o item pede de estudo, e o que o Lucas já estudou\n\n")
	escreva("Situação de cada fonte: `concluida` = estudada. As lições aprovadas (o texto que o Lucas estudou) estão em `licoes/`.\n\n")
	vistos := map[string]bool{}
	licoes := 0
	linha := func(rotulo string, refs []referenciaProjeto, texto string) {
		if len(refs) == 0 {
			escreva("- %s: %s\n", rotulo, texto)
			return
		}
		for _, r := range refs {
			nome := primeiro(r.Caminho, r.Texto)
			if vistos[nome] {
				continue
			}
			vistos[nome] = true
			situacao := primeiro(r.Status, "não ligada a uma trilha")
			if r.Tipo == "modulo" || r.Tipo == "trilha" {
				situacao = fmt.Sprintf("%d de %d etapas concluídas", r.Concluidas, r.Total)
			}
			escreva("- %s: %s (%s)", rotulo, nome, situacao)
			if r.Tipo == "etapa" && r.EtapaID > 0 && licoes < 10 {
				if arq := a.licaoParaBlog(ctx, r.EtapaID); arq != "" {
					licoes++
					nomeArq := fmt.Sprintf("licoes/%02d.md", licoes)
					arquivos[nomeArq] = arq
					escreva(" · lição em `%s`", nomeArq)
				}
			}
			escreva("\n")
		}
	}
	for _, e := range it.Escolhido {
		rotulo := "Escolhido para estudar"
		if e.Situacao != "" {
			rotulo += " [" + e.Situacao + "]"
		}
		linha(rotulo, e.Refs, e.Fonte)
	}
	for _, l := range it.Estudo {
		linha("Candidato", l.Refs, l.Texto)
	}
	if it.Trilhas71 != nil {
		linha("Trilha do item", it.Trilhas71.Refs, it.Trilhas71.Texto)
	}
	if it.Livros71 != nil {
		linha("Livro do item", it.Livros71.Refs, it.Livros71.Texto)
	}
	for _, c := range it.Capitulos {
		linha("Capítulo", c.Refs, c.Livro+" cap. "+c.Caps+" · "+c.Assunto)
	}

	escreva("\n## As engrenagens (como o Lucas quer o artigo)\n\n")
	o := p.Opcoes
	escreva("- Horizonte: %s\n- Leitor: %s\n- Formato: %s\n", textoDaOpcao("horizonte", o.Horizonte), textoDaOpcao("nivel", o.Nivel), textoDaOpcao("formato", o.Formato))
	if p.Modo == "sugerir" {
		escreva("- Abrangência: %s\n", textoDaOpcao("abrangencia", o.Abrangencia))
	}
	escreva("- Tamanho: %s\n- Código: %s\n", textoDaOpcao("tamanho", o.Tamanho), textoDaOpcao("codigo", o.Codigo))

	ideias, err := a.ideiasDoBlog(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if len(ideias) > 0 {
		escreva("\n## Ideias que o Lucas já guardou ou escreveu (não repita; pode sugerir continuação)\n\n")
		for _, x := range ideias {
			escreva("- %s\n", x)
		}
	}
	if p.Pedido != "" {
		escreva("\n## Deste pedido\n\n%s\n", p.Pedido)
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, nil
}

// licaoParaBlog: a lição aprovada da etapa (vazio se não há).
func (a *app) licaoParaBlog(ctx context.Context, etapaID int64) string {
	var chave, titulo, slug string
	if err := a.db.QueryRowContext(ctx, `SELECT e.chave, e.titulo, t.slug FROM etapas e JOIN trilhas t ON t.id = e.trilha_id WHERE e.id = ?`,
		etapaID).Scan(&chave, &titulo, &slug); err != nil {
		return ""
	}
	texto, _ := a.conteudo.ler(caminhoLicao(slug, chave))
	if strings.TrimSpace(texto) == "" {
		return ""
	}
	return "<!-- etapa: " + titulo + " -->\n\n" + texto
}

// ideiasDoBlog: "título — tese [guardada/escrita · item]" das sugestões marcadas de todos os itens
// (menos as do próprio pedido e as descartadas).
func (a *app) ideiasDoBlog(ctx context.Context, exceto int64) ([]string, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasBlog+` WHERE marcas != '{}' AND id != ? ORDER BY id`, exceto)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ideias []string
	for rows.Next() {
		p, err := lerPedidoBlog(rows)
		if err != nil {
			return nil, err
		}
		var res resultadoBlog
		json.Unmarshal(p.Resultado, &res)
		for chave, m := range p.Marcas {
			if m.Estado == "descartada" {
				continue
			}
			n, _ := strconv.Atoi(chave)
			titulo, tese := p.Titulo, res.Tese
			if p.Modo == "sugerir" {
				if n >= len(res.Sugestoes) {
					continue
				}
				titulo, tese = res.Sugestoes[n].Titulo, res.Sugestoes[n].Tese
			}
			ideias = append(ideias, fmt.Sprintf("%s — %s [%s · item %s]", titulo, tese, m.Estado, p.ItemID))
		}
	}
	sort.Strings(ideias)
	return ideias, rows.Err()
}
