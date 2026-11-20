package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// Entrada, fase 7 (T41): "Meus arquivos" — os cursos do Telegram que estão no Streaming
// (curso › módulo › aula, S55 de lá), vistos e baixados daqui. O Streaming continua o dono do
// Telegram: a Trilha só repassa a API dele; nada baixa sozinho (D8, D9 do plano-entrada).

// handleArquivos: GET /api/arquivos devolve o /api/meus-videos do Streaming como veio, mais o
// link dele (para "Assistir" abrir no Streaming).
func (a *app) handleArquivos(w http.ResponseWriter, r *http.Request) {
	dados := map[string]any{}
	if st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodGet, "/api/meus-videos", nil, &dados); err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	dados["link_streaming"] = a.integra.streamingLink
	responderJSON(w, http.StatusOK, dados)
}

// handleBaixarArquivos: {"link","grupo","mensagens"} → POST /api/meus-videos/telegram com
// {link, tudo:true, pasta:grupo, mensagens}: baixa só as aulas escolhidas, cada uma na pasta
// do curso e do módulo dela.
func (a *app) handleBaixarArquivos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Link      string `json:"link"`
		Grupo     string `json:"grupo"`
		Mensagens []int  `json:"mensagens"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if !strings.HasPrefix(strings.TrimSpace(corpo.Link), "https://t.me/") || len(corpo.Mensagens) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("mande o link do Telegram e as aulas para baixar")})
		return
	}
	pedido := map[string]any{"link": corpo.Link, "tudo": true, "pasta": corpo.Grupo, "mensagens": corpo.Mensagens}
	saida := map[string]any{}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodPost, "/api/meus-videos/telegram", pedido, &saida)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, saida)
}

// handleApagarPastaArquivos: {"pasta"} → POST /api/meus-videos/apagar-pasta: a lixeira do curso
// ou do módulo (apaga as baixadas da pasta e das subpastas).
func (a *app) handleApagarPastaArquivos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pasta string `json:"pasta"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if strings.TrimSpace(corpo.Pasta) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("mande a pasta")})
		return
	}
	saida := map[string]any{}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodPost, "/api/meus-videos/apagar-pasta", corpo, &saida)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, saida)
}

