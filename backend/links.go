package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Links inventados são um risco da IA (DECISOES.md §11, risco 3): o servidor testa cada link
// externo da lição antes de ela chegar ao Lucas. Lição 4 de regras/criar-trilha.md: alguns sites
// recusam robôs (403, 429) e abrem no navegador; isso não é "quebrado", é "não confirmado".

type linkConferido struct {
	URL      string `json:"url"`
	Situacao string `json:"situacao"` // ok · quebrado · nao_confirmado
	Detalhe  string `json:"detalhe,omitempty"`
}

var reLink = regexp.MustCompile(`https?://[^\s<>()\[\]"'` + "`" + `]+`)

func linksDoTexto(texto string) []string {
	vistos := map[string]bool{}
	var lista []string
	for _, u := range reLink.FindAllString(texto, -1) {
		u = strings.TrimRight(u, ".,;:!?*_")
		if !vistos[u] {
			vistos[u] = true
			lista = append(lista, u)
		}
	}
	if len(lista) > 40 {
		lista = lista[:40]
	}
	return lista
}

func conferirLinks(ctx context.Context, texto string) []linkConferido {
	urls := linksDoTexto(texto)
	res := make([]linkConferido, len(urls))
	cliente := &http.Client{Timeout: 12 * time.Second}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res[i] = conferirLink(ctx, cliente, u)
		}(i, u)
	}
	wg.Wait()
	return res
}

func conferirLink(ctx context.Context, cliente *http.Client, u string) linkConferido {
	l := linkConferido{URL: u}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		l.Situacao, l.Detalhe = "quebrado", "endereço inválido"
		return l
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) TrilhaDeEstudo/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*;q=0.8")
	resp, err := cliente.Do(req)
	if err != nil {
		var dns *net.DNSError
		if errors.As(err, &dns) && dns.IsNotFound {
			l.Situacao, l.Detalhe = "quebrado", "o domínio não existe"
		} else {
			l.Situacao, l.Detalhe = "nao_confirmado", "não respondeu"
		}
		return l
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	resp.Body.Close()
	switch {
	case resp.StatusCode < 400:
		l.Situacao = "ok"
	case resp.StatusCode == 404 || resp.StatusCode == 410:
		l.Situacao, l.Detalhe = "quebrado", resp.Status
	default:
		l.Situacao, l.Detalhe = "nao_confirmado", resp.Status
	}
	return l
}
