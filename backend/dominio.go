package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Fase 7 (T28): domínio e revisão.
//
//   - Revisão espaçada: toda etapa concluída volta depois com perguntas rápidas (sorteadas dos
//     exercícios dela, corrigidas pelo servidor, sem a IA; sem exercícios, o Lucas se avalia).
//     Acertou (nota mínima), o intervalo sobe um degrau; foi mais ou menos, repete o degrau; errou,
//     volta para o primeiro.
//   - Simulado intercalado: perguntas abertas dos bancos de várias etapas, misturadas (as etapas
//     fracas primeiro); a IA corrige na fila, como o simulado da etapa.
//   - Pontos fracos: o que ele errou da última vez, por conceito, juntando exercícios, simulado,
//     revisão e intercalado. Sem IA: é contagem.
//   - Reforço: proposta da IA de etapas novas para uma etapa fraca (reforco.go).
//   - Painel: tudo isso por trilha.

const (
	tipoRevisao     = "revisao"
	tipoIntercalado = "intercalado"
)

// intervalosRevisao: a escada, em dias. O degrau 0 é o primeiro (um dia depois de concluir).
var intervalosRevisao = []int{1, 3, 7, 14, 30, 60, 120}

const perguntasPorRevisao = 5

type revisao struct {
	EtapaChave string `json:"etapa_chave"`
	Nivel      int    `json:"nivel"`
	Proxima    string `json:"proxima"`
	Ultima     string `json:"ultima"`
	UltimaNota int    `json:"ultima_nota"` // -1 = ainda não houve
	Intervalo  int    `json:"intervalo"`   // dias do degrau atual
	Pendente   bool   `json:"pendente"`    // a próxima já chegou
}

func diaDe(momento string) time.Time {
	t, err := time.Parse(time.RFC3339, momento)
	if err != nil {
		return hoje()
	}
	l := t.In(fuso)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, fuso)
}

// garantirRevisoes: toda etapa concluída ganha a sua revisão (a primeira, no dia seguinte ao que
// foi concluída). trilhaID 0 = todas.
func (a *app) garantirRevisoes(ctx context.Context, trilhaID int64) error {
	rows, err := a.db.QueryContext(ctx, `SELECT e.trilha_id, e.chave, e.concluido_em FROM etapas e
		LEFT JOIN revisoes r ON r.trilha_id = e.trilha_id AND r.etapa_chave = e.chave
		WHERE e.status = 'concluida' AND r.etapa_chave IS NULL AND (? = 0 OR e.trilha_id = ?)`, trilhaID, trilhaID)
	if err != nil {
		return err
	}
	type nova struct {
		trilha  int64
		chave   string
		proxima string
	}
	var novas []nova
	for rows.Next() {
		var n nova
		var concluido string
		if err := rows.Scan(&n.trilha, &n.chave, &concluido); err != nil {
			rows.Close()
			return err
		}
		n.proxima = diaDe(concluido).AddDate(0, 0, intervalosRevisao[0]).Format(formatoData)
		novas = append(novas, n)
	}
	rows.Close()
	for _, n := range novas {
		if _, err := a.db.ExecContext(ctx, `INSERT OR IGNORE INTO revisoes (trilha_id, etapa_chave, nivel, proxima) VALUES (?, ?, 0, ?)`,
			n.trilha, n.chave, n.proxima); err != nil {
			return err
		}
	}
	return nil
}

func completarRevisao(r *revisao) {
	r.Intervalo = intervalosRevisao[min(max(r.Nivel, 0), len(intervalosRevisao)-1)]
	r.Pendente = r.Proxima <= hoje().Format(formatoData)
}

func (a *app) revisaoDa(ctx context.Context, trilhaID int64, chave string) (*revisao, error) {
	var r revisao
	err := a.db.QueryRowContext(ctx, `SELECT etapa_chave, nivel, proxima, ultima, ultima_nota FROM revisoes WHERE trilha_id = ? AND etapa_chave = ?`,
		trilhaID, chave).Scan(&r.EtapaChave, &r.Nivel, &r.Proxima, &r.Ultima, &r.UltimaNota)
	if err != nil {
		return nil, err // sem linha: a etapa não está concluída
	}
	completarRevisao(&r)
	return &r, nil
}

