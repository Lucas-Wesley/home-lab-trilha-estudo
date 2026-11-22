package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Verificar material (T53): antes de estudar, a IA confere se o material da etapa ensina de verdade
// o tópico dela — o que a etapa exige, o que cada material cobre (coberto, raso ou ausente, com o
// minuto ou a seção), o quanto cobre (0–100) e o que falta, com a fonte primária onde achar.
//
// Não muda nada na etapa: é um laudo (como a conferência do resumo, T30), guardado em
// `verificacoes`. Vídeo sem transcrição entra antes na Entrada (YouTube pelo yt-dlp/Whisper, aula do
// Telegram pelo Streaming + Whisper): a verificação fica `aguardando` e vai para a fila do Claude
// sozinha quando os textos ficam prontos (`avancarVerificacoes`, chamada pela fila da Entrada e
// pelo GET). O que falta vira, na tela, um "Sugerir material" com o pedido já escrito.

const tipoVerificar = "verificar_material" // a tarefa da fila

const (
	verAguardando  = "aguardando"
	verVerificando = "verificando"
	verPronta      = "pronta"
	verFalhou      = "falhou"
)

// Como o texto de cada material chega à IA.
const (
	textoPronto        = "pronto"        // livro, Acervo, documento, vídeo já transcrito
	textoWeb           = "web"           // a IA abre a página
	textoTranscrevendo = "transcrevendo" // o vídeo está na Entrada, sendo lido
	textoSem           = "sem_texto"     // não dá para ler (o aviso diz por quê)
)

type materialVerificado struct {
	material
	Indice  int    `json:"indice"` // a posição na etapa quando foi verificado
	Texto   string `json:"texto"`
	FonteID int64  `json:"fonte_id,omitempty"`
	Aviso   string `json:"aviso,omitempty"`
}

type verificacao struct {
	ID           int64                `json:"id"`
	TrilhaID     int64                `json:"trilha_id"`
	EtapaChave   string               `json:"etapa_chave"`
	Estado       string               `json:"estado"`
	Pedido       string               `json:"pedido"`
	Materiais    []materialVerificado `json:"materiais"`
	Resultado    json.RawMessage      `json:"resultado,omitempty"`
	Erro         string               `json:"erro"`
	TarefaID     int64                `json:"tarefa_id"`
	Tarefa       *tarefa              `json:"tarefa,omitempty"`
	CriadoEm     string               `json:"criado_em"`
	AtualizadoEm string               `json:"atualizado_em"`
}

// laudo: o que a IA devolve (e o que fica em `resultado`).
type laudo struct {
	Topicos []struct {
		Topico     string `json:"topico"`
		Situacao   string `json:"situacao"`
		Onde       string `json:"onde"`
		Comentario string `json:"comentario"`
	} `json:"topicos"`
	Materiais []struct {
		Material     int    `json:"material"`
		Lido         bool   `json:"lido"`
		Cobertura    int    `json:"cobertura"`
		Profundidade string `json:"profundidade"`
		Comentario   string `json:"comentario"`
		Fora         string `json:"fora"`
	} `json:"materiais"`
	Cobertura int    `json:"cobertura"`
	Veredito  string `json:"veredito"`
	Faltas    []struct {
		Ponto     string `json:"ponto"`
		OndeAchar string `json:"onde_achar"`
	} `json:"faltas"`
	Resumo string `json:"resumo"`
}

