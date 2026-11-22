package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizarEmbed(t *testing.T) {
	casos := []struct{ entrada, baixar, chave string }{
		{"https://www.youtube.com/embed/AAAAAAAAAAA?rel=0&controls=0", "https://www.youtube.com/watch?v=AAAAAAAAAAA", "youtube:AAAAAAAAAAA"},
		{"https://www.youtube-nocookie.com/embed/BBBBBBBBBBB", "https://www.youtube.com/watch?v=BBBBBBBBBBB", "youtube:BBBBBBBBBBB"},
		{"https://youtu.be/CCCCCCCCCCC?t=3", "https://www.youtube.com/watch?v=CCCCCCCCCCC", "youtube:CCCCCCCCCCC"},
		{"https://player.vimeo.com/video/123456?h=abc123&badge=0", "https://player.vimeo.com/video/123456?h=abc123", "vimeo:123456"},
		{"https://vimeo.com/987/ff00", "https://vimeo.com/987/ff00", "vimeo:987"},
	}
	for _, c := range casos {
		b, k, ok := normalizarEmbed(c.entrada)
		if !ok || b != c.baixar || k != c.chave {
			t.Errorf("%s → %q %q %v", c.entrada, b, k, ok)
		}
	}
	for _, ruim := range []string{"https://www.youtube.com/embed/curto", "https://exemplo.com/video/1", "javascript:alert(1)", "https://player.vimeo.com/video/abc"} {
		if _, _, ok := normalizarEmbed(ruim); ok {
			t.Errorf("aceitou %s", ruim)
		}
	}
}

func pedirAulaJSON(t *testing.T, h http.Handler, corpo string) (int, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/cursos/aulas", strings.NewReader(corpo)))
	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec.Code, resp
}

func TestAulaEmbutidaBaixaPeloYtdlp(t *testing.T) {
	a, _ := appDeCursos(t)
	a.ytdlpBin = filepath.Join("testdata", "yt-dlp-curso-falso")
	h := novoServidor(a, nil, "teste")
	corpo := `{"plataforma":"enzoneto","curso":"Métodos de finalização","modulo":"Partida limpa - Grelhados","titulo":"Grelhado perfeito",
		"pagina":"https://aluno.enzoneto.com/area/conteudo/aula/7643874","embed":"https://player.vimeo.com/video/555?h=k1","ordem":1}`
	code, resp := pedirAulaJSON(t, h, corpo)
	if code != http.StatusAccepted {
		t.Fatalf("pedir %d %v", code, resp)
	}
	aula := resp["aula"].(map[string]any)
	if aula["qualidade"] != "1080p" || aula["duracao"] != float64(125) || aula["tamanho_estimado"] != float64(1200) {
		t.Fatalf("estimativa %v", aula)
	}
	if !a.baixarProxima(context.Background()) {
		t.Fatal("não baixou")
	}
	au, _ := a.aulaPorID(context.Background(), int64(aula["id"].(float64)))
	if au.Estado != "pronta" || au.Arquivo != filepath.Join("Chef Enzo Neto", "Métodos de finalização", "Partida limpa - Grelhados", "Grelhado perfeito.mp4") {
		t.Fatalf("depois de baixar %+v", au)
	}
	if b, _ := os.ReadFile(filepath.Join(a.cursosDir, au.Arquivo)); string(b) != "video embutido" {
		t.Fatalf("arquivo %q", b)
	}
	sobras, _ := filepath.Glob(filepath.Join(a.cursosDir, "Chef Enzo Neto", "*", "*", "*.baixando*"))
	if len(sobras) > 0 {
		t.Fatalf("sobrou %v", sobras)
	}
	// O mesmo vídeo de outra página (outro h na query) é a mesma aula.
	code, resp = pedirAulaJSON(t, h, strings.Replace(corpo, "h=k1", "h=k2", 1))
	if code != 200 || resp["repetida"] != true {
		t.Fatalf("repetida %d %v", code, resp)
	}
	// Vimeo privado fora do site: o erro do yt-dlp volta.
	code, resp = pedirAulaJSON(t, h, strings.Replace(corpo, "video/555?h=k1", "video/666?h=falha", 1))
	if code != http.StatusBadGateway || !strings.Contains(resp["erro"].(string), "private") {
		t.Fatalf("falha %d %v", code, resp)
	}
}

