package main

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Projeto de estudo (T52): a leitura dos documentos do caderno (ROADMAP.md, ESTADO.md, PROMPT.md e
// itens/). O documento muda com o projeto, então o parser é tolerante: reconhece as peças pelo
// formato (cabeçalho, campo em negrito, tabela) e ignora o que não conhece, sem quebrar.

type faseProjeto struct {
	Titulo   string         `json:"titulo"`
	Objetivo string         `json:"objetivo,omitempty"`
	Itens    []*itemProjeto `json:"itens"`
	repo     string         // "Repositório: …" solto na fase (a trilha paralela do Rust)
}

type campoProjeto struct {
	Rotulo string `json:"rotulo"`
	Texto  string `json:"texto"`
}

// linhaEstudo: uma linha do "Estudar antes" (ou uma célula da 7.1) com as referências já ligadas
// às trilhas do app.
type linhaEstudo struct {
	Marca string              `json:"marca,omitempty"` // ✔ ◐ ⏳ ✖
	Texto string              `json:"texto"`
	Refs  []referenciaProjeto `json:"refs"`
}

// capituloProjeto: uma linha da 7.2 (capítulos de um livro) que cai neste item.
type capituloProjeto struct {
	Livro   string              `json:"livro"` // "T23"
	Caps    string              `json:"caps"`  // "1 e 2"
	Assunto string              `json:"assunto"`
	Revisao bool                `json:"revisao,omitempty"` // "(e revisto em 4.1)"
	Refs    []referenciaProjeto `json:"refs"`
}

// estudoEscolhido: uma linha da tabela "2. Estudo" do caderno do item, a lista que a IA montou com
// o Lucas na fase de estudo (a que vale; o roadmap só traz candidatos).
type estudoEscolhido struct {
	Fonte    string              `json:"fonte"`
	PorQue   string              `json:"por_que,omitempty"`
	Situacao string              `json:"situacao,omitempty"`
	Refs     []referenciaProjeto `json:"refs"`
}

type itemProjeto struct {
	ID             string            `json:"id"`
	Titulo         string            `json:"titulo"`
	Fase           string            `json:"fase"`
	Feito          bool              `json:"feito"`
	Situacao       string            `json:"situacao"` // a_fazer · em_andamento · feito
	Estado         string            `json:"estado"`   // a linha "Estado" do caderno do item
	Implementar    string            `json:"implementar"`
	Pronto         string            `json:"pronto"`
	Repositorio    string            `json:"repositorio"`
	Outros         []campoProjeto    `json:"outros"`
	Estudo         []linhaEstudo     `json:"estudo"`
	Livros71       *linhaEstudo      `json:"livros_71,omitempty"`
	Trilhas71      *linhaEstudo      `json:"trilhas_71,omitempty"`
	Capitulos      []capituloProjeto `json:"capitulos"`
	Escolhido      []estudoEscolhido `json:"estudo_escolhido,omitempty"`
	Caderno        string            `json:"caderno,omitempty"`
	CadernoArquivo string            `json:"caderno_arquivo,omitempty"`
	Link           string            `json:"link"`
	LinkCaderno    string            `json:"link_caderno,omitempty"`
	Markdown       string            `json:"markdown"` // o trecho do item no roadmap, como está
	Prompt         string            `json:"prompt"`
}

type livroRoadmap struct {
	ID     string // "T23"
	Titulo string
}

type roadmapProjeto struct {
	Fases  []*faseProjeto
	Itens  []*itemProjeto // na ordem do documento
	porID  map[string]*itemProjeto
	Livros []livroRoadmap
	Faixa  []blocoFaixa // a 7.3: os livros em ordem
}

// blocoFaixa: uma linha da faixa de leitura (ROADMAP §7.3), "T23 cap. 1 a 6 (o domínio…)".
type blocoFaixa struct {
	Numero     int                 `json:"numero"`
	Livro      string              `json:"livro"`   // "T23"
	Caps       string              `json:"caps"`    // "1 a 6"
	Assunto    string              `json:"assunto"` // o que está entre parênteses
	Antes      string              `json:"antes"`   // "de preferência antes de"
	Refs       []referenciaProjeto `json:"refs"`
	Concluidas int                 `json:"concluidas"`
	Total      int                 `json:"total"`
	Situacao   string              `json:"situacao"` // lido · atual · a_ler
}

type estadoProjeto struct {
	AtualizadoEm string `json:"atualizado_em"`
	ItemAtual    string `json:"item_atual"`
	ItemID       string `json:"item_id,omitempty"`
	Fase         string `json:"fase"`
	Proximos     string `json:"proximos"` // markdown
	ProximoID    string `json:"proximo_id,omitempty"`
}

var (
	reFaseProjeto  = regexp.MustCompile(`^(?:Fase\s+\d+|Trilha paralela)\b`)
	reItemProjeto  = regexp.MustCompile(`^(\d+\.\d+|R\d+)\.?\s+(.*)$`)
	reCampoProjeto = regexp.MustCompile(`^[-*]\s+\*\*([^*]+?):?\*\*:?\s*(.*)$`)
	reSubProjeto   = regexp.MustCompile(`^\s+[-*]\s+(.*)$`)
	reIDProjeto    = regexp.MustCompile(`R\d+|\d+\.(?:\d+|x)`)
	reFaixaIDs     = regexp.MustCompile(`(R\d+|\d+\.(?:\d+|x))(?:\s+a\s+(R\d+|\d+\.(?:\d+|x)))?`)
	reLivroProjeto = regexp.MustCompile(`^\*\*(T\d+)\s*·\s*([^*]+)\*\*`)
	reNegrito      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reParenteses   = regexp.MustCompile(`\([^()]*\)`)
	reEstadoCad    = regexp.MustCompile(`(?m)^[-*]\s+\*\*Estado:?\*\*:?\s*(.+)$`)
	reLeituraFaixa = regexp.MustCompile(`^(T\d+)\s+caps?\.?\s+([^(]+?)\s*(?:\((.+)\))?$`)
	reBlocoPrompt  = regexp.MustCompile("(?s)```text\\s*\\n(.*?)\\n```")
)

// normalizar: minúsculas, sem acento e só letras e números separados por um espaço
// ("Cap. 1 · Introdução" → "cap 1 introducao").
func normalizar(s string) string {
	var b strings.Builder
	espaco := true
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			espaco = false
		case !espaco:
			b.WriteByte(' ')
			espaco = true
		}
	}
	return strings.TrimSpace(b.String())
}