func (a *app) revisoesDaTrilha(ctx context.Context, trilhaID int64) (map[string]*revisao, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT etapa_chave, nivel, proxima, ultima, ultima_nota FROM revisoes WHERE trilha_id = ?`, trilhaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := map[string]*revisao{}
	for rows.Next() {
		var r revisao
		if err := rows.Scan(&r.EtapaChave, &r.Nivel, &r.Proxima, &r.Ultima, &r.UltimaNota); err != nil {
			return nil, err
		}
		completarRevisao(&r)
		lista[r.EtapaChave] = &r
	}
	return lista, rows.Err()
}

type revisaoPendente struct {
	TrilhaID     int64  `json:"trilha_id"`
	TrilhaTitulo string `json:"trilha_titulo"`
	EtapaID      int64  `json:"etapa_id"`
	EtapaTitulo  string `json:"etapa_titulo"`
	Chave        string `json:"chave"`
	Proxima      string `json:"proxima"`
	Nivel        int    `json:"nivel"`
	Atraso       int    `json:"atraso"` // dias depois do previsto
	Exercicios   bool   `json:"exercicios"`
}

// revisoesPendentes: as que já chegaram, nas trilhas ativas (trilhaID 0) ou numa trilha.
func (a *app) revisoesPendentes(ctx context.Context, trilhaID int64) ([]revisaoPendente, error) {
	if err := a.garantirRevisoes(ctx, trilhaID); err != nil {
		return nil, err
	}
	dia := hoje()
	rows, err := a.db.QueryContext(ctx, `SELECT r.trilha_id, t.titulo, t.slug, e.id, e.titulo, e.chave, r.proxima, r.nivel
		FROM revisoes r JOIN etapas e ON e.trilha_id = r.trilha_id AND e.chave = r.etapa_chave
		JOIN trilhas t ON t.id = r.trilha_id JOIN modulos m ON m.id = e.modulo_id
		WHERE e.status = 'concluida' AND r.proxima <= ? AND (? = 0 AND t.estado = 'ativa' OR r.trilha_id = ?)
		ORDER BY r.proxima, t.id, m.ordem, e.ordem`, dia.Format(formatoData), trilhaID, trilhaID)
	if err != nil {
		return nil, err
	}
	lista := []revisaoPendente{}
	slugs := []string{}
	for rows.Next() {
		var x revisaoPendente
		var slug string
		if err := rows.Scan(&x.TrilhaID, &x.TrilhaTitulo, &slug, &x.EtapaID, &x.EtapaTitulo, &x.Chave, &x.Proxima, &x.Nivel); err != nil {
			rows.Close()
			return nil, err
		}
		if d, err := lerData(x.Proxima); err == nil {
			x.Atraso = int(dia.Sub(d).Hours() / 24)
		}
		lista = append(lista, x)
		slugs = append(slugs, slug)
	}
	rows.Close()
	for i := range lista {
		lista[i].Exercicios = len(a.lerExercicios(slugs[i], lista[i].Chave)) > 0
	}
	return lista, rows.Err()
}

func (a *app) handleRevisoes(w http.ResponseWriter, r *http.Request) {
	lista, err := a.revisoesPendentes(r.Context(), 0)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"revisoes": lista, "hoje": hoje().Format(formatoData)})
}

// ---------- fazer a revisão de uma etapa ----------

type perguntaRevisao struct {
	Indice int `json:"indice"` // no conjunto de exercícios da etapa
	exercicioSemGabarito
}

// escolherParaRevisao: até n exercícios, os que ele errou antes primeiro, o resto sorteado.
func escolherParaRevisao(exs []exercicio, errados map[string]bool, n int) []int {
	var antes, resto []int
	for i, x := range exs {
		if errados[normalizarResposta(x.Enunciado)] {
			antes = append(antes, i)
		} else {
			resto = append(resto, i)
		}
	}
	rand.Shuffle(len(antes), func(i, j int) { antes[i], antes[j] = antes[j], antes[i] })
	rand.Shuffle(len(resto), func(i, j int) { resto[i], resto[j] = resto[j], resto[i] })
	escolhidos := append(antes, resto...)
	if len(escolhidos) > n {
		escolhidos = escolhidos[:n]
	}
	sort.Ints(escolhidos)
	return escolhidos
}

// erradosDaEtapa: os enunciados que ele errou na última vez que respondeu (exercícios e revisão).
func (a *app) erradosDaEtapa(ctx context.Context, trilhaID int64, chave string) map[string]bool {
	errados := map[string]bool{}
	rows, err := a.db.QueryContext(ctx, `SELECT resultado FROM tentativas WHERE trilha_id = ? AND etapa_chave = ?
		AND tipo IN (?, ?) AND estado = ? ORDER BY id`, trilhaID, chave, tipoExercicios, tipoRevisao, tentCorrigida)
	if err != nil {
		return errados
	}
	defer rows.Close()
	for rows.Next() {
		var resultado string
		if rows.Scan(&resultado) != nil {
			continue
		}
		var cs []correcaoExercicio
		if json.Unmarshal([]byte(resultado), &cs) != nil {
			continue
		}
		for _, c := range cs {
			errados[normalizarResposta(c.Enunciado)] = !c.Certa
		}
	}
	return errados
}

func (a *app) handleRevisaoDaEtapa(w http.ResponseWriter, r *http.Request) {
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if err := a.garantirRevisoes(ctx, e.TrilhaID); err != nil {
		a.responderErro(w, err)
		return
	}
	rv, _ := a.revisaoDa(ctx, e.TrilhaID, e.Chave)
	slug := a.slugDaTrilha(ctx, e.TrilhaID)
	exs := a.lerExercicios(slug, e.Chave)
	perguntas := []perguntaRevisao{}
	for _, i := range escolherParaRevisao(exs, a.erradosDaEtapa(ctx, e.TrilhaID, e.Chave), perguntasPorRevisao) {
		x := exs[i]
		perguntas = append(perguntas, perguntaRevisao{i, exercicioSemGabarito{x.Tipo, x.Enunciado, x.Alternativas, x.Conceito}})
	}
	secoes := []string{}
	if len(perguntas) == 0 {
		if licao, err := a.conteudo.ler(caminhoLicao(slug, e.Chave)); err == nil {
			for _, s := range secoesDaLicao(licao, "") {
				secoes = append(secoes, s["titulo"])
			}
		}
	}
	modo := "exercicios"
	if len(perguntas) == 0 {
		modo = "autoavaliacao"
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"revisao": rv, "concluida": e.Status == "concluida", "modo": modo, "perguntas": perguntas,
		"secoes": secoes, "intervalos": intervalosRevisao, "nota_minima": a.notaMinima(ctx),
	})
}

var notasAutoavaliacao = map[string]int{"bem": 100, "parcial": 60, "esqueci": 0}

func (a *app) handleFazerRevisao(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Respostas []struct {
			Indice int `json:"indice"`
			respostaExercicio
		} `json:"respostas"`
		Autoavaliacao string `json:"autoavaliacao"` // bem · parcial · esqueci (sem exercícios)
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if e.Status != "concluida" {
		a.responderErro(w, erroDeEntrada{errors.New("a revisão é das etapas concluídas")})
		return
	}
	if err := a.garantirRevisoes(ctx, e.TrilhaID); err != nil {
		a.responderErro(w, err)
		return
	}
	rv, _ := a.revisaoDa(ctx, e.TrilhaID, e.Chave)
	if rv == nil {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	minima := a.notaMinima(ctx)
	var nota, acertos, total int
	var respostas, resultado []byte
	var subir, manter bool
	if len(corpo.Respostas) > 0 {
		exs := a.lerExercicios(a.slugDaTrilha(ctx, e.TrilhaID), e.Chave)
		var escolhidos []exercicio
		var dadas []respostaExercicio
		vistos := map[int]bool{}
		for _, x := range corpo.Respostas {
			if x.Indice < 0 || x.Indice >= len(exs) || vistos[x.Indice] {
				a.responderErro(w, erroDeEntrada{errors.New("exercício fora da etapa ou repetido; recarregue a página")})
				return
			}
			vistos[x.Indice] = true
			escolhidos = append(escolhidos, exs[x.Indice])
			dadas = append(dadas, x.respostaExercicio)
		}
		var correcoes []correcaoExercicio
		correcoes, acertos = corrigirExercicios(escolhidos, dadas)
		total = len(escolhidos)
		nota = (acertos*100 + total/2) / total
		respostas, _ = json.Marshal(corpo.Respostas)
		resultado, _ = json.Marshal(correcoes)
		subir, manter = nota >= minima, nota >= 50
	} else {
		n, ok := notasAutoavaliacao[corpo.Autoavaliacao]
		if !ok {
			a.responderErro(w, erroDeEntrada{errors.New("responda às perguntas ou diga como foi (bem, parcial, esqueci)")})
			return
		}
		nota, total = n, 1
		if n == 100 {
			acertos = 1
		}
		respostas, _ = json.Marshal(map[string]string{"autoavaliacao": corpo.Autoavaliacao})
		resultado, _ = json.Marshal(map[string]string{"autoavaliacao": corpo.Autoavaliacao})
		subir, manter = corpo.Autoavaliacao == "bem", corpo.Autoavaliacao == "parcial"
	}
	nivel, efeito := rv.Nivel, "manteve"
	switch {
	case subir:
		nivel, efeito = min(rv.Nivel+1, len(intervalosRevisao)-1), "subiu"
	case !manter:
		nivel, efeito = 0, "voltou"
	}
	dia := hoje()
	proxima := dia.AddDate(0, 0, intervalosRevisao[nivel]).Format(formatoData)
	id, err := a.novaTentativa(ctx, e, tipoRevisao, tentCorrigida, nota, acertos, total, string(respostas), string(resultado), 0)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE revisoes SET nivel = ?, proxima = ?, ultima = ?, ultima_nota = ? WHERE trilha_id = ? AND etapa_chave = ?`,
		nivel, proxima, dia.Format(formatoData), nota, e.TrilhaID, e.Chave); err != nil {
		a.responderErro(w, err)
		return
	}
	novo, _ := a.revisaoDa(ctx, e.TrilhaID, e.Chave)
	t, err := lerTentativa(a.db.QueryRowContext(ctx, `SELECT `+colunasTentativa+` FROM tentativas WHERE id = ?`, id))
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, map[string]any{"tentativa": t, "revisao": novo, "efeito": efeito})
}

