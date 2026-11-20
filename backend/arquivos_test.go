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
					"videos":[{"mensagem":2,"nome":"Aula 1","caminho":["Curso A","Módulo 1"],"tamanho":100,"ja":true,"video":7},
						{"mensagem":3,"nome":"Aula 2","caminho":["Curso A","Módulo 1"],"tamanho":200}]}],
				"pedidos":[]}`))
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
