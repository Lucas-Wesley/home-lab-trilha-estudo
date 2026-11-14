package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type cliente struct {
	t *testing.T
	h http.Handler
	a *app
}

func novoCliente(t *testing.T) *cliente {
	a := novoApp(t)
	return &cliente{t: t, h: novoServidor(a, fstest.MapFS{"index.html": {Data: []byte("x")}}, "teste"), a: a}
}

// chamar faz o pedido e decodifica a resposta em destino (se não for nil); devolve o status.
func (c *cliente) chamar(metodo, caminho string, corpo, destino any) int {
	c.t.Helper()
	var leitor *bytes.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	} else {
		leitor = bytes.NewReader(nil)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, httptest.NewRequest(metodo, caminho, leitor))
	if destino != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), destino); err != nil {
			c.t.Fatalf("%s %s: resposta ilegível (%d): %s", metodo, caminho, rec.Code, rec.Body.String())
		}
	}
	return rec.Code
}

func (c *cliente) ok(metodo, caminho string, corpo, destino any) {
	c.t.Helper()
	var qualquer any
	if destino == nil {
		destino = &qualquer
	}
	if s := c.chamar(metodo, caminho, corpo, destino); s >= 300 {
		c.t.Fatalf("%s %s: status %d %v", metodo, caminho, s, destino)
	}
}

func TestCriarTrilhaModuloEtapaELicao(t *testing.T) {
	c := novoCliente(t)
	var tr trilha
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Home lab: rede", "profundidade": 3}, &tr)
	if tr.Slug != "home-lab-rede" || tr.Estado != "rascunho" {
		t.Fatalf("trilha criada errada: %+v", tr)
	}
	// O mesmo título ganha outro slug.
	var outra trilha
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Home lab: rede"}, &outra)
	if outra.Slug != "home-lab-rede-2" {
		t.Fatalf("slug repetido: %q", outra.Slug)
	}

	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "Fundamentos"}, &tr)
	if len(tr.Modulos) != 1 {
		t.Fatalf("módulo não apareceu: %+v", tr.Modulos)
	}
	licao := "# Camadas\n\nTexto."
	var e1, e2 etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "Modelo em camadas", "minutos": 40, "bloom": "entender", "licao": licao}, &e1)
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{
		"titulo": "Endereço IP e sub-rede", "minutos": 50, "requer": []int64{e1.ID},
		"materiais": []map[string]any{{"tipo": "web", "titulo": "RFC 1918", "referencia": "https://www.rfc-editor.org/rfc/rfc1918"}},
	}, &e2)
	if e1.Chave != "modelo-em-camadas" || !e1.TemLicao || e2.TemLicao {
		t.Fatalf("etapas: %+v / %+v", e1, e2)
	}

	var lida etapa
	c.ok("GET", "/api/etapas/1", nil, &lida)
	if lida.Licao == nil || *lida.Licao != licao {
		t.Fatalf("lição não voltou: %+v", lida.Licao)
	}
	var lida2 etapa
	c.ok("GET", "/api/etapas/2", nil, &lida2)
	if len(lida2.Requer) != 1 || lida2.Requer[0] != e1.ID || len(lida2.Materiais) != 1 {
		t.Fatalf("requisito ou material sumiu: %+v", lida2)
	}

	// A lição e a estrutura ficam no conteúdo.
	if _, err := os.Stat(filepath.Join(c.a.conteudo.raiz, "trilhas", "home-lab-rede", "licoes", "modelo-em-camadas.md")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(c.a.conteudo.raiz, "trilhas", "home-lab-rede", "trilha.json"))
	if err != nil || !strings.Contains(string(b), `"endereco-ip-e-sub-rede"`) {
		t.Fatalf("trilha.json: %v %s", err, b)
	}

	// Lição vazia apaga o arquivo; sem "licao" no corpo, não mexe.
	c.ok("PUT", "/api/etapas/1", map[string]any{"titulo": "Modelo em camadas", "bloom": "entender"}, &lida)
	if !lida.TemLicao {
		t.Fatal("PUT sem lição não deveria apagar a lição")
	}
	c.ok("PUT", "/api/etapas/1", map[string]any{"titulo": "Modelo em camadas", "licao": ""}, &lida)
	if lida.TemLicao {
		t.Fatal("lição vazia deveria apagar o arquivo")
	}
}