// ---------- simulado intercalado ----------

type perguntaIntercalada struct {
	EtapaChave  string `json:"etapa_chave"`
	EtapaID     int64  `json:"etapa_id"`
	EtapaTitulo string `json:"etapa_titulo"`
	Indice      int    `json:"indice"`
	Pergunta    string `json:"pergunta"`
	Conceito    string `json:"conceito"`
}

// bancoIntercalado: as perguntas do simulado das etapas já estudadas (concluídas ou em andamento).
func (a *app) bancoIntercalado(tr trilha) map[string][]perguntaSimulado {
	banco := map[string][]perguntaSimulado{}
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			if e.Status != "concluida" && e.Status != "em_andamento" {
				continue
			}
			if qs := a.lerSimulado(tr.Slug, e.Chave); len(qs) > 0 {
				banco[e.Chave] = qs
			}
		}
	}
	return banco
}

// sortearIntercalado: n perguntas, uma etapa por vez (as fracas primeiro), sem repetir etapa
// enquanto houver outra.
func sortearIntercalado(tr trilha, banco map[string][]perguntaSimulado, fracas map[string]bool, n int) []perguntaIntercalada {
	type fonte struct {
		e      etapa
		ordem  []int
		usadas int
	}
	var fortes, fracasL []*fonte
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			qs := banco[e.Chave]
			if len(qs) == 0 {
				continue
			}
			f := &fonte{e: e, ordem: rand.Perm(len(qs))}
			if fracas[e.Chave] {
				fracasL = append(fracasL, f)
			} else {
				fortes = append(fortes, f)
			}
		}
	}
	rand.Shuffle(len(fracasL), func(i, j int) { fracasL[i], fracasL[j] = fracasL[j], fracasL[i] })
	rand.Shuffle(len(fortes), func(i, j int) { fortes[i], fortes[j] = fortes[j], fortes[i] })
	fontes := append(fracasL, fortes...)
	lista := []perguntaIntercalada{}
	for len(lista) < n {
		andou := false
		for _, f := range fontes {
			if len(lista) >= n || f.usadas >= len(f.ordem) {
				continue
			}
			i := f.ordem[f.usadas]
			f.usadas++
			q := banco[f.e.Chave][i]
			lista = append(lista, perguntaIntercalada{f.e.Chave, f.e.ID, f.e.Titulo, i, q.Pergunta, q.Conceito})
			andou = true
		}
		if !andou {
			break
		}
	}
	// Misturadas: a ordem das etapas não pode dar a pista.
	rand.Shuffle(len(lista), func(i, j int) { lista[i], lista[j] = lista[j], lista[i] })
	return lista
}

