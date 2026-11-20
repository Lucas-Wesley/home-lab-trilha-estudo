package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Fase 6 (T27): material de apoio vindo dos outros apps, pela rede interna do Docker.
//
//	livro   referencia = id do livro na Biblioteca; localizador = "cap. 3", "caps. 2, 4–6" ou "p. 45–60"
//	acervo  referencia = pasta de um Tema ("temas/redes") ou id de uma Referência; localizador = um
//	        arquivo do Tema (opcional)
//
// A Trilha não guarda o texto: lê na hora de chamar a IA (lição, espinha dorsal) e escreve na pasta
// da tarefa, em material/. O link de cada material passa por GET /api/material/abrir, que acha o
// ponto do livro (a âncora do capítulo ou da página) e redireciona para a Biblioteca ou o Acervo.

const (
	limiteTextoMaterial = 300_000 // caracteres por material na pasta da tarefa
	limiteTextoEspinha  = 60_000  // por capítulo, na espinha dorsal (a IA julga o tamanho, não decora)
	limiteEspinhaTotal  = 1_500_000
)

// ---------- o que a Biblioteca devolve (R33 de lá) ----------

type livroBiblioteca struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Format   string `json:"format"`
	HasCover bool   `json:"has_cover"`
}

type capituloLivro struct {
	N          int    `json:"n"`
	Title      string `json:"title"`
	Level      int    `json:"level"`
	FirstPage  int    `json:"first_page"`
	LastPage   int    `json:"last_page"`
	Chars      int    `json:"chars"`
	Highlights int    `json:"highlights"`
	Anchor     string `json:"anchor"`
}

type sumarioLivro struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Author   string          `json:"author"`
	Format   string          `json:"format"`
	Indexed  bool            `json:"indexed"`
	Error    string          `json:"error,omitempty"`
	Pages    int             `json:"pages"`
	Chapters []capituloLivro `json:"chapters"`
}

type textoLivro struct {
	Parts []struct {
		Title     string `json:"title"`
		FirstPage int    `json:"first_page"`
		LastPage  int    `json:"last_page"`
		Passages  []struct {
			Page int    `json:"page"`
			Text string `json:"text"`
		} `json:"passages"`
		Highlights []struct {
			Quote string `json:"quote"`
			Note  string `json:"note"`
			Page  int    `json:"page"`
		} `json:"highlights"`
	} `json:"parts"`
}

// ---------- o que o Acervo devolve ----------

type temaAcervo struct {
	Caminho   string `json:"caminho"`
	Titulo    string `json:"titulo"`
	Resumo    string `json:"resumo"`
	Pai       string `json:"pai"`
	Capitulos int    `json:"capitulos"`
}

type referenciaAcervo struct {
	ID     string            `json:"id"`
	Titulo string            `json:"titulo"`
	Tipo   string            `json:"tipo"`
	Estado string            `json:"estado"`
	URL    string            `json:"url"`
	Textos map[string]string `json:"textos,omitempty"`
}

// ---------- telas: o que dá para escolher ----------

// handleIntegracoes: quais apps estão ligados (a tela esconde o que não está).
func (a *app) handleIntegracoes(w http.ResponseWriter, r *http.Request) {
	responderJSON(w, http.StatusOK, map[string]bool{
		"biblioteca": a.integra.biblioteca != "",
		"acervo":     a.integra.acervo != "",
		"flashcards": a.integra.flashcards != "",
	})
}

func (a *app) handleLivros(w http.ResponseWriter, r *http.Request) {
	livros := []livroBiblioteca{}
	if st, err := chamarApp(r.Context(), a.integra.biblioteca, "Biblioteca", http.MethodGet, "/api/books", nil, &livros); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, livros)
}

func (a *app) handleSumarioLivro(w http.ResponseWriter, r *http.Request) {
	s, st, err := a.sumarioDoLivro(r.Context(), r.PathValue("id"))
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, s)
}

func (a *app) sumarioDoLivro(ctx context.Context, id string) (sumarioLivro, int, error) {
	var s sumarioLivro
	ctx, cancelar := context.WithTimeout(ctx, 60*time.Second) // o PDF lê o sumário na hora (até 30 s lá)
	defer cancelar()
	st, err := chamarApp(ctx, a.integra.biblioteca, "Biblioteca", http.MethodGet, "/api/books/"+url.PathEscape(id)+"/chapters", nil, &s)
	s.Chapters = vazioSeNulo(s.Chapters)
	return s, st, err
}

