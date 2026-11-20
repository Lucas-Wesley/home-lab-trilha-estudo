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

// Cursos de plataformas (T50): as aulas da Branas.io e da LINUXtips, baixadas em vídeo, na
// qualidade máxima, para a aba "Cursos" da Entrada. Os donos dos dois cursos autorizaram o Lucas.
// A extensão do Chrome (E15 de lá) acha o manifesto HLS no player e manda com o nome da aula;
// aqui o ffmpeg baixa (decifra o AES-128 da LINUXtips com a chave que o próprio servidor entrega
// a quem manda o Referer do player) e grava em CURSOS_DIR/<plataforma>/<curso>/<módulo>/<aula>.mp4.
// Um download por vez, numa fila própria (não gasta crédito do Claude).

var plataformasCurso = map[string]string{
	"branas":    "Branas.io",
	"linuxtips": "LINUXtips",
}

const agenteNavegador = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/157.0.0.0 Safari/537.36"

type aulaCurso struct {
	ID           int64  `json:"id"`
	Plataforma   string `json:"plataforma"`
	Curso        string `json:"curso"`
	Modulo       string `json:"modulo"`
	Titulo       string `json:"titulo"`
	Pagina       string `json:"pagina"`
	Manifesto    string `json:"-"`
	Referer      string `json:"-"`
	Variante     string `json:"-"`
	Qualidade    string `json:"qualidade"`
	Duracao      int    `json:"duracao"`
	Estimado     int64  `json:"tamanho_estimado"`
	Tamanho      int64  `json:"tamanho"`
	Progresso    int    `json:"progresso"`
	Estado       string `json:"estado"` // na_fila · baixando · pronta · erro
	Erro         string `json:"erro"`
	Arquivo      string `json:"-"` // relativo a CURSOS_DIR
	CriadoEm     string `json:"criado_em"`
	AtualizadoEm string `json:"atualizado_em"`
}

const esquemaCursos = `
CREATE TABLE IF NOT EXISTS aulas_curso (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	plataforma    TEXT NOT NULL,
	curso         TEXT NOT NULL,
	modulo        TEXT NOT NULL DEFAULT '',
	titulo        TEXT NOT NULL,
	pagina        TEXT NOT NULL UNIQUE,  -- o endereço da aula na plataforma
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
	atualizado_em TEXT NOT NULL
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
	duracao, estimado, tamanho, progresso, estado, erro, arquivo, criado_em, atualizado_em`

func lerAula(s interface{ Scan(...any) error }) (aulaCurso, error) {
	var au aulaCurso
	err := s.Scan(&au.ID, &au.Plataforma, &au.Curso, &au.Modulo, &au.Titulo, &au.Pagina, &au.Manifesto, &au.Referer,
		&au.Variante, &au.Qualidade, &au.Duracao, &au.Estimado, &au.Tamanho, &au.Progresso, &au.Estado, &au.Erro,
		&au.Arquivo, &au.CriadoEm, &au.AtualizadoEm)
	return au, err
}

