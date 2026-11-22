package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestVerificarMaterial(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, arq map[string]string) (string, error) {
		if strings.Contains(arq["pedido.md"], "FALHE") {
			return "", errors.New("limite da assinatura")
		}
		return `{"topicos":[{"topico":"Por que o NAT existe","situacao":"coberto","onde":"Material 2, 00:10","comentario":"ok"},
{"topico":"CGNAT","situacao":"inventado","onde":"","comentario":""},{"topico":"  ","situacao":"ausente","onde":"","comentario":""}],
"materiais":[{"material":1,"lido":true,"cobertura":150,"profundidade":"na_medida","comentario":"bom","fora":""},
{"material":9,"lido":true,"cobertura":10,"profundidade":"abaixo","comentario":"não existe","fora":""}],
"cobertura":-5,"veredito":"talvez","faltas":[{"ponto":"Port forwarding","onde_achar":"RFC 3022"}],"resumo":"Complemente."}`, nil
	}
	c := clienteFragmentos(t, f)
	trilhaComEtapa(c)
	var e etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "NAT na prática", "objetivo": "Configurar o NAT.", "criterio": "Explicar o masquerade.",
		"materiais": []map[string]any{
			{"tipo": "documento", "titulo": "WAN", "referencia": "05-wan", "localizador": "23-nat", "papel": "principal"},
			{"tipo": "youtube", "titulo": "Aula de NAT", "referencia": "https://youtu.be/AAAAAAAAAAA", "localizador": "", "papel": "principal"},
			{"tipo": "web", "titulo": "RFC 3022", "referencia": "https://www.rfc-editor.org/rfc/rfc3022", "localizador": "", "papel": "apoio"},
			{"tipo": "streaming", "titulo": "Vídeo solto", "referencia": "http://streaming/api/video/proprio/3.mp4", "localizador": "", "papel": "apoio"},
		}}, &e)
	url := fmt.Sprintf("/api/etapas/%d/verificar", e.ID)

	// O vídeo sem transcrição entra na Entrada e a verificação espera.
	var v verificacao
	if s := c.chamar("POST", url, map[string]any{"pedido": "Foco em OpenWrt."}, &v); s != http.StatusCreated {
		t.Fatalf("verificar: %d", s)
	}
	if v.Estado != verAguardando || len(v.Materiais) != 4 || v.Materiais[1].Texto != textoTranscrevendo || v.Materiais[1].FonteID == 0 ||
		v.Materiais[0].Texto != textoPronto || v.Materiais[2].Texto != textoWeb || v.Materiais[3].Texto != textoSem {
		t.Fatalf("aguardando: %+v", v)
	}
	if s := c.chamar("POST", url, nil, nil); s != http.StatusBadRequest {
		t.Fatalf("duas verificações ao mesmo tempo: %d", s)
	}
	c.a.esvaziarEntrada(context.Background())
	var lista []verificacao
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/verificacoes", e.ID), nil, &lista)
	if len(lista) != 1 || lista[0].Estado != verVerificando || lista[0].Tarefa == nil || lista[0].Materiais[1].Texto != textoPronto {
		t.Fatalf("depois da transcrição: %+v", lista)
	}
	c.a.fila.esvaziar(context.Background())

	pedido := f.arquivos[0]["pedido.md"]
	for _, trecho := range []string{"Configurar o NAT.", "Explicar o masquerade.", "NAT na prática ← esta", "`documentos/05-wan.md`",
		"rfc3022", "Material 2 · Aula de NAT", "material/02-youtube.md", "O texto não veio", "Deste pedido: Foco em OpenWrt."} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md sem %q:\n%s", trecho, pedido)
		}
	}
	if strings.TrimSpace(f.arquivos[0]["material/02-youtube.md"]) == "" || !strings.Contains(f.pedidos[0].regra, "verificar se o material cobre") ||
		!strings.Contains(f.pedidos[0].ferramentas, "WebFetch") {
		t.Errorf("transcrição, regra ou ferramentas: %v %q", f.arquivos[0], f.pedidos[0].ferramentas)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/verificacoes", e.ID), nil, &lista)
	var l laudo
	json.Unmarshal(lista[0].Resultado, &l)
	if lista[0].Estado != verPronta || l.Cobertura != 0 || l.Veredito != "complementar" || len(l.Topicos) != 2 || l.Topicos[1].Situacao != "raso" ||
		len(l.Materiais) != 1 || l.Materiais[0].Cobertura != 100 || l.Faltas[0].OndeAchar != "RFC 3022" {
		t.Fatalf("laudo: %+v %+v", lista[0], l)
	}

	// Só um material, já legível: vai direto para a fila; a falha fica registrada.
	c.ok("POST", url, map[string]any{"materiais": []int{0}, "pedido": "FALHE"}, &v)
	if v.Estado != verVerificando || len(v.Materiais) != 1 {
		t.Fatalf("um material: %+v", v)
	}
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/verificacoes", e.ID), nil, &lista)
	if len(lista) != 2 || lista[0].Estado != verFalhou || !strings.Contains(lista[0].Erro, "limite") {
		t.Fatalf("falha: %+v", lista[0])
	}

	// Material fora da etapa e etapa sem material: 400.
	if s := c.chamar("POST", url, map[string]any{"materiais": []int{7}}, nil); s != http.StatusBadRequest {
		t.Fatalf("material inexistente: %d", s)
	}
	var vazia etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "Sem material"}, &vazia)
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/verificar", vazia.ID), nil, nil); s != http.StatusBadRequest {
		t.Fatalf("etapa sem material: %d", s)
	}
}
