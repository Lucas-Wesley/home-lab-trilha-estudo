package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestRevisarRoteiro(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"mudancas":[
{"acao":"editar","etapa_chave":"nat","titulo":"","objetivo":"Explicar o NAT e o CGNAT.","criterio":"","bloom":"","minutos":45,"modulo":"","depois_de":"","motivo":"O alvo pede CGNAT."},
{"acao":"editar","etapa_chave":"nat","titulo":"NAT","objetivo":"","criterio":"","bloom":"","minutos":0,"modulo":"","depois_de":"","motivo":"nada muda"},
{"acao":"novo","etapa_chave":"","titulo":"Máscaras","objetivo":"Calcular sub-redes.","criterio":"Calcular 3 sub-redes.","bloom":"aplicar","minutos":40,"modulo":"Internet","depois_de":"ip-e-dhcp","motivo":"Falta base."},
{"acao":"mover","etapa_chave":"ip-e-dhcp","titulo":"","objetivo":"","criterio":"","bloom":"","minutos":0,"modulo":"Base","depois_de":"","motivo":"É base."},
{"acao":"remover","etapa_chave":"nao-existe","titulo":"","objetivo":"","criterio":"","bloom":"","minutos":0,"modulo":"","depois_de":"","motivo":"x"}],
"notas":"Roteiro curto.","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	trilhaComEtapa(c) // módulo "Internet": ip-e-dhcp, nat
	c.ok("PUT", "/api/trilhas/1", map[string]any{"titulo": "Rede", "profundidade": 4, "nivel_partida": "Dev web", "nivel_alvo": "SRE"}, nil)

	// A trilha inteira.
	var p proposta
	c.ok("POST", "/api/trilhas/1/roteiro", map[string]any{"pedido": "Mudei o alvo."}, &p)
	if p.EtapaChave != "@roteiro" || p.Tarefa == nil {
		t.Fatalf("proposta: %+v", p)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	if !strings.Contains(pedido, "Nível alvo: SRE") || !strings.Contains(pedido, "**1.2** `nat`") || !strings.Contains(pedido, "Deste pedido: Mudei o alvo.") ||
		strings.Contains(pedido, "## Foco") || !strings.Contains(f.pedidos[0].regra, "revisar o roteiro") {
		t.Fatalf("pedido:\n%s", pedido)
	}
	var r struct {
		Proposta *proposta `json:"proposta"`
	}
	c.ok("GET", "/api/trilhas/1/roteiro", nil, &r)
	var cr conteudoRoteiro
	json.Unmarshal(r.Proposta.Conteudo, &cr)
	// Saem a edição que não muda nada e a etapa que não existe; "Base" é módulo novo.
	if len(cr.Mudancas) != 3 || cr.Mudancas[0].Atual == nil || cr.Mudancas[0].Atual.Objetivo != "Explicar o NAT." || cr.Mudancas[0].Numero != "1.2" ||
		!cr.Mudancas[2].ModuloNovo || !strings.Contains(cr.Notas, "nao-existe") {
		t.Fatalf("conteúdo: %+v", cr)
	}

	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", r.Proposta.ID), map[string]any{"mudancas": []int{0, 1, 2}}, nil)
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	if len(tr.Modulos) != 2 || tr.Modulos[1].Titulo != "Base" || tr.Modulos[1].Etapas[0].Chave != "ip-e-dhcp" {
		t.Fatalf("módulos: %+v", tr.Modulos)
	}
	es := tr.Modulos[0].Etapas
	// "Máscaras" ia depois de ip-e-dhcp, que saiu do módulo: vai para o fim.
	if len(es) != 2 || es[0].Chave != "nat" || es[0].Objetivo != "Explicar o NAT e o CGNAT." || es[0].Minutos != 45 || es[0].Titulo != "NAT" ||
		es[1].Titulo != "Máscaras" || es[1].Criterio != "Calcular 3 sub-redes." {
		t.Fatalf("etapas: %+v", es)
	}

	// Um tópico só: o pedido traz o foco e a proposta tem chave própria.
	c.ok("POST", "/api/trilhas/1/roteiro", map[string]any{"etapa_chave": "nat"}, &p)
	c.a.fila.esvaziar(context.Background())
	if p.EtapaChave != "@roteiro:nat" || !strings.Contains(f.arquivos[1]["pedido.md"], "## Foco: só o tópico `nat`") {
		t.Fatalf("foco: %s\n%s", p.EtapaChave, f.arquivos[1]["pedido.md"])
	}
	c.ok("GET", "/api/trilhas/1/roteiro?etapa=nat", nil, &r)
	if r.Proposta == nil || r.Proposta.ID != p.ID {
		t.Fatalf("roteiro do tópico: %+v", r.Proposta)
	}
	if s := c.chamar("POST", "/api/trilhas/1/roteiro", map[string]any{"etapa_chave": "nao-existe"}, nil); s != 400 {
		t.Fatalf("tópico que não existe: %d", s)
	}
}

func TestEncaixeNoTopicoComRevisao(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"encaixes":[{"etapa_chave":"nat","localizador":"seção 2.3","papel":"principal","nota":"NAT.","motivo":"x"}],"novos":[],
"revisoes":[{"etapa_chave":"nat","titulo":"","objetivo":"Explicar NAT e CGNAT.","criterio":"","bloom":"","minutos":0,"motivo":"O material mostra o CGNAT."}],
"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	trilhaComEtapa(c)
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{"tipo": "documento", "titulo": "WAN", "referencia": "05-wan"}, "etapa_chave": "nat"}, nil)
	c.a.fila.esvaziar(context.Background())
	if !strings.Contains(f.arquivos[0]["pedido.md"], "## Foco: o Lucas mandou este material para o tópico **1.2** `nat`") {
		t.Fatalf("pedido:\n%s", f.arquivos[0]["pedido.md"])
	}
	var lista []struct {
		Proposta proposta `json:"proposta"`
		Foco     string   `json:"foco"`
	}
	c.ok("GET", "/api/trilhas/1/encaixes?etapa=ip-e-dhcp", nil, &lista)
	if len(lista) != 0 {
		t.Fatalf("filtro: %+v", lista)
	}
	c.ok("GET", "/api/trilhas/1/encaixes?etapa=nat", nil, &lista)
	if len(lista) != 1 || lista[0].Foco != "nat" {
		t.Fatalf("lista: %+v", lista)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].Proposta.ID), map[string]any{"encaixes": []int{0}, "revisoes": []int{0}}, nil)
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	nat := tr.Modulos[0].Etapas[1]
	if nat.Objetivo != "Explicar NAT e CGNAT." || len(nat.Materiais) != 2 {
		t.Fatalf("nat: %+v", nat)
	}
}
