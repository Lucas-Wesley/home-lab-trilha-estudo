package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Entrada, fase 9 (T43): material "curso" — pastas do Telegram encaixadas na trilha pelos
// títulos das aulas. Nos testes, o Streaming é o servidor falso e a IA, a falsa de sempre.

func TestLerEscreverCursos(t *testing.T) {
	ref := "telegram:https://t.me/c/1::Grupo/Curso A | telegram:https://t.me/c/2::MBA/Curso DDD"
	lista := lerCursos(ref)
	if len(lista) != 2 || lista[0].Link != "https://t.me/c/1" || lista[0].Caminho != "Grupo/Curso A" ||
		lista[1].Caminho != "MBA/Curso DDD" {
		t.Fatalf("ler: %+v", lista)
	}
	if escreverCursos(lista) != ref {
		t.Fatalf("escrever: %q", escreverCursos(lista))
	}
	// Uma pasta só, sem o separador.
	if l := lerCursos("telegram:https://t.me/c/9::G/C"); len(l) != 1 {
		t.Fatalf("uma pasta: %+v", l)
	}
	// Lixo não é curso: sem telegram:, sem link t.me, sem caminho.
	for _, ref := range []string{"", "https://exemplo.org", "telegram:https://x.com/1::G", "telegram:https://t.me/1::", "telegram::G"} {
		if l := lerCursos(ref); len(l) != 0 {
			t.Fatalf("%q: %+v", ref, l)
		}
	}
}

func TestLerAulas(t *testing.T) {
	casos := []struct {
		loc  string
		want []int
	}{
		{"A12", []int{12}},
		{"A12–A15", []int{12, 13, 14, 15}},
		{"A3-A4", []int{3, 4}},
		{"A3, A7", []int{3, 7}},
		{"A2 A2 A5", []int{2, 5}},              // sem repetir
		{"aulas A1, A3 e A5.", []int{1, 3, 5}}, // pontuação e palavras em volta
		{"cap. 4", nil},                        // sem o prefixo A não é aula
		{"p. 45–60", nil},
		{"o material todo", nil},
		{"", nil},
	}
	for _, c := range casos {
		got := lerAulas(c.loc)
		if len(got) != len(c.want) {
			t.Fatalf("%q: %v", c.loc, got)
		}
		for i, n := range got {
			if n != c.want[i] {
				t.Fatalf("%q: %v", c.loc, got)
			}
		}
	}
}

// A mesma árvore inventada do frontend/src/pastas.test.js — o servidor monta igual.
func TestArvoreDosArquivos(t *testing.T) {
	var dados meusVideosStreaming
	json.Unmarshal([]byte(`{
		"videos":[
			{"id":7,"pasta":"Grupo/Curso A/Módulo 1","nome":"Aula 1","tamanho":100,"duracao":60},
			{"id":9,"pasta":"Viagens","nome":"Chile","tamanho":50,"duracao":30}],
		"salvos":[
			{"tudo":true,"link":"https://t.me/c/1","pasta":"Grupo","videos":[
				{"mensagem":2,"nome":"Aula 1","caminho":["Curso A","Módulo 1"],"tamanho":100,"duracao":60,"video":7},
				{"mensagem":3,"nome":"Aula 2","caminho":["Curso A","Módulo 1"],"tamanho":200,"duracao":120},
				{"mensagem":4,"nome":"Outra","caminho":["Curso B"],"tamanho":300,"duracao":180}]},
			{"tudo":false,"link":"https://t.me/c/9","pasta":"Ignorada","videos":[
				{"mensagem":1,"nome":"Parcial","caminho":["X"],"tamanho":1,"duracao":1}]}]}`), &dados)

	raiz := arvoreDosArquivos(&dados)
	if raiz.Total != 4 || len(raiz.Pastas) != 2 || raiz.Pastas[0].Nome != "Grupo" || raiz.Pastas[1].Nome != "Viagens" {
		t.Fatalf("raiz: %d %v", raiz.Total, raiz.Pastas)
	}
	grupo := raiz.Pastas[0]
	if grupo.Total != 3 || len(grupo.Pastas) != 2 || grupo.Pastas[0].Nome != "Curso A" || grupo.Pastas[1].Nome != "Curso B" {
		t.Fatalf("grupo: %+v", grupo)
	}
	mod := grupo.Pastas[0].Pastas[0]
	if mod.Nome != "Módulo 1" || len(mod.Aulas) != 2 ||
		mod.Aulas[0].Nome != "Aula 1" || mod.Aulas[0].Video == nil || mod.Aulas[0].Video.ID != 7 ||
		mod.Aulas[1].Nome != "Aula 2" || mod.Aulas[1].Video != nil || mod.Aulas[1].Chave != "https://t.me/c/1#3" {
		t.Fatalf("módulo: %+v", mod)
	}
	if a := grupo.Pastas[1].Aulas[0]; a.Nome != "Outra" || a.Chave != "https://t.me/c/1#4" {
		t.Fatalf("curso B: %+v", a)
	}
	// Só no disco: pasta própria, depois dos salvos; o vídeo 7 não repete.
	viagens := raiz.Pastas[1]
	if viagens.Total != 1 || viagens.Aulas[0].Nome != "Chile" || viagens.Aulas[0].Chave != "video:9" {
		t.Fatalf("disco: %+v", viagens)
	}
	// acharPasta equivalente.
	if raiz.achar("Grupo/Curso A/Módulo 1") != mod || raiz.achar("Grupo/Sumiu") != nil {
		t.Fatal("achar")
	}
}

