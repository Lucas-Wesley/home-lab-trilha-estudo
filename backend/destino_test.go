package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// fonteFragmentada: a fonte pronta, fragmentada (3 trechos) e confirmada — o ponto de partida
// dos testes de destino.
func fonteFragmentada(t *testing.T, c *cliente) fonte {
	t.Helper()
	inicial := fontePronta(t, c)
	var pedida fonte
	c.chamar("POST", fmt.Sprintf("/api/entrada/%d/fragmentar", inicial.ID), nil, &pedida)
	c.a.fila.esvaziar(context.Background())
	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if f2.Estado != "fragmentada" || len(f2.Fragmentos) != 3 {
		t.Fatalf("fragmentada: %+v", f2)
	}
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/confirmar", inicial.ID), nil, nil)
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	return f2
}

// iaDaEntrada: a resposta falsa por tipo de tarefa, pelo cabeçalho do pedido.md.
func iaDaEntrada(f *iaFalsa, destino, encaixe string) {
	f.resposta = func(_ int, arq map[string]string) (string, error) {
		pedido := arq["pedido.md"]
		switch {
		case strings.Contains(pedido, "# Pedido: destino"):
			return destino, nil
		case strings.Contains(pedido, "# Pedido: encaixar"):
			return encaixe, nil
		default:
			return tresFragmentos, nil
		}
	}
}

// encaixeNovoLocalizador: a IA propõe um tópico novo num módulo novo, com o trecho dado.
func encaixeNovoLocalizador(localizador string) string {
	return fmt.Sprintf(`{"encaixes":[],"novos":[
{"titulo":"Novo tópico","objetivo":"Explicar o trecho.","bloom":"entender","minutos":30,
"modulo":"Vídeos","depois_de":"","localizador":%q,"nota":"Trecho.","motivo":"Falta."}],
"revisoes":[],"notas":"","respostas":[]}`, localizador)
}

func TestLerIntervalos(t *testing.T) {
	casos := []struct {
		localizador string
		inicio, fim int
	}{
		{"06:00–10:00", 360, 600},
		{"06:00-10:00", 360, 600}, // hífen comum também vale
		{"1:02:05–1:03:00", 3725, 3780},
		{" 06:00 – 10:00 ", 360, 600},
	}
	for _, x := range casos {
		xs, err := lerIntervalos(x.localizador)
		if err != nil || len(xs) != 1 || xs[0][0] != x.inicio || xs[0][1] != x.fim {
			t.Errorf("%q: %v %v", x.localizador, xs, err)
		}
	}
	xs, err := lerIntervalos("06:00–10:00; 25:10–31:00")
	if err != nil || len(xs) != 2 || xs[1] != [2]int{1510, 1860} {
		t.Fatalf("dois intervalos: %v %v", xs, err)
	}
	if got := escreverIntervalos(xs); got != "06:00–10:00; 25:10–31:00" {
		t.Fatalf("escreverIntervalos: %q", got)
	}
	for _, errado := range []string{"", "06:00", "10:00–06:00", "06:00–06:00", "abc", "06:00–10:00–12:00"} {
		if _, err := lerIntervalos(errado); err == nil {
			t.Errorf("%q devia falhar", errado)
		}
	}
}

func TestTrechoDaTranscricao(t *testing.T) {
	transcricao := "# Vídeo\n\n- Canal: Teste\n\n[00:00] Um.\n\n[00:10] Dois.\n\n[00:20] Três.\n\n[00:30] Quatro.\n\n"
	got := trechoDaTranscricao(transcricao, 10, 30)
	if !strings.Contains(got, "[00:10] Dois.") || !strings.Contains(got, "[00:20] Três.") {
		t.Fatalf("trecho de 10 a 30: %q", got)
	}
	if strings.Contains(got, "[00:00]") || strings.Contains(got, "[00:30]") || strings.Contains(got, "Canal") {
		t.Fatalf("fora do intervalo: %q", got)
	}
	if got := trechoDaTranscricao(transcricao, 0, 10); !strings.Contains(got, "[00:00] Um.") || len(strings.Split(strings.TrimSpace(got), "\n\n")) != 1 {
		t.Fatalf("só o primeiro bloco: %q", got)
	}
	if got := trechoDaTranscricao(transcricao, 50, 90); got != "" {
		t.Fatalf("intervalo fora do vídeo: %q", got)
	}
}

