package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Entrada, fase 1 (T35): ler um vídeo do YouTube sem IA. O yt-dlp devolve as informações e a
// legenda em json3; a transcrição em texto com marcas [mm:ss] vai para o conteúdo
// (entrada/<id>/transcricao.md), fora do banco e do GitHub. Código adaptado do Acervo
// (acervo/backend/youtube.go), sem as partes de Instagram e de Referência.

var idDeVideo = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

var (
	errLinkInvalido = errors.New("o link não é um vídeo nem uma playlist do YouTube")
	errSemLegenda   = errors.New("o vídeo não tem legenda")
	errSemWhisper   = errors.New("o vídeo não tem legenda e o Whisper não está configurado (WHISPER_URL)")
	errSemFala      = errors.New("a transcrição do Whisper voltou vazia: o áudio pode estar sem fala")
)

// videoDoLink aceita os formatos de link mais comuns e devolve o id do vídeo:
// youtube.com/watch?v=, youtu.be/, /shorts/, /live/, /embed/, m.youtube.com, com ou sem &t=.
func videoDoLink(bruto string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errLinkInvalido
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	var id string
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	case "youtube.com", "music.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else if partes := strings.Split(strings.Trim(u.Path, "/"), "/"); len(partes) == 2 &&
			(partes[0] == "shorts" || partes[0] == "live" || partes[0] == "embed") {
			id = partes[1]
		}
	}
	if !idDeVideo.MatchString(id) {
		return "", errLinkInvalido
	}
	return id, nil
}

// ehPlaylist: link com list= e sem v= (watch?list=… ou /playlist?list=…). O link com v= e list=
// é um vídeo só: o Lucas compartilhou um vídeo que estava numa playlist.
func ehPlaylist(bruto string) bool {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil {
		return false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	if host != "youtube.com" && host != "music.youtube.com" {
		return false
	}
	q := u.Query()
	return q.Get("list") != "" && q.Get("v") == ""
}

type infoVideo struct {
	ID           string                     `json:"id"`
	Titulo       string                     `json:"title"`
	Canal        string                     `json:"channel"`
	Duracao      float64                    `json:"duration"`
	DuracaoTexto string                     `json:"duration_string"`
	Idioma       string                     `json:"language"`
	URL          string                     `json:"webpage_url"`
	Publicado    string                     `json:"upload_date"`
	Capitulos    []capituloInfo             `json:"chapters"`
	Legendas     map[string]json.RawMessage `json:"subtitles"`
	Automaticas  map[string]json.RawMessage `json:"automatic_captions"`
}

// capituloInfo é um capítulo do autor do vídeo, como o yt-dlp devolve no -J.
type capituloInfo struct {
	Inicio float64 `json:"start_time"`
	Fim    float64 `json:"end_time"`
	Titulo string  `json:"title"`
}

// escolhaDeLegenda: manual no idioma do vídeo → automática original (-orig) →
// manual em português → qualquer manual → automática no idioma do vídeo.
func escolhaDeLegenda(info infoVideo) (lingua string, automatica bool, ok bool) {
	base := func(l string) string { return strings.SplitN(l, "-", 2)[0] }
	manuais := chaves(info.Legendas)
	autos := chaves(info.Automaticas)
	video := info.Idioma

	if video != "" {
		for _, l := range manuais {
			if l == video || base(l) == base(video) {
				return l, false, true
			}
		}
	}
	for _, l := range autos {
		if strings.HasSuffix(l, "-orig") {
			return l, true, true
		}
	}
	for _, l := range manuais {
		if base(l) == "pt" {
			return l, false, true
		}
	}
	if len(manuais) > 0 {
		return manuais[0], false, true
	}
	if video != "" {
		for _, l := range autos {
			if l == video {
				return l, true, true
			}
		}
	}
	return "", false, false
}

func chaves(m map[string]json.RawMessage) []string {
	var lista []string
	for k := range m {
		if k != "live_chat" {
			lista = append(lista, k)
		}
	}
	sort.Strings(lista)
	return lista
}

// idiomaBase devolve "pt", "en"... a partir de "pt-BR", "en-orig".
func idiomaBase(l string) string {
	return strings.ToLower(strings.SplitN(l, "-", 2)[0])
}

func (a *app) ytdlp(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.ytdlpBin, append(append([]string{}, a.ytdlpExtra...), args...)...)
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(erros.String())
		if i := strings.LastIndex(msg, "ERROR:"); i >= 0 {
			msg = msg[i:]
		}
		return nil, fmt.Errorf("yt-dlp falhou: %s", primeiraLinha(msg, err.Error()))
	}
	return saida.Bytes(), nil
}

