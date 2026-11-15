package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Mini-teste de nível (T9, fase 7): opcional na criação, a partir do mapa (entrada A) ou da lista
// e da espinha (B e C). A IA escreve de 8 a 12 perguntas de escolha espalhadas pelos itens, do
// básico ao avançado; o servidor corrige na hora e sugere onde começar e o nível de partida. É uma
// avaliação, não conteúdo: não vira proposta (aprovar as perguntas estragaria o teste). O Lucas
// decide se usa a sugestão (marca o recorte e o nível ao aprovar o mapa), e o resultado vai para o
// pedido do plano.

const tipoNivelamento = "nivelamento"

type perguntaNivel struct {
	Enunciado    string   `json:"enunciado"`
	Alternativas []string `json:"alternativas"`
	Correta      int      `json:"correta"`
	Explicacao   string   `json:"explicacao"`
	Item         string   `json:"item"`  // a chave do item do mapa ou da lista
	Nivel        string   `json:"nivel"` // basico · intermediario · avancado
}

type itemNivel struct {
	Chave   string `json:"chave"`
	Titulo  string `json:"titulo"`
	Nivel   string `json:"nivel,omitempty"`
	Acertos int    `json:"acertos"`
	Total   int    `json:"total"`
}

type resultadoNivel struct {
	Acertos       int                 `json:"acertos"`
	Total         int                 `json:"total"`
	PorNivel      map[string][2]int   `json:"por_nivel"` // nível → [acertos, total]
	Itens         []itemNivel         `json:"itens"`
	Dominados     []string            `json:"dominados"`  // itens com tudo certo
	ComecarEm     string              `json:"comecar_em"` // o primeiro item, na ordem, com erro
	NivelSugerido string              `json:"nivel_sugerido"`
	Correcoes     []correcaoExercicio `json:"correcoes"`
}

type nivelamento struct {
	Estado     string          `json:"estado"` // gerando · pronto · corrigido · falhou
	TarefaID   int64           `json:"tarefa_id,omitempty"`
	PropostaID int64           `json:"proposta_id"` // o mapa ou a lista de que saiu
	Erro       string          `json:"erro,omitempty"`
	Itens      []itemNivel     `json:"itens"`
	Perguntas  []perguntaNivel `json:"perguntas"`
	Respostas  []int           `json:"respostas,omitempty"` // -1 = não sei
	Resultado  *resultadoNivel `json:"resultado,omitempty"`
}

const esquemaNivelamento = `{"type":"object","properties":{
"perguntas":{"type":"array","items":{"type":"object","properties":{
  "enunciado":{"type":"string"},
  "alternativas":{"type":"array","items":{"type":"string"},"description":"de 3 a 5"},
  "correta":{"type":"integer","description":"o índice da certa, a partir de 0"},
  "explicacao":{"type":"string","description":"por que a certa é certa, mostrada depois"},
  "item":{"type":"string","description":"a chave do item do mapa ou da lista que a pergunta testa"},
  "nivel":{"type":"string","enum":["basico","intermediario","avancado"]}},
  "required":["enunciado","alternativas","correta","explicacao","item","nivel"]}}},
"required":["perguntas"]}`

func (a *app) nivelamentoDa(ctx context.Context, trilhaID int64) *nivelamento {
	var bruto string
	if err := a.db.QueryRowContext(ctx, `SELECT nivelamento FROM criacoes WHERE trilha_id = ?`, trilhaID).Scan(&bruto); err != nil || bruto == "" {
		return nil
	}
	var n nivelamento
	if json.Unmarshal([]byte(bruto), &n) != nil {
		return nil
	}
	return &n
}

