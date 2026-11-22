package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Projeto de estudo (T52): a aba mostra o caderno do projeto (repositório público
// github.com/Lucas-Wesley/projeto-de-estudo) e monta o prompt "Conversar com a IA". Só LÊ o
// caderno: baixa o repositório inteiro num tarball, guarda em memória e no disco
// (DATA_DIR/projeto/caderno.tar.gz) e atualiza no máximo a cada 10 min (ou pelo botão). Se o
// GitHub falhar, serve a última cópia com aviso.

const (
	limitePacoteProjeto  = 30 << 20 // o tarball inteiro
	limiteArquivoProjeto = 2 << 20  // cada .md
)

type cadernoProjeto struct {
	repo      string // https://github.com/Lucas-Wesley/projeto-de-estudo (links)
	ramo      string
	tarball   string // de onde baixar
	pasta     string // DATA_DIR/projeto
	intervalo time.Duration
	cliente   *http.Client

	mu        sync.Mutex
	arquivos  map[string]string // caminho dentro do repositório → conteúdo (só .md)
	baixadoEm time.Time
	tentadoEm time.Time
	erro      string
	lidoDisco bool
}

// novoCadernoProjeto: PROJETO_REPO (o link do GitHub), PROJETO_RAMO (main) e, se quiser outro
// endereço para o pacote, PROJETO_TARBALL.
func novoCadernoProjeto(dataDir string) *cadernoProjeto {
	repo := strings.TrimSuffix(strings.TrimSuffix(env("PROJETO_REPO", "https://github.com/Lucas-Wesley/projeto-de-estudo"), "/"), ".git")
	ramo := env("PROJETO_RAMO", "main")
	tarball := os.Getenv("PROJETO_TARBALL")
	if tarball == "" {
		tarball = strings.Replace(repo, "https://github.com/", "https://codeload.github.com/", 1) + "/tar.gz/refs/heads/" + ramo
	}
	return &cadernoProjeto{repo: repo, ramo: ramo, tarball: tarball, pasta: filepath.Join(dataDir, "projeto"),
		intervalo: 10 * time.Minute, cliente: &http.Client{Timeout: 30 * time.Second}}
}

// lerPacoteProjeto abre o tar.gz do GitHub e devolve os .md, sem a pasta de cima
// ("projeto-de-estudo-main/ROADMAP.md" → "ROADMAP.md").
func lerPacoteProjeto(dados []byte) (map[string]string, error) {
	gz, err := gzip.NewReader(bytes.NewReader(dados))
	if err != nil {
		return nil, fmt.Errorf("o pacote não é um tar.gz: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(io.LimitReader(gz, 4*limitePacoteProjeto))
	arquivos := map[string]string{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("pacote corrompido: %w", err)
		}
		if h.Typeflag != tar.TypeReg || h.Size > limiteArquivoProjeto {
			continue
		}
		nome := path.Clean(h.Name)
		if i := strings.Index(nome, "/"); i >= 0 {
			nome = nome[i+1:]
		}
		if !strings.HasSuffix(strings.ToLower(nome), ".md") || strings.HasPrefix(nome, "..") {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, limiteArquivoProjeto))
		if err != nil {
			return nil, err
		}
		arquivos[nome] = string(b)
	}
	if _, ok := arquivos["ROADMAP.md"]; !ok {
		return nil, errors.New("o repositório não tem ROADMAP.md")
	}
	return arquivos, nil
}

func (c *cadernoProjeto) baixar(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.tarball, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.cliente.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o GitHub respondeu %d", resp.StatusCode)
	}
	dados, err := io.ReadAll(io.LimitReader(resp.Body, limitePacoteProjeto+1))
	if err != nil {
		return nil, err
	}
	if len(dados) > limitePacoteProjeto {
		return nil, errors.New("o pacote passou de 30 MB")
	}
	return dados, nil
}

func (c *cadernoProjeto) arquivoDisco() string { return filepath.Join(c.pasta, "caderno.tar.gz") }

