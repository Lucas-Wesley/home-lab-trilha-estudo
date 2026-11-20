package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Entrada, fase 10 (T44): a aula do Telegram processada — a Trilha pede o download ao
// Streaming (se a aula ainda não estiver no servidor), transcreve o áudio no Whisper do Mac
// (WHISPER_URL, nunca no Claude) e a fonte fica pronta para o mesmo caminho de um vídeo do
// YouTube: fragmentar, destino, encaixe. Nada baixa sozinho: a fonte só existe depois do
// clique em "Processar" (D9 do plano-entrada).

// handleEntradaTelegram: POST /api/entrada/telegram {"refs":["telegram:<link>#<msg>", …],
// "fragmentar":true} cria uma fonte `telegram` por aula. A mesma ref devolve a fonte que
// já existe; "fragmentar" marca a fonte para entrar na fila de divisão quando ficar pronta.
func (a *app) handleEntradaTelegram(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Refs       []string `json:"refs"`
		Fragmentar bool     `json:"fragmentar"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(corpo.Refs) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("mande as aulas para processar (refs)")})
		return
	}
	ctx := r.Context()
	var dados *meusVideosStreaming
	fontes := make([]fonte, 0, len(corpo.Refs))
	for _, bruta := range corpo.Refs {
		ref := strings.TrimSpace(bruta)
		link, msg, err := referenciaTelegram(ref)
		if err != nil {
			a.responderErro(w, erroDeEntrada{err})
			return
		}
		if f, err := lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE tipo = 'telegram' AND video_id = ?`, ref)); err == nil {
			// A fonte já existe: se ainda está sendo processada, a opção nova vale.
			if corpo.Fragmentar && (f.Estado == "na_fila" || f.Estado == "lendo" || f.Estado == "baixando" || f.Estado == "transcrevendo") {
				a.db.ExecContext(ctx, `UPDATE fontes SET fragmentar = 1 WHERE id = ?`, f.ID)
			}
			fontes = append(fontes, f)
			continue
		}
		if dados == nil {
			var st int
			dados, st, err = a.meusVideos(ctx)
			if err != nil {
				erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
				return
			}
		}
		achada := dados.acharAulaSalva(link, msg)
		if !achada.achou {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("aula não encontrada no Streaming: %s", ref)})
			return
		}
		v := dados.aulaDaSalva(achada)
		f, err := a.criarFonte(ctx, "telegram", ref, ref, v.Nome, "")
		if err != nil {
			a.responderErro(w, err)
			return
		}
		f.Canal = canalDaAula(achada.grupo, v.Caminho)
		f.Duracao = int(v.Duracao)
		f.Fragmentar = corpo.Fragmentar
		if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET canal = ?, duracao = ?, fragmentar = ? WHERE id = ?`,
			f.Canal, f.Duracao, corpo.Fragmentar, f.ID); err != nil {
			a.responderErro(w, err)
			return
		}
		fontes = append(fontes, f)
	}
	a.acordarEntrada()
	responderJSON(w, http.StatusAccepted, fontes)
}

// canalDaAula: o "canal" de uma fonte telegram — grupo › curso › módulo.
func canalDaAula(grupo string, caminho []string) string {
	partes := append([]string{grupo}, caminho...)
	return strings.Join(partes, " › ")
}

// ---------- a fila: lerTelegram é o caso `telegram` do lerProximaFonte ----------

// aulaTelegram: o que a fila precisa da aula — onde ela está e quem ela é.
type aulaTelegram struct {
	Estado  string // baixada · na_fila · no_telegram
	Nome    string
	Canal   string
	Tamanho int64
	Duracao int
	Video   int64 // id do vídeo baixado no Streaming
	Grupo   string
	Link    string
	Msg     int
}

// estadoDaAulaTelegram: o estado de uma aula pela referência estável (telegram:<link>#<msg>).
// Usado pelo handler de Meus arquivos e pela fila da Entrada.
func (a *app) estadoDaAulaTelegram(ctx context.Context, ref string) (*aulaTelegram, int, error) {
	link, msg, err := referenciaTelegram(ref)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	dados, st, err := a.meusVideos(ctx)
	if err != nil {
		return nil, st, err
	}
	achada := dados.acharAulaSalva(link, msg)
	if !achada.achou {
		return nil, http.StatusNotFound, errors.New("aula não encontrada no Streaming")
	}
	v := dados.aulaDaSalva(achada)
	au := &aulaTelegram{
		Nome: v.Nome, Canal: canalDaAula(achada.grupo, v.Caminho), Tamanho: v.Tamanho,
		Duracao: int(v.Duracao), Video: v.Video, Grupo: achada.grupo, Link: link, Msg: msg,
	}
	if v.Video != 0 {
		au.Estado = "baixada"
		return au, 0, nil
	}
	for _, p := range dados.Pedidos {
		if p.Tudo && p.Link == link && (p.Estado == "na fila" || p.Estado == "baixando") &&
			(len(p.Mensagens) == 0 || contemInt(p.Mensagens, msg)) {
			au.Estado = "na_fila"
			return au, 0, nil
		}
	}
	au.Estado = "no_telegram"
	return au, 0, nil
}

// baixarAulaStreaming: o pedido de download de uma aula só — o "Baixar esta aula" da etapa e
// o que a fila manda quando processa uma aula que ainda está só no Telegram (D9: só a pedido).
func (a *app) baixarAulaStreaming(ctx context.Context, link, grupo string, msg int) (int, error) {
	pedido := map[string]any{"link": link, "tudo": true, "pasta": grupo, "mensagens": []int{msg}}
	saida := map[string]any{}
	return chamarApp(ctx, a.integra.streaming, "Streaming", http.MethodPost, "/api/meus-videos/telegram", pedido, &saida)
}

// lerTelegram: a tarefa da fila para uma fonte `telegram`. Devolve o estado final da fonte
// ("baixando", "pronta") ou o erro. Se a aula ainda não está no servidor, pede o download e
// devolve "baixando" — a fila passa para a próxima fonte e a sonda a traz de volta depois,
// sem ficar parada esperando um download de 1 GB.
func (a *app) lerTelegram(ctx context.Context, f fonte) (string, error) {
	au, _, err := a.estadoDaAulaTelegram(ctx, f.VideoID)
	if err != nil {
		return "", err
	}
	// O que o catálogo sabe da aula fica na fonte (o título pode ter vindo vazio).
	a.db.ExecContext(ctx, `UPDATE fontes SET titulo = ?, canal = ?, duracao = ? WHERE id = ?`, au.Nome, au.Canal, au.Duracao, f.ID)
	if au.Estado != "baixada" {
		if au.Estado == "no_telegram" {
			if _, err := a.baixarAulaStreaming(ctx, au.Link, au.Grupo, au.Msg); err != nil {
				return "", err
			}
		}
		return "baixando", nil
	}
	if a.whisper == "" {
		return "", errors.New("o Whisper não está configurado (WHISPER_URL)")
	}
	if a.integra.streaming == "" {
		return "", errors.New("o Streaming não está configurado (STREAMING_URL)")
	}
	a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'transcrevendo' WHERE id = ?`, f.ID)
	// O ffmpeg lê o vídeo direto da url interna do Streaming (STREAMING_URL, não o link
	// do navegador) — o arquivo nem precisa baixar para a Trilha.
	videoURL := fmt.Sprintf("%s/api/video/proprio/%d.mp4", a.integra.streaming, au.Video)
	info := infoVideo{Titulo: au.Nome, Canal: au.Canal, URL: f.VideoID, Duracao: float64(au.Duracao), Idioma: "pt"}
	transcricao, err := a.transcreverAulaTelegram(ctx, videoURL, info)
	if err != nil {
		return "", err
	}
	if err := a.conteudo.gravar(fmt.Sprintf("entrada %d: transcrição lida", f.ID),
		map[string]*string{pastaDaFonte(f.ID) + "/transcricao.md": &transcricao}); err != nil {
		return "", err
	}
	return "pronta", nil
}

