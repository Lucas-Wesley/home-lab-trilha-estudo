package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Entrada, fase 7 (T41): "Meus arquivos" — repasse ao Streaming. Nos testes ele é este
// servidor falso: nada de Telegram nem disco de verdade.

// streamingDeCursos: um Streaming falso que devolve um curso com duas aulas (uma baixada) e
// anota o que a Trilha pede.
type pedidoStreaming struct {
	metodo string
	corpo  map[string]any
}

func streamingDeCursos(t *testing.T, pedidos *[]pedidoStreaming) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*pedidos = append(*pedidos, pedidoStreaming{metodo: r.Method + " " + r.URL.Path})
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/meus-videos":
			w.Write([]byte(`{"livre":5000000000,"telegram":{"instalado":true},
				"videos":[{"id":7,"pasta":"Grupo/Curso A/Módulo 1","nome":"Aula 1","tamanho":100,"duracao":60}],
				"salvos":[{"tudo":true,"link":"https://t.me/c/1","pasta":"Grupo",
					"videos":[{"mensagem":2,"nome":"Aula 1","caminho":["Curso A","Módulo 1"],"tamanho":100,"duracao":60,"ja":true,"video":7},
						{"mensagem":3,"nome":"Aula 2","caminho":["Curso A","Módulo 1"],"tamanho":200,"duracao":120},
						{"mensagem":4,"nome":"Aula 3","caminho":["Curso B"],"tamanho":300,"duracao":180}]}],
				"pedidos":[{"id":5,"tudo":true,"link":"https://t.me/c/1","pasta":"Grupo","estado":"baixando","mensagens":[3]}]}`))
		case r.Method == "POST" && r.URL.Path == "/api/meus-videos/telegram":
			var c map[string]any
			json.NewDecoder(r.Body).Decode(&c)
			(*pedidos)[len(*pedidos)-1].corpo = c
			w.Write([]byte(`{"id":9,"estado":"na fila"}`))
		case r.Method == "POST" && r.URL.Path == "/api/meus-videos/apagar-pasta":
			var c map[string]any
			json.NewDecoder(r.Body).Decode(&c)
			(*pedidos)[len(*pedidos)-1].corpo = c
			w.Write([]byte(`{"apagados":2,"liberado":300}`))
		case r.Method == "DELETE" && r.URL.Path == "/api/meus-videos/7":
			w.WriteHeader(204)
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"erro":"não achei"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestArquivosDevolveComoVeio(t *testing.T) {
	c := novoCliente(t)
	pedidos := []pedidoStreaming{}
	c.a.integra.streaming = streamingDeCursos(t, &pedidos).URL
	c.a.integra.streamingLink = "https://streaming.internal"

	var dados map[string]any
	c.ok("GET", "/api/arquivos", nil, &dados)
	if dados["link_streaming"] != "https://streaming.internal" {
		t.Fatalf("sem link_streaming: %v", dados["link_streaming"])
	}
	videos, _ := dados["videos"].([]any)
	if len(videos) != 1 || len(dados["salvos"].([]any)) != 1 {
		t.Fatalf("veio errado: %v", dados)
	}
}

func TestArquivosRepasses(t *testing.T) {
	c := novoCliente(t)
	pedidos := []pedidoStreaming{}
	c.a.integra.streaming = streamingDeCursos(t, &pedidos).URL

	// Baixar: vira o pedido "tudo" do Streaming, com a pasta do grupo e as mensagens.
	var saida map[string]any
	c.ok("POST", "/api/arquivos/baixar",
		map[string]any{"link": "https://t.me/c/1", "grupo": "Grupo", "mensagens": []int{3}}, &saida)
	u := pedidos[len(pedidos)-1]
	if u.metodo != "POST /api/meus-videos/telegram" ||
		u.corpo["tudo"] != true || u.corpo["pasta"] != "Grupo" ||
		len(u.corpo["mensagens"].([]any)) != 1 || u.corpo["mensagens"].([]any)[0] != 3.0 {
		t.Fatalf("o Streaming recebeu: %v %+v", u.metodo, u.corpo)
	}
	if saida["estado"] != "na fila" {
		t.Fatalf("a resposta dele não voltou: %v", saida)
	}

	// Corpo inválido: sem mensagens, ou um link que não é do Telegram → 400 sem chamar ele.
	antes := len(pedidos)
	for _, corpo := range []map[string]any{
		{"link": "https://t.me/c/1", "grupo": "G", "mensagens": []int{}},
		{"link": "https://outro.site/x", "grupo": "G", "mensagens": []int{3}},
	} {
		if s := c.chamar("POST", "/api/arquivos/baixar", corpo, nil); s != 400 {
			t.Fatalf("%v: %d", corpo, s)
		}
	}
	if len(pedidos) != antes {
		t.Fatal("corpo inválido foi parar no Streaming")
	}

	// Apagar a pasta e um vídeo.
	c.ok("POST", "/api/arquivos/apagar-pasta", map[string]any{"pasta": "Grupo/Curso A"}, &saida)
	u = pedidos[len(pedidos)-1]
	if u.metodo != "POST /api/meus-videos/apagar-pasta" || u.corpo["pasta"] != "Grupo/Curso A" {
		t.Fatalf("apagar-pasta: %v %+v", u.metodo, u.corpo)
	}
	if saida["apagados"] != 2.0 {
		t.Fatalf("a resposta dele não voltou: %v", saida)
	}
	if s := c.chamar("DELETE", "/api/arquivos/videos/7", nil, nil); s != 204 {
		t.Fatalf("apagar vídeo: %d", s)
	}
	if pedidos[len(pedidos)-1].metodo != "DELETE /api/meus-videos/7" {
		t.Fatalf("o vídeo foi para: %s", pedidos[len(pedidos)-1].metodo)
	}
}