// streamingDosCursos: dois grupos do Telegram com um curso cada — o dobro do falso da fase 7.
func streamingDosCursos(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/meus-videos" {
			w.Write([]byte(`{"livre":5000000000,
				"videos":[{"id":7,"pasta":"Grupo/Curso A","nome":"Aula 1","tamanho":100,"duracao":60}],
				"salvos":[
					{"tudo":true,"link":"https://t.me/c/1","pasta":"Grupo","videos":[
						{"mensagem":2,"nome":"Aula 1","caminho":["Curso A"],"tamanho":100,"duracao":60,"video":7},
						{"mensagem":3,"nome":"Aula 2","caminho":["Curso A"],"tamanho":200,"duracao":120}]},
					{"tudo":true,"link":"https://t.me/c/2","pasta":"MBA","videos":[
						{"mensagem":5,"nome":"DDD tático","caminho":["Curso DDD"],"tamanho":300,"duracao":600},
						{"mensagem":6,"nome":"Live","caminho":["Curso DDD"],"tamanho":400,"duracao":3600}]}]}`))
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEncaixeCurso(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"encaixes":[
{"etapa_chave":"nat","localizador":"A1–A2","papel":"principal","nota":"Base de NAT.","motivo":"x"},
{"etapa_chave":"nat","localizador":"A9","papel":"apoio","nota":"y","motivo":"y"}],
"novos":[
{"titulo":"DDD tático","objetivo":"Aplicar aggregates e repos.","bloom":"aplicar","minutos":45,"modulo":"Internet","depois_de":"","localizador":"A3","nota":"n","motivo":"m"},
{"titulo":"Sem número","objetivo":"x","bloom":"entender","minutos":30,"modulo":"Internet","depois_de":"","localizador":"cap. 4","nota":"","motivo":""}],
"notas":"A maioria dos títulos diz o assunto.\nPrecisam ser ouvidas: A4","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	c.a.integra.streaming = streamingDosCursos(t).URL
	c.a.integra.streamingLink = "https://streaming.internal"
	trilhaComEtapa(c)

	referencia := "telegram:https://t.me/c/1::Grupo/Curso A | telegram:https://t.me/c/2::MBA/Curso DDD"
	material := map[string]any{"tipo": "curso", "titulo": "Curso A + Curso DDD", "referencia": referencia}
	var p proposta
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": material}, &p)
	c.a.fila.esvaziar(context.Background())

	// O sumário no pedido: títulos por nível, aulas numeradas A1…A4 seguidas entre os cursos.
	sumario := f.arquivos[0]["material/01-curso.md"]
	for _, trecho := range []string{"## Curso A", "- A1 · Aula 1 (1 min)", "- A2 · Aula 2 (2 min)",
		"## Curso DDD", "- A3 · DDD tático (10 min)", "- A4 · Live (60 min)"} {
		if !strings.Contains(sumario, trecho) {
			t.Fatalf("sumário sem %q:\n%s", trecho, sumario)
		}
	}
	if !strings.Contains(f.arquivos[0]["pedido.md"], "material/01-curso.md") {
		t.Fatal("pedido não aponta para o sumário")
	}

	// O mapa ficou na proposta: número → referência do Telegram e → nome da aula.
	var ce conteudoEncaixe
	var lista []struct {
		Proposta proposta `json:"proposta"`
	}
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &lista)
	json.Unmarshal(lista[0].Proposta.Conteudo, &ce)
	if len(ce.Aulas) != 4 || ce.Aulas["A1"] != "telegram:https://t.me/c/1#2" || ce.Aulas["A3"] != "telegram:https://t.me/c/2#5" ||
		ce.Nomes["A3"] != "DDD tático" {
		t.Fatalf("mapa: %+v", ce.Aulas)
	}
	// limparEncaixe tirou o tópico "Sem número" (localizador sem A<n>); A9 segue para aprovação.
	if len(ce.Novos) != 1 || len(ce.Encaixes) != 2 || !strings.Contains(ce.Notas, "Sem número") || !strings.Contains(ce.Notas, "Precisam ser ouvidas") {
		t.Fatalf("conteúdo: %+v", ce)
	}

	// Aprovar os dois encaixes e o tópico novo.
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].Proposta.ID),
		map[string]any{"encaixes": []int{0, 1}, "novos": []int{0}}, nil)
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	nat := tr.Modulos[0].Etapas[1]
	if nat.Chave != "nat" {
		t.Fatalf("etapas: %+v", tr.Modulos[0].Etapas)
	}
	// Duas partes na ordem dos números, como materiais streaming do Telegram; A9 não está no
	// mapa → ignorada e registrada nas notas da proposta.
	if len(nat.Materiais) != 3 {
		t.Fatalf("materiais da etapa: %+v", nat.Materiais)
	}
	m1, m2 := nat.Materiais[1], nat.Materiais[2]
	if m1.Tipo != "streaming" || m1.Referencia != "telegram:https://t.me/c/1#2" || m1.Titulo != "Aula 1" || m1.Papel != "principal" ||
		m2.Referencia != "telegram:https://t.me/c/1#3" || m2.Titulo != "Aula 2" {
		t.Fatalf("partes: %+v %+v", m1, m2)
	}
	// O tópico novo nasceu com o material da aula A3.
	nova := tr.Modulos[0].Etapas[2]
	if nova.Titulo != "DDD tático" || len(nova.Materiais) != 1 || nova.Materiais[0].Referencia != "telegram:https://t.me/c/2#5" ||
		nova.Materiais[0].Tipo != "streaming" || nova.Materiais[0].Papel != "principal" {
		t.Fatalf("novo tópico: %+v", nova)
	}
	// A9 foi ignorada e registrada nas notas.
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &lista)
	json.Unmarshal(lista[0].Proposta.Conteudo, &ce)
	if !strings.Contains(ce.Notas, "A9") {
		t.Fatalf("notas: %q", ce.Notas)
	}
}