func (c *cadernoProjeto) lerDoDisco() {
	c.lidoDisco = true
	dados, err := os.ReadFile(c.arquivoDisco())
	if err != nil {
		return
	}
	arquivos, err := lerPacoteProjeto(dados)
	if err != nil {
		log.Printf("projeto: cópia do disco ilegível: %v", err)
		return
	}
	c.arquivos = arquivos
	if info, err := os.Stat(c.arquivoDisco()); err == nil {
		c.baixadoEm = info.ModTime()
	}
}

func (c *cadernoProjeto) salvarNoDisco(dados []byte) {
	if err := os.MkdirAll(c.pasta, 0o755); err != nil {
		log.Printf("projeto: %v", err)
		return
	}
	tmp := c.arquivoDisco() + ".tmp"
	if err := os.WriteFile(tmp, dados, 0o644); err != nil {
		log.Printf("projeto: %v", err)
		return
	}
	if err := os.Rename(tmp, c.arquivoDisco()); err != nil {
		log.Printf("projeto: %v", err)
	}
}

// obter devolve os arquivos do caderno, baixando de novo se passou o intervalo (ou se forcar).
// aviso vem preenchido quando a cópia servida não é a do GitHub de agora.
func (c *cadernoProjeto) obter(ctx context.Context, forcar bool) (map[string]string, time.Time, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.lidoDisco && c.arquivos == nil {
		c.lerDoDisco()
	}
	if forcar || c.tentadoEm.IsZero() || time.Since(c.tentadoEm) >= c.intervalo {
		c.tentadoEm = time.Now()
		dados, err := c.baixar(ctx)
		var arquivos map[string]string
		if err == nil {
			arquivos, err = lerPacoteProjeto(dados)
		}
		if err == nil {
			c.arquivos, c.baixadoEm, c.erro = arquivos, time.Now(), ""
			c.salvarNoDisco(dados)
		} else {
			c.erro = err.Error()
			log.Printf("projeto: não consegui baixar o caderno: %v", err)
		}
	}
	if c.arquivos == nil {
		return nil, time.Time{}, "", fmt.Errorf("não consegui baixar o caderno do GitHub (%s) e não há cópia guardada", c.erro)
	}
	aviso := ""
	if c.erro != "" {
		aviso = fmt.Sprintf("Não consegui atualizar pelo GitHub (%s). Mostrando a cópia de %s.",
			c.erro, c.baixadoEm.In(fuso).Format("02/01 15:04"))
	}
	return c.arquivos, c.baixadoEm, aviso, nil
}

// ---------- a resposta da API ----------

type livroProjeto struct {
	ID         string `json:"id"` // "T21"
	TrilhaID   int64  `json:"trilha_id,omitempty"`
	Titulo     string `json:"titulo"`
	Concluidas int    `json:"concluidas"`
	Total      int    `json:"total"`
}

type contagemRefs struct {
	Casadas    int      `json:"casadas"`
	Parciais   int      `json:"parciais"`
	NaoCasadas int      `json:"nao_casadas"`
	SemPar     []string `json:"sem_par"`
}

type respostaProjeto struct {
	Repositorio string         `json:"repositorio"`
	LinkEstado  string         `json:"link_estado"`
	BaixadoEm   string         `json:"baixado_em"`
	Aviso       string         `json:"aviso,omitempty"`
	Estado      estadoProjeto  `json:"estado"`
	Livros      []livroProjeto `json:"livros"`
	Fases       []*faseProjeto `json:"fases"`
	Referencias contagemRefs   `json:"referencias"`
}

func (c *cadernoProjeto) linkArquivo(caminho string) string {
	return c.repo + "/blob/" + c.ramo + "/" + caminho
}

