package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type estudoDaEtapa struct {
	Exercicios []struct {
		Tipo         string   `json:"tipo"`
		Enunciado    string   `json:"enunciado"`
		Alternativas []string `json:"alternativas"`
		Correta      *int     `json:"correta"`
		Aceitas      []string `json:"aceitas"`
	} `json:"exercicios"`
	Simulado []struct {
		Pergunta string `json:"pergunta"`
		Esperado string `json:"esperado"`
	} `json:"simulado"`
	Tentativas       []tentativa `json:"tentativas"`
	NotaMinima       int         `json:"nota_minima"`
	CriterioAtingido bool        `json:"criterio_atingido"`
	Flashcards       struct {
		Disponivel bool              `json:"disponivel"`
		Envios     []envioFlashcards `json:"envios"`
	} `json:"flashcards"`
}

// gravarLicao: a lição da etapa NAT escrita à mão (os exercícios, o simulado e os cartões saem dela).
func gravarLicao(c *cliente, etapaID int64) {
	c.ok("PUT", fmt.Sprintf("/api/etapas/%d", etapaID), map[string]any{"titulo": "NAT", "objetivo": "Explicar o NAT.", "bloom": "entender",
		"licao": "# NAT\n\n## Por que existe\n\nFaltam endereços IPv4.\n\n## Como funciona\n\nTroca o endereço de origem."}, nil)
}

