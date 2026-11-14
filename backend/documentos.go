package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

func (a *app) handleDocumentos(w http.ResponseWriter, r *http.Request) {
	lista, err := a.conteudo.documentos()
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, lista)
}

func slugDocumento(r *http.Request) (string, error) {
	slug := r.PathValue("slug")
	if !slugValido.MatchString(slug) {
		return "", erroDeEntrada{fmt.Errorf("nome de documento inválido: %q (use letras minúsculas, números e hífens)", slug)}
	}
	return slug, nil
}

func (a *app) handleDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := slugDocumento(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	texto, err := a.conteudo.ler("documentos/" + slug + ".md")
	if errors.Is(err, fs.ErrNotExist) {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, documento{Slug: slug, Titulo: tituloDoMarkdown(texto, slug), Conteudo: texto})
}

func (a *app) handleSalvarDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := slugDocumento(r)
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
	if err := a.conteudo.gravar("documento "+slug, map[string]*string{"documentos/" + slug + ".md": &corpo.Conteudo}); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleDocumento(w, r)
}

func (a *app) handleExcluirDocumento(w http.ResponseWriter, r *http.Request) {
	slug, err := slugDocumento(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.conteudo.gravar("documento "+slug+" excluído", map[string]*string{"documentos/" + slug + ".md": nil}); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