const destinoJSON = `{"destinos":[
{"fragmento":1,"trilhas":[{"trilha_id":1,"motivo":"É rede."},{"trilha_id":99,"motivo":"Inventada."}],"trilha_nova":"","motivo_nenhuma":""},
{"fragmento":2,"trilhas":[],"trilha_nova":"","motivo_nenhuma":"É opinião, não vira aula."},
{"fragmento":3,"trilhas":[{"trilha_id":2,"motivo":"É base."}],"trilha_nova":"Assunto novo","motivo_nenhuma":""},
{"fragmento":9,"trilhas":[],"trilha_nova":"","motivo_nenhuma":""}],
"notas":"Destino de teste."}`

func TestSugerirDestinos(t *testing.T) {
	f := &iaFalsa{}
	iaDaEntrada(f, destinoJSON, "{}")
	c := clienteFragmentos(t, f)
	inicial := fonteFragmentada(t, c)
	trilhaComEtapa(c) // trilha 1 "Rede"
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Conhecimento de base"}, nil)

	var pedida fonte
	if s := c.chamar("POST", fmt.Sprintf("/api/entrada/%d/destino", inicial.ID),
		map[string]string{"pedido": "dê atenção à Rede"}, &pedida); s != http.StatusAccepted {
		t.Fatalf("destino: %d %+v", s, pedida)
	}
	if pedida.TarefaDestinoID == 0 || pedida.PedidoDestino != "dê atenção à Rede" ||
		pedida.TarefaDestino == nil || pedida.TarefaDestino.Tipo != tipoDestino {
		t.Fatalf("enfileirando destino: %+v", pedida)
	}
	// Na fila não dá para pedir de novo.
	if s := c.chamar("POST", fmt.Sprintf("/api/entrada/%d/destino", inicial.ID), nil, nil); s != http.StatusBadRequest {
		t.Fatalf("destino de novo na fila: %d", s)
	}
	c.a.fila.esvaziar(context.Background())

	pedido := f.arquivos[1]["pedido.md"] // a 2ª tarefa (a 1ª é a de fragmentar)
	for _, trecho := range []string{"# Pedido: destino", `1. 00:00–00:20 — "Começo" (conteudo)`,
		`2. 00:20–00:40 — "Opinião do autor" (opiniao)`, "- Trilha 1 — \"Rede\"", "Módulo \"Internet\": IP e DHCP · NAT",
		"- Trilha 2 — \"Conhecimento de base\"", "dê atenção à Rede"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("o pedido devia ter %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.pedidos[1].regra, "trilhas de cada fragmento") {
		t.Fatalf("a regra devia ser a do destino: %s", f.pedidos[1].regra)
	}

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", inicial.ID), nil, &f2)
	if !strings.Contains(f2.Notas, "Destino de teste.") || !strings.Contains(f2.Notas, "a trilha 99") ||
		!strings.Contains(f2.Notas, "o fragmento 9") || f2.Erro != "" {
		t.Fatalf("notas do destino (o erro fica limpo): %q / %q", f2.Notas, f2.Erro)
	}
	fr1, fr2, fr3 := f2.Fragmentos[0], f2.Fragmentos[1], f2.Fragmentos[2]
	// A sugestão guarda o que a IA disse e já fica marcada (a trilha 99 sai nas notas).
	if fr1.Sugestao == nil || len(fr1.Sugestao.Trilhas) != 1 || fr1.Sugestao.Trilhas[0].TrilhaID != 1 ||
		fr1.Sugestao.Trilhas[0].Motivo != "É rede." {
		t.Fatalf("sugestão do 1: %+v", fr1.Sugestao)
	}
	if len(fr1.Destinos) != 1 || fr1.Destinos[0].TrilhaID != 1 || fr1.Destinos[0].Titulo != "Rede" || fr1.Destinos[0].PropostaID != 0 {
		t.Fatalf("destinos marcados do 1: %+v", fr1.Destinos)
	}
	if fr2.Sugestao == nil || fr2.Sugestao.MotivoNenhuma == "" || len(fr2.Destinos) != 0 {
		t.Fatalf("a opinião sem destino: %+v %+v", fr2.Sugestao, fr2.Destinos)
	}
	if fr3.Sugestao == nil || fr3.Sugestao.TrilhaNova != "Assunto novo" || len(fr3.Destinos) != 1 || fr3.Destinos[0].TrilhaID != 2 {
		t.Fatalf("sugestão do 3: %+v %+v", fr3.Sugestao, fr3.Destinos)
	}
}

func TestDestinosEEnviar(t *testing.T) {
	f := &iaFalsa{}
	// Nos dois encaixes o novo usa um trecho que está nos intervalos mandados (00:20–00:40
	// está nos dois: a trilha A tem {00:00–00:20; 00:20–00:40} e a B, {00:20–00:40}).
	iaDaEntrada(f, destinoJSON, encaixeNovoLocalizador("00:20–00:40"))
	c := clienteFragmentos(t, f)
	f2 := fonteFragmentada(t, c)
	trilhaComEtapa(c)                                                   // trilha 1 "Rede"
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Base"}, nil) // trilha 2

	fr1, fr2, fr3 := f2.Fragmentos[0], f2.Fragmentos[1], f2.Fragmentos[2]
	// fr1 → trilha A(1); fr2 → trilhas A(1) e B(2); fr3 fica sem destino.
	var atualizado fragmento
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr1.ID), map[string]any{"trilhas": []int64{1}}, &atualizado)
	if len(atualizado.Destinos) != 1 || atualizado.Destinos[0].Titulo != "Rede" {
		t.Fatalf("PUT destinos fr1: %+v", atualizado)
	}
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr2.ID), map[string]any{"trilhas": []int64{1, 2}}, nil)
	// A trilha que não existe é recusada.
	if s := c.chamar("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr1.ID), map[string]any{"trilhas": []int64{99}}, nil); s != 400 {
		t.Fatalf("trilha inexistente: %d", s)
	}
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr3.ID), map[string]any{"sem_destino": true}, &atualizado)
	if atualizado.Estado != "sem_destino" || atualizado.SemDestinoEm == "" {
		t.Fatalf("sem destino: %+v", atualizado)
	}
	// Voltar atrás: marca trilhas de novo, o sem_destino_em limpa (variável nova: o campo
	// vazio não vem no JSON por omitempty).
	var deVolta fragmento
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr3.ID), map[string]any{"trilhas": []int64{2}}, &deVolta)
	if deVolta.Estado != "confirmado" || deVolta.SemDestinoEm != "" {
		t.Fatalf("desfez o sem destino: %+v", deVolta)
	}
	// E tira a trilha de novo: confirmado sem nenhum destino marcado.
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr3.ID), map[string]any{"trilhas": []int64{}}, nil)

	var enviados []struct {
		TrilhaID   int64  `json:"trilha_id"`
		Titulo     string `json:"titulo"`
		PropostaID int64  `json:"proposta_id"`
	}
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/enviar", f2.ID), nil, &enviados)
	if len(enviados) != 2 || enviados[0].TrilhaID != 1 || enviados[1].TrilhaID != 2 || enviados[0].PropostaID == 0 {
		t.Fatalf("dois encaixes: %+v", enviados)
	}
	c.a.fila.esvaziar(context.Background())

	// O encaixe da trilha A tem os intervalos dos dois fragmentos; o da B, o do fr2.
	for i, intervalos := range []string{"00:00–00:20; 00:20–00:40", "00:20–00:40"} {
		var bruto string
		if err := c.a.db.QueryRow(`SELECT material FROM encaixes WHERE proposta_id = ?`, enviados[i].PropostaID).Scan(&bruto); err != nil {
			t.Fatal(err)
		}
		var m material
		if err := json.Unmarshal([]byte(bruto), &m); err != nil {
			t.Fatal(err)
		}
		if m.Tipo != "youtube" || m.Localizador != intervalos || m.Referencia != f2.URL || !strings.Contains(m.Titulo, "(Canal de Teste)") {
			t.Fatalf("material do encaixe %d: %+v", i, m)
		}
		var pedido string
		if err := c.a.db.QueryRow(`SELECT pedido FROM propostas WHERE id = ?`, enviados[i].PropostaID).Scan(&pedido); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(pedido, "pelo seu intervalo") {
			t.Fatalf("o pedido do encaixe %d: %q", i, pedido)
		}
	}

	var f3 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", f2.ID), nil, &f3)
	for _, fr := range f3.Fragmentos {
		switch fr.ID {
		case fr1.ID:
			if fr.Estado != "enviado" || len(fr.Destinos) != 1 || fr.Destinos[0].PropostaID == 0 || fr.Destinos[0].Estado != "aguardando" {
				t.Fatalf("fr1 enviado: %+v", fr)
			}
		case fr2.ID:
			if fr.Estado != "enviado" || len(fr.Destinos) != 2 || fr.Destinos[0].Estado != "aguardando" || fr.Destinos[1].Estado != "aguardando" {
				t.Fatalf("fr2 enviado às duas trilhas: %+v", fr)
			}
		default:
			// Confirmado sem trilha marcada NÃO vai a "enviado" (foi a lugar nenhum).
			if fr.Estado != "confirmado" || len(fr.Destinos) != 0 {
				t.Fatalf("o terceiro continua confirmado e sem destino: %+v", fr)
			}
		}
	}
	// Agora o terceiro fica sem destino e o resumo conta.
	c.ok("PUT", fmt.Sprintf("/api/fragmentos/%d/destinos", fr3.ID), map[string]any{"sem_destino": true}, nil)
	var lista []fonte
	c.ok("GET", "/api/entrada", nil, &lista)
	if len(lista) != 1 || lista[0].ResumoFragmentos != "3 fragmentos · 2 enviados · 1 sem destino" {
		t.Fatalf("resumo da entrada: %+v", lista)
	}
	// Nada mais a mandar.
	if s := c.chamar("POST", fmt.Sprintf("/api/entrada/%d/enviar", f2.ID), nil, nil); s != 400 {
		t.Fatalf("enviar sem destino marcado: %d", s)
	}
}

