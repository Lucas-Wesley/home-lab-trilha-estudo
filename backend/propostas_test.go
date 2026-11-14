package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// iaFalsa guarda o que recebeu (o pedido.md e os outros arquivos) e devolve a resposta da vez.
type iaFalsa struct {
	pedidos  []pedidoClaude
	arquivos []map[string]string
	resposta func(n int, arquivos map[string]string) (string, error)
}

func (f *iaFalsa) chamar(_ context.Context, p pedidoClaude) (respostaClaude, error) {
	arq := map[string]string{}
	entradas, _ := os.ReadDir(p.pasta)
	for _, e := range entradas {
		b, _ := os.ReadFile(filepath.Join(p.pasta, e.Name()))
		arq[e.Name()] = string(b)
	}
	f.pedidos = append(f.pedidos, p)
	f.arquivos = append(f.arquivos, arq)
	texto, err := f.resposta(len(f.pedidos), arq)
	if err != nil {
		return respostaClaude{}, err
	}
	return respostaClaude{Estruturado: json.RawMessage(texto)}, nil
}

func clienteComIA(t *testing.T, f *iaFalsa) *cliente {
	c := novoCliente(t)
	c.a.ia.chamar = f.chamar
	c.a.ia.conferirLinks = func(context.Context, string) []linkConferido {
		return []linkConferido{{URL: "https://exemplo.org/x", Situacao: "quebrado", Detalhe: "404 Not Found"}}
	}
	return c
}

// trilhaComEtapa: trilha "Rede" com um documento e duas etapas; devolve o id da segunda (NAT).
func trilhaComEtapa(c *cliente) int64 {
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Rede", "profundidade": 4, "instrucoes": "Exemplos com OpenWrt."}, nil)
	c.ok("PUT", "/api/trilhas/1/documentos/05-wan", map[string]any{"conteudo": "# WAN\n\n## 2.3 NAT\n\nTexto."}, nil)
	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "Internet"}, nil)
	var ip, nat etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "IP e DHCP", "objetivo": "Explicar o DHCP."}, &ip)
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "NAT", "objetivo": "Explicar o NAT.", "bloom": "entender",
		"minutos": 30, "criterio": "Explicar por que o NAT existe.", "requer": []int64{ip.ID},
		"materiais": []map[string]any{{"tipo": "documento", "titulo": "WAN", "referencia": "05-wan", "localizador": "23-nat", "papel": "principal"}}}, &nat)
	return nat.ID
}

