package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Reforço (fase 7, T28; PEDAGOGIA §1, "domínio antes de avançar"): numa etapa em que o Lucas
// errou muito, a IA propõe etapas de reforço, com o diagnóstico do que falta. É uma proposta da
// etapa como as outras (pedido, comentário geral ou em etapa:N, ajuste); aprovar insere as
// marcadas no mesmo módulo, antes ou depois da etapa, do tipo "reforco" e com o foco que a lição
// delas deve trabalhar. A lição de cada uma é gerada depois, como a de qualquer etapa.

const tipoReforco = "reforco"

type etapaDeReforco struct {
	Titulo   string `json:"titulo"`
	Objetivo string `json:"objetivo"`
	Bloom    string `json:"bloom"`
	Minutos  int    `json:"minutos"`
	Posicao  string `json:"posicao"` // antes · depois (da etapa reforçada)
	Foco     string `json:"foco"`    // o que a lição dela deve trabalhar
	Motivo   string `json:"motivo"`
}

type conteudoReforco struct {
	Diagnostico string               `json:"diagnostico"`
	Etapas      []etapaDeReforco     `json:"etapas"`
	Notas       string               `json:"notas,omitempty"`
	Respostas   []respostaComentario `json:"respostas,omitempty"`
}

const esquemaReforco = `{"type":"object","properties":{
"diagnostico":{"type":"string","description":"o que ele não domina e por quê (a causa provável: um pré-requisito, uma confusão entre dois conceitos, falta de prática), em poucas frases"},
"etapas":{"type":"array","items":{"type":"object","properties":{
  "titulo":{"type":"string"},
  "objetivo":{"type":"string","description":"uma frase com o verbo de Bloom"},
  "bloom":{"type":"string","enum":["lembrar","entender","aplicar","analisar","avaliar","criar"]},
  "minutos":{"type":"integer"},
  "posicao":{"type":"string","enum":["antes","depois"],"description":"antes: falta uma base que a etapa supõe; depois: reforçar o que a etapa ensina"},
  "foco":{"type":"string","description":"o que a lição desta etapa deve trabalhar, concretamente (vai para quem gerar a lição)"},
  "motivo":{"type":"string","description":"qual erro dele esta etapa resolve"}},
  "required":["titulo","objetivo","bloom","minutos","posicao","foco","motivo"]}},
"notas":{"type":"string"},` + esquemaRespostas + `},
"required":["diagnostico","etapas","notas","respostas"]}`