func TestSlugificar(t *testing.T) {
	casos := map[string]string{
		"Home lab: rede":               "home-lab-rede",
		"História do Brasil — Império": "historia-do-brasil-imperio",
		"Teste de fumaça":              "teste-de-fumaca",
		"  ¿¿  ":                       "sem-titulo",
	}
	for entrada, quer := range casos {
		if got := slugificar(entrada); got != quer {
			t.Errorf("slugificar(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}

func TestValidacoes(t *testing.T) {
	c := novoCliente(t)
	if s := c.chamar("POST", "/api/trilhas", map[string]any{"titulo": " "}, nil); s != 400 {
		t.Fatalf("título vazio: %d", s)
	}
	if s := c.chamar("POST", "/api/trilhas", map[string]any{"titulo": "X", "profundidade": 9}, nil); s != 400 {
		t.Fatalf("profundidade 9: %d", s)
	}
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "X"}, nil)
	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "M"}, nil)
	if s := c.chamar("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "E", "bloom": "decorar"}, nil); s != 400 {
		t.Fatalf("bloom inválido: %d", s)
	}
	if s := c.chamar("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "E",
		"materiais": []map[string]any{{"tipo": "web", "titulo": "x", "referencia": "javascript:alert(1)"}}}, nil); s != 400 {
		t.Fatalf("material web sem http: %d", s)
	}
	var e etapa
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "E"}, &e)
	if s := c.chamar("PUT", "/api/etapas/1", map[string]any{"titulo": "E", "requer": []int64{1}}, nil); s != 400 {
		t.Fatalf("requisito de si mesma: %d", s)
	}
	if s := c.chamar("GET", "/api/trilhas/1/documentos/..%2F..%2Fetc", nil, nil); s != 400 {
		t.Fatalf("nome de documento com barra: %d", s)
	}
	if s := c.chamar("DELETE", "/api/modulos/1", nil, nil); s != 400 {
		t.Fatalf("módulo com etapas não pode sair: %d", s)
	}
}

func TestOrdemEStatus(t *testing.T) {
	c := novoCliente(t)
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "T"}, nil)
	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "A"}, nil)
	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "B"}, nil)
	for _, titulo := range []string{"um", "dois", "tres"} {
		c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": titulo}, nil)
	}
	// "um" vai para o módulo B; B passa à frente de A.
	var tr trilha
	c.ok("PUT", "/api/trilhas/1/ordem", map[string]any{"modulos": []map[string]any{
		{"id": 2, "etapas": []int64{1}}, {"id": 1, "etapas": []int64{3, 2}},
	}}, &tr)
	if tr.Modulos[0].Titulo != "B" || tr.Modulos[0].Etapas[0].Titulo != "um" || tr.Modulos[1].Etapas[0].Titulo != "tres" {
		t.Fatalf("ordem: %+v", tr.Modulos)
	}
	// Ordem incompleta é recusada.
	if s := c.chamar("PUT", "/api/trilhas/1/ordem", map[string]any{"modulos": []map[string]any{{"id": 1, "etapas": []int64{2}}}}, nil); s != 400 {
		t.Fatalf("ordem incompleta: %d", s)
	}

	var e etapa
	c.ok("POST", "/api/etapas/2/status", map[string]string{"status": "concluida"}, &e)
	if e.Status != "concluida" || e.ConcluidoEm == "" || e.IniciadoEm == "" {
		t.Fatalf("status: %+v", e)
	}
	c.ok("POST", "/api/etapas/2/status", map[string]string{"status": "disponivel"}, &e)
	if e.ConcluidoEm != "" || e.IniciadoEm != "" {
		t.Fatalf("voltar a disponível limpa as datas: %+v", e)
	}
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	if tr.Progresso.Total != 3 {
		t.Fatalf("progresso: %+v", tr.Progresso)
	}
}

func TestDistribuir(t *testing.T) {
	carregarFuso("America/Sao_Paulo")
	segunda, _ := lerData("2026-11-16") // segunda-feira
	etapas := []etapa{
		{ID: 1, Minutos: 30, Status: "concluida"},
		{ID: 2, Minutos: 50},
		{ID: 3, Minutos: 20},
		{ID: 4, Minutos: 0}, // sem estimativa: minutosPadrao
	}
	// Segunda, quarta e sexta, 60 min por dia.
	f := distribuir(etapas, []int{1, 3, 5}, 60, segunda)
	// 50 + 10 enchem segunda; os 10 que sobram da etapa 3 e os 45 da etapa 4 cabem na quarta.
	quer := []fatia{
		{2, "2026-11-16", 50}, {3, "2026-11-16", 10}, {3, "2026-11-18", 10}, {4, "2026-11-18", 45},
	}
	if len(f) != len(quer) {
		t.Fatalf("fatias: %+v", f)
	}
	for i := range quer {
		if f[i] != quer[i] {
			t.Fatalf("fatia %d: %+v, queria %+v", i, f[i], quer[i])
		}
	}
	// Começando num sábado, pula para segunda.
	sabado, _ := lerData("2026-11-14")
	if f := distribuir(etapas[1:2], []int{1}, 60, sabado); f[0].Data != "2026-11-16" {
		t.Fatalf("começo: %+v", f)
	}
}