func TestCursoViraTrilha(t *testing.T) {
	a, srv := appDeCursos(t)
	h := novoServidor(a, nil, "teste")
	ctx := context.Background()
	video := func(n string) string {
		return `"manifesto":"` + srv.URL + `/v/playlist.m3u8?aula=` + n + `","referer":"https://player.exemplo.com/"`
	}
	base := `"plataforma":"enzoneto","curso":"As praças da cozinha prática","trilha":true`
	code, r1 := pedirAulaJSON(t, h, `{`+base+`,"modulo":"Praça pronta - Bases e temperos","titulo":"Bases prontas","ordem":1,
		"pagina":"https://aluno.enzoneto.com/area/conteudo/aula/1",`+video("1")+`}`)
	if code != http.StatusAccepted || r1["trilha_id"] == nil {
		t.Fatalf("1ª aula %d %v", code, r1)
	}
	trilhaID := int64(r1["trilha_id"].(float64))
	tr, err := a.trilhaCompleta(ctx, trilhaID)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Grupo != "Chef Enzo Neto" || tr.Titulo != "As praças da cozinha prática" || tr.Estado != "ativa" || len(tr.Modulos) != 1 {
		t.Fatalf("trilha %+v", tr)
	}
	e := tr.Modulos[0].Etapas[0]
	if e.Chave != "aula-1" || e.Minutos != 1 || len(e.Materiais) != 1 || e.Materiais[0].Referencia != "/api/cursos/aulas/1/video" || e.Materiais[0].Papel != "principal" {
		t.Fatalf("etapa %+v", e)
	}

	// Uma aula só de texto, num módulo novo, e depois uma aula que vem antes dela no curso.
	code, r3 := pedirAulaJSON(t, h, `{`+base+`,"modulo":"Guia da praça pronta","titulo":"Guia em PDF","ordem":30,
		"pagina":"https://aluno.enzoneto.com/area/conteudo/aula/30","texto":"Leia o guia. Anexos: guia.pdf"}`)
	if code != http.StatusAccepted || r3["aula"].(map[string]any)["estado"] != "texto" || r3["trilha_id"] != r1["trilha_id"] {
		t.Fatalf("aula de texto %d %v", code, r3)
	}
	// A primeira aula do módulo 1 de novo, agora com o texto da página (atualiza e ganha o documento).
	code, _ = pedirAulaJSON(t, h, `{`+base+`,"modulo":"Praça pronta - Bases e temperos","titulo":"Bases prontas","ordem":1,
		"pagina":"https://aluno.enzoneto.com/area/conteudo/aula/1","texto":"Vamos começar o módulo.",`+video("1")+`}`)
	if code != 200 {
		t.Fatalf("repetida %d", code)
	}
	tr, _ = a.trilhaCompleta(ctx, trilhaID)
	if len(tr.Modulos) != 2 || tr.Modulos[1].Titulo != "Guia da praça pronta" {
		t.Fatalf("módulos %+v", tr.Modulos)
	}
	texto := tr.Modulos[1].Etapas[0]
	if texto.Objetivo != "Ler a aula: Guia em PDF" || len(texto.Materiais) != 1 || texto.Materiais[0].Tipo != "documento" || texto.Materiais[0].Papel != "principal" {
		t.Fatalf("etapa de texto %+v", texto)
	}
	primeira := tr.Modulos[0].Etapas[0]
	if len(primeira.Materiais) != 2 || primeira.Materiais[1].Referencia != "aula-1" {
		t.Fatalf("texto que chegou depois %+v", primeira.Materiais)
	}
	doc, err := a.conteudo.ler(caminhoDocumento(tr.Slug, "aula-1"))
	if err != nil || !strings.Contains(doc, "# Bases prontas") || !strings.Contains(doc, "Vamos começar") {
		t.Fatalf("documento %q %v", doc, err)
	}

	// Uma aula que vem entre as duas (ordem 2, mesmo módulo da 1ª) entra logo depois da 1ª.
	if _, err := a.db.Exec(`INSERT INTO aulas_plataforma (plataforma, curso, modulo, titulo, pagina, video, manifesto, referer, estado, ordem, texto, criado_em, atualizado_em)
		VALUES ('enzoneto', 'As praças da cozinha prática', 'Praça pronta - Bases e temperos', 'Tablete de cebola', 'https://x/2', 'pagina:https://x/2', '', '', 'texto', 2, 'Cebola.', 't', 't')`); err != nil {
		t.Fatal(err)
	}
	if _, err := a.sincronizarTrilhaDoCurso(ctx, "enzoneto", "As praças da cozinha prática"); err != nil {
		t.Fatal(err)
	}
	tr, _ = a.trilhaCompleta(ctx, trilhaID)
	if n := len(tr.Modulos[0].Etapas); n != 2 || tr.Modulos[0].Etapas[1].Titulo != "Tablete de cebola" {
		t.Fatalf("aula do meio %+v", tr.Modulos[0].Etapas)
	}

	// A etapa apagada pelo Lucas não volta.
	if _, err := a.db.Exec(`DELETE FROM etapas WHERE trilha_id = ? AND chave = 'aula-30'`, trilhaID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.sincronizarTrilhaDoCurso(ctx, "enzoneto", "As praças da cozinha prática"); err != nil {
		t.Fatal(err)
	}
	tr, _ = a.trilhaCompleta(ctx, trilhaID)
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			if e.Chave == "aula-30" {
				t.Fatal("a etapa apagada voltou")
			}
		}
	}

	// A lista dos cursos diz qual trilha é de qual curso; a aula sozinha traz o texto.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/cursos/aulas", nil))
	if !strings.Contains(rec.Body.String(), `"trilhas":[{"plataforma":"enzoneto","curso":"As praças da cozinha prática","trilha_id":`) ||
		strings.Contains(rec.Body.String(), "Vamos começar") {
		t.Fatalf("lista %s", rec.Body)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/cursos/aulas/1", nil))
	if !strings.Contains(rec.Body.String(), "Vamos começar") {
		t.Fatalf("aula %s", rec.Body)
	}
}

