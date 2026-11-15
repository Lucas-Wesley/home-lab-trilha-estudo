package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// exerciciosDoNAT: aprova dois exercícios na etapa NAT (o primeiro: certa = 0; o segundo: "origem").
func exerciciosDoNAT(t *testing.T, c *cliente, f *iaFalsa, nat int64) {
	t.Helper()
	gravarLicao(c, nat)
	var p proposta
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/exercicios", nat), map[string]any{}, &p)
	c.a.fila.esvaziar(context.Background())
	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=exercicios", nat), nil, &lista)
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[len(lista)-1].ID), nil, nil)
}

const exerciciosFalsos = `{"exercicios":[
{"tipo":"escolha","enunciado":"Por que o NAT existe?","alternativas":["Faltam endereços IPv4","Segurança","Velocidade"],"correta":0,"explicacao":"Porque faltam.","conceito":"motivo do NAT"},
{"tipo":"completar","enunciado":"O NAT troca o endereço de ____.","aceitas":["origem"],"explicacao":"De origem.","conceito":"troca de endereço"}],
"notas":"","respostas":[]}`

func TestRevisaoEspacadaEPontosFracos(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) { return exerciciosFalsos, nil }
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c)
	exerciciosDoNAT(t, c, f, nat)

	// Errou o primeiro: ponto fraco "motivo do NAT", e a etapa fica fraca (50% < 80%).
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/exercicios/tentativas", nat), map[string]any{"respostas": []any{
		map[string]any{"escolha": 1}, map[string]any{"texto": "origem"}}}, nil)
	var est struct {
		PontosFracos []pontoFraco `json:"pontos_fracos"`
		Sinais       *sinaisEtapa `json:"sinais"`
		Revisao      *revisao     `json:"revisao"`
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if len(est.PontosFracos) != 1 || est.PontosFracos[0].Conceito != "motivo do NAT" || est.PontosFracos[0].Resposta != "Segurança" ||
		est.Sinais == nil || len(est.Sinais.Motivos) != 1 || est.Revisao != nil {
		t.Fatalf("estudo: %+v %+v", est.PontosFracos, est.Sinais)
	}

	// Revisão só de etapa concluída.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/revisao", nat), map[string]any{"autoavaliacao": "bem"}, nil); s != 400 {
		t.Fatalf("revisão de etapa não concluída: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/status", nat), map[string]any{"status": "concluida"}, nil)
	var rv struct {
		Revisao   *revisao          `json:"revisao"`
		Modo      string            `json:"modo"`
		Perguntas []perguntaRevisao `json:"perguntas"`
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/revisao", nat), nil, &rv)
	amanha := hoje().AddDate(0, 0, 1).Format(formatoData)
	if rv.Revisao == nil || rv.Revisao.Proxima != amanha || rv.Revisao.Pendente || rv.Modo != "exercicios" || len(rv.Perguntas) != 2 {
		t.Fatalf("revisão: %+v %+v", rv.Revisao, rv)
	}
	// Ainda não chegou: o Hoje não mostra; chegou (o dia passou), mostra.
	var pend struct {
		Revisoes []revisaoPendente `json:"revisoes"`
	}
	c.ok("PUT", "/api/trilhas/1/meta", map[string]any{"dias": []int{0, 1, 2, 3, 4, 5, 6}, "minutos": 60}, nil)
	c.ok("GET", "/api/revisoes", nil, &pend)
	if len(pend.Revisoes) != 0 {
		t.Fatalf("revisão antes do dia: %+v", pend)
	}
	c.a.db.Exec(`UPDATE revisoes SET proxima = ?`, hoje().AddDate(0, 0, -2).Format(formatoData))
	c.ok("GET", "/api/revisoes", nil, &pend)
	if len(pend.Revisoes) != 1 || pend.Revisoes[0].EtapaID != nat || pend.Revisoes[0].Atraso != 2 || !pend.Revisoes[0].Exercicios {
		t.Fatalf("revisões pendentes: %+v", pend)
	}

	// Acertou tudo: sobe um degrau (3 dias) e o ponto fraco sai.
	respostas := []any{}
	for _, p := range rv.Perguntas {
		if p.Tipo == "escolha" {
			respostas = append(respostas, map[string]any{"indice": p.Indice, "escolha": 0})
		} else {
			respostas = append(respostas, map[string]any{"indice": p.Indice, "texto": "Origem"})
		}
	}
	var feito struct {
		Tentativa tentativa `json:"tentativa"`
		Revisao   *revisao  `json:"revisao"`
		Efeito    string    `json:"efeito"`
	}
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/revisao", nat), map[string]any{"respostas": respostas}, &feito)
	if feito.Tentativa.Nota != 100 || feito.Efeito != "subiu" || feito.Revisao.Nivel != 1 || feito.Revisao.Proxima != hoje().AddDate(0, 0, 3).Format(formatoData) {
		t.Fatalf("revisão certa: %+v %+v %s", feito.Tentativa, feito.Revisao, feito.Efeito)
	}
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/estudo", nat), nil, &est)
	if len(est.PontosFracos) != 0 {
		t.Fatalf("o ponto fraco acertado continua: %+v", est.PontosFracos)
	}
	// Esqueceu: volta para o primeiro degrau.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/revisao", nat), map[string]any{"autoavaliacao": "esqueci"}, &feito)
	if feito.Efeito != "voltou" || feito.Revisao.Nivel != 0 || feito.Revisao.Proxima != amanha {
		t.Fatalf("esqueci: %+v %s", feito.Revisao, feito.Efeito)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/revisao", nat), map[string]any{"respostas": []any{map[string]any{"indice": 9}}}, nil); s != 400 {
		t.Fatalf("índice fora: %d", s)
	}

	// O painel junta tudo.
	var painel struct {
		Etapas       []linhaPainel  `json:"etapas"`
		PontosFracos []pontoFraco   `json:"pontos_fracos"`
		Resumo       map[string]int `json:"resumo"`
		Revisoes     struct {
			EmDia int `json:"em_dia"`
		} `json:"revisoes"`
	}
	c.ok("GET", "/api/trilhas/1/painel", nil, &painel)
	if len(painel.Etapas) != 2 || painel.Etapas[1].Revisao == nil || painel.Etapas[1].Sinais.Revisao.Vezes != 2 ||
		painel.Revisoes.EmDia != 1 || painel.Resumo["media_exercicios"] != 50 {
		t.Fatalf("painel: %+v", painel)
	}
}