// ancoraGitHub: o id que o GitHub dá a um título ("0.1 O repositório" → "01-o-repositório").
func ancoraGitHub(titulo string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(titulo)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

type tabelaMD struct {
	secao  string // fase · 71 · 72
	fase   *faseProjeto
	livro  string
	linhas []string
}

func celulas(linha string) []string {
	linha = strings.TrimSpace(linha)
	linha = strings.TrimPrefix(linha, "|")
	linha = strings.TrimSuffix(linha, "|")
	partes := strings.Split(linha, "|")
	for i := range partes {
		partes[i] = strings.TrimSpace(partes[i])
	}
	return partes
}

// cabecalhoESeparador separa o cabeçalho e as linhas de dados (pula a linha "|---|").
func (t tabelaMD) cabecalhoELinhas() ([]string, [][]string) {
	if len(t.linhas) < 2 {
		return nil, nil
	}
	cab := celulas(t.linhas[0])
	var linhas [][]string
	for _, l := range t.linhas[1:] {
		c := celulas(l)
		if strings.Trim(strings.Join(c, ""), "-: ") == "" {
			continue
		}
		linhas = append(linhas, c)
	}
	return cab, linhas
}

// coluna acha a coluna pelo começo do nome normalizado; -1 se não houver.
func coluna(cab []string, nomes ...string) int {
	for i, c := range cab {
		n := normalizar(c)
		for _, nome := range nomes {
			if strings.HasPrefix(n, nome) {
				return i
			}
		}
	}
	return -1
}

func celula(linha []string, i int) string {
	if i < 0 || i >= len(linha) {
		return ""
	}
	return linha[i]
}

func tirarMarcaFeito(s string) (string, bool) {
	feito := strings.Contains(s, "✅")
	s = strings.ReplaceAll(s, "✅", "")
	return strings.Join(strings.Fields(s), " "), feito
}

// marcaEstudo separa o ✔ ◐ ⏳ ✖ do começo da linha.
func marcaEstudo(s string) (string, string) {
	s = strings.TrimSpace(s)
	r, n := utf8.DecodeRuneInString(s)
	if strings.ContainsRune("✔✓◐⏳✖✗", r) {
		resto := strings.TrimSpace(s[n:])
		resto = strings.TrimPrefix(resto, "️")
		return string(r), strings.TrimSpace(resto)
	}
	return "", s
}

// lerRoadmap lê o ROADMAP.md: fases (### Fase N / ### Trilha paralela), itens (#### N.M), a tabela
// da trilha paralela (itens R), a 7.1 (item a item) e a 7.2 (capítulos dos livros).
func lerRoadmap(texto string) *roadmapProjeto {
	r := &roadmapProjeto{porID: map[string]*itemProjeto{}}
	var (
		secao   string
		fase    *faseProjeto
		item    *itemProjeto
		campo   string
		tabelas []*tabelaMD
		tabela  *tabelaMD
		livro   string
	)
	novoItem := func(id, titulo string) *itemProjeto {
		titulo, feito := tirarMarcaFeito(titulo)
		it := &itemProjeto{ID: id, Titulo: titulo, Feito: feito, Fase: fase.Titulo,
			Outros: []campoProjeto{}, Estudo: []linhaEstudo{}, Capitulos: []capituloProjeto{}}
		if _, existe := r.porID[id]; !existe {
			r.porID[id] = it
			r.Itens = append(r.Itens, it)
			fase.Itens = append(fase.Itens, it)
		}
		return it
	}
	acrescentar := func(destino *string, s string) {
		if *destino == "" {
			*destino = s
		} else {
			*destino += "\n" + s
		}
	}

	for _, bruta := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		linha := strings.TrimRight(bruta, " \t")
		aparada := strings.TrimSpace(linha)

		if strings.HasPrefix(aparada, "|") {
			if tabela == nil {
				tabela = &tabelaMD{secao: secao, fase: fase, livro: livro}
				if item == nil || secao != "fase" {
					tabelas = append(tabelas, tabela)
				}
			}
			tabela.linhas = append(tabela.linhas, aparada)
			if item != nil && secao == "fase" {
				acrescentar(&item.Markdown, linha)
			}
			continue
		}
		tabela = nil

		switch {
		case strings.HasPrefix(aparada, "## "):
			secao, fase, item, livro = "", nil, nil, ""
			continue
		case strings.HasPrefix(aparada, "### "):
			titulo := strings.TrimSpace(aparada[4:])
			item, livro = nil, ""
			fase = nil
			n := normalizar(titulo)
			switch {
			case reFaseProjeto.MatchString(titulo):
				fase = &faseProjeto{Titulo: titulo, Itens: []*itemProjeto{}}
				r.Fases = append(r.Fases, fase)
				secao = "fase"
			case strings.Contains(n, "item a item"):
				secao = "71"
			case strings.Contains(n, "capitulo por capitulo"):
				secao = "72"
			case strings.Contains(n, "faixa de leitura"):
				secao = "73"
			default:
				secao = ""
			}
			continue
		case strings.HasPrefix(aparada, "#### "):
			item, campo = nil, ""
			if fase != nil {
				if m := reItemProjeto.FindStringSubmatch(strings.TrimSpace(aparada[5:])); m != nil {
					item = novoItem(m[1], m[2])
				}
			}
			continue
		}

		switch secao {
		case "fase":
			if item == nil {
				switch {
				case fase.Objetivo == "" && strings.HasPrefix(aparada, "*") && !strings.HasPrefix(aparada, "**") && strings.HasSuffix(aparada, "*"):
					fase.Objetivo = strings.Trim(aparada, "* ")
				case strings.HasPrefix(normalizar(aparada), "repositorio"):
					if i := strings.Index(aparada, ":"); i >= 0 {
						fase.repo = strings.TrimSuffix(strings.TrimSpace(aparada[i+1:]), ".")
					}
				}
				continue
			}
			if aparada == "---" {
				continue
			}
			acrescentar(&item.Markdown, linha)
			if m := reCampoProjeto.FindStringSubmatch(linha); m != nil {
				rotulo, valor := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
				switch normalizar(rotulo) {
				case "implementar":
					campo, item.Implementar = "implementar", valor
				case "estudar antes", "estudar":
					campo = "estudo"
					if valor != "" {
						marca, t := marcaEstudo(valor)
						item.Estudo = append(item.Estudo, linhaEstudo{Marca: marca, Texto: t})
					}
				case "pronto quando":
					campo, item.Pronto = "pronto", valor
				case "repositorio", "repositorios":
					campo, item.Repositorio = "repositorio", valor
				default:
					campo = "outro"
					item.Outros = append(item.Outros, campoProjeto{Rotulo: rotulo, Texto: valor})
				}
				continue
			}
			if aparada == "" {
				continue
			}
			texto := aparada
			if m := reSubProjeto.FindStringSubmatch(linha); m != nil {
				texto = strings.TrimSpace(m[1])
				if campo == "estudo" {
					marca, t := marcaEstudo(texto)
					item.Estudo = append(item.Estudo, linhaEstudo{Marca: marca, Texto: t})
					continue
				}
				texto = "- " + texto
			}
			switch campo {
			case "implementar":
				acrescentar(&item.Implementar, texto)
			case "pronto":
				acrescentar(&item.Pronto, texto)
			case "repositorio":
				acrescentar(&item.Repositorio, texto)
			case "outro":
				acrescentar(&item.Outros[len(item.Outros)-1].Texto, texto)
			case "estudo":
				if n := len(item.Estudo); n > 0 {
					item.Estudo[n-1].Texto += " " + texto
				}
			}
		case "72":
			if m := reLivroProjeto.FindStringSubmatch(aparada); m != nil {
				livro = m[1]
				r.Livros = append(r.Livros, livroRoadmap{ID: m[1], Titulo: strings.TrimSpace(m[2])})
			}
		}
	}

	for _, t := range tabelas {
		switch t.secao {
		case "fase":
			r.lerTabelaFase(t)
		case "71":
			r.lerTabela71(t)
		case "72":
			r.lerTabela72(t)
		case "73":
			r.lerTabela73(t)
		}
	}
	for _, f := range r.Fases {
		for _, it := range f.Itens {
			if it.Repositorio == "" {
				it.Repositorio = f.repo
			}
		}
	}
	return r
}