func (a *app) handleReforco(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pedido string `json:"pedido"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	pedido := strings.TrimSpace(corpo.Pedido)
	if pedido == "" {
		if pontos, _ := a.pontosFracosDaEtapa(r.Context(), e); len(pontos) == 0 {
			a.responderErro(w, erroDeEntrada{errors.New("a etapa não tem erros registrados: diga no pedido o que reforçar")})
			return
		}
	}
	p, err := a.novaProposta(r.Context(), e.TrilhaID, e.Chave, tipoReforco, pedido, 0, "Reforço: "+e.Titulo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, p)
}

func (a *app) gerarReforco(ctx context.Context, t *tarefa) error {
	p, err := a.propostaPorID(ctx, t.PropostaID)
	if err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, p.TrilhaID)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	arquivos, comentarios, err := a.montarPedidoReforco(ctx, tr, p)
	if err != nil {
		return err
	}
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("reforco.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read,Glob,Grep", esquema: esquemaReforco, modelo: t.Modelo,
		instrucao: "Proponha o reforço de uma etapa de uma trilha de estudo em que o Lucas errou. Tudo o que você precisa saber está em " +
			"`pedido.md`, nesta pasta: leia-o inteiro primeiro.",
	})
	if err != nil {
		return err
	}
	var c conteudoReforco
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	c.Etapas = limparReforco(c.Etapas)
	c.Diagnostico = textoDaIA(c.Diagnostico)
	if len(c.Etapas) == 0 {
		return errors.New("o Claude não devolveu nenhuma etapa de reforço")
	}
	bruto, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return a.guardarProposta(ctx, p, string(bruto), c.Respostas, comentarios)
}

// limparReforco: título obrigatório, no máximo três etapas, valores dentro do que o app aceita.
func limparReforco(lista []etapaDeReforco) []etapaDeReforco {
	var boas []etapaDeReforco
	for _, x := range lista {
		x.Titulo, x.Objetivo, x.Foco, x.Motivo = strings.TrimSpace(x.Titulo), strings.TrimSpace(x.Objetivo), textoDaIA(x.Foco), textoDaIA(x.Motivo)
		if x.Titulo == "" {
			continue
		}
		if !contem(niveisBloom, x.Bloom) {
			x.Bloom = ""
		}
		if x.Posicao != "antes" {
			x.Posicao = "depois"
		}
		if x.Minutos <= 0 {
			x.Minutos = 30
		}
		x.Minutos = min(x.Minutos, 240)
		boas = append(boas, x)
		if len(boas) == 3 {
			break
		}
	}
	return boas
}

func (a *app) montarPedidoReforco(ctx context.Context, tr trilha, p proposta) (map[string]string, map[int64]bool, error) {
	var e etapa
	var modulo string
	for _, m := range tr.Modulos {
		for _, x := range m.Etapas {
			if x.Chave == p.EtapaChave {
				e, modulo = x, m.Titulo
			}
		}
	}
	if e.ID == 0 {
		return nil, nil, errors.New("a etapa desta proposta não existe mais na trilha")
	}
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, nil, err
	}
	profundidade := e.Profundidade
	if profundidade == 0 {
		profundidade = tr.Profundidade
	}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: reforço da etapa \"%s\"\n\n", e.Titulo)
	escreva("## A trilha\n\n- Título: %s\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
		if strings.TrimSpace(campo[1]) != "" {
			escreva("- %s: %s\n", campo[0], campo[1])
		}
	}
	escreva("\n## A etapa\n\n- Título: %s\n- Módulo: %s\n", e.Titulo, modulo)
	if e.Objetivo != "" {
		escreva("- Objetivo: %s\n", e.Objetivo)
	}
	if b, ok := nomesBloom[e.Bloom]; ok {
		escreva("- Nível de Bloom do objetivo: %s\n", b)
	}
	escreva("- Profundidade: %s\n", nomesProfundidade[profundidade])
	escreva("- Situação: %s\n", e.Status)

	arquivos := map[string]string{}
	if licao, err := a.conteudo.ler(caminhoLicao(tr.Slug, e.Chave)); err == nil && strings.TrimSpace(licao) != "" {
		arquivos["licao.md"] = licao
		escreva("\nA lição aprovada da etapa está em `licao.md`. O reforço não repete a lição: ataca o que ele errou, de outro jeito.\n")
	} else {
		escreva("\nA etapa ainda não tem lição aprovada.\n")
	}

	pontos, sinais := a.pontosFracosDaEtapa(ctx, e)
	escreva("\n## Como ele foi\n\n")
	if sinais != nil {
		for _, x := range []struct {
			nome string
			r    *resumoNotas
		}{{"Exercícios", sinais.Exercicios}, {"Simulado", sinais.Simulado}, {"Revisões", sinais.Revisao}} {
			if x.r != nil {
				escreva("- %s: última nota %d, melhor %d, %d vez(es)\n", x.nome, x.r.Ultima, x.r.Melhor, x.r.Vezes)
			}
		}
	}
	escreva("- Nota mínima (critério): %d%%\n", a.notaMinima(ctx))
	escreva("\n## O que ele errou da última vez (pontos fracos)\n\n")
	if len(pontos) == 0 {
		escreva("(Nada registrado: o pedido abaixo diz o que reforçar.)\n")
	}
	for i, x := range pontos {
		escreva("%d. **%s** (errou %d, acertou %d; em %s)\n", i+1, x.Conceito, x.Erros, x.Acertos, x.Onde)
		if x.Exemplo != "" && x.Exemplo != x.Conceito {
			escreva("   - Pergunta: %s\n", x.Exemplo)
		}
		escreva("   - Resposta dele: %s\n", padrao(x.Resposta, "(em branco)"))
		if x.Esperado != "" {
			escreva("   - Esperado / o que faltou: %s\n", x.Esperado)
		}
	}

	escreva("\n## A trilha inteira, na ordem (não proponha o que outra etapa já cobre)\n\n")
	for _, m := range tr.Modulos {
		escreva("- %s\n", m.Titulo)
		for _, x := range m.Etapas {
			marca := ""
			if x.ID == e.ID {
				marca = "  ← esta"
			}
			escreva("  - %s%s%s\n", x.Titulo, entreParenteses(map[string]string{"base": "base", "reforco": "reforço"}[x.Tipo]), marca)
		}
	}

	escreva("\n## Instruções do Lucas\n\nVão da mais permanente à mais pontual; em conflito, vale a mais específica.\n\n")
	escreva("- Globais: %s\n", padrao(cfg.InstrucoesGlobais, "(nenhuma)"))
	escreva("- Da trilha: %s\n", padrao(tr.Instrucoes, "(nenhuma)"))
	escreva("- Deste pedido: %s\n", padrao(p.Pedido, "(nenhuma)"))

	comentarios := map[int64]bool{}
	if p.AnteriorID > 0 {
		ant, err := a.propostaPorID(ctx, p.AnteriorID)
		if err != nil {
			return nil, nil, err
		}
		if err := a.completar(ctx, &ant); err != nil {
			return nil, nil, err
		}
		arquivos["versao-anterior.json"] = string(ant.Conteudo)
		escreva("\n## É um ajuste da versão %d\n\n", ant.Versao)
		escreva("A versão anterior está em `versao-anterior.json`. Parta dela: mude o que os comentários pedem e mantenha o resto. Os comentários numa etapa proposta dizem `etapa:N` (N a partir de 1, na ordem da versão anterior). Responda **cada** comentário em `respostas`, pelo número.\n\n")
		for _, c := range ant.Comentarios {
			if c.Autor != "lucas" {
				continue
			}
			comentarios[c.ID] = true
			escreva("- Comentário %d", c.ID)
			if c.Ancora != "" {
				escreva(" (em %s)", c.Ancora)
			}
			escreva(": %s\n", c.Texto)
		}
		if len(comentarios) == 0 {
			escreva("(Sem comentários: o pedido acima diz o que ajustar.)\n")
		}
	}
	arquivos["pedido.md"] = s.String()
	return arquivos, comentarios, nil
}

// aprovarReforco insere as etapas marcadas no módulo da etapa reforçada: as "antes" logo antes
// dela (com os mesmos pré-requisitos, e viram pré-requisito dela se ela ainda não foi concluída),
// as "depois" logo depois (com ela de pré-requisito). Levam o material da etapa como apoio.
func (a *app) aprovarReforco(w http.ResponseWriter, r *http.Request, p proposta) {
	var corpo struct {
		Aceitos []int `json:"aceitos"`
	}
	if r.ContentLength != 0 {
		if err := lerCorpo(r, &corpo); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	ctx := r.Context()
	alvo, err := a.etapaPorChave(ctx, p.TrilhaID, p.EtapaChave)
	if err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa desta proposta não existe mais na trilha")})
		return
	}
	if err := a.semVersaoGerando(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	var c conteudoReforco
	if err := json.Unmarshal(p.Conteudo, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(corpo.Aceitos) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("marque as etapas que entram na trilha (ou descarte a proposta)")})
		return
	}
	var antes, depois []etapaDeReforco
	vistos := map[int]bool{}
	for _, i := range corpo.Aceitos {
		if i < 0 || i >= len(c.Etapas) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("etapa %d não está na proposta", i+1)})
			return
		}
		if vistos[i] {
			continue
		}
		vistos[i] = true
		if c.Etapas[i].Posicao == "antes" {
			antes = append(antes, c.Etapas[i])
		} else {
			depois = append(depois, c.Etapas[i])
		}
	}
	apoio := []material{}
	for _, m := range alvo.Materiais {
		m.Papel = "apoio"
		apoio = append(apoio, m)
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	inserir := func(x etapaDeReforco, ordem int, requer []int64, antesDela bool) (string, error) {
		nova := etapa{TrilhaID: alvo.TrilhaID, Titulo: x.Titulo, Objetivo: x.Objetivo, Bloom: x.Bloom, Tipo: tipoReforco,
			Minutos: x.Minutos, Foco: x.Foco, Requer: requer, Materiais: apoio}
		if err := validarEtapa(&nova); err != nil {
			return "", erroDeEntrada{err}
		}
		chave, err := slugLivre(ctx, tx, slugificar(x.Titulo), `SELECT COUNT(*) FROM etapas WHERE trilha_id = ? AND chave = ?`, alvo.TrilhaID)
		if err != nil {
			return "", err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE etapas SET ordem = ordem + 1 WHERE modulo_id = ? AND ordem >= ?`, alvo.ModuloID, ordem); err != nil {
			return "", err
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo) VALUES (?, ?, ?, ?, ?)`,
			alvo.TrilhaID, alvo.ModuloID, ordem, chave, nova.Titulo)
		if err != nil {
			return "", err
		}
		if nova.ID, err = res.LastInsertId(); err != nil {
			return "", err
		}
		if err := a.salvarEtapa(ctx, tx, nova); err != nil {
			return "", err
		}
		if antesDela {
			// Vira pré-requisito da etapa, se ela ainda não foi feita.
			if alvo.Status != "concluida" && alvo.Status != "pulada" {
				if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO etapa_requisitos (etapa_id, requer_id) VALUES (?, ?)`, alvo.ID, nova.ID); err != nil {
					return "", err
				}
			}
		}
		return chave, nil
	}
	ordemAlvo := alvo.Ordem
	var chaves []string
	for _, x := range antes {
		chave, err := inserir(x, ordemAlvo, append([]int64{}, alvo.Requer...), true)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		chaves = append(chaves, chave)
		ordemAlvo++
	}
	for i, x := range depois {
		chave, err := inserir(x, ordemAlvo+1+i, []int64{alvo.ID}, false)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		chaves = append(chaves, chave)
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(ctx, alvo.TrilhaID)
	if err := a.exportar(ctx, alvo.TrilhaID, fmt.Sprintf("reforço da etapa %s: %s (proposta v%d aprovada)", alvo.Chave, strings.Join(chaves, ", "), p.Versao), nil); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.marcarAprovada(ctx, p); err != nil {
		a.responderErro(w, err)
		return
	}
	a.responderProposta(w, r, p.ID, http.StatusOK)
}
