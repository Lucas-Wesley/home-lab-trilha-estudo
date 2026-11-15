package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// IA = Claude Code headless (T4), no mesmo molde do Acervo: "claude -p" com a regra do tipo de
// tarefa, as ferramentas liberadas e um JSON Schema para a resposta. A conta vem do Hub
// (CLAUDE_TOKEN_ARQUIVO, lido a cada chamada); fora do homelab, CLAUDE_CODE_OAUTH_TOKEN no .env.

type configIA struct {
	claude        string // o executável (CLAUDE_BIN)
	modelo        string // padrão de todas as tarefas (CLAUDE_MODEL); vazio = o padrão do Claude Code
	regras        string // pasta das regras que vêm com o app (REGRAS_DIR)
	regrasEditada string // pasta das regras editadas em Ajustes (DATA_DIR/regras)
	tarefas       string // pasta de trabalho de cada tarefa (DATA_DIR/tarefas)

	// chamar faz a chamada de verdade; os testes trocam por uma IA de mentira.
	chamar func(ctx context.Context, p pedidoClaude) (respostaClaude, error)
	// conferirLinks testa os links externos de um texto (os testes trocam para não sair à rede).
	conferirLinks func(ctx context.Context, texto string) []linkConferido
}

type pedidoClaude struct {
	pasta       string   // diretório de trabalho (os arquivos do pedido estão nele)
	extras      []string // outras pastas que ele pode ler
	regra       string   // texto da regra
	instrucao   string   // o pedido específico desta tarefa
	ferramentas string
	esquema     string // JSON Schema da resposta
	modelo      string
}

type respostaClaude struct {
	Resultado   string          `json:"result"`
	Erro        bool            `json:"is_error"`
	Estruturado json.RawMessage `json:"structured_output"`
}

const ferramentasProibidas = "Bash,Write,Edit,NotebookEdit,Task"

func ambienteDoClaude() []string {
	arquivo := os.Getenv("CLAUDE_TOKEN_ARQUIVO")
	if arquivo == "" {
		return nil
	}
	b, err := os.ReadFile(arquivo)
	token := strings.TrimSpace(string(b))
	if err != nil || token == "" {
		return nil
	}
	// Repetida, a variável vale pelo último valor: o do arquivo passa na frente do .env.
	return append(os.Environ(), "CLAUDE_CODE_OAUTH_TOKEN="+token)
}

func primeiraLinha(s, alternativa string) string {
	s = strings.TrimSpace(strings.SplitN(strings.TrimSpace(s), "\n", 2)[0])
	if s == "" {
		return alternativa
	}
	return s
}

func (c *configIA) chamarClaude(ctx context.Context, p pedidoClaude) (respostaClaude, error) {
	var r respostaClaude
	prompt := p.instrucao + "\n\nSiga as regras abaixo.\n\n" + p.regra
	args := []string{"-p", prompt, "--output-format", "json",
		"--allowedTools", p.ferramentas, "--disallowedTools", ferramentasProibidas}
	for _, d := range p.extras {
		args = append(args, "--add-dir", d)
	}
	if p.esquema != "" {
		args = append(args, "--json-schema", p.esquema)
	}
	if p.modelo != "" {
		args = append(args, "--model", p.modelo)
	}
	cmd := exec.CommandContext(ctx, c.claude, args...)
	cmd.Env = ambienteDoClaude()
	cmd.Dir = p.pasta
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	errExec := cmd.Run()
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if jerr := json.Unmarshal(saida.Bytes(), &r); jerr != nil {
		msg := primeiraLinha(erros.String(), primeiraLinha(saida.String(), "sem resposta"))
		if errExec != nil && msg == "sem resposta" {
			msg = errExec.Error()
		}
		return r, fmt.Errorf("o Claude falhou: %s", msg)
	}
	if r.Erro || errExec != nil {
		msg := primeiraLinha(r.Resultado, primeiraLinha(erros.String(), "erro sem mensagem"))
		if strings.Contains(msg, "Not logged in") || strings.Contains(msg, "/login") || strings.Contains(msg, "OAuth") {
			return r, errors.New("o Claude recusou o login: o token da conta ativa venceu ou falta (no homelab, troque de conta no Hub, na tela Claude; fora dele, CLAUDE_CODE_OAUTH_TOKEN no .env)")
		}
		return r, fmt.Errorf("o Claude falhou: %s", msg)
	}
	return r, nil
}

// dadosDaResposta lê a resposta estruturada; sem ela, procura um JSON no texto.
func dadosDaResposta(r respostaClaude, destino any) error {
	if len(r.Estruturado) > 0 && string(r.Estruturado) != "null" {
		return json.Unmarshal(r.Estruturado, destino)
	}
	texto := r.Resultado
	ini, fim := strings.Index(texto, "{"), strings.LastIndex(texto, "}")
	if ini < 0 || fim < ini {
		return fmt.Errorf("o Claude não devolveu dados: %s", primeiraLinha(texto, "resposta vazia"))
	}
	return json.Unmarshal([]byte(texto[ini:fim+1]), destino)
}

