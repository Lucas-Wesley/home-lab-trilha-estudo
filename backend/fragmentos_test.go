package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// clienteFragmentos: o app de teste com o yt-dlp e o Claude falsos.
func clienteFragmentos(t *testing.T, f *iaFalsa) *cliente {
	t.Helper()
	c := clienteEntrada(t)
	c.a.ia.chamar = f.chamar
	return c
}

// fontePronta manda o vídeo falso para a Entrada e espera a leitura (transcrição de 1 minuto).
func fontePronta(t *testing.T, c *cliente) fonte {
	t.Helper()
	var fontes []fonte
	c.chamar("POST", "/api/entrada", map[string]string{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA"}, &fontes)
	c.a.esvaziarEntrada(context.Background())
	var f fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f)
	if f.Estado != "pronta" {
		t.Fatalf("a fonte devia estar pronta: %+v", f)
	}
	return f
}

// tresFragmentos: a resposta de teste da IA — 3 trechos dentro dos 60 s do vídeo falso.
const tresFragmentos = `{"fragmentos":[
{"inicio":"00:00","fim":"00:20","titulo":"Começo","resumo":"Resumo do começo.","assuntos":["redes"],"tipo":"conteudo"},
{"inicio":"00:20","fim":"00:40","titulo":"Opinião do autor","resumo":"Resumo da opinião.","assuntos":["opinião"],"tipo":"opiniao"},
{"inicio":"00:40","fim":"00:59","titulo":"Fim","resumo":"Resumo do fim.","assuntos":["dns"],"tipo":"conteudo"}],
"notas":"Divisão de teste."}`

func TestFragmentar(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, _ map[string]string) (string, error) { return tresFragmentos, nil }
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)

	var pedida fonte
	if s := c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID),
		map[string]string{"pedido": "ignore a propaganda"}, &pedida); s != http.StatusAccepted {
		t.Fatalf("fragmentar: %d %+v", s, pedida)
	}
	if pedida.Estado != "fragmentando" || pedida.TarefaID == 0 || pedida.Pedido != "ignore a propaganda" ||
		pedida.Tarefa == nil || pedida.Tarefa.Tipo != tipoFragmentar {
		t.Fatalf("enfileirando: %+v", pedida)
	}
	// Com a tarefa na fila não dá para fragmentar de novo.
	if s := c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil); s != http.StatusBadRequest {
		t.Fatalf("fragmentar fonte fragmentando: %d", s)
	}
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if f2.Estado != "fragmentada" || f2.Notas != "Divisão de teste." || f2.Erro != "" {
		t.Fatalf("depois da tarefa: %+v", f2)
	}
	if len(f2.Fragmentos) != 3 {
		t.Fatalf("3 fragmentos, vieram %d: %+v", len(f2.Fragmentos), f2.Fragmentos)
	}
	for i, fr := range f2.Fragmentos {
		if fr.Estado != "sugerido" || fr.FonteID != inicial.ID || fr.Ordem != i {
			t.Fatalf("fragmento %d: %+v", i, fr)
		}
	}
	fim := f2.Fragmentos[2]
	if fim.Titulo != "Fim" || fim.Inicio != 40 || fim.Fim != 59 || fim.Tipo != "conteudo" ||
		len(fim.Assuntos) != 1 || fim.Assuntos[0] != "dns" {
		t.Fatalf("último fragmento: %+v", fim)
	}
	if f2.Tarefa == nil || f2.Tarefa.Estado != "concluida" || f2.Tarefa.Titulo != "Fragmentar: Vídeo AAAAAAAAAAA" {
		t.Fatalf("a tarefa no GET: %+v", f2.Tarefa)
	}

	// O pedido que a IA recebeu: cabeçalho, dados do vídeo, capítulos, o pedido do Lucas,
	// e a transcrição num arquivo à parte.
	pedido := f.arquivos[0]["pedido.md"]
	for _, trecho := range []string{
		"# Pedido: fragmentar", `Vídeo: "Vídeo AAAAAAAAAAA"`, "Canal: Canal de Teste",
		"Duração: 01:00", "00:00 Introdução", "00:30 O assunto", "## Pedido do Lucas\n\nignore a propaganda",
	} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md devia ter %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.arquivos[0]["transcricao.md"], "[00:00] Olá") {
		t.Errorf("transcricao.md:\n%s", f.arquivos[0]["transcricao.md"])
	}
	if f.pedidos[0].ferramentas != "Read" {
		t.Errorf("só Read: %q", f.pedidos[0].ferramentas)
	}
}

