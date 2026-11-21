package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// O ROADMAP.md, o ESTADO.md e o PROMPT.md de testdata/projeto são cópias reais do caderno
// (21/11/2026): o parser tem de aguentar o documento como ele é.

func lerFixtureProjeto(t *testing.T, nome string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "projeto", nome))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRoadmapReal(t *testing.T) {
	r := lerRoadmap(lerFixtureProjeto(t, "ROADMAP.md"))
	if len(r.Fases) != 11 {
		t.Fatalf("fases = %d, esperava 11 (Fase 0 a 9 + a trilha paralela)", len(r.Fases))
	}
	if len(r.Itens) != 60 {
		t.Fatalf("itens = %d, esperava 60 (56 + R1 a R4)", len(r.Itens))
	}
	if r.Fases[0].Objetivo == "" || !strings.HasPrefix(r.Fases[0].Titulo, "Fase 0") {
		t.Errorf("fase 0: %+v", r.Fases[0])
	}

	it := r.porID["0.1"]
	if it == nil || it.Titulo != "O repositório e o ambiente" || it.Repositorio != "`banco`" ||
		!strings.Contains(it.Implementar, "go.mod") || !strings.Contains(it.Pronto, "make test") {
		t.Fatalf("0.1: %+v", it)
	}
	if len(it.Estudo) != 4 || it.Estudo[0].Marca != "✔" || it.Estudo[0].Texto != "T13 › Fundamentos › Por que Go e o ambiente" {
		t.Errorf("0.1 estudo: %+v", it.Estudo)
	}
	if it.Livros71 == nil || !strings.HasPrefix(it.Livros71.Texto, "T21 cap. 1, 2 e 21") || it.Trilhas71 == nil {
		t.Errorf("0.1 sem a linha da 7.1: %+v %+v", it.Livros71, it.Trilhas71)
	}
	if len(it.Capitulos) != 1 || it.Capitulos[0].Livro != "T21" || it.Capitulos[0].Caps != "1, 2 e 21" {
		t.Errorf("0.1 capítulos: %+v", it.Capitulos)
	}

	// 6.4 tem campos fora do padrão ("Por quê aqui", "Antes").
	if o := r.porID["6.4"].Outros; len(o) != 2 || o[0].Rotulo != "Por quê aqui" {
		t.Errorf("6.4 outros: %+v", o)
	}
	// A trilha paralela: itens na tabela e o repositório solto abaixo dela.
	r1 := r.porID["R1"]
	if r1 == nil || r1.Titulo != "Primeiro Rust" || !strings.Contains(r1.Repositorio, "rust-estudos") ||
		len(r1.Estudo) != 1 || r1.Trilhas71 == nil || len(r1.Outros) != 1 || r1.Outros[0].Rotulo != "Quando" {
		t.Fatalf("R1: %+v", r1)
	}
	// Intervalos da 7.1: "2.1 a 2.6" e "7.6 e 8.x".
	if r.porID["2.4"].Trilhas71 == nil || r.porID["2.7"].Trilhas71.Texto == r.porID["2.4"].Trilhas71.Texto {
		t.Error("2.1 a 2.6 não chegou ao 2.4 (ou passou do 2.6)")
	}
	if r.porID["8.3"].Trilhas71 == nil || !strings.Contains(r.porID["8.3"].Trilhas71.Texto, "T43") {
		t.Error("7.6 e 8.x não chegou ao 8.3")
	}
	if r.porID["7.2"].Trilhas71 != nil {
		t.Error("7.2 não tem linha na 7.1")
	}
	// 7.2: "1.2 (e revisto em 4.1)" — revisão no 4.1.
	var revisto bool
	for _, c := range r.porID["4.1"].Capitulos {
		if c.Livro == "T23" && c.Caps == "8" {
			revisto = c.Revisao
		}
	}
	if !revisto {
		t.Errorf("T23 cap. 8 deveria ser revisão no 4.1: %+v", r.porID["4.1"].Capitulos)
	}
	// Os três livros cobertos inteiros: 27 + 14 + 18 capítulos distintos.
	caps := map[string]map[int]bool{}
	for _, it := range r.Itens {
		for _, c := range it.Capitulos {
			if caps[c.Livro] == nil {
				caps[c.Livro] = map[int]bool{}
			}
			for _, n := range numerosDe(c.Caps) {
				caps[c.Livro][n] = true
			}
		}
	}
	if len(caps["T21"]) != 27 || len(caps["T22"]) != 14 || len(caps["T23"]) != 18 {
		t.Errorf("capítulos: T21 %d, T22 %d, T23 %d", len(caps["T21"]), len(caps["T22"]), len(caps["T23"]))
	}
	if len(r.Livros) != 3 {
		t.Errorf("livros da 7.2: %+v", r.Livros)
	}

	e := lerEstado(lerFixtureProjeto(t, "ESTADO.md"), r)
	if e.AtualizadoEm != "21/11/2026" || e.Fase != "preparação" || e.ItemID != "" ||
		!strings.Contains(e.Proximos, "Primeiro item") {
		t.Errorf("estado: %+v", e)
	}
	if !strings.Contains(modeloPrompt(lerFixtureProjeto(t, "PROMPT.md")), "{{estudo}}") {
		t.Error("o modelo do PROMPT.md não veio")
	}
}

