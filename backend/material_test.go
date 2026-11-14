package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// appsFalsos: uma Biblioteca (um livro de 3 capítulos, indexado, com uma marcação no cap. 2) e um
// Acervo (um Tema e uma Referência), como as APIs de lá respondem.
func appsFalsos(t *testing.T) (biblioteca, acervo *httptest.Server, enviado *string) {
	t.Helper()
	var envio string
	biblioteca = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/books":
			w.Write([]byte(`[{"id":"livro1","title":"Redes de Computadores","author":"Kurose","format":"pdf","has_cover":true}]`))
		case r.Method == "POST" && r.URL.Path == "/api/books":
			r.ParseMultipartForm(1 << 20)
			f, h, err := r.FormFile("file")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"nenhum arquivo enviado"}`))
				return
			}
			b, _ := io.ReadAll(f)
			envio = h.Filename + ":" + string(b) + ":" + r.FormValue("title")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			w.Write([]byte(`{"id":"livro2","title":"Meu PDF","format":"pdf"}`))
		case r.URL.Path == "/api/books/livro1/chapters":
			w.Write([]byte(`{"id":"livro1","title":"Redes de Computadores","author":"Kurose","format":"pdf","indexed":true,"pages":300,"chapters":[
				{"n":1,"title":"Introdução","level":1,"first_page":1,"last_page":40,"chars":90000,"highlights":0,"anchor":"{\"pagina\":1}"},
				{"n":2,"title":"Camada de rede","level":1,"first_page":41,"last_page":90,"chars":120000,"highlights":1,"anchor":"{\"pagina\":41}"},
				{"n":3,"title":"NAT","level":2,"first_page":91,"last_page":100,"chars":20000,"highlights":0,"anchor":"{\"pagina\":91}"}]}`))
		case r.URL.Path == "/api/books/livro1/text":
			q := r.URL.Query()
			if q.Get("from") != "" {
				fmt.Fprintf(w, `{"parts":[{"title":"Páginas %s a %s","first_page":%s,"last_page":%s,"passages":[{"page":%s,"text":"Texto das páginas."}],"highlights":[]}]}`,
					q.Get("from"), q.Get("to"), q.Get("from"), q.Get("to"), q.Get("from"))
				return
			}
			var partes []string
			for _, n := range strings.Split(q.Get("chapters"), ",") {
				marcas := `[]`
				if n == "2" {
					marcas = `[{"quote":"O roteador encaminha pacotes","note":"importante","page":42}]`
				}
				partes = append(partes, fmt.Sprintf(`{"title":"Capítulo %s","first_page":1,"last_page":2,"passages":[{"page":42,"text":"Texto do capítulo %s."}],"highlights":%s}`, n, n, marcas))
			}
			w.Write([]byte(`{"parts":[` + strings.Join(partes, ",") + `]}`))
		case r.URL.Path == "/api/books/livro1/anchor":
			w.Write([]byte(`{"anchor":"{\"pagina\":` + r.URL.Query().Get("page") + `}"}`))
		case r.URL.Path == "/api/books/livro1/cover":
			w.Header().Set("Content-Type", "image/png")
			w.Write([]byte("PNG"))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(404)
			w.Write([]byte(`{"error":"livro não encontrado"}`))
		}
	}))
	acervo = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/temas":
			w.Write([]byte(`[{"caminho":"temas/redes/nat","titulo":"NAT","resumo":"Notas sobre NAT.","pai":"temas/redes","capitulos":2},{"caminho":"temas/redes","titulo":"Redes","resumo":"","pai":"","capitulos":0}]`))
		case "/api/pacotes/temas/redes/nat":
			w.Write([]byte(`{"tema":"temas/redes/nat","arquivos":{"temas/redes/nat/indice.md":{"conteudo":"# NAT\n\nÍndice do Tema."},"temas/redes/nat/cgnat.md":{"conteudo":"# CGNAT\n\nA operadora também faz NAT."}}}`))
		case "/api/referencias":
			w.Write([]byte(`[{"id":"r1","titulo":"Vídeo sobre NAT","tipo":"video","estado":"pronto","url":"https://youtu.be/x"},{"id":"r2","titulo":"","tipo":"artigo","estado":"novo"}]`))
		case "/api/referencias/r1":
			w.Write([]byte(`{"id":"r1","titulo":"Vídeo sobre NAT","tipo":"video","estado":"pronto","url":"https://youtu.be/x","textos":{"organizado":"Texto organizado do vídeo."}}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"erro":"Tema não encontrado."}`))
		}
	}))
	t.Cleanup(biblioteca.Close)
	t.Cleanup(acervo.Close)
	return biblioteca, acervo, &envio
}

func ligarApps(c *cliente, bib, ace *httptest.Server) {
	c.a.integra.biblioteca, c.a.integra.bibliotecaLink = bib.URL, "https://biblioteca.lan"
	c.a.integra.acervo, c.a.integra.acervoLink = ace.URL, "https://acervo.lan"
}

func TestTrechoDoLivro(t *testing.T) {
	casos := []struct {
		loc     string
		caps    []int
		de, ate int
	}{
		{"cap. 3", []int{3}, 0, 0},
		{"caps. 2, 4–6", []int{2, 4, 5, 6}, 0, 0},
		{"Capítulos 1 a 2", []int{1, 2}, 0, 0},
		{"p. 45–60", nil, 45, 60},
		{"pág. 12", nil, 12, 12},
		{"p. 30-10", nil, 30, 30},
		{"", nil, 0, 0},
		{"a seção sobre NAT", nil, 0, 0},
	}
	for _, x := range casos {
		caps, de, ate := trechoDoLivro(x.loc)
		if !reflect.DeepEqual(caps, x.caps) || de != x.de || ate != x.ate {
			t.Errorf("%q: %v %d %d", x.loc, caps, de, ate)
		}
	}
}

func TestEscolherMaterialDosOutrosApps(t *testing.T) {
	bib, ace, enviado := appsFalsos(t)
	c := novoCliente(t)
	var lig map[string]bool
	c.ok("GET", "/api/integracoes", nil, &lig)
	if lig["biblioteca"] || lig["acervo"] {
		t.Fatalf("sem configurar: %+v", lig)
	}
	if s := c.chamar("GET", "/api/biblioteca/livros", nil, nil); s != 502 {
		t.Fatalf("Biblioteca não configurada: %d", s)
	}
	ligarApps(c, bib, ace)
	c.ok("GET", "/api/integracoes", nil, &lig)
	if !lig["biblioteca"] || !lig["acervo"] {
		t.Fatalf("configurado: %+v", lig)
	}
	var livros []livroBiblioteca
	c.ok("GET", "/api/biblioteca/livros", nil, &livros)
	if len(livros) != 1 || livros[0].Title != "Redes de Computadores" {
		t.Fatalf("livros: %+v", livros)
	}
	var s sumarioLivro
	c.ok("GET", "/api/biblioteca/livros/livro1", nil, &s)
	if len(s.Chapters) != 3 || !s.Indexed {
		t.Fatalf("sumário: %+v", s)
	}
	var e map[string]string
	if st := c.chamar("GET", "/api/biblioteca/livros/outro", nil, &e); st != 404 || e["erro"] != "livro não encontrado" {
		t.Fatalf("livro que não existe: %d %+v", st, e)
	}
	var temas []temaAcervo
	c.ok("GET", "/api/acervo/temas", nil, &temas)
	if len(temas) != 2 || temas[0].Caminho != "temas/redes" {
		t.Fatalf("temas (em ordem): %+v", temas)
	}
	var refs []referenciaAcervo
	c.ok("GET", "/api/acervo/referencias", nil, &refs)
	if len(refs) != 1 || refs[0].ID != "r1" {
		t.Fatalf("referências com texto: %+v", refs)
	}

	// A capa passa como imagem.
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/biblioteca/livros/livro1/capa", nil))
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Body.String() != "PNG" {
		t.Fatalf("capa: %d %s %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}

	// O PDF passa direto para a Biblioteca; o erro de lá volta no formato daqui.
	var corpo bytes.Buffer
	mp := multipart.NewWriter(&corpo)
	mp.WriteField("title", "Meu PDF")
	parte, _ := mp.CreateFormFile("file", "meu.pdf")
	parte.Write([]byte("%PDF-1.4"))
	mp.Close()
	req := httptest.NewRequest("POST", "/api/biblioteca/livros", &corpo)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	rec = httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if rec.Code != 201 || *enviado != "meu.pdf:%PDF-1.4:Meu PDF" || !strings.Contains(rec.Body.String(), `"livro2"`) {
		t.Fatalf("enviar PDF: %d %s (%q)", rec.Code, rec.Body.String(), *enviado)
	}
	req = httptest.NewRequest("POST", "/api/biblioteca/livros", strings.NewReader("x"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=nada")
	rec = httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), `"erro":"nenhum arquivo enviado"`) {
		t.Fatalf("PDF sem arquivo: %d %s", rec.Code, rec.Body.String())
	}

	// Abrir o material: o livro no ponto (capítulo ou página), o Tema, a Referência.
	abrir := func(q string) string {
		rec := httptest.NewRecorder()
		c.h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/material/abrir?"+q, nil))
		if rec.Code != 302 {
			t.Fatalf("abrir %s: %d %s", q, rec.Code, rec.Body.String())
		}
		return rec.Header().Get("Location")
	}
	if l := abrir("tipo=livro&referencia=livro1&localizador=cap.%202"); l != `https://biblioteca.lan/#/livro/livro1?em=%7B%22pagina%22%3A41%7D` {
		t.Errorf("livro no capítulo: %s", l)
	}
	if l := abrir("tipo=livro&referencia=livro1&localizador=p.%2095"); l != `https://biblioteca.lan/#/livro/livro1?em=%7B%22pagina%22%3A95%7D` {
		t.Errorf("livro na página: %s", l)
	}
	if l := abrir("tipo=livro&referencia=livro1"); l != "https://biblioteca.lan/#/livro/livro1" {
		t.Errorf("livro do começo: %s", l)
	}
	if l := abrir("tipo=acervo&referencia=temas/redes/nat"); l != "https://acervo.lan/#/ler/temas/redes/nat/indice.md" {
		t.Errorf("Tema: %s", l)
	}
	if l := abrir("tipo=acervo&referencia=temas/redes/nat&localizador=cgnat.md"); l != "https://acervo.lan/#/ler/temas/redes/nat/cgnat.md" {
		t.Errorf("arquivo do Tema: %s", l)
	}
	if l := abrir("tipo=acervo&referencia=r1"); l != "https://acervo.lan/#/referencia/r1" {
		t.Errorf("Referência: %s", l)
	}
}