func TestFragmentarLimpeza(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, _ map[string]string) (string, error) {
		return `{"fragmentos":[
{"inicio":"00:00","fim":"00:30","titulo":"Começo","resumo":"R","assuntos":[],"tipo":"conteudo"},
{"inicio":"00:10","fim":"00:45","titulo":"Sobreposto","resumo":"R","assuntos":[],"tipo":"conteudo"},
{"inicio":"00:40","fim":"00:35","titulo":"Invertido","resumo":"R","assuntos":[],"tipo":"conteudo"},
{"inicio":"02:00","fim":"02:30","titulo":"Fora do vídeo","resumo":"R","assuntos":[],"tipo":"conteudo"},
{"inicio":"00:50","fim":"00:55","titulo":"","resumo":"R","assuntos":[],"tipo":"conteudo"},
{"inicio":"00:45","fim":"01:10","titulo":"Fim passando","resumo":"R","assuntos":[],"tipo":"mentira"}],
"notas":"Teste da limpeza."}`, nil
	}
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if f2.Estado != "fragmentada" {
		t.Fatalf("estado: %+v", f2)
	}
	// "Fora do vídeo", "Invertido" e o sem título saem; a sobreposição fecha "Começo" no início
	// de "Sobreposto"; o fim que passa do vídeo fecha na duração; o tipo inventado vira "conteudo".
	if len(f2.Fragmentos) != 3 {
		t.Fatalf("deviam sobrar 3: %+v", f2.Fragmentos)
	}
	if f2.Fragmentos[0].Inicio != 0 || f2.Fragmentos[0].Fim != 10 {
		t.Fatalf("o fim do primeiro vira o início do segundo: %+v", f2.Fragmentos[0])
	}
	if f2.Fragmentos[1].Inicio != 10 || f2.Fragmentos[1].Fim != 45 {
		t.Fatalf("segundo: %+v", f2.Fragmentos[1])
	}
	if f2.Fragmentos[2].Inicio != 45 || f2.Fragmentos[2].Fim != 60 || f2.Fragmentos[2].Tipo != "conteudo" {
		t.Fatalf("terceiro (fim cortado na duração, tipo saneado): %+v", f2.Fragmentos[2])
	}
	for _, trecho := range []string{"00:40–00:35", "02:00–02:30", "00:50–00:55", "O app tirou"} {
		if !strings.Contains(f2.Notas, trecho) {
			t.Errorf("as notas deviam contar o que saiu (%q): %q", trecho, f2.Notas)
		}
	}
}

func TestFragmentarDeNovoMantemConfirmados(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, _ map[string]string) (string, error) {
		if n == 1 {
			return tresFragmentos, nil
		}
		return `{"fragmentos":[
{"inicio":"00:59","fim":"01:00","titulo":"Novo trecho","resumo":"R","assuntos":["novo"],"tipo":"conteudo"}],
"notas":"De novo."}`, nil
	}
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
	c.a.fila.esvaziar(context.Background())

	var lista []fragmento
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/confirmar", inicial.ID), nil, &lista)
	if len(lista) != 3 || lista[0].Estado != "confirmado" {
		t.Fatalf("confirmar: %+v", lista)
	}

	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID),
		map[string]string{"pedido": "de novo"}, nil)
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if f2.Estado != "fragmentada" || len(f2.Fragmentos) != 4 {
		t.Fatalf("3 confirmados + 1 novo: %+v", f2.Fragmentos)
	}
	var confirmados, sugeridos int
	for _, fr := range f2.Fragmentos {
		if fr.Estado == "confirmado" {
			confirmados++
		} else if fr.Estado == "sugerido" && fr.Titulo == "Novo trecho" {
			sugeridos++
		}
	}
	if confirmados != 3 || sugeridos != 1 {
		t.Fatalf("confirmados %d, sugeridos %d: %+v", confirmados, sugeridos, f2.Fragmentos)
	}
	// A segunda vez, o pedido lista os confirmados para a IA não repetir.
	pedido := f.arquivos[1]["pedido.md"]
	for _, trecho := range []string{"Fragmentos já confirmados", "00:00–00:20 — Começo", "00:40–00:59 — Fim"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md da segunda vez devia ter %q:\n%s", trecho, pedido)
		}
	}
}

func TestFragmentoEditar(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, _ map[string]string) (string, error) { return tresFragmentos, nil }
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	fr := f2.Fragmentos[0]

	var editado fragmento
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d", fr.ID), map[string]any{
		"inicio": "00:02", "fim": "00:19", "titulo": "Começo editado", "resumo": "Outro resumo.",
		"assuntos": []string{"redes", "tcp"}, "tipo": "opiniao",
	}, &editado)
	if editado.Inicio != 2 || editado.Fim != 19 || editado.Titulo != "Começo editado" ||
		editado.Tipo != "opiniao" || len(editado.Assuntos) != 2 || editado.Estado != "sugerido" {
		t.Fatalf("editado: %+v", editado)
	}
	// Validações: tempo ilegível, fim antes do início, além do vídeo e cruzando o seguinte.
	for _, corpo := range []map[string]any{
		{"inicio": "abc"},
		{"inicio": "00:10", "fim": "00:05"},
		{"fim": "02:00"},
		{"fim": "00:30"},
		{"tipo": "inventado"},
		{"titulo": "   "},
	} {
		if s := c.chamar("PUT", fmt.Sprintf("/api/fragmentos/%d", fr.ID), corpo, nil); s != http.StatusBadRequest {
			t.Fatalf("PUT %v: %d", corpo, s)
		}
	}
}

