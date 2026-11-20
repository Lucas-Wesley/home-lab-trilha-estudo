package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Entrada, fase 3 (T37): a IA divide a fonte em fragmentos por assunto — a única leitura da
// transcrição inteira (D3 do plano). O Lucas vê os fragmentos, ajusta os tempos, junta, separa,
// descarta e confirma; a fase 4 (destinos.go) dá o destino de cada um.

const tipoFragmentar = "fragmentar" // a tarefa da fila

var tiposFragmento = []string{"conteudo", "opiniao", "fora"}

type fragmento struct {
	ID           int64            `json:"id"`
	FonteID      int64            `json:"fonte_id"`
	Ordem        int              `json:"ordem"`
	Inicio       int              `json:"inicio"` // segundos
	Fim          int              `json:"fim"`    // segundos
	Titulo       string           `json:"titulo"`
	Resumo       string           `json:"resumo"`
	Assuntos     []string         `json:"assuntos"`
	Tipo         string           `json:"tipo"`               // conteudo · opiniao · fora
	Estado       string           `json:"estado"`             // sugerido · confirmado · sem_destino · enviado · descartado
	Sugestao     *sugestaoDestino `json:"sugestao,omitempty"` // o que a IA sugeriu de destino (T38)
	Destinos     []destino        `json:"destinos,omitempty"` // as trilhas marcadas para ele (T38)
	SemDestinoEm string           `json:"sem_destino_em,omitempty"`
	CriadoEm     string           `json:"criado_em"`
	AtualizadoEm string           `json:"atualizado_em"`
}

// sugestaoDestino: o que a IA respondeu para um fragmento na tarefa "destino".
type sugestaoDestino struct {
	Trilhas       []sugestaoTrilha `json:"trilhas"`
	TrilhaNova    string           `json:"trilha_nova,omitempty"`
	MotivoNenhuma string           `json:"motivo_nenhuma,omitempty"`
}

type sugestaoTrilha struct {
	TrilhaID int64  `json:"trilha_id"`
	Titulo   string `json:"titulo"`
	Motivo   string `json:"motivo"`
}

// destino: uma trilha marcada para o fragmento, e o encaixe quando já foi mandado.
type destino struct {
	TrilhaID   int64  `json:"trilha_id"`
	Titulo     string `json:"titulo"`
	PropostaID int64  `json:"proposta_id,omitempty"`
	Estado     string `json:"estado,omitempty"` // do encaixe: aguardando · aprovada · descartada · falhou
}

const colunasFragmento = `id, fonte_id, ordem, inicio, fim, titulo, resumo, assuntos, tipo, estado,
	sugestao, sem_destino_em, criado_em, atualizado_em FROM fragmentos`

func lerFragmento(row interface{ Scan(...any) error }) (fragmento, error) {
	var f fragmento
	var assuntos, sug string
	err := row.Scan(&f.ID, &f.FonteID, &f.Ordem, &f.Inicio, &f.Fim, &f.Titulo, &f.Resumo, &assuntos,
		&f.Tipo, &f.Estado, &sug, &f.SemDestinoEm, &f.CriadoEm, &f.AtualizadoEm)
	if err == nil {
		f.Assuntos = []string{}
		json.Unmarshal([]byte(assuntos), &f.Assuntos)
		if sug != "" {
			var s sugestaoDestino
			if json.Unmarshal([]byte(sug), &s) == nil {
				f.Sugestao = &s
			}
		}
	}
	return f, err
}

func (a *app) fragmentosDaFonte(ctx context.Context, fonteID int64) ([]fragmento, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasFragmento+` WHERE fonte_id = ? ORDER BY ordem, inicio`, fonteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []fragmento{}
	for rows.Next() {
		f, err := lerFragmento(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, f)
	}
	return lista, rows.Err()
}