func TestLicaoRecebeOTextoDoLivroEDoAcervo(t *testing.T) {
	bib, ace, _ := appsFalsos(t)
	f := &iaFalsa{resposta: func(int, map[string]string) (string, error) {
		return `{"licao":"# NAT\n\nTexto.","notas":"","respostas":[]}`, nil
	}}
	c := clienteComIA(t, f)
	ligarApps(c, bib, ace)
	nat := trilhaComEtapa(c)
	var e etapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d", nat), nil, &e)
	e.Materiais = append(e.Materiais,
		material{Tipo: "livro", Titulo: "Redes de Computadores", Referencia: "livro1", Localizador: "cap. 2", Papel: "apoio"},
		material{Tipo: "acervo", Titulo: "NAT (Acervo)", Referencia: "temas/redes/nat", Papel: "apoio"},
		material{Tipo: "livro", Titulo: "Livro sem trecho", Referencia: "livro1", Papel: "aprofundar"},
		material{Tipo: "livro", Titulo: "Livro que sumiu", Referencia: "sumiu", Localizador: "cap. 1", Papel: "aprofundar"},
	)
	c.ok("PUT", fmt.Sprintf("/api/etapas/%d", nat), e, nil)
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{}, nil)
	c.a.fila.esvaziar(context.Background())
	if len(f.arquivos) != 1 {
		t.Fatalf("a IA não foi chamada")
	}
	arq, pedido := f.arquivos[0], f.arquivos[0]["pedido.md"]
	livro, tema, sumario := arq["material/02-livro.md"], arq["material/03-acervo.md"], arq["material/04-livro.md"]
	if !strings.Contains(livro, "[p. 42] Texto do capítulo 2.") || !strings.Contains(livro, "«O roteador encaminha pacotes» (p. 42) — nota dele: importante") {
		t.Errorf("texto do livro:\n%s", livro)
	}
	if !strings.Contains(tema, "Índice do Tema.") || !strings.Contains(tema, "A operadora também faz NAT.") || strings.Index(tema, "Índice") > strings.Index(tema, "CGNAT") {
		t.Errorf("texto do Tema (índice primeiro):\n%s", tema)
	}
	if !strings.Contains(sumario, "| 3 | — NAT | 91–100 | 20000 | 0 |") || !strings.Contains(sumario, "não diz o trecho") {
		t.Errorf("livro sem trecho vira sumário:\n%s", sumario)
	}
	for _, trecho := range []string{
		"[apoio · livro] Redes de Computadores (cap. 2) — texto em `material/02-livro.md`",
		"[apoio · acervo] NAT (Acervo) — texto em `material/03-acervo.md`",
		"Livro que sumiu (cap. 1) — o texto não veio (livro não encontrado)",
		"não copiar",
	} {
		if !strings.Contains(pedido, trecho) && !strings.Contains(pedido, strings.Replace(trecho, "não copiar", "sem copiar", 1)) {
			t.Errorf("o pedido não diz %q:\n%s", trecho, pedido)
		}
	}
}

