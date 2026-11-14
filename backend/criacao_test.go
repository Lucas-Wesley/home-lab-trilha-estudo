package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const planoFalso = `{"trilha":{"titulo":"AWS essencial","objetivo":"Ao final, você será capaz de escolher o serviço certo.",
"descricao":"Serviços essenciais.","recorte":"EC2 e S3; fica de fora: ML.","nivel_partida":"já usa Linux","nivel_alvo":"intermediário",
"modulos":[{"titulo":"Base","descricao":"","etapas":[
 {"chave":"nuvem","titulo":"O que é nuvem","objetivo":"Explicar o modelo de nuvem.","bloom":"entender","tipo":"base","minutos":30,
  "criterio":"Explicar IaaS × PaaS.","requer":[],"materiais":[{"tipo":"web","titulo":"AWS","referencia":"https://exemplo.org/x","papel":"principal","nota":"[oficial]"}]},
 {"chave":"ec2","titulo":"EC2","objetivo":"Aplicar EC2.","bloom":"aplicar","tipo":"normal","minutos":120,
  "criterio":"","requer":["nuvem"],"materiais":[{"tipo":"documento","titulo":"Notas","referencia":"nao-existe","papel":"apoio"}]}]}]},
"notas":"Conferi com o guia da certificação.","respostas":[]}`