func (a *app) salvarNivelamento(ctx context.Context, trilhaID int64, n *nivelamento) error {
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE criacoes SET nivelamento = ?, atualizado_em = ? WHERE trilha_id = ?`, string(b), agora(), trilhaID)
	return err
}

// paraTela: sem o gabarito enquanto ele não respondeu.
func (n *nivelamento) paraTela() *nivelamento {
	if n == nil {
		return nil
	}
	c := *n
	if c.Estado != "corrigido" {
		c.Perguntas = make([]perguntaNivel, len(n.Perguntas))
		for i, p := range n.Perguntas {
			c.Perguntas[i] = perguntaNivel{Enunciado: p.Enunciado, Alternativas: p.Alternativas, Correta: -1, Item: p.Item, Nivel: p.Nivel}
		}
	}
	return &c
}

// itensParaNivelamento: os itens do mapa (com o nível de cada um) ou da lista/espinha, da
// proposta que aguarda o Lucas ou da que ele aprovou.
func (a *app) itensParaNivelamento(ctx context.Context, c criacao) (int64, []itemNivel, error) {
	var tipo string
	var aprovada int64
	switch c.Passo {
	case tipoMapa, tipoValidacao:
		tipo = c.Passo
	case tipoPlano:
		if c.Escolhas.MapaID > 0 {
			tipo, aprovada = tipoMapa, c.Escolhas.MapaID
		} else {
			tipo, aprovada = tipoValidacao, c.Escolhas.ValidacaoID
		}
	default:
		return 0, nil, erroDeEntrada{errors.New("o mini-teste sai do mapa ou da lista: espere a IA montá-los")}
	}
	var p proposta
	var err error
	if aprovada > 0 {
		p, err = a.propostaPorID(ctx, aprovada)
	} else {
		p, err = lerPropostaLinha(a.db.QueryRowContext(ctx, `SELECT `+colunasProposta+` FROM propostas
			WHERE trilha_id = ? AND etapa_chave = '' AND tipo = ? AND estado = ? ORDER BY versao DESC LIMIT 1`, c.TrilhaID, tipo, propAguardando))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, erroDeEntrada{errors.New("o mini-teste sai do mapa ou da lista: espere a IA montá-los")}
	}
	if err != nil {
		return 0, nil, err
	}
	var itens []itemNivel
	if tipo == tipoMapa {
		var cm conteudoMapa
		if err := json.Unmarshal(p.Conteudo, &cm); err != nil {
			return 0, nil, err
		}
		for _, m := range cm.Mapa {
			for _, e := range m.Etapas {
				itens = append(itens, itemNivel{Chave: e.Chave, Titulo: e.Titulo, Nivel: e.Nivel})
			}
		}
	} else {
		var cv conteudoValidacao
		if err := json.Unmarshal(p.Conteudo, &cv); err != nil {
			return 0, nil, err
		}
		for _, it := range cv.Itens {
			itens = append(itens, itemNivel{Chave: it.Chave, Titulo: it.Titulo})
		}
	}
	if len(itens) == 0 {
		return 0, nil, erroDeEntrada{errors.New("o mapa ou a lista está vazio")}
	}
	return p.ID, itens, nil
}

func (a *app) handlePedirNivelamento(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	c, err := a.criacaoDa(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if n := a.nivelamentoDa(ctx, id); n != nil && n.Estado == "gerando" {
		a.responderErro(w, erroDeEntrada{errors.New("a IA já está escrevendo o mini-teste")})
		return
	}
	propostaID, itens, err := a.itensParaNivelamento(ctx, c)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	modelo := a.modeloPara(ctx, tipoNivelamento)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoNivelamento, 0, "Mini-teste de nível: "+c.Tema, modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.salvarNivelamento(ctx, id, &nivelamento{Estado: "gerando", TarefaID: tarefaID, PropostaID: propostaID, Itens: itens}); err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	a.responderCriacao(w, r, id, http.StatusCreated)
}

// criacaoDaTarefa: a criação cujo mini-teste é esta tarefa.
func (a *app) criacaoDaTarefa(ctx context.Context, tarefaID int64) (int64, error) {
	var id int64
	err := a.db.QueryRowContext(ctx, `SELECT trilha_id FROM criacoes WHERE nivelamento != '' AND json_extract(nivelamento, '$.tarefa_id') = ?`, tarefaID).Scan(&id)
	return id, err
}

func (a *app) gerarNivelamento(ctx context.Context, t *tarefa) error {
	trilhaID, err := a.criacaoDaTarefa(ctx, t.ID)
	if err != nil {
		return errors.New("a criação deste mini-teste não existe mais")
	}
	c, err := a.criacaoDa(ctx, trilhaID)
	if err != nil {
		return err
	}
	n := a.nivelamentoDa(ctx, trilhaID)
	if n == nil {
		return errors.New("o mini-teste foi apagado")
	}
	tr, err := a.trilhaCompleta(ctx, trilhaID)
	if err != nil {
		return err
	}
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: mini-teste de nível\n\n- Tema: %s\n", c.Tema)
	if c.Pedido != "" {
		escreva("- O que o Lucas quer com isto: %s\n", c.Pedido)
	}
	escreva("- Profundidade escolhida: %s\n", nomesProfundidade[tr.Profundidade])
	if tr.NivelPartida != "" {
		escreva("- Nível de partida que ele declarou ou a IA deduziu: %s\n", tr.NivelPartida)
	}
	escreva("\n## Os itens (na ordem de estudo)\n\nCada pergunta testa um item: use a chave em `item`.\n\n")
	for _, it := range n.Itens {
		escreva("- `%s` — %s%s\n", it.Chave, it.Titulo, entreParenteses(it.Nivel))
	}
	escreva("\n## Instruções do Lucas\n\n- Globais: %s\n- Da trilha: %s\n", padrao(cfg.InstrucoesGlobais, "(nenhuma)"), padrao(tr.Instrucoes, "(nenhuma)"))
	if err := os.WriteFile(filepath.Join(pasta, "pedido.md"), []byte(s.String()), 0o644); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("nivelamento.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read", esquema: esquemaNivelamento, modelo: t.Modelo,
		instrucao: "Escreva um mini-teste de nível. Tudo o que você precisa saber está em `pedido.md`, nesta pasta: leia-o inteiro primeiro.",
	})
	if err != nil {
		return err
	}
	var x struct {
		Perguntas []perguntaNivel `json:"perguntas"`
	}
	if err := dadosDaResposta(resp, &x); err != nil {
		return err
	}
	n.Perguntas = limparNivelamento(x.Perguntas, n.Itens)
	if len(n.Perguntas) < 3 {
		return errors.New("o Claude não devolveu perguntas bem formadas para o mini-teste")
	}
	n.Estado, n.Erro, n.Respostas, n.Resultado = "pronto", "", nil, nil
	return a.salvarNivelamento(ctx, trilhaID, n)
}

func limparNivelamento(lista []perguntaNivel, itens []itemNivel) []perguntaNivel {
	existe := map[string]bool{}
	for _, it := range itens {
		existe[it.Chave] = true
	}
	var boas []perguntaNivel
	for _, p := range lista {
		p.Enunciado, p.Explicacao = strings.TrimSpace(p.Enunciado), textoDaIA(p.Explicacao)
		if p.Enunciado == "" || len(p.Alternativas) < 2 || p.Correta < 0 || p.Correta >= len(p.Alternativas) || !existe[p.Item] {
			continue
		}
		if !contem([]string{"basico", "intermediario", "avancado"}, p.Nivel) {
			p.Nivel = "intermediario"
		}
		boas = append(boas, p)
		if len(boas) == 15 {
			break
		}
	}
	return boas
}

// aoTerminarNivelamento: falhou ou foi cancelado, o mini-teste fica com o erro (e pode ser pedido de novo).
func (a *app) aoTerminarNivelamento(t *tarefa) {
	ctx := context.Background()
	id, err := a.criacaoDaTarefa(ctx, t.ID)
	if err != nil {
		return
	}
	if n := a.nivelamentoDa(ctx, id); n != nil && n.Estado == "gerando" {
		n.Estado, n.Erro = "falhou", padrao(t.Erro, "cancelado")
		if err := a.salvarNivelamento(ctx, id, n); err != nil {
			log.Printf("mini-teste da trilha %d: %v", id, err)
		}
	}
}

func (a *app) handleResponderNivelamento(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Respostas []int `json:"respostas"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	n := a.nivelamentoDa(ctx, id)
	if n == nil || (n.Estado != "pronto" && n.Estado != "corrigido") {
		a.responderErro(w, erroDeEntrada{errors.New("o mini-teste não está pronto")})
		return
	}
	if len(corpo.Respostas) != len(n.Perguntas) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("vieram %d respostas para %d perguntas; recarregue a página", len(corpo.Respostas), len(n.Perguntas))})
		return
	}
	n.Respostas = corpo.Respostas
	n.Resultado = corrigirNivelamento(n.Itens, n.Perguntas, corpo.Respostas)
	n.Estado = "corrigido"
	if err := a.salvarNivelamento(ctx, id, n); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderCriacao(w, r, id, http.StatusOK)
}