// handleCapaLivro repassa a capa (a Biblioteca fica em outro endereço, com outro certificado).
func (a *app) handleCapaLivro(w http.ResponseWriter, r *http.Request) {
	a.repassar(w, r, a.integra.biblioteca, "Biblioteca", "/api/books/"+url.PathEscape(r.PathValue("id"))+"/cover", nil)
}

// handleEnviarLivro: um PDF (ou EPUB) próprio sobe para a Biblioteca (T15), e a Trilha só liga.
// O formulário passa direto, sem ser lido aqui.
func (a *app) handleEnviarLivro(w http.ResponseWriter, r *http.Request) {
	a.repassar(w, r, a.integra.biblioteca, "Biblioteca", "/api/books", r.Body)
}

// repassar: o pedido vai ao outro app como veio; a resposta volta como veio, com o erro no formato
// da Trilha ({"erro": ...}; a Biblioteca usa "error").
func (a *app) repassar(w http.ResponseWriter, r *http.Request, base, nome, caminho string, corpo io.Reader) {
	if base == "" {
		erroJSON(w, http.StatusServiceUnavailable, fmt.Sprintf("o %s não está configurado neste servidor", nome))
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), r.Method, base+caminho, corpo)
	if err != nil {
		erroInterno(w, err)
		return
	}
	if corpo != nil {
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		req.ContentLength = r.ContentLength
	}
	res, err := clienteApps.Do(req)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, fmt.Sprintf("o %s não respondeu; ele está no ar?", nome))
		return
	}
	defer res.Body.Close()
	tipo := res.Header.Get("Content-Type")
	if res.StatusCode >= 300 && strings.HasPrefix(tipo, "application/json") {
		var e struct {
			Erro  string `json:"erro"`
			Error string `json:"error"`
		}
		json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&e)
		erroJSON(w, res.StatusCode, primeiro(e.Erro, e.Error, fmt.Sprintf("o %s respondeu %d", nome, res.StatusCode)))
		return
	}
	if tipo != "" {
		w.Header().Set("Content-Type", tipo)
	}
	if cache := res.Header.Get("Cache-Control"); cache != "" {
		w.Header().Set("Cache-Control", cache)
	}
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}

func (a *app) handleTemasAcervo(w http.ResponseWriter, r *http.Request) {
	temas := []temaAcervo{}
	if st, err := chamarApp(r.Context(), a.integra.acervo, "Acervo", http.MethodGet, "/api/temas", nil, &temas); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	sort.SliceStable(temas, func(i, j int) bool { return temas[i].Caminho < temas[j].Caminho })
	responderJSON(w, http.StatusOK, temas)
}

// handleReferenciasAcervo: as Referências do Acervo que já têm texto (as que ainda estão sendo
// lidas ou falharam não servem de material).
func (a *app) handleReferenciasAcervo(w http.ResponseWriter, r *http.Request) {
	var todas []referenciaAcervo
	if st, err := chamarApp(r.Context(), a.integra.acervo, "Acervo", http.MethodGet, "/api/referencias", nil, &todas); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	lista := []referenciaAcervo{}
	for _, x := range todas {
		if strings.TrimSpace(x.Titulo) != "" && x.Estado != "novo" && x.Estado != "erro" {
			lista = append(lista, referenciaAcervo{ID: x.ID, Titulo: x.Titulo, Tipo: x.Tipo, Estado: x.Estado, URL: x.URL})
		}
	}
	responderJSON(w, http.StatusOK, lista)
}

// ---------- abrir um material ----------

// handleAbrirMaterial: GET /api/material/abrir?tipo=livro&referencia=<id>&localizador=cap.%203
// redireciona para o livro (no ponto) ou para o Tema/Referência do Acervo.
func (a *app) handleAbrirMaterial(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tipo, ref, loc := q.Get("tipo"), strings.TrimSpace(q.Get("referencia")), strings.TrimSpace(q.Get("localizador"))
	if ref == "" {
		erroJSON(w, http.StatusBadRequest, "material sem referência")
		return
	}
	var destino string
	switch tipo {
	case "livro":
		destino = a.integra.bibliotecaLink + "/#/livro/" + url.PathEscape(ref)
		if ancora := a.ancoraDoLivro(r.Context(), ref, loc); ancora != "" {
			destino += "?em=" + url.QueryEscape(ancora)
		}
	case "acervo":
		destino = linkDoAcervo(a.integra.acervoLink, ref, loc)
	default:
		erroJSON(w, http.StatusBadRequest, "só livro e Acervo abrem por aqui")
		return
	}
	http.Redirect(w, r, destino, http.StatusFound)
}

