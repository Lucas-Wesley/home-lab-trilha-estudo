package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Entrada (T35): a área para onde chegam os vídeos (do YouTube nesta fase; do Acervo e do
// Telegram nas próximas). Cada vídeo é uma fonte: as informações ficam no banco e a transcrição
// no conteúdo (entrada/<id>/transcricao.md).

type fonte struct {
	ID          int64      `json:"id"`
	Tipo        string     `json:"tipo"`
	URL         string     `json:"url"`
	VideoID     string     `json:"video_id"`
	Titulo      string     `json:"titulo"`
	Canal       string     `json:"canal"`
	Duracao     int        `json:"duracao"` // segundos
	Idioma      string     `json:"idioma"`
	Capitulos   []capitulo `json:"capitulos"`
	Estado      string     `json:"estado"` // na_fila · lendo · pronta · fragmentando · fragmentada · erro
	Erro        string     `json:"erro"`
	Playlist    string     `json:"playlist"`
	CriadoEm    string     `json:"criado_em"`
	Transcricao string     `json:"transcricao,omitempty"`
	// T37: a tarefa que fragmentou (ou fragmenta) a fonte, o pedido do Lucas para ela,
	// os fragmentos e a tarefa em si (só na página da fonte).
	TarefaID   int64       `json:"tarefa_id"`
	Pedido     string      `json:"pedido"`
	Notas      string      `json:"notas"` // o que a IA disse da divisão em fragmentos
	Fragmentos []fragmento `json:"fragmentos,omitempty"`
	Tarefa     *tarefa     `json:"tarefa,omitempty"`
}

// capitulo é um capítulo do autor do vídeo (início e fim em segundos), guardado em JSON na fonte.
type capitulo struct {
	Inicio float64 `json:"inicio"`
	Fim    float64 `json:"fim"`
	Titulo string  `json:"titulo"`
}

func pastaDaFonte(id int64) string {
	return "entrada/" + strconv.FormatInt(id, 10)
}

const colunasFonte = `id, tipo, url, video_id, titulo, canal, duracao, idioma, capitulos, estado, erro,
	playlist, criado_em, COALESCE(tarefa_id, 0), pedido, notas FROM fontes`

func lerFonte(row interface{ Scan(...any) error }) (fonte, error) {
	var f fonte
	var capitulos string
	err := row.Scan(&f.ID, &f.Tipo, &f.URL, &f.VideoID, &f.Titulo, &f.Canal, &f.Duracao, &f.Idioma,
		&capitulos, &f.Estado, &f.Erro, &f.Playlist, &f.CriadoEm, &f.TarefaID, &f.Pedido, &f.Notas)
	if err == nil {
		f.Capitulos = []capitulo{}
		json.Unmarshal([]byte(capitulos), &f.Capitulos)
	}
	return f, err
}

func (a *app) fontePorID(ctx context.Context, id int64) (fonte, error) {
	f, err := lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return f, errNaoEncontrou
	}
	return f, err
}

// criarFonte registra o vídeo na Entrada; o mesmo vídeo duas vezes não duplica (o UNIQUE):
// devolve a fonte que já existe.
func (a *app) criarFonte(ctx context.Context, tipo, link, videoID, titulo, playlist string) (fonte, error) {
	if _, err := a.db.ExecContext(ctx, `INSERT OR IGNORE INTO fontes (tipo, url, video_id, titulo, playlist, criado_em)
		VALUES (?, ?, ?, ?, ?, ?)`, tipo, link, videoID, titulo, playlist, agora()); err != nil {
		return fonte{}, err
	}
	return lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE tipo = ? AND video_id = ?`, tipo, videoID))
}

// ---------- a fila do yt-dlp ----------
// Uma leitura por vez, separada da fila do Claude (não usa crédito), com 15 s de pausa entre
// vídeos: o YouTube bloqueia quem pede muito seguido. Se o app para no meio, o que estava
// "lendo" volta para a fila no próximo início.

// limiteLeitura: o máximo para ler um vídeo (informações, legenda e, sem legenda, o áudio no
// Whisper, que numa live de 3 h pode levar uns 20 min no Mac).
const limiteLeitura = 45 * time.Minute

func (a *app) recuperarEntrada(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'na_fila' WHERE estado = 'lendo'`)
	return err
}

// acordarEntrada avisa o laço da Entrada que chegou fonte (chamar depois do commit).
func (a *app) acordarEntrada() {
	select {
	case a.acordaEntrada <- struct{}{}:
	default:
	}
}

