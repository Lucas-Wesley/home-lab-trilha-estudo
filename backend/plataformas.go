package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Cursos de plataformas (T50, T51, T55): as aulas da Branas.io, LINUXtips, Full Cycle, Tech Leads Club e Chef Enzo Neto, baixadas
// em vídeo, na qualidade máxima, para a aba "Cursos" da Entrada. Os donos da Branas.io e da LINUXtips autorizaram o Lucas; nas
// outras ele é aluno. T55: o vídeo pode vir também embutido do YouTube ou do Vimeo (baixado pelo yt-dlp), a aula pode ser só
// texto (a página sem vídeo), e o curso pode virar trilha sozinho (cursos_trilha.go).
// A extensão do Chrome (E15 de lá) acha o manifesto HLS no player e manda com o nome da aula;
// aqui o ffmpeg baixa (decifra o AES-128 da LINUXtips com a chave que o próprio servidor entrega
// a quem manda o Referer do player) e grava em CURSOS_DIR/<plataforma>/<curso>/<módulo>/<aula>.mp4.
// Um download por vez, numa fila própria (não gasta crédito do Claude).

var plataformasCurso = map[string]string{
	"branas":    "Branas.io",
	"linuxtips": "LINUXtips",
	"fullcycle": "Full Cycle",
	"techleads": "Tech Leads Club",
	"enzoneto":  "Chef Enzo Neto",
}

const agenteNavegador = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/157.0.0.0 Safari/537.36"

type aulaCurso struct {
	ID           int64       `json:"id"`
	Plataforma   string      `json:"plataforma"`
	Curso        string      `json:"curso"`
	Modulo       string      `json:"modulo"`
	Titulo       string      `json:"titulo"`
	Pagina       string      `json:"pagina"`
	Ordem        int         `json:"ordem"`
	Texto        string      `json:"texto,omitempty"` // só no GET de uma aula (a lista leva tem_texto)
	TemTexto     bool        `json:"tem_texto"`
	Anexos       []anexoAula `json:"anexos"` // T56: os PDFs da aula, guardados na Trilha
	Embed        string      `json:"-"`      // YouTube/Vimeo, quando o vídeo não vem por HLS
	Manifesto    string      `json:"-"`
	Referer      string      `json:"-"`
	Variante     string      `json:"-"`
	Qualidade    string      `json:"qualidade"`
	Duracao      int         `json:"duracao"`
	Estimado     int64       `json:"tamanho_estimado"`
	Tamanho      int64       `json:"tamanho"`
	Progresso    int         `json:"progresso"`
	Estado       string      `json:"estado"` // na_fila · baixando · pronta · erro · texto (aula sem vídeo, só o texto da página)
	Erro         string      `json:"erro"`
	Arquivo      string      `json:"-"` // relativo a CURSOS_DIR
	CriadoEm     string      `json:"criado_em"`
	AtualizadoEm string      `json:"atualizado_em"`
}

const esquemaCursos = `
CREATE TABLE IF NOT EXISTS aulas_plataforma (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	plataforma    TEXT NOT NULL,
	curso         TEXT NOT NULL,
	modulo        TEXT NOT NULL DEFAULT '',
	titulo        TEXT NOT NULL,
	pagina        TEXT NOT NULL,         -- o endereço da aula na plataforma (na LINUXtips é o do curso)
	video         TEXT NOT NULL UNIQUE,  -- a chave: o HLS da variante escolhida, sem a query (youtube:<id>, vimeo:<id>, pagina:<url>)
	manifesto     TEXT NOT NULL,         -- o HLS que o player pediu (mestre ou de uma variante)
	referer       TEXT NOT NULL,         -- o que o player manda (o servidor do vídeo confere)
	variante      TEXT NOT NULL DEFAULT '', -- o HLS da maior qualidade, escolhido na estimativa
	qualidade     TEXT NOT NULL DEFAULT '',
	duracao       INTEGER NOT NULL DEFAULT 0,
	estimado      INTEGER NOT NULL DEFAULT 0,
	tamanho       INTEGER NOT NULL DEFAULT 0,
	progresso     INTEGER NOT NULL DEFAULT 0,
	estado        TEXT NOT NULL DEFAULT 'na_fila',
	erro          TEXT NOT NULL DEFAULT '',
	arquivo       TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL,
	ordem         INTEGER NOT NULL DEFAULT 0,  -- T55: a posição no curso (a extensão manda na ordem da plataforma)
	texto         TEXT NOT NULL DEFAULT '',    -- T55: o texto da página da aula (transcrição, notas, anexos)
	embed         TEXT NOT NULL DEFAULT '',    -- T55: o vídeo do YouTube/Vimeo, quando não há HLS
	na_trilha     INTEGER NOT NULL DEFAULT 0   -- T55: já entrou na trilha do curso (apagar a etapa não a traz de volta)
);

-- T55: o curso que virou trilha (uma por plataforma e curso; a trilha apagada solta o curso).
CREATE TABLE IF NOT EXISTS cursos_trilha (
	plataforma TEXT NOT NULL,
	curso      TEXT NOT NULL,
	trilha_id  INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	PRIMARY KEY (plataforma, curso)
);
`

