package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// descarta e confirma; a fase 4 dá o destino de cada um.

const tipoFragmentar = "fragmentar" // a tarefa da fila

var tiposFragmento = []string{"conteudo", "opiniao", "fora"}

type fragmento struct {
	ID           int64    `json:"id"`
	FonteID      int64    `json:"fonte_id"`
	Ordem        int      `json:"ordem"`
	Inicio       int      `json:"inicio"` // segundos
	Fim          int      `json:"fim"`    // segundos
	Titulo       string   `json:"titulo"`
	Resumo       string   `json:"resumo"`
	Assuntos     []string `json:"assuntos"`
	Tipo         string   `json:"tipo"`   // conteudo · opiniao · fora
	Estado       string   `json:"estado"` // sugerido · confirmado · descartado
	CriadoEm     string   `json:"criado_em"`
	AtualizadoEm string   `json:"atualizado_em"`
}

const colunasFragmento = `id, fonte_id, ordem, inicio, fim, titulo, resumo, assuntos, tipo, estado,
	criado_em, atualizado_em FROM fragmentos`

func lerFragmento(row interface{ Scan(...any) error }) (fragmento, error) {
	var f fragmento
	var assuntos string
	err := row.Scan(&f.ID, &f.FonteID, &f.Ordem, &f.Inicio, &f.Fim, &f.Titulo, &f.Resumo, &assuntos,
		&f.Tipo, &f.Estado, &f.CriadoEm, &f.AtualizadoEm)
	if err == nil {
		f.Assuntos = []string{}
		json.Unmarshal([]byte(assuntos), &f.Assuntos)
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

// fragmentar é a tarefa: a IA lê a transcrição uma vez e devolve os trechos por assunto, que o
// servidor confere e grava como sugerido.
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
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	if err := gravarNaPasta(pasta, pedidoDaFragmentacao(f, transcricao, confirmados)); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("fragmentar.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read", esquema: esquemaFragmentos, modelo: t.Modelo,
		instrucao: "Divida o vídeo em fragmentos por assunto. Tudo está em `pedido.md` e `transcricao.md`, nesta pasta: leia os dois inteiros antes.",
	})
	if err != nil {
		return err
	}
	var r struct {
		Fragmentos []fragmentoIA `json:"fragmentos"`
		Notas      string        `json:"notas"`
	}
	if err := dadosDaResposta(resp, &r); err != nil {
		return err
	}
	bons, tirados := limparFragmentos(f.Duracao, r.Fragmentos)
	if len(bons) == 0 {
		return fmt.Errorf("a IA não devolveu fragmentos válidos%s", entreParenteses(strings.TrimSpace(textoDaIA(r.Notas))))
	}
	nota := textoDaIA(r.Notas)
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
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET estado = 'fragmentada', erro = ? WHERE id = ?`, nota, f.ID); err != nil {
		return err
	}
	return tx.Commit()
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
	modelo := a.modeloPara(ctx, tipoFragmentar)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM fragmentos WHERE fonte_id = ? AND estado = 'sugerido'`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoFragmentar, 0, "Fragmentar: "+primeiraLinha(f.Titulo, f.URL), modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET estado = 'fragmentando', erro = '', tarefa_id = ?, pedido = ? WHERE id = ?`,
		tarefaID, strings.TrimSpace(corpo.Pedido), id); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	f, _ = a.entradaCompleta(ctx, id)
	responderJSON(w, http.StatusAccepted, f)
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
