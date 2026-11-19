package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clienteEntrada: o app de teste com o yt-dlp falso (YTDLP_BIN apontando para testdata).
func clienteEntrada(t *testing.T) *cliente {
	t.Helper()
	c := novoCliente(t)
	c.a.ytdlpBin = filepath.Join("testdata", "yt-dlp-falso")
	return c
}

func TestVideoDoLink(t *testing.T) {
	validos := []string{
		"https://www.youtube.com/watch?v=AAAAAAAAAAA",
		"https://youtu.be/AAAAAAAAAAA",
		"https://youtu.be/AAAAAAAAAAA?t=30",
		"https://www.youtube.com/shorts/AAAAAAAAAAA",
		"https://www.youtube.com/live/AAAAAAAAAAA",
		"https://m.youtube.com/watch?v=AAAAAAAAAAA",
		"https://www.youtube.com/watch?v=AAAAAAAAAAA&t=1h2m3s",
		"https://www.youtube.com/embed/AAAAAAAAAAA",
	}
	for _, link := range validos {
		id, err := videoDoLink(link)
		if err != nil || id != "AAAAAAAAAAA" {
			t.Errorf("%q: %q %v", link, id, err)
		}
	}
	invalidos := []string{
		"https://vimeo.com/12345",
		"https://www.youtube.com/playlist?list=PLteste",
		"https://www.youtube.com/watch?v=curto",
	}
	for _, link := range invalidos {
		if id, err := videoDoLink(link); err == nil {
			t.Errorf("%q devia falhar, deu %q", link, id)
		}
	}
}

func TestEhPlaylist(t *testing.T) {
	casos := []struct {
		link string
		eh   bool
	}{
		{"https://www.youtube.com/playlist?list=PLteste", true},
		{"https://www.youtube.com/watch?list=PLteste", true},
		{"https://www.youtube.com/watch?v=AAAAAAAAAAA&list=PLteste", false}, // um vídeo que estava numa playlist
		{"https://www.youtube.com/watch?v=AAAAAAAAAAA", false},
		{"https://youtu.be/AAAAAAAAAAA?list=PLteste", false},
	}
	for _, x := range casos {
		if eh := ehPlaylist(x.link); eh != x.eh {
			t.Errorf("%q: %v", x.link, eh)
		}
	}
}

func TestLegendaParaCru(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "legenda.json3"))
	if err != nil {
		t.Fatal(err)
	}
	info := infoVideo{
		Titulo: "Vídeo de teste", Canal: "Canal de Teste",
		URL: "https://www.youtube.com/watch?v=AAAAAAAAAAA", DuracaoTexto: "1:00",
		Capitulos: []capituloInfo{
			{Inicio: 0, Fim: 30, Titulo: "Introdução"},
			{Inicio: 30, Fim: 60, Titulo: "O assunto"},
		},
	}
	cru, err := legendaParaCru(b, info)
	if err != nil {
		t.Fatal(err)
	}
	for _, trecho := range []string{
		"# Vídeo de teste", "- Canal: Canal de Teste", "- Duração: 1:00",
		"- Capítulos:\n  - 00:00 Introdução\n  - 00:30 O assunto\n",
		"[00:00] Olá, hoje vamos falar de redes. Primeiro assunto.",
		"[00:10] Segundo assunto.",
		"[00:21] Terceiro assunto.",
		"[1:02:05] Fim.",
	} {
		if !strings.Contains(cru, trecho) {
			t.Errorf("a transcrição devia ter %q:\n%s", trecho, cru)
		}
	}
}

func TestEntradaVideo(t *testing.T) {
	c := clienteEntrada(t)
	var fontes []fonte
	if s := c.chamar("POST", "/api/entrada", map[string]string{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA"}, &fontes); s != http.StatusAccepted {
		t.Fatalf("status %d: %v", s, fontes)
	}
	if len(fontes) != 1 || fontes[0].Tipo != "youtube" || fontes[0].VideoID != "AAAAAAAAAAA" || fontes[0].Estado != "na_fila" {
		t.Fatalf("fonte criada errada: %+v", fontes)
	}
	c.a.esvaziarEntrada(context.Background())

	var f fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f)
	if f.Estado != "pronta" || f.Titulo != "Vídeo AAAAAAAAAAA" || f.Canal != "Canal de Teste" ||
		f.Duracao != 60 || f.Idioma != "pt-BR" {
		t.Fatalf("fonte lida errada: %+v", f)
	}
	if len(f.Capitulos) != 2 || f.Capitulos[0].Titulo != "Introdução" || f.Capitulos[1].Inicio != 30 {
		t.Fatalf("capítulos: %+v", f.Capitulos)
	}
	if !strings.Contains(f.Transcricao, "- 00:00 Introdução") ||
		!strings.Contains(f.Transcricao, "[00:00] Olá, hoje vamos falar de redes.") ||
		!strings.Contains(f.Transcricao, "[00:10] Segundo assunto.") {
		t.Fatalf("transcrição:\n%s", f.Transcricao)
	}

	// O mesmo vídeo de novo não cria outra fonte (o UNIQUE), mesmo com outro formato de link.
	var deNovo []fonte
	c.chamar("POST", "/api/entrada", map[string]string{"url": "https://youtu.be/AAAAAAAAAAA"}, &deNovo)
	if len(deNovo) != 1 || deNovo[0].ID != fontes[0].ID {
		t.Fatalf("duplicou: %+v", deNovo)
	}
	var lista []fonte
	c.ok("GET", "/api/entrada", nil, &lista)
	if len(lista) != 1 {
		t.Fatalf("lista: %+v", lista)
	}
}

