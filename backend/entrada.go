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
	// T38: a tarefa que sugere as trilhas dos fragmentos confirmados.
	TarefaDestinoID int64   `json:"tarefa_destino_id"`
	PedidoDestino   string  `json:"pedido_destino"`
	TarefaDestino   *tarefa `json:"tarefa_destino,omitempty"`
	// T44: a fonte telegram marcada "dividir em fragmentos quando terminar".
	Fragmentar bool `json:"fragmentar"`
	// T38: na lista, "5 fragmentos · 3 enviados · 1 sem destino".
	ResumoFragmentos string `json:"resumo_fragmentos,omitempty"`
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
	playlist, criado_em, COALESCE(tarefa_id, 0), pedido, notas, COALESCE(tarefa_destino, 0), pedido_destino,
	fragmentar
	FROM fontes`

func lerFonte(row interface{ Scan(...any) error }) (fonte, error) {
	var f fonte
	var capitulos string
	var fragmentar int
	err := row.Scan(&f.ID, &f.Tipo, &f.URL, &f.VideoID, &f.Titulo, &f.Canal, &f.Duracao, &f.Idioma,
		&capitulos, &f.Estado, &f.Erro, &f.Playlist, &f.CriadoEm, &f.TarefaID, &f.Pedido, &f.Notas,
		&f.TarefaDestinoID, &f.PedidoDestino, &fragmentar)
	if err == nil {
		f.Capitulos = []capitulo{}
		json.Unmarshal([]byte(capitulos), &f.Capitulos)
		f.Fragmentar = fragmentar != 0
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
	// A que estava "baixando" fica como está: o download continua no Streaming e a sonda a
	// traz de volta para a fila. A que estava lendo/transcrevendo recomeça na fila.
	_, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'na_fila' WHERE estado IN ('lendo', 'transcrevendo')`)
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
			case <-time.After(a.sondaTelegram):
				// T44: a fonte que está "baixando" volta para a fila e é verificada de novo —
				// assim a fila nunca fica parada esperando um download do Telegram.
				a.promoverBaixando(ctx)
			}
		}
	}
}

// lerProximaFonte pega a fonte mais antiga na fila e a lê; false quando não havia nenhuma.
func (a *app) lerProximaFonte(ctx context.Context) bool {
	var f fonte
	var fragmentar int
	err := a.db.QueryRowContext(ctx, `SELECT id, url, video_id, tipo, titulo, fragmentar FROM fontes WHERE estado = 'na_fila' ORDER BY id LIMIT 1`).
		Scan(&f.ID, &f.URL, &f.VideoID, &f.Tipo, &f.Titulo, &fragmentar)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			log.Printf("entrada: %v", err)
		}
		return false
	}
	f.Fragmentar = fragmentar != 0
	if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'lendo' WHERE id = ?`, f.ID); err != nil {
		log.Printf("entrada: %v", err)
		return false
	}
	// Um yt-dlp, um ffmpeg ou um Whisper travado não pode parar a fila para sempre.
	cl, cancelar := context.WithTimeout(ctx, limiteLeitura)
	if f.Tipo == "telegram" {
		// T44: devolve o estado final — "baixando" (o download foi pedido e a fonte espera
		// a sonda a trazer de volta) ou "pronta" (transcrita).
		var estado string
		estado, err = a.lerTelegram(cl, f)
		cancelar()
		if err != nil {
			log.Printf("entrada: fonte %d: %v", f.ID, err)
			a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'erro', erro = ? WHERE id = ?`, err.Error(), f.ID)
		} else {
			a.db.ExecContext(ctx, `UPDATE fontes SET estado = ?, erro = '' WHERE id = ?`, estado, f.ID)
			// "Dividir em fragmentos quando terminar": pronta → a fila do Claude.
			if estado == "pronta" && f.Fragmentar {
				if err := a.pedirFragmentar(ctx, f.ID, f.Titulo, ""); err != nil {
					log.Printf("entrada: fonte %d, fragmentar: %v", f.ID, err)
				}
			}
		}
	} else {
		err = a.lerVideo(cl, f)
		cancelar()
		if err != nil {
			log.Printf("entrada: fonte %d: %v", f.ID, err)
			a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'erro', erro = ? WHERE id = ?`, err.Error(), f.ID)
		} else {
			a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'pronta', erro = '' WHERE id = ?`, f.ID)
			if f.Fragmentar {
				a.dividirSozinho(ctx, f.ID)
			}
		}
	}
	// T53: a verificação de material que esperava esta transcrição segue para a IA.
	a.avancarVerificacoes(ctx)
	// A pausa entre vídeos (o YouTube bloqueia quem pede muito seguido); zero nos testes.
	select {
	case <-ctx.Done():
	case <-time.After(a.pausaEntrada):
	}
	return true
}

