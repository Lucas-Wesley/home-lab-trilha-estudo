package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Entrada, fase 10 (T44): processar aulas do Telegram. Nos testes, o Streaming é um servidor
// falso (httptest), o Whisper outro e o ffmpeg o script de testdata/ — nada de Telegram, de
// Whisper ou de ffmpeg de verdade.

// streamingDeAulas: um Streaming falso com duas aulas do mesmo grupo. `baixadas` (msg → id
// do vídeo) diz quais já estão no servidor; o POST /telegram marca a mensagem como baixada.
func streamingDeAulas(t *testing.T, pedidos *[]pedidoStreaming, baixadas map[int]int64) *httptest.Server {
	t.Helper()
	salva := func(msg int, nome, caminhoJSON string, duracao int) string {
		video := ""
		if id, ok := baixadas[msg]; ok {
			video = fmt.Sprintf(`,"video":%d`, id)
		}
		return fmt.Sprintf(`{"mensagem":%d,"nome":%q,"caminho":[%s],"tamanho":100,"duracao":%d%s}`,
			msg, nome, caminhoJSON, duracao, video)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*pedidos = append(*pedidos, pedidoStreaming{metodo: r.Method + " " + r.URL.Path})
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/meus-videos":
			fmt.Fprintf(w, `{"livre":5000000000,"telegram":{"instalado":true},"salvos":[{"tudo":true,
				"link":"https://t.me/c/1","pasta":"Grupo","videos":[%s,%s]}],"pedidos":[]}`,
				salva(2, "Aula 1", `"Curso A","Módulo 1"`, 60),
				salva(3, "Aula 2", `"Curso B"`, 120))
		case r.Method == "POST" && r.URL.Path == "/api/meus-videos/telegram":
			var c struct {
				Mensagens []int `json:"mensagens"`
			}
			var corpo map[string]any
			json.NewDecoder(r.Body).Decode(&corpo)
			(*pedidos)[len(*pedidos)-1].corpo = corpo
			msgs, _ := corpo["mensagens"].([]any)
			for _, m := range msgs {
				c.Mensagens = append(c.Mensagens, int(m.(float64)))
			}
			for _, m := range c.Mensagens {
				baixadas[m] = int64(90 + m) // o download "terminou": a próxima leitura a acha baixada
			}
			w.Write([]byte(`{"id":9,"estado":"na fila"}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"erro":"não achei"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// whisperDeTeste: devolve dois segmentos por pedaço; com dois pedaços, o teste confere os
// minutos somados (o segundo começa em 10:00).
func whisperDeTeste(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions" || r.ParseMultipartForm(4<<20) != nil {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"language":"pt","segments":[{"start":0.0,"text":"Fala um."},{"start":12.0,"text":"Fala dois."}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func clienteTelegram(t *testing.T, baixadas map[int]int64, pedidos *[]pedidoStreaming) *cliente {
	t.Helper()
	c := novoCliente(t)
	c.a.integra.streaming = streamingDeAulas(t, pedidos, baixadas).URL
	c.a.integra.streamingLink = "https://streaming.internal"
	c.a.ffmpegBin = filepath.Join("testdata", "ffmpeg-falso")
	c.a.whisper = whisperDeTeste(t).URL
	return c
}

func fonteDoTeste(c *cliente, id int64) fonte {
	var f fonte
	c.ok("GET", fmt.Sprintf("/api/entrada/%d", id), nil, &f)
	return f
}

// O caminho inteiro: aula baixada → fonte pronta com a transcrição e os minutos somados;
// aula no Telegram → pede o download, fica "baixando" e a fila segue para a próxima fonte.
func TestEntradaTelegram(t *testing.T) {
	pedidos := []pedidoStreaming{}
	c := clienteTelegram(t, map[int]int64{2: 7}, &pedidos)
	argsLog := filepath.Join(t.TempDir(), "ffmpeg-args.txt")
	t.Setenv("FFMPEG_ARGS_LOG", argsLog)
	ctx := context.Background()

	var fontes []fonte
	c.ok("POST", "/api/entrada/telegram", map[string]any{
		"refs": []string{"telegram:https://t.me/c/1#3", "telegram:https://t.me/c/1#2"},
	}, &fontes)
	if len(fontes) != 2 || fontes[0].Tipo != "telegram" || fontes[0].VideoID != "telegram:https://t.me/c/1#3" ||
		fontes[0].Titulo != "Aula 2" || fontes[0].Canal != "Grupo › Curso B" || fontes[0].Estado != "na_fila" {
		t.Fatalf("fontes: %+v", fontes)
	}

	// Uma passada da fila: a fonte 1 (só no Telegram) pede o download e fica "baixando";
	// a fonte 2 (baixada) transcreve e fica pronta — a fila não parou na primeira.
	c.a.esvaziarEntrada(ctx)
	f1, f2 := fonteDoTeste(c, fontes[0].ID), fonteDoTeste(c, fontes[1].ID)
	if f1.Estado != "baixando" {
		t.Fatalf("f1 devia estar baixando: %+v", f1)
	}
	if f2.Estado != "pronta" {
		t.Fatalf("f2 devia estar pronta: %+v", f2)
	}
	var baixar *pedidoStreaming
	for i := range pedidos {
		if pedidos[i].metodo == "POST /api/meus-videos/telegram" {
			baixar = &pedidos[i]
		}
	}
	if baixar == nil || len(baixar.corpo["mensagens"].([]any)) != 1 || baixar.corpo["mensagens"].([]any)[0] != 3.0 {
		t.Fatalf("o download pedido: %+v", baixar)
	}
	// A transcrição da fonte 2 tem os minutos somados entre os pedaços do ffmpeg falso.
	if !strings.Contains(f2.Transcricao, "Transcrição: Whisper") ||
		!strings.Contains(f2.Transcricao, "[00:00] Fala um.") ||
		!strings.Contains(f2.Transcricao, "[10:00] Fala um.") {
		t.Fatalf("transcrição:\n%s", f2.Transcricao)
	}
	// O ffmpeg recebeu a url interna do vídeo (STREAMING_URL), não o link do navegador.
	args, _ := os.ReadFile(argsLog)
	if !strings.Contains(string(args), c.a.integra.streaming+"/api/video/proprio/7.mp4") ||
		strings.Contains(string(args), "streaming.internal") ||
		!strings.Contains(string(args), "-ac 1 -ar 16000") {
		t.Fatalf("args do ffmpeg: %s", args)
	}

	// A sonda traz a fonte de volta: baixada no falso → transcreve e fica pronta.
	if !c.a.promoverBaixando(ctx) {
		t.Fatal("não tinha nada baixando para promover")
	}
	c.a.esvaziarEntrada(ctx)
	f1 = fonteDoTeste(c, fontes[0].ID)
	if f1.Estado != "pronta" || !strings.Contains(f1.Transcricao, "[10:00]") {
		t.Fatalf("depois da sonda: %+v", f1)
	}

	// A mesma ref devolve a fonte que já existe.
	var outra []fonte
	c.ok("POST", "/api/entrada/telegram", map[string]any{"refs": []string{"telegram:https://t.me/c/1#2"}}, &outra)
	if len(outra) != 1 || outra[0].ID != fontes[1].ID {
		t.Fatalf("dedup: %+v", outra)
	}
	// Ref inválida e aula que não existe no catálogo: 400 sem criar fonte.
	if s := c.chamar("POST", "/api/entrada/telegram", map[string]any{"refs": []string{"telegram:https://t.me/c/1#99"}}, nil); s != 400 {
		t.Fatalf("aula fora do catálogo: %d", s)
	}
	if s := c.chamar("POST", "/api/entrada/telegram", map[string]any{"refs": []string{"youtube:xx"}}, nil); s != 400 {
		t.Fatalf("ref inválida: %d", s)
	}
}

// Sem WHISPER_URL: erro "o Whisper não está configurado"; "Tentar de novo" depois de
// configurar transcreve.
func TestEntradaTelegramSemWhisper(t *testing.T) {
	pedidos := []pedidoStreaming{}
	c := clienteTelegram(t, map[int]int64{2: 7}, &pedidos)
	c.a.whisper = ""
	ctx := context.Background()

	var fontes []fonte
	c.ok("POST", "/api/entrada/telegram", map[string]any{"refs": []string{"telegram:https://t.me/c/1#2"}}, &fontes)
	c.a.esvaziarEntrada(ctx)
	f := fonteDoTeste(c, fontes[0].ID)
	if f.Estado != "erro" || !strings.Contains(f.Erro, "Whisper não está configurado") {
		t.Fatalf("sem Whisper devia dar erro: %+v", f)
	}

	c.a.whisper = whisperDeTeste(t).URL
	c.ok("POST", fmt.Sprintf("/api/entrada/%d/tentar", fontes[0].ID), nil, nil)
	c.a.esvaziarEntrada(ctx)
	if f = fonteDoTeste(c, fontes[0].ID); f.Estado != "pronta" {
		t.Fatalf("depois do Whisper: %+v", f)
	}
}

// "Dividir em fragmentos quando terminar": a fonte pronta entra na fila de fragmentar
// sozinha (a tarefa fica na_fila para a fila do Claude).
func TestEntradaTelegramFragmentar(t *testing.T) {
	pedidos := []pedidoStreaming{}
	f := &iaFalsa{}
	c := clienteComIA(t, f)
	c.a.integra.streaming = streamingDeAulas(t, &pedidos, map[int]int64{2: 7}).URL
	c.a.integra.streamingLink = "https://streaming.internal"
	c.a.ffmpegBin = filepath.Join("testdata", "ffmpeg-falso")
	c.a.whisper = whisperDeTeste(t).URL
	ctx := context.Background()

	var fontes []fonte
	c.ok("POST", "/api/entrada/telegram", map[string]any{
		"refs": []string{"telegram:https://t.me/c/1#2"}, "fragmentar": true,
	}, &fontes)
	c.a.esvaziarEntrada(ctx)
	f2 := fonteDoTeste(c, fontes[0].ID)
	if f2.Estado != "fragmentando" || f2.TarefaID == 0 {
		t.Fatalf("devia estar fragmentando: %+v", f2)
	}
	var tipo string
	if err := c.a.db.QueryRowContext(ctx, `SELECT tipo FROM tarefas WHERE id = ?`, f2.TarefaID).Scan(&tipo); err != nil || tipo != "fragmentar" {
		t.Fatalf("tarefa: %v %q", err, tipo)
	}
}

// O encaixe da fonte telegram: material streaming com a ref e intervalo; o pedido lê só o
// trecho da transcrição da fonte.
func TestEncaixeFonteTelegram(t *testing.T) {
	c := novoCliente(t)
	ctx := context.Background()
	f, err := c.a.criarFonte(ctx, "telegram", "telegram:https://t.me/c/1#2", "telegram:https://t.me/c/1#2", "Aula 2", "")
	if err != nil {
		t.Fatal(err)
	}
	transcricao := "# Aula 2\n\n[00:00] Antes do trecho.\n\n[00:10] Dentro do trecho.\n\n[00:40] Depois do trecho.\n"
	if err := c.a.conteudo.gravar("teste", map[string]*string{pastaDaFonte(f.ID) + "/transcricao.md": &transcricao}); err != nil {
		t.Fatal(err)
	}
	m := material{Tipo: "streaming", Titulo: "Aula 2", Referencia: "telegram:https://t.me/c/1#2", Localizador: "00:05–00:25"}
	arquivos := map[string]string{}
	lidos := c.a.arquivosDoMaterial(ctx, []material{m}, arquivos)
	if lidos[0].Erro != "" {
		t.Fatalf("lido: %s", lidos[0].Erro)
	}
	texto := arquivos[lidos[0].Arquivo]
	if !strings.Contains(texto, "Dentro do trecho") || strings.Contains(texto, "Antes do trecho") || strings.Contains(texto, "Depois do trecho") {
		t.Fatalf("só o trecho devia vir:\n%s", texto)
	}
	// O localizador do encaixe precisa cair dentro do intervalo mandado.
	if !localizadorDentro("00:10–00:20", [][2]int{{5, 25}}) || localizadorDentro("00:10–00:30", [][2]int{{5, 25}}) {
		t.Fatal("localizadorDentro")
	}
	// criarEncaixe aceita o streaming com ref telegram: e recusa um streaming qualquer.
	trilhaComEtapa(c)
	if _, err := c.a.criarEncaixe(ctx, trilha{ID: 1}, m, "", ""); err != nil {
		t.Fatalf("encaixe telegram: %v", err)
	}
	if _, err := c.a.criarEncaixe(ctx, trilha{ID: 1}, material{Tipo: "streaming", Referencia: "https://x/video.mp4"}, "", ""); err == nil {
		t.Fatal("streaming sem telegram: devia falhar")
	}
}
