package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBlog(t *testing.T) {
	pacote := pacoteDeTeste(t, map[string]string{
		"ROADMAP.md": lerFixtureProjeto(t, "ROADMAP.md"),
		"ESTADO.md":  lerFixtureProjeto(t, "ESTADO.md"),
		"PROMPT.md":  lerFixtureProjeto(t, "PROMPT.md"),
		"itens/0.1-o-repositorio-e-o-ambiente.md": "# 0.1\n\n- **Estado:** estudo\n\n## 2. Estudo\n" +
			"| Fonte | Por quê (o que ela destrava) | Situação |\n|---|---|---|\n" +
			"| T13 › Fundamentos › Por que Go e o ambiente | O toolchain | estudado |\n\nDECISÃO-DO-CADERNO\n",
	})
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(pacote) }))
	defer gh.Close()

	f := &iaFalsa{}
	sugestoes := `{"sugestoes":[{"titulo":"Bases sólidas: Go, Git e CI","alternativas":["A"," "],"tese":"O chão vem antes.","leitor":"x",
"formato":"fundamentos","horizonte":"atemporal","arco":{"comeco":"a","meio":"b","fim":"c"},
"topicos":[{"topico":"Exit code","porque":"o CI decide por ele","obrigatorio":true,"situacao":"estudado"},{"topico":" ","porque":"","obrigatorio":false}],
"exemplo_central":"o pipe que esconde a falha","fora":"instalação","fontes":["go.dev"],"base":[{"fonte":"T13 › Fundamentos","situacao":"inventada"}]},
{"titulo":"  ","tese":"vazia"}],"notas":"Comece pelo primeiro."}`
	f.resposta = func(_ int, arq map[string]string) (string, error) {
		switch {
		case strings.Contains(arq["pedido.md"], "FALHE"):
			return "", errors.New("limite da assinatura")
		case strings.Contains(arq["pedido.md"], "# Pedido: tópicos"):
			return `{"tese":"t","titulo_melhor":"","topicos":[{"topico":"Workflow e job","porque":"p","obrigatorio":true,"situacao":"talvez","onde":""}],
"arco":{"comeco":"a","meio":"b","fim":"c"},"fora":"","armadilhas":["confundir job com step"],"perguntas":[],"fontes":[],"notas":"n"}`, nil
		}
		return sugestoes, nil
	}
	c := novoCliente(t)
	c.a.ia.chamar = f.chamar
	c.a.projeto = &cadernoProjeto{repo: "https://github.com/x/projeto", ramo: "main", tarball: gh.URL, pasta: t.TempDir(),
		intervalo: time.Hour, cliente: gh.Client()}

	// A trilha 13 com a etapa do caderno concluída e com lição aprovada.
	momento := agora()
	for _, q := range []string{
		`INSERT INTO trilhas (id, slug, titulo, criado_em, atualizado_em) VALUES (13, 'go', 'Go do zero', '` + momento + `', '` + momento + `')`,
		`INSERT INTO modulos (id, trilha_id, ordem, titulo) VALUES (1, 13, 0, 'Fundamentos')`,
		`INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo, status) VALUES (13, 1, 0, 'por-que-go', 'Por que Go e o ambiente', 'concluida')`,
	} {
		if _, err := c.a.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	licao := "# Por que Go\n\nTEXTO-DA-LICAO"
	if err := c.a.conteudo.gravar("lição", map[string]*string{caminhoLicao("go", "por-que-go"): &licao}); err != nil {
		t.Fatal(err)
	}

	// Erros de entrada.
	for _, corpo := range []map[string]any{{"modo": "outro"}, {"modo": "topicos"}, {"modo": "sugerir", "opcoes": map[string]any{"formato": "poema"}}} {
		if s := c.chamar("POST", "/api/projeto/itens/0.1/blog", corpo, nil); s != http.StatusBadRequest {
			t.Errorf("%v: %d", corpo, s)
		}
	}
	if s := c.chamar("POST", "/api/projeto/itens/9.9/blog", map[string]any{"modo": "sugerir"}, nil); s != http.StatusNotFound {
		t.Errorf("item inexistente: %d", s)
	}

	// Sugerir: 2 artigos, aprofundamento por padrão.
	var p pedidoBlog
	if s := c.chamar("POST", "/api/projeto/itens/0.1/blog", map[string]any{"modo": "sugerir", "titulo": "ignorado",
		"opcoes": map[string]any{"quantas": 2, "horizonte": "misto"}, "pedido": "Foco em CI."}, &p); s != http.StatusCreated {
		t.Fatalf("sugerir: %d", s)
	}
	if p.Estado != blogGerando || p.Titulo != "" || p.Opcoes.Nivel != "aprofundamento" || p.Opcoes.Quantas != 2 || p.Tarefa == nil {
		t.Fatalf("pedido: %+v", p)
	}
	if s := c.chamar("POST", "/api/projeto/itens/0.1/blog", map[string]any{"modo": "sugerir"}, nil); s != http.StatusBadRequest {
		t.Errorf("dois ao mesmo tempo: %d", s)
	}
	c.a.fila.esvaziar(context.Background())
	pedido := f.arquivos[0]["pedido.md"]
	for _, trecho := range []string{"# Pedido: artigos do blog a partir do item 0.1", "Sugira 2 artigo(s)", "**Repositório:** `banco`",
		"`caderno.md`", "T13 › Fundamentos › Por que Go e o ambiente (concluida) · lição em `licoes/01.md`",
		"núcleo atemporal + uma seção datada", "aprofundamento:", "Foco em CI."} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md sem %q:\n%s", trecho, pedido)
		}
	}
	if !strings.Contains(f.arquivos[0]["caderno.md"], "DECISÃO-DO-CADERNO") || !strings.Contains(f.arquivos[0]["licoes/01.md"], "TEXTO-DA-LICAO") ||
		!strings.Contains(f.pedidos[0].regra, "Quem escreve é ele") {
		t.Errorf("caderno, lição ou regra: %v", f.arquivos[0])
	}
	var lista []pedidoBlog
	c.ok("GET", "/api/projeto/itens/0.1/blog", nil, &lista)
	var res resultadoBlog
	json.Unmarshal(lista[0].Resultado, &res)
	if lista[0].Estado != blogPronta || len(res.Sugestoes) != 1 || len(res.Sugestoes[0].Alternativas) != 1 || len(res.Sugestoes[0].Topicos) != 1 ||
		res.Sugestoes[0].Topicos[0].Situacao != "" || res.Sugestoes[0].Base[0].Situacao != "fora_do_item" || res.Notas == "" {
		t.Fatalf("resultado: %+v %+v", lista[0], res)
	}

	// Marcar: guardar a primeira; a 2ª não existe.
	c.ok("PUT", "/api/blog/1/marcas/0", map[string]any{"estado": "guardada"}, &p)
	if p.Marcas["0"].Estado != "guardada" {
		t.Fatalf("marca: %+v", p.Marcas)
	}
	if s := c.chamar("PUT", "/api/blog/1/marcas/1", map[string]any{"estado": "guardada"}, nil); s != http.StatusBadRequest {
		t.Errorf("sugestão inexistente: %d", s)
	}
	if s := c.chamar("PUT", "/api/blog/1/marcas/0", map[string]any{"estado": "lida"}, nil); s != http.StatusBadRequest {
		t.Errorf("estado inválido: %d", s)
	}

	// Tópicos para um título: a ideia guardada vai no pedido; situação desconhecida vira fora_do_item.
	c.ok("POST", "/api/projeto/itens/0.1/blog", map[string]any{"modo": "topicos", "titulo": "GitHub Actions e o movimento ágil", "nota": "CI é cultura"}, &p)
	c.a.fila.esvaziar(context.Background())
	pedido = f.arquivos[1]["pedido.md"]
	for _, trecho := range []string{"# Pedido: tópicos para um título do blog", "> GitHub Actions e o movimento ágil", "> CI é cultura",
		"Bases sólidas: Go, Git e CI — O chão vem antes. [guardada · item 0.1]"} {
		if !strings.Contains(pedido, trecho) {
			t.Errorf("pedido.md (tópicos) sem %q:\n%s", trecho, pedido)
		}
	}
	p, _ = c.a.pedidoBlogPorID(context.Background(), p.ID)
	res = resultadoBlog{}
	json.Unmarshal(p.Resultado, &res)
	if p.Estado != blogPronta || len(res.Topicos) != 1 || res.Topicos[0].Situacao != "fora_do_item" || res.Arco == nil || len(res.Armadilhas) != 1 {
		t.Fatalf("tópicos: %+v %+v", p, res)
	}
	c.ok("PUT", "/api/blog/2/marcas/0", map[string]any{"estado": "escrita", "link": "https://lucaswesley.com/x"}, &p)
	var semMarca pedidoBlog
	c.ok("PUT", "/api/blog/2/marcas/0", map[string]any{"estado": ""}, &semMarca)
	if len(semMarca.Marcas) != 0 {
		t.Errorf("tirar a marca: %+v", p.Marcas)
	}

	// A falha fica registrada; o pedido com marca continua na lista depois de 5 novos.
	c.ok("POST", "/api/projeto/itens/0.1/blog", map[string]any{"modo": "sugerir", "pedido": "FALHE"}, &p)
	c.a.fila.esvaziar(context.Background())
	p, _ = c.a.pedidoBlogPorID(context.Background(), p.ID)
	if p.Estado != blogFalhou || !strings.Contains(p.Erro, "limite") {
		t.Fatalf("falha: %+v", p)
	}
	for i := 0; i < 5; i++ {
		c.ok("POST", "/api/projeto/itens/0.1/blog", map[string]any{"modo": "sugerir"}, nil)
		c.a.fila.esvaziar(context.Background())
	}
	c.ok("GET", "/api/projeto/itens/0.1/blog", nil, &lista)
	if len(lista) != 6 || lista[5].ID != 1 {
		t.Errorf("lista com a guardada: %d %+v", len(lista), lista[len(lista)-1].ID)
	}
}