func TestCriacaoAIAMonta(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		switch n {
		case 1:
			return `{"perfil":"tecnologia","eixos":["objetivo","nível"],"panorama":"A AWS tem 200 serviços.","perguntas":[
				{"id":"q1","texto":"Para quê?","por_que":"muda o foco","opcoes":["trabalho","certificação"],"multipla":false},
				{"id":"q2","texto":"Já usa Linux?","por_que":"muda a base","opcoes":["sim","não"],"multipla":false}],
				"suficiente":false,"notas":"","respostas":[]}`, nil
		case 2:
			return `{"perfil":"tecnologia","eixos":[],"panorama":"","perguntas":[],"suficiente":true,"notas":"Fechado.","respostas":[]}`, nil
		case 3:
			return `{"objetivo":"Ao final, você será capaz de escolher o serviço certo.","mapa":[{"titulo":"Base","descricao":"",
				"etapas":[{"chave":"nuvem","titulo":"O que é nuvem","resumo":"","nivel":"basico"},{"chave":"ec2","titulo":"EC2","resumo":"","nivel":"basico"},
				{"chave":"ml","titulo":"SageMaker","resumo":"","nivel":"avancado"}]}],"recorte_sugerido":["nuvem","ec2"],"por_que_recorte":"",
				"nivel_partida":"usa Linux","nivel_alvo":"intermediário","notas":"","respostas":[]}`, nil
		default:
			return planoFalso, nil
		}
	}
	c := clienteComIA(t, f)
	c.ok("PUT", "/api/ajustes", map[string]any{"modelo_criacao": "opus"}, nil)

	if s := c.chamar("POST", "/api/criacoes", map[string]any{"entrada": "ia", "tema": " "}, nil); s != 400 {
		t.Fatalf("sem tema: %d", s)
	}
	if s := c.chamar("POST", "/api/criacoes", map[string]any{"entrada": "ia", "tema": "AWS", "entrevista": "enorme"}, nil); s != 400 {
		t.Fatalf("tamanho desconhecido: %d", s)
	}
	var r respostaCriacao
	if s := c.chamar("POST", "/api/criacoes", map[string]any{"entrada": "ia", "tema": "AWS", "pedido": "Para o trabalho; evitar ML.",
		"entrevista": "padrao", "profundidade": 3, "instrucoes": "Exemplos com o homelab."}, &r); s != 201 {
		t.Fatalf("criar: %d", s)
	}
	id := r.Criacao.TrilhaID
	if r.Criacao.Passo != tipoEntrevista || len(r.Propostas) != 0 || r.Limite.Rodadas != 2 {
		t.Fatalf("começo: %+v", r)
	}
	c.ok("PUT", fmt.Sprintf("/api/trilhas/%d/documentos/notas", id), map[string]any{"conteudo": "# Notas\n\nTexto."}, nil)
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{}, &r)
	if len(r.Propostas) != 1 || r.Propostas[0].Tarefa.Modelo != "opus" {
		t.Fatalf("primeira rodada: %+v", r.Propostas)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{}, nil); s != 400 {
		t.Fatalf("duas propostas ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	for _, trecho := range []string{"Tema: AWS", "Para o trabalho; evitar ML.", "rodada 1 de no máximo 2", "pré-pesquisa",
		"`documentos/notas.md`", "Da trilha: Exemplos com o homelab."} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido da entrevista sem %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.pedidos[0].regra, "a entrevista antes do mapa") || f.pedidos[0].modelo != "opus" || len(f.pedidos[0].extras) != 1 {
		t.Errorf("regra, modelo ou extras: %q %v", f.pedidos[0].modelo, f.pedidos[0].extras)
	}

	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	ent := r.Propostas[0]
	if ent.Estado != propAguardando {
		t.Fatalf("entrevista: %+v", ent)
	}
	// Pergunta inventada: recusada. Depois, a segunda rodada.
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", ent.ID), map[string]any{"respostas": []map[string]any{{"pergunta": "q9", "sugira": true}}}, nil); s != 400 {
		t.Fatalf("pergunta desconhecida: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", ent.ID), map[string]any{"proximo": "rodada", "comentario": "Foco em custo.",
		"respostas": []map[string]any{{"pergunta": "q1", "escolhas": []string{"trabalho"}}, {"pergunta": "q2", "sugira": true}}}, &r)
	if len(r.Criacao.Rodadas) != 1 || r.Criacao.Passo != tipoEntrevista || len(r.Propostas) != 2 {
		t.Fatalf("depois da rodada 1: %+v", r)
	}
	c.a.fila.esvaziar(context.Background())
	pedido = f.arquivos[1]["pedido.md"]
	for _, trecho := range []string{"escolheu: trabalho", "não sei, sugira", "Comentário dele: Foco em custo.", "última rodada"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido da rodada 2 sem %q:\n%s", trecho, pedido)
		}
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	ent2 := r.Propostas[1]
	// O limite (2 rodadas) não deixa pedir uma terceira.
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", ent2.ID), map[string]any{"proximo": "rodada"}, nil); s != 400 {
		t.Fatalf("rodada além do limite: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", ent2.ID), map[string]any{"proximo": "mapa"}, &r)
	if r.Criacao.Passo != tipoMapa {
		t.Fatalf("passo depois da entrevista: %s", r.Criacao.Passo)
	}
	c.a.fila.esvaziar(context.Background())
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	mapa := r.Propostas[2]
	if mapa.Tipo != tipoMapa || mapa.Estado != propAguardando || !strings.Contains(f.pedidos[2].regra, "mapa do domínio") {
		t.Fatalf("mapa: %+v", mapa)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", mapa.ID), map[string]any{"recorte": []string{}}, nil); s != 400 {
		t.Fatalf("recorte vazio: %d", s)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", mapa.ID), map[string]any{"recorte": []string{"inventada"}}, nil); s != 400 {
		t.Fatalf("recorte com etapa fora do mapa: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", mapa.ID), map[string]any{"recorte": []string{"nuvem", "ec2"},
		"nivel_partida": "usa Linux no homelab", "profundidade": 4}, &r)
	if r.Criacao.Passo != tipoPlano || len(r.Criacao.Escolhas.Recorte) != 2 {
		t.Fatalf("depois do mapa: %+v", r.Criacao)
	}
	c.a.fila.esvaziar(context.Background())
	pedido = f.arquivos[3]["pedido.md"]
	for _, trecho := range []string{"✔ `nuvem`", "  `ml`", "Nível de partida: usa Linux no homelab", "4. Aprofundado", "escolher o serviço certo"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido do plano sem %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.pedidos[3].regra, "o plano da trilha") || !strings.Contains(f.pedidos[3].regra, "criar uma trilha") {
		t.Errorf("a regra do plano leva o método junto")
	}

	// A revisão automática apontou o que falha nos critérios.
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	plano := r.Propostas[3]
	var cp conteudoPlano
	if err := json.Unmarshal(plano.Conteudo, &cp); err != nil {
		t.Fatal(err)
	}
	achou := map[string]bool{}
	for _, x := range cp.Revisao {
		achou[x.Criterio+" "+x.Onde] = true
		if x.Grave {
			t.Errorf("aviso grave inesperado: %+v", x)
		}
	}
	for _, k := range []string{"C3 etapa:ec2", "C15 etapa:ec2", "C9 etapa:ec2", "C10 etapa:ec2", "C11 "} {
		if !achou[k] {
			t.Errorf("revisão sem %q: %+v", k, cp.Revisao)
		}
	}
	if achou["C3 etapa:nuvem"] {
		t.Errorf("etapa boa com aviso: %+v", cp.Revisao)
	}

	// Comentário numa etapa e ajuste: a v2 recebe a anterior e o comentário.
	var com proposta
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/comentarios", plano.ID), map[string]any{"texto": "Divida esta.", "ancora": "etapa:ec2"}, &com)
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/ajustar", plano.ID), map[string]any{}, nil)
	c.a.fila.esvaziar(context.Background())
	if f.arquivos[4]["versao-anterior.json"] == "" || !strings.Contains(f.arquivos[4]["pedido.md"], "(em etapa:ec2): Divida esta.") {
		t.Errorf("ajuste do plano sem a versão anterior ou o comentário:\n%s", f.arquivos[4]["pedido.md"])
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	v2 := r.Propostas[4]
	if v2.Versao != 2 || r.Propostas[3].Estado != propSubstituida {
		t.Fatalf("v2: %+v / v1 %s", v2, r.Propostas[3].Estado)
	}

	// Aprovar importa a trilha no rascunho que já existia.
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", v2.ID), map[string]any{}, &r)
	if r.Criacao.Passo != passoPronta {
		t.Fatalf("passo final: %s", r.Criacao.Passo)
	}
	var tr trilha
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d", id), nil, &tr)
	if tr.Titulo != "AWS essencial" || tr.Instrucoes != "Exemplos com o homelab." || tr.Profundidade != 4 || tr.NivelPartida != "já usa Linux" ||
		len(tr.Modulos) != 1 || len(tr.Modulos[0].Etapas) != 2 || len(tr.Documentos) != 1 || tr.Estado != "rascunho" {
		t.Fatalf("trilha criada: %+v", tr)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{}, nil); s != 400 {
		t.Fatalf("gerar depois de pronta: %d", s)
	}
}