func (a *app) handleIntercalado(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	tr, err := a.trilhaCompleta(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	n := 5
	if x, err := strconv.Atoi(r.URL.Query().Get("n")); err == nil && x >= 1 && x <= 15 {
		n = x
	}
	sinais, err := a.sinaisDaTrilha(ctx, tr)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	fracas := map[string]bool{}
	for k, s := range sinais.etapas {
		if len(s.Motivos) > 0 {
			fracas[k] = true
		}
	}
	banco := a.bancoIntercalado(tr)
	total := 0
	for _, qs := range banco {
		total += len(qs)
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"perguntas": sortearIntercalado(tr, banco, fracas, n), "etapas": len(banco), "banco": total,
	})
}

func (a *app) handleTentativaIntercalado(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Respostas []struct {
			EtapaChave string `json:"etapa_chave"`
			Pergunta   int    `json:"pergunta"`
			Resposta   string `json:"resposta"`
		} `json:"respostas"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	tr, err := a.trilhaCompleta(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	titulos := map[string]string{}
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			titulos[e.Chave] = e.Titulo
		}
	}
	banco := a.bancoIntercalado(tr)
	var respondidas []respondidaSimulado
	usadas := map[string]bool{}
	for _, x := range corpo.Respostas {
		x.Resposta = strings.TrimSpace(x.Resposta)
		chave := fmt.Sprintf("%s#%d", x.EtapaChave, x.Pergunta)
		if x.Resposta == "" || usadas[chave] {
			continue
		}
		qs := banco[x.EtapaChave]
		if x.Pergunta < 0 || x.Pergunta >= len(qs) {
			a.responderErro(w, erroDeEntrada{errors.New("pergunta fora dos bancos das etapas estudadas; recarregue a página")})
			return
		}
		if len([]rune(x.Resposta)) > 8000 {
			a.responderErro(w, erroDeEntrada{errors.New("uma resposta passou de 8000 caracteres")})
			return
		}
		usadas[chave] = true
		q := qs[x.Pergunta]
		respondidas = append(respondidas, respondidaSimulado{Pergunta: q.Pergunta, Esperado: q.Esperado, Conceito: q.Conceito,
			Resposta: x.Resposta, EtapaChave: x.EtapaChave, EtapaTitulo: titulos[x.EtapaChave]})
	}
	if len(respondidas) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("responda ao menos uma pergunta")})
		return
	}
	var corrigindo int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tentativas WHERE trilha_id = ? AND etapa_chave = '' AND estado = ?`,
		id, tentCorrigindo).Scan(&corrigindo); err != nil {
		a.responderErro(w, err)
		return
	}
	if corrigindo > 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a IA ainda está corrigindo o simulado intercalado anterior")})
		return
	}
	modelo := a.modeloPara(ctx, tipoCorrecao)
	respostas, _ := json.Marshal(respondidas)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoCorrecao, 0, "Correção do simulado intercalado: "+tr.Titulo, modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	tid, err := a.novaTentativa(ctx, etapa{TrilhaID: id}, tipoIntercalado, tentCorrigindo, 0, 0, len(respondidas), string(respostas), "", tarefaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	a.responderTentativa(w, r, tid, http.StatusCreated)
}