func linkDoAcervo(base, ref, loc string) string {
	if !strings.HasPrefix(ref, "temas/") {
		return base + "/#/referencia/" + url.PathEscape(ref)
	}
	arquivo := "indice.md"
	if strings.HasSuffix(loc, ".md") && !strings.Contains(loc, "..") {
		arquivo = strings.TrimPrefix(loc, "/")
	}
	return base + "/#/ler/" + ref + "/" + arquivo
}

// ancoraDoLivro: onde o trecho começa, no formato do leitor da Biblioteca (vazio = o começo).
func (a *app) ancoraDoLivro(ctx context.Context, id, loc string) string {
	caps, de, _ := trechoDoLivro(loc)
	if len(caps) > 0 {
		s, _, err := a.sumarioDoLivro(ctx, id)
		if err == nil && caps[0] >= 1 && caps[0] <= len(s.Chapters) {
			return s.Chapters[caps[0]-1].Anchor
		}
		return ""
	}
	if de > 0 {
		var r struct {
			Anchor string `json:"anchor"`
		}
		if _, err := chamarApp(ctx, a.integra.biblioteca, "Biblioteca", http.MethodGet,
			fmt.Sprintf("/api/books/%s/anchor?page=%d", url.PathEscape(id), de), nil, &r); err == nil {
			return r.Anchor
		}
	}
	return ""
}

var reNumeros = regexp.MustCompile(`(\d+)(?:\s*(?:-|–|—|a|até)\s*(\d+))?`)

// trechoDoLivro lê o localizador de um livro: "cap. 3", "caps. 2, 4–6", "capítulo 7" (capítulos,
// na numeração do sumário da Biblioteca) ou "p. 45–60", "pág. 12" (páginas). Outro texto: nada.
func trechoDoLivro(loc string) (caps []int, de, ate int) {
	l := strings.ToLower(strings.TrimSpace(loc))
	if l == "" {
		return nil, 0, 0
	}
	achados := reNumeros.FindAllStringSubmatch(l, -1)
	if len(achados) == 0 {
		return nil, 0, 0
	}
	if strings.HasPrefix(l, "p") {
		de, _ = strconv.Atoi(achados[0][1])
		ate = de
		if achados[0][2] != "" {
			ate, _ = strconv.Atoi(achados[0][2])
		}
		if ate < de {
			ate = de
		}
		return nil, de, ate
	}
	if !strings.HasPrefix(l, "cap") {
		return nil, 0, 0
	}
	vistos := map[int]bool{}
	for _, m := range achados {
		ini, _ := strconv.Atoi(m[1])
		fim := ini
		if m[2] != "" {
			fim, _ = strconv.Atoi(m[2])
		}
		for n := ini; n <= fim && len(caps) < 100; n++ {
			if n > 0 && !vistos[n] {
				vistos[n] = true
				caps = append(caps, n)
			}
		}
	}
	return caps, 0, 0
}

// ---------- o texto do material, para a IA ----------

func cortar(texto string, limite int) (string, bool) {
	if utf8.RuneCountInString(texto) <= limite {
		return texto, false
	}
	r := []rune(texto)
	return string(r[:limite]), true
}

