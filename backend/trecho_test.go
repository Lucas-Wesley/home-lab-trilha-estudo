package main

import (
	"fmt"
	"testing"
)

// Entrada, fase 5 (T39): o "visto" do trecho de YouTube — grava o intervalo, põe a etapa em
// andamento no primeiro sinal (tocar ou ver) e conclui quando todos os trechos foram vistos e a
// etapa não tem lição aprovada.

const urlDoVideo = "https://www.youtube.com/watch?v=EOmzo5d0F9Y"

// etapaComTrecho: trilha "Vídeo" com uma etapa cujo material principal é um trecho de YouTube.
func etapaComTrecho(c *cliente, localizador string) (int64, string, string) {
	c.t.Helper()
	var tr trilha
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Vídeo", "profundidade": 3}, &tr)
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/modulos", tr.ID), map[string]any{"titulo": "Módulo"}, nil)
	var e etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "Aula", "materiais": []map[string]any{{
		"tipo": "youtube", "titulo": "A", "referencia": urlDoVideo,
		"localizador": localizador, "papel": "principal"}}}, &e)
	return e.ID, e.Chave, tr.Slug
}

func visto(referencia, localizador string) map[string]any {
	return map[string]any{"referencia": referencia, "localizador": localizador}
}

func TestTrechoVistoGravaAndamentoEConclui(t *testing.T) {
	c := novoCliente(t)
	id, _, _ := etapaComTrecho(c, "00:30–01:00; 02:00–03:00")

	// Tocar já põe a etapa em andamento (o primeiro sinal conta).
	var e etapa
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id), map[string]any{"tocou": true}, &e)
	if e.Status != "em_andamento" || e.IniciadoEm == "" {
		t.Fatalf("tocou: status = %q, iniciado = %q", e.Status, e.IniciadoEm)
	}

	// O primeiro intervalo visto grava e volta no GET; a etapa segue em andamento (falta um).
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id), visto(urlDoVideo, "00:30–01:00"), &e)
	if e.Status != "em_andamento" || len(e.TrechosVistos) != 1 ||
		e.TrechosVistos[0].Referencia != urlDoVideo || e.TrechosVistos[0].Localizador != "00:30–01:00" {
		t.Fatalf("primeiro visto: %+v", e)
	}

	// Repetido não duplica (INSERT OR IGNORE).
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id), visto(urlDoVideo, "00:30-01:00"), &e)
	if len(e.TrechosVistos) != 1 {
		t.Fatalf("visto repetido duplicou: %+v", e.TrechosVistos)
	}

	// O último intervalo conclui a etapa, que não tem lição.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id), visto(urlDoVideo, "02:00–03:00"), &e)
	if e.Status != "concluida" || e.ConcluidoEm == "" || len(e.TrechosVistos) != 2 {
		t.Fatalf("tudo visto: status = %q, vistos = %+v", e.Status, e.TrechosVistos)
	}
}

func TestTrechoVistoDeOutraEtapaDa400(t *testing.T) {
	c := novoCliente(t)
	id, _, _ := etapaComTrecho(c, "00:30–01:00")

	// Um intervalo que não é material principal da etapa não vale.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id),
		visto(urlDoVideo, "05:00–06:00"), nil); s != 400 {
		t.Fatalf("intervalo fora do material: %d", s)
	}
	// Nem um material apoio, nem um trecho de outro vídeo.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id),
		visto("https://youtu.be/outro", "00:30–01:00"), nil); s != 400 {
		t.Fatalf("outro vídeo: %d", s)
	}
	// Nem um pedaço inventado do intervalo (o trecho vale inteiro).
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id),
		visto(urlDoVideo, "00:40–01:00"), nil); s != 400 {
		t.Fatalf("pedaço do intervalo: %d", s)
	}
}

func TestTrechoVistoComLicaoNaoConcluiSozinho(t *testing.T) {
	c := novoCliente(t)
	id, chave, slug := etapaComTrecho(c, "00:30–01:00")

	// Com a lição aprovada, o critério de sempre manda: os trechos não concluem a etapa.
	licao := "# Aula\n\nTexto."
	if err := c.a.conteudo.gravar("x", map[string]*string{caminhoLicao(slug, chave): &licao}); err != nil {
		t.Fatal(err)
	}
	var e etapa
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/trechos/visto", id), visto(urlDoVideo, "00:30–01:00"), &e)
	if !e.TemLicao {
		t.Fatal("a lição não foi vista pela etapa")
	}
	if e.Status != "em_andamento" {
		t.Fatalf("com lição, o trecho visto não conclui: %q", e.Status)
	}
}