func TestAulaSemVideoNemTexto(t *testing.T) {
	a, _ := appDeCursos(t)
	h := novoServidor(a, nil, "teste")
	code, resp := pedirAulaJSON(t, h, `{"plataforma":"enzoneto","curso":"C","titulo":"A","pagina":"https://aluno.enzoneto.com/area/conteudo/aula/9"}`)
	if code != 400 || !strings.Contains(resp["erro"].(string), "nem texto") {
		t.Fatalf("%d %v", code, resp)
	}
}

func anexar(t *testing.T, h http.Handler, aula int64, nome, conteudo string) *httptest.ResponseRecorder {
	t.Helper()
	var corpo bytes.Buffer
	f := multipart.NewWriter(&corpo)
	f.WriteField("nome", nome)
	f.WriteField("url", "https://aluno.enzoneto.com/area/arquivo/anexo?n=x")
	p, _ := f.CreateFormFile("file", nome)
	p.Write([]byte(conteudo))
	f.Close()
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/cursos/aulas/%d/anexos", aula), &corpo)
	req.Header.Set("Content-Type", f.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAnexosDaAula(t *testing.T) {
	a, _ := appDeCursos(t)
	h := novoServidor(a, nil, "teste")
	ctx := context.Background()
	code, resp := pedirAulaJSON(t, h, `{"plataforma":"enzoneto","curso":"Congelamento inteligente 2026","modulo":"Materiais em PDF",
		"titulo":"PDF para download e impressão","pagina":"https://aluno.enzoneto.com/area/conteudo/aula/9521338","texto":"## Anexos\n\n- Tabela.pdf","ordem":21,"trilha":true}`)
	if code != http.StatusAccepted {
		t.Fatalf("aula %d %v", code, resp)
	}
	aula := int64(resp["aula"].(map[string]any)["id"].(float64))
	trilhaID := int64(resp["trilha_id"].(float64))

	if rec := anexar(t, h, aula, "Tabela.pdf", "<html>login</html>"); rec.Code != 400 {
		t.Fatalf("aceitou um não-PDF: %d %s", rec.Code, rec.Body)
	}
	rec := anexar(t, h, aula, "Tabela de Durabilidade e Etiquetas-print.pdf", "%PDF-1.4 tabela")
	if rec.Code != http.StatusCreated {
		t.Fatalf("anexar %d %s", rec.Code, rec.Body)
	}
	var r struct{ Anexo anexoAula }
	json.Unmarshal(rec.Body.Bytes(), &r)
	// O mesmo nome de novo troca o arquivo, sem duplicar.
	if rec := anexar(t, h, aula, "Tabela de Durabilidade e Etiquetas-print.pdf", "%PDF-1.4 tabela nova"); rec.Code != http.StatusOK {
		t.Fatalf("de novo %d %s", rec.Code, rec.Body)
	}
	anexos, _ := a.anexosDaAula(ctx, aula)
	if len(anexos) != 1 || anexos[0].Arquivo != filepath.Join("Chef Enzo Neto", "Congelamento inteligente 2026", "Materiais em PDF", "PDF para download e impressão - Tabela de Durabilidade e Etiquetas-print.pdf") {
		t.Fatalf("anexos %+v", anexos)
	}

	// O PDF abre pela Trilha.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/cursos/anexos/%d", r.Anexo.ID), nil))
	if rec.Code != 200 || rec.Body.String() != "%PDF-1.4 tabela nova" || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("abrir %d %q %v", rec.Code, rec.Body, rec.Header())
	}
	// A etapa da aula (sem vídeo) ganha o PDF como material principal.
	tr, _ := a.trilhaCompleta(ctx, trilhaID)
	ms := tr.Modulos[0].Etapas[0].Materiais
	if len(ms) != 2 || ms[1].Tipo != "arquivo" || ms[1].Papel != "principal" || ms[1].Referencia != fmt.Sprintf("/api/cursos/anexos/%d", r.Anexo.ID) {
		t.Fatalf("materiais %+v", ms)
	}
	// A lista e a aula mostram os anexos; apagar a aula apaga o PDF.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/cursos/aulas", nil))
	if !strings.Contains(rec.Body.String(), `"anexos":[{"id":`) {
		t.Fatalf("lista %s", rec.Body)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("DELETE", fmt.Sprintf("/api/cursos/aulas/%d", aula), nil))
	if _, err := os.Stat(filepath.Join(a.cursosDir, anexos[0].Arquivo)); !os.IsNotExist(err) {
		t.Fatal("o PDF ficou no disco")
	}
}

func TestSemODiscoAFilaEspera(t *testing.T) {
	a, srv := appDeCursos(t)
	a.cursosMarca = ".disco-dos-cursos"
	h := novoServidor(a, nil, "teste")
	ctx := context.Background()
	code, resp := pedirAulaJSON(t, h, `{"plataforma":"linuxtips","curso":"C","titulo":"A","pagina":"https://mesa.linuxtips.io/a",
		"manifesto":"`+srv.URL+`/v/playlist.m3u8","referer":"https://player.exemplo.com/"}`)
	if code != http.StatusAccepted {
		t.Fatalf("pedir %d %v", code, resp)
	}
	id := int64(resp["aula"].(map[string]any)["id"].(float64))
	// O SSD fora: nada sai da fila, a lista avisa e o PDF não é aceito.
	if a.baixarProxima(ctx) {
		t.Fatal("baixou sem o disco")
	}
	if au, _ := a.aulaPorID(ctx, id); au.Estado != "na_fila" {
		t.Fatalf("estado %s", au.Estado)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/cursos/aulas", nil))
	if !strings.Contains(rec.Body.String(), "não está montado") {
		t.Fatalf("lista sem aviso %s", rec.Body)
	}
	if rec := anexar(t, h, id, "x.pdf", "%PDF-1.4"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("anexo sem disco %d", rec.Code)
	}
	// O SSD de volta (a marca na pasta): a fila anda.
	os.WriteFile(filepath.Join(a.cursosDir, ".disco-dos-cursos"), nil, 0o644)
	if !a.baixarProxima(ctx) {
		t.Fatal("não baixou com o disco")
	}
	if au, _ := a.aulaPorID(ctx, id); au.Estado != "pronta" {
		t.Fatalf("estado %s %s", au.Estado, au.Erro)
	}
}