// textoDoLivro devolve o trecho do livro em Markdown (com a página de cada parte e as marcações do
// Lucas), ou o sumário quando o material não diz o trecho.
func (a *app) textoDoLivro(ctx context.Context, id, loc string) (string, error) {
	s, _, err := a.sumarioDoLivro(ctx, id)
	if err != nil {
		return "", err
	}
	caps, de, ate := trechoDoLivro(loc)
	if len(caps) == 0 && de == 0 {
		return sumarioEmMarkdown(s) + "\n> O material não diz o trecho: use o sumário para achar a parte que serve a esta etapa e cite o capítulo e as páginas. O texto do livro não veio junto.\n", nil
	}
	if !s.Indexed {
		return sumarioEmMarkdown(s) + "\n> O texto deste livro não está disponível (" + padrao(s.Error, "ainda não indexado na Biblioteca") + "). Use só o sumário.\n", nil
	}
	caminho := fmt.Sprintf("/api/books/%s/text?from=%d&to=%d", url.PathEscape(id), de, ate)
	if len(caps) > 0 {
		var ns []string
		for _, n := range caps {
			if n <= len(s.Chapters) {
				ns = append(ns, strconv.Itoa(n))
			}
		}
		if len(ns) == 0 {
			return "", fmt.Errorf("o livro tem %d capítulos; %q não existe", len(s.Chapters), loc)
		}
		caminho = fmt.Sprintf("/api/books/%s/text?chapters=%s", url.PathEscape(id), strings.Join(ns, ","))
	}
	var t textoLivro
	ctx, cancelar := context.WithTimeout(ctx, 60*time.Second)
	defer cancelar()
	if _, err := chamarApp(ctx, a.integra.biblioteca, "Biblioteca", http.MethodGet, caminho, nil, &t); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s%s\n\nTrecho: %s. O número entre colchetes é a página do livro: cite-a.\n", s.Title, sufixo(" — ", s.Author), loc)
	for _, p := range t.Parts {
		fmt.Fprintf(&b, "\n## %s (p. %d–%d)\n\n", p.Title, p.FirstPage, p.LastPage)
		for _, x := range p.Passages {
			fmt.Fprintf(&b, "[p. %d] %s\n\n", x.Page, strings.TrimSpace(x.Text))
		}
		if len(p.Highlights) > 0 {
			b.WriteString("### Marcações do Lucas neste trecho (o que ele achou importante)\n\n")
			for _, h := range p.Highlights {
				fmt.Fprintf(&b, "- «%s» (p. %d)%s\n", strings.TrimSpace(h.Quote), h.Page, sufixo(" — nota dele: ", strings.TrimSpace(h.Note)))
			}
			b.WriteString("\n")
		}
	}
	texto, cortado := cortar(b.String(), limiteTextoMaterial)
	if cortado {
		texto += "\n\n> (O trecho foi cortado aqui: passa do limite. O resto está no livro.)\n"
	}
	return texto, nil
}

func sumarioEmMarkdown(s sumarioLivro) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s%s\n\n", s.Title, sufixo(" — ", s.Author))
	fmt.Fprintf(&b, "Formato: %s · %d páginas · id na Biblioteca: `%s`\n\n## Sumário\n\n", s.Format, s.Pages, s.ID)
	b.WriteString("| Cap. | Título | Páginas | Tamanho (caracteres) | Marcações do Lucas |\n|---|---|---|---|---|\n")
	for _, c := range s.Chapters {
		titulo := strings.Repeat("— ", max(c.Level-1, 0)) + strings.ReplaceAll(c.Title, "|", "/")
		fmt.Fprintf(&b, "| %d | %s | %d–%d | %d | %d |\n", c.N, titulo, c.FirstPage, c.LastPage, c.Chars, c.Highlights)
	}
	return b.String()
}

// textoDoAcervo: um Tema inteiro (o índice e os capítulos) ou um arquivo dele, ou o texto
// organizado de uma Referência.
func (a *app) textoDoAcervo(ctx context.Context, ref, loc string) (string, error) {
	if !strings.HasPrefix(ref, "temas/") {
		var x referenciaAcervo
		if _, err := chamarApp(ctx, a.integra.acervo, "Acervo", http.MethodGet, "/api/referencias/"+url.PathEscape(ref), nil, &x); err != nil {
			return "", err
		}
		texto := primeiro(x.Textos["organizado"], x.Textos["traducao"], x.Textos["cru"])
		if strings.TrimSpace(texto) == "" {
			return "", errors.New("a Referência ainda não tem texto")
		}
		texto, _ = cortar(fmt.Sprintf("# %s\n\nReferência do Acervo (%s)%s\n\n%s", x.Titulo, x.Tipo, sufixo(" · ", x.URL), texto), limiteTextoMaterial)
		return texto, nil
	}
	var p struct {
		Arquivos map[string]struct {
			Conteudo string `json:"conteudo"`
		} `json:"arquivos"`
	}
	if _, err := chamarApp(ctx, a.integra.acervo, "Acervo", http.MethodGet, "/api/pacotes/"+ref, nil, &p); err != nil {
		return "", err
	}
	var nomes []string
	for nome := range p.Arquivos {
		if loc == "" || strings.HasSuffix(nome, "/"+strings.TrimPrefix(loc, "/")) {
			nomes = append(nomes, nome)
		}
	}
	if len(nomes) == 0 {
		return "", fmt.Errorf("o Tema não tem o arquivo %q", loc)
	}
	// O índice primeiro; o resto em ordem.
	sort.Slice(nomes, func(i, j int) bool {
		ii, ij := strings.HasSuffix(nomes[i], "/indice.md"), strings.HasSuffix(nomes[j], "/indice.md")
		if ii != ij {
			return ii
		}
		return nomes[i] < nomes[j]
	})
	var b strings.Builder
	fmt.Fprintf(&b, "Tema do Acervo `%s` (notas do Lucas, revisadas por ele).\n", ref)
	for _, nome := range nomes {
		fmt.Fprintf(&b, "\n---\n\n<!-- arquivo: %s -->\n\n%s\n", nome, strings.TrimSpace(p.Arquivos[nome].Conteudo))
	}
	texto, cortado := cortar(b.String(), limiteTextoMaterial)
	if cortado {
		texto += "\n\n> (Cortado aqui: o Tema passa do limite.)\n"
	}
	return texto, nil
}