func (a *app) fragmentoPorID(ctx context.Context, id int64) (fragmento, error) {
	f, err := lerFragmento(a.db.QueryRowContext(ctx, `SELECT `+colunasFragmento+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return f, errNaoEncontrou
	}
	return f, err
}

// ---------- o tempo "mm:ss"/"h:mm:ss" da transcrição ----------

var tempoFragmento = regexp.MustCompile(`^(?:(\d+):)?(\d+):(\d{2})$`)

func segundosDoTempo(s string) (int, error) {
	m := tempoFragmento.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("tempo ilegível: %q", s)
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	se, _ := strconv.Atoi(m[3])
	return h*3600 + mi*60 + se, nil
}

// segundosDe: na API o tempo vem como "mm:ss" ou como número de segundos.
func segundosDe(v any) (int, error) {
	switch x := v.(type) {
	case float64:
		return int(x), nil
	case string:
		s, err := segundosDoTempo(x)
		if err != nil {
			return 0, erroDeEntrada{err}
		}
		return s, nil
	}
	return 0, erroDeEntrada{fmt.Errorf("tempo inválido: %v", v)}
}

func tempoDos(segundos int) string {
	return marcaDeTempo(int64(segundos) * 1000)
}

// ---------- a tarefa: a IA fragmenta ----------

const esquemaFragmentos = `{"type":"object","properties":{
"fragmentos":{"type":"array","items":{"type":"object","properties":{
  "inicio":{"type":"string","description":"mm:ss ou h:mm:ss, como na transcrição"},
  "fim":{"type":"string","description":"mm:ss ou h:mm:ss"},
  "titulo":{"type":"string"},
  "resumo":{"type":"string","description":"2 a 4 frases: o que o trecho ensina"},
  "assuntos":{"type":"array","items":{"type":"string"}},
  "tipo":{"type":"string","enum":["conteudo","opiniao","fora"]}},
  "required":["inicio","fim","titulo","resumo","assuntos","tipo"]}},
"notas":{"type":"string"}},
"required":["fragmentos","notas"]}`

// fragmentoIA: como a IA devolve cada fragmento (tempos em texto).
type fragmentoIA struct {
	Inicio   string   `json:"inicio"`
	Fim      string   `json:"fim"`
	Titulo   string   `json:"titulo"`
	Resumo   string   `json:"resumo"`
	Assuntos []string `json:"assuntos"`
	Tipo     string   `json:"tipo"`
}

func (a *app) fonteDaTarefa(ctx context.Context, tarefaID int64) (fonte, error) {
	f, err := lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE tarefa_id = ?`, tarefaID))
	if errors.Is(err, sql.ErrNoRows) {
		return f, errNaoEncontrou
	}
	return f, err
}

// limparFragmentos confere o que a IA devolveu antes de gravar: tempos válidos, fim > inicio,
// dentro da duração (fim que passa fecha no fim do vídeo), em ordem e sem sobreposição (o fim do
// primeiro vira o início do segundo). O que não passa volta na lista "tirados".
func limparFragmentos(duracao int, entrada []fragmentoIA) ([]fragmento, []string) {
	var bons []fragmento
	var tirados []string
	for _, fr := range entrada {
		ini, err1 := segundosDoTempo(fr.Inicio)
		fim, err2 := segundosDoTempo(fr.Fim)
		nome := fmt.Sprintf("%s–%s", fr.Inicio, fr.Fim)
		switch {
		case err1 != nil || err2 != nil:
			tirados = append(tirados, nome+" (tempo ilegível)")
		case duracao > 0 && ini >= duracao:
			tirados = append(tirados, nome+" (fora do vídeo)")
		default:
			if duracao > 0 {
				fim = min(fim, duracao)
			}
			if fim <= ini {
				tirados = append(tirados, nome+" (fim não passa do início)")
				continue
			}
			titulo := textoDaIA(fr.Titulo)
			if titulo == "" {
				tirados = append(tirados, nome+" (sem título)")
				continue
			}
			tipo := strings.TrimSpace(fr.Tipo)
			if !contem(tiposFragmento, tipo) {
				tipo = "conteudo"
			}
			var assuntos []string
			for _, a := range fr.Assuntos {
				if s := textoDaIA(a); s != "" {
					assuntos = append(assuntos, s)
				}
			}
			bons = append(bons, fragmento{
				Inicio: ini, Fim: fim, Titulo: titulo, Resumo: textoDaIA(fr.Resumo),
				Assuntos: assuntos, Tipo: tipo,
			})
		}
	}
	// Em ordem e sem sobreposição: o fim de um vira o início do seguinte; o que esvazia sai.
	sort.SliceStable(bons, func(i, j int) bool { return bons[i].Inicio < bons[j].Inicio })
	var finais []fragmento
	for i, b := range bons {
		if i > 0 && b.Inicio < finais[len(finais)-1].Fim {
			finais[len(finais)-1].Fim = b.Inicio
		}
		if b.Inicio < b.Fim {
			finais = append(finais, b)
		} else {
			tirados = append(tirados, b.Titulo+" (sumiu na sobreposição)")
		}
	}
	return finais, tirados
}

// pedidoDaFragmentacao: o pedido.md que a IA lê antes da transcrição.
func pedidoDaFragmentacao(f fonte, transcricao string, confirmados []fragmento) map[string]string {
	var s strings.Builder
	s.WriteString("# Pedido: fragmentar\n\n")
	fmt.Fprintf(&s, "- Vídeo: %q (%s)\n", f.Titulo, f.URL)
	if f.Canal != "" {
		fmt.Fprintf(&s, "- Canal: %s\n", f.Canal)
	}
	if f.Duracao > 0 {
		fmt.Fprintf(&s, "- Duração: %s\n", tempoDos(f.Duracao))
	}
	s.WriteString("- A transcrição está em `transcricao.md`, em blocos de 10 s com as marcas [mm:ss].\n")
	if len(f.Capitulos) > 0 {
		s.WriteString("\n## Capítulos do autor (pista dos assuntos, não a verdade)\n\n")
		for _, c := range f.Capitulos {
			fmt.Fprintf(&s, "- %s %s\n", tempoDos(int(c.Inicio)), c.Titulo)
		}
	}
	if p := strings.TrimSpace(f.Pedido); p != "" {
		fmt.Fprintf(&s, "\n## Pedido do Lucas\n\n%s\n", p)
	}
	if len(confirmados) > 0 {
		s.WriteString("\n## Fragmentos já confirmados (estão certos; não repita nem mude)\n\n")
		for _, fr := range confirmados {
			fmt.Fprintf(&s, "- %s–%s — %s\n", tempoDos(fr.Inicio), tempoDos(fr.Fim), fr.Titulo)
		}
	}
	return map[string]string{"pedido.md": s.String(), "transcricao.md": transcricao}
}

// ---------- transcrição longa em partes (T48) ----------

// limiteParte: quantos caracteres de transcrição vão numa chamada só (~30 mil tokens, umas 2 h de
// fala). Acima disso a divisão vai em partes do mesmo tamanho, cortadas entre blocos [mm:ss]: a IA
// lê cada parte com atenção em vez de passar o olho em 200 mil tokens de uma vez.
var limiteParte = 120_000 // var: os testes baixam

// parteTranscricao: um pedaço da transcrição, do bloco que começa em `inicio` até `fim`
// (o início da parte seguinte, ou a duração do vídeo na última; 0 = sem fim conhecido).
type parteTranscricao struct {
	Inicio, Fim int
	Texto       string
}

// partesDaTranscricao corta a transcrição em partes de tamanho parecido, sempre no começo de um
// bloco. O cabeçalho (antes do primeiro bloco) vai só na primeira parte. Transcrição que cabe no
// limite volta inteira, numa parte só.
func partesDaTranscricao(transcricao string, limite, duracao int) []parteTranscricao {
	if len(transcricao) <= limite || limite <= 0 {
		return []parteTranscricao{{Inicio: 0, Fim: duracao, Texto: transcricao}}
	}
	type bloco struct {
		ini   int // byte onde começa
		tempo int
	}
	var blocos []bloco
	pos := 0
	for _, linha := range strings.SplitAfter(transcricao, "\n") {
		if m := marcaDoBloco.FindStringSubmatch(linha); m != nil {
			if t, err := segundosDoTempo(m[1]); err == nil {
				blocos = append(blocos, bloco{pos, t})
			}
		}
		pos += len(linha)
	}
	if len(blocos) < 2 {
		return []parteTranscricao{{Inicio: 0, Fim: duracao, Texto: transcricao}}
	}
	n := (len(transcricao) + limite - 1) / limite
	alvo := len(transcricao) / n
	var cortes []bloco // onde cada parte (a partir da 2ª) começa
	for _, b := range blocos[1:] {
		if len(cortes) == n-1 {
			break
		}
		if b.ini >= (len(cortes)+1)*alvo {
			cortes = append(cortes, b)
		}
	}
	var partes []parteTranscricao
	ini, tempo := 0, 0
	for _, c := range cortes {
		partes = append(partes, parteTranscricao{Inicio: tempo, Fim: c.tempo, Texto: transcricao[ini:c.ini]})
		ini, tempo = c.ini, c.tempo
	}
	return append(partes, parteTranscricao{Inicio: tempo, Fim: duracao, Texto: transcricao[ini:]})
}

// juntarNaFronteira: o primeiro fragmento de uma parte com o mesmo título do último da anterior,
// colado na fronteira, é o mesmo assunto que continuou — vira um fragmento só.
func juntarNaFronteira(anteriores, novos []fragmento) []fragmento {
	if len(anteriores) == 0 || len(novos) == 0 {
		return append(anteriores, novos...)
	}
	ult := &anteriores[len(anteriores)-1]
	pri := novos[0]
	if strings.EqualFold(strings.TrimSpace(ult.Titulo), strings.TrimSpace(pri.Titulo)) && pri.Inicio-ult.Fim <= 60 {
		ult.Fim = max(ult.Fim, pri.Fim)
		if len(pri.Resumo) > len(ult.Resumo) {
			ult.Resumo = pri.Resumo
		}
		for _, as := range pri.Assuntos {
			if !contem(ult.Assuntos, as) && len(ult.Assuntos) < 5 {
				ult.Assuntos = append(ult.Assuntos, as)
			}
		}
		novos = novos[1:]
	}
	return append(anteriores, novos...)
}

// dentroDaParte prende os fragmentos de uma parte ao intervalo dela (a IA às vezes começa em 00:00).
func dentroDaParte(p parteTranscricao, frs []fragmento) []fragmento {
	var bons []fragmento
	for _, fr := range frs {
		fr.Inicio = max(fr.Inicio, p.Inicio)
		if p.Fim > 0 {
			fr.Fim = min(fr.Fim, p.Fim)
		}
		if fr.Inicio < fr.Fim {
			bons = append(bons, fr)
		}
	}
	return bons
}

// respostaFragmentos: o que a IA devolve em cada chamada.
type respostaFragmentos struct {
	Fragmentos []fragmentoIA `json:"fragmentos"`
	Notas      string        `json:"notas"`
}

// fragmentarParte chama a IA para uma parte (ou para a transcrição inteira, quando é uma só).
func (a *app) fragmentarParte(ctx context.Context, t *tarefa, f fonte, regra string, arquivos map[string]string, k int) (respostaFragmentos, error) {
	var r respostaFragmentos
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10), strconv.Itoa(k))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return r, err
	}
	defer os.RemoveAll(pasta)
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return r, err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: regra, ferramentas: "Read", esquema: esquemaFragmentos, modelo: t.Modelo,
		instrucao: "Divida o vídeo em fragmentos por assunto. Tudo está em `pedido.md` e `transcricao.md`, nesta pasta: leia os dois inteiros antes.",
	})
	if err != nil {
		return r, err
	}
	err = dadosDaResposta(resp, &r)
	return r, err
}

// fragmentar é a tarefa: a IA lê a transcrição e devolve os trechos por assunto, que o servidor
// confere e grava como sugerido. Transcrição longa vai em partes (T48), uma chamada por parte, em
// sequência; a parte que falha duas vezes fica de fora com um aviso nas notas.
func (a *app) fragmentar(ctx context.Context, t *tarefa) error {
	f, err := a.fonteDaTarefa(ctx, t.ID)
	if errors.Is(err, errNaoEncontrou) {
		return errors.New("a fonte desta tarefa não existe mais")
	}
	if err != nil {
		return err
	}
	transcricao, _ := a.conteudo.ler(pastaDaFonte(f.ID) + "/transcricao.md")
	if strings.TrimSpace(transcricao) == "" {
		return errors.New("a fonte não tem transcrição para fragmentar")
	}
	var confirmados []fragmento
	todos, err := a.fragmentosDaFonte(ctx, f.ID)
	if err != nil {
		return err
	}
	for _, fr := range todos {
		if fr.Estado == "confirmado" {
			confirmados = append(confirmados, fr)
		}
	}
	rg, err := a.ia.lerRegra("fragmentar.md")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10)))

	partes := partesDaTranscricao(transcricao, limiteParte, f.Duracao)
	var bons []fragmento
	var tirados, notas, falhas []string
	var ultimoErro error
	for k, p := range partes {
		arquivos := pedidoDaFragmentacao(f, p.Texto, confirmados)
		if len(partes) > 1 {
			a.db.ExecContext(ctx, `UPDATE fontes SET notas = ? WHERE id = ?`,
				fmt.Sprintf("Dividindo em %d partes: parte %d (%s–%s)…", len(partes), k+1, tempoDos(p.Inicio), tempoDos(max(p.Fim, p.Inicio))), f.ID)
			arquivos["pedido.md"] += pedidoDaParte(partes, k, bons)
		}
		r, err := a.fragmentarParte(ctx, t, f, rg.Texto, arquivos, k)
		if err != nil && len(partes) > 1 && ctx.Err() == nil {
			r, err = a.fragmentarParte(ctx, t, f, rg.Texto, arquivos, k) // uma segunda chance
		}
		if err != nil {
			if len(partes) == 1 || ctx.Err() != nil {
				return err
			}
			ultimoErro = err
			falhas = append(falhas, fmt.Sprintf("%s–%s", tempoDos(p.Inicio), tempoDos(max(p.Fim, p.Inicio))))
			log.Printf("entrada: fonte %d, parte %d de %d: %v", f.ID, k+1, len(partes), err)
			continue
		}
		frs, fora := limparFragmentos(f.Duracao, r.Fragmentos)
		tirados = append(tirados, fora...)
		if len(partes) > 1 {
			frs = dentroDaParte(p, frs)
		}
		bons = juntarNaFronteira(bons, frs)
		if n := textoDaIA(r.Notas); n != "" {
			notas = append(notas, n)
		}
	}
	if len(bons) == 0 {
		if ultimoErro != nil {
			return ultimoErro
		}
		return fmt.Errorf("a IA não devolveu fragmentos válidos%s", entreParenteses(strings.TrimSpace(strings.Join(notas, " "))))
	}
	nota := strings.Join(notas, " ")
	if len(partes) > 1 {
		nota = strings.TrimSpace(fmt.Sprintf("Dividido em %d partes (vídeo longo). %s", len(partes), nota))
	}
	if len(falhas) > 0 {
		nota += "\n\nNão consegui dividir " + strings.Join(falhas, ", ") + ": peça de novo só esse trecho no pedido."
	}
	if len(tirados) > 0 {
		nota = strings.TrimSpace(nota + "\n\n(O app tirou: " + strings.Join(tirados, "; ") + ".)")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, fr := range bons {
		assuntos, _ := json.Marshal(fr.Assuntos)
		if _, err := tx.ExecContext(ctx, `INSERT INTO fragmentos (fonte_id, ordem, inicio, fim, titulo, resumo, assuntos, tipo, criado_em, atualizado_em)
			VALUES (?, 0, ?, ?, ?, ?, ?, ?, ?, ?)`,
			f.ID, fr.Inicio, fr.Fim, fr.Titulo, fr.Resumo, string(assuntos), fr.Tipo, agora(), agora()); err != nil {
			return err
		}
	}
	if err := renumerarFragmentos(ctx, tx, f.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET estado = 'fragmentada', erro = '', notas = ? WHERE id = ?`, nota, f.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Modo automático (T46): dividiu, já pede as trilhas (Haiku, só os resumos dos fragmentos).
	if f.Fragmentar {
		if err := a.pedirDestinos(ctx, f, ""); err != nil {
			log.Printf("entrada: fonte %d, destino automático: %v", f.ID, err)
		}
	}
	return nil
}

// pedidoDaParte: o que a IA precisa saber quando lê só uma parte do vídeo.
func pedidoDaParte(partes []parteTranscricao, k int, anteriores []fragmento) string {
	p := partes[k]
	var s strings.Builder
	fmt.Fprintf(&s, "\n## Parte %d de %d\n\n", k+1, len(partes))
	fim := "o fim do vídeo"
	if p.Fim > 0 {
		fim = tempoDos(p.Fim)
	}
	fmt.Fprintf(&s, "O vídeo é longo e vai em partes: `transcricao.md` tem **só** de %s até %s. Divida só esse trecho, com os tempos do vídeo inteiro (como estão na transcrição). As outras partes já foram ou serão divididas em outras chamadas.\n", tempoDos(p.Inicio), fim)
	if len(anteriores) > 0 {
		u := anteriores[len(anteriores)-1]
		fmt.Fprintf(&s, "\nA parte anterior terminou no fragmento **%q** (%s–%s): %s\nSe esta parte começa ainda nesse assunto, comece o primeiro fragmento em %s com **exatamente o mesmo título** (o app junta os dois).\n",
			u.Titulo, tempoDos(u.Inicio), tempoDos(u.Fim), u.Resumo, tempoDos(p.Inicio))
	}
	return s.String()
}

// renumerarFragmentos põe a ordem de volta por início (os confirmados mantêm o lugar no vídeo).
func renumerarFragmentos(ctx context.Context, tx *sql.Tx, fonteID int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM fragmentos WHERE fonte_id = ? AND estado != 'descartado' ORDER BY inicio, id`, fonteID)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET ordem = ? WHERE id = ?`, i, id); err != nil {
			return err
		}
	}
	return nil
}