func TestEncaixeYoutubePedido(t *testing.T) {
	f := &iaFalsa{}
	iaDaEntrada(f, destinoJSON, encaixeNovoLocalizador("00:00–00:20"))
	c := clienteFragmentos(t, f)
	inicial := fonteFragmentada(t, c)
	trilhaComEtapa(c)

	// O material youtube traz só os trechos dos intervalos, com os títulos dos fragmentos
	// (a transcrição falsa tem falas em [00:00], [00:10], [00:20] e [1:02:05]).
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{
		"tipo": "youtube", "titulo": "Vídeo", "referencia": inicial.URL, "localizador": "00:00–00:20; 1:02:00–1:03:00"}}, nil)
	c.a.fila.esvaziar(context.Background())

	arqs := f.arquivos[1] // a 2ª chamada à IA (a 1ª é a de fragmentar)
	pedido := arqs["pedido.md"]
	if !strings.Contains(pedido, "- Tipo: youtube") || !strings.Contains(pedido, "- Só este trecho: 00:00–00:20; 1:02:00–1:03:00") {
		t.Fatalf("o material no pedido:\n%s", pedido)
	}
	texto := arqs["material/01-youtube.md"]
	if texto == "" {
		t.Fatalf("o material devia ter ido para a pasta: %v", arqs)
	}
	if !strings.Contains(texto, "## 00:00–00:20 · Começo") || !strings.Contains(texto, "Resumo do começo.") ||
		!strings.Contains(texto, "## 1:02:00–1:03:00") {
		t.Fatalf("cabeçalhos com o título e o resumo dos fragmentos:\n%s", texto)
	}
	if !strings.Contains(texto, "[00:00]") || !strings.Contains(texto, "[00:10]") || !strings.Contains(texto, "[1:02:05]") ||
		strings.Contains(texto, "[00:20]") {
		t.Fatalf("só as falas dos intervalos:\n%s", texto)
	}

	// Sem o vídeo na Entrada, o encaixe falha com a mensagem pedida.
	c.ok("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{
		"tipo": "youtube", "referencia": "https://www.youtube.com/watch?v=BBBBBBBBBBB", "localizador": "00:00–01:00"}}, nil)
	c.a.fila.esvaziar(context.Background())
	var encaixes []struct {
		Proposta proposta `json:"proposta"`
	}
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &encaixes)
	var falhou *proposta
	for i := range encaixes {
		if encaixes[i].Proposta.Estado == "falhou" {
			falhou = &encaixes[i].Proposta
		}
	}
	if falhou == nil || !strings.Contains(falhou.Erro, "o vídeo não está na Entrada") {
		t.Fatalf("o erro devia dizer que o vídeo não está na Entrada: %+v", encaixes)
	}
}