// lerTabelaFase: a trilha paralela tem os itens numa tabela (Item | Quando | Estudar | Implementar
// | Pronto quando).
func (r *roadmapProjeto) lerTabelaFase(t *tabelaMD) {
	cab, linhas := t.cabecalhoELinhas()
	cItem := coluna(cab, "item")
	if cItem < 0 || t.fase == nil {
		return
	}
	cQuando, cEstudar := coluna(cab, "quando"), coluna(cab, "estudar")
	cImpl, cPronto := coluna(cab, "implementar"), coluna(cab, "pronto")
	cRepo := coluna(cab, "repositorio")
	fase := t.fase
	for _, l := range linhas {
		bruto := celula(l, cItem)
		ids := reIDProjeto.FindAllString(bruto, 1)
		if len(ids) == 0 {
			continue
		}
		id := ids[0]
		titulo := reNegrito.ReplaceAllStringFunc(bruto, func(s string) string {
			if strings.Contains(s, id) {
				return ""
			}
			return strings.Trim(s, "*")
		})
		titulo = strings.Replace(titulo, id, "", 1)
		titulo = strings.TrimLeft(strings.TrimSpace(titulo), "·-–—: ")
		titulo, feito := tirarMarcaFeito(titulo)
		if _, existe := r.porID[id]; existe {
			continue
		}
		it := &itemProjeto{ID: id, Titulo: titulo, Feito: feito, Fase: fase.Titulo,
			Outros: []campoProjeto{}, Estudo: []linhaEstudo{}, Capitulos: []capituloProjeto{},
			Implementar: celula(l, cImpl), Pronto: celula(l, cPronto), Repositorio: celula(l, cRepo)}
		if q := celula(l, cQuando); q != "" {
			it.Outros = append(it.Outros, campoProjeto{Rotulo: "Quando", Texto: q})
		}
		if e := celula(l, cEstudar); e != "" && e != "—" {
			marca, txt := marcaEstudo(e)
			it.Estudo = append(it.Estudo, linhaEstudo{Marca: marca, Texto: txt})
		}
		it.Markdown = t.linhas[0] + "\n" + t.linhas[1] + "\n" + "| " + strings.Join(l, " | ") + " |"
		r.porID[id] = it
		r.Itens = append(r.Itens, it)
		fase.Itens = append(fase.Itens, it)
	}
}

