package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mestreFalso = "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000000,RESOLUTION=640x360\n360p/video.m3u8\n" +
	"#EXT-X-STREAM-INF:BANDWIDTH=8000000,RESOLUTION=1920x1080\n1080p/video.m3u8\n" +
	"#EXT-X-STREAM-INF:BANDWIDTH=4000000,RESOLUTION=1280x720\n720p/video.m3u8\n"

const varianteFalsa = "#EXTM3U\n#EXT-X-KEY:METHOD=AES-128,URI=\"/v/key/k1\"\n#EXT-X-TARGETDURATION:4\n" +
	"#EXTINF:4.0,\nvideo0.dts\n#EXTINF:4.0,\nvideo1.dts\n#EXTINF:2.5,\nvideo2.dts\n#EXT-X-ENDLIST\n"

// servidorDeVideo: um CDN falso que, como o da LINUXtips, só responde com o Referer do player.
func servidorDeVideo(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://player.exemplo.com/" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch {
		case r.URL.Path == "/v/playlist.m3u8":
			w.Write([]byte(mestreFalso))
		case r.URL.Path == "/v/1080p/video.m3u8":
			w.Write([]byte(varianteFalsa))
		case strings.HasSuffix(r.URL.Path, ".dts"):
			w.Header().Set("Content-Length", "1000")
			if r.Method == http.MethodGet {
				w.Write(make([]byte, 1000))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func appDeCursos(t *testing.T) (*app, *httptest.Server) {
	a := novoApp(t)
	srv := servidorDeVideo(t)
	a.clienteCursos = srv.Client()
	a.cursosDir = t.TempDir()
	a.ffmpegBin = filepath.Join("testdata", "ffmpeg-curso-falso")
	a.acordaCursos = make(chan struct{}, 1)
	return a, srv
}

func TestManifestoHLS(t *testing.T) {
	vs := variantesHLS("https://cdn.x/v/playlist.m3u8", mestreFalso)
	if len(vs) != 3 {
		t.Fatalf("variantes: %+v", vs)
	}
	if m := melhorVariante(vs); m.URL != "https://cdn.x/v/1080p/video.m3u8" || m.Altura != 1080 {
		t.Fatalf("melhor: %+v", m)
	}
	pedacos, dur, cifrado := pedacosHLS("https://cdn.x/v/1080p/video.m3u8", varianteFalsa)
	if len(pedacos) != 3 || pedacos[0] != "https://cdn.x/v/1080p/video0.dts" || dur != 10.5 || !cifrado {
		t.Fatalf("pedaços %v, duração %v, cifrado %v", pedacos, dur, cifrado)
	}
	// O Panda põe a URL inteira de cada pedaço; um manifesto sem variantes não tem mestre.
	p2, _, c2 := pedacosHLS("https://b.x/a/720p/video.m3u8", "#EXTM3U\n#EXTINF:4.0,\nhttps://cdn.y/720p/video0.ts\n")
	if p2[0] != "https://cdn.y/720p/video0.ts" || c2 || len(variantesHLS("u", "#EXTM3U\n#EXTINF:4,\na.ts\n")) != 0 {
		t.Fatalf("panda: %v %v", p2, c2)
	}
}

func TestCaminhoDaAula(t *testing.T) {
	got := caminhoDaAula(aulaCurso{Plataforma: "branas", Curso: "Clean Code / Arquitetura", Modulo: "Aula 6 (02/03/2026 às 19:00)", Titulo: "Arquitetura de Microservices"})
	want := filepath.Join("Branas.io", "Clean Code Arquitetura", "Aula 6 (02 03 2026 às 19 00)", "Arquitetura de Microservices.mp4")
	if got != want {
		t.Fatalf("caminho %q, queria %q", got, want)
	}
	if c := caminhoDaAula(aulaCurso{Plataforma: "linuxtips", Curso: "System Design", Titulo: "  "}); c != filepath.Join("LINUXtips", "System Design", "sem nome.mp4") {
		t.Fatalf("sem módulo: %q", c)
	}
	args := strings.Join(argsFFmpeg("https://cdn/v.m3u8", "", "https://player/", "/tmp/a.mp4"), " ")
	for _, parte := range []string{"-headers Referer: https://player/\r\n", "-allowed_segment_extensions ALL", "-extension_picky 0", "-c copy", "-i https://cdn/v.m3u8"} {
		if !strings.Contains(args, parte) {
			t.Fatalf("faltou %q em %q", parte, args)
		}
	}
}

func TestEstimarEPedirAula(t *testing.T) {
	a, srv := appDeCursos(t)
	h := novoServidor(a, nil, "teste")
	corpo := `{"plataforma":"linuxtips","curso":"System Design T4","modulo":"Módulo 1","titulo":"Modelo OSI",
		"pagina":"https://mesa.linuxtips.io/aula/1","manifesto":"` + srv.URL + `/v/playlist.m3u8","referer":"https://player.exemplo.com/"}`

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/estimar", strings.NewReader(corpo)))
	if rec.Code != 200 {
		t.Fatalf("estimar %d %s", rec.Code, rec.Body)
	}
	var est map[string]any
	json.Unmarshal(rec.Body.Bytes(), &est)
	// 3 pedaços medidos de 1000 B; 10,5 s → 11; a maior qualidade.
	if est["qualidade"] != "1080p" || est["duracao"] != float64(11) || est["tamanho_estimado"] != float64(3000) || est["cifrado"] != true {
		t.Fatalf("estimativa %v", est)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas", strings.NewReader(corpo)))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("pedir %d %s", rec.Code, rec.Body)
	}
	var resp struct{ Aula aulaCurso }
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Aula.Estado != "na_fila" || resp.Aula.Qualidade != "1080p" {
		t.Fatalf("aula %+v", resp.Aula)
	}

	// A fila baixa: o arquivo vai para a pasta do curso e a aula fica pronta.
	if !a.baixarProxima(context.Background()) {
		t.Fatal("não baixou")
	}
	au, _ := a.aulaPorID(context.Background(), resp.Aula.ID)
	if au.Estado != "pronta" || au.Tamanho != int64(len("video de mentira")) || au.Arquivo != filepath.Join("LINUXtips", "System Design T4", "Módulo 1", "Modelo OSI.mp4") {
		t.Fatalf("depois de baixar %+v", au)
	}
	if _, err := os.Stat(filepath.Join(a.cursosDir, au.Arquivo)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.cursosDir, au.Arquivo+".baixando")); !os.IsNotExist(err) {
		t.Fatal("ficou o arquivo temporário")
	}

	// O mesmo vídeo de novo (na LINUXtips a página não muda entre as aulas; aqui, outra página e
	// outro token na query): volta a mesma aula, sem baixar outra vez.
	outro := strings.Replace(strings.Replace(corpo, "/aula/1", "/aula/2", 1), "playlist.m3u8", "playlist.m3u8?token=x", 1)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas", strings.NewReader(outro)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"repetida":true`) {
		t.Fatalf("repetida %d %s", rec.Code, rec.Body)
	}

	// O vídeo toca pela Trilha (com Range).
	req := httptest.NewRequest("GET", "/api/cursos/aulas/1/video", nil)
	req.Header.Set("Range", "bytes=0-4")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "video" {
		t.Fatalf("vídeo %d %q", rec.Code, rec.Body)
	}

	// Apagar tira a aula e o arquivo.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/cursos/aulas/1", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("apagar %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(a.cursosDir, au.Arquivo)); !os.IsNotExist(err) {
		t.Fatal("o vídeo ficou")
	}
}