// lerVideo é a tarefa da fila da Entrada: lê as informações e a legenda do vídeo e grava a
// transcrição no conteúdo. O que o yt-dlp achou fica na fonte mesmo quando a leitura falha.
func (a *app) lerVideo(ctx context.Context, f fonte) error {
	bruto, err := a.ytdlp(ctx, "-J", "--skip-download", "--no-playlist", f.URL)
	if err != nil {
		return err
	}
	var info infoVideo
	if err := json.Unmarshal(bruto, &info); err != nil {
		return fmt.Errorf("resposta do yt-dlp ilegível: %w", err)
	}

	lingua, automatica, temLegenda := escolhaDeLegenda(info)
	idioma := info.Idioma
	if idioma == "" {
		idioma = strings.TrimSuffix(lingua, "-orig")
	}
	capitulos, _ := json.Marshal(capitulosDaInfo(info))
	if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET titulo = ?, canal = ?, duracao = ?, idioma = ?, capitulos = ? WHERE id = ?`,
		info.Titulo, info.Canal, int(info.Duracao), idioma, string(capitulos), f.ID); err != nil {
		return err
	}

	var transcricao string
	switch {
	case temLegenda:
		transcricao, err = a.baixarLegenda(ctx, f.URL, lingua, automatica, info)
	case a.whisper == "":
		err = errSemWhisper
	default:
		transcricao, err = a.transcreverVideo(ctx, info, f.URL)
	}
	if err != nil {
		return err
	}
	return a.conteudo.gravar(fmt.Sprintf("entrada %d: transcrição lida", f.ID),
		map[string]*string{pastaDaFonte(f.ID) + "/transcricao.md": &transcricao})
}

func capitulosDaInfo(info infoVideo) []capitulo {
	lista := make([]capitulo, 0, len(info.Capitulos))
	for _, c := range info.Capitulos {
		lista = append(lista, capitulo{Inicio: c.Inicio, Fim: c.Fim, Titulo: c.Titulo})
	}
	return lista
}

// baixarLegenda baixa a legenda escolhida em json3 numa pasta temporária e monta a transcrição.
func (a *app) baixarLegenda(ctx context.Context, link, lingua string, automatica bool, info infoVideo) (string, error) {
	tmp, err := os.MkdirTemp("", "legenda-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	tipo := "--write-subs"
	if automatica {
		tipo = "--write-auto-subs"
	}
	if _, err := a.ytdlp(ctx, "--skip-download", "--no-playlist", tipo, "--sub-langs", lingua,
		"--sub-format", "json3", "-o", filepath.Join(tmp, "legenda"), link); err != nil {
		return "", err
	}
	legenda, err := os.ReadFile(filepath.Join(tmp, "legenda."+lingua+".json3"))
	if err != nil {
		return "", errSemLegenda
	}
	return legendaParaCru(legenda, info)
}

const blocoMs = 10_000 // uma marca de tempo a cada ~10 s (o minuto do fragmento depende disso)

var espacos = regexp.MustCompile(`\s+`)

// legendaParaCru transforma a legenda json3 em texto corrido com marcas [mm:ss].
func legendaParaCru(legenda []byte, info infoVideo) (string, error) {
	var dados struct {
		Events []struct {
			Inicio  int64 `json:"tStartMs"`
			Anexado int   `json:"aAppend"`
			Segs    []struct {
				Texto string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(legenda, &dados); err != nil {
		return "", fmt.Errorf("legenda ilegível: %w", err)
	}

	type bloco struct {
		inicio int64
		texto  []string
	}
	var blocos []bloco
	for _, ev := range dados.Events {
		if len(ev.Segs) == 0 || ev.Anexado != 0 {
			continue
		}
		var b strings.Builder
		for _, s := range ev.Segs {
			b.WriteString(s.Texto)
		}
		texto := strings.TrimSpace(strings.ReplaceAll(b.String(), "\n", " "))
		if texto == "" {
			continue
		}
		if len(blocos) == 0 || ev.Inicio-blocos[len(blocos)-1].inicio >= blocoMs {
			blocos = append(blocos, bloco{inicio: ev.Inicio})
		}
		atual := &blocos[len(blocos)-1]
		atual.texto = append(atual.texto, texto)
	}
	if len(blocos) == 0 {
		return "", errSemLegenda
	}

	var s strings.Builder
	s.WriteString(cabecalhoDaTranscricao(info, ""))
	for _, b := range blocos {
		fmt.Fprintf(&s, "[%s] %s\n\n", marcaDeTempo(b.inicio), espacos.ReplaceAllString(strings.Join(b.texto, " "), " "))
	}
	return s.String(), nil
}

// cabecalhoDaTranscricao: título, canal, link, duração e, se houver, os capítulos do autor.
func cabecalhoDaTranscricao(info infoVideo, extra string) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Canal: %s\n- URL: %s\n- Duração: %s\n", info.Titulo, info.Canal, info.URL, duracaoTexto(info))
	if info.Publicado != "" {
		fmt.Fprintf(&s, "- Publicado: %s\n", info.Publicado)
	}
	if extra != "" {
		fmt.Fprintf(&s, "- %s\n", extra)
	}
	if len(info.Capitulos) > 0 {
		s.WriteString("- Capítulos:\n")
		for _, c := range info.Capitulos {
			fmt.Fprintf(&s, "  - %s %s\n", marcaDeTempo(int64(c.Inicio*1000)), c.Titulo)
		}
	}
	s.WriteString("\n")
	return s.String()
}

func duracaoTexto(info infoVideo) string {
	if info.DuracaoTexto != "" {
		return info.DuracaoTexto
	}
	return marcaDeTempo(int64(info.Duracao * 1000))
}

func marcaDeTempo(ms int64) string {
	s := ms / 1000
	h, m, seg := s/3600, s%3600/60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, seg)
	}
	return fmt.Sprintf("%02d:%02d", m, seg)
}

// ---------- vídeo sem legenda: o Whisper transcreve o áudio (T35) ----------

var clienteDoWhisper = &http.Client{Timeout: 40 * time.Minute} // uma live pode ter horas de áudio

type trechoFalado struct {
	Inicio float64
	Texto  string
}

// transcreverVideo baixa só o áudio (que não fica guardado), manda ao Whisper e monta a mesma
// transcrição de uma legenda: cabeçalho + [mm:ss] texto.
func (a *app) transcreverVideo(ctx context.Context, info infoVideo, link string) (string, error) {
	tmp, err := os.MkdirTemp("", "video-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	if _, err := a.ytdlp(ctx, "-q", "--no-playlist", "-f", "bestaudio", "-o", filepath.Join(tmp, "audio.%(ext)s"), link); err != nil {
		return "", err
	}
	achados, _ := filepath.Glob(filepath.Join(tmp, "audio.*"))
	if len(achados) == 0 {
		return "", errors.New("o yt-dlp não baixou o áudio do vídeo")
	}
	trechos, err := a.pedirTranscricao(ctx, achados[0], info)
	if err != nil {
		return "", err
	}
	if len(trechos) == 0 {
		return "", errSemFala
	}
	return cabecalhoDaTranscricao(info, "Transcrição: Whisper (o vídeo não tem legenda)") + falasEmBlocos(trechos), nil
}

// pedirTranscricao manda o áudio ao Whisper (API da OpenAI, POST /v1/audio/transcriptions) e
// devolve os segmentos com o tempo de cada um (response_format=verbose_json).
func (a *app) pedirTranscricao(ctx context.Context, arquivo string, info infoVideo) ([]trechoFalado, error) {
	f, err := os.Open(arquivo)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	parte, _ := mw.CreateFormFile("file", filepath.Base(arquivo))
	if _, err := io.Copy(parte, f); err != nil {
		return nil, err
	}
	mw.WriteField("model", "whisper-large-v3")
	mw.WriteField("response_format", "verbose_json")
	if idioma := idiomaBase(info.Idioma); idioma != "" {
		mw.WriteField("language", idioma)
	}
	if prompt, _ := cortar(info.Titulo+". "+info.Canal, 300); prompt != "" {
		mw.WriteField("prompt", prompt)
	}
	mw.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.whisper+"/v1/audio/transcriptions", &corpo)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := clienteDoWhisper.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o Whisper respondeu %d: %s", res.StatusCode, primeiroErro(b))
	}
	var r struct {
		Segmentos []struct {
			Inicio float64 `json:"start"`
			Texto  string  `json:"text"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("resposta do Whisper ilegível: %w", err)
	}
	var trechos []trechoFalado
	for _, s := range r.Segmentos {
		if t := strings.TrimSpace(s.Texto); t != "" {
			trechos = append(trechos, trechoFalado{Inicio: s.Inicio, Texto: t})
		}
	}
	return trechos, nil
}