// lidoDoMaterial: o arquivo escrito para a IA, ou por que não deu.
type lidoDoMaterial struct {
	Arquivo string
	Erro    string
}

// arquivosDoMaterial lê o texto de cada material de livro ou do Acervo para a pasta da tarefa.
// Falhou um, a IA segue sem ele e o pedido diz por quê.
func (a *app) arquivosDoMaterial(ctx context.Context, materiais []material, arquivos map[string]string) map[int]lidoDoMaterial {
	lidos := map[int]lidoDoMaterial{}
	for i, m := range materiais {
		var texto string
		var err error
		switch m.Tipo {
		case "livro":
			if a.integra.biblioteca == "" {
				err = errors.New("a Biblioteca não está configurada")
			} else {
				texto, err = a.textoDoLivro(ctx, m.Referencia, m.Localizador)
			}
		case "acervo":
			if a.integra.acervo == "" {
				err = errors.New("o Acervo não está configurado")
			} else {
				texto, err = a.textoDoAcervo(ctx, m.Referencia, m.Localizador)
			}
		case "youtube":
			// T38: só os trechos dos intervalos, da transcrição da fonte da Entrada.
			texto, err = a.trechosDoVideo(ctx, m.Referencia, m.Localizador)
		default:
			continue
		}
		if err != nil {
			lidos[i] = lidoDoMaterial{Erro: err.Error()}
			continue
		}
		nome := fmt.Sprintf("material/%02d-%s.md", i+1, m.Tipo)
		arquivos[nome] = texto
		lidos[i] = lidoDoMaterial{Arquivo: nome}
	}
	return lidos
}

// descreverMaterial: a linha do material no pedido.md (o arquivo com o texto, quando há).
func descreverMaterial(m material, lido lidoDoMaterial) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- [%s · %s] %s", m.Papel, m.Tipo, m.Titulo)
	switch m.Tipo {
	case "documento":
		fmt.Fprintf(&b, " — arquivo `documentos/%s.md`", m.Referencia)
		if m.Localizador != "" {
			fmt.Fprintf(&b, ", seção com âncora `#%s`", m.Localizador)
		}
	case "livro", "acervo":
		if m.Localizador != "" {
			fmt.Fprintf(&b, " (%s)", m.Localizador)
		}
		switch {
		case lido.Arquivo != "":
			fmt.Fprintf(&b, " — texto em `%s`", lido.Arquivo)
		case lido.Erro != "":
			fmt.Fprintf(&b, " — o texto não veio (%s): não invente o conteúdo dele", lido.Erro)
		}
	default:
		if m.Referencia != "" {
			fmt.Fprintf(&b, " — %s", m.Referencia)
		}
		if m.Localizador != "" {
			fmt.Fprintf(&b, " (%s)", m.Localizador)
		}
	}
	if m.Nota != "" {
		fmt.Fprintf(&b, ". Nota: %s", m.Nota)
	}
	b.WriteString("\n")
	return b.String()
}

// ---------- a pasta da tarefa ----------