func TestPedirAulaRecusa(t *testing.T) {
	a, srv := appDeCursos(t)
	h := novoServidor(a, nil, "teste")
	casos := map[string]string{
		"plataforma": `{"plataforma":"udemy","curso":"c","titulo":"t","pagina":"https://x/1","manifesto":"` + srv.URL + `/v/playlist.m3u8","referer":"https://player.exemplo.com/"}`,
		"sem título": `{"plataforma":"branas","curso":"c","titulo":" ","pagina":"https://x/1","manifesto":"` + srv.URL + `/v/playlist.m3u8","referer":"https://player.exemplo.com/"}`,
		"não é m3u8": `{"plataforma":"branas","curso":"c","titulo":"t","pagina":"https://x/1","manifesto":"` + srv.URL + `/v/a.mp4","referer":"https://player.exemplo.com/"}`,
		"http":       `{"plataforma":"branas","curso":"c","titulo":"t","pagina":"https://x/1","manifesto":"http://cdn/v.m3u8","referer":"https://player.exemplo.com/"}`,
	}
	for nome, corpo := range casos {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas", strings.NewReader(corpo)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", nome, rec.Code, rec.Body)
		}
	}
	// Sem o Referer certo, o CDN recusa: a estimativa avisa (502), sem criar a aula.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas", strings.NewReader(
		`{"plataforma":"branas","curso":"c","titulo":"t","pagina":"https://x/1","manifesto":"`+srv.URL+`/v/playlist.m3u8","referer":"https://outro.com/"}`)))
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "403") {
		t.Fatalf("referer errado %d %s", rec.Code, rec.Body)
	}
}

func TestDownloadComErroVoltaPelaTentativa(t *testing.T) {
	a, _ := appDeCursos(t)
	t0 := agora()
	a.db.Exec(`INSERT INTO aulas_plataforma (plataforma, curso, titulo, pagina, video, manifesto, referer, variante, duracao, criado_em, atualizado_em)
		VALUES ('branas', 'Curso', 'Aula', 'https://app.branas.io/1', 'https://cdn/falha.m3u8', 'https://cdn/falha.m3u8', 'https://p/', 'https://cdn/falha.m3u8', 10, ?, ?)`, t0, t0)
	a.baixarProxima(context.Background())
	au, _ := a.aulaPorID(context.Background(), 1)
	if au.Estado != "erro" || !strings.Contains(au.Erro, "erro de mentira") {
		t.Fatalf("%+v", au)
	}
	h := novoServidor(a, nil, "teste")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas/1/tentar", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("tentar %d %s", rec.Code, rec.Body)
	}
	if au, _ = a.aulaPorID(context.Background(), 1); au.Estado != "na_fila" {
		t.Fatalf("depois de tentar %+v", au)
	}
	// Recuperar: a que estava baixando quando o app parou volta para a fila.
	a.db.Exec(`UPDATE aulas_plataforma SET estado = 'baixando' WHERE id = 1`)
	a.recuperarCursos(context.Background())
	if au, _ = a.aulaPorID(context.Background(), 1); au.Estado != "na_fila" {
		t.Fatalf("recuperar %+v", au)
	}
}