// Fase 8 (T42): a aula do Telegram pela referência estável "telegram:<link>#<msg>".
func TestAulaArquivos(t *testing.T) {
	c := novoCliente(t)
	pedidos := []pedidoStreaming{}
	c.a.integra.streaming = streamingDeCursos(t, &pedidos).URL
	c.a.integra.streamingLink = "https://streaming.internal"

	estado := func(ref string) (map[string]any, int) {
		var saida map[string]any
		s := c.chamar("GET", "/api/arquivos/aula?ref="+ref, nil, &saida)
		return saida, s
	}

	// Baixada: vem com o id e a url do vídeo no Streaming.
	saida, s := estado("telegram:https://t.me/c/1%232")
	if s != 200 || saida["estado"] != "baixada" || saida["video"] != 7.0 ||
		saida["url"] != "https://streaming.internal/api/video/proprio/7.mp4" {
		t.Fatalf("baixada: %d %v", s, saida)
	}
	// Na fila: há um pedido baixando com a mensagem.
	saida, s = estado("telegram:https://t.me/c/1%233")
	if s != 200 || saida["estado"] != "na_fila" {
		t.Fatalf("na fila: %d %v", s, saida)
	}
	// Só no Telegram: sem vídeo e sem pedido, com nome/tamanho/duração.
	saida, s = estado("telegram:https://t.me/c/1%234")
	if s != 200 || saida["estado"] != "no_telegram" || saida["nome"] != "Aula 3" || saida["tamanho"] != 300.0 {
		t.Fatalf("no telegram: %d %v", s, saida)
	}
	// Não está em nenhum salvo: 404.
	if _, s = estado("telegram:https://t.me/c/1%2399"); s != 404 {
		t.Fatalf("não achada: %d", s)
	}
	// Referência malformada: 400.
	if _, s = estado("telegram:sem-mensagem"); s != 400 {
		t.Fatalf("ref inválida: %d", s)
	}

	// Baixar uma aula: repassa só a mensagem dela, com a pasta do grupo.
	var saida2 map[string]any
	c.ok("POST", "/api/arquivos/aula/baixar", map[string]any{"ref": "telegram:https://t.me/c/1#4"}, &saida2)
	u := pedidos[len(pedidos)-1]
	if u.metodo != "POST /api/meus-videos/telegram" || u.corpo["tudo"] != true ||
		u.corpo["pasta"] != "Grupo" || len(u.corpo["mensagens"].([]any)) != 1 || u.corpo["mensagens"].([]any)[0] != 4.0 {
		t.Fatalf("o Streaming recebeu: %v %+v", u.metodo, u.corpo)
	}
	if s := c.chamar("POST", "/api/arquivos/aula/baixar", map[string]any{"ref": "telegram:https://t.me/c/1#99"}, nil); s != 404 {
		t.Fatalf("baixar não achada: %d", s)
	}
}

// Fase 8 (T42): o pacote que a tela monta (cursoParaTrilha.js) importa como trilha ativa, com os
// materiais streaming apontando para a mensagem do Telegram.
func TestImportarTrilhaDoCurso(t *testing.T) {
	c := novoCliente(t)
	var res resultadoImportacao
	c.ok("POST", "/api/importar", map[string]any{
		"trilha": map[string]any{
			"titulo": "Curso X", "objetivo": "Assistir às aulas de Curso X",
			"descricao": "Curso do Telegram Grupo, pasta Curso X.", "estado": "ativa",
			"modulos": []map[string]any{{
				"titulo": "Curso X",
				"etapas": []map[string]any{
					{"chave": "modulo-1", "titulo": "Módulo 1", "objetivo": "Assistir às aulas de Módulo 1",
						"tipo": "normal", "minutos": 3,
						"materiais": []map[string]any{
							{"tipo": "streaming", "papel": "principal", "titulo": "Aula 1", "referencia": "telegram:https://t.me/c/1#1"},
							{"tipo": "streaming", "papel": "principal", "titulo": "Aula 2", "referencia": "telegram:https://t.me/c/1#2"},
						}},
					{"chave": "modulo-2", "titulo": "Módulo 2", "objetivo": "Assistir às aulas de Módulo 2",
						"tipo": "normal", "minutos": 3,
						"materiais": []map[string]any{
							{"tipo": "streaming", "papel": "principal", "titulo": "Aula 3", "referencia": "telegram:https://t.me/c/1#3"},
						}},
				},
			}},
		},
	}, &res)
	if res.Etapas != 2 {
		t.Fatalf("importação: %+v", res)
	}
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	if tr.Estado != "ativa" {
		t.Fatalf("estado da trilha: %q", tr.Estado)
	}
	ms := tr.Modulos[0].Etapas[0].Materiais
	if len(ms) != 2 || ms[0].Referencia != "telegram:https://t.me/c/1#1" || ms[0].Tipo != "streaming" || ms[0].Papel != "principal" {
		t.Fatalf("materiais: %+v", ms)
	}
}

func TestArquivosErros(t *testing.T) {
	c := novoCliente(t)

	// Streaming fora: 502 com a mensagem.
	if s := c.chamar("GET", "/api/arquivos", nil, nil); s != 502 {
		t.Fatalf("fora: %d", s)
	}

	// Streaming quebrado: a mensagem dele volta com o status que ele mandou.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"erro":"link inválido"}`))
	}))
	defer srv.Close()
	c.a.integra.streaming = srv.URL
	var e map[string]any
	if s := c.chamar("POST", "/api/arquivos/apagar-pasta", map[string]any{"pasta": "G"}, &e); s != 400 || e["erro"] != "link inválido" {
		t.Fatalf("erro dele: %d %v", s, e)
	}
}