func TestLicaoPropostaComentarAjustarAprovar(t *testing.T) {
	var idComentario int64
	f := &iaFalsa{}
	f.resposta = func(n int, _ map[string]string) (string, error) {
		if n == 1 {
			return `{"licao":"# NAT\n\n## Por que existe\n\nTexto v1.","notas":"Achei um erro no 05.","respostas":[]}`, nil
		}
		return fmt.Sprintf(`{"licao":"# NAT\n\n## Por que existe\n\nTexto v2, com IPv6.","notas":"","respostas":[{"comentario":%d,"resposta":"Comparei com o IPv6 na seção 2."},{"comentario":999,"resposta":"inventado"}]}`, idComentario), nil
	}
	c := clienteComIA(t, f)
	c.ok("PUT", "/api/ajustes", map[string]any{"instrucoes_globais": "Português do Brasil.", "modelo_licao": "sonnet"}, nil)
	nat := trilhaComEtapa(c)

	var p proposta
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{"pedido": "Compare com o CGNAT."}, &p); s != 201 {
		t.Fatalf("pedir lição: %d %+v", s, p)
	}
	if p.Estado != propGerando || p.Versao != 1 || p.Tarefa == nil || p.Tarefa.Estado != naFila || p.Tarefa.Modelo != "sonnet" {
		t.Fatalf("proposta nova: %+v %+v", p, p.Tarefa)
	}
	// Enquanto gera, não dá para pedir outra.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("segunda proposta ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())

	// O pedido levou a etapa, o lugar dela, o material e as três instruções.
	pedido := f.arquivos[0]["pedido.md"]
	for _, trecho := range []string{"Explicar o NAT.", "Nível de Bloom do objetivo: entender", "4. Aprofundado", "Pré-requisitos",
		"IP e DHCP", "`documentos/05-wan.md`", "`#23-nat`", "Globais: Português do Brasil.", "Da trilha: Exemplos com OpenWrt.",
		"Deste pedido: Compare com o CGNAT.", "← esta"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md sem %q:\n%s", trecho, pedido)
		}
	}
	if len(f.pedidos[0].extras) != 1 || !strings.HasSuffix(filepath.ToSlash(f.pedidos[0].extras[0]), "trilhas/rede/documentos") {
		t.Errorf("pastas extras: %v", f.pedidos[0].extras)
	}
	if !strings.Contains(f.pedidos[0].regra, "a lição de uma etapa") || f.pedidos[0].modelo != "sonnet" {
		t.Errorf("regra ou modelo: %q %q", f.pedidos[0].regra[:40], f.pedidos[0].modelo)
	}

	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &lista)
	if len(lista) != 1 || lista[0].Estado != propAguardando {
		t.Fatalf("depois de gerar: %+v", lista)
	}
	var cl conteudoLicao
	json.Unmarshal(lista[0].Conteudo, &cl)
	if !strings.Contains(cl.Licao, "Texto v1") || cl.Notas == "" || len(cl.Links) != 1 || cl.Links[0].Situacao != "quebrado" {
		t.Fatalf("conteúdo v1: %+v", cl)
	}
	v1 := lista[0].ID

	// Ajustar sem dizer o quê: recusado.
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/ajustar", v1), map[string]any{}, nil); s != 400 {
		t.Fatalf("ajuste sem comentário nem pedido: %d", s)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/comentarios", v1), map[string]any{"texto": "  "}, nil); s != 400 {
		t.Fatalf("comentário vazio: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/comentarios", v1), map[string]any{"texto": "Compare com o IPv6.", "ancora": "secao:por-que-existe", "citacao": "Texto v1."}, &p)
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/comentarios", v1), map[string]any{"texto": "apagar"}, &p)
	if len(p.Comentarios) != 2 {
		t.Fatalf("comentários: %+v", p.Comentarios)
	}
	idComentario = p.Comentarios[0].ID
	c.ok("DELETE", fmt.Sprintf("/api/comentarios/%d", p.Comentarios[1].ID), nil, &p)
	if len(p.Comentarios) != 1 {
		t.Fatalf("comentário apagado: %+v", p.Comentarios)
	}

	var v2 proposta
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/ajustar", v1), map[string]any{"pedido": "Mais curto."}, &v2); s != 201 || v2.Versao != 2 || v2.AnteriorID != v1 {
		t.Fatalf("ajustar: %d %+v", s, v2)
	}
	// Enquanto a v2 gera, a v1 continua aguardando, mas não pode ser aprovada.
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", v1), nil, nil); s != 400 {
		t.Fatalf("aprovar com versão nova gerando: %d", s)
	}
	c.a.fila.esvaziar(context.Background())

	pedido2 := f.arquivos[1]["pedido.md"]
	if !strings.Contains(f.arquivos[1]["versao-anterior.md"], "Texto v1") ||
		!strings.Contains(pedido2, fmt.Sprintf("Comentário %d (em secao:por-que-existe), sobre o trecho: «Texto v1.»: Compare com o IPv6.", idComentario)) ||
		!strings.Contains(pedido2, "Deste pedido: Mais curto.") {
		t.Fatalf("pedido do ajuste:\n%s", pedido2)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &lista)
	if lista[0].Estado != propSubstituida || lista[1].Estado != propAguardando {
		t.Fatalf("estados depois do ajuste: %s %s", lista[0].Estado, lista[1].Estado)
	}
	// A resposta da IA ao comentário fica na v2; a resposta a um comentário que não existe, não.
	if r := lista[1].Comentarios; len(r) != 1 || r[0].Autor != "ia" || r[0].RespondeA != idComentario {
		t.Fatalf("respostas da IA: %+v", r)
	}
	// Comentário da IA não é apagável.
	if s := c.chamar("DELETE", fmt.Sprintf("/api/comentarios/%d", lista[1].Comentarios[0].ID), nil, nil); s != 400 {
		t.Fatalf("apagar resposta da IA: %d", s)
	}

	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[1].ID), nil, &p)
	if p.Estado != propAprovada {
		t.Fatalf("aprovada: %+v", p)
	}
	var e etapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d", nat), nil, &e)
	if e.Licao == nil || !strings.Contains(*e.Licao, "Texto v2") {
		t.Fatalf("a lição aprovada não foi para a etapa: %+v", e.Licao)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/comentarios", p.ID), map[string]any{"texto": "x"}, nil); s != 400 {
		t.Fatalf("comentar proposta aprovada: %d", s)
	}

	// "Pedir mudança" numa lição aprovada: a IA recebe a lição em vigor.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{"pedido": "Acrescente um diagrama."}, &p)
	c.a.fila.esvaziar(context.Background())
	if p.Versao != 3 || !strings.Contains(f.arquivos[2]["licao-atual.md"], "Texto v2") || !strings.Contains(f.arquivos[2]["pedido.md"], "licao-atual.md") {
		t.Fatalf("mudança: v%d %v", p.Versao, f.arquivos[2])
	}
}