func TestCapacidadePelaDataFinal(t *testing.T) {
	carregarFuso("America/Sao_Paulo")
	inicio, _ := lerData("2026-11-16")
	// Seg a sex até 27/11: 10 dias; 430 min → 43/dia → 45.
	n, err := capacidadeDiaria(meta{Dias: []int{1, 2, 3, 4, 5}, Fim: "2026-11-27"}, inicio, 430)
	if err != nil || n != 45 {
		t.Fatalf("capacidade: %d %v", n, err)
	}
	if _, err := capacidadeDiaria(meta{Dias: []int{0}, Fim: "2026-11-20"}, inicio, 100); err == nil {
		t.Fatal("sem domingo até a data final deveria dar erro")
	}
}

func TestMetaCronogramaEHoje(t *testing.T) {
	carregarFuso("America/Sao_Paulo")
	agora := time.Date(2026, 11, 16, 12, 0, 0, 0, fuso) // segunda
	relogio = func() time.Time { return agora }
	t.Cleanup(func() { relogio = time.Now })

	c := novoCliente(t)
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "T"}, nil)
	c.ok("POST", "/api/trilhas/1/modulos", map[string]any{"titulo": "M"}, nil)
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "a", "minutos": 60}, nil)
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "b", "minutos": 60}, nil)
	c.ok("POST", "/api/modulos/1/etapas", map[string]any{"titulo": "c", "minutos": 30}, nil)

	if s := c.chamar("PUT", "/api/trilhas/1/meta", map[string]any{"dias": []int{}, "minutos": 60}, nil); s != 400 {
		t.Fatalf("meta sem dias: %d", s)
	}
	var tr trilha
	c.ok("PUT", "/api/trilhas/1/meta", map[string]any{"dias": []int{1, 2, 3, 4, 5}, "minutos": 60}, &tr)
	if tr.Estado != "ativa" || tr.Meta.Inicio != "2026-11-16" {
		t.Fatalf("meta: %+v", tr)
	}

	type respHoje struct {
		Data    string       `json:"data"`
		Trilhas []trilhaHoje `json:"trilhas"`
	}
	var h respHoje
	c.ok("GET", "/api/hoje", nil, &h)
	if len(h.Trilhas) != 1 || len(h.Trilhas[0].Hoje) != 1 || h.Trilhas[0].Hoje[0].Titulo != "a" || h.Trilhas[0].Previsao != "2026-11-18" {
		t.Fatalf("hoje: %+v", h)
	}

	// Dois dias depois, sem estudar: "a" e "b" atrasadas; hoje é "c".
	agora = time.Date(2026, 11, 18, 9, 0, 0, 0, fuso)
	c.ok("GET", "/api/hoje", nil, &h)
	th := h.Trilhas[0]
	if len(th.Atrasadas) != 2 || len(th.Hoje) != 1 || th.Hoje[0].Titulo != "c" {
		t.Fatalf("atrasadas: %+v", th)
	}
	// Recalcular a partir de hoje: nada atrasado, "a" hoje.
	c.ok("POST", "/api/trilhas/1/recalcular", nil, nil)
	c.ok("GET", "/api/hoje", nil, &h)
	th = h.Trilhas[0]
	if len(th.Atrasadas) != 0 || th.Hoje[0].Titulo != "a" || th.Previsao != "2026-11-20" {
		t.Fatalf("recalculado: %+v", th)
	}
	// Fez "a" hoje: o próximo dia aparece só quando hoje não tem mais nada pendente.
	c.ok("POST", "/api/etapas/1/status", map[string]string{"status": "concluida"}, nil)
	c.ok("GET", "/api/hoje", nil, &h)
	if th = h.Trilhas[0]; th.Proximo == nil || th.Proximo.Data != "2026-11-19" || th.Proximo.Itens[0].Titulo != "b" {
		t.Fatalf("próximo: %+v", th.Proximo)
	}
}

func pacoteExemplo(substituir bool, etapas ...map[string]any) map[string]any {
	return map[string]any{
		"substituir": substituir,
		"documentos": []map[string]any{{"slug": "01-openwrt", "conteudo": "# OpenWrt\n\nTexto."}},
		"trilha": map[string]any{
			"slug": "home-lab-rede", "titulo": "Home lab: rede", "profundidade": 3,
			"modulos": []map[string]any{{"titulo": "Fundamentos", "etapas": etapas}},
		},
	}
}