// ---------- o manifesto ----------

type varianteHLS struct {
	URL     string
	Banda   int64
	Largura int
	Altura  int
}

// estimativa: a maior qualidade do manifesto, a duração e o tamanho previsto.
type estimativa struct {
	Variante  string `json:"-"`
	Qualidade string `json:"qualidade"`
	Duracao   int    `json:"duracao"`
	Tamanho   int64  `json:"tamanho_estimado"`
	Cifrado   bool   `json:"cifrado"`
}

var reAtributo = regexp.MustCompile(`([A-Z0-9-]+)=("[^"]*"|[^,]*)`)

func atributosM3U8(linha string) map[string]string {
	attrs := map[string]string{}
	if i := strings.IndexByte(linha, ':'); i >= 0 {
		for _, m := range reAtributo.FindAllStringSubmatch(linha[i+1:], -1) {
			attrs[m[1]] = strings.Trim(m[2], `"`)
		}
	}
	return attrs
}

func resolverURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

// variantesHLS: as variantes de um manifesto mestre (vazio se for o de uma variante só).
func variantesHLS(base, texto string) []varianteHLS {
	var vs []varianteHLS
	linhas := strings.Split(strings.ReplaceAll(texto, "\r", ""), "\n")
	for i, l := range linhas {
		if !strings.HasPrefix(l, "#EXT-X-STREAM-INF:") {
			continue
		}
		a := atributosM3U8(l)
		v := varianteHLS{}
		v.Banda, _ = strconv.ParseInt(a["BANDWIDTH"], 10, 64)
		if avg, err := strconv.ParseInt(a["AVERAGE-BANDWIDTH"], 10, 64); err == nil && avg > 0 {
			v.Banda = avg
		}
		if w, h, ok := strings.Cut(a["RESOLUTION"], "x"); ok {
			v.Largura, _ = strconv.Atoi(w)
			v.Altura, _ = strconv.Atoi(h)
		}
		for j := i + 1; j < len(linhas); j++ {
			if u := strings.TrimSpace(linhas[j]); u != "" && !strings.HasPrefix(u, "#") {
				v.URL = resolverURL(base, u)
				break
			}
		}
		if v.URL != "" {
			vs = append(vs, v)
		}
	}
	return vs
}

// melhorVariante: a de maior resolução; no empate, a de mais banda.
func melhorVariante(vs []varianteHLS) varianteHLS {
	sort.SliceStable(vs, func(i, j int) bool {
		if vs[i].Altura != vs[j].Altura {
			return vs[i].Altura > vs[j].Altura
		}
		return vs[i].Banda > vs[j].Banda
	})
	return vs[0]
}

// pedacosHLS: os pedaços de um manifesto de variante, a duração somada e se há chave.
func pedacosHLS(base, texto string) (pedacos []string, duracao float64, cifrado bool) {
	proximoEhPedaco := false
	for _, l := range strings.Split(strings.ReplaceAll(texto, "\r", ""), "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "#EXTINF:"):
			n := strings.TrimSuffix(strings.SplitN(strings.TrimPrefix(l, "#EXTINF:"), ",", 2)[0], ",")
			d, _ := strconv.ParseFloat(n, 64)
			duracao += d
			proximoEhPedaco = true
		case strings.HasPrefix(l, "#EXT-X-KEY:"):
			if m := atributosM3U8(l)["METHOD"]; m != "" && m != "NONE" {
				cifrado = true
			}
		case l != "" && !strings.HasPrefix(l, "#") && proximoEhPedaco:
			pedacos = append(pedacos, resolverURL(base, l))
			proximoEhPedaco = false
		}
	}
	return
}

