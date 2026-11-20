package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Entrada, fase 6 (T40): "Trazer do Acervo" e a limpeza dos fragmentos "sem destino" em 30 dias.

// acervoDeVideos: um Acervo falso com uma Referência de vídeo completa e uma de artigo.
func acervoDeVideos(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/referencias/rv1":
			w.Write([]byte(`{"id":"rv1","titulo":"Introdução a Redes 1","tipo":"video","estado":"validado",
				"url":"https://youtu.be/EOmzo5d0F9Y","video_id":"EOmzo5d0F9Y","canal":"Akita","duracao":"46:58",
				"textos":{"cru":"[00:00] Introdução.\n[00:30] O que é rede."}}`))
		case "/api/referencias/ra1":
			w.Write([]byte(`{"id":"ra1","titulo":"Artigo","tipo":"artigo","estado":"validado",
				"url":"https://exemplo.org/a","textos":{"cru":"texto"}}`))
		case "/api/referencias/rs1":
			w.Write([]byte(`{"id":"rs1","titulo":"Vídeo sem cru","tipo":"video","estado":"novo",
				"url":"https://youtu.be/EOmzo5d0F9Y","textos":{}}`))
		default:
			w.WriteHeader(404)
			w.Write([]byte(`{"erro":"não achei"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTrazerDoAcervo(t *testing.T) {
	c := novoCliente(t)
	c.a.integra.acervo = acervoDeVideos(t).URL

	// Traz a Referência: fonte pronta, com a transcrição gravada no conteúdo.
	var f fonte
	c.ok("POST", "/api/entrada/acervo", map[string]any{"referencia": "rv1"}, &f)
	if f.Tipo != "acervo" || f.VideoID != "rv1" || f.Estado != "pronta" ||
		f.URL != "https://www.youtube.com/watch?v=EOmzo5d0F9Y" ||
		f.Titulo != "Introdução a Redes 1" || f.Canal != "Akita" || f.Duracao != 2818 {
		t.Fatalf("fonte trazida errada: %+v", f)
	}
	cru, err := c.a.conteudo.ler(pastaDaFonte(f.ID) + "/transcricao.md")
	if err != nil || cru != "[00:00] Introdução.\n[00:30] O que é rede." {
		t.Fatalf("transcrição: %v %q", err, cru)
	}

	// A mesma Referência de novo devolve a fonte que existe (não duplica).
	var deNovo fonte
	c.ok("POST", "/api/entrada/acervo", map[string]any{"referencia": "rv1"}, &deNovo)
	if deNovo.ID != f.ID {
		t.Fatalf("a mesma Referência criou outra fonte: %d × %d", f.ID, deNovo.ID)
	}
	var total int
	if err := c.a.db.QueryRow(`SELECT COUNT(*) FROM fontes`).Scan(&total); err != nil || total != 1 {
		t.Fatalf("devia haver 1 fonte, há %d", total)
	}

	// Referência que não é vídeo e vídeo sem transcrição crua não entram.
	if s := c.chamar("POST", "/api/entrada/acervo", map[string]any{"referencia": "ra1"}, nil); s != 400 {
		t.Fatalf("artigo: %d", s)
	}
	if s := c.chamar("POST", "/api/entrada/acervo", map[string]any{"referencia": "rs1"}, nil); s != 400 {
		t.Fatalf("vídeo sem cru: %d", s)
	}
}

func TestTrazerDoAcervoVideoJaNaEntrada(t *testing.T) {
	c := novoCliente(t)
	c.a.integra.acervo = acervoDeVideos(t).URL

	// O vídeo já entrou pelo YouTube: trazer a Referência devolve essa fonte, sem criar outra.
	var lista []fonte
	c.ok("POST", "/api/entrada", map[string]any{"url": "https://www.youtube.com/watch?v=EOmzo5d0F9Y"}, &lista)
	if len(lista) != 1 || lista[0].Tipo != "youtube" {
		t.Fatalf("fonte youtube: %+v", lista)
	}
	var f fonte
	c.ok("POST", "/api/entrada/acervo", map[string]any{"referencia": "rv1"}, &f)
	if f.ID != lista[0].ID || f.Tipo != "youtube" {
		t.Fatalf("devolveu a fonte errada: %+v", f)
	}
}

func TestSegundosDoTempoDeDuracao(t *testing.T) {
	for tempo, queremos := range map[string]int{"46:58": 2818, "1:02:03": 3723} {
		if s, err := segundosDoTempo(tempo); err != nil || s != queremos {
			t.Fatalf("%q = %d, %v (queria %d)", tempo, s, err, queremos)
		}
	}
}

// fragmentoSemDestino: um fragmento na fonte 1, no estado e com o carimbo pedidos.
func fragmentoSemDestino(t *testing.T, c *cliente, estado, semDestinoEm string) int64 {
	t.Helper()
	res, err := c.a.db.Exec(`INSERT INTO fragmentos (fonte_id, ordem, inicio, fim, titulo, estado, sem_destino_em, criado_em, atualizado_em)
		VALUES (1, 0, 0, 60, 'x', ?, ?, ?, ?)`, estado, semDestinoEm, agora(), agora())
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestLimpezaSemDestinoEm30Dias(t *testing.T) {
	c := novoCliente(t)
	if _, err := c.a.criarFonte(t.Context(), "acervo", "https://x", "f1", "F", ""); err != nil {
		t.Fatal(err)
	}
	em := func(dias int) string {
		return time.Now().AddDate(0, 0, -dias).UTC().Format(time.RFC3339)
	}
	velho := fragmentoSemDestino(t, c, "sem_destino", em(31))
	novo := fragmentoSemDestino(t, c, "sem_destino", em(29))
	enviado := fragmentoSemDestino(t, c, "enviado", em(31))
	confirmado := fragmentoSemDestino(t, c, "confirmado", em(31))

	n, err := c.a.apagarSemDestino(time.Now())
	if err != nil || n != 1 {
		t.Fatalf("apagou %d (%v), queria 1", n, err)
	}
	var sobrou int
	for _, id := range []int64{novo, enviado, confirmado} {
		c.a.db.QueryRow(`SELECT COUNT(*) FROM fragmentos WHERE id = ?`, id).Scan(&sobrou)
		if sobrou != 1 {
			t.Fatalf("o fragmento %d não devia ter sumido", id)
		}
	}
	c.a.db.QueryRow(`SELECT COUNT(*) FROM fragmentos WHERE id = ?`, velho).Scan(&sobrou)
	if sobrou != 0 {
		t.Fatal("o fragmento de 31 dias ficou")
	}

	// A fonte sem fragmentos continua (o Lucas apaga se quiser).
	c.a.db.QueryRow(`SELECT COUNT(*) FROM fontes WHERE id = 1`).Scan(&sobrou)
	if sobrou != 1 {
		t.Fatal("a fonte vazia foi apagada junto")
	}
}