func TestSimuladoIntercalado(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		if strings.Contains(arq["pedido.md"], "correção do simulado intercalado") {
			return `{"correcoes":[{"pergunta":1,"nota":30,"veredito":"errado","comentario":"Não.","faltou":"o motivo"},
{"pergunta":2,"nota":100,"veredito":"certo","comentario":"Sim.","faltou":""}],"resumo":"Revise."}`, nil
		}
		if strings.Contains(arq["pedido.md"], "\"IP e DHCP\"") {
			return `{"perguntas":[{"pergunta":"Para que serve o DHCP?","esperado":"dar endereço","conceito":"dhcp"}],"notas":"","respostas":[]}`, nil
		}
		return `{"perguntas":[{"pergunta":"Por que o NAT existe?","esperado":"faltam IPv4","conceito":"motivo"}],"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c)
	ip := nat - 1
	for _, id := range []int64{ip, nat} {
		c.ok("PUT", fmt.Sprintf("/api/etapas/%d", id), map[string]any{"titulo": map[int64]string{ip: "IP e DHCP", nat: "NAT"}[id],
			"licao": "# Lição\n\n## Parte\n\nTexto da etapa."}, nil)
		c.ok("POST", fmt.Sprintf("/api/etapas/%d/simulado", id), map[string]any{}, nil)
		c.a.fila.esvaziar(context.Background())
		var lista []proposta
		c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=simulado", id), nil, &lista)
		c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), nil, nil)
	}
	// Nenhuma estudada ainda: nada para intercalar.
	var sorteio struct {
		Perguntas []perguntaIntercalada `json:"perguntas"`
		Etapas    int                   `json:"etapas"`
	}
	c.ok("GET", "/api/trilhas/1/intercalado", nil, &sorteio)
	if len(sorteio.Perguntas) != 0 {
		t.Fatalf("intercalado sem etapas estudadas: %+v", sorteio)
	}
	for _, id := range []int64{ip, nat} {
		c.ok("POST", fmt.Sprintf("/api/etapas/%d/status", id), map[string]any{"status": "concluida"}, nil)
	}
	c.ok("GET", "/api/trilhas/1/intercalado?n=5", nil, &sorteio)
	if len(sorteio.Perguntas) != 2 || sorteio.Etapas != 2 {
		t.Fatalf("sorteio: %+v", sorteio)
	}
	respostas := []any{}
	for _, p := range sorteio.Perguntas {
		respostas = append(respostas, map[string]any{"etapa_chave": p.EtapaChave, "pergunta": p.Indice, "resposta": "Resposta de " + p.EtapaChave})
	}
	var tt tentativa
	if s := c.chamar("POST", "/api/trilhas/1/intercalado", map[string]any{"respostas": respostas}, &tt); s != 201 {
		t.Fatalf("responder: %d", s)
	}
	if s := c.chamar("POST", "/api/trilhas/1/intercalado", map[string]any{"respostas": respostas}, nil); s != 400 {
		t.Fatalf("dois intercalados ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[len(f.arquivos)-1]
	if !strings.Contains(pedido["pedido.md"], "lição em `licoes/") || pedido["licoes/nat.md"] == "" || pedido["licoes/ip-e-dhcp.md"] == "" {
		t.Fatalf("pedido do intercalado: %v", pedido)
	}
	var painel struct {
		PontosFracos []pontoFraco `json:"pontos_fracos"`
		Intercalado  struct {
			Tentativas []tentativa `json:"tentativas"`
		} `json:"intercalado"`
	}
	c.ok("GET", "/api/trilhas/1/painel", nil, &painel)
	if len(painel.Intercalado.Tentativas) != 1 || painel.Intercalado.Tentativas[0].Estado != tentCorrigida || painel.Intercalado.Tentativas[0].Nota != 65 {
		t.Fatalf("intercalado no painel: %+v", painel.Intercalado)
	}
	// A primeira pergunta corrigida (errada) vira ponto fraco da etapa dela.
	var primeira respondidaSimulado
	var guardadas []respondidaSimulado
	json.Unmarshal(painel.Intercalado.Tentativas[0].Respostas, &guardadas)
	primeira = guardadas[0]
	if len(painel.PontosFracos) != 1 || painel.PontosFracos[0].EtapaChave != primeira.EtapaChave || painel.PontosFracos[0].Onde != "intercalado" {
		t.Fatalf("pontos fracos do intercalado: %+v (primeira %+v)", painel.PontosFracos, primeira)
	}
}