func TestExerciciosPropostaETentativa(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		return `{"exercicios":[
{"tipo":"escolha","enunciado":"Por que o NAT existe?","alternativas":["Faltam endereços IPv4","Segurança","Velocidade"],"correta":0,"explicacao":"Porque faltam.","conceito":"motivo"},
{"tipo":"completar","enunciado":"O NAT troca o endereço de ____.","aceitas":["origem","source"],"explicacao":"De origem.","conceito":"troca"},
{"tipo":"escolha","enunciado":"Mal formado","alternativas":["a","b"],"correta":5,"explicacao":"x","conceito":"x"},
{"tipo":"completar","enunciado":"Sem aceitas ____","aceitas":[" "],"explicacao":"x","conceito":"x"}],
"notas":"Cobri o motivo e a troca.","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	c.ok("PUT", "/api/ajustes", map[string]any{"modelo_estudo": "haiku", "nota_minima": "50%"}, nil)
	nat := trilhaComEtapa(c)

	// Sem lição, não há de onde tirar os exercícios.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/exercicios", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("exercícios sem lição: %d", s)
	}
	gravarLicao(c, nat)
	var p proposta
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/exercicios", nat), map[string]any{"pedido": "Foque no motivo."}, &p); s != 201 {
		t.Fatalf("pedir exercícios: %d", s)
	}
	if p.Tipo != tipoExercicios || p.Tarefa == nil || p.Tarefa.Modelo != "haiku" {
		t.Fatalf("proposta: %+v %+v", p, p.Tarefa)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	if !strings.Contains(pedido, "# Pedido: exercícios da etapa \"NAT\"") || !strings.Contains(pedido, "Deste pedido: Foque no motivo.") ||
		!strings.Contains(f.arquivos[0]["licao.md"], "Faltam endereços IPv4") || !strings.Contains(f.pedidos[0].regra, "os exercícios de uma etapa") {
		t.Fatalf("pedido:\n%s", pedido)
	}
	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=exercicios", nat), nil, &lista)
	var ce conteudoExercicios
	json.Unmarshal(lista[0].Conteudo, &ce)
	if lista[0].Estado != propAguardando || len(ce.Exercicios) != 2 || !strings.Contains(ce.Notas, "tirou 2 exercício") {
		t.Fatalf("proposta gerada: %s %+v", lista[0].Estado, ce)
	}
	// A lista de lições não mistura os exercícios.
	var licoes []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas", nat), nil, &licoes)
	if len(licoes) != 0 {
		t.Fatalf("propostas de lição: %+v", licoes)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), nil, &p)

	// A tela recebe os exercícios sem o gabarito.
	var est estudoDaEtapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if len(est.Exercicios) != 2 || est.Exercicios[0].Correta != nil || est.Exercicios[1].Aceitas != nil || est.NotaMinima != 50 || est.CriterioAtingido {
		t.Fatalf("estudo: %+v", est)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/exercicios/tentativas", nat), map[string]any{"respostas": []any{map[string]any{"escolha": 0}}}, nil); s != 400 {
		t.Fatalf("respostas a menos: %d", s)
	}
	var tt tentativa
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/exercicios/tentativas", nat), map[string]any{"respostas": []any{
		map[string]any{"escolha": 1}, map[string]any{"texto": "  ORIGEM. "}}}, &tt)
	var corr []correcaoExercicio
	json.Unmarshal(tt.Resultado, &corr)
	if tt.Nota != 50 || tt.Acertos != 1 || tt.Total != 2 || corr[0].Certa || corr[0].Correta != "Faltam endereços IPv4" || corr[0].Resposta != "Segurança" || !corr[1].Certa {
		t.Fatalf("tentativa: %+v %+v", tt, corr)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if !est.CriterioAtingido || len(est.Tentativas) != 1 {
		t.Fatalf("critério com 50%% e mínima 50%%: %+v", est)
	}
	var e etapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d", nat), nil, &e)
	if e.Status != "em_andamento" {
		t.Fatalf("fazer os exercícios começa a etapa: %s", e.Status)
	}
	if s := c.chamar("PUT", "/api/ajustes", map[string]any{"nota_minima": "120"}, nil); s != 400 {
		t.Fatalf("nota mínima 120: %d", s)
	}
}

func TestSimuladoBancoCresceEIACorrige(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		switch {
		case strings.Contains(arq["pedido.md"], "# Pedido: correção do simulado"):
			if strings.Contains(arq["pedido.md"], "FALHE") {
				return "", errors.New("limite da assinatura")
			}
			return `{"correcoes":[{"pergunta":1,"nota":90,"veredito":"certo","comentario":"Bem explicado.","faltou":""},
{"pergunta":2,"nota":150,"veredito":"parcial","comentario":"Faltou o porquê.","faltou":"o motivo"}],"resumo":"Revise a seção 2."}`, nil
		case strings.Contains(arq["pedido.md"], "O banco do simulado já tem"):
			return `{"perguntas":[{"pergunta":"Por que o NAT existe?","esperado":"repetida","conceito":"x"},{"pergunta":"Compare NAT e CGNAT.","esperado":"- dois níveis (essencial)","conceito":"cgnat"}],"notas":"","respostas":[]}`, nil
		}
		return `{"perguntas":[{"pergunta":"Por que o NAT existe?","esperado":"- faltam IPv4 (essencial)","conceito":"motivo"},{"pergunta":"O que o NAT troca?","esperado":"- a origem","conceito":"troca"}],"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c)
	gravarLicao(c, nat)
	gerarEAprovar := func(pedido string) {
		var p proposta
		c.ok("POST", fmt.Sprintf("/api/etapas/%d/simulado", nat), map[string]any{"pedido": pedido}, &p)
		c.a.fila.esvaziar(context.Background())
		var lista []proposta
		c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=simulado", nat), nil, &lista)
		c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[len(lista)-1].ID), nil, nil)
	}
	gerarEAprovar("")
	gerarEAprovar("Mais sobre CGNAT.")
	if !strings.Contains(f.arquivos[1]["pedido.md"], "1. Por que o NAT existe?") {
		t.Fatalf("o segundo pedido não levou o banco:\n%s", f.arquivos[1]["pedido.md"])
	}
	var est estudoDaEtapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	// A repetida não entra; o gabarito não vai para a tela.
	if len(est.Simulado) != 3 || est.Simulado[2].Pergunta != "Compare NAT e CGNAT." || est.Simulado[0].Esperado != "" {
		t.Fatalf("banco: %+v", est.Simulado)
	}

	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/simulado/tentativas", nat), map[string]any{"respostas": []any{map[string]any{"pergunta": 0, "resposta": " "}}}, nil); s != 400 {
		t.Fatalf("sem resposta: %d", s)
	}
	var tt tentativa
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/simulado/tentativas", nat), map[string]any{"respostas": []any{
		map[string]any{"pergunta": 0, "resposta": "Porque faltam IPv4."}, map[string]any{"pergunta": 2, "resposta": "Um é dentro do outro."}}}, &tt)
	if tt.Estado != tentCorrigindo || tt.Total != 2 || tt.TarefaID == 0 {
		t.Fatalf("tentativa: %+v", tt)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/simulado/tentativas", nat), map[string]any{"respostas": []any{map[string]any{"pergunta": 0, "resposta": "x"}}}, nil); s != 400 {
		t.Fatalf("segunda correção ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[len(f.arquivos)-1]["pedido.md"]
	if !strings.Contains(pedido, "**Gabarito:** - dois níveis (essencial)") || !strings.Contains(pedido, "Um é dentro do outro.") {
		t.Fatalf("pedido da correção:\n%s", pedido)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	got := est.Tentativas[0]
	var res struct {
		Itens []struct {
			Nota     int    `json:"nota"`
			Pergunta string `json:"pergunta"`
		} `json:"itens"`
		Resumo string `json:"resumo"`
	}
	json.Unmarshal(got.Resultado, &res)
	if got.Estado != tentCorrigida || got.Nota != 95 || got.Acertos != 1 || res.Itens[1].Nota != 100 || res.Resumo != "Revise a seção 2." {
		t.Fatalf("correção: %+v %+v", got, res)
	}
	// O simulado não conta para o critério automático (que é dos exercícios).
	if est.CriterioAtingido {
		t.Fatalf("critério pelo simulado")
	}
	// Correção que falha: a tentativa fica com o erro.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/simulado/tentativas", nat), map[string]any{"respostas": []any{map[string]any{"pergunta": 1, "resposta": "FALHE"}}}, &tt)
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if est.Tentativas[0].Estado != tentFalhou || !strings.Contains(est.Tentativas[0].Erro, "limite") {
		t.Fatalf("falha: %+v", est.Tentativas[0])
	}
}

func TestEnviarLicaoAoFlashcards(t *testing.T) {
	var recebido map[string]any
	fc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/baralhos":
			w.Write([]byte(`[{"id":"b1","nome":"Rede"}]`))
		case "/api/textos":
			b, _ := io.ReadAll(r.Body)
			json.Unmarshal(b, &recebido)
			if recebido["baralho"] == "" && recebido["baralho_id"] == "" {
				w.WriteHeader(400)
				w.Write([]byte(`{"erro":"Escolha um baralho ou dê o nome do baralho novo."}`))
				return
			}
			w.WriteHeader(201)
			w.Write([]byte(`{"id":42}`))
		}
	}))
	defer fc.Close()
	c := novoCliente(t)
	c.a.integra = integracoes{flashcards: fc.URL, flashcardsLink: "https://flashcards.lan", trilhaLink: "https://trilha.lan"}
	nat := trilhaComEtapa(c)
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/flashcards", nat), map[string]any{"baralho": "Rede"}, nil); s != 400 {
		t.Fatalf("sem lição: %d", s)
	}
	gravarLicao(c, nat)
	var baralhos []map[string]string
	c.ok("GET", "/api/flashcards/baralhos", nil, &baralhos)
	if len(baralhos) != 1 || baralhos[0]["nome"] != "Rede" {
		t.Fatalf("baralhos: %+v", baralhos)
	}
	// O erro do Flashcards volta para a tela e não é anotado como envio.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/flashcards", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("sem baralho: %d", s)
	}
	var r struct {
		GeracaoID int64             `json:"geracao_id"`
		Envios    []envioFlashcards `json:"envios"`
	}
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/flashcards", nat), map[string]any{"baralho_id": "b1", "tipo": "leitura", "instrucao": " só o NAT "}, &r)
	if r.GeracaoID != 42 || len(r.Envios) != 1 {
		t.Fatalf("envio: %+v", r)
	}
	link := fmt.Sprintf("https://trilha.lan/#/trilhas/1/etapas/%d", nat)
	secoes, _ := recebido["secoes"].([]any)
	if recebido["origem"] != "trilha" || recebido["link"] != link || recebido["titulo"] != "Rede › NAT" || recebido["instrucao"] != "só o NAT" ||
		recebido["tipo"] != "leitura" || !strings.Contains(recebido["texto"].(string), "Faltam endereços IPv4") || len(secoes) != 2 ||
		secoes[0].(map[string]any)["link"] != link+"#por-que-existe" {
		t.Fatalf("o que foi ao Flashcards: %+v", recebido)
	}
	var est estudoDaEtapa
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if !est.Flashcards.Disponivel || len(est.Flashcards.Envios) != 1 || est.Flashcards.Envios[0].GeracaoID != 42 {
		t.Fatalf("envios: %+v", est.Flashcards)
	}
}

func TestSecoesDaLicaoComoOLeitor(t *testing.T) {
	s := secoesDaLicao("# T\n\n## 2.1 Reserva de IP: IP fora × static\n\n## Como funciona\n\n## Como funciona\n", "L")
	if len(s) != 3 || s[0]["link"] != "L#21-reserva-de-ip-ip-fora--static" || s[2]["link"] != "L#como-funciona-1" {
		t.Fatalf("seções: %+v", s)
	}
	if normalizarResposta("  Endereço  de ORIGEM. ") != "endereco de origem" || normalizarResposta("50%") != "50%" {
		t.Fatalf("normalizar")
	}
}