func TestSugerirMaterial(t *testing.T) {
	bib, ace, _ := appsFalsos(t)
	f := &iaFalsa{resposta: func(n int, arq map[string]string) (string, error) {
		return `{"materiais":[
{"tipo":"web","titulo":"RFC 3022","referencia":"https://www.rfc-editor.org/rfc/rfc3022","localizador":"","papel":"apoio","nota":"[padrão] NAT tradicional","motivo":"o padrão"},
{"tipo":"web","titulo":"Link morto","referencia":"https://exemplo.org/x","localizador":"","papel":"apoio","nota":"","motivo":"x"},
{"tipo":"livro","titulo":"Redes de Computadores","referencia":"livro1","localizador":"cap. 3","papel":"principal","nota":"[livro] o capítulo de NAT","motivo":"seu livro"},
{"tipo":"livro","titulo":"Inventado","referencia":"livro9","localizador":"","papel":"apoio","nota":"","motivo":"x"},
{"tipo":"acervo","titulo":"NAT","referencia":"temas/redes/nat","localizador":"","papel":"aprofundar","nota":"","motivo":"suas notas"},
{"tipo":"documento","titulo":"WAN","referencia":"05-wan","localizador":"23-nat","papel":"apoio","nota":"","motivo":"já está"},
{"tipo":"documento","titulo":"Outro","referencia":"99-nada","localizador":"","papel":"apoio","nota":"","motivo":"x"}],
"notas":"Procurei na web e nos seus livros.","respostas":[]}`, nil
	}}
	c := clienteComIA(t, f)
	ligarApps(c, bib, ace)
	nat := trilhaComEtapa(c)
	var p proposta
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/materiais", nat), map[string]any{"pedido": "Algo sobre CGNAT."}, &p); s != 201 || p.Tipo != tipoMateriais {
		t.Fatalf("pedir material: %d %+v", s, p)
	}
	c.a.fila.esvaziar(context.Background())
	arq := f.arquivos[0]
	if !strings.Contains(arq["pedido.md"], "# Pedido: material da etapa \"NAT\"") || !strings.Contains(arq["pedido.md"], "[principal · documento] WAN") ||
		!strings.Contains(arq["biblioteca.md"], "| `livro1` | Redes de Computadores | Kurose | pdf |") || !strings.Contains(arq["acervo.md"], "`temas/redes/nat` — NAT: Notas sobre NAT.") ||
		!strings.Contains(f.pedidos[0].regra, "sugerir material") || !strings.Contains(arq["pedido.md"], "Deste pedido: Algo sobre CGNAT.") {
		t.Fatalf("pedido:\n%s\n%v", arq["pedido.md"], f.pedidos[0].regra[:40])
	}
	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=materiais", nat), nil, &lista)
	var cm conteudoMateriais
	json.Unmarshal(lista[0].Conteudo, &cm)
	if lista[0].Estado != propAguardando || len(cm.Materiais) != 4 {
		t.Fatalf("proposta: %s %+v", lista[0].Estado, cm)
	}
	for _, motivo := range []string{"Inventado\" (não está na Biblioteca)", "99-nada\" (a trilha não tem)", "WAN\" (já está na etapa)"} {
		if !strings.Contains(cm.Notas, motivo) {
			t.Errorf("as notas não dizem %q: %s", motivo, cm.Notas)
		}
	}
	if cm.Materiais[1].Situacao != "quebrado" || cm.Materiais[0].Situacao != "" {
		t.Errorf("links conferidos: %+v", cm.Materiais)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), map[string]any{"aceitos": []int{}}, nil); s != 400 {
		t.Fatalf("aprovar sem marcar: %d", s)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), map[string]any{"aceitos": []int{7}}, nil); s != 400 {
		t.Fatalf("aprovar índice fora: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), map[string]any{"aceitos": []int{2, 0, 2}}, &p)
	if p.Estado != propAprovada {
		t.Fatalf("estado: %s", p.Estado)
	}
	var e etapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d", nat), nil, &e)
	if len(e.Materiais) != 3 || e.Materiais[1].Referencia != "livro1" || e.Materiais[1].Localizador != "cap. 3" || e.Materiais[2].Titulo != "RFC 3022" {
		t.Fatalf("material da etapa: %+v", e.Materiais)
	}
}

func TestCriacaoAPartirDeMaterial(t *testing.T) {
	bib, ace, _ := appsFalsos(t)
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		if n == 1 {
			return `{"itens":[{"chave":"introducao","titulo":"Introdução às redes","nota":"","fonte":"Redes, cap. 1"},
				{"chave":"camada-de-rede","titulo":"Camada de rede","nota":"","fonte":"Redes, cap. 2–3"},
				{"chave":"cgnat","titulo":"CGNAT","nota":"","fonte":"Tema redes/nat"}],
				"sugestoes":[{"id":"s1","acao":"observar","item":"cgnat","titulo":"Raso","motivo":"as notas são curtas para o nível 4"},
				{"id":"s2","acao":"inserir","antes_de":"camada-de-rede","titulo":"Endereços IP","motivo":"o livro supõe"}],
				"nivel_partida":"","nivel_alvo":"","notas":"","respostas":[]}`, nil
		}
		return `{"trilha":{"titulo":"Redes pelo Kurose","objetivo":"x","descricao":"","recorte":"","modulos":[{"titulo":"M","etapas":[
			{"chave":"introducao","titulo":"Introdução","objetivo":"Explicar a internet.","bloom":"entender","minutos":40,"criterio":"x","requer":[],
			 "materiais":[{"tipo":"livro","titulo":"Redes de Computadores","referencia":"livro1","localizador":"cap. 1","papel":"principal","nota":""}]},
			{"chave":"camada-de-rede","titulo":"Camada de rede","objetivo":"Explicar o roteamento.","bloom":"entender","minutos":60,"criterio":"x","requer":["introducao"],
			 "materiais":[{"tipo":"livro","titulo":"Outro livro","referencia":"livro9","localizador":"cap. 2","papel":"principal","nota":""},
			              {"tipo":"acervo","titulo":"NAT","referencia":"temas/redes/nat","localizador":"","papel":"apoio","nota":""}]}]}]},
			"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	ligarApps(c, bib, ace)
	if s := c.chamar("POST", "/api/criacoes", map[string]any{"entrada": "material", "material": []map[string]any{{"tipo": "web", "titulo": "x", "referencia": "https://x"}}}, nil); s != 400 {
		t.Fatalf("material da web na entrada C: %d", s)
	}
	// Sem livro, sem Tema e sem documento: a IA não começa.
	var r respostaCriacao
	c.ok("POST", "/api/criacoes", map[string]any{"entrada": "material", "tema": "Vazio"}, &r)
	if s := c.chamar("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", r.Criacao.TrilhaID), map[string]any{}, nil); s != 400 {
		t.Fatalf("gerar sem material: %d", s)
	}

	c.ok("POST", "/api/criacoes", map[string]any{"entrada": "material", "pedido": "Para entender a rede de casa.", "profundidade": 4, "material": []map[string]any{
		{"tipo": "livro", "titulo": "Redes de Computadores", "referencia": "livro1", "localizador": "caps. 1-2"},
		{"tipo": "acervo", "titulo": "NAT", "referencia": "temas/redes/nat"},
	}}, &r)
	id := r.Criacao.TrilhaID
	if r.Criacao.Tema != "Redes de Computadores" || r.Criacao.Passo != tipoValidacao || len(r.Criacao.Material) != 2 || r.Criacao.Material[0].Papel != "principal" {
		t.Fatalf("criação: %+v", r.Criacao)
	}
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{}, &r)
	if !strings.HasPrefix(r.Propostas[0].Tarefa.Titulo, "Espinha dorsal: Redes de Computadores") {
		t.Errorf("título da tarefa: %s", r.Propostas[0].Tarefa.Titulo)
	}
	c.a.fila.esvaziar(context.Background())
	arq, pedido := f.arquivos[0], f.arquivos[0]["pedido.md"]
	if !strings.Contains(pedido, "# Pedido: espinha dorsal do material") || !strings.Contains(pedido, "Forma de criar: a partir de material") ||
		!strings.Contains(pedido, "- Livro **Redes de Computadores** (referência `livro1`), só caps. 1-2 — sumário em `material/livro-1-sumario.md`; texto de 2 parte(s) em `material/livro-1/`") ||
		!strings.Contains(pedido, "- Acervo **NAT** (referência `temas/redes/nat`) — texto em `material/acervo-2.md`") ||
		!strings.Contains(f.pedidos[0].regra, "espinha dorsal a partir de material") {
		t.Fatalf("pedido da espinha:\n%s", pedido)
	}
	if !strings.Contains(arq["material/livro-1/cap-02.md"], "[p. 42] Texto do capítulo 2.") || arq["material/livro-1/cap-03.md"] != "" ||
		!strings.Contains(arq["material/livro-1-sumario.md"], "| 2 | Camada de rede | 41–90 | 120000 | 1 |") {
		t.Fatalf("arquivos da espinha: %v", reflect.ValueOf(arq).MapKeys())
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	esp := r.Propostas[0]
	var cv conteudoValidacao
	json.Unmarshal(esp.Conteudo, &cv)
	if cv.Itens[1].Fonte != "Redes, cap. 2–3" {
		t.Fatalf("a fonte do item: %+v", cv.Itens)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", esp.ID), map[string]any{"aceitas": []string{"s2"}}, &r)
	c.a.fila.esvaziar(context.Background())
	plano := f.arquivos[1]["pedido.md"]
	if !strings.Contains(plano, "## A espinha dorsal aprovada") || !strings.Contains(plano, "- `camada-de-rede` — Camada de rede · fonte: Redes, cap. 2–3") ||
		!strings.Contains(plano, "inserir \"Endereços IP\"") || f.arquivos[1]["material/livro-1-sumario.md"] == "" || f.arquivos[1]["material/livro-1/cap-01.md"] != "" {
		t.Fatalf("pedido do plano (C):\n%s", plano)
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	var cp conteudoPlano
	json.Unmarshal(r.Propostas[1].Conteudo, &cp)
	var c10 []string
	for _, a := range cp.Revisao {
		if a.Criterio == "C10" {
			c10 = append(c10, a.Texto)
		}
	}
	if len(c10) != 1 || !strings.Contains(c10[0], "Outro livro") {
		t.Fatalf("revisão do material: %+v", cp.Revisao)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", r.Propostas[1].ID), map[string]any{}, &r)
	var tr trilha
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d", id), nil, &tr)
	m := tr.Modulos[0].Etapas[0].Materiais
	if r.Criacao.Passo != passoPronta || len(m) != 1 || m[0].Tipo != "livro" || m[0].Localizador != "cap. 1" {
		t.Fatalf("trilha criada: %s %+v", r.Criacao.Passo, m)
	}
}