// ---------- regras (T6): editáveis em Ajustes, sem deploy ----------

type regra struct {
	Arquivo   string `json:"arquivo"`
	Titulo    string `json:"titulo"`
	Para      string `json:"para"`
	Editada   bool   `json:"editada"`
	Texto     string `json:"texto,omitempty"`
	Original  string `json:"original,omitempty"`
	EditadaEm string `json:"editada_em,omitempty"`
}

// As regras que o app usa. criar-trilha.md é o método da fase 4 (já editável).
var regrasDoApp = []regra{
	{Arquivo: "licao.md", Titulo: "Lição da etapa", Para: "Gerar lição e pedir mudança numa lição"},
	{Arquivo: "exercicios.md", Titulo: "Exercícios", Para: "Gerar os exercícios da etapa (escolha e completar)"},
	{Arquivo: "simulado.md", Titulo: "Simulado", Para: "Gerar perguntas abertas para o banco do simulado"},
	{Arquivo: "corrigir-simulado.md", Titulo: "Corrigir o simulado", Para: "Corrigir as respostas do simulado"},
	{Arquivo: "reforco.md", Titulo: "Reforço", Para: "Propor etapas de reforço para uma etapa em que errei"},
	{Arquivo: "materiais.md", Titulo: "Sugerir material", Para: "Procurar material para uma etapa (web, Biblioteca, Acervo, documentos)"},
	{Arquivo: "entrevista.md", Titulo: "Entrevista", Para: "Criar trilha (a IA monta): pré-pesquisa e perguntas"},
	{Arquivo: "mapa.md", Titulo: "Mapa do assunto", Para: "Criar trilha (a IA monta): o mapa completo e o recorte sugerido"},
	{Arquivo: "validar-lista.md", Titulo: "Validar a lista", Para: "Criar trilha (eu tenho a lista): o que falta na lista"},
	{Arquivo: "espinha.md", Titulo: "Espinha dorsal", Para: "Criar trilha (a partir de material): o material vira a sequência de etapas"},
	{Arquivo: "nivelamento.md", Titulo: "Mini-teste de nível", Para: "Criar trilha: perguntas para saber onde começar (opcional)"},
	{Arquivo: "plano.md", Titulo: "Plano da trilha", Para: "Criar trilha: módulos e etapas (vai junto com o método abaixo)"},
	{Arquivo: "criar-trilha.md", Titulo: "Criar trilha (método)", Para: "O método e os critérios C1–C15, que vão junto com o plano"},
}

func regraConhecida(arquivo string) bool {
	for _, r := range regrasDoApp {
		if r.Arquivo == arquivo {
			return true
		}
	}
	return false
}