const esquemaVerificacao = `{"type":"object","properties":{
"topicos":{"type":"array","description":"o que a etapa exige que se aprenda, um ponto por item, e se o material ensina","items":{"type":"object","properties":{
  "topico":{"type":"string"},
  "situacao":{"type":"string","enum":["coberto","raso","ausente"]},
  "onde":{"type":"string","description":"Material N e o minuto (mm:ss) ou a seção em que aparece; vazio se ausente"},
  "comentario":{"type":"string","description":"uma frase: o que o material faz com o ponto (ou o que falta nele)"}},
  "required":["topico","situacao","onde","comentario"]}},
"materiais":{"type":"array","items":{"type":"object","properties":{
  "material":{"type":"integer","description":"o número do material no pedido (1, 2…)"},
  "lido":{"type":"boolean","description":"false quando o texto não veio ou a página não abriu"},
  "cobertura":{"type":"integer","description":"0 a 100: quanto da etapa este material, sozinho, ensina na profundidade pedida"},
  "profundidade":{"type":"string","enum":["abaixo","na_medida","acima"]},
  "comentario":{"type":"string","description":"duas frases: o que ele ensina bem e o que deixa a desejar"},
  "fora":{"type":"string","description":"o que ele traz fora da etapa (dá para pular), ou vazio"}},
  "required":["material","lido","cobertura","profundidade","comentario","fora"]}},
"cobertura":{"type":"integer","description":"0 a 100: quanto da etapa os materiais, juntos, ensinam na profundidade pedida"},
"veredito":{"type":"string","enum":["suficiente","complementar","insuficiente"]},
"faltas":{"type":"array","items":{"type":"object","properties":{
  "ponto":{"type":"string","description":"o que falta aprender (ou está raso)"},
  "onde_achar":{"type":"string","description":"a fonte primária ou de referência que cobre o ponto: documentação oficial, RFC, paper, livro e capítulo"}},
  "required":["ponto","onde_achar"]}},
"resumo":{"type":"string","description":"duas ou três frases para o Lucas: dá para estudar só com isso? o que complementar primeiro?"}},
"required":["topicos","materiais","cobertura","veredito","faltas","resumo"]}`

const colunasVerificacao = `id, trilha_id, etapa_chave, estado, pedido, materiais, resultado, erro,
	COALESCE(tarefa_id, 0), criado_em, atualizado_em FROM verificacoes`

func lerVerificacao(row interface{ Scan(...any) error }) (verificacao, error) {
	var v verificacao
	var materiais, resultado string
	err := row.Scan(&v.ID, &v.TrilhaID, &v.EtapaChave, &v.Estado, &v.Pedido, &materiais, &resultado, &v.Erro,
		&v.TarefaID, &v.CriadoEm, &v.AtualizadoEm)
	if err == nil {
		v.Materiais = []materialVerificado{}
		json.Unmarshal([]byte(materiais), &v.Materiais)
		if resultado != "" {
			v.Resultado = json.RawMessage(resultado)
		}
	}
	return v, err
}

// handleVerificar: POST /api/etapas/{id}/verificar {"pedido":"…","materiais":[0,2]} — sem
// "materiais", verifica todos os da etapa.
func (a *app) handleVerificar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pedido    string `json:"pedido"`
		Materiais []int  `json:"materiais"`
	}
	if r.ContentLength != 0 {
		if err := lerCorpo(r, &corpo); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if len(e.Materiais) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa não tem material para verificar: adicione ou encaixe um antes")})
		return
	}
	indices := corpo.Materiais
	if len(indices) == 0 {
		for i := range e.Materiais {
			indices = append(indices, i)
		}
	}
	var ativa int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM verificacoes WHERE trilha_id = ? AND etapa_chave = ? AND estado IN (?, ?)`,
		e.TrilhaID, e.Chave, verAguardando, verVerificando).Scan(&ativa); err != nil {
		a.responderErro(w, err)
		return
	}
	if ativa > 0 {
		a.responderErro(w, erroDeEntrada{errors.New("já há uma verificação em andamento nesta etapa")})
		return
	}
	var dados *meusVideosStreaming
	vistos := map[int]bool{}
	var mats []materialVerificado
	for _, i := range indices {
		if i < 0 || i >= len(e.Materiais) {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("material %d não está na etapa", i+1)})
			return
		}
		if vistos[i] {
			continue
		}
		vistos[i] = true
		mats = append(mats, a.prepararMaterial(ctx, e.Materiais[i], i, &dados))
	}
	bruto, _ := json.Marshal(mats)
	momento := agora()
	res, err := a.db.ExecContext(ctx, `INSERT INTO verificacoes (trilha_id, etapa_chave, estado, pedido, materiais, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, e.TrilhaID, e.Chave, verAguardando, strings.TrimSpace(corpo.Pedido), string(bruto), momento, momento)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	id, _ := res.LastInsertId()
	a.acordarEntrada()
	a.avancarVerificacoes(ctx)
	v, err := a.verificacaoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, v)
}

