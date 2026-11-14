package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

// Documentos são material próprio em Markdown de uma trilha (ex.: os docs de estudo do homelab na
// trilha de rede). Pertencem à trilha: ficam em trilhas/<slug>/documentos/, aparecem só nela e saem
// junto quando ela é excluída. Entram à mão, em qualquer forma de criar a trilha.

func caminhoDocumento(slugTrilha, doc string) string {
	return "trilhas/" + slugTrilha + "/documentos/" + doc + ".md"
}

// trilhaDaRota lê {id} da rota e devolve o slug da trilha.
func (a *app) trilhaDaRota(r *http.Request) (string, error) {
	id, err := idDaRota(r)
	if err != nil {
		return "", err
	}
	slug := a.slugDaTrilha(r.Context(), id)
	if slug == "" {
		return "", errNaoEncontrou
	}
	return slug, nil
}

func nomeDocumento(r *http.Request) (string, error) {
	doc := r.PathValue("doc")
	if !slugValido.MatchString(doc) {
		return "", erroDeEntrada{fmt.Errorf("nome de documento inválido: %q (use letras minúsculas, números e hífens)", doc)}
	}
	return doc, nil
}

func (a *app) handleDocumentos(w http.ResponseWriter, r *http.Request) {
	slug, err := a.trilhaDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	lista, err := a.conteudo.documentos("trilhas/" + slug + "/documentos")
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, lista)
}

func (a *app) handleDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := a.trilhaDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	doc, err := nomeDocumento(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	texto, err := a.conteudo.ler(caminhoDocumento(slug, doc))
	if errors.Is(err, fs.ErrNotExist) {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, documento{Slug: doc, Titulo: tituloDoMarkdown(texto, doc), Conteudo: texto})
}

func (a *app) handleSalvarDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := a.trilhaDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	doc, err := nomeDocumento(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Conteudo string `json:"conteudo"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if strings.TrimSpace(corpo.Conteudo) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("o documento está vazio")})
		return
	}
	if err := a.conteudo.gravar(slug+": documento "+doc, map[string]*string{caminhoDocumento(slug, doc): &corpo.Conteudo}); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleDocumento(w, r)
}

func (a *app) handleExcluirDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := a.trilhaDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	doc, err := nomeDocumento(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.conteudo.gravar(slug+": documento "+doc+" excluído", map[string]*string{caminhoDocumento(slug, doc): nil}); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
