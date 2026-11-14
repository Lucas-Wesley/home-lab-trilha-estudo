package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// conteudo guarda os textos (lições e documentos) em Markdown, numa pasta versionada pelo git
// local (T3): nunca vai para o GitHub, e o histórico permite ver e desfazer qualquer mudança.
//
//	conteudo/
//	  trilhas/<slug>/licoes/<chave>.md  a lição de cada etapa
//	  trilhas/<slug>/documentos/<d>.md  material próprio da trilha (ex.: os docs de estudo/)
//	  trilhas/<slug>/trilha.json      a estrutura exportada do banco, para o histórico
type conteudo struct {
	raiz       string
	autorNome  string
	autorEmail string
	git        bool // false quando o git não existe no ambiente: grava os arquivos sem versionar

	mu sync.Mutex
}

func abrirConteudo(raiz, autorNome, autorEmail string) (*conteudo, error) {
	if err := os.MkdirAll(filepath.Join(raiz, "trilhas"), 0o755); err != nil {
		return nil, err
	}
	c := &conteudo{raiz: raiz, autorNome: autorNome, autorEmail: autorEmail}
	if _, err := exec.LookPath("git"); err != nil {
		log.Print("conteúdo: git não encontrado, os textos ficam sem histórico")
		return c, nil
	}
	c.git = true
	if _, err := os.Stat(filepath.Join(raiz, ".git")); errors.Is(err, fs.ErrNotExist) {
		if _, err := c.rodarGit(context.Background(), "init", "-q", "-b", "main"); err != nil {
			return nil, fmt.Errorf("git init do conteúdo: %w", err)
		}
	}
	return c, nil
}

func (c *conteudo) rodarGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = c.raiz
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+c.autorNome, "GIT_AUTHOR_EMAIL="+c.autorEmail,
		"GIT_COMMITTER_NAME="+c.autorNome, "GIT_COMMITTER_EMAIL="+c.autorEmail,
	)
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(erros.String()))
	}
	return saida.String(), nil
}

// gravar escreve (ou apaga, com texto nil) arquivos e faz um commit com a mensagem.
func (c *conteudo) gravar(mensagem string, arquivos map[string]*string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for rel, texto := range arquivos {
		caminho, err := c.caminho(rel)
		if err != nil {
			return err
		}
		if texto == nil {
			if err := os.Remove(caminho); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(caminho), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(caminho, []byte(*texto), 0o644); err != nil {
			return err
		}
	}
	c.commit(mensagem)
	return nil
}

// apagarPasta remove uma pasta inteira do conteúdo (ex.: a trilha excluída) e faz o commit.
func (c *conteudo) apagarPasta(mensagem, rel string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	caminho, err := c.caminho(rel)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(caminho); err != nil {
		return err
	}
	c.commit(mensagem)
	return nil
}

func (c *conteudo) commit(mensagem string) {
	if !c.git {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.rodarGit(ctx, "add", "-A"); err != nil {
		log.Printf("conteúdo: git add: %v", err)
		return
	}
	if out, _ := c.rodarGit(ctx, "status", "--porcelain"); strings.TrimSpace(out) == "" {
		return
	}
	if _, err := c.rodarGit(ctx, "commit", "-q", "-m", mensagem); err != nil {
		log.Printf("conteúdo: git commit: %v", err)
	}
}

func (c *conteudo) ler(rel string) (string, error) {
	caminho, err := c.caminho(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(caminho)
	return string(b), err
}

// caminho impede que um nome vindo da API saia da pasta do conteúdo.
func (c *conteudo) caminho(rel string) (string, error) {
	limpo := filepath.Clean(filepath.FromSlash(rel))
	if limpo == "." || filepath.IsAbs(limpo) || strings.HasPrefix(limpo, "..") {
		return "", fmt.Errorf("caminho inválido: %q", rel)
	}
	return filepath.Join(c.raiz, limpo), nil
}

var slugValido = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// documento é um material próprio em Markdown de uma trilha (ex.: os docs de estudo do homelab).
type documento struct {
	Slug     string `json:"slug"`
	Titulo   string `json:"titulo"`
	Conteudo string `json:"conteudo,omitempty"`
}

// documentos lista os .md de uma pasta do conteúdo (pasta que não existe = nenhum documento).
func (c *conteudo) documentos(pasta string) ([]documento, error) {
	lista := []documento{}
	dir, err := c.caminho(pasta)
	if err != nil {
		return nil, err
	}
	entradas, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return lista, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		texto, err := c.ler(pasta + "/" + e.Name())
		if err != nil {
			return nil, err
		}
		lista = append(lista, documento{Slug: slug, Titulo: tituloDoMarkdown(texto, slug)})
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Slug < lista[j].Slug })
	return lista, nil
}

// tituloDoMarkdown pega o primeiro "# título" do texto.
func tituloDoMarkdown(texto, padrao string) string {
	for _, linha := range strings.Split(texto, "\n") {
		if t, ok := strings.CutPrefix(strings.TrimSpace(linha), "# "); ok {
			return strings.TrimSpace(t)
		}
	}
	return padrao
}