// handleVerificacoes: GET /api/etapas/{id}/verificacoes — as 5 últimas, da mais nova para a mais
// velha (antes, promove as que esperavam a transcrição).
func (a *app) handleVerificacoes(w http.ResponseWriter, r *http.Request) {
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	a.avancarVerificacoes(ctx)
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasVerificacao+` WHERE trilha_id = ? AND etapa_chave = ? ORDER BY id DESC LIMIT 5`,
		e.TrilhaID, e.Chave)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	lista := []verificacao{}
	for rows.Next() {
		v, err := lerVerificacao(rows)
		if err != nil {
			rows.Close()
			a.responderErro(w, err)
			return
		}
		lista = append(lista, v)
	}
	rows.Close()
	for i := range lista {
		a.completarVerificacao(ctx, &lista[i])
	}
	responderJSON(w, http.StatusOK, lista)
}

func (a *app) verificacaoPorID(ctx context.Context, id int64) (verificacao, error) {
	v, err := lerVerificacao(a.db.QueryRowContext(ctx, `SELECT `+colunasVerificacao+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return v, errNaoEncontrou
	}
	if err == nil {
		a.completarVerificacao(ctx, &v)
	}
	return v, err
}

// completarVerificacao: a tarefa (para a tela dizer "na fila" ou "verificando").
func (a *app) completarVerificacao(ctx context.Context, v *verificacao) {
	if v.TarefaID == 0 || v.Estado != verVerificando {
		return
	}
	if t, err := lerTarefa(a.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, v.TarefaID)); err == nil {
		v.Tarefa = t
	}
}

// prepararMaterial: como o texto do material vai chegar à IA. Vídeo sem transcrição entra na
// Entrada agora (e, se a leitura tinha falhado, tenta de novo).
func (a *app) prepararMaterial(ctx context.Context, m material, i int, dados **meusVideosStreaming) materialVerificado {
	mv := materialVerificado{material: m, Indice: i, Texto: textoPronto}
	sem := func(aviso string) materialVerificado {
		mv.Texto, mv.Aviso = textoSem, aviso
		return mv
	}
	switch m.Tipo {
	case "livro":
		if a.integra.biblioteca == "" {
			return sem("a Biblioteca não está configurada")
		}
	case "acervo":
		if a.integra.acervo == "" {
			return sem("o Acervo não está configurado")
		}
	case "documento":
	case "web":
		mv.Texto = textoWeb
	case "youtube":
		id, err := a.fonteDoVideo(ctx, m.Referencia)
		if errors.Is(err, sql.ErrNoRows) {
			v, e := videoDoLink(m.Referencia)
			if e != nil {
				return sem("o link do YouTube não foi reconhecido")
			}
			f, e := a.criarFonte(ctx, "youtube", "https://www.youtube.com/watch?v="+v, v, "", "")
			if e != nil {
				return sem(e.Error())
			}
			id, err = f.ID, nil
		}
		if err != nil {
			return sem(err.Error())
		}
		mv.FonteID = id
		mv.Texto, mv.Aviso = a.textoDaFonte(ctx, id, true)
	case "streaming":
		if !strings.HasPrefix(m.Referencia, "telegram:") {
			return sem("vídeo do Streaming sem transcrição na Trilha: a transcrição dele, se houver, é um documento da etapa")
		}
		f, _, err := a.fonteDaAulaTelegram(ctx, m.Referencia, false, dados)
		if err != nil {
			return sem(err.Error())
		}
		mv.FonteID = f.ID
		mv.Texto, mv.Aviso = a.textoDaFonte(ctx, f.ID, true)
	default:
		return sem("tipo de material que a verificação não lê")
	}
	return mv
}