// ---------- handlers ----------

// handleEntradaFragmentar põe a divisão na fila. Refazer apaga os sugeridos e mantém os
// confirmados (que vão no pedido para a IA não repetir).
func (a *app) handleEntradaFragmentar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pedido string `json:"pedido"`
	}
	// O pedido é opcional: um corpo vazio vale.
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&corpo); err != nil && !errors.Is(err, io.EOF) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("corpo inválido: %w", err)})
		return
	}
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if f.Estado != "pronta" && f.Estado != "fragmentada" {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("só dá para fragmentar fonte pronta ou já fragmentada (esta está %s)", f.Estado)})
		return
	}
	if err := a.pedirFragmentar(ctx, id, f.Titulo, strings.TrimSpace(corpo.Pedido)); err != nil {
		a.responderErro(w, err)
		return
	}
	f, _ = a.entradaCompleta(ctx, id)
	responderJSON(w, http.StatusAccepted, f)
}

// pedirFragmentar põe a divisão na fila: apaga os sugeridos (os confirmados ficam e vão no
// pedido), cria a tarefa `fragmentar` e marca a fonte "fragmentando". Usado pelo botão da
// página da fonte e pelo "dividir em fragmentos quando terminar" da fonte telegram (T44).
func (a *app) pedirFragmentar(ctx context.Context, fonteID int64, titulo, pedido string) error {
	modelo := a.modeloPara(ctx, tipoFragmentar)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM fragmentos WHERE fonte_id = ? AND estado = 'sugerido'`, fonteID); err != nil {
		return err
	}
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoFragmentar, 0, "Fragmentar: "+primeiraLinha(titulo, strconv.FormatInt(fonteID, 10)), modelo)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET estado = 'fragmentando', erro = '', notas = '', tarefa_id = ?, pedido = ? WHERE id = ?`,
		tarefaID, pedido, fonteID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.fila.acordar()
	return nil
}