// ---------- sinais: notas, pontos fracos ----------

type resumoNotas struct {
	Ultima int    `json:"ultima"`
	Melhor int    `json:"melhor"`
	Vezes  int    `json:"vezes"`
	Quando string `json:"quando"`
}

type sinaisEtapa struct {
	Exercicios *resumoNotas `json:"exercicios"`
	Simulado   *resumoNotas `json:"simulado"`
	Revisao    *resumoNotas `json:"revisoes"`
	Motivos    []string     `json:"motivos"` // por que a etapa está fraca; vazio = não está
}

type pontoFraco struct {
	EtapaChave  string `json:"etapa_chave"`
	EtapaID     int64  `json:"etapa_id"`
	EtapaTitulo string `json:"etapa_titulo"`
	Conceito    string `json:"conceito"`
	Erros       int    `json:"erros"`
	Acertos     int    `json:"acertos"`
	Quando      string `json:"quando"`
	Exemplo     string `json:"exemplo"`  // a última pergunta que ele errou
	Resposta    string `json:"resposta"` // o que respondeu
	Esperado    string `json:"esperado"` // a certa, ou o que faltou
	Onde        string `json:"onde"`     // exercícios · simulado · revisão · intercalado
}

type sinaisTrilha struct {
	etapas      map[string]*sinaisEtapa
	pontos      []pontoFraco
	intercalado []tentativa
}

