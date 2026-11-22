package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Vídeo embutido (T55): nas plataformas que não usam HLS (a Cademí do Chef Enzo Neto deixa cada escola escolher
// onde hospeda: YouTube não listado, Vimeo…), a extensão manda o endereço do player embutido e o
// yt-dlp baixa a maior qualidade, com o Referer da página (o Vimeo privado só toca no site da escola).

var reIDYouTube = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// normalizarEmbed: o endereço que o yt-dlp baixa e a chave da aula (youtube:<id>, vimeo:<id>).
func normalizarEmbed(endereco string) (baixar, chave string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(endereco))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", false
	}
	host := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), "m.")
	partes := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch host {
	case "youtube.com", "youtube-nocookie.com", "youtu.be":
		id := ""
		switch {
		case host == "youtu.be":
			id = partes[0]
		case u.Path == "/watch":
			id = u.Query().Get("v")
		case len(partes) >= 2 && (partes[0] == "embed" || partes[0] == "shorts" || partes[0] == "live" || partes[0] == "v"):
			id = partes[1]
		}
		if !reIDYouTube.MatchString(id) {
			return "", "", false
		}
		return "https://www.youtube.com/watch?v=" + id, "youtube:" + id, true
	case "player.vimeo.com":
		if len(partes) >= 2 && partes[0] == "video" {
			if _, err := strconv.ParseUint(partes[1], 10, 64); err == nil {
				b := "https://player.vimeo.com/video/" + partes[1]
				if h := u.Query().Get("h"); h != "" {
					b += "?h=" + url.QueryEscape(h)
				}
				return b, "vimeo:" + partes[1], true
			}
		}
	case "vimeo.com":
		if len(partes) >= 1 {
			if _, err := strconv.ParseUint(partes[0], 10, 64); err == nil {
				b := "https://vimeo.com/" + partes[0]
				if len(partes) >= 2 && partes[1] != "" {
					b += "/" + partes[1]
				}
				return b, "vimeo:" + partes[0], true
			}
		}
	}
	return "", "", false
}

// estimarEmbed: o yt-dlp lê o vídeo (sem baixar) e diz a duração, a maior qualidade e o tamanho.
func (a *app) estimarEmbed(ctx context.Context, endereco, referer string) (estimativa, error) {
	bruto, err := a.ytdlp(ctx, "-J", "--no-playlist", "-f", "bv*+ba/b", "--referer", referer, endereco)
	if err != nil {
		return estimativa{}, err
	}
	var info struct {
		Duracao    float64 `json:"duration"`
		Altura     int     `json:"height"`
		Tamanho    int64   `json:"filesize"`
		Aproximado int64   `json:"filesize_approx"`
		Formatos   []struct {
			Tamanho    int64 `json:"filesize"`
			Aproximado int64 `json:"filesize_approx"`
			Altura     int   `json:"height"`
		} `json:"requested_formats"`
	}
	if err := json.Unmarshal(bruto, &info); err != nil {
		return estimativa{}, errors.New("o yt-dlp não devolveu as informações do vídeo")
	}
	e := estimativa{Variante: endereco, Duracao: int(info.Duracao + 0.5)}
	altura := info.Altura
	for _, f := range info.Formatos {
		if f.Tamanho > 0 {
			e.Tamanho += f.Tamanho
		} else {
			e.Tamanho += f.Aproximado
		}
		if f.Altura > altura {
			altura = f.Altura
		}
	}
	if e.Tamanho == 0 {
		e.Tamanho = info.Tamanho
	}
	if e.Tamanho == 0 {
		e.Tamanho = info.Aproximado
	}
	if altura > 0 {
		e.Qualidade = fmt.Sprintf("%dp", altura)
	}
	return e, nil
}

var reProgressoYtdlp = regexp.MustCompile(`^\[download\]\s+([\d.]+)%`)

// baixarEmbed: o yt-dlp baixa o melhor vídeo e o melhor áudio e junta num MP4 (o ffmpeg do container).
func (a *app) baixarEmbed(ctx context.Context, au aulaCurso) (int64, string, error) {
	rel, final, err := a.destinoDaAula(ctx, au)
	if err != nil {
		return 0, "", err
	}
	base := final + ".baixando"
	defer func() {
		sobras, _ := filepath.Glob(globSeguro(base) + "*")
		for _, s := range sobras {
			os.Remove(s)
		}
	}()
	args := append(append([]string{}, a.ytdlpExtra...), "--no-playlist", "--newline", "--no-part",
		"-f", "bv*+ba/b", "--merge-output-format", "mp4", "--remux-video", "mp4",
		"--referer", au.Referer, "-o", base+".%(ext)s", au.Embed)
	cmd := exec.CommandContext(ctx, a.ytdlpBin, args...)
	saida, err := cmd.StdoutPipe()
	if err != nil {
		return 0, "", err
	}
	var erros strings.Builder
	cmd.Stderr = &erros
	if err := cmd.Start(); err != nil {
		return 0, "", fmt.Errorf("yt-dlp: %w", err)
	}
	// O vídeo e o áudio vêm separados (dois 0–100%): a primeira passada vale até 90%.
	fase, anterior := 0, 0.0
	sc := bufio.NewScanner(saida)
	for sc.Scan() {
		m := reProgressoYtdlp.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		pct, _ := strconv.ParseFloat(m[1], 64)
		if pct+20 < anterior {
			fase++
		}
		anterior = pct
		p := int(pct * 0.9)
		if fase > 0 {
			p = 90 + int(pct/10)
		}
		if p > 99 {
			p = 99
		}
		a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET progresso = ? WHERE id = ?`, p, au.ID)
	}
	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(erros.String())
		if i := strings.LastIndex(msg, "ERROR:"); i >= 0 {
			msg = msg[i:]
		}
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		return 0, "", fmt.Errorf("o yt-dlp falhou: %v %s", err, msg)
	}
	baixado := base + ".mp4"
	if _, err := os.Stat(baixado); err != nil {
		achados, _ := filepath.Glob(globSeguro(base) + ".*")
		if len(achados) == 0 {
			return 0, "", errors.New("o yt-dlp terminou sem gravar o vídeo")
		}
		baixado = achados[0]
	}
	info, err := os.Stat(baixado)
	if err != nil || info.Size() == 0 {
		return 0, "", errors.New("o yt-dlp terminou sem gravar o vídeo")
	}
	if err := os.Rename(baixado, final); err != nil {
		return 0, "", err
	}
	return info.Size(), rel, nil
}

// globSeguro: o caminho sem os caracteres que o Glob leria como padrão ([ ] * ?).
func globSeguro(caminho string) string {
	return regexp.MustCompile(`[\[\]*?\\]`).ReplaceAllString(caminho, `\$0`)
}
