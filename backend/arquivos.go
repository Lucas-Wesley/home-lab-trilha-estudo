package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// Entrada, fase 7 (T41): "Meus arquivos" — os cursos do Telegram que estão no Streaming
// (curso › módulo › aula, S55 de lá), vistos e baixados daqui. O Streaming continua o dono do
// Telegram: a Trilha só repassa a API dele; nada baixa sozinho (D8, D9 do plano-entrada).

// handleArquivos: GET /api/arquivos devolve o /api/meus-videos do Streaming como veio, mais o
// link dele (para "Assistir" abrir no Streaming).
func (a *app) handleArquivos(w http.ResponseWriter, r *http.Request) {
	dados := map[string]any{}
	if st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodGet, "/api/meus-videos", nil, &dados); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	dados["link_streaming"] = a.integra.streamingLink
	responderJSON(w, http.StatusOK, dados)
}

// handleBaixarArquivos: {"link","grupo","mensagens"} → POST /api/meus-videos/telegram com
// {link, tudo:true, pasta:grupo, mensagens}: baixa só as aulas escolhidas, cada uma na pasta
// do curso e do módulo dela.
func (a *app) handleBaixarArquivos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Link      string `json:"link"`
		Grupo     string `json:"grupo"`
		Mensagens []int  `json:"mensagens"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if !strings.HasPrefix(strings.TrimSpace(corpo.Link), "https://t.me/") || len(corpo.Mensagens) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("mande o link do Telegram e as aulas para baixar")})
		return
	}
	pedido := map[string]any{"link": corpo.Link, "tudo": true, "pasta": corpo.Grupo, "mensagens": corpo.Mensagens}
	saida := map[string]any{}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodPost, "/api/meus-videos/telegram", pedido, &saida)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, saida)
}

// handleApagarPastaArquivos: {"pasta"} → POST /api/meus-videos/apagar-pasta: a lixeira do curso
// ou do módulo (apaga as baixadas da pasta e das subpastas).
func (a *app) handleApagarPastaArquivos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pasta string `json:"pasta"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if strings.TrimSpace(corpo.Pasta) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("mande a pasta")})
		return
	}
	saida := map[string]any{}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodPost, "/api/meus-videos/apagar-pasta", corpo, &saida)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, saida)
}

// handleApagarVideoArquivos: DELETE /api/arquivos/videos/{id} → DELETE /api/meus-videos/{id}:
// apaga um vídeo baixado do servidor.
func (a *app) handleApagarVideoArquivos(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodDelete, "/api/meus-videos/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