func TestRoadmapTolerante(t *testing.T) {
	r := lerRoadmap("# Nada\n\n### Fase 1: algo\n#### 1.1 Sem campos ✅\ntexto solto\n#### x Não é item\n## Fim\n")
	if len(r.Itens) != 1 || !r.Itens[0].Feito || r.Itens[0].Titulo != "Sem campos" {
		t.Fatalf("itens: %+v", r.Itens)
	}
	if lerRoadmap("").Itens != nil {
		t.Error("vazio")
	}
}

func TestLerReferencias(t *testing.T) {
	casos := []struct {
		texto string
		alvos []string
	}{
		{"T13 › Concorrência › Goroutines; Channels e select", []string{"T13 › Concorrência › Goroutines", "T13 › Concorrência › Channels e select"}},
		{"T22 cap. 3, 4 e 8 (modelos)", []string{"T22 cap. 3", "T22 cap. 4", "T22 cap. 8"}},
		{"T22 cap. 6 a 8", []string{"T22 cap. 6", "T22 cap. 7", "T22 cap. 8"}},
		{"T31 e T28 (objetos de valor)", []string{"T31", "T28"}},
		{"**T38 › PIX** · T32 › Bônus · Comunicação e resiliência", []string{"T38 › PIX", "T32 › Bônus · Comunicação e resiliência"}},
		{"T24 › 06, 17 SQLC, 18 UOW", []string{"T24 › 06", "T24 › 17 SQLC", "T24 › 18 UOW"}},
		{"T10 › Números e bits › Ponto flutuante (IEEE 754). É aqui que você vê.", []string{"T10 › Números e bits › Ponto flutuante"}},
		{"T22 cap. 4 (log, LSM) e T37 › Por dentro de um banco", []string{"T22 cap. 4", "T37 › Por dentro de um banco"}},
		{"Prometheus na prática", nil},
	}
	for _, c := range casos {
		var obtidos []string
		for _, a := range lerReferencias(c.texto) {
			obtidos = append(obtidos, a.Texto)
		}
		if strings.Join(obtidos, " | ") != strings.Join(c.alvos, " | ") {
			t.Errorf("%q:\n obtive  %q\n esperava %q", c.texto, obtidos, c.alvos)
		}
	}
	// "10 e 33" são dois números; "módulos 1 a 3", três.
	if a := lerReferencias("T25 › 10 e 33"); len(a) != 2 || a[1].Caminho[0].Num != 33 {
		t.Errorf("10 e 33: %+v", a)
	}
	if a := lerReferencias("T36 › módulos 1 a 3"); len(a) != 3 || !a[2].Caminho[0].Modulo || a[2].Caminho[0].Num != 3 {
		t.Errorf("módulos 1 a 3: %+v", a)
	}
}