func primeiroErro(b []byte) string {
	s, _ := cortar(strings.TrimSpace(string(b)), 200)
	return s
}

// falasEmBlocos: uma marca [mm:ss] a cada ~10 s, como a transcrição de uma legenda.
func falasEmBlocos(trechos []trechoFalado) string {
	var s strings.Builder
	inicio := -1.0
	var bloco []string
	fechar := func() {
		if len(bloco) > 0 {
			fmt.Fprintf(&s, "[%s] %s\n\n", marcaDeTempo(int64(inicio*1000)), strings.Join(bloco, " "))
		}
		bloco = nil
	}
	for _, t := range trechos {
		if inicio < 0 || t.Inicio-inicio >= blocoMs/1000 {
			fechar()
			inicio = t.Inicio
		}
		bloco = append(bloco, t.Texto)
	}
	fechar()
	return s.String()
}

// expandirPlaylist cria uma fonte por vídeo da playlist (o --flat-playlist lê só os ids e os
// títulos; cada vídeo é lido na fila depois).
func (a *app) expandirPlaylist(ctx context.Context, link string) ([]fonte, error) {
	bruto, err := a.ytdlp(ctx, "--flat-playlist", "-J", link)
	if err != nil {
		return nil, err
	}
	var lista struct {
		Titulo   string `json:"title"`
		Entradas []struct {
			ID     string `json:"id"`
			Titulo string `json:"title"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(bruto, &lista); err != nil {
		return nil, fmt.Errorf("resposta do yt-dlp ilegível: %w", err)
	}
	if len(lista.Entradas) == 0 {
		return nil, erroDeEntrada{errors.New("a playlist está vazia")}
	}
	fontes := make([]fonte, 0, len(lista.Entradas))
	for _, e := range lista.Entradas {
		if !idDeVideo.MatchString(e.ID) {
			continue
		}
		f, err := a.criarFonte(ctx, "youtube", "https://www.youtube.com/watch?v="+e.ID, e.ID, e.Titulo, lista.Titulo)
		if err != nil {
			return nil, err
		}
		fontes = append(fontes, f)
	}
	return fontes, nil
}