// handleEntradaConfirmar: todos os sugeridos da fonte viram confirmados.
func (a *app) handleEntradaConfirmar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	if _, err := a.fontePorID(ctx, id); err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE fragmentos SET estado = 'confirmado', atualizado_em = ? WHERE fonte_id = ? AND estado = 'sugerido'`, agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	lista, err := a.fragmentosDaFonte(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, lista)
}

// validarTemposDoFragmento confere os tempos editados como a limpeza da tarefa confere (3.3):
// fim > inicio, dentro da duração e sem cruzar outro fragmento da fonte.
func (a *app) validarTemposDoFragmento(ctx context.Context, f fonte, ini, fim int, excetoID int64) error {
	switch {
	case ini < 0:
		return erroDeEntrada{errors.New("o início não pode ser antes do vídeo")}
	case fim <= ini:
		return erroDeEntrada{errors.New("o fim tem que ser depois do início")}
	case f.Duracao > 0 && fim > f.Duracao:
		return erroDeEntrada{fmt.Errorf("o fim (%s) passa do vídeo (%s)", tempoDos(fim), tempoDos(f.Duracao))}
	}
	rows, err := a.db.QueryContext(ctx, `SELECT titulo, inicio, fim FROM fragmentos WHERE fonte_id = ? AND estado != 'descartado' AND id != ?`, f.ID, excetoID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var i, fm int
		if err := rows.Scan(&t, &i, &fm); err != nil {
			return err
		}
		if ini < fm && fim > i {
			return erroDeEntrada{fmt.Errorf("cruza com o fragmento %q (%s–%s)", t, tempoDos(i), tempoDos(fm))}
		}
	}
	return rows.Err()
}

func (a *app) handleFragmentoEditar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Inicio   any       `json:"inicio"` // "mm:ss" ou segundos
		Fim      any       `json:"fim"`
		Titulo   *string   `json:"titulo"`
		Resumo   *string   `json:"resumo"`
		Assuntos *[]string `json:"assuntos"`
		Tipo     *string   `json:"tipo"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	fr, err := a.fragmentoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	f, err := a.fontePorID(ctx, fr.FonteID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if corpo.Inicio != nil {
		if fr.Inicio, err = segundosDe(corpo.Inicio); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	if corpo.Fim != nil {
		if fr.Fim, err = segundosDe(corpo.Fim); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	if corpo.Titulo != nil {
		fr.Titulo = strings.TrimSpace(*corpo.Titulo)
	}
	if corpo.Resumo != nil {
		fr.Resumo = strings.TrimSpace(*corpo.Resumo)
	}
	if corpo.Assuntos != nil {
		var assuntos []string
		for _, s := range *corpo.Assuntos {
			if s = strings.TrimSpace(s); s != "" {
				assuntos = append(assuntos, s)
			}
		}
		fr.Assuntos = assuntos
	}
	if corpo.Tipo != nil {
		tipo := strings.TrimSpace(*corpo.Tipo)
		if !contem(tiposFragmento, tipo) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("tipo desconhecido: %q", tipo)})
			return
		}
		fr.Tipo = tipo
	}
	if fr.Titulo == "" {
		a.responderErro(w, erroDeEntrada{errors.New("o fragmento precisa de um título")})
		return
	}
	if err := a.validarTemposDoFragmento(ctx, f, fr.Inicio, fr.Fim, fr.ID); err != nil {
		a.responderErro(w, err)
		return
	}
	assuntos, _ := json.Marshal(fr.Assuntos)
	if _, err := a.db.ExecContext(ctx, `UPDATE fragmentos SET inicio = ?, fim = ?, titulo = ?, resumo = ?, assuntos = ?, tipo = ?, atualizado_em = ? WHERE id = ?`,
		fr.Inicio, fr.Fim, fr.Titulo, fr.Resumo, string(assuntos), fr.Tipo, agora(), fr.ID); err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, fr)
}