func TestLicaoQueFalhaECancelamento(t *testing.T) {
	f := &iaFalsa{resposta: func(int, map[string]string) (string, error) {
		return "", errors.New("o Claude recusou o login")
	}}
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c)

	var p proposta
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{}, &p)
	c.a.fila.esvaziar(context.Background())
	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &lista)
	if lista[0].Estado != propFalhou || !strings.Contains(lista[0].Erro, "login") || lista[0].Tarefa.Estado != falhou {
		t.Fatalf("falha: %+v", lista[0])
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/descartar", p.ID), nil, &p)

	// Cancelar uma tarefa na fila leva a proposta junto, e dá para pedir de novo.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{}, &p)
	if s := c.chamar("POST", fmt.Sprintf("/api/tarefas/%d/cancelar", p.Tarefa.ID), nil, nil); s != 204 {
		t.Fatalf("cancelar: %d", s)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &lista)
	if lista[1].Estado != propFalhou || lista[1].Tarefa.Estado != cancelada {
		t.Fatalf("cancelada: %+v", lista[1])
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/tarefas/%d/cancelar", p.Tarefa.ID), nil, nil); s != 400 {
		t.Fatalf("cancelar de novo: %d", s)
	}
	var tarefas []tarefa
	c.ok("GET", "/api/tarefas", nil, &tarefas)
	if len(tarefas) != 2 || tarefas[0].TrilhaID != 1 || tarefas[0].EtapaChave != "nat" {
		t.Fatalf("tarefas: %+v", tarefas)
	}
	// A IA errou o formato: a tarefa falha com a mensagem.
	f.resposta = func(int, map[string]string) (string, error) { return `{"licao":"  ","notas":"","respostas":[]}`, nil }
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", nat), map[string]any{}, &p)
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &lista)
	if lista[2].Estado != propFalhou || !strings.Contains(lista[2].Erro, "vazia") {
		t.Fatalf("lição vazia: %+v", lista[2])
	}
}

func TestAjustesERegras(t *testing.T) {
	c := novoCliente(t)
	if s := c.chamar("PUT", "/api/ajustes", map[string]any{"modelo_licao": "gpt"}, nil); s != 400 {
		t.Fatalf("modelo desconhecido: %d", s)
	}
	var x ajustes
	c.ok("PUT", "/api/ajustes", map[string]any{"instrucoes_globais": " Sem jargão. ", "modelo_licao": "claude-opus-5-5"}, &x)
	if x.InstrucoesGlobais != "Sem jargão." || x.ModeloLicao != "claude-opus-5-5" {
		t.Fatalf("ajustes: %+v", x)
	}

	var regras []regra
	c.ok("GET", "/api/regras", nil, &regras)
	if len(regras) != len(regrasDoApp) || regras[0].Arquivo != "licao.md" || regras[0].Editada || regras[0].Texto != "" {
		t.Fatalf("regras: %+v", regras)
	}
	var rg regra
	c.ok("PUT", "/api/regras/licao.md", map[string]any{"texto": "# Minha regra"}, &rg)
	if !rg.Editada || rg.Texto != "# Minha regra" || !strings.Contains(rg.Original, "a lição de uma etapa") {
		t.Fatalf("regra editada: %+v", rg.Editada)
	}
	if s := c.chamar("PUT", "/api/regras/outra.md", map[string]any{"texto": "x"}, nil); s != 404 {
		t.Fatalf("regra que não existe: %d", s)
	}
	if s := c.chamar("PUT", "/api/regras/licao.md", map[string]any{"texto": " "}, nil); s != 400 {
		t.Fatalf("regra vazia: %d", s)
	}
	c.ok("DELETE", "/api/regras/licao.md", nil, &rg)
	if rg.Editada || rg.Texto != rg.Original {
		t.Fatalf("restaurar: %+v", rg.Editada)
	}
}

func TestConferirLinks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sumiu":
			w.WriteHeader(404)
		case "/robo":
			w.WriteHeader(403)
		}
	}))
	defer srv.Close()
	texto := fmt.Sprintf("Veja [a doc](%s/ok), %s/sumiu. e <%s/robo>; de novo %s/ok", srv.URL, srv.URL, srv.URL, srv.URL)
	if l := linksDoTexto(texto); len(l) != 3 {
		t.Fatalf("links achados: %v", l)
	}
	res := conferirLinks(context.Background(), texto)
	esperado := []string{"ok", "quebrado", "nao_confirmado"}
	for i, l := range res {
		if l.Situacao != esperado[i] {
			t.Errorf("%s: %s, esperado %s", l.URL, l.Situacao, esperado[i])
		}
	}
}