// montarProjeto junta o caderno e o banco: lê os documentos, liga as referências às trilhas,
// acha o caderno de cada item e monta o prompt.
func (a *app) montarProjeto(ctx context.Context, arquivos map[string]string) (respostaProjeto, error) {
	c := a.projeto
	r := lerRoadmap(arquivos["ROADMAP.md"])
	cat, err := a.catalogo(ctx)
	if err != nil {
		return respostaProjeto{}, err
	}
	resp := respostaProjeto{Repositorio: c.repo, LinkEstado: c.linkArquivo("ESTADO.md"),
		Estado: lerEstado(arquivos["ESTADO.md"], r), Livros: []livroProjeto{}, Fases: r.Fases}
	if resp.Fases == nil {
		resp.Fases = []*faseProjeto{}
	}
	resp.Referencias.SemPar = []string{}
	semPar := map[string]bool{}
	contar := func(refs []referenciaProjeto, texto string) {
		if len(refs) == 0 {
			resp.Referencias.NaoCasadas++
			if t := limparRef(texto); t != "" && !semPar[t] {
				semPar[t] = true
				resp.Referencias.SemPar = append(resp.Referencias.SemPar, t)
			}
			return
		}
		for _, ref := range refs {
			switch {
			case ref.Tipo == "":
				resp.Referencias.NaoCasadas++
				if !semPar[ref.Texto] {
					semPar[ref.Texto] = true
					resp.Referencias.SemPar = append(resp.Referencias.SemPar, ref.Texto)
				}
			case ref.Parcial:
				resp.Referencias.Parciais++
			default:
				resp.Referencias.Casadas++
			}
		}
	}
	ligar := func(l *linhaEstudo) {
		if l == nil {
			return
		}
		l.Refs = cat.referencias(l.Texto)
		contar(l.Refs, l.Texto)
	}

	modelo := modeloPrompt(arquivos["PROMPT.md"])
	for _, it := range r.Itens {
		for i := range it.Estudo {
			ligar(&it.Estudo[i])
		}
		ligar(it.Livros71)
		ligar(it.Trilhas71)
		for i := range it.Capitulos {
			cp := &it.Capitulos[i]
			cp.Refs = cat.referencias(cp.Livro + " cap. " + cp.Caps)
			contar(cp.Refs, cp.Livro+" cap. "+cp.Caps)
		}

		it.Link = c.linkArquivo("ROADMAP.md") + "#" + ancoraGitHub(it.ID+" "+it.Titulo)
		if caminho, texto := cadernoDoItem(arquivos, it.ID); caminho != "" {
			it.CadernoArquivo = strings.TrimPrefix(caminho, "itens/")
			it.Caderno = texto
			it.LinkCaderno = c.linkArquivo(caminho)
			it.Estado = estadoDoCaderno(texto)
			it.Escolhido = estudoDoCaderno(texto)
			for i := range it.Escolhido {
				it.Escolhido[i].Refs = cat.referencias(it.Escolhido[i].Fonte)
			}
		}
		switch {
		case it.Feito || normalizar(it.Estado) == "feito":
			it.Situacao = "feito"
		case it.CadernoArquivo != "" || resp.Estado.ItemID == it.ID:
			it.Situacao = "em_andamento"
		default:
			it.Situacao = "a_fazer"
		}
		it.Prompt = promptDoItem(modelo, it, resp.Estado)
	}
	for _, it := range r.Itens {
		if it.Situacao != "feito" {
			resp.Estado.ProximoID = it.ID
			break
		}
	}

	// Os livros (a 7.2): capítulos concluídos / total, da própria trilha.
	ids := map[string]string{}
	for _, l := range r.Livros {
		ids[l.ID] = l.Titulo
	}
	for _, l := range []string{"T21", "T22", "T23"} {
		if _, ok := ids[l]; !ok {
			ids[l] = ""
		}
	}
	var chaves []string
	for id := range ids {
		chaves = append(chaves, id)
	}
	sort.Slice(chaves, func(i, j int) bool {
		return len(chaves[i]) < len(chaves[j]) || len(chaves[i]) == len(chaves[j]) && chaves[i] < chaves[j]
	})
	for _, id := range chaves {
		lv := livroProjeto{ID: id, Titulo: ids[id]}
		var tid int64
		fmt.Sscanf(id, "T%d", &tid)
		if t := cat[tid]; t != nil {
			lv.TrilhaID, lv.Titulo = t.ID, t.Titulo
			lv.Total = len(t.Etapas)
			for _, e := range t.Etapas {
				if e.Status == "concluida" {
					lv.Concluidas++
				}
			}
		}
		resp.Livros = append(resp.Livros, lv)
	}
	return resp, nil
}