var nomesTentativa = map[string]string{tipoExercicios: "exercícios", tipoSimulado: "simulado", tipoRevisao: "revisão", tipoIntercalado: "intercalado"}

// itemDeSinal: uma resposta, de qualquer tipo de tentativa, do jeito que conta para os pontos fracos.
type itemDeSinal struct {
	chave, conceito, pergunta, resposta, esperado string
	certo                                         bool
}

func itensDaTentativa(t tentativa, chave string) []itemDeSinal {
	var itens []itemDeSinal
	switch t.Tipo {
	case tipoExercicios, tipoRevisao:
		var cs []correcaoExercicio
		if json.Unmarshal(t.Resultado, &cs) != nil {
			return nil // autoavaliação
		}
		for _, c := range cs {
			itens = append(itens, itemDeSinal{chave, c.Conceito, c.Enunciado, c.Resposta, c.Correta, c.Certa})
		}
	case tipoSimulado, tipoIntercalado:
		var r struct {
			Itens []struct {
				respondidaSimulado
				Veredito string `json:"veredito"`
				Faltou   string `json:"faltou"`
			} `json:"itens"`
		}
		if json.Unmarshal(t.Resultado, &r) != nil {
			return nil
		}
		for _, x := range r.Itens {
			k := chave
			if x.EtapaChave != "" {
				k = x.EtapaChave
			}
			itens = append(itens, itemDeSinal{k, x.Conceito, x.Pergunta, x.Resposta, primeiro(x.Faltou, x.Esperado), x.Veredito == "certo"})
		}
	}
	return itens
}