// textoDaFonte: a transcrição da fonte está pronta, a caminho ou falhou. tentar: a fonte em erro
// volta para a fila da Entrada (só quando o Lucas pede a verificação, não a cada sonda).
func (a *app) textoDaFonte(ctx context.Context, id int64, tentar bool) (string, string) {
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		return textoSem, "o vídeo saiu da Entrada"
	}
	switch f.Estado {
	case "pronta", "fragmentando", "fragmentada":
		if t, _ := a.conteudo.ler(pastaDaFonte(id) + "/transcricao.md"); strings.TrimSpace(t) == "" {
			return textoSem, "o vídeo não tem transcrição"
		}
		return textoPronto, ""
	case "erro":
		if tentar {
			if _, err := a.db.ExecContext(ctx, `UPDATE fontes SET estado = 'na_fila', erro = '' WHERE id = ? AND estado = 'erro'`, id); err == nil {
				return textoTranscrevendo, ""
			}
		}
		return textoSem, "a transcrição falhou: " + f.Erro
	default: // na_fila · lendo · baixando · transcrevendo
		return textoTranscrevendo, ""
	}
}

// avancarVerificacoes: as que esperavam a Entrada e já têm todos os textos (ou desistiram de algum)
// vão para a fila do Claude.
func (a *app) avancarVerificacoes(ctx context.Context) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasVerificacao+` WHERE estado = ?`, verAguardando)
	if err != nil {
		log.Printf("verificações: %v", err)
		return
	}
	var lista []verificacao
	for rows.Next() {
		if v, err := lerVerificacao(rows); err == nil {
			lista = append(lista, v)
		}
	}
	rows.Close()
	for _, v := range lista {
		pendente, mudou := false, false
		for i, m := range v.Materiais {
			if m.Texto != textoTranscrevendo {
				continue
			}
			texto, aviso := a.textoDaFonte(ctx, m.FonteID, false)
			if texto != m.Texto {
				v.Materiais[i].Texto, v.Materiais[i].Aviso, mudou = texto, aviso, true
			}
			if texto == textoTranscrevendo {
				pendente = true
			}
		}
		if mudou {
			bruto, _ := json.Marshal(v.Materiais)
			a.db.ExecContext(ctx, `UPDATE verificacoes SET materiais = ?, atualizado_em = ? WHERE id = ?`, string(bruto), agora(), v.ID)
		}
		if !pendente {
			if err := a.enfileirarVerificacao(ctx, v); err != nil {
				log.Printf("verificação %d: %v", v.ID, err)
			}
		}
	}
}

func (a *app) enfileirarVerificacao(ctx context.Context, v verificacao) error {
	titulo := "Verificar o material"
	if e, err := a.etapaPorChave(ctx, v.TrilhaID, v.EtapaChave); err == nil {
		titulo += ": " + e.Titulo
	}
	modelo := a.modeloPara(ctx, tipoVerificar) // antes da transação: o banco tem uma conexão só
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoVerificar, 0, titulo, modelo)
	if err != nil {
		return err
	}
	// Só uma promove (a fila da Entrada e o GET podem chegar juntas).
	res, err := tx.ExecContext(ctx, `UPDATE verificacoes SET estado = ?, tarefa_id = ?, atualizado_em = ? WHERE id = ? AND estado = ?`,
		verVerificando, tarefaID, agora(), v.ID, verAguardando)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	a.fila.acordar()
	return nil
}

// ---------- a tarefa ----------

