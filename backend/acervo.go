package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Entrada, fase 6 (T40): trazer uma Referência de vídeo do Acervo para a Entrada — a
// transcrição de lá (textos.cru, blocos de ~30 s) entra sem baixar de novo e sem IA.

// fontePorTipoEVideo: a fonte de um tipo com aquele video_id (youtube: id do vídeo; acervo: id
// da Referência).
func (a *app) fontePorTipoEVideo(ctx context.Context, tipo, videoID string) (fonte, error) {
	return lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE tipo = ? AND video_id = ?`, tipo, videoID))
}

// fontePorURL: a fonte (de qualquer tipo) que já tem aquele link — a mesma url não duplica o vídeo.
func (a *app) fontePorURL(ctx context.Context, link string) (fonte, error) {
	return lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE url = ?`, link))
}

// handleEntradaAcervo: {"referencia":"<id do Acervo>"} traz a Referência de vídeo para a
// Entrada. A mesma Referência (ou o mesmo vídeo já vindo do YouTube) devolve a fonte existente.
func (a *app) handleEntradaAcervo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Referencia string `json:"referencia"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	id := strings.TrimSpace(corpo.Referencia)
	if id == "" {
		a.responderErro(w, erroDeEntrada{errors.New("mande a Referência do Acervo")})
		return
	}
	ctx := r.Context()
	var x referenciaAcervo
	if st, err := chamarApp(ctx, a.integra.acervo, "Acervo", http.MethodGet, "/api/referencias/"+url.PathEscape(id), nil, &x); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	videoID, errVideo := videoDoLink(x.URL)
	if x.Tipo != "video" || errVideo != nil {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("a Referência %q não é um vídeo do YouTube (tipo %q)", id, x.Tipo)})
		return
	}
	cru := strings.TrimSpace(x.Textos["cru"])
	if cru == "" {
		a.responderErro(w, erroDeEntrada{errors.New("a Referência ainda não tem a transcrição crua (textos.cru)")})
		return
	}

	// A mesma Referência de novo — ou o vídeo dela já vindo do YouTube — devolve a fonte que existe.
	link := "https://www.youtube.com/watch?v=" + videoID
	if f, err := a.fontePorTipoEVideo(ctx, "acervo", x.ID); err == nil {
		responderJSON(w, http.StatusOK, f)
		return
	}
	if f, err := a.fontePorURL(ctx, link); err == nil {
		responderJSON(w, http.StatusOK, f)
		return
	}

	f, err := a.criarFonte(ctx, "acervo", link, x.ID, x.Titulo, "")
	if err != nil {
		a.responderErro(w, err)
		return
	}
	duracao, _ := segundosDoTempo(x.Duracao) // "46:58" ou "1:02:03" → segundos; ilegível fica 0
	if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET canal = ?, duracao = ?, estado = 'pronta' WHERE id = ?`,
		x.Canal, duracao, f.ID); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.conteudo.gravar("entrada "+x.ID+": transcrição do Acervo", map[string]*string{
		pastaDaFonte(f.ID) + "/transcricao.md": &cru,
	}); err != nil {
		a.responderErro(w, err)
		return
	}
	f, err = a.fontePorID(ctx, f.ID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, f)
}