// sinaisDaTrilha lê todas as tentativas corrigidas, da mais antiga para a mais nova.
func (a *app) sinaisDaTrilha(ctx context.Context, tr trilha) (sinaisTrilha, error) {
	s := sinaisTrilha{etapas: map[string]*sinaisEtapa{}, intercalado: []tentativa{}}
	rows, err := a.db.QueryContext(ctx, `SELECT etapa_chave, `+colunasTentativa+` FROM tentativas WHERE trilha_id = ? ORDER BY id`, tr.ID)
	if err != nil {
		return s, err
	}
	type porConceito struct {
		pontoFraco
		ultimoCerto bool
		ordem       int
	}
	conceitos := map[string]*porConceito{}
	var ordem []string
	n := 0
	for rows.Next() {
		var chave string
		var t tentativa
		var respostas, resultado string
		if err := rows.Scan(&chave, &t.ID, &t.Tipo, &t.Estado, &t.Nota, &t.Acertos, &t.Total, &respostas, &resultado, &t.Erro, &t.TarefaID, &t.CriadoEm, &t.AtualizadoEm); err != nil {
			rows.Close()
			return s, err
		}
		t.Respostas = json.RawMessage(padrao(respostas, "[]"))
		if resultado != "" {
			t.Resultado = json.RawMessage(resultado)
		}
		if t.Tipo == tipoIntercalado {
			s.intercalado = append(s.intercalado, t)
		}
		if t.Estado != tentCorrigida {
			continue
		}
		if chave != "" {
			se := s.etapas[chave]
			if se == nil {
				se = &sinaisEtapa{Motivos: []string{}}
				s.etapas[chave] = se
			}
			alvo := map[string]**resumoNotas{tipoExercicios: &se.Exercicios, tipoSimulado: &se.Simulado, tipoRevisao: &se.Revisao}[t.Tipo]
			if alvo != nil {
				if *alvo == nil {
					*alvo = &resumoNotas{}
				}
				x := *alvo
				x.Ultima, x.Melhor, x.Vezes, x.Quando = t.Nota, max(x.Melhor, t.Nota), x.Vezes+1, t.CriadoEm
			}
		}
		for _, it := range itensDaTentativa(t, chave) {
			if it.chave == "" {
				continue
			}
			nome := strings.TrimSpace(it.conceito)
			if nome == "" {
				nome = it.pergunta
			}
			k := it.chave + "\x00" + normalizarResposta(nome)
			c := conceitos[k]
			if c == nil {
				c = &porConceito{pontoFraco: pontoFraco{EtapaChave: it.chave, Conceito: nome}}
				conceitos[k] = c
				ordem = append(ordem, k)
			}
			n++
			c.ordem, c.ultimoCerto, c.Quando = n, it.certo, t.CriadoEm
			if it.certo {
				c.Acertos++
			} else {
				c.Erros++
				c.Exemplo, c.Resposta, c.Esperado, c.Onde = it.pergunta, it.resposta, it.esperado, nomesTentativa[t.Tipo]
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return s, err
	}
	// Ponto fraco: o conceito que ele errou da última vez que apareceu.
	etapas := map[string]etapa{}
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			etapas[e.Chave] = e
		}
	}
	var fracos []*porConceito
	for _, k := range ordem {
		c := conceitos[k]
		e, existe := etapas[c.EtapaChave]
		if c.ultimoCerto || !existe {
			continue
		}
		c.EtapaID, c.EtapaTitulo = e.ID, e.Titulo
		fracos = append(fracos, c)
	}
	sort.SliceStable(fracos, func(i, j int) bool {
		if fracos[i].Erros != fracos[j].Erros {
			return fracos[i].Erros > fracos[j].Erros
		}
		return fracos[i].ordem > fracos[j].ordem
	})
	s.pontos = []pontoFraco{}
	porEtapa := map[string]int{}
	for _, c := range fracos {
		s.pontos = append(s.pontos, c.pontoFraco)
		porEtapa[c.EtapaChave]++
	}
	minima := a.notaMinima(ctx)
	for k, se := range s.etapas {
		if se.Exercicios != nil && se.Exercicios.Ultima < minima {
			se.Motivos = append(se.Motivos, fmt.Sprintf("exercícios em %d%% (mínimo %d%%)", se.Exercicios.Ultima, minima))
		}
		if se.Simulado != nil && se.Simulado.Ultima < 60 {
			se.Motivos = append(se.Motivos, fmt.Sprintf("simulado em %d", se.Simulado.Ultima))
		}
		if se.Revisao != nil && se.Revisao.Ultima < minima {
			se.Motivos = append(se.Motivos, fmt.Sprintf("última revisão em %d%%", se.Revisao.Ultima))
		}
		if n := porEtapa[k]; n > 0 && len(se.Motivos) == 0 {
			se.Motivos = append(se.Motivos, fmt.Sprintf("%d ponto(s) fraco(s)", n))
		}
	}
	for k, n := range porEtapa {
		if s.etapas[k] == nil {
			s.etapas[k] = &sinaisEtapa{Motivos: []string{fmt.Sprintf("%d ponto(s) fraco(s)", n)}}
		}
	}
	return s, nil
}