func TestFragmentoDividirJuntarDescartar(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, _ map[string]string) (string, error) { return tresFragmentos, nil }
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	primeiroID := f2.Fragmentos[0].ID

	var metade fragmento
	c.ok("POST", fmt.Sprintf("/api/fragmentos/%d/dividir", primeiroID), map[string]string{"em": "00:10"}, &metade)
	if metade.Inicio != 10 || metade.Fim != 20 || metade.Titulo != "Começo (continuação)" {
		t.Fatalf("a metade nova: %+v", metade)
	}
	// Fora do trecho não divide.
	if s := c.chamar("POST", fmt.Sprintf("/api/fragmentos/%d/dividir", metade.ID), map[string]string{"em": "00:50"}, nil); s != http.StatusBadRequest {
		t.Fatalf("dividir fora: %d", s)
	}
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if len(f2.Fragmentos) != 4 || f2.Fragmentos[0].Fim != 10 || f2.Fragmentos[1].Inicio != 10 {
		t.Fatalf("depois de dividir: %+v", f2.Fragmentos)
	}

	// Juntar: volta a ser um (início do primeiro, fim do segundo, títulos com " + ").
	var junto fragmento
	c.ok("POST", fmt.Sprintf("/api/fragmentos/%d/juntar", primeiroID), nil, &junto)
	if junto.Inicio != 0 || junto.Fim != 20 || junto.Titulo != "Começo + Começo (continuação)" {
		t.Fatalf("junto: %+v", junto)
	}
	// O último não tem seguinte.
	ultimo := f2.Fragmentos[len(f2.Fragmentos)-1]
	if s := c.chamar("POST", fmt.Sprintf("/api/fragmentos/%d/juntar", ultimo.ID), nil, nil); s != http.StatusBadRequest {
		t.Fatalf("juntar sem seguinte: %d", s)
	}

	// Descartar sai da lista; "mostrar descartados" o traz de volta (o GET tem todos).
	var morto fragmento
	c.ok("POST", fmt.Sprintf("/api/fragmentos/%d/descartar", junto.ID), nil, &morto)
	if morto.Estado != "descartado" {
		t.Fatalf("descartado: %+v", morto)
	}
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if len(f2.Fragmentos) != 3 || f2.Fragmentos[0].Estado != "descartado" {
		t.Fatalf("com o descartado: %+v", f2.Fragmentos)
	}

	// Confirmar: todos os sugeridos da fonte viram confirmados.
	var confirmados []fragmento
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/confirmar", inicial.ID), nil, &confirmados)
	for _, fr := range confirmados {
		if fr.Estado == "descartado" {
			continue
		}
		if fr.Estado != "confirmado" {
			t.Fatalf("ficou %s: %+v", fr.Estado, fr)
		}
	}
}

func TestFragmentarTarefaFalha(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(_ int, _ map[string]string) (string, error) {
		return "", fmt.Errorf("o Claude de teste falhou")
	}
	c := clienteFragmentos(t, f)
	inicial := fontePronta(t, c)
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if f2.Estado != "pronta" || !strings.Contains(f2.Erro, "Claude de teste falhou") {
		t.Fatalf("a fonte devia voltar a pronta com o erro: %+v", f2)
	}
	if f2.Tarefa == nil || f2.Tarefa.Estado != "falhou" || !strings.Contains(f2.Tarefa.Erro, "Claude de teste falhou") {
		t.Fatalf("a tarefa: %+v", f2.Tarefa)
	}
	// Dá para tentar de novo.
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, nil)
}

func TestModeloEntrada(t *testing.T) {
	c := novoCliente(t)
	c.ok("PUT", "/api/ajustes", map[string]any{"modelo_entrada": "haiku"}, nil)
	if m := c.a.modeloPara(context.Background(), tipoFragmentar); m != "haiku" {
		t.Fatalf("modeloPara(fragmentar): %q", m)
	}
	c.ok("PUT", "/api/ajustes", map[string]any{"modelo_entrada": ""}, nil)
	if m := c.a.modeloPara(context.Background(), tipoFragmentar); m != "" {
		t.Fatalf("vazio = o padrão: %q", m)
	}
}

func TestRegraFragmentarConhecida(t *testing.T) {
	if !regraConhecida("fragmentar.md") {
		t.Fatal("fragmentar.md devia estar em regrasDoApp")
	}
	c := novoCliente(t)
	rg, err := c.a.ia.lerRegra("fragmentar.md")
	if err != nil || !strings.Contains(rg.Texto, "fragmentos") {
		t.Fatalf("lerRegra: %v", err)
	}
}