func (a *app) verificarMaterial(ctx context.Context, t *tarefa) error {
	v, err := lerVerificacao(a.db.QueryRowContext(ctx, `SELECT `+colunasVerificacao+` WHERE tarefa_id = ?`, t.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("a verificação desta tarefa não existe mais")
	}
	if err != nil {
		return err
	}
	tr, err := a.trilhaCompleta(ctx, v.TrilhaID)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	arquivos, err := a.montarPedidoVerificacao(ctx, tr, v)
	if err != nil {
		return err
	}
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	var extras []string
	if caminho, err := a.conteudo.caminho("trilhas/" + tr.Slug + "/documentos"); err == nil {
		if _, err := os.Stat(caminho); err == nil {
			extras = append(extras, caminho)
		}
	}
	rg, err := a.ia.lerRegra("verificar-material.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, extras: extras, regra: rg.Texto, esquema: esquemaVerificacao, modelo: t.Modelo,
		ferramentas: "Read,Glob,Grep,WebFetch",
		instrucao: "Verifique se o material cobre a etapa. Tudo está em `pedido.md`, nesta pasta: leia o pedido e " +
			"cada material inteiro antes de julgar.",
	})
	if err != nil {
		return err
	}
	var l laudo
	if err := dadosDaResposta(resp, &l); err != nil {
		return err
	}
	limparLaudo(&l, len(v.Materiais))
	bruto, _ := json.Marshal(l)
	_, err = a.db.ExecContext(ctx, `UPDATE verificacoes SET estado = ?, resultado = ?, erro = '', atualizado_em = ? WHERE id = ?`,
		verPronta, string(bruto), agora(), v.ID)
	return err
}

// limparLaudo: números no intervalo, enums conhecidos, material que existe, listas nunca nulas.
func limparLaudo(l *laudo, total int) {
	l.Cobertura = min(100, max(0, l.Cobertura))
	if !contem([]string{"suficiente", "complementar", "insuficiente"}, l.Veredito) {
		l.Veredito = "complementar"
	}
	l.Resumo = textoDaIA(l.Resumo)
	topicos := l.Topicos[:0]
	for _, x := range l.Topicos {
		x.Topico = textoDaIA(x.Topico)
		if x.Topico == "" {
			continue
		}
		if !contem([]string{"coberto", "raso", "ausente"}, x.Situacao) {
			x.Situacao = "raso"
		}
		x.Comentario, x.Onde = textoDaIA(x.Comentario), strings.TrimSpace(x.Onde)
		topicos = append(topicos, x)
	}
	l.Topicos = topicos
	vistos := map[int]bool{}
	materiais := l.Materiais[:0]
	for _, x := range l.Materiais {
		if x.Material < 1 || x.Material > total || vistos[x.Material] {
			continue
		}
		vistos[x.Material] = true
		x.Cobertura = min(100, max(0, x.Cobertura))
		if !contem([]string{"abaixo", "na_medida", "acima"}, x.Profundidade) {
			x.Profundidade = "na_medida"
		}
		x.Comentario, x.Fora = textoDaIA(x.Comentario), textoDaIA(x.Fora)
		materiais = append(materiais, x)
	}
	l.Materiais = materiais
	faltas := l.Faltas[:0]
	for _, x := range l.Faltas {
		x.Ponto, x.OndeAchar = textoDaIA(x.Ponto), textoDaIA(x.OndeAchar)
		if x.Ponto != "" {
			faltas = append(faltas, x)
		}
	}
	l.Faltas = faltas
}