// pontosFracosDaEtapa: para a tela da etapa e o pedido do reforço.
func (a *app) pontosFracosDaEtapa(ctx context.Context, e etapa) ([]pontoFraco, *sinaisEtapa) {
	tr, err := a.trilhaCompleta(ctx, e.TrilhaID)
	if err != nil {
		return []pontoFraco{}, nil
	}
	s, err := a.sinaisDaTrilha(ctx, tr)
	if err != nil {
		return []pontoFraco{}, nil
	}
	lista := []pontoFraco{}
	for _, p := range s.pontos {
		if p.EtapaChave == e.Chave {
			lista = append(lista, p)
		}
	}
	return lista, s.etapas[e.Chave]
}

// ---------- painel ----------

type linhaPainel struct {
	ID      int64        `json:"id"`
	Chave   string       `json:"chave"`
	Titulo  string       `json:"titulo"`
	Modulo  string       `json:"modulo"`
	Status  string       `json:"status"`
	Tipo    string       `json:"tipo"`
	Sinais  *sinaisEtapa `json:"sinais"`
	Revisao *revisao     `json:"revisao"`
}

func (a *app) handlePainel(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	if err := a.garantirRevisoes(ctx, id); err != nil {
		a.responderErro(w, err)
		return
	}
	tr, err := a.trilhaCompleta(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	s, err := a.sinaisDaTrilha(ctx, tr)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	revs, err := a.revisoesDaTrilha(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	pendentes, err := a.revisoesPendentes(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	linhas := []linhaPainel{}
	somaEx, nEx, emDia, fracas := 0, 0, 0, 0
	for _, m := range tr.Modulos {
		for _, e := range m.Etapas {
			l := linhaPainel{ID: e.ID, Chave: e.Chave, Titulo: e.Titulo, Modulo: m.Titulo, Status: e.Status, Tipo: e.Tipo,
				Sinais: s.etapas[e.Chave]}
			if e.Status == "concluida" {
				l.Revisao = revs[e.Chave]
				if l.Revisao != nil && !l.Revisao.Pendente {
					emDia++
				}
			}
			if l.Sinais != nil {
				if l.Sinais.Exercicios != nil {
					somaEx += l.Sinais.Exercicios.Ultima
					nEx++
				}
				if len(l.Sinais.Motivos) > 0 {
					fracas++
				}
			}
			linhas = append(linhas, l)
		}
	}
	media := -1
	if nEx > 0 {
		media = (somaEx + nEx/2) / nEx
	}
	banco := a.bancoIntercalado(tr)
	perguntas := 0
	for _, qs := range banco {
		perguntas += len(qs)
	}
	inter := s.intercalado
	sort.SliceStable(inter, func(i, j int) bool { return inter[i].ID > inter[j].ID })
	if len(inter) > 10 {
		inter = inter[:10]
	}
	responderJSON(w, http.StatusOK, map[string]any{
		"trilha":        map[string]any{"id": tr.ID, "titulo": tr.Titulo, "estado": tr.Estado, "progresso": tr.Progresso},
		"nota_minima":   a.notaMinima(ctx),
		"etapas":        linhas,
		"pontos_fracos": s.pontos,
		"revisoes":      map[string]any{"pendentes": pendentes, "em_dia": emDia, "intervalos": intervalosRevisao},
		"resumo":        map[string]any{"media_exercicios": media, "etapas_fracas": fracas},
		"intercalado":   map[string]any{"etapas": len(banco), "perguntas": perguntas, "tentativas": inter},
	})
}