func TestEncaixeTrilhaVaziaYoutube(t *testing.T) {
	f := &iaFalsa{}
	// A IA só pode propor tópicos novos (a trilha não tem módulos) com localizador de intervalo.
	iaDaEntrada(f, destinoJSON, encaixeNovoLocalizador("00:00–00:20"))
	c := clienteFragmentos(t, f)
	inicial := fonteFragmentada(t, c)
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Vazia"}, nil)

	var p proposta
	if s := c.chamar("POST", "/api/trilhas/1/encaixar", map[string]any{"material": map[string]any{
		"tipo": "youtube", "titulo": "Vídeo", "referencia": inicial.URL, "localizador": "00:00–00:20"}}, &p); s != 201 {
		t.Fatalf("encaixe em trilha vazia com youtube vale: %d %+v", s, p)
	}
	c.a.fila.esvaziar(context.Background())

	var lista []struct {
		Proposta proposta `json:"proposta"`
	}
	c.ok("GET", "/api/trilhas/1/encaixes", nil, &lista)
	if len(lista) != 1 || lista[0].Proposta.Estado != "aguardando" {
		t.Fatalf("a proposta: %+v", lista)
	}
	var ce conteudoEncaixe
	if err := json.Unmarshal(lista[0].Proposta.Conteudo, &ce); err != nil {
		t.Fatal(err)
	}
	if len(ce.Novos) != 1 || !ce.Novos[0].ModuloNovo || len(ce.Encaixes) != 0 {
		t.Fatalf("só tópicos e módulo novos: %+v", ce)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].Proposta.ID), map[string]any{"novos": []int{0}}, nil)

	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	if len(tr.Modulos) != 1 || tr.Modulos[0].Titulo != "Vídeos" || len(tr.Modulos[0].Etapas) != 1 {
		t.Fatalf("módulo e etapa criados: %+v", tr.Modulos)
	}
	m := tr.Modulos[0].Etapas[0].Materiais[0]
	if m.Tipo != "youtube" || m.Papel != "principal" || m.Localizador != "00:00–00:20" || m.Referencia != inicial.URL {
		t.Fatalf("o material youtube principal: %+v", m)
	}
}