// corrigirNivelamento: acertos por item e por nível; onde começar = o primeiro item, na ordem,
// com erro; nível sugerido pelo que ele acertou em cada nível (60% conta como sabe).
func corrigirNivelamento(itens []itemNivel, perguntas []perguntaNivel, respostas []int) *resultadoNivel {
	res := &resultadoNivel{PorNivel: map[string][2]int{}, Dominados: []string{}, Itens: []itemNivel{}}
	porItem := map[string]*itemNivel{}
	for _, it := range itens {
		x := it
		x.Acertos, x.Total = 0, 0
		porItem[it.Chave] = &x
	}
	for i, p := range perguntas {
		escolha := respostas[i]
		certa := escolha == p.Correta
		c := correcaoExercicio{Certa: certa, Enunciado: p.Enunciado, Correta: p.Alternativas[p.Correta], Explicacao: p.Explicacao, Conceito: p.Item}
		if escolha >= 0 && escolha < len(p.Alternativas) {
			c.Resposta = p.Alternativas[escolha]
		}
		res.Correcoes = append(res.Correcoes, c)
		res.Total++
		pn := res.PorNivel[p.Nivel]
		pn[1]++
		if certa {
			res.Acertos++
			pn[0]++
		}
		res.PorNivel[p.Nivel] = pn
		if it := porItem[p.Item]; it != nil {
			it.Total++
			if certa {
				it.Acertos++
			}
		}
	}
	for _, it := range itens {
		x := porItem[it.Chave]
		if x.Total == 0 {
			continue
		}
		res.Itens = append(res.Itens, *x)
		if x.Acertos == x.Total {
			res.Dominados = append(res.Dominados, x.Chave)
		} else if res.ComecarEm == "" {
			res.ComecarEm = x.Chave
		}
	}
	sabe := func(nivel string) bool {
		pn, ok := res.PorNivel[nivel]
		return !ok || pn[1] == 0 || pn[0]*100 >= pn[1]*60
	}
	switch {
	case !sabe("basico"):
		res.NivelSugerido = "iniciante"
	case !sabe("intermediario"):
		res.NivelSugerido = "básico"
	case !sabe("avancado"):
		res.NivelSugerido = "intermediário"
	default:
		res.NivelSugerido = "avançado"
	}
	return res
}

// escreverNivelamento: o resultado no pedido do plano.
func escreverNivelamento(s *strings.Builder, n *nivelamento) {
	if n == nil || n.Estado != "corrigido" || n.Resultado == nil {
		return
	}
	r := n.Resultado
	fmt.Fprintf(s, "\n## Mini-teste de nível (ele fez)\n\nAcertou %d de %d. Nível sugerido pelo teste: %s.\n\n", r.Acertos, r.Total, r.NivelSugerido)
	for _, it := range r.Itens {
		fmt.Fprintf(s, "- `%s` %s: %d de %d\n", it.Chave, it.Titulo, it.Acertos, it.Total)
	}
	fmt.Fprintf(s, "\nO que ele acertou inteiro pode ir mais rápido (etapa mais curta, ou só retomada na lição seguinte); onde errou precisa da etapa completa. O recorte e o nível que ele confirmou valem mais que o teste.\n")
}
