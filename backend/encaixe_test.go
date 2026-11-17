package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestEncaixarMaterial(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"encaixes":[
{"etapa_chave":"nat","localizador":"seção 2.3","papel":"principal","nota":"Explica o NAT.","motivo":"É o assunto da etapa."},
{"etapa_chave":"nao-existe","localizador":"","papel":"apoio","nota":"x","motivo":"x"}],
"novos":[
{"titulo":"CGNAT","objetivo":"Explicar o CGNAT.","bloom":"entender","minutos":40,"modulo":"Internet","depois_de":"nat","localizador":"seção 2.4","nota":"CGNAT.","motivo":"Falta."},
{"titulo":"IPv6","objetivo":"Explicar o IPv6.","bloom":"voar","minutos":0,"modulo":"Futuro","depois_de":"","localizador":"","nota":"","motivo":"Falta."}],
"notas":"Um documento sobre WAN.","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	trilhaComEtapa(c) // módulo "Internet": ip-e-dhcp, nat; documento 05-wan

	if s := c.chamar("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{"tipo": "streaming", "referencia": "x"}}, nil); s != 400 {
		t.Fatalf("tipo que não encaixa: %d", s)
	}
	var p proposta
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{"tipo": "documento", "titulo": "WAN", "referencia": "05-wan"}, "pedido": "Veja o CGNAT."}, &p)
	// Dois materiais podem estar na fila ao mesmo tempo.
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{"tipo": "web", "titulo": "Artigo", "referencia": "https://exemplo.org/nat"}}, nil)
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	if !strings.Contains(pedido, "**1.2** `nat` — NAT") || !strings.Contains(pedido, "documentos/05-wan.md") || !strings.Contains(pedido, "Deste pedido: Veja o CGNAT.") ||
		!strings.Contains(f.pedidos[0].regra, "encaixar um material") || strings.Contains(f.pedidos[0].ferramentas, "WebFetch") || !strings.Contains(f.pedidos[1].ferramentas, "WebFetch") {
		t.Fatalf("pedido:\n%s\nferramentas: %s / %s", pedido, f.pedidos[0].ferramentas, f.pedidos[1].ferramentas)
	}

	var lista []struct {
		Proposta proposta `json:"proposta"`
		Material material `json:"material"`
	}
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &lista)
	if len(lista) != 2 || lista[1].Material.Referencia != "05-wan" {
		t.Fatalf("lista: %+v", lista)
	}
	var ce conteudoEncaixe
	json.Unmarshal(lista[1].Proposta.Conteudo, &ce)
	if len(ce.Encaixes) != 1 || ce.Encaixes[0].Numero != "1.2" || len(ce.Novos) != 2 || ce.Novos[1].Bloom != "entender" || !ce.Novos[1].ModuloNovo ||
		!strings.Contains(ce.Notas, "nao-existe") {
		t.Fatalf("conteúdo: %+v", ce)
	}

	// Ajuste: a versão nova reaproveita o material da primeira.
	pid := lista[1].Proposta.ID
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/ajustar", pid), map[string]any{"pedido": "Sem o IPv6."}, nil)
	c.a.fila.esvaziar(context.Background())
	if !strings.Contains(f.arquivos[2]["pedido.md"], "documentos/05-wan.md") || f.arquivos[2]["versao-anterior.json"] == "" {
		t.Fatalf("ajuste:\n%s", f.arquivos[2]["pedido.md"])
	}
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &lista)
	nova := lista[1].Proposta

	// Aprovar sem marcar nada é recusado; com o encaixe e o CGNAT, só eles entram.
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", nova.ID), map[string]any{}, nil); s != 400 {
		t.Fatalf("aprovar sem marcar: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", nova.ID), map[string]any{"encaixes": []int{0}, "novos": []int{0}}, nil)
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	es := tr.Modulos[0].Etapas
	if len(tr.Modulos) != 1 || len(es) != 3 || es[1].Chave != "nat" || es[2].Titulo != "CGNAT" || es[2].Materiais[0].Localizador != "seção 2.4" || es[2].Materiais[0].Papel != "principal" {
		t.Fatalf("trilha depois: %+v", es)
	}
	var nat etapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d", es[1].ID), nil, &nat)
	ultimo := nat.Materiais[len(nat.Materiais)-1]
	if len(nat.Materiais) != 2 || ultimo.Localizador != "seção 2.3" || ultimo.Nota != "Explica o NAT." {
		t.Fatalf("material da etapa: %+v", nat.Materiais)
	}
	// O mesmo material no mesmo trecho não entra de novo.
	_, _, tirados := limparEncaixe(tr, ultimo, []encaixeEtapa{{EtapaChave: "nat", Localizador: "seção 2.3"}}, nil)
	if len(tirados) != 1 {
		t.Fatalf("repetido: %v", tirados)
	}
}