// catalogoDeTeste imita as trilhas reais (títulos copiados do banco).
func catalogoDeTeste() catalogoTrilhas {
	c := catalogoTrilhas{}
	t13 := c.trilha(13, "Golang do zero ao backend em produção")
	m := t13.modulo(1, "Concorrência")
	m.etapa(1, "Goroutines e o modelo de concorrência", "concluida")
	m.etapa(2, "Channels e select", "disponivel")
	t21 := c.trilha(21, "Livro: Fundamentos da Arquitetura de Software")
	m = t21.modulo(2, "Introdução")
	m.etapa(3, "Cap. 1 · Introdução", "concluida")
	m = t21.modulo(3, "Parte I · Fundamentos")
	m.etapa(4, "Cap. 2 · Pensamento arquitetural", "disponivel")
	t24 := c.trilha(24, "Go Expert (Full Cycle)")
	m = t24.modulo(4, "09 - APIs")
	m.etapa(5, "1-Falando sobre APIs", "disponivel")
	m = t24.modulo(5, "10 - Multithreading")
	m.etapa(6, "1-Iniciando com Processos", "concluida")
	t32 := c.trilha(32, "Microsserviços, eventos e Kafka")
	m = t32.modulo(6, "011 - Microsserviços e Arquitetura baseada a eventos")
	m.etapa(7, "002 - Introdução a Microserviços", "disponivel")
	m = t32.modulo(7, "Bônus · Comunicação e resiliência")
	m.etapa(8, "001 - Introdução", "disponivel")
	t36 := c.trilha(36, "Rust do zero")
	t36.modulo(8, "Primeiros passos")
	t36.modulo(9, "Ownership: o coração do Rust")
	t18 := c.trilha(18, "System Design T4")
	m = t18.modulo(10, "Módulo 1: Fundamentos e protocolos de rede")
	m.etapa(9, "DNS (Domain Name Service)", "disponivel")
	t9 := c.trilha(9, "DDD")
	t9.modulo(11, "Tático I: o modelo em código").etapa(10, "Value objects", "disponivel")
	t9.modulo(12, "Tático II: serviços, eventos e persistência")
	return c
}

func TestCasarReferencias(t *testing.T) {
	c := catalogoDeTeste()
	casos := []struct {
		texto, tipo, caminho, status string
		parcial                      bool
	}{
		{"T13 › Concorrência › Goroutines", "etapa", "T13 › Concorrência › Goroutines e o modelo de concorrência", "concluida", false},
		{"T13 › Concorrência", "modulo", "T13 › Concorrência", "", false},
		{"T21 cap. 1", "etapa", "T21 › Introdução › Cap. 1 · Introdução", "concluida", false},
		{"T24 › 10 Multithreading", "modulo", "T24 › 10 - Multithreading", "concluida", false},
		{"T32 › 002", "etapa", "T32 › 011 - Microsserviços e Arquitetura baseada a eventos › 002 - Introdução a Microserviços", "disponivel", false},
		{"T32 › Bônus · Comunicação e resiliência", "modulo", "T32 › Bônus · Comunicação e resiliência", "", false},
		{"T36 › módulos 2", "modulo", "T36 › Ownership: o coração do Rust", "", false},
		{"T36 › Ownership", "modulo", "T36 › Ownership: o coração do Rust", "", false},
		{"T18 › M1 › DNS", "etapa", "T18 › Módulo 1: Fundamentos e protocolos de rede › DNS (Domain Name Service)", "disponivel", false},
		{"T9 › Tático I › Value objects", "etapa", "T9 › Tático I: o modelo em código › Value objects", "disponivel", false},
		{"T13 › Concorrência › Inexistente", "modulo", "T13 › Concorrência", "", true},
		{"T13 › Nada disso", "trilha", "T13 · Golang do zero ao backend em produção", "", true},
		{"T99 › Trilha que não existe", "", "", "", false},
	}
	for _, k := range casos {
		refs := c.referencias(k.texto)
		if len(refs) != 1 {
			t.Errorf("%q: %d refs", k.texto, len(refs))
			continue
		}
		r := refs[0]
		if r.Tipo != k.tipo || r.Caminho != k.caminho || r.Status != k.status || r.Parcial != k.parcial {
			t.Errorf("%q: obtive tipo=%q caminho=%q status=%q parcial=%v", k.texto, r.Tipo, r.Caminho, r.Status, r.Parcial)
		}
	}
	// "Tático I" não pode cair no "Tático II".
	if r := c.referencias("T9 › Tático I")[0]; r.ModuloID != 11 {
		t.Errorf("Tático I casou com o módulo %d", r.ModuloID)
	}
}