// gravarNaPasta escreve os arquivos do pedido (com subpastas, como material/).
func gravarNaPasta(pasta string, arquivos map[string]string) error {
	for nome, texto := range arquivos {
		caminho := filepath.Join(pasta, filepath.FromSlash(nome))
		if !strings.HasPrefix(caminho, filepath.Clean(pasta)+string(filepath.Separator)) {
			return fmt.Errorf("arquivo fora da pasta da tarefa: %q", nome)
		}
		if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(caminho, []byte(texto), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// ---------- entrada C: o material vira a espinha dorsal ----------

// arquivosDaEspinha escreve, para cada fonte da criação, o que a IA precisa para montar a sequência:
// de um livro, o sumário (com o tamanho de cada capítulo) e, com textoCompleto, o texto dos
// capítulos escolhidos (todos, se nenhum), cada um cortado num limite; de um Tema, o texto dele.
// Devolve as linhas do pedido.md que dizem onde está cada coisa.
func (a *app) arquivosDaEspinha(ctx context.Context, fontes []material, textoCompleto bool, arquivos map[string]string) string {
	var b strings.Builder
	total := 0
	for i, f := range fontes {
		n := i + 1
		switch f.Tipo {
		case "livro":
			fmt.Fprintf(&b, "- Livro **%s** (referência `%s`)", f.Titulo, f.Referencia)
			if f.Localizador != "" {
				fmt.Fprintf(&b, ", só %s", f.Localizador)
			}
			s, _, err := a.sumarioDoLivro(ctx, f.Referencia)
			if err != nil {
				fmt.Fprintf(&b, " — não deu para ler (%s)\n", err)
				continue
			}
			sumario := fmt.Sprintf("material/livro-%d-sumario.md", n)
			arquivos[sumario] = sumarioEmMarkdown(s)
			fmt.Fprintf(&b, " — sumário em `%s`", sumario)
			if !textoCompleto {
				b.WriteString("\n")
				continue
			}
			if !s.Indexed {
				fmt.Fprintf(&b, "; o texto não está disponível (%s), julgue pelo sumário\n", padrao(s.Error, "não indexado"))
				continue
			}
			caps, de, ate := trechoDoLivro(f.Localizador)
			if len(caps) == 0 && de == 0 {
				for _, c := range s.Chapters {
					caps = append(caps, c.N)
				}
			}
			pasta := fmt.Sprintf("material/livro-%d", n)
			lidos, cortados := 0, 0
			ler := func(nome, caminho string) {
				if total >= limiteEspinhaTotal {
					cortados++
					return
				}
				var t textoLivro
				c, cancelar := context.WithTimeout(ctx, 60*time.Second)
				defer cancelar()
				if _, err := chamarApp(c, a.integra.biblioteca, "Biblioteca", http.MethodGet, caminho, nil, &t); err != nil {
					return
				}
				var sb strings.Builder
				for _, p := range t.Parts {
					fmt.Fprintf(&sb, "# %s (p. %d–%d)\n\n", p.Title, p.FirstPage, p.LastPage)
					for _, x := range p.Passages {
						fmt.Fprintf(&sb, "[p. %d] %s\n\n", x.Page, strings.TrimSpace(x.Text))
					}
				}
				texto, cortado := cortar(sb.String(), limiteTextoEspinha)
				if cortado {
					texto += "\n\n> (Cortado aqui; o capítulo continua no livro. O tamanho real está no sumário.)\n"
				}
				arquivos[pasta+"/"+nome] = texto
				total += utf8.RuneCountInString(texto)
				lidos++
			}
			if len(caps) > 0 {
				for _, c := range caps {
					if c >= 1 && c <= len(s.Chapters) {
						ler(fmt.Sprintf("cap-%02d.md", c), fmt.Sprintf("/api/books/%s/text?chapters=%d", url.PathEscape(f.Referencia), c))
					}
				}
			} else {
				ler(fmt.Sprintf("p-%d-%d.md", de, ate), fmt.Sprintf("/api/books/%s/text?from=%d&to=%d", url.PathEscape(f.Referencia), de, ate))
			}
			fmt.Fprintf(&b, "; texto de %d parte(s) em `%s/`", lidos, pasta)
			if cortados > 0 {
				fmt.Fprintf(&b, " (%d ficaram de fora pelo limite total: julgue-as pelo sumário)", cortados)
			}
			b.WriteString("\n")
		case "acervo":
			fmt.Fprintf(&b, "- Acervo **%s** (referência `%s`)", f.Titulo, f.Referencia)
			if !textoCompleto {
				b.WriteString("\n")
				continue
			}
			texto, err := a.textoDoAcervo(ctx, f.Referencia, f.Localizador)
			if err != nil {
				fmt.Fprintf(&b, " — não deu para ler (%s)\n", err)
				continue
			}
			nome := fmt.Sprintf("material/acervo-%d.md", n)
			arquivos[nome] = texto
			total += utf8.RuneCountInString(texto)
			fmt.Fprintf(&b, " — texto em `%s`\n", nome)
		}
	}
	return b.String()
}