func TestLimparEncaixeYoutube(t *testing.T) {
	tr := trilha{Modulos: []modulo{{Titulo: "M", Etapas: []etapa{{Chave: "e1", Titulo: "E1"}}}}}
	m := material{Tipo: "youtube", Referencia: "u", Localizador: "00:00–00:20; 00:40–00:59"}
	encs, novos, tirados := limparEncaixe(tr, m,
		[]encaixeEtapa{
			{EtapaChave: "e1", Localizador: "00:00–00:20", Papel: "principal"}, // o intervalo inteiro
			{EtapaChave: "e1", Localizador: "00:05–00:10", Papel: "apoio"},     // um pedaço dele
			{EtapaChave: "e1", Localizador: "00:25–00:30", Papel: "apoio"},     // fora dos intervalos
			{EtapaChave: "e1", Localizador: "cap. 2", Papel: "apoio"},          // nem intervalo é
			{EtapaChave: "e1", Localizador: "", Papel: "apoio"},                // vazio não vale
		},
		[]encaixeNovo{
			{Titulo: "Bom", Modulo: "M", Localizador: "00:45–00:50"},
			{Titulo: "Fora", Modulo: "M", Localizador: "01:00–01:30"},
		})
	if len(encs) != 2 || encs[0].Localizador != "00:00–00:20" || encs[1].Localizador != "00:05–00:10" {
		t.Fatalf("os encaixes dentro dos intervalos ficam: %+v", encs)
	}
	if len(novos) != 1 || novos[0].Titulo != "Bom" {
		t.Fatalf("o novo com intervalo bom fica: %+v", novos)
	}
	if len(tirados) != 4 {
		t.Fatalf("três encaixes e um novo fora: %v", tirados)
	}
}