// idsDoTexto expande "2.1 a 2.6", "7.6 e 8.x", "R1" nos itens que existem, na ordem do roadmap.
func (r *roadmapProjeto) idsDoTexto(texto string) []string {
	var ids []string
	visto := map[string]bool{}
	pos := map[string]int{}
	for i, it := range r.Itens {
		pos[it.ID] = i
	}
	add := func(id string) {
		if !visto[id] {
			visto[id] = true
			ids = append(ids, id)
		}
	}
	for _, m := range reFaixaIDs.FindAllStringSubmatch(texto, -1) {
		de, ate := m[1], m[2]
		if strings.HasSuffix(de, ".x") {
			prefixo := strings.TrimSuffix(de, "x")
			for _, it := range r.Itens {
				if strings.HasPrefix(it.ID, prefixo) {
					add(it.ID)
				}
			}
			continue
		}
		i, ok := pos[de]
		if !ok {
			continue
		}
		if ate == "" {
			add(de)
			continue
		}
		j, ok := pos[ate]
		if !ok || j < i {
			add(de)
			continue
		}
		for k := i; k <= j; k++ {
			add(r.Itens[k].ID)
		}
	}
	return ids
}

// lerTabela71: Item | Livros (capítulos) | Trilhas e cursos.
func (r *roadmapProjeto) lerTabela71(t *tabelaMD) {
	cab, linhas := t.cabecalhoELinhas()
	cItem, cLivros, cTrilhas := coluna(cab, "item"), coluna(cab, "livro"), coluna(cab, "trilha", "curso")
	if cItem < 0 {
		return
	}
	for _, l := range linhas {
		bruto := celula(l, cItem)
		chave := bruto
		if m := reNegrito.FindStringSubmatch(bruto); m != nil {
			chave = m[1]
		}
		for _, id := range r.idsDoTexto(chave) {
			it := r.porID[id]
			if v := celula(l, cLivros); v != "" && v != "—" && v != "-" {
				it.Livros71 = &linhaEstudo{Texto: v}
			}
			if v := celula(l, cTrilhas); v != "" && v != "—" && v != "-" {
				it.Trilhas71 = &linhaEstudo{Texto: v}
			}
		}
	}
}