func (a *app) rodarEntrada(ctx context.Context) {
	for {
		if !a.lerProximaFonte(ctx) {
			select {
			case <-ctx.Done():
				return
			case <-a.acordaEntrada:
			}
		}
	}
}

// lerProximaFonte pega a fonte mais antiga na fila e a lê; false quando não havia nenhuma.
func (a *app) lerProximaFonte(ctx context.Context) bool {
	var f fonte
	err := a.db.QueryRowContext(ctx, `SELECT id, url FROM fontes WHERE estado = 'na_fila' ORDER BY id LIMIT 1`).
		Scan(&f.ID, &f.URL)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			log.Printf("entrada: %v", err)
		}
		return false
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'lendo' WHERE id = ?`, f.ID); err != nil {
		log.Printf("entrada: %v", err)
		return false
	}
	// Um yt-dlp ou um Whisper travado não pode parar a fila para sempre.
	cl, cancelar := context.WithTimeout(ctx, limiteLeitura)
	err = a.lerVideo(cl, f)
	cancelar()
	if err != nil {
		log.Printf("entrada: fonte %d: %v", f.ID, err)
		a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'erro', erro = ? WHERE id = ?`, err.Error(), f.ID)
	} else {
		a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'pronta', erro = '' WHERE id = ?`, f.ID)
	}
	// A pausa entre vídeos (o YouTube bloqueia quem pede muito seguido); zero nos testes.
	select {
	case <-ctx.Done():
	case <-time.After(a.pausaEntrada):
	}
	return true
}

// esvaziarEntrada lê tudo o que está na fila, um por vez (os testes usam no lugar do laço).
func (a *app) esvaziarEntrada(ctx context.Context) {
	for a.lerProximaFonte(ctx) {
	}
}

// ---------- handlers ----------

func (a *app) handleEntradaPost(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		URL string `json:"url"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	link := strings.TrimSpace(corpo.URL)
	if link == "" {
		a.responderErro(w, erroDeEntrada{errors.New("mande o link do vídeo ou da playlist")})
		return
	}
	var fontes []fonte
	var err error
	if ehPlaylist(link) {
		fontes, err = a.expandirPlaylist(r.Context(), link)
	} else {
		var id string
		id, err = videoDoLink(link)
		if err == nil {
			var f fonte
			f, err = a.criarFonte(r.Context(), "youtube", "https://www.youtube.com/watch?v="+id, id, "", "")
			fontes = []fonte{f}
		}
	}
	if err != nil {
		if errors.Is(err, errLinkInvalido) {
			a.responderErro(w, erroDeEntrada{err})
		} else {
			a.responderErro(w, err)
		}
		return
	}
	a.acordarEntrada()
	responderJSON(w, http.StatusAccepted, fontes)
}

func (a *app) handleEntradaListar(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+colunasFonte+` ORDER BY id DESC`)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer rows.Close()
	lista := []fonte{}
	for rows.Next() {
		f, err := lerFonte(rows)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		lista = append(lista, f)
	}
	responderJSON(w, http.StatusOK, lista)
}

// entradaCompleta é a fonte da página: transcrição, fragmentos e a tarefa que a fragmentou (T37).
func (a *app) entradaCompleta(ctx context.Context, id int64) (fonte, error) {
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		return f, err
	}
	f.Transcricao, _ = a.conteudo.ler(pastaDaFonte(id) + "/transcricao.md")
	if f.Fragmentos, err = a.fragmentosDaFonte(ctx, id); err != nil {
		return f, err
	}
	if f.TarefaID > 0 {
		if t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, f.TarefaID)); err == nil {
			f.Tarefa = t
		}
	}
	return f, nil
}

func (a *app) handleEntradaFonte(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	f, err := a.entradaCompleta(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, f)
}

func (a *app) handleEntradaTentar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	f, err := a.fontePorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if f.Estado != "erro" {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("só dá para tentar de novo uma fonte em erro (esta está %s)", f.Estado)})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE fontes SET estado = 'na_fila', erro = '' WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.acordarEntrada()
	f.Estado, f.Erro = "na_fila", ""
	responderJSON(w, http.StatusOK, f)
}

func (a *app) handleEntradaExcluir(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.fontePorID(r.Context(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM fontes WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.conteudo.apagarPasta(fmt.Sprintf("entrada %d: fonte excluída", id), pastaDaFonte(id)); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
