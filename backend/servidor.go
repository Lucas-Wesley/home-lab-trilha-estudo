package main

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

func novoServidor(db *sql.DB, web fs.FS, versao string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth(db, versao))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		erroJSON(w, http.StatusNotFound, "rota não encontrada")
	})
	mux.Handle("/", arquivosDoApp(web))
	return mux
}

func handleHealth(db *sql.DB, versao string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			erroJSON(w, http.StatusServiceUnavailable, "banco indisponível")
			return
		}
		responderJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": versao})
	}
}

// arquivosDoApp serve o build do Vue. As rotas ficam no hash (#/trilhas), então qualquer caminho
// que não é arquivo (inclusive uma pasta, que o FileServer listaria) volta ao index.html. O
// service worker e o index nunca ficam em cache.
func arquivosDoApp(web fs.FS) http.Handler {
	servidor := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nome := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if !arquivoComum(web, nome) {
			if !arquivoComum(web, "index.html") {
				http.Error(w, "o build do front não está no binário (rode o build-web)", http.StatusServiceUnavailable)
				return
			}
			nome = "index.html"
			r.URL.Path = "/"
		}
		switch {
		case nome == "index.html" || nome == "sw.js" || strings.HasSuffix(nome, ".webmanifest"):
			w.Header().Set("Cache-Control", "no-cache")
		case strings.HasPrefix(nome, "assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		servidor.ServeHTTP(w, r)
	})
}

func arquivoComum(web fs.FS, nome string) bool {
	if nome == "" || nome == "." {
		return false
	}
	info, err := fs.Stat(web, nome)
	return err == nil && info.Mode().IsRegular()
}

func responderJSON(w http.ResponseWriter, status int, dados any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(dados); err != nil {
		log.Print(err)
	}
}

func erroJSON(w http.ResponseWriter, status int, mensagem string) {
	responderJSON(w, status, map[string]string{"erro": mensagem})
}