// lerRegra devolve a versão editada, se houver, e a original.
func (c *configIA) lerRegra(arquivo string) (regra, error) {
	r := regra{Arquivo: arquivo}
	for _, x := range regrasDoApp {
		if x.Arquivo == arquivo {
			r = x
		}
	}
	b, err := os.ReadFile(filepath.Join(c.regras, arquivo))
	if err != nil {
		return r, fmt.Errorf("regra %s não encontrada: %w", arquivo, err)
	}
	r.Original = string(b)
	r.Texto = r.Original
	caminho := filepath.Join(c.regrasEditada, arquivo)
	if b, err := os.ReadFile(caminho); err == nil {
		r.Texto, r.Editada = string(b), true
		if info, err := os.Stat(caminho); err == nil {
			r.EditadaEm = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return r, err
	}
	return r, nil
}

func (a *app) handleRegras(w http.ResponseWriter, r *http.Request) {
	lista := []regra{}
	for _, x := range regrasDoApp {
		rg, err := a.ia.lerRegra(x.Arquivo)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		rg.Texto, rg.Original = "", ""
		lista = append(lista, rg)
	}
	responderJSON(w, http.StatusOK, lista)
}

func arquivoDaRegra(r *http.Request) (string, error) {
	arquivo := r.PathValue("arquivo")
	if !regraConhecida(arquivo) {
		return "", errNaoEncontrou
	}
	return arquivo, nil
}

func (a *app) handleRegra(w http.ResponseWriter, r *http.Request) {
	arquivo, err := arquivoDaRegra(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	rg, err := a.ia.lerRegra(arquivo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, rg)
}

func (a *app) handleSalvarRegra(w http.ResponseWriter, r *http.Request) {
	arquivo, err := arquivoDaRegra(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Texto string `json:"texto"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if strings.TrimSpace(corpo.Texto) == "" {
		a.responderErro(w, erroDeEntrada{errors.New("a regra está vazia; para voltar à original, use Restaurar")})
		return
	}
	if err := os.MkdirAll(a.ia.regrasEditada, 0o755); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := os.WriteFile(filepath.Join(a.ia.regrasEditada, arquivo), []byte(corpo.Texto), 0o644); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleRegra(w, r)
}

func (a *app) handleRestaurarRegra(w http.ResponseWriter, r *http.Request) {
	arquivo, err := arquivoDaRegra(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := os.Remove(filepath.Join(a.ia.regrasEditada, arquivo)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		a.responderErro(w, err)
		return
	}
	a.handleRegra(w, r)
}

// ---------- ajustes ----------

// Instruções globais (§5.1) e o modelo de cada tipo de tarefa (T5). Modelo vazio = o padrão
// (CLAUDE_MODEL, ou o do Claude Code).
type ajustes struct {
	InstrucoesGlobais string `json:"instrucoes_globais"`
	ModeloLicao       string `json:"modelo_licao"`
	ModeloCriacao     string `json:"modelo_criacao"`
	ModeloEstudo      string `json:"modelo_estudo"` // exercícios, simulado e a correção (fase 5)
	NotaMinima        string `json:"nota_minima"`   // critério automático: % nos exercícios (vazio = 80)
	ModeloPadrao      string `json:"modelo_padrao"` // só leitura: CLAUDE_MODEL
}

var chavesAjustes = map[string]func(*ajustes) *string{
	"instrucoes_globais": func(x *ajustes) *string { return &x.InstrucoesGlobais },
	"modelo_licao":       func(x *ajustes) *string { return &x.ModeloLicao },
	"modelo_criacao":     func(x *ajustes) *string { return &x.ModeloCriacao },
	"modelo_estudo":      func(x *ajustes) *string { return &x.ModeloEstudo },
	"nota_minima":        func(x *ajustes) *string { return &x.NotaMinima },
}

func (a *app) lerAjustes(ctx context.Context) (ajustes, error) {
	x := ajustes{ModeloPadrao: a.ia.modelo}
	rows, err := a.db.QueryContext(ctx, `SELECT chave, valor FROM ajustes`)
	if err != nil {
		return x, err
	}
	defer rows.Close()
	for rows.Next() {
		var chave, valor string
		if err := rows.Scan(&chave, &valor); err != nil {
			return x, err
		}
		if campo, ok := chavesAjustes[chave]; ok {
			*campo(&x) = valor
		}
	}
	return x, rows.Err()
}

var modelosAceitos = []string{"", "opus", "sonnet", "haiku"}

func (a *app) handleAjustes(w http.ResponseWriter, r *http.Request) {
	x, err := a.lerAjustes(r.Context())
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, x)
}

func (a *app) handleSalvarAjustes(w http.ResponseWriter, r *http.Request) {
	var x ajustes
	if err := lerCorpo(r, &x); err != nil {
		a.responderErro(w, err)
		return
	}
	x.NotaMinima = strings.TrimSuffix(strings.TrimSpace(x.NotaMinima), "%")
	if n, err := strconv.Atoi(x.NotaMinima); x.NotaMinima != "" && (err != nil || n < 1 || n > 100) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("a nota mínima vai de 1 a 100 (%%), não %q", x.NotaMinima)})
		return
	}
	for _, m := range []*string{&x.ModeloLicao, &x.ModeloCriacao, &x.ModeloEstudo} {
		*m = strings.TrimSpace(*m)
		if !contem(modelosAceitos, *m) && !strings.HasPrefix(*m, "claude-") {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("modelo desconhecido: %q (use opus, sonnet, haiku ou o nome completo)", *m)})
			return
		}
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	for chave, campo := range chavesAjustes {
		if _, err := tx.ExecContext(r.Context(), `INSERT INTO ajustes (chave, valor, atualizado_em) VALUES (?, ?, ?)
			ON CONFLICT (chave) DO UPDATE SET valor = excluded.valor, atualizado_em = excluded.atualizado_em`,
			chave, strings.TrimSpace(*campo(&x)), agora()); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleAjustes(w, r)
}

// modeloPara: o do tipo de tarefa em Ajustes, senão o padrão. Os passos da criação usam um só.
func (a *app) modeloPara(ctx context.Context, tipo string) string {
	if ehCriacao(tipo) || tipo == tipoReforco || tipo == tipoNivelamento {
		tipo = "criacao"
	}
	if ehEstudo(tipo) || tipo == tipoCorrecao {
		tipo = "estudo"
	}
	if tipo == tipoMateriais {
		tipo = "licao"
	}
	var valor string
	a.db.QueryRowContext(ctx, `SELECT valor FROM ajustes WHERE chave = ?`, "modelo_"+tipo).Scan(&valor)
	return primeiro(valor, a.ia.modelo)
}

func primeiro(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}