// handleApagarVideoArquivos: DELETE /api/arquivos/videos/{id} → DELETE /api/meus-videos/{id}:
// apaga um vídeo baixado do servidor.
func (a *app) handleApagarVideoArquivos(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	st, err := chamarApp(r.Context(), a.integra.streaming, "Streaming", http.MethodDelete, "/api/meus-videos/"+strconv.FormatInt(id, 10), nil, nil)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- fase 8 (T42): uma aula do Telegram pela referência estável ----------

// videoSalvo: uma aula do catálogo do Telegram (`caminho` = curso › módulo dentro do grupo).
type videoSalvo struct {
	Mensagem int      `json:"mensagem"`
	Nome     string   `json:"nome"`
	Caminho  []string `json:"caminho"`
	Tamanho  int64    `json:"tamanho"`
	Duracao  segundos `json:"duracao"`
	Video    int64    `json:"video"` // id do baixado; 0 = só no Telegram
}

// meusVideosStreaming: só os campos do /api/meus-videos do Streaming que a Trilha usa.
type meusVideosStreaming struct {
	Videos []struct {
		ID      int64    `json:"id"`
		Pasta   string   `json:"pasta"`
		Nome    string   `json:"nome"`
		Tamanho int64    `json:"tamanho"`
		Duracao segundos `json:"duracao"`
	} `json:"videos"`
	Salvos []struct {
		Tudo   bool         `json:"tudo"`
		Link   string       `json:"link"`
		Pasta  string       `json:"pasta"`
		Titulo string       `json:"titulo"`
		Videos []videoSalvo `json:"videos"`
	} `json:"salvos"`
	Pedidos []struct {
		Tudo      bool   `json:"tudo"`
		Link      string `json:"link"`
		Estado    string `json:"estado"`
		Mensagens []int  `json:"mensagens"`
	} `json:"pedidos"`
}

type aulaSalva struct {
	grupo    string // a pasta raiz do grupo (para onde o download vai)
	indiceS  int
	indiceV  int
	achou    bool
	mensagem int
	link     string
}

// acharAulaSalva: a mensagem <msg> do grupo <link> no catálogo dos salvos do Streaming.
func (d *meusVideosStreaming) acharAulaSalva(link string, msg int) aulaSalva {
	for i := range d.Salvos {
		s := &d.Salvos[i]
		if !s.Tudo || s.Link != link {
			continue
		}
		for j, v := range s.Videos {
			if v.Mensagem == msg {
				grupo := s.Pasta
				if grupo == "" {
					grupo = s.Titulo
				}
				return aulaSalva{grupo: grupo, indiceS: i, indiceV: j, achou: true, mensagem: msg, link: link}
			}
		}
	}
	return aulaSalva{}
}

func (d *meusVideosStreaming) aulaDaSalva(a aulaSalva) *videoSalvo {
	return &d.Salvos[a.indiceS].Videos[a.indiceV]
}

// referenciaTelegram: "telegram:<link do grupo>#<mensagem>" → as duas partes (a referência não
// muda quando a aula é baixada, apagada ou baixada de novo — o id no Streaming muda; a mensagem não).
func referenciaTelegram(ref string) (link string, msg int, err error) {
	resto, ok := strings.CutPrefix(strings.TrimSpace(ref), "telegram:")
	if !ok {
		return "", 0, errors.New("a referência precisa começar com telegram:")
	}
	i := strings.LastIndex(resto, "#")
	if i < 0 {
		return "", 0, errors.New(`a referência do telegram precisa de "#<mensagem>"`)
	}
	msg, err = strconv.Atoi(resto[i+1:])
	if err != nil || msg <= 0 || !strings.HasPrefix(resto[:i], "https://t.me/") {
		return "", 0, fmt.Errorf("referência do telegram inválida: %q", ref)
	}
	return resto[:i], msg, nil
}

// handleAulaArquivos: GET /api/arquivos/aula?ref=telegram:<link>#<msg> diz o estado da aula:
// baixada (com a url do vídeo), na_fila ou no_telegram. A etapa pergunta antes de tocar.
func (a *app) handleAulaArquivos(w http.ResponseWriter, r *http.Request) {
	au, st, err := a.estadoDaAulaTelegram(r.Context(), r.URL.Query().Get("ref"))
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	saida := map[string]any{"estado": au.Estado, "nome": au.Nome, "tamanho": au.Tamanho, "duracao": au.Duracao}
	if au.Estado == "baixada" {
		saida["video"] = au.Video
		saida["url"] = fmt.Sprintf("%s/api/video/proprio/%d.mp4", a.integra.streamingLink, au.Video)
	}
	responderJSON(w, http.StatusOK, saida)
}

// handleBaixarAulaArquivos: {"ref":"telegram:…"} baixa só aquela aula (o "Baixar esta aula" da
// etapa — D9: só por clique).
func (a *app) handleBaixarAulaArquivos(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Ref string `json:"ref"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	link, msg, err := referenciaTelegram(corpo.Ref)
	if err != nil {
		a.responderErro(w, erroDeEntrada{err})
		return
	}
	dados, st, err := a.meusVideos(r.Context())
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	achada := dados.acharAulaSalva(link, msg)
	if !achada.achou {
		erroJSON(w, http.StatusNotFound, "aula não encontrada no Streaming")
		return
	}
	st, err = a.baixarAulaStreaming(r.Context(), link, achada.grupo, msg)
	if err != nil {
		erroJSON(w, statusOu(st, http.StatusBadGateway), err.Error())
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{})
}

func (a *app) meusVideos(ctx context.Context) (*meusVideosStreaming, int, error) {
	dados := &meusVideosStreaming{}
	st, err := chamarApp(ctx, a.integra.streaming, "Streaming", http.MethodGet, "/api/meus-videos", nil, dados)
	return dados, st, err
}

func contemInt(lista []int, v int) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}

// segundos: a duração que o Streaming manda vem do ffprobe, com fração ("duracao": 25.067).
// Lê número inteiro ou fracionado e arredonda.
type segundos int

func (s *segundos) UnmarshalJSON(b []byte) error {
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	*s = segundos(math.Round(f))
	return nil
}