func (a *app) clienteVideo() *http.Client {
	if a.clienteCursos != nil {
		return a.clienteCursos
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (a *app) pedirVideo(ctx context.Context, metodo, endereco, referer string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, metodo, endereco, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agenteNavegador)
	req.Header.Set("Referer", referer)
	resp, err := a.clienteVideo().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("o servidor do vídeo respondeu %d", resp.StatusCode)
	}
	return resp, nil
}

func (a *app) lerManifesto(ctx context.Context, endereco, referer string) (string, error) {
	resp, err := a.pedirVideo(ctx, http.MethodGet, endereco, referer)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(strings.TrimSpace(string(b)), "#EXTM3U") {
		return "", errors.New("o endereço não devolveu um manifesto HLS")
	}
	return string(b), nil
}

// estimar lê o manifesto, escolhe a maior qualidade e prevê o tamanho pelo tamanho real de três
// pedaços (começo, meio e fim): a BANDWIDTH do manifesto é o pico e chega a errar 3×.
func (a *app) estimar(ctx context.Context, manifesto, referer string) (estimativa, error) {
	texto, err := a.lerManifesto(ctx, manifesto, referer)
	if err != nil {
		return estimativa{}, err
	}
	e := estimativa{Variante: manifesto}
	var banda int64
	if vs := variantesHLS(manifesto, texto); len(vs) > 0 {
		v := melhorVariante(vs)
		e.Variante, banda = v.URL, v.Banda
		if v.Altura > 0 {
			e.Qualidade = fmt.Sprintf("%dp", v.Altura)
		}
		if texto, err = a.lerManifesto(ctx, v.URL, referer); err != nil {
			return estimativa{}, err
		}
	}
	pedacos, duracao, cifrado := pedacosHLS(e.Variante, texto)
	if len(pedacos) == 0 {
		return estimativa{}, errors.New("o manifesto não tem pedaços de vídeo")
	}
	e.Duracao, e.Cifrado = int(duracao+0.5), cifrado
	var soma, medidos int64
	for _, i := range []int{0, len(pedacos) / 2, len(pedacos) - 1} {
		if resp, err := a.pedirVideo(ctx, http.MethodHead, pedacos[i], referer); err == nil {
			resp.Body.Close()
			if resp.ContentLength > 0 {
				soma += resp.ContentLength
				medidos++
			}
		}
	}
	if medidos > 0 {
		e.Tamanho = soma / medidos * int64(len(pedacos))
	} else {
		e.Tamanho = banda * int64(e.Duracao) / 8
	}
	return e, nil
}

// chaveDoVideo: o mesmo vídeo pedido de novo (de outra página ou com outro token na query).
func chaveDoVideo(variante string) string {
	if u, err := url.Parse(variante); err == nil {
		u.RawQuery, u.Fragment = "", ""
		return u.String()
	}
	return variante
}

// ---------- o arquivo ----------

var reNomeRuim = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)

func nomeDePasta(s string) string {
	s = strings.Join(strings.Fields(reNomeRuim.ReplaceAllString(s, " ")), " ")
	s = strings.Trim(s, ". ")
	if len(s) > 120 {
		s = strings.TrimSpace(s[:120])
	}
	if s == "" {
		return "sem nome"
	}
	return s
}

// caminhoDaAula: <plataforma>/<curso>/<módulo>/<aula>.mp4 (sem módulo, direto na pasta do curso).
func caminhoDaAula(au aulaCurso) string {
	partes := []string{nomeDePasta(plataformasCurso[au.Plataforma]), nomeDePasta(au.Curso)}
	if strings.TrimSpace(au.Modulo) != "" {
		partes = append(partes, nomeDePasta(au.Modulo))
	}
	partes = append(partes, nomeDePasta(au.Titulo)+".mp4")
	return filepath.Join(partes...)
}

