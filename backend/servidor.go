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

// app junta o que os handlers usam.
type app struct {
	db       *sql.DB
	conteudo *conteudo
	ia       *configIA
	fila     *fila
	integra  integracoes
}

// ligarIA registra os tipos de tarefa na fila (fase 3).
func (a *app) ligarIA() {
	if a.ia.chamar == nil {
		a.ia.chamar = a.ia.chamarClaude
	}
	a.fila.executores["licao"] = a.gerarLicao
	for _, tipo := range tiposCriacao {
		a.fila.executores[tipo] = a.gerarCriacao
	}
	for _, tipo := range tiposEstudo {
		a.fila.executores[tipo] = a.gerarEstudo
	}
	a.fila.executores[tipoCorrecao] = a.corrigirSimulado
	a.fila.aoTerminar = a.aoTerminarTarefa
}

func novoServidor(a *app, web fs.FS, versao string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth(a.db, versao))

	mux.HandleFunc("GET /api/hoje", a.handleHoje)

	mux.HandleFunc("GET /api/trilhas", a.handleListarTrilhas)
	mux.HandleFunc("POST /api/trilhas", a.handleCriarTrilha)
	mux.HandleFunc("GET /api/trilhas/{id}", a.handleTrilha)
	mux.HandleFunc("PUT /api/trilhas/{id}", a.handleEditarTrilha)
	mux.HandleFunc("DELETE /api/trilhas/{id}", a.handleExcluirTrilha)
	mux.HandleFunc("POST /api/trilhas/{id}/estado", a.handleEstadoTrilha)
	mux.HandleFunc("PUT /api/trilhas/{id}/ordem", a.handleOrdem)
	mux.HandleFunc("PUT /api/trilhas/{id}/meta", a.handleMeta)
	mux.HandleFunc("POST /api/trilhas/{id}/recalcular", a.handleRecalcular)
	mux.HandleFunc("GET /api/trilhas/{id}/cronograma", a.handleCronograma)
	mux.HandleFunc("POST /api/trilhas/{id}/modulos", a.handleCriarModulo)

	mux.HandleFunc("PUT /api/modulos/{id}", a.handleEditarModulo)
	mux.HandleFunc("DELETE /api/modulos/{id}", a.handleExcluirModulo)
	mux.HandleFunc("POST /api/modulos/{id}/etapas", a.handleCriarEtapa)

	mux.HandleFunc("GET /api/etapas/{id}", a.handleEtapa)
	mux.HandleFunc("PUT /api/etapas/{id}", a.handleEditarEtapa)
	mux.HandleFunc("DELETE /api/etapas/{id}", a.handleExcluirEtapa)
	mux.HandleFunc("POST /api/etapas/{id}/status", a.handleStatusEtapa)

	mux.HandleFunc("GET /api/trilhas/{id}/documentos", a.handleDocumentos)
	mux.HandleFunc("GET /api/trilhas/{id}/documentos/{doc}", a.handleDocumento)
	mux.HandleFunc("PUT /api/trilhas/{id}/documentos/{doc}", a.handleSalvarDocumento)
	mux.HandleFunc("DELETE /api/trilhas/{id}/documentos/{doc}", a.handleExcluirDocumento)

	mux.HandleFunc("POST /api/importar", a.handleImportar)

	// Fase 3: IA, propostas e comentários (DECISOES.md §5).
	mux.HandleFunc("POST /api/etapas/{id}/licao", a.handleGerarLicao)
	mux.HandleFunc("GET /api/etapas/{id}/propostas", a.handlePropostasDaEtapa)
	mux.HandleFunc("POST /api/propostas/{id}/comentarios", a.handleComentar)
	mux.HandleFunc("POST /api/propostas/{id}/ajustar", a.handleAjustar)
	mux.HandleFunc("POST /api/propostas/{id}/aprovar", a.handleAprovar)
	mux.HandleFunc("POST /api/propostas/{id}/descartar", a.handleDescartar)
	mux.HandleFunc("DELETE /api/comentarios/{id}", a.handleExcluirComentario)
	// Fase 5: estudar a etapa (T26).
	mux.HandleFunc("GET /api/etapas/{id}/estudo", a.handleEstudo)
	mux.HandleFunc("POST /api/etapas/{id}/exercicios", a.handleGerarEstudo(tipoExercicios))
	mux.HandleFunc("POST /api/etapas/{id}/simulado", a.handleGerarEstudo(tipoSimulado))
	mux.HandleFunc("POST /api/etapas/{id}/exercicios/tentativas", a.handleTentativaExercicios)
	mux.HandleFunc("POST /api/etapas/{id}/simulado/tentativas", a.handleTentativaSimulado)
	mux.HandleFunc("POST /api/etapas/{id}/flashcards", a.handleEnviarFlashcards)
	mux.HandleFunc("GET /api/flashcards/baralhos", a.handleBaralhos)
	mux.HandleFunc("POST /api/criacoes", a.handleCriarComIA)
	mux.HandleFunc("GET /api/trilhas/{id}/criacao", a.handleCriacao)
	mux.HandleFunc("POST /api/trilhas/{id}/criacao/gerar", a.handleGerarPasso)
	mux.HandleFunc("GET /api/tarefas", a.handleTarefas)
	mux.HandleFunc("POST /api/tarefas/{id}/cancelar", a.handleCancelarTarefa)
	mux.HandleFunc("GET /api/ajustes", a.handleAjustes)
	mux.HandleFunc("PUT /api/ajustes", a.handleSalvarAjustes)
	mux.HandleFunc("GET /api/regras", a.handleRegras)
	mux.HandleFunc("GET /api/regras/{arquivo}", a.handleRegra)
	mux.HandleFunc("PUT /api/regras/{arquivo}", a.handleSalvarRegra)
	mux.HandleFunc("DELETE /api/regras/{arquivo}", a.handleRestaurarRegra)

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

func erroInterno(w http.ResponseWriter, err error) {
	log.Print(err)
	erroJSON(w, http.StatusInternalServerError, "erro interno: "+err.Error())
}
