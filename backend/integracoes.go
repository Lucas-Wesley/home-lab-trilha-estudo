package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Os outros apps do ecossistema, pela rede interna do Docker (compose.prod.yaml). Fase 5: o
// Flashcards; fase 6: Biblioteca e Acervo; Entrada fase 7 (T41): o Streaming (cursos do Telegram).

var clienteApps = &http.Client{}

// chamarApp fala com a API de outro app e devolve a mensagem de erro dele, se houver.
func chamarApp(ctx context.Context, base, nome, metodo, caminho string, corpo, destino any) (int, error) {
	if base == "" {
		return 0, fmt.Errorf("o %s não está configurado neste servidor", nome)
	}
	if _, tem := ctx.Deadline(); !tem {
		var cancelar context.CancelFunc
		ctx, cancelar = context.WithTimeout(ctx, 20*time.Second)
		defer cancelar()
	}
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return 0, err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, base+caminho, leitor)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clienteApps.Do(req)
	if err != nil {
		log.Printf("%s: %v", nome, err)
		return 0, fmt.Errorf("o %s não respondeu; ele está no ar?", nome)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode >= 300 {
		var e struct {
			Erro  string `json:"erro"`
			Error string `json:"error"` // a Biblioteca
		}
		json.Unmarshal(b, &e)
		if e.Erro = primeiro(e.Erro, e.Error); e.Erro == "" {
			e.Erro = fmt.Sprintf("o %s respondeu %d", nome, res.StatusCode)
		}
		return res.StatusCode, errors.New(e.Erro)
	}
	if destino != nil {
		return res.StatusCode, json.Unmarshal(b, destino)
	}
	return res.StatusCode, nil
}

// statusOu: o status do outro app quando é um erro do pedido (4xx); senão, o alternativo.
func statusOu(status, alternativo int) int {
	if status >= 400 && status < 500 {
		return status
	}
	return alternativo
}