// argsFFmpeg: baixa o HLS sem recodificar (cópia direta) para um MP4 que toca no navegador.
// Os -allowed_* e -extension_picky aceitam os pedaços ".dts" da LINUXtips (Bunny).
func argsFFmpeg(variante, referer, saida string) []string {
	return []string{"-nostdin", "-y", "-loglevel", "error", "-progress", "pipe:1",
		"-user_agent", agenteNavegador, "-headers", "Referer: " + referer + "\r\n",
		"-allowed_extensions", "ALL", "-allowed_segment_extensions", "ALL", "-extension_picky", "0",
		"-i", variante, "-c", "copy", "-bsf:a", "aac_adtstoasc", "-movflags", "+faststart", "-f", "mp4", saida}
}

func espacoLivre(pasta string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(pasta, &st); err != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize)
}

// ---------- o banco ----------

const colunasAula = `id, plataforma, curso, modulo, titulo, pagina, manifesto, referer, variante, qualidade,
	duracao, estimado, tamanho, progresso, estado, erro, arquivo, criado_em, atualizado_em, ordem, texto, embed`

func lerAula(s interface{ Scan(...any) error }) (aulaCurso, error) {
	var au aulaCurso
	err := s.Scan(&au.ID, &au.Plataforma, &au.Curso, &au.Modulo, &au.Titulo, &au.Pagina, &au.Manifesto, &au.Referer,
		&au.Variante, &au.Qualidade, &au.Duracao, &au.Estimado, &au.Tamanho, &au.Progresso, &au.Estado, &au.Erro,
		&au.Arquivo, &au.CriadoEm, &au.AtualizadoEm, &au.Ordem, &au.Texto, &au.Embed)
	au.TemTexto = strings.TrimSpace(au.Texto) != ""
	return au, err
}

func (a *app) aulaPorID(ctx context.Context, id int64) (aulaCurso, error) {
	au, err := lerAula(a.db.QueryRowContext(ctx, `SELECT `+colunasAula+` FROM aulas_plataforma WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return au, errNaoEncontrou
	}
	return au, err
}

// ---------- a fila de downloads ----------

type downloadAtual struct {
	mu       sync.Mutex
	id       int64
	cancelar context.CancelFunc
}

func (a *app) acordarCursos() {
	select {
	case a.acordaCursos <- struct{}{}:
	default:
	}
}

// recuperarCursos: o download que estava no meio quando o app parou volta para a fila.
func (a *app) recuperarCursos(ctx context.Context) error {
	_, err := a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET estado = 'na_fila', progresso = 0 WHERE estado = 'baixando'`)
	return err
}

func (a *app) rodarCursos(ctx context.Context) {
	for {
		for a.baixarProxima(ctx) {
		}
		select {
		case <-ctx.Done():
			return
		case <-a.acordaCursos:
		case <-time.After(time.Minute):
		}
	}
}

// discoDosCursos (T57): os cursos ficam no SSD externo, montado com nofail. Se ele não subiu ou caiu,
// /srv/ssd é uma pasta vazia do disco do sistema e o vídeo iria parar lá sem ninguém ver. Com
// CURSOS_MARCA, a pasta dos cursos só vale com o arquivo-marca dentro; sem ele, a fila espera.
func (a *app) discoDosCursos() error {
	if a.cursosDir == "" {
		return errors.New("CURSOS_DIR não configurado")
	}
	if a.cursosMarca == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(a.cursosDir, a.cursosMarca)); err != nil {
		return errors.New("o disco dos cursos (o SSD) não está montado: os downloads esperam ele voltar")
	}
	return nil
}

func (a *app) baixarProxima(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	// Sem o disco, nada sai da fila: a aula fica "na fila" e a fila tenta de novo em 1 min.
	if err := a.discoDosCursos(); err != nil {
		var n int
		if a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM aulas_plataforma WHERE estado = 'na_fila'`).Scan(&n) == nil && n > 0 {
			log.Printf("cursos: %d aula(s) esperando: %v", n, err)
		}
		return false
	}
	au, err := lerAula(a.db.QueryRowContext(ctx, `SELECT `+colunasAula+` FROM aulas_plataforma WHERE estado = 'na_fila' ORDER BY id LIMIT 1`))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("cursos: %v", err)
		}
		return false
	}
	cl, cancelar := context.WithTimeout(ctx, 6*time.Hour)
	a.download.mu.Lock()
	a.download.id, a.download.cancelar = au.ID, cancelar
	a.download.mu.Unlock()
	defer func() {
		cancelar()
		a.download.mu.Lock()
		a.download.id, a.download.cancelar = 0, nil
		a.download.mu.Unlock()
	}()
	a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET estado = 'baixando', progresso = 0, erro = '', atualizado_em = ? WHERE id = ?`, agora(), au.ID)
	tamanho, arquivo, err := a.baixarAula(cl, au)
	if errors.Is(cl.Err(), context.Canceled) && ctx.Err() == nil {
		return true // apagada no meio: a linha já não existe
	}
	if err != nil {
		log.Printf("cursos: aula %d: %v", au.ID, err)
		a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET estado = 'erro', erro = ?, atualizado_em = ? WHERE id = ?`, err.Error(), agora(), au.ID)
		return true
	}
	a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET estado = 'pronta', progresso = 100, tamanho = ?, arquivo = ?, erro = '', atualizado_em = ? WHERE id = ?`,
		tamanho, arquivo, agora(), au.ID)
	return true
}