func TestImportar(t *testing.T) {
	c := novoCliente(t)
	var res resultadoImportacao
	c.ok("POST", "/api/importar", pacoteExemplo(false,
		map[string]any{"chave": "camadas", "titulo": "Camadas", "minutos": 30, "licao": "# Camadas"},
		map[string]any{"chave": "ip", "titulo": "IP", "requer": []string{"camadas"},
			"materiais": []map[string]any{{"tipo": "documento", "titulo": "OpenWrt", "referencia": "01-openwrt", "papel": "principal"}}},
	), &res)
	if res.Etapas != 2 || res.Documentos != 1 || res.TrilhaID != 1 {
		t.Fatalf("importação: %+v", res)
	}
	var docs []documento
	c.ok("GET", "/api/trilhas/1/documentos", nil, &docs)
	if len(docs) != 1 || docs[0].Titulo != "OpenWrt" {
		t.Fatalf("documentos: %+v", docs)
	}

	// De novo sem "substituir": conflito.
	if s := c.chamar("POST", "/api/importar", pacoteExemplo(false, map[string]any{"chave": "x", "titulo": "X"}), nil); s != 409 {
		t.Fatalf("reimportar sem substituir: %d", s)
	}
	// Requisito que vem depois é recusado.
	if s := c.chamar("POST", "/api/importar", pacoteExemplo(true,
		map[string]any{"chave": "ip", "titulo": "IP", "requer": []string{"camadas"}},
		map[string]any{"chave": "camadas", "titulo": "Camadas"},
	), nil); s != 400 {
		t.Fatalf("requisito depois: %d", s)
	}

	// Nova versão: "camadas" concluída fica concluída; "ip" sai; "dhcp" entra.
	c.ok("POST", "/api/etapas/1/status", map[string]string{"status": "concluida"}, nil)
	c.ok("POST", "/api/importar", pacoteExemplo(true,
		map[string]any{"chave": "camadas", "titulo": "Camadas (revisto)"},
		map[string]any{"chave": "dhcp", "titulo": "DHCP", "requer": []string{"camadas"}},
	), &res)
	if res.Mantidas != 1 || res.Etapas != 2 {
		t.Fatalf("nova versão: %+v", res)
	}
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	es := tr.Modulos[0].Etapas
	if es[0].Status != "concluida" || es[0].Titulo != "Camadas (revisto)" || es[1].Chave != "dhcp" || es[0].TemLicao {
		t.Fatalf("depois da nova versão: %+v", es)
	}
}

func TestDocumentosDaTrilha(t *testing.T) {
	c := novoCliente(t)
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Rede"}, nil)
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "AWS"}, nil)
	var d documento
	c.ok("PUT", "/api/trilhas/1/documentos/02-wifi", map[string]any{"conteudo": "# Wi-Fi\n\nCanais."}, &d)
	if d.Titulo != "Wi-Fi" {
		t.Fatalf("documento salvo: %+v", d)
	}
	var docs []documento
	c.ok("GET", "/api/trilhas/1/documentos", nil, &docs)
	if len(docs) != 1 || docs[0].Slug != "02-wifi" || docs[0].Conteudo != "" {
		t.Fatalf("lista da trilha 1: %+v", docs)
	}
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	if len(tr.Documentos) != 1 || tr.Documentos[0].Titulo != "Wi-Fi" {
		t.Fatalf("detalhe da trilha sem o documento: %+v", tr.Documentos)
	}
	// Cada trilha vê só os seus.
	c.ok("GET", "/api/trilhas/2/documentos", nil, &docs)
	if len(docs) != 0 {
		t.Fatalf("a trilha 2 não tem documentos: %+v", docs)
	}
	if s := c.chamar("GET", "/api/trilhas/2/documentos/02-wifi", nil, nil); s != 404 {
		t.Fatalf("documento de outra trilha: %d", s)
	}
	if s := c.chamar("GET", "/api/trilhas/9/documentos", nil, nil); s != 404 {
		t.Fatalf("trilha que não existe: %d", s)
	}
	if s := c.chamar("PUT", "/api/trilhas/1/documentos/x", map[string]any{"conteudo": "  "}, nil); s != 400 {
		t.Fatalf("documento vazio: %d", s)
	}
	// Excluir a trilha leva os documentos junto.
	if s := c.chamar("DELETE", "/api/trilhas/1", nil, nil); s != 204 {
		t.Fatalf("excluir trilha: %d", s)
	}
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Rede"}, nil)
	c.ok("GET", "/api/trilhas/3/documentos", nil, &docs)
	if len(docs) != 0 {
		t.Fatalf("documento sobrou da trilha excluída: %+v", docs)
	}
}