func TestCriacaoComLista(t *testing.T) {
	f := &iaFalsa{}
	f.resposta = func(n int, arq map[string]string) (string, error) {
		if n == 1 {
			return `{"itens":[{"chave":"ec2","titulo":"EC2","nota":""},{"chave":"s3","titulo":"S3","nota":""},{"chave":"iam","titulo":"IAM","nota":""}],
				"sugestoes":[{"id":"s1","acao":"inserir","antes_de":"iam","titulo":"Responsabilidade compartilhada","motivo":"base do IAM"},
				{"id":"s2","acao":"observar","item":"s3","titulo":"S3 antes de EC2?","motivo":"mais simples"}],
				"nivel_partida":"","nivel_alvo":"","notas":"","respostas":[]}`, nil
		}
		return `{"trilha":{"titulo":"AWS","objetivo":"x","descricao":"","recorte":"","modulos":[{"titulo":"M","etapas":[
			{"chave":"ec2","titulo":"EC2","objetivo":"x","bloom":"entender","minutos":30,"criterio":"x","requer":["iam"],"materiais":[]},
			{"chave":"iam","titulo":"IAM","objetivo":"x","bloom":"entender","minutos":30,"criterio":"x","requer":[],"materiais":[]}]}]},
			"notas":"","respostas":[]}`, nil
	}
	c := clienteComIA(t, f)
	if s := c.chamar("POST", "/api/criacoes", map[string]any{"entrada": "lista", "tema": "AWS", "lista": "EC2"}, nil); s != 400 {
		t.Fatalf("lista de um item: %d", s)
	}
	var r respostaCriacao
	c.ok("POST", "/api/criacoes", map[string]any{"entrada": "lista", "tema": "AWS", "lista": "1. EC2\n- S3\n\n* IAM"}, &r)
	id := r.Criacao.TrilhaID
	if r.Criacao.Passo != tipoValidacao {
		t.Fatalf("passo: %s", r.Criacao.Passo)
	}
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{}, nil)
	c.a.fila.esvaziar(context.Background())
	if !strings.Contains(f.arquivos[0]["pedido.md"], "1. EC2\n2. S3\n3. IAM\n") || !strings.Contains(f.pedidos[0].regra, "validar a lista") {
		t.Errorf("pedido da validação:\n%s", f.arquivos[0]["pedido.md"])
	}
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	val := r.Propostas[0]
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", val.ID), map[string]any{"aceitas": []string{"s9"}}, nil); s != 400 {
		t.Fatalf("sugestão desconhecida: %d", s)
	}
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/aprovar", val.ID), map[string]any{"aceitas": []string{"s1"}, "pedido": "Sem custos."}, &r)
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[1]["pedido.md"]
	aceitas, recusadas, _ := strings.Cut(pedido, "recusou")
	if !strings.Contains(aceitas, "Responsabilidade compartilhada") || !strings.Contains(recusadas, "S3 antes de EC2?") || !strings.Contains(pedido, "Deste pedido: Sem custos.") {
		t.Errorf("pedido do plano (B):\n%s", pedido)
	}
	// Plano com pré-requisito depois da etapa: aviso grave, e aprovar é recusado.
	c.ok("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, &r)
	plano := r.Propostas[1]
	var cp conteudoPlano
	json.Unmarshal(plano.Conteudo, &cp)
	if len(cp.Revisao) != 1 || !cp.Revisao[0].Grave || !strings.Contains(cp.Revisao[0].Texto, "vem depois") {
		t.Fatalf("revisão do plano inválido: %+v", cp.Revisao)
	}
	if s := c.chamar("POST", fmt.Sprintf("/api/propostas/%d/aprovar", plano.ID), map[string]any{}, nil); s != 400 {
		t.Fatalf("aprovar plano inválido: %d", s)
	}
	// Descartar e gerar de novo o mesmo passo.
	c.ok("POST", fmt.Sprintf("/api/propostas/%d/descartar", plano.ID), nil, nil)
	c.ok("POST", fmt.Sprintf("/api/trilhas/%d/criacao/gerar", id), map[string]any{"pedido": "IAM primeiro."}, &r)
	if len(r.Propostas) != 3 || r.Propostas[2].Tipo != tipoPlano || r.Propostas[2].Pedido != "IAM primeiro." {
		t.Fatalf("gerar de novo: %+v", r.Propostas)
	}
	// Excluir a trilha leva a criação junto.
	c.a.fila.esvaziar(context.Background())
	c.ok("DELETE", fmt.Sprintf("/api/trilhas/%d", id), nil, nil)
	if s := c.chamar("GET", fmt.Sprintf("/api/trilhas/%d/criacao", id), nil, nil); s != 404 {
		t.Fatalf("criação depois de excluir: %d", s)
	}
}