// T46: o modo automático (a extensão do Chrome) divide e sugere as trilhas sem clique; o Lucas
// confirma depois, e as trilhas marcadas ficam com os fragmentos.
func TestModoAutomatico(t *testing.T) {
	f := &iaFalsa{}
	iaDaEntrada(f, destinoJSON, "{}")
	c := clienteFragmentos(t, f)
	trilhaComEtapa(c) // trilha 1 "Rede"
	c.ok("POST", "/api/trilhas", map[string]any{"titulo": "Conhecimento de base"}, nil)

	var fontes []fonte
	c.chamar("POST", "/api/entrada", map[string]any{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA", "automatico": true}, &fontes)
	if len(fontes) != 1 || !fontes[0].Fragmentar {
		t.Fatalf("a fonte devia sair marcada para o automático: %+v", fontes)
	}
	c.a.esvaziarEntrada(context.Background())
	c.a.fila.esvaziar(context.Background())
	c.a.fila.esvaziar(context.Background())

	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f2)
	if f2.Estado != "fragmentada" || len(f2.Fragmentos) != 3 || f2.Fragmentos[0].Estado != "sugerido" {
		t.Fatalf("devia estar fragmentada, com os sugeridos: %+v", f2)
	}
	if len(f2.Fragmentos[0].Destinos) != 1 || f2.Fragmentos[0].Destinos[0].TrilhaID != 1 {
		t.Fatalf("a trilha sugerida devia vir marcada no sugerido: %+v", f2.Fragmentos[0].Destinos)
	}
	var lista []fonte
	c.ok("GET", "/api/entrada", nil, &lista)
	if len(lista) != 1 || !strings.Contains(lista[0].ResumoFragmentos, "→ Conhecimento de base, Rede") {
		t.Fatalf("a lista devia dizer as trilhas: %+v", lista)
	}
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/confirmar", fontes[0].ID), nil, nil)
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f2)
	if f2.Fragmentos[0].Estado != "confirmado" || len(f2.Fragmentos[0].Destinos) != 1 {
		t.Fatalf("confirmar devia manter as trilhas: %+v", f2.Fragmentos[0])
	}
}

// T46: vídeo mais longo que o limite fica pronto, com uma nota, sem gastar a IA.
func TestModoAutomaticoVideoLongo(t *testing.T) {
	antes := limiteAutomatico
	limiteAutomatico = 30 // o vídeo falso tem 60 s
	t.Cleanup(func() { limiteAutomatico = antes })
	f := &iaFalsa{}
	iaDaEntrada(f, destinoJSON, "{}")
	c := clienteFragmentos(t, f)
	var fontes []fonte
	c.chamar("POST", "/api/entrada", map[string]any{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA", "automatico": true}, &fontes)
	c.a.esvaziarEntrada(context.Background())
	c.a.fila.esvaziar(context.Background())
	var f2 fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", fontes[0].ID), nil, &f2)
	if f2.Estado != "pronta" || !strings.Contains(f2.Notas, "não dividi sozinho") || len(f.pedidos) != 0 {
		t.Fatalf("o longo devia ficar pronto, com a nota e sem IA: %+v (%d pedidos)", f2, len(f.pedidos))
	}
}