// lerTabela72: Cap. | Assunto | Lido no item, de um livro (o "**T23 · …**" acima da tabela).
func (r *roadmapProjeto) lerTabela72(t *tabelaMD) {
	if t.livro == "" {
		return
	}
	cab, linhas := t.cabecalhoELinhas()
	cCap, cAssunto, cItem := coluna(cab, "cap"), coluna(cab, "assunto"), coluna(cab, "lido", "item")
	if cCap < 0 || cItem < 0 {
		return
	}
	for _, l := range linhas {
		caps := strings.ReplaceAll(celula(l, cCap), "*", "")
		lido := celula(l, cItem)
		// O que está entre parênteses com "revisto" é revisão; outros parênteses são comentário.
		revisao := map[string]bool{}
		for _, p := range reParenteses.FindAllString(lido, -1) {
			if strings.Contains(normalizar(p), "revist") {
				for _, id := range r.idsDoTexto(p) {
					revisao[id] = true
				}
			}
		}
		fora := reParenteses.ReplaceAllString(lido, " ")
		ids := r.idsDoTexto(fora)
		for id := range revisao {
			ids = append(ids, id)
		}
		for _, id := range ids {
			it := r.porID[id]
			it.Capitulos = append(it.Capitulos, capituloProjeto{Livro: t.livro, Caps: caps,
				Assunto: strings.ReplaceAll(celula(l, cAssunto), "**", ""), Revisao: revisao[id] && !strings.Contains(fora, id)})
		}
	}
	for _, it := range r.Itens {
		sort.SliceStable(it.Capitulos, func(i, j int) bool { return !it.Capitulos[i].Revisao && it.Capitulos[j].Revisao })
	}
}

// lerEstado lê o ESTADO.md: os campos em negrito e a seção "Próximos passos".
func lerEstado(texto string, r *roadmapProjeto) estadoProjeto {
	var e estadoProjeto
	var proximos []string
	dentro := false
	for _, linha := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		aparada := strings.TrimSpace(linha)
		if strings.HasPrefix(aparada, "#") {
			dentro = strings.HasPrefix(aparada, "## ") && strings.Contains(normalizar(aparada), "proximo")
			continue
		}
		if dentro {
			proximos = append(proximos, linha)
			continue
		}
		if m := reCampoProjeto.FindStringSubmatch(aparada); m != nil {
			valor := strings.TrimSpace(m[2])
			switch normalizar(m[1]) {
			case "atualizado em":
				e.AtualizadoEm = valor
			case "item atual", "item":
				e.ItemAtual = valor
			case "fase":
				e.Fase = valor
			}
		}
	}
	e.Proximos = strings.TrimSpace(strings.Join(proximos, "\n"))
	if r != nil {
		for _, id := range reIDProjeto.FindAllString(e.ItemAtual, -1) {
			if _, ok := r.porID[id]; ok {
				e.ItemID = id
				break
			}
		}
	}
	return e
}