func TestEntradaPlaylist(t *testing.T) {
	c := clienteEntrada(t)
	var fontes []fonte
	if s := c.chamar("POST", "/api/entrada", map[string]string{"url": "https://www.youtube.com/playlist?list=PLteste"}, &fontes); s != http.StatusAccepted {
		t.Fatalf("status %d", s)
	}
	if len(fontes) != 3 {
		t.Fatalf("3 vídeos na playlist, veio %d", len(fontes))
	}
	for i, id := range []string{"AAAAAAAAAAA", "BBBBBBBBBBB", "CCCCCCCCCCC"} {
		if fontes[i].VideoID != id || fontes[i].Estado != "na_fila" || fontes[i].Playlist != "Curso de teste" {
			t.Fatalf("fonte %d errada: %+v", i, fontes[i])
		}
	}
}

func TestEntradaLinkInvalido(t *testing.T) {
	c := clienteEntrada(t)
	var corpo map[string]string
	if s := c.chamar("POST", "/api/entrada", map[string]string{"url": "https://vimeo.com/12345"}, &corpo); s != http.StatusBadRequest {
		t.Fatalf("status %d", s)
	}
	if !strings.Contains(corpo["erro"], "YouTube") {
		t.Fatalf("mensagem: %v", corpo)
	}
}

func TestEntradaSemLegenda(t *testing.T) {
	c := clienteEntrada(t)
	var fontes []fonte
	c.chamar("POST", "/api/entrada", map[string]string{"url": "https://www.youtube.com/watch?v=SSSSSSSSSSS"}, &fontes)
	c.a.esvaziarEntrada(context.Background())

	var f fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f)
	if f.Estado != "erro" || !strings.Contains(f.Erro, "não tem legenda") || !strings.Contains(f.Erro, "Whisper") {
		t.Fatalf("sem WHISPER_URL devia dar erro: %+v", f)
	}

	// Com o Whisper (falso) configurado, "Tentar de novo" transcreve o áudio.
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" {
			w.WriteHeader(404)
			return
		}
		if err := r.ParseMultipartForm(4 << 20); err != nil || r.FormValue("response_format") != "verbose_json" {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"language":"pt","segments":[{"start":0.0,"text":"Fala um."},{"start":12.5,"text":"Fala dois."}]}`))
	}))
	defer whisper.Close()
	c.a.whisper = whisper.URL

	var tentou fonte
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/tentar", fontes[0].ID), nil, &tentou)
	if tentou.Estado != "na_fila" {
		t.Fatalf("tentar: %+v", tentou)
	}
	c.a.esvaziarEntrada(context.Background())

	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f)
	if f.Estado != "pronta" {
		t.Fatalf("depois do Whisper: %+v", f)
	}
	if !strings.Contains(f.Transcricao, "Transcrição: Whisper") ||
		!strings.Contains(f.Transcricao, "[00:00] Fala um.") ||
		!strings.Contains(f.Transcricao, "[00:12] Fala dois.") {
		t.Fatalf("transcrição do Whisper:\n%s", f.Transcricao)
	}
}

func TestEntradaExcluir(t *testing.T) {
	c := clienteEntrada(t)
	var fontes []fonte
	c.chamar("POST", "/api/entrada", map[string]string{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA"}, &fontes)
	c.a.esvaziarEntrada(context.Background())
	if s := c.chamar("DELETE", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, nil); s != http.StatusNoContent {
		t.Fatalf("status %d", s)
	}
	if _, err := c.a.conteudo.ler(pastaDaFonte(fontes[0].ID) + "/transcricao.md"); err == nil {
		t.Fatal("a transcrição devia ter ido embora com a fonte")
	}
	if s := c.chamar("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, nil); s != http.StatusNotFound {
		t.Fatalf("a fonte devia ter sido apagada: %d", s)
	}
}

// GET /compartilhar é a share_target do Android (T36): o fallback do app devolve o index.html e o
// front acha o link no texto e manda para /api/entrada.
func TestCompartilharDevolveOApp(t *testing.T) {
	rec := pedir(novoTeste(t), "GET", "/compartilhar?text=V%C3%ADdeo%20https://youtu.be/AAAAAAAAAAA")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Trilha") {
		t.Fatalf("compartilhar: %d %s", rec.Code, rec.Body.String())
	}
}
