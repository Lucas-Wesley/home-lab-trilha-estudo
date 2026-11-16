package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestVideoaulaNotasResumoEConferencia(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		if strings.Contains(arq["pedido.md"], "FALHE") {
			return "", errors.New("limite da assinatura")
		}
		return `{"veredito":"parcial","nota":140,"certos":["O NAT troca a origem."],
"erros":[{"trecho":"o NAT protege a rede","correcao":"A aula diz que o NAT não é firewall.","onde":"03:10"}],
"faltou":[{"ponto":"Por que faltam endereços IPv4.","onde":"01:05"}],"comentario":"Reveja o começo."}`, nil
	}
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c) // material: o documento 05-wan (faz as vezes da transcrição)

	var an anotacoes
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/anotacoes", nat), nil, &an)
	if an.Notas != "" || an.Resumo != "" || len(an.Conferencias) != 0 {
		t.Fatalf("anotações novas: %+v", an)
	}
	// Salvar um campo não apaga o outro.
	c.ok("PUT", fmt.Sprintf("/api/etapas/%d/anotacoes", nat), map[string]any{"notas": "[01:00] faltam IPv4"}, nil)
	c.ok("PUT", fmt.Sprintf("/api/etapas/%d/anotacoes", nat), map[string]any{"resumo": "curto"}, &an)
	if an.Notas != "[01:00] faltam IPv4" || an.Resumo != "curto" {
		t.Fatalf("salvar: %+v", an)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/resumo/conferir", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("resumo curto demais: %d", s)
	}
	resumo := "O NAT troca o endereço de origem dos pacotes e o NAT protege a rede contra ataques."
	var tt tentativa
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/resumo/conferir", nat), map[string]any{"resumo": resumo}, &tt)
	if tt.Estado != tentCorrigindo || tt.Tipo != tipoResumo || tt.TarefaID == 0 {
		t.Fatalf("conferência: %+v", tt)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/resumo/conferir", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("duas conferências ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	if !strings.Contains(pedido, resumo) || !strings.Contains(pedido, "faltam IPv4") || !strings.Contains(f.arquivos[0]["aula/05-wan.md"], "## 2.3 NAT") ||
		!strings.Contains(f.pedidos[0].regra, "conferir o resumo de uma videoaula") {
		t.Fatalf("pedido:\n%s\narquivos: %v", pedido, f.arquivos[0])
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/anotacoes", nat), nil, &an)
	var res struct {
		Veredito string `json:"veredito"`
		Erros    []struct {
			Onde string `json:"onde"`
		} `json:"erros"`
	}
	got := an.Conferencias[0]
	json.Unmarshal(got.Resultado, &res)
	if an.Resumo != resumo || got.Estado != tentCorrigida || got.Nota != 100 || res.Veredito != "parcial" || res.Erros[0].Onde != "03:10" {
		t.Fatalf("conferida: %+v %+v", got, res)
	}
	// A conferência não entra no painel de domínio nem no estudo dos exercícios.
	var est estudoDaEtapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if est.CriterioAtingido {
		t.Fatalf("critério pela conferência")
	}
	c.ok("GET", "/api/trilhas/1/painel", nil, nil)

	// Conferência que falha fica com o erro.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/resumo/conferir", nat), map[string]any{"resumo": resumo + " FALHE"}, nil)
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/anotacoes", nat), nil, &an)
	if an.Conferencias[0].Estado != tentFalhou || !strings.Contains(an.Conferencias[0].Erro, "limite") {
		t.Fatalf("falha: %+v", an.Conferencias[0])
	}
}