// limiteAutomatico: vídeo mais longo que isso não é dividido sozinho (T46). A divisão lê a
// transcrição inteira (em partes, T48); um curso de 13 h são uns 200 mil tokens, e aí o Lucas decide.
var limiteAutomatico = 3 * 60 * 60 // var: os testes baixam

// dividirSozinho: o modo automático depois da legenda — divide, ou deixa uma nota se é longo.
func (a *app) dividirSozinho(ctx context.Context, id int64) {
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		return
	}
	if f.Duracao > limiteAutomatico {
		nota := fmt.Sprintf("Vídeo de %s: não dividi sozinho porque gasta bastante do limite da assinatura. Divida quando quiser: vai em partes de umas 2 h (dá para pedir só um trecho).", tempoDos(f.Duracao))
		a.db.ExecContext(ctx, `UPDATE fontes SET notas = ? WHERE id = ?`, nota, id)
		return
	}
	if err := a.pedirFragmentar(ctx, id, f.Titulo, ""); err != nil {
		log.Printf("entrada: fonte %d, fragmentar: %v", id, err)
	}
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
		// T46: dividir em fragmentos e sugerir as trilhas sozinho quando a legenda chegar.
		Automatico bool `json:"automatico"`
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
	if corpo.Automatico {
		for i := range fontes {
			if _, err := a.db.ExecContext(r.Context(), `UPDATE fontes SET fragmentar = 1 WHERE id = ? AND estado = 'na_fila'`, fontes[i].ID); err == nil {
				fontes[i].Fragmentar = true
			}
		}
	}
	a.acordarEntrada()
	responderJSON(w, http.StatusAccepted, fontes)
}

func (a *app) handleEntradaListar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasFonte+` ORDER BY id DESC`)
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
	if err := rows.Err(); err != nil {
		a.responderErro(w, err)
		return
	}
	rows.Close()
	resumos, err := a.resumosDaEntrada(ctx)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	for i := range lista {
		lista[i].ResumoFragmentos = resumos[lista[i].ID]
	}
	responderJSON(w, http.StatusOK, lista)
}

// resumosDaEntrada: por fonte, "N fragmentos · X enviados · Y sem destino" (só o que há).
func (a *app) resumosDaEntrada(ctx context.Context) (map[int64]string, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT fonte_id, estado, COUNT(*) FROM fragmentos
		WHERE estado != 'descartado' GROUP BY fonte_id, estado`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	porFonte := map[int64]map[string]int{}
	for rows.Next() {
		var fonteID, n int64
		var estado string
		if err := rows.Scan(&fonteID, &estado, &n); err != nil {
			return nil, err
		}
		if porFonte[fonteID] == nil {
			porFonte[fonteID] = map[string]int{}
		}
		porFonte[fonteID][estado] = int(n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	resumos := map[int64]string{}
	for fonteID, m := range porFonte {
		total := 0
		for _, n := range m {
			total += n
		}
		partes := []string{fmt.Sprintf("%d fragmentos", total)}
		if m["enviado"] > 0 {
			partes = append(partes, fmt.Sprintf("%d enviados", m["enviado"]))
		}
		if m["sem_destino"] > 0 {
			partes = append(partes, fmt.Sprintf("%d sem destino", m["sem_destino"]))
		}
		resumos[fonteID] = strings.Join(partes, " · ")
	}
	// T46: as trilhas marcadas (pela IA ou pelo Lucas) e ainda não enviadas: "→ AWS, Redes".
	trows, err := a.db.QueryContext(ctx, `SELECT DISTINCT f.fonte_id, t.titulo FROM fragmento_destinos d
		JOIN fragmentos f ON f.id = d.fragmento_id JOIN trilhas t ON t.id = d.trilha_id
		WHERE d.proposta_id IS NULL AND f.estado IN ('sugerido', 'confirmado') ORDER BY f.fonte_id, t.titulo`)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	sugeridas := map[int64][]string{}
	for trows.Next() {
		var fonteID int64
		var titulo string
		if err := trows.Scan(&fonteID, &titulo); err != nil {
			return nil, err
		}
		sugeridas[fonteID] = append(sugeridas[fonteID], titulo)
	}
	for fonteID, nomes := range sugeridas {
		resumos[fonteID] += " · → " + strings.Join(nomes, ", ")
	}
	return resumos, trows.Err()
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
	destinos, err := a.destinosDaFonte(ctx, id)
	if err != nil {
		return f, err
	}
	for i := range f.Fragmentos {
		f.Fragmentos[i].Destinos = destinos[f.Fragmentos[i].ID]
	}
	if f.TarefaID > 0 {
		if t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, f.TarefaID)); err == nil {
			f.Tarefa = t
		}
	}
	if f.TarefaDestinoID > 0 {
		if t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, f.TarefaDestinoID)); err == nil {
			f.TarefaDestino = t
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
