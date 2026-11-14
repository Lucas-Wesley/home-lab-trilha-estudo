package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func novoApp(t *testing.T) *app {
	t.Helper()
	pasta := t.TempDir()
	db, err := abrirBanco(filepath.Join(pasta, "teste.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	c, err := abrirConteudo(filepath.Join(pasta, "conteudo"), "Teste", "teste@exemplo")
	if err != nil {
		t.Fatal(err)
	}
	a := &app{db: db, conteudo: c, fila: novaFila(db), ia: &configIA{
		regras:        filepath.Join("..", "regras"),
		regrasEditada: filepath.Join(pasta, "regras"),
		tarefas:       filepath.Join(pasta, "tarefas"),
		chamar: func(context.Context, pedidoClaude) (respostaClaude, error) {
			return respostaClaude{}, errors.New("sem IA neste teste")
		},
	}}
	a.ligarIA()
	return a
}

func novoTeste(t *testing.T) http.Handler {
	t.Helper()
	web := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>Trilha</title>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
		"sw.js":         {Data: []byte("// sw")},
	}
	return novoServidor(novoApp(t), web, "teste")
}

func pedir(h http.Handler, metodo, caminho string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(metodo, caminho, nil))
	return rec
}

func TestHealth(t *testing.T) {
	rec := pedir(novoTeste(t), "GET", "/api/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var corpo map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&corpo); err != nil {
		t.Fatal(err)
	}
	if corpo["status"] != "ok" || corpo["version"] != "teste" {
		t.Fatalf("corpo inesperado: %v", corpo)
	}
}

func TestRotaDeAPIDesconhecidaDevolveJSON(t *testing.T) {
	rec := pedir(novoTeste(t), "GET", "/api/nada")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"erro"`) {
		t.Fatalf("esperava erro em JSON: %s", rec.Body.String())
	}
}

func TestAppServeIndexEArquivos(t *testing.T) {
	h := novoTeste(t)

	rec := pedir(h, "GET", "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Trilha") {
		t.Fatalf("index: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("index não deve ficar em cache")
	}

	rec = pedir(h, "GET", "/assets/app.js")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}

	// Caminho que não é arquivo volta ao app (as rotas ficam no hash).
	rec = pedir(h, "GET", "/qualquer/coisa")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Trilha") {
		t.Fatalf("fallback: %d %s", rec.Code, rec.Body.String())
	}

	// Uma pasta também volta ao app, nunca a uma listagem dos arquivos.
	rec = pedir(h, "GET", "/assets/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Trilha") || strings.Contains(rec.Body.String(), "app.js") {
		t.Fatalf("pasta: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSemBuildDoFrontAvisa(t *testing.T) {
	h := novoServidor(novoApp(t), fstest.MapFS{".gitkeep": {}}, "teste")

	rec := pedir(h, "GET", "/")
	if rec.Code != http.StatusServiceUnavailable || strings.Contains(rec.Body.String(), ".gitkeep") {
		t.Fatalf("sem build: %d %s", rec.Code, rec.Body.String())
	}
	// A API continua de pé sem o front.
	if rec := pedir(h, "GET", "/api/health"); rec.Code != http.StatusOK {
		t.Fatalf("health sem build: %d", rec.Code)
	}
}