// destinoDaAula: o caminho do vídeo (relativo a CURSOS_DIR e o absoluto), com a pasta já criada.
func (a *app) destinoDaAula(ctx context.Context, au aulaCurso) (string, string, error) {
	rel := caminhoDaAula(au)
	final := filepath.Join(a.cursosDir, rel)
	// Outra aula com o mesmo nome no mesmo lugar: o id desempata.
	if _, err := os.Stat(final); err == nil {
		var outra int64
		if a.db.QueryRowContext(ctx, `SELECT id FROM aulas_plataforma WHERE arquivo = ? AND id <> ?`, rel, au.ID).Scan(&outra) == nil {
			rel = strings.TrimSuffix(rel, ".mp4") + fmt.Sprintf(" (%d).mp4", au.ID)
			final = filepath.Join(a.cursosDir, rel)
		}
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return "", "", fmt.Errorf("não deu para criar a pasta: %w", err)
	}
	return rel, final, nil
}

func (a *app) baixarAula(ctx context.Context, au aulaCurso) (int64, string, error) {
	if a.cursosDir == "" {
		return 0, "", errors.New("CURSOS_DIR não configurado")
	}
	if au.Embed != "" {
		return a.baixarEmbed(ctx, au)
	}
	variante := au.Variante
	if variante == "" {
		e, err := a.estimar(ctx, au.Manifesto, au.Referer)
		if err != nil {
			return 0, "", err
		}
		variante = e.Variante
	}
	rel, final, err := a.destinoDaAula(ctx, au)
	if err != nil {
		return 0, "", err
	}
	temp := final + ".baixando"
	defer os.Remove(temp)

	cmd := exec.CommandContext(ctx, a.ffmpegBin, argsFFmpeg(variante, au.Referer, temp)...)
	saida, err := cmd.StdoutPipe()
	if err != nil {
		return 0, "", err
	}
	var erros strings.Builder
	cmd.Stderr = &erros
	if err := cmd.Start(); err != nil {
		return 0, "", fmt.Errorf("ffmpeg: %w", err)
	}
	// O -progress escreve out_time_us=…: vira o percentual, gravado no máximo a cada 2 s.
	ultimo := time.Time{}
	sc := bufio.NewScanner(saida)
	for sc.Scan() {
		v, ok := strings.CutPrefix(sc.Text(), "out_time_us=")
		if !ok || au.Duracao <= 0 || time.Since(ultimo) < 2*time.Second {
			continue
		}
		us, _ := strconv.ParseInt(v, 10, 64)
		p := int(us / 10000 / int64(au.Duracao))
		if p > 99 {
			p = 99
		}
		ultimo = time.Now()
		a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET progresso = ? WHERE id = ?`, p, au.ID)
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(erros.String())
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		return 0, "", fmt.Errorf("o ffmpeg falhou: %v %s", err, msg)
	}
	info, err := os.Stat(temp)
	if err != nil || info.Size() == 0 {
		return 0, "", errors.New("o ffmpeg terminou sem gravar o vídeo")
	}
	if err := os.Rename(temp, final); err != nil {
		return 0, "", err
	}
	return info.Size(), rel, nil
}

// ---------- handlers ----------

type pedidoAula struct {
	Plataforma string `json:"plataforma"`
	Curso      string `json:"curso"`
	Modulo     string `json:"modulo"`
	Titulo     string `json:"titulo"`
	Pagina     string `json:"pagina"`
	Manifesto  string `json:"manifesto"`
	Referer    string `json:"referer"`
	// T55: o vídeo embutido (YouTube/Vimeo) quando não há HLS; o texto da página; a posição no
	// curso; e se o curso deve virar (ou atualizar) a trilha dele.
	Embed  string `json:"embed"`
	Texto  string `json:"texto"`
	Ordem  int    `json:"ordem"`
	Trilha bool   `json:"trilha"`
}

const limiteTextoAula = 200000 // caracteres: uma transcrição de 2 h tem uns 120 mil

func validarVideo(manifesto, referer string) error {
	m, err := url.Parse(manifesto)
	if err != nil || m.Scheme != "https" || !strings.Contains(m.Path, ".m3u8") {
		return erroDeEntrada{errors.New("o manifesto precisa ser um endereço https de .m3u8")}
	}
	if r, err := url.Parse(referer); err != nil || r.Scheme != "https" {
		return erroDeEntrada{errors.New("o referer precisa ser um endereço https")}
	}
	return nil
}

// videoDoPedido: de onde vem o vídeo (hls, embed ou nenhum) e a estimativa dele.
func (a *app) videoDoPedido(ctx context.Context, p *pedidoAula) (string, estimativa, error) {
	switch {
	case p.Manifesto != "":
		if err := validarVideo(p.Manifesto, p.Referer); err != nil {
			return "", estimativa{}, err
		}
		e, err := a.estimar(ctx, p.Manifesto, p.Referer)
		return "hls", e, err
	case p.Embed != "":
		baixar, _, ok := normalizarEmbed(p.Embed)
		if !ok {
			return "", estimativa{}, erroDeEntrada{errors.New("o vídeo embutido precisa ser do YouTube ou do Vimeo")}
		}
		p.Embed = baixar
		if p.Referer == "" {
			p.Referer = p.Pagina
		}
		e, err := a.estimarEmbed(ctx, p.Embed, p.Referer)
		return "embed", e, err
	}
	return "", estimativa{}, nil
}

// POST /api/cursos/estimar {manifesto | embed, referer} → a qualidade máxima, a duração e o tamanho previsto.
func (a *app) handleEstimarAula(w http.ResponseWriter, r *http.Request) {
	var p pedidoAula
	if err := lerCorpo(r, &p); err != nil {
		a.responderErro(w, err)
		return
	}
	tipo, e, err := a.videoDoPedido(r.Context(), &p)
	if err != nil {
		var entrada erroDeEntrada
		if errors.As(err, &entrada) {
			a.responderErro(w, err)
		} else {
			erroJSON(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	if tipo == "" {
		a.responderErro(w, erroDeEntrada{errors.New("mande o manifesto (HLS) ou o vídeo embutido")})
		return
	}
	resp := map[string]any{"qualidade": e.Qualidade, "duracao": e.Duracao, "tamanho_estimado": e.Tamanho, "cifrado": e.Cifrado, "tipo": tipo}
	if a.cursosDir != "" {
		resp["livre"] = espacoLivre(a.cursosDir)
	}
	// A aula já pedida (o mesmo vídeo) volta junto, para a extensão dizer.
	var id int64
	var estado string
	if a.db.QueryRowContext(r.Context(), `SELECT id, estado FROM aulas_plataforma WHERE video = ?`, chaveDoPedido(tipo, p, e)).Scan(&id, &estado) == nil {
		resp["aula"] = map[string]any{"id": id, "estado": estado}
	}
	responderJSON(w, http.StatusOK, resp)
}

// chaveDoPedido: o que faz a aula única — o vídeo (HLS sem a query, youtube:<id>, vimeo:<id>) ou,
// na aula sem vídeo, a página.
func chaveDoPedido(tipo string, p pedidoAula, e estimativa) string {
	switch tipo {
	case "hls":
		return chaveDoVideo(e.Variante)
	case "embed":
		_, chave, _ := normalizarEmbed(p.Embed)
		return chave
	}
	return "pagina:" + strings.TrimRight(strings.SplitN(p.Pagina, "#", 2)[0], "/")
}

// POST /api/cursos/aulas: põe a aula na fila de download (202). O mesmo vídeo de novo: a que
// já está baixada ou na fila volta como está (200, "repetida"), com o texto, a ordem e o módulo
// atualizados; a que deu erro volta para a fila. Sem vídeo e com texto (T55), a aula fica "texto".
// Com "trilha": true, o curso vira trilha (ou ganha a aula nova, cursos_trilha.go).
func (a *app) handlePedirAula(w http.ResponseWriter, r *http.Request) {
	var p pedidoAula
	if err := lerCorpo(r, &p); err != nil {
		a.responderErro(w, err)
		return
	}
	p.Curso, p.Modulo, p.Titulo, p.Texto = strings.TrimSpace(p.Curso), strings.TrimSpace(p.Modulo), strings.TrimSpace(p.Titulo), strings.TrimSpace(p.Texto)
	if plataformasCurso[p.Plataforma] == "" {
		a.responderErro(w, erroDeEntrada{errors.New("plataforma desconhecida")})
		return
	}
	if p.Curso == "" || p.Titulo == "" || !strings.HasPrefix(p.Pagina, "https://") {
		a.responderErro(w, erroDeEntrada{errors.New("mande o curso, o nome da aula e o endereço da página")})
		return
	}
	if len([]rune(p.Texto)) > limiteTextoAula {
		p.Texto = string([]rune(p.Texto)[:limiteTextoAula])
	}
	ctx := r.Context()
	tipo, e, err := a.videoDoPedido(ctx, &p)
	if err != nil {
		var entrada erroDeEntrada
		if errors.As(err, &entrada) {
			a.responderErro(w, err)
		} else {
			erroJSON(w, http.StatusBadGateway, err.Error())
		}
		return
	}
	if tipo == "" && p.Texto == "" {
		a.responderErro(w, erroDeEntrada{errors.New("a aula não tem vídeo nem texto")})
		return
	}
	video := chaveDoPedido(tipo, p, e)
	estadoNovo := "na_fila"
	if tipo == "" {
		estadoNovo = "texto"
	}
	var id int64
	var estado string
	t := agora()
	if a.db.QueryRowContext(ctx, `SELECT id, estado FROM aulas_plataforma WHERE video = ?`, video).Scan(&id, &estado) == nil && estado != "erro" {
		// Repetida: o que a página diz agora (texto, ordem, módulo) atualiza; o vídeo fica como está.
		a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET texto = CASE WHEN ? <> '' THEN ? ELSE texto END,
			ordem = CASE WHEN ? > 0 THEN ? ELSE ordem END, modulo = CASE WHEN ? <> '' THEN ? ELSE modulo END, atualizado_em = ? WHERE id = ?`,
			p.Texto, p.Texto, p.Ordem, p.Ordem, p.Modulo, p.Modulo, t, id)
		au, err := a.aulaPorID(ctx, id)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		resp := map[string]any{"aula": au, "repetida": true}
		a.trilhaDoPedido(ctx, p, resp)
		responderJSON(w, http.StatusOK, resp)
		return
	}
	if id > 0 {
		_, err = a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET plataforma = ?, curso = ?, modulo = ?, titulo = ?, pagina = ?,
			manifesto = ?, referer = ?, variante = ?, qualidade = ?, duracao = ?, estimado = ?, progresso = 0, estado = ?,
			erro = '', ordem = ?, texto = ?, embed = ?, atualizado_em = ? WHERE id = ?`,
			p.Plataforma, p.Curso, p.Modulo, p.Titulo, p.Pagina, p.Manifesto, p.Referer, e.Variante, e.Qualidade, e.Duracao, e.Tamanho,
			estadoNovo, p.Ordem, p.Texto, p.Embed, t, id)
	} else {
		var res sql.Result
		res, err = a.db.ExecContext(ctx, `INSERT INTO aulas_plataforma (plataforma, curso, modulo, titulo, pagina, video, manifesto,
			referer, variante, qualidade, duracao, estimado, estado, ordem, texto, embed, criado_em, atualizado_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Plataforma, p.Curso, p.Modulo, p.Titulo, p.Pagina, video, p.Manifesto, p.Referer, e.Variante, e.Qualidade, e.Duracao, e.Tamanho,
			estadoNovo, p.Ordem, p.Texto, p.Embed, t, t)
		if err == nil {
			id, err = res.LastInsertId()
		}
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if estadoNovo == "na_fila" {
		a.acordarCursos()
	}
	au, err := a.aulaPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	resp := map[string]any{"aula": au}
	a.trilhaDoPedido(ctx, p, resp)
	responderJSON(w, http.StatusAccepted, resp)
}