// O mesmo encaixe numa trilha vazia (a criada por "Criar trilha a partir destes cursos"):
// só tópicos e módulos novos.
func TestEncaixeCursoTrilhaVazia(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"encaixes":[],"novos":[
{"titulo":"Fundamentos","objetivo":"Entender a base.","bloom":"entender","minutos":30,"modulo":"Começo","depois_de":"","localizador":"A1","nota":"","motivo":""},
{"titulo":"Prática","objetivo":"Aplicar.","bloom":"aplicar","minutos":45,"modulo":"Começo","depois_de":"","localizador":"A2","nota":"","motivo":""}],
"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	c.a.integra.streaming = streamingDosCursos(t).URL
	// Trilha vazia como a tela cria: só título, rascunho.
	var tr trilha
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Curso A + Curso DDD"}, &tr)

	material := map[string]any{"tipo": "curso", "titulo": "Curso A + Curso DDD",
		"referencia": "telegram:https://t.me/c/1::Grupo/Curso A | telegram:https://t.me/c/2::MBA/Curso DDD"}
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/encaixar", tr.ID), map[string]any{"material": material}, nil)
	c.a.fila.esvaziar(context.Background())

	var lista []struct {
		Proposta proposta `json:"proposta"`
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/encaixes", tr.ID), nil, &lista)
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].Proposta.ID),
		map[string]any{"novos": []int{0, 1}}, nil)
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d", tr.ID), nil, &tr)
	if len(tr.Modulos) != 1 || tr.Modulos[0].Titulo != "Começo" || len(tr.Modulos[0].Etapas) != 2 {
		t.Fatalf("trilha: %+v", tr.Modulos)
	}
	if tr.Modulos[0].Etapas[0].Materiais[0].Referencia != "telegram:https://t.me/c/1#2" {
		t.Fatalf("material: %+v", tr.Modulos[0].Etapas[0].Materiais)
	}
}