func (a *app) aulaPorID(ctx context.Context, id int64) (aulaCurso, error) {
	au, err := lerAula(a.db.QueryRowContext(ctx, `SELECT `+colunasAula+` FROM aulas_curso WHERE id = ?`, id))
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
	_, err := a.db.ExecContext(ctx, `UPDATE aulas_curso SET estado = 'na_fila', progresso = 0 WHERE estado = 'baixando'`)
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

func (a *app) baixarProxima(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	au, err := lerAula(a.db.QueryRowContext(ctx, `SELECT `+colunasAula+` FROM aulas_curso WHERE estado = 'na_fila' ORDER BY id LIMIT 1`))
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
	a.db.ExecContext(ctx, `UPDATE aulas_curso SET estado = 'baixando', progresso = 0, erro = '', atualizado_em = ? WHERE id = ?`, agora(), au.ID)
	tamanho, arquivo, err := a.baixarAula(cl, au)
	if errors.Is(cl.Err(), context.Canceled) && ctx.Err() == nil {
		return true // apagada no meio: a linha já não existe
	}
	if err != nil {
		log.Printf("cursos: aula %d: %v", au.ID, err)
		a.db.ExecContext(ctx, `UPDATE aulas_curso SET estado = 'erro', erro = ?, atualizado_em = ? WHERE id = ?`, err.Error(), agora(), au.ID)
		return true
	}
	a.db.ExecContext(ctx, `UPDATE aulas_curso SET estado = 'pronta', progresso = 100, tamanho = ?, arquivo = ?, erro = '', atualizado_em = ? WHERE id = ?`,
		tamanho, arquivo, agora(), au.ID)
	return true
}

func (a *app) baixarAula(ctx context.Context, au aulaCurso) (int64, string, error) {
	if a.cursosDir == "" {
		return 0, "", errors.New("CURSOS_DIR não configurado")
	}
	variante := au.Variante
	if variante == "" {
		e, err := a.estimar(ctx, au.Manifesto, au.Referer)
		if err != nil {
			return 0, "", err
		}
		variante = e.Variante
	}
	rel := caminhoDaAula(au)
	final := filepath.Join(a.cursosDir, rel)
	// Outra aula com o mesmo nome no mesmo lugar: o id desempata.
	if _, err := os.Stat(final); err == nil {
		var outra int64
		if a.db.QueryRowContext(ctx, `SELECT id FROM aulas_curso WHERE arquivo = ? AND id <> ?`, rel, au.ID).Scan(&outra) == nil {
			rel = strings.TrimSuffix(rel, ".mp4") + fmt.Sprintf(" (%d).mp4", au.ID)
			final = filepath.Join(a.cursosDir, rel)
		}
	}
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		return 0, "", fmt.Errorf("não deu para criar a pasta: %w", err)
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
		a.db.ExecContext(ctx, `UPDATE aulas_curso SET progresso = ? WHERE id = ?`, p, au.ID)
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
}

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

// POST /api/cursos/estimar {manifesto, referer} → a qualidade máxima, a duração e o tamanho previsto.
func (a *app) handleEstimarAula(w http.ResponseWriter, r *http.Request) {
	var p pedidoAula
	if err := lerCorpo(r, &p); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := validarVideo(p.Manifesto, p.Referer); err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.estimar(r.Context(), p.Manifesto, p.Referer)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	resp := map[string]any{"qualidade": e.Qualidade, "duracao": e.Duracao, "tamanho_estimado": e.Tamanho, "cifrado": e.Cifrado}
	if a.cursosDir != "" {
		resp["livre"] = espacoLivre(a.cursosDir)
	}
	// A aula já pedida (pelo endereço da página) volta junto, para a extensão dizer.
	var id int64
	var estado string
	if a.db.QueryRowContext(r.Context(), `SELECT id, estado FROM aulas_curso WHERE pagina = ?`, p.Pagina).Scan(&id, &estado) == nil {
		resp["aula"] = map[string]any{"id": id, "estado": estado}
	}
	responderJSON(w, http.StatusOK, resp)
}

// POST /api/cursos/aulas: põe a aula na fila de download (202). A mesma página de novo: a que
// já está baixada ou na fila volta como está (200, "repetida"); a que deu erro volta para a fila.
func (a *app) handlePedirAula(w http.ResponseWriter, r *http.Request) {
	var p pedidoAula
	if err := lerCorpo(r, &p); err != nil {
		a.responderErro(w, err)
		return
	}
	p.Curso, p.Modulo, p.Titulo = strings.TrimSpace(p.Curso), strings.TrimSpace(p.Modulo), strings.TrimSpace(p.Titulo)
	if plataformasCurso[p.Plataforma] == "" {
		a.responderErro(w, erroDeEntrada{errors.New("plataforma desconhecida")})
		return
	}
	if p.Curso == "" || p.Titulo == "" || !strings.HasPrefix(p.Pagina, "https://") {
		a.responderErro(w, erroDeEntrada{errors.New("mande o curso, o nome da aula e o endereço da página")})
		return
	}
	if err := validarVideo(p.Manifesto, p.Referer); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	var id int64
	var estado string
	if a.db.QueryRowContext(ctx, `SELECT id, estado FROM aulas_curso WHERE pagina = ?`, p.Pagina).Scan(&id, &estado) == nil && estado != "erro" {
		au, err := a.aulaPorID(ctx, id)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		responderJSON(w, http.StatusOK, map[string]any{"aula": au, "repetida": true})
		return
	}
	e, err := a.estimar(ctx, p.Manifesto, p.Referer)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	t := agora()
	if id > 0 {
		_, err = a.db.ExecContext(ctx, `UPDATE aulas_curso SET plataforma = ?, curso = ?, modulo = ?, titulo = ?, manifesto = ?,
			referer = ?, variante = ?, qualidade = ?, duracao = ?, estimado = ?, progresso = 0, estado = 'na_fila', erro = '',
			atualizado_em = ? WHERE id = ?`,
			p.Plataforma, p.Curso, p.Modulo, p.Titulo, p.Manifesto, p.Referer, e.Variante, e.Qualidade, e.Duracao, e.Tamanho, t, id)
	} else {
		var res sql.Result
		res, err = a.db.ExecContext(ctx, `INSERT INTO aulas_curso (plataforma, curso, modulo, titulo, pagina, manifesto, referer,
			variante, qualidade, duracao, estimado, criado_em, atualizado_em) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Plataforma, p.Curso, p.Modulo, p.Titulo, p.Pagina, p.Manifesto, p.Referer, e.Variante, e.Qualidade, e.Duracao, e.Tamanho, t, t)
		if err == nil {
			id, err = res.LastInsertId()
		}
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.acordarCursos()
	au, err := a.aulaPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusAccepted, map[string]any{"aula": au})
}

// GET /api/cursos/aulas: todas as aulas (a tela agrupa por plataforma › curso › módulo) e o espaço livre.
func (a *app) handleListarAulas(w http.ResponseWriter, r *http.Request) {
	linhas, err := a.db.QueryContext(r.Context(), `SELECT `+colunasAula+` FROM aulas_curso ORDER BY plataforma, curso, modulo, id`)
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
		aulas = append(aulas, au)
	}
	resp := map[string]any{"aulas": aulas, "plataformas": plataformasCurso}
	if a.cursosDir != "" {
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
	res, err := a.db.ExecContext(r.Context(), `UPDATE aulas_curso SET estado = 'na_fila', erro = '', progresso = 0, atualizado_em = ?
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
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM aulas_curso WHERE id = ?`, id); err != nil {
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