// handleFragmentoDividir: {"em":"12:30"} vira dois — o segundo é "(continuação)".
func (a *app) handleFragmentoDividir(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Em any `json:"em"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	fr, err := a.fragmentoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	em, err := segundosDe(corpo.Em)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if em <= fr.Inicio || em >= fr.Fim {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("o corte tem que ser dentro de %s–%s", tempoDos(fr.Inicio), tempoDos(fr.Fim))})
		return
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO fragmentos (fonte_id, ordem, inicio, fim, titulo, resumo, assuntos, tipo, estado, criado_em, atualizado_em)
		SELECT fonte_id, 0, ?, fim, titulo || ' (continuação)', resumo, assuntos, tipo, estado, ?, ? FROM fragmentos WHERE id = ?`,
		em, agora(), agora(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET fim = ?, atualizado_em = ? WHERE id = ?`, em, agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := renumerarFragmentos(ctx, tx, fr.FonteID); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	novoID, _ := res.LastInsertId()
	fr, _ = a.fragmentoPorID(ctx, novoID)
	responderJSON(w, http.StatusCreated, fr)
}

// handleFragmentoJuntar: junta com o seguinte (início do primeiro, fim do segundo, títulos com " + ").
func (a *app) handleFragmentoJuntar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	fr, err := a.fragmentoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	prox, err := lerFragmento(a.db.QueryRowContext(ctx, `SELECT `+colunasFragmento+`
		WHERE fonte_id = ? AND estado != 'descartado' AND (inicio > ? OR (inicio = ? AND id > ?)) ORDER BY inicio, id LIMIT 1`,
		fr.FonteID, fr.Inicio, fr.Inicio, fr.ID))
	if errors.Is(err, sql.ErrNoRows) {
		a.responderErro(w, erroDeEntrada{errors.New("não tem fragmento seguinte para juntar")})
		return
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	estado := "sugerido"
	if fr.Estado == "confirmado" && prox.Estado == "confirmado" {
		estado = "confirmado"
	}
	assuntos := fr.Assuntos
	for _, s := range prox.Assuntos {
		if !contem(assuntos, s) {
			assuntos = append(assuntos, s)
		}
	}
	assuntosJSON, _ := json.Marshal(assuntos)
	resumo := strings.TrimSpace(strings.TrimSpace(fr.Resumo) + " " + strings.TrimSpace(prox.Resumo))
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET fim = ?, titulo = ?, resumo = ?, assuntos = ?, estado = ?, atualizado_em = ? WHERE id = ?`,
		prox.Fim, strings.TrimSpace(fr.Titulo+" + "+prox.Titulo), resumo, string(assuntosJSON), estado, agora(), fr.ID); err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM fragmentos WHERE id = ?`, prox.ID); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := renumerarFragmentos(ctx, tx, fr.FonteID); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	fr, _ = a.fragmentoPorID(ctx, fr.ID)
	responderJSON(w, http.StatusOK, fr)
}

func (a *app) handleFragmentoDescartar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	fr, err := a.fragmentoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE fragmentos SET estado = 'descartado', atualizado_em = ? WHERE id = ?`, agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	fr.Estado = "descartado"
	responderJSON(w, http.StatusOK, fr)
}