func (a *app) montarPedidoVerificacao(ctx context.Context, tr trilha, v verificacao) (map[string]string, error) {
	var e etapa
	var mod modulo
	for _, m := range tr.Modulos {
		for _, x := range m.Etapas {
			if x.Chave == v.EtapaChave {
				e, mod = x, m
			}
		}
	}
	if e.ID == 0 {
		return nil, errors.New("a etapa desta verificação não existe mais na trilha")
	}
	cfg, err := a.lerAjustes(ctx)
	if err != nil {
		return nil, err
	}
	profundidade := e.Profundidade
	if profundidade == 0 {
		profundidade = tr.Profundidade
	}
	var s strings.Builder
	escreva := func(format string, args ...any) { fmt.Fprintf(&s, format, args...) }
	escreva("# Pedido: verificar o material da etapa \"%s\"\n\n", e.Titulo)
	escreva("## A trilha\n\n- Título: %s\n", tr.Titulo)
	for _, campo := range [][2]string{{"Objetivo", tr.Objetivo}, {"Recorte", tr.Recorte}, {"Nível de partida", tr.NivelPartida}, {"Nível alvo", tr.NivelAlvo}} {
		if strings.TrimSpace(campo[1]) != "" {
			escreva("- %s: %s\n", campo[0], campo[1])
		}
	}
	escreva("\n## A etapa (é contra ela que o material é medido)\n\n- Título: %s\n- Módulo: %s\n", e.Titulo, mod.Titulo)
	if e.Objetivo != "" {
		escreva("- Objetivo: %s\n", e.Objetivo)
	}
	if e.Bloom != "" {
		escreva("- Nível de Bloom do objetivo: %s\n", e.Bloom)
	}
	escreva("- Profundidade: %s\n", nomesProfundidade[profundidade])
	if e.Criterio != "" {
		escreva("- Critério de domínio: %s\n", e.Criterio)
	}
	if e.Foco != "" {
		escreva("- Foco: %s\n", e.Foco)
	}
	if e.Minutos > 0 {
		escreva("- Tempo previsto: %d min\n", e.Minutos)
	}
	escreva("\n## As etapas do módulo (o que é de outra etapa não é falta desta)\n\n")
	for _, x := range mod.Etapas {
		marca := ""
		if x.Chave == e.Chave {
			marca = " ← esta"
		}
		escreva("- %s%s\n", x.Titulo, marca)
	}

	arquivos := map[string]string{}
	var legiveis []material
	for _, m := range v.Materiais {
		if m.Texto == textoPronto {
			legiveis = append(legiveis, m.material)
		} else {
			legiveis = append(legiveis, material{Tipo: "-"}) // fora da leitura, mantém o número
		}
	}
	lidos := a.arquivosDoMaterial(ctx, legiveis, arquivos)
	escreva("\n## O material a verificar\n\n")
	algum := false
	for i, m := range v.Materiais {
		escreva("### Material %d · %s (%s, papel %s)\n\n", i+1, m.Titulo, m.Tipo, m.Papel)
		if m.Nota != "" {
			escreva("- Nota do Lucas: %s\n", m.Nota)
		}
		switch {
		case m.Texto == textoWeb:
			algum = true
			escreva("- Página web: %s — abra com WebFetch e leia inteira. Se não abrir, `lido: false`.\n", m.Referencia)
		case m.Texto == textoPronto && m.Tipo == "documento":
			algum = true
			escreva("- Arquivo `documentos/%s.md`", m.Referencia)
			if m.Localizador != "" {
				escreva(", a seção com âncora `#%s`", m.Localizador)
			}
			escreva(" (pasta adicionada ao seu acesso; procure com Glob se o caminho relativo não abrir).\n")
		case m.Texto == textoPronto && lidos[i].Arquivo != "":
			algum = true
			if m.Localizador != "" {
				escreva("- Trecho: %s\n", m.Localizador)
			}
			if m.Tipo == "youtube" || m.Tipo == "streaming" {
				escreva("- Transcrição automática do vídeo, com o minuto entre colchetes: `%s`\n", lidos[i].Arquivo)
			} else {
				escreva("- Texto: `%s`\n", lidos[i].Arquivo)
			}
		default:
			motivo := m.Aviso
			if lidos[i].Erro != "" {
				motivo = lidos[i].Erro
			}
			escreva("- **O texto não veio** (%s): `lido: false` e não invente o conteúdo.\n", padrao(motivo, "motivo desconhecido"))
		}
		escreva("\n")
	}
	if !algum {
		return nil, errors.New("nenhum material pôde ser lido" + entreParenteses(avisosDaVerificacao(v)))
	}
	escreva("## Instruções do Lucas\n\nVão da mais permanente à mais pontual; em conflito, vale a mais específica.\n\n")
	escreva("- Globais: %s\n", padrao(cfg.InstrucoesGlobais, "(nenhuma)"))
	escreva("- Da trilha: %s\n", padrao(tr.Instrucoes, "(nenhuma)"))
	escreva("- Deste pedido: %s\n", padrao(v.Pedido, "(nenhuma)"))
	arquivos["pedido.md"] = s.String()
	return arquivos, nil
}

func avisosDaVerificacao(v verificacao) string {
	var xs []string
	for _, m := range v.Materiais {
		if m.Aviso != "" {
			xs = append(xs, m.Titulo+": "+m.Aviso)
		}
	}
	return strings.Join(xs, "; ")
}