func TestReforcoInsereEtapas(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		if strings.Contains(arq["pedido.md"], "# Pedido: reforço") {
			return `{"diagnostico":"Confunde o motivo.","etapas":[
{"titulo":"Endereços IPv4 e escassez","objetivo":"Explicar a escassez.","bloom":"entender","minutos":20,"posicao":"antes","foco":"A conta dos 4 bilhões.","motivo":"Errou o motivo."},
{"titulo":"NAT na prática","objetivo":"Aplicar o NAT.","bloom":"inventado","minutos":0,"posicao":"depois","foco":"Tabela de tradução no OpenWrt.","motivo":"Pouca prática."},
{"titulo":" ","objetivo":"","bloom":"","minutos":10,"posicao":"depois","foco":"","motivo":""}],"notas":"","respostas":[]}`, nil
		}
		return exerciciosFalsos, nil
	}
	c := clienteComIA(t, f)
	nat := trilhaComEtapa(c)
	// Sem erros e sem pedido, não há o que reforçar.
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/reforco", nat), map[string]any{}, nil); s != 400 {
		t.Fatalf("reforço sem erros: %d", s)
	}
	exerciciosDoNAT(t, c, f, nat)
	// gravarLicao regrava a etapa só com a lição: volta o requisito e o material.
	c.ok("PUT", fmt.Sprintf("/api/etapas/%d", nat), map[string]any{"titulo": "NAT", "objetivo": "Explicar o NAT.", "requer": []int64{nat - 1},
		"licao":     "# NAT\n\n## Por que existe\n\nFaltam endereços IPv4.",
		"materiais": []map[string]any{{"tipo": "documento", "titulo": "WAN", "referencia": "05-wan", "papel": "principal"}}}, nil)
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/exercicios/tentativas", nat), map[string]any{"respostas": []any{
		map[string]any{"escolha": 2}, map[string]any{"texto": "destino"}}}, nil)
	var p proposta
	if s := c.chamar("POST", fmt.Sprintf("/api/etapas/%d/reforco", nat), map[string]any{"pedido": "Com exemplos do OpenWrt."}, &p); s != 201 {
		t.Fatalf("pedir reforço: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[len(f.arquivos)-1]["pedido.md"]
	for _, trecho := range []string{"# Pedido: reforço da etapa \"NAT\"", "**motivo do NAT**", "Resposta dele: Velocidade", "Esperado / o que faltou: origem",
		"Exercícios: última nota 0", "Deste pedido: Com exemplos do OpenWrt.", "NAT  ← esta"} {
		if !strings.Contains(pedido, trecho) {
			t.Fatalf("pedido do reforço sem %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.pedidos[len(f.pedidos)-1].regra, "o reforço de uma etapa") {
		t.Fatalf("regra do reforço")
	}
	var lista []proposta
	c.ok("GET", fmt.Sprintf("/api/etapas/%d/propostas?tipo=reforco", nat), nil, &lista)
	var cr conteudoReforco
	json.Unmarshal(lista[0].Conteudo, &cr)
	if len(cr.Etapas) != 2 || cr.Etapas[1].Bloom != "" || cr.Etapas[1].Minutos != 30 {
		t.Fatalf("reforço limpo: %+v", cr)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), map[string]any{"aceitos": []int{}}, nil); s != 400 {
		t.Fatalf("aprovar sem marcar: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", lista[0].ID), map[string]any{"aceitos": []int{0, 1}}, nil)
	var tr trilha
	c.ok("GET", "/api/trilhas/1", nil, &tr)
	var titulos []string
	porTitulo := map[string]etapa{}
	for _, e := range tr.Modulos[0].Etapas {
		titulos = append(titulos, e.Titulo)
		porTitulo[e.Titulo] = e
	}
	if strings.Join(titulos, " | ") != "IP e DHCP | Endereços IPv4 e escassez | NAT | NAT na prática" {
		t.Fatalf("ordem: %v", titulos)
	}
	base, pratica, alvo := porTitulo["Endereços IPv4 e escassez"], porTitulo["NAT na prática"], porTitulo["NAT"]
	if base.Tipo != "reforco" || base.Foco != "A conta dos 4 bilhões." || len(base.Requer) != 1 || base.Requer[0] != nat-1 {
		t.Fatalf("etapa antes: %+v", base)
	}
	if len(pratica.Requer) != 1 || pratica.Requer[0] != nat || len(pratica.Materiais) != 1 || pratica.Materiais[0].Papel != "apoio" {
		t.Fatalf("etapa depois: %+v", pratica)
	}
	if len(alvo.Requer) != 2 {
		t.Fatalf("a etapa reforçada ganha a de antes como requisito: %+v", alvo.Requer)
	}
	// A lição da etapa de reforço recebe o foco.
	c.ok("POST", fmt.Sprintf("/api/etapas/%d/licao", pratica.ID), map[string]any{}, nil)
	f.resposta = func(int, map[string]string) (string, error) {
		return `{"licao":"# NAT na prática","notas":"","respostas":[]}`, nil
	}
	c.a.fila.esvaziar(context.Background())
	pedido = f.arquivos[len(f.arquivos)-1]["pedido.md"]
	if !strings.Contains(pedido, "etapa de **reforço**") || !strings.Contains(pedido, "Foco pedido para esta lição: Tabela de tradução no OpenWrt.") {
		t.Fatalf("pedido da lição de reforço:\n%s", pedido)
	}
}

func TestMiniTesteDeNivel(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		switch {
		case strings.Contains(arq["pedido.md"], "# Pedido: mini-teste"):
			return `{"perguntas":[
{"enunciado":"Q1","alternativas":["a","b","c"],"correta":0,"explicacao":"e","item":"nuvem","nivel":"basico"},
{"enunciado":"Q2","alternativas":["a","b","c"],"correta":1,"explicacao":"e","item":"ec2","nivel":"basico"},
{"enunciado":"Q3","alternativas":["a","b","c"],"correta":2,"explicacao":"e","item":"ml","nivel":"avancado"},
{"enunciado":"Q4","alternativas":["a","b"],"correta":7,"explicacao":"mal formada","item":"ec2","nivel":"basico"},
{"enunciado":"Q5","alternativas":["a","b"],"correta":0,"explicacao":"item que não existe","item":"zzz","nivel":"basico"}]}`, nil
		case strings.Contains(arq["pedido.md"], "# Pedido: mapa"):
			return `{"objetivo":"Escolher o serviço.","mapa":[{"titulo":"Base","descricao":"","etapas":[
{"chave":"nuvem","titulo":"O que é nuvem","resumo":"","nivel":"basico"},{"chave":"ec2","titulo":"EC2","resumo":"","nivel":"basico"},
{"chave":"ml","titulo":"SageMaker","resumo":"","nivel":"avancado"}]}],"recorte_sugerido":["nuvem","ec2"],"por_que_recorte":"",
"nivel_partida":"","nivel_alvo":"","notas":"","respostas":[]}`, nil
		case strings.Contains(arq["pedido.md"], "# Pedido: plano"):
			return planoFalso, nil
		}
		return `{"perfil":"tecnologia","eixos":[],"panorama":"","perguntas":[],"suficiente":true,"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	var r respostaCriacao
	c.ok("POST", "/api/criacoes", map[string]any{"entrada": "ia", "tema": "AWS", "entrevista": "rapida", "profundidade": 3}, &r)
	id := r.Criacao.TrilhaID
	url := fmt.Sprintf("/api/trilhas/%d/criacao", id)
	c.ok("POST", url+"/gerar", map[string]any{}, &r)
	c.a.fila.esvaziar(context.Background())
	// Na entrevista ainda não há mapa.
	if s := c.chamar("POST", url+"/nivelamento", nil, nil); s != 400 {
		t.Fatalf("mini-teste sem mapa: %d", s)
	}
	c.ok("GET", url, nil, &r)
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", r.Propostas[0].ID), map[string]any{"proximo": "mapa"}, &r)
	c.a.fila.esvaziar(context.Background())
	c.ok("POST", url+"/nivelamento", nil, &r)
	if r.Nivelamento == nil || r.Nivelamento.Estado != "gerando" || len(r.Nivelamento.Itens) != 3 {
		t.Fatalf("pedir o mini-teste: %+v", r.Nivelamento)
	}
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", url, nil, &r)
	n := r.Nivelamento
	if n.Estado != "pronto" || len(n.Perguntas) != 3 || n.Perguntas[0].Correta != -1 || n.Perguntas[0].Explicacao != "" {
		t.Fatalf("mini-teste pronto (sem gabarito): %+v", n)
	}
	if s := c.chamar("POST", url+"/nivelamento/respostas", map[string]any{"respostas": []int{0}}, nil); s != 400 {
		t.Fatalf("respostas a menos: %d", s)
	}
	// Acertou nuvem, errou EC2, não sabe SageMaker.
	c.ok("POST", url+"/nivelamento/respostas", map[string]any{"respostas": []int{0, 2, -1}}, &r)
	res := r.Nivelamento.Resultado
	if r.Nivelamento.Estado != "corrigido" || res.Acertos != 1 || res.ComecarEm != "ec2" || strings.Join(res.Dominados, ",") != "nuvem" ||
		res.NivelSugerido != "iniciante" || r.Nivelamento.Perguntas[1].Correta != 1 {
		t.Fatalf("resultado (1 de 2 no básico = iniciante): %+v", res)
	}
	// O resultado vai para o pedido do plano.
	var mapa proposta
	for _, p := range r.Propostas {
		if p.Tipo == tipoMapa {
			mapa = p
		}
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", mapa.ID), map[string]any{"recorte": []string{"nuvem", "ec2"}}, nil)
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[len(f.arquivos)-1]["pedido.md"]
	if !strings.Contains(pedido, "## Mini-teste de nível (ele fez)") || !strings.Contains(pedido, "Acertou 1 de 3") || !strings.Contains(pedido, "`ec2` EC2: 0 de 1") {
		t.Fatalf("pedido do plano:\n%s", pedido)
	}
}

func TestCorrigirNivelamento(t *testing.T) {
	itens := []itemNivel{{Chave: "a"}, {Chave: "b"}, {Chave: "c"}}
	qs := []perguntaNivel{
		{Alternativas: []string{"x", "y"}, Correta: 0, Item: "a", Nivel: "basico"},
		{Alternativas: []string{"x", "y"}, Correta: 0, Item: "b", Nivel: "intermediario"},
		{Alternativas: []string{"x", "y"}, Correta: 0, Item: "c", Nivel: "avancado"},
	}
	if r := corrigirNivelamento(itens, qs, []int{0, 0, 0}); r.NivelSugerido != "avançado" || r.ComecarEm != "" || len(r.Dominados) != 3 {
		t.Fatalf("tudo certo: %+v", r)
	}
	if r := corrigirNivelamento(itens, qs, []int{1, 1, 1}); r.NivelSugerido != "iniciante" || r.ComecarEm != "a" {
		t.Fatalf("tudo errado: %+v", r)
	}
	if r := corrigirNivelamento(itens, qs, []int{0, 0, 1}); r.NivelSugerido != "intermediário" || r.ComecarEm != "c" {
		t.Fatalf("até o intermediário: %+v", r)
	}
}
