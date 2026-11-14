package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var version = "dev"

//go:embed all:web/dist
var webDist embed.FS

func main() {
	dataDir := env("DATA_DIR", "./data")
	addr := env("ADDR", ":8080")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := abrirBanco(filepath.Join(dataDir, "trilha.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	carregarFuso(env("FUSO", "America/Sao_Paulo"))

	c, err := abrirConteudo(filepath.Join(dataDir, "conteudo"),
		env("GIT_AUTOR_NOME", "Trilha de Estudo"), env("GIT_AUTOR_EMAIL", "trilha@homelab.local"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	web, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	a := &app{db: db, conteudo: c, fila: novaFila(db), ia: &configIA{
		claude:        env("CLAUDE_BIN", "claude"),
		modelo:        os.Getenv("CLAUDE_MODEL"),
		regras:        env("REGRAS_DIR", "../regras"),
		regrasEditada: filepath.Join(dataDir, "regras"),
		tarefas:       filepath.Join(dataDir, "tarefas"),
		conferirLinks: conferirLinks,
	}, integra: integracoes{
		flashcards:     strings.TrimSuffix(os.Getenv("FLASHCARDS_URL"), "/"),
		flashcardsLink: strings.TrimSuffix(env("FLASHCARDS_LINK", "https://flashcards.internal"), "/"),
		trilhaLink:     strings.TrimSuffix(env("TRILHA_LINK", "https://trilha.internal"), "/"),
	}}
	a.ligarIA()
	if err := a.fila.recuperar(ctx); err != nil {
		log.Fatal(err)
	}
	go a.fila.rodar(ctx)

	srv := &http.Server{
		Addr:              addr,
		Handler:           novoServidor(a, web, version),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("trilha %s em %s", version, addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Print(err)
	}
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

func agora() string {
	return time.Now().UTC().Format(time.RFC3339)
}