// trilhaDoPedido: com "trilha": true, o curso vira trilha (ou ganha a aula). Falhar aqui não
// desfaz a aula: o erro volta junto, para a extensão mostrar.
func (a *app) trilhaDoPedido(ctx context.Context, p pedidoAula, resp map[string]any) {
	if !p.Trilha {
		return
	}
	id, err := a.sincronizarTrilhaDoCurso(ctx, p.Plataforma, p.Curso)
	if err != nil {
		log.Printf("cursos: trilha de %s/%s: %v", p.Plataforma, p.Curso, err)
		resp["trilha_erro"] = err.Error()
		return
	}
	resp["trilha_id"] = id
}

// GET /api/cursos/aulas/{id}: uma aula, com o texto da página (a etapa pergunta se o vídeo já baixou).
func (a *app) handleAula(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	au, err := a.aulaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if au.Anexos, err = a.anexosDaAula(r.Context(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, au)
}

// GET /api/cursos/aulas: todas as aulas (a tela agrupa por plataforma › curso › módulo) e o espaço livre.
func (a *app) handleListarAulas(w http.ResponseWriter, r *http.Request) {
	// Os anexos antes da lista: o banco tem uma conexão só, e a lista aberta a seguraria.
	anexos, err := a.anexosDasAulas(r.Context())
	if err != nil {
		a.responderErro(w, err)
		return
	}
	linhas, err := a.db.QueryContext(r.Context(), `SELECT `+colunasAula+` FROM aulas_plataforma ORDER BY plataforma, curso, ordem, id`)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer linhas.Close()
	aulas := []aulaCurso{}
	for linhas.Next() {
		au, err := lerAula(linhas)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		au.Texto = "" // a lista leva só o tem_texto
		au.Anexos = anexos[au.ID]
		if au.Anexos == nil {
			au.Anexos = []anexoAula{}
		}
		aulas = append(aulas, au)
	}
	trilhas, err := a.trilhasDosCursos(r.Context())
	if err != nil {
		a.responderErro(w, err)
		return
	}
	resp := map[string]any{"aulas": aulas, "plataformas": plataformasCurso, "trilhas": trilhas}
	if err := a.discoDosCursos(); err != nil {
		resp["aviso_disco"] = err.Error()
	} else {
		resp["livre"] = espacoLivre(a.cursosDir)
	}
	responderJSON(w, http.StatusOK, resp)
}

// POST /api/cursos/aulas/{id}/tentar: a aula com erro volta para a fila.
func (a *app) handleTentarAula(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	res, err := a.db.ExecContext(r.Context(), `UPDATE aulas_plataforma SET estado = 'na_fila', erro = '', progresso = 0, atualizado_em = ?
		WHERE id = ? AND estado = 'erro'`, agora(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("só a aula com erro volta para a fila")})
		return
	}
	a.acordarCursos()
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/cursos/aulas/{id}: apaga a aula e o vídeo (para o download, se estiver baixando).
func (a *app) handleApagarAula(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	au, err := a.aulaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.apagarArquivosDosAnexos(r.Context(), id)
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM aulas_plataforma WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.download.mu.Lock()
	if a.download.id == id && a.download.cancelar != nil {
		a.download.cancelar()
	}
	a.download.mu.Unlock()
	if au.Arquivo != "" && a.cursosDir != "" {
		os.Remove(filepath.Join(a.cursosDir, au.Arquivo))
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/cursos/aulas/{id}/video: o MP4 (com Range, para o player do navegador pular).
func (a *app) handleVideoAula(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	au, err := a.aulaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if au.Estado != "pronta" || au.Arquivo == "" {
		erroJSON(w, http.StatusConflict, "a aula ainda não foi baixada")
		return
	}
	arq, err := os.Open(filepath.Join(a.cursosDir, au.Arquivo))
	if err != nil {
		erroJSON(w, http.StatusNotFound, "o vídeo não está na pasta (o disco está ligado?)")
		return
	}
	defer arq.Close()
	info, err := arq.Stat()
	if err != nil {
		a.responderErro(w, err)
		return
	}
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, r, filepath.Base(au.Arquivo), info.ModTime(), arq)
}