// modeloPrompt tira o bloco ```text do PROMPT.md; sem ele, usa um modelo mínimo.
func modeloPrompt(texto string) string {
	if m := reBlocoPrompt.FindStringSubmatch(strings.ReplaceAll(texto, "\r\n", "\n")); m != nil {
		return m[1]
	}
	return "Projeto de estudo · item {{id}} — {{titulo}}\nFase atual: {{fase}}\n\n" +
		"Leia projeto-de-estudo/CLAUDE.md, METODOLOGIA.md, ESTADO.md e itens/{{arquivo}}.\n\n" +
		"Estudo indicado:\n{{estudo}}\n\nEtapas já concluídas na Trilha: {{concluidas}}\n"
}

// cadernoDoItem acha itens/<id>-*.md entre os arquivos do caderno.
func cadernoDoItem(arquivos map[string]string, id string) (string, string) {
	var achado string
	for caminho := range arquivos {
		if strings.HasPrefix(caminho, "itens/"+id+"-") && strings.HasSuffix(caminho, ".md") {
			if achado == "" || caminho < achado {
				achado = caminho
			}
		}
	}
	if achado == "" {
		return "", ""
	}
	return achado, arquivos[achado]
}

func estadoDoCaderno(texto string) string {
	if m := reEstadoCad.FindStringSubmatch(texto); m != nil {
		return strings.TrimSpace(strings.ReplaceAll(m[1], "**", ""))
	}
	return ""
}

// estudoDoCaderno lê a primeira tabela da seção "Estudo" do caderno (colunas Fonte, Por quê e
// Situação). A linha de exemplo do modelo ("Tn › Módulo › Etapa") e as linhas vazias ficam de fora.
func estudoDoCaderno(texto string) []estudoEscolhido {
	var t tabelaMD
	naSecao := false
	for _, linha := range strings.Split(texto, "\n") {
		l := strings.TrimSpace(linha)
		if strings.HasPrefix(l, "## ") {
			if naSecao {
				break
			}
			naSecao = strings.Contains(normalizar(l), "estudo")
			continue
		}
		if !naSecao {
			continue
		}
		if strings.HasPrefix(l, "|") {
			t.linhas = append(t.linhas, l)
		} else if len(t.linhas) > 0 {
			break
		}
	}
	cab, linhas := t.cabecalhoELinhas()
	if cab == nil {
		return nil
	}
	cFonte := coluna(cab, "fonte")
	if cFonte < 0 {
		cFonte = 0
	}
	cPorque, cSit := coluna(cab, "por que", "porque"), coluna(cab, "situacao")
	var lista []estudoEscolhido
	for _, l := range linhas {
		fonte := celula(l, cFonte)
		if fonte == "" || normalizar(fonte) == "tn modulo etapa" {
			continue
		}
		lista = append(lista, estudoEscolhido{
			Fonte: fonte, PorQue: celula(l, cPorque), Situacao: celula(l, cSit), Refs: []referenciaProjeto{},
		})
	}
	return lista
}

// lerTabela73: a faixa de leitura (# | Leitura | De preferência antes de). Linha que não segue
// "Tn cap. … (assunto)" é ignorada.
func (r *roadmapProjeto) lerTabela73(t *tabelaMD) {
	cab, linhas := t.cabecalhoELinhas()
	cLeitura, cAntes := coluna(cab, "leitura"), coluna(cab, "de preferencia", "antes")
	if cLeitura < 0 {
		return
	}
	for _, l := range linhas {
		m := reLeituraFaixa.FindStringSubmatch(strings.ReplaceAll(celula(l, cLeitura), "*", ""))
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(strings.TrimSpace(celula(l, 0)))
		r.Faixa = append(r.Faixa, blocoFaixa{Numero: n, Livro: m[1], Caps: strings.TrimSpace(m[2]),
			Assunto: strings.TrimSpace(m[3]), Antes: celula(l, cAntes), Refs: []referenciaProjeto{}})
	}
}