// linhaDoPrompt: "✔ T13 › Concorrência › Goroutines e o modelo…; T13 › …".
func linhaDoPrompt(l *linhaEstudo) string {
	var partes []string
	for _, ref := range l.Refs {
		if ref.Tipo != "" && !ref.Parcial {
			partes = append(partes, ref.Caminho)
		} else {
			partes = append(partes, ref.Texto)
		}
	}
	texto := strings.Join(partes, "; ")
	if texto == "" {
		texto = limparRef(l.Texto)
	}
	if l.Marca != "" {
		texto = l.Marca + " " + texto
	}
	return texto
}

func promptDoItem(modelo string, it *itemProjeto, estado estadoProjeto) string {
	fase := it.Estado
	if fase == "" && estado.ItemID == it.ID && estado.Fase != "" {
		fase = estado.Fase
	}
	if fase == "" {
		if it.Feito {
			fase = "feito"
		} else {
			fase = "problema"
		}
	}
	arquivo := it.CadernoArquivo
	if arquivo == "" {
		arquivo = it.ID + "-" + slugificar(it.Titulo) + ".md"
	}

	var estudo []string
	var concluidas []string
	visto := map[string]bool{}
	juntar := func(l *linhaEstudo, rotulo string) {
		if l == nil {
			return
		}
		estudo = append(estudo, "- "+rotulo+linhaDoPrompt(l))
		for _, ref := range l.Refs {
			for _, c := range ref.concluidas {
				if !visto[c] {
					visto[c] = true
					concluidas = append(concluidas, c)
				}
			}
		}
	}
	for i := range it.Estudo {
		juntar(&it.Estudo[i], "")
	}
	juntar(it.Livros71, "Livros (seção 7.1): ")
	juntar(it.Trilhas71, "Trilhas e cursos (seção 7.1): ")
	for i := range it.Capitulos {
		cp := &it.Capitulos[i]
		for _, ref := range cp.Refs {
			for _, c := range ref.concluidas {
				if !visto[c] {
					visto[c] = true
					concluidas = append(concluidas, c)
				}
			}
		}
	}
	textoEstudo := strings.Join(estudo, "\n")
	if textoEstudo == "" {
		textoEstudo = "(o roadmap não indica estudo para este item)"
	}
	textoConcluidas := "nenhuma"
	if len(concluidas) > 0 {
		textoConcluidas = "\n- " + strings.Join(concluidas, "\n- ")
	}
	return strings.NewReplacer(
		"{{id}}", it.ID,
		"{{titulo}}", it.Titulo,
		"{{fase}}", fase,
		"{{arquivo}}", arquivo,
		"{{estudo}}", textoEstudo,
		"{{concluidas}}", textoConcluidas,
	).Replace(modelo)
}

func (a *app) responderProjeto(w http.ResponseWriter, r *http.Request, forcar bool) {
	if a.projeto == nil {
		erroJSON(w, http.StatusServiceUnavailable, "o projeto de estudo não está configurado")
		return
	}
	arquivos, baixadoEm, aviso, err := a.projeto.obter(r.Context(), forcar)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	resp, err := a.montarProjeto(r.Context(), arquivos)
	if err != nil {
		erroInterno(w, err)
		return
	}
	resp.BaixadoEm = baixadoEm.UTC().Format(time.RFC3339)
	resp.Aviso = aviso
	responderJSON(w, http.StatusOK, resp)
}

// GET /api/projeto: o caderno (cópia de até 10 min) ligado às trilhas.
func (a *app) handleProjeto(w http.ResponseWriter, r *http.Request) { a.responderProjeto(w, r, false) }

// POST /api/projeto/atualizar: baixa de novo do GitHub agora.
func (a *app) handleAtualizarProjeto(w http.ResponseWriter, r *http.Request) {
	a.responderProjeto(w, r, true)
}