// pacoteDeTeste monta um tar.gz como o do GitHub (com a pasta de cima).
func pacoteDeTeste(t *testing.T, arquivos map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "projeto-de-estudo-main/", Typeflag: tar.TypeDir, Mode: 0o755})
	for nome, conteudo := range arquivos {
		if err := tw.WriteHeader(&tar.Header{Name: "projeto-de-estudo-main/" + nome, Typeflag: tar.TypeReg,
			Mode: 0o644, Size: int64(len(conteudo))}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(conteudo))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestAPIProjeto(t *testing.T) {
	pacote := pacoteDeTeste(t, map[string]string{
		"ROADMAP.md":                      lerFixtureProjeto(t, "ROADMAP.md"),
		"ESTADO.md":                       strings.Replace(lerFixtureProjeto(t, "ESTADO.md"), "nenhum (o projeto ainda não começou)", "0.1 · O repositório e o ambiente", 1),
		"PROMPT.md":                       lerFixtureProjeto(t, "PROMPT.md"),
		"itens/_MODELO.md":                "# modelo",
		"itens/0.2-dinheiro-sem-float.md": "# 0.2\n\n- **Estado:** checagem\n- **Onde paramos:** x\n",
		"codigo/nao-md.go":                "package x",
	})
	var falhar atomic.Bool
	var pedidos atomic.Int32
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pedidos.Add(1)
		if falhar.Load() {
			http.Error(w, "fora", http.StatusServiceUnavailable)
			return
		}
		w.Write(pacote)
	}))
	defer gh.Close()

	c := novoCliente(t)
	pasta := t.TempDir()
	c.a.projeto = &cadernoProjeto{repo: "https://github.com/Lucas-Wesley/projeto-de-estudo", ramo: "main",
		tarball: gh.URL, pasta: pasta, intervalo: time.Hour, cliente: gh.Client()}

	// Trilha 21 com o capítulo 1 concluído (ids fixos: o roadmap fala em T21).
	agora := agora()
	exec := func(q string, args ...any) {
		if _, err := c.a.db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO trilhas (id, slug, titulo, criado_em, atualizado_em) VALUES (21, 'fa', 'Livro: Fundamentos', ?, ?)`, agora, agora)
	exec(`INSERT INTO modulos (id, trilha_id, ordem, titulo) VALUES (1, 21, 0, 'Introdução'), (2, 21, 1, 'Parte I')`)
	exec(`INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo, status) VALUES
		(21, 1, 0, 'c1', 'Cap. 1 · Introdução', 'concluida'), (21, 2, 0, 'c2', 'Cap. 2 · Pensamento arquitetural', 'disponivel')`)

	var resp respostaProjeto
	if st := c.chamar("GET", "/api/projeto", nil, &resp); st != http.StatusOK {
		t.Fatalf("status %d", st)
	}
	if len(resp.Fases) != 11 || resp.Aviso != "" || resp.BaixadoEm == "" {
		t.Fatalf("resposta: fases %d aviso %q", len(resp.Fases), resp.Aviso)
	}
	if resp.Estado.ItemID != "0.1" || resp.Estado.ProximoID != "0.1" {
		t.Errorf("estado: %+v", resp.Estado)
	}
	itens := map[string]*itemProjeto{}
	for _, f := range resp.Fases {
		for _, it := range f.Itens {
			itens[it.ID] = it
		}
	}
	i01, i02 := itens["0.1"], itens["0.2"]
	if i01.Situacao != "em_andamento" || i02.Situacao != "em_andamento" || i02.Estado != "checagem" ||
		i02.CadernoArquivo != "0.2-dinheiro-sem-float.md" || itens["0.3"].Situacao != "a_fazer" {
		t.Errorf("situação: 0.1 %q · 0.2 %q/%q/%q · 0.3 %q", i01.Situacao, i02.Situacao, i02.Estado, i02.CadernoArquivo, itens["0.3"].Situacao)
	}
	if !strings.HasSuffix(i02.LinkCaderno, "/blob/main/itens/0.2-dinheiro-sem-float.md") ||
		!strings.HasSuffix(i01.Link, "ROADMAP.md#01-o-repositório-e-o-ambiente") {
		t.Errorf("links: %q %q", i02.LinkCaderno, i01.Link)
	}
	cap1 := i01.Capitulos[0].Refs[0]
	if cap1.Tipo != "etapa" || cap1.Status != "concluida" || cap1.TrilhaID != 21 {
		t.Errorf("T21 cap. 1: %+v", cap1)
	}
	// O prompt: campos trocados, a fase do ESTADO para o item atual, o caderno que existe.
	p := i01.Prompt
	for _, trecho := range []string{"item 0.1 — O repositório e o ambiente", "Fase atual: preparação",
		"itens/0.1-o-repositorio-e-o-ambiente.md", "- ✔ T13 › Fundamentos › Por que Go e o ambiente",
		"Livros (seção 7.1): T21 › Introdução › Cap. 1 · Introdução", "Etapas já concluídas na Trilha: \n- T21 › Introdução › Cap. 1 · Introdução"} {
		if !strings.Contains(p, trecho) {
			t.Errorf("prompt sem %q:\n%s", trecho, p)
		}
	}
	if strings.Contains(p, "{{") {
		t.Errorf("sobrou campo no prompt:\n%s", p)
	}
	if !strings.Contains(i02.Prompt, "Fase atual: checagem") || !strings.Contains(i02.Prompt, "itens/0.2-dinheiro-sem-float.md") ||
		!strings.Contains(itens["0.3"].Prompt, "Fase atual: problema") || !strings.Contains(itens["0.3"].Prompt, "Etapas já concluídas na Trilha: nenhuma") {
		t.Error("prompt do 0.2/0.3 errado")
	}
	if len(resp.Livros) != 3 || resp.Livros[0].ID != "T21" || resp.Livros[0].Concluidas != 1 || resp.Livros[0].Total != 2 {
		t.Errorf("livros: %+v", resp.Livros)
	}
	if resp.Referencias.Casadas == 0 || resp.Referencias.NaoCasadas == 0 {
		t.Errorf("contagem: %+v", resp.Referencias)
	}

	// Dentro do intervalo não baixa de novo; o botão força.
	c.chamar("GET", "/api/projeto", nil, nil)
	if pedidos.Load() != 1 {
		t.Errorf("baixou %d vezes", pedidos.Load())
	}
	falhar.Store(true)
	resp = respostaProjeto{}
	if st := c.chamar("POST", "/api/projeto/atualizar", nil, &resp); st != http.StatusOK || !strings.Contains(resp.Aviso, "503") || len(resp.Fases) != 11 {
		t.Fatalf("GitHub fora: status %d aviso %q fases %d", st, resp.Aviso, len(resp.Fases))
	}

	// Outro processo (o app reiniciado) com o GitHub fora: serve a cópia do disco, com aviso.
	c2 := novoCliente(t)
	c2.a.projeto = &cadernoProjeto{repo: "https://github.com/x/y", ramo: "main", tarball: gh.URL, pasta: pasta,
		intervalo: time.Hour, cliente: gh.Client()}
	resp = respostaProjeto{}
	if st := c2.chamar("GET", "/api/projeto", nil, &resp); st != http.StatusOK || resp.Aviso == "" || len(resp.Fases) != 11 {
		t.Fatalf("cópia do disco: status %d aviso %q", st, resp.Aviso)
	}
	// Sem nada guardado e sem GitHub: erro claro.
	c3 := novoCliente(t)
	c3.a.projeto = &cadernoProjeto{tarball: gh.URL, pasta: t.TempDir(), intervalo: time.Hour, cliente: gh.Client()}
	var erro map[string]string
	if st := c3.chamar("GET", "/api/projeto", nil, &erro); st != http.StatusBadGateway || erro["erro"] == "" {
		t.Errorf("sem cópia: %d %v", st, erro)
	}
}

func TestPacoteSemRoadmap(t *testing.T) {
	if _, err := lerPacoteProjeto(pacoteDeTeste(t, map[string]string{"README.md": "x"})); err == nil {
		t.Error("aceitou pacote sem ROADMAP.md")
	}
	if _, err := lerPacoteProjeto([]byte("não é gzip")); err == nil {
		t.Error("aceitou lixo")
	}
}