// promoverBaixando: a sonda da fila — a aula que estava "baixando" volta para a fila e é
// verificada de novo (baixada → transcreve; ainda não → volta a "baixando").
func (a *app) promoverBaixando(ctx context.Context) bool {
	res, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'na_fila' WHERE estado = 'baixando'`)
	if err != nil {
		log.Printf("entrada: %v", err)
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// ---------- áudio → Whisper ----------

// segundosPorPedaco: o Whisper do Mac recebe o áudio em pedaços de 10 min — como o Acervo
// corta podcasts (acervo/backend/podcast.go). Variável para o teste usar pedaços curtos.
var segundosPorPedaco = 600

// transcreverAulaTelegram corta o áudio do vídeo em pedaços, manda um a um ao Whisper e monta
// a transcrição com [mm:ss] somando o começo de cada pedaço. O áudio é apagado ao sair.
func (a *app) transcreverAulaTelegram(ctx context.Context, videoURL string, info infoVideo) (string, error) {
	tmp, err := os.MkdirTemp("", "telegram-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	pedacos, err := a.cortarAudioTelegram(ctx, videoURL, tmp)
	if err != nil {
		return "", err
	}
	var todos []trechoFalado
	for i, p := range pedacos {
		trechos, err := a.pedirTranscricao(ctx, p, info)
		if err != nil {
			return "", fmt.Errorf("a transcrição do pedaço %d de %d falhou: %w", i+1, len(pedacos), err)
		}
		for _, t := range trechos {
			t.Inicio += float64(i * segundosPorPedaco)
			todos = append(todos, t)
		}
	}
	if len(todos) == 0 {
		return "", errSemFala
	}
	return cabecalhoDaTranscricao(info, "Transcrição: Whisper (aula do Telegram)") + falasEmBlocos(todos), nil
}

// cortarAudioTelegram: ffmpeg em mono 16 kHz FLAC (o formato que o Whisper usa, ~8 MB por
// pedaço), lendo o vídeo direto da url do Streaming.
func (a *app) cortarAudioTelegram(ctx context.Context, videoURL, pasta string) ([]string, error) {
	cmd := exec.CommandContext(ctx, a.ffmpegBin, "-v", "error", "-i", videoURL, "-vn", "-ac", "1", "-ar", "16000",
		"-f", "segment", "-segment_time", strconv.Itoa(segundosPorPedaco), "-c:a", "flac",
		filepath.Join(pasta, "pedaco-%03d.flac"))
	var erros bytes.Buffer
	cmd.Stderr = &erros
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg falhou: %s", primeiraLinha(strings.TrimSpace(erros.String()), err.Error()))
	}
	pedacos, _ := filepath.Glob(filepath.Join(pasta, "pedaco-*.flac"))
	sort.Strings(pedacos)
	if len(pedacos) == 0 {
		return nil, errors.New("o ffmpeg não gerou áudio do vídeo")
	}
	return pedacos, nil
}
