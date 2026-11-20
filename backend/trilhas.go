package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type meta struct {
	Dias    []int  `json:"dias"`    // 0 = domingo … 6 = sábado
	Minutos int    `json:"minutos"` // por dia de estudo; 0 = calcular pela data final
	Inicio  string `json:"inicio"`  // AAAA-MM-DD
	Fim     string `json:"fim"`     // AAAA-MM-DD, opcional
}

type progresso struct {
	Total         int `json:"total"`
	Concluidas    int `json:"concluidas"`
	Puladas       int `json:"puladas"`
	EmAndamento   int `json:"em_andamento"`
	MinutosTotal  int `json:"minutos_total"`
	MinutosFeitos int `json:"minutos_feitos"`
}

type trilha struct {
	ID           int64     `json:"id"`
	Slug         string    `json:"slug"`
	Titulo       string    `json:"titulo"`
	Objetivo     string    `json:"objetivo"`
	Descricao    string    `json:"descricao"`
	Recorte      string    `json:"recorte"`
	Instrucoes   string    `json:"instrucoes"`
	NivelPartida string    `json:"nivel_partida"`
	NivelAlvo    string    `json:"nivel_alvo"`
	Profundidade int       `json:"profundidade"`
	Estado       string    `json:"estado"`
	Meta         meta      `json:"meta"`
	CriadoEm     string    `json:"criado_em"`
	AtualizadoEm string    `json:"atualizado_em"`
	Progresso    progresso `json:"progresso"`
	Modulos      []modulo  `json:"modulos,omitempty"`
	// Documentos da trilha (só nome e título), no detalhe.
	Documentos []documento `json:"documentos,omitempty"`
}

type modulo struct {
	ID        int64   `json:"id"`
	Ordem     int     `json:"ordem"`
	Titulo    string  `json:"titulo"`
	Descricao string  `json:"descricao"`
	Etapas    []etapa `json:"etapas"`
}

type material struct {
	Tipo        string `json:"tipo"`
	Titulo      string `json:"titulo"`
	Referencia  string `json:"referencia"`
	Localizador string `json:"localizador"`
	Papel       string `json:"papel"`
	Nota        string `json:"nota"`
}

type etapa struct {
	ID           int64      `json:"id"`
	TrilhaID     int64      `json:"trilha_id"`
	ModuloID     int64      `json:"modulo_id"`
	Ordem        int        `json:"ordem"`
	Chave        string     `json:"chave"`
	Titulo       string     `json:"titulo"`
	Objetivo     string     `json:"objetivo"`
	Bloom        string     `json:"bloom"`
	Profundidade int        `json:"profundidade"`
	Tipo         string     `json:"tipo"`
	Minutos      int        `json:"minutos"`
	Criterio     string     `json:"criterio"`
	Foco         string     `json:"foco"` // o que a lição deve trabalhar (a etapa de reforço nasce com ele, T28)
	Status       string     `json:"status"`
	IniciadoEm   string     `json:"iniciado_em"`
	ConcluidoEm  string     `json:"concluido_em"`
	Requer       []int64    `json:"requer"`
	Materiais    []material `json:"materiais"`
	TemLicao     bool       `json:"tem_licao"`
	Licao        *string    `json:"licao,omitempty"`
	// T39: os intervalos principais do YouTube já vistos até o fim (só no GET da etapa).
	TrechosVistos []trechoVisto `json:"trechos_vistos,omitempty"`
}

var (
	niveisBloom     = []string{"lembrar", "entender", "aplicar", "analisar", "avaliar", "criar"}
	statusEtapa     = []string{"disponivel", "em_andamento", "concluida", "pulada"}
	estadosTrilha   = []string{"rascunho", "ativa", "pausada", "concluida"}
	tiposMaterial   = []string{"documento", "web", "livro", "acervo", "streaming", "youtube"}
	papeisMaterial  = []string{"principal", "apoio", "aprofundar"}
	tiposEtapa      = []string{"normal", "base", "reforco", "projeto"}
	errNaoEncontrou = errors.New("não encontrado")
)

func contem(lista []string, v string) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}

// slugificar: "Home lab: rede" → "home-lab-rede".
func slugificar(texto string) string {
	var b strings.Builder
	hifen := false
	for _, r := range norm.NFD.String(strings.ToLower(texto)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			// acento separado pela NFD: some
		case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			hifen = false
		case !hifen && b.Len() > 0:
			b.WriteByte('-')
			hifen = true
		}
	}
	s := strings.TrimSuffix(b.String(), "-")
	if len(s) > 60 {
		s = strings.TrimSuffix(s[:60], "-")
	}
	if s == "" {
		s = "sem-titulo"
	}
	return s
}

// ---------- leitura ----------

const colunasTrilha = `id, slug, titulo, objetivo, descricao, recorte, instrucoes, nivel_partida, nivel_alvo,
	profundidade, estado, meta_dias, meta_minutos, meta_inicio, meta_fim, criado_em, atualizado_em`

func lerTrilhaLinha(row interface{ Scan(...any) error }) (trilha, error) {
	var t trilha
	var dias string
	err := row.Scan(&t.ID, &t.Slug, &t.Titulo, &t.Objetivo, &t.Descricao, &t.Recorte, &t.Instrucoes,
		&t.NivelPartida, &t.NivelAlvo, &t.Profundidade, &t.Estado, &dias, &t.Meta.Minutos,
		&t.Meta.Inicio, &t.Meta.Fim, &t.CriadoEm, &t.AtualizadoEm)
	t.Meta.Dias = lerDias(dias)
	return t, err
}

func lerDias(s string) []int {
	dias := []int{}
	for _, p := range strings.Split(s, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n >= 0 && n <= 6 {
			dias = append(dias, n)
		}
	}
	return dias
}

func escreverDias(dias []int) string {
	partes := []string{}
	visto := map[int]bool{}
	for d := 0; d <= 6; d++ {
		for _, x := range dias {
			if x == d && !visto[d] {
				partes = append(partes, strconv.Itoa(d))
				visto[d] = true
			}
		}
	}
	return strings.Join(partes, ",")
}

func (a *app) progressoDe(ctx context.Context, trilhaID int64) (progresso, error) {
	var p progresso
	err := a.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(SUM(status = 'concluida'), 0), COALESCE(SUM(status = 'pulada'), 0),
		COALESCE(SUM(status = 'em_andamento'), 0), COALESCE(SUM(minutos), 0),
		COALESCE(SUM(CASE WHEN status IN ('concluida', 'pulada') THEN minutos ELSE 0 END), 0)
		FROM etapas WHERE trilha_id = ?`, trilhaID).
		Scan(&p.Total, &p.Concluidas, &p.Puladas, &p.EmAndamento, &p.MinutosTotal, &p.MinutosFeitos)
	return p, err
}

func (a *app) listarTrilhas(ctx context.Context) ([]trilha, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasTrilha+` FROM trilhas ORDER BY
		CASE estado WHEN 'ativa' THEN 0 WHEN 'rascunho' THEN 1 WHEN 'pausada' THEN 2 ELSE 3 END, atualizado_em DESC`)
	if err != nil {
		return nil, err
	}
	lista := []trilha{}
	for rows.Next() {
		t, err := lerTrilhaLinha(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		lista = append(lista, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range lista {
		if lista[i].Progresso, err = a.progressoDe(ctx, lista[i].ID); err != nil {
			return nil, err
		}
	}
	return lista, nil
}

// trilhaCompleta devolve a trilha com módulos, etapas, requisitos e materiais (sem o texto das lições).
func (a *app) trilhaCompleta(ctx context.Context, id int64) (trilha, error) {
	t, err := lerTrilhaLinha(a.db.QueryRowContext(ctx, `SELECT `+colunasTrilha+` FROM trilhas WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, errNaoEncontrou
	}
	if err != nil {
		return t, err
	}
	if t.Progresso, err = a.progressoDe(ctx, id); err != nil {
		return t, err
	}

	rows, err := a.db.QueryContext(ctx, `SELECT id, ordem, titulo, descricao FROM modulos WHERE trilha_id = ? ORDER BY ordem, id`, id)
	if err != nil {
		return t, err
	}
	t.Modulos = []modulo{}
	indice := map[int64]int{}
	for rows.Next() {
		var m modulo
		if err := rows.Scan(&m.ID, &m.Ordem, &m.Titulo, &m.Descricao); err != nil {
			rows.Close()
			return t, err
		}
		m.Etapas = []etapa{}
		indice[m.ID] = len(t.Modulos)
		t.Modulos = append(t.Modulos, m)
	}
	rows.Close()

	etapas, err := a.etapasDaTrilha(ctx, id)
	if err != nil {
		return t, err
	}
	for _, e := range etapas {
		if i, ok := indice[e.ModuloID]; ok {
			t.Modulos[i].Etapas = append(t.Modulos[i].Etapas, e)
		}
	}
	t.Documentos, err = a.conteudo.documentos("trilhas/" + t.Slug + "/documentos")
	return t, err
}

const colunasEtapa = `e.id, e.trilha_id, e.modulo_id, e.ordem, e.chave, e.titulo, e.objetivo, e.bloom,
	e.profundidade, e.tipo, e.minutos, e.criterio, e.status, e.iniciado_em, e.concluido_em, e.foco`

func lerEtapaLinha(row interface{ Scan(...any) error }) (etapa, error) {
	var e etapa
	err := row.Scan(&e.ID, &e.TrilhaID, &e.ModuloID, &e.Ordem, &e.Chave, &e.Titulo, &e.Objetivo, &e.Bloom,
		&e.Profundidade, &e.Tipo, &e.Minutos, &e.Criterio, &e.Status, &e.IniciadoEm, &e.ConcluidoEm, &e.Foco)
	e.Requer, e.Materiais = []int64{}, []material{}
	return e, err
}

// etapasDaTrilha devolve as etapas na ordem de estudo (módulo, depois etapa), com requisitos e materiais.
func (a *app) etapasDaTrilha(ctx context.Context, trilhaID int64) ([]etapa, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasEtapa+` FROM etapas e JOIN modulos m ON m.id = e.modulo_id
		WHERE e.trilha_id = ? ORDER BY m.ordem, m.id, e.ordem, e.id`, trilhaID)
	if err != nil {
		return nil, err
	}
	lista := []etapa{}
	indice := map[int64]int{}
	for rows.Next() {
		e, err := lerEtapaLinha(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		indice[e.ID] = len(lista)
		lista = append(lista, e)
	}
	rows.Close()

	rows, err = a.db.QueryContext(ctx, `SELECT r.etapa_id, r.requer_id FROM etapa_requisitos r
		JOIN etapas e ON e.id = r.etapa_id WHERE e.trilha_id = ? ORDER BY r.etapa_id, r.requer_id`, trilhaID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var de, requer int64
		if err := rows.Scan(&de, &requer); err != nil {
			rows.Close()
			return nil, err
		}
		if i, ok := indice[de]; ok {
			lista[i].Requer = append(lista[i].Requer, requer)
		}
	}
	rows.Close()

	rows, err = a.db.QueryContext(ctx, `SELECT x.etapa_id, x.tipo, x.titulo, x.referencia, x.localizador, x.papel, x.nota
		FROM materiais x JOIN etapas e ON e.id = x.etapa_id WHERE e.trilha_id = ? ORDER BY x.etapa_id, x.ordem, x.id`, trilhaID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var de int64
		var m material
		if err := rows.Scan(&de, &m.Tipo, &m.Titulo, &m.Referencia, &m.Localizador, &m.Papel, &m.Nota); err != nil {
			rows.Close()
			return nil, err
		}
		if i, ok := indice[de]; ok {
			lista[i].Materiais = append(lista[i].Materiais, m)
		}
	}
	rows.Close()

	var slug string
	if err := a.db.QueryRowContext(ctx, `SELECT slug FROM trilhas WHERE id = ?`, trilhaID).Scan(&slug); err != nil {
		return nil, err
	}
	for i := range lista {
		_, err := a.conteudo.ler(caminhoLicao(slug, lista[i].Chave))
		lista[i].TemLicao = err == nil
	}
	return lista, rows.Err()
}

func caminhoLicao(slugTrilha, chave string) string {
	return "trilhas/" + slugTrilha + "/licoes/" + chave + ".md"
}

// ---------- exportação para o git do conteúdo ----------

// exportar grava trilha.json (a estrutura, sem o progresso) e os arquivos extras num só commit.
func (a *app) exportar(ctx context.Context, trilhaID int64, mensagem string, extras map[string]*string) error {
	t, err := a.trilhaCompleta(ctx, trilhaID)
	if err != nil {
		return err
	}
	type etapaExportada struct {
		Chave        string     `json:"chave"`
		Titulo       string     `json:"titulo"`
		Objetivo     string     `json:"objetivo,omitempty"`
		Bloom        string     `json:"bloom,omitempty"`
		Profundidade int        `json:"profundidade,omitempty"`
		Tipo         string     `json:"tipo"`
		Minutos      int        `json:"minutos"`
		Criterio     string     `json:"criterio,omitempty"`
		Foco         string     `json:"foco,omitempty"`
		Requer       []string   `json:"requer,omitempty"`
		Materiais    []material `json:"materiais,omitempty"`
	}
	type moduloExportado struct {
		Titulo    string           `json:"titulo"`
		Descricao string           `json:"descricao,omitempty"`
		Etapas    []etapaExportada `json:"etapas"`
	}
	chaves := map[int64]string{}
	for _, m := range t.Modulos {
		for _, e := range m.Etapas {
			chaves[e.ID] = e.Chave
		}
	}
	mods := []moduloExportado{}
	for _, m := range t.Modulos {
		me := moduloExportado{Titulo: m.Titulo, Descricao: m.Descricao, Etapas: []etapaExportada{}}
		for _, e := range m.Etapas {
			req := []string{}
			for _, r := range e.Requer {
				req = append(req, chaves[r])
			}
			me.Etapas = append(me.Etapas, etapaExportada{e.Chave, e.Titulo, e.Objetivo, e.Bloom, e.Profundidade,
				e.Tipo, e.Minutos, e.Criterio, e.Foco, req, e.Materiais})
		}
		mods = append(mods, me)
	}
	b, err := json.MarshalIndent(map[string]any{
		"slug": t.Slug, "titulo": t.Titulo, "objetivo": t.Objetivo, "descricao": t.Descricao,
		"recorte": t.Recorte, "instrucoes": t.Instrucoes, "nivel_partida": t.NivelPartida,
		"nivel_alvo": t.NivelAlvo, "profundidade": t.Profundidade, "modulos": mods,
	}, "", "  ")
	if err != nil {
		return err
	}
	texto := string(b) + "\n"
	arquivos := map[string]*string{"trilhas/" + t.Slug + "/trilha.json": &texto}
	for k, v := range extras {
		arquivos[k] = v
	}
	return a.conteudo.gravar(fmt.Sprintf("%s: %s", t.Slug, mensagem), arquivos)
}

// ---------- validação ----------

func validarTrilha(t *trilha) error {
	t.Titulo = strings.TrimSpace(t.Titulo)
	if t.Titulo == "" {
		return errors.New("a trilha precisa de um título")
	}
	if t.Profundidade == 0 {
		t.Profundidade = 3
	}
	if t.Profundidade < 1 || t.Profundidade > 5 {
		return errors.New("profundidade vai de 1 (panorama) a 5 (deep dive)")
	}
	return nil
}

func validarEtapa(e *etapa) error {
	e.Titulo = strings.TrimSpace(e.Titulo)
	if e.Titulo == "" {
		return errors.New("a etapa precisa de um título")
	}
	if e.Bloom != "" && !contem(niveisBloom, e.Bloom) {
		return fmt.Errorf("nível de Bloom desconhecido: %q", e.Bloom)
	}
	if e.Profundidade < 0 || e.Profundidade > 5 {
		return errors.New("profundidade da etapa vai de 0 (a da trilha) a 5")
	}
	if e.Tipo == "" {
		e.Tipo = "normal"
	}
	if !contem(tiposEtapa, e.Tipo) {
		return fmt.Errorf("tipo de etapa desconhecido: %q", e.Tipo)
	}
	if e.Minutos < 0 || e.Minutos > 24*60 {
		return errors.New("minutos da etapa fora do intervalo")
	}
	for i := range e.Materiais {
		m := &e.Materiais[i]
		m.Titulo = strings.TrimSpace(m.Titulo)
		if m.Titulo == "" {
			return errors.New("todo material precisa de um título")
		}
		if !contem(tiposMaterial, m.Tipo) {
			return fmt.Errorf("tipo de material desconhecido: %q", m.Tipo)
		}
		if m.Papel == "" {
			m.Papel = "apoio"
		}
		if !contem(papeisMaterial, m.Papel) {
			return fmt.Errorf("papel de material desconhecido: %q", m.Papel)
		}
		if m.Tipo == "web" && !strings.HasPrefix(m.Referencia, "https://") && !strings.HasPrefix(m.Referencia, "http://") {
			return fmt.Errorf("o material %q precisa de um endereço http(s)", m.Titulo)
		}
	}
	return nil
}

// slugLivre devolve o slug, ou o slug com -2, -3… se já existir na tabela (para trilhas) ou na trilha (para etapas).
func slugLivre(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, base, consulta string, args ...any) (string, error) {
	for n := 1; ; n++ {
		s := base
		if n > 1 {
			s = fmt.Sprintf("%s-%d", base, n)
		}
		var existe int
		if err := q.QueryRowContext(ctx, consulta, append(args, s)...).Scan(&existe); err != nil {
			return "", err
		}
		if existe == 0 {
			return s, nil
		}
	}
}

// ---------- escrita ----------

func (a *app) criarTrilha(ctx context.Context, t trilha) (int64, error) {
	if err := validarTrilha(&t); err != nil {
		return 0, erroDeEntrada{err}
	}
	base := t.Slug
	if base == "" {
		base = slugificar(t.Titulo)
	} else if !slugValido.MatchString(base) {
		return 0, erroDeEntrada{fmt.Errorf("slug inválido: %q", base)}
	}
	slug, err := slugLivre(ctx, a.db, base, `SELECT COUNT(*) FROM trilhas WHERE slug = ?`)
	if err != nil {
		return 0, err
	}
	agora := agora()
	res, err := a.db.ExecContext(ctx, `INSERT INTO trilhas (slug, titulo, objetivo, descricao, recorte, instrucoes,
		nivel_partida, nivel_alvo, profundidade, estado, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'rascunho', ?, ?)`,
		slug, t.Titulo, t.Objetivo, t.Descricao, t.Recorte, t.Instrucoes, t.NivelPartida, t.NivelAlvo, t.Profundidade, agora, agora)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, a.exportar(ctx, id, "criada", nil)
}

// erroDeEntrada vira 400 na resposta; os outros erros, 500.
type erroDeEntrada struct{ error }

func (a *app) tocarTrilha(ctx context.Context, id int64) {
	a.db.ExecContext(ctx, `UPDATE trilhas SET atualizado_em = ? WHERE id = ?`, agora(), id)
}

// salvarEtapa grava os campos, requisitos e materiais de uma etapa já existente.
func (a *app) salvarEtapa(ctx context.Context, tx *sql.Tx, e etapa) error {
	if _, err := tx.ExecContext(ctx, `UPDATE etapas SET titulo = ?, objetivo = ?, bloom = ?, profundidade = ?, tipo = ?,
		minutos = ?, criterio = ?, foco = ? WHERE id = ?`,
		e.Titulo, e.Objetivo, e.Bloom, e.Profundidade, e.Tipo, e.Minutos, e.Criterio, strings.TrimSpace(e.Foco), e.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM etapa_requisitos WHERE etapa_id = ?`, e.ID); err != nil {
		return err
	}
	for _, r := range e.Requer {
		if r == e.ID {
			return erroDeEntrada{errors.New("uma etapa não pode ser pré-requisito dela mesma")}
		}
		var mesma int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM etapas WHERE id = ? AND trilha_id = ?`, r, e.TrilhaID).Scan(&mesma); err != nil {
			return err
		}
		if mesma == 0 {
			return erroDeEntrada{fmt.Errorf("pré-requisito %d não é desta trilha", r)}
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO etapa_requisitos (etapa_id, requer_id) VALUES (?, ?)`, e.ID, r); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM materiais WHERE etapa_id = ?`, e.ID); err != nil {
		return err
	}
	for i, m := range e.Materiais {
		if _, err := tx.ExecContext(ctx, `INSERT INTO materiais (etapa_id, ordem, tipo, titulo, referencia, localizador, papel, nota)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, e.ID, i, m.Tipo, m.Titulo, m.Referencia, m.Localizador, m.Papel, m.Nota); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) criarEtapa(ctx context.Context, moduloID int64, e etapa, licao *string) (int64, error) {
	if err := validarEtapa(&e); err != nil {
		return 0, erroDeEntrada{err}
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT trilha_id FROM modulos WHERE id = ?`, moduloID).Scan(&e.TrilhaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errNaoEncontrou
		}
		return 0, err
	}
	base := e.Chave
	if base == "" {
		base = slugificar(e.Titulo)
	} else if !slugValido.MatchString(base) {
		return 0, erroDeEntrada{fmt.Errorf("chave inválida: %q", base)}
	}
	chave, err := slugLivre(ctx, tx, base, `SELECT COUNT(*) FROM etapas WHERE trilha_id = ? AND chave = ?`, e.TrilhaID)
	if err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo)
		VALUES (?, ?, (SELECT COALESCE(MAX(ordem), -1) + 1 FROM etapas WHERE modulo_id = ?), ?, ?)`,
		e.TrilhaID, moduloID, moduloID, chave, e.Titulo)
	if err != nil {
		return 0, err
	}
	if e.ID, err = res.LastInsertId(); err != nil {
		return 0, err
	}
	if err := a.salvarEtapa(ctx, tx, e); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	a.tocarTrilha(ctx, e.TrilhaID)
	extras := map[string]*string{}
	if licao != nil {
		slug := a.slugDaTrilha(ctx, e.TrilhaID)
		extras[caminhoLicao(slug, chave)] = licao
	}
	return e.ID, a.exportar(ctx, e.TrilhaID, "etapa "+chave+" criada", extras)
}

func (a *app) slugDaTrilha(ctx context.Context, id int64) string {
	var slug string
	a.db.QueryRowContext(ctx, `SELECT slug FROM trilhas WHERE id = ?`, id).Scan(&slug)
	return slug
}

func (a *app) etapaPorID(ctx context.Context, id int64) (etapa, error) {
	e, err := lerEtapaLinha(a.db.QueryRowContext(ctx, `SELECT `+colunasEtapa+` FROM etapas e WHERE e.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return e, errNaoEncontrou
	}
	if err != nil {
		return e, err
	}
	todas, err := a.etapasDaTrilha(ctx, e.TrilhaID)
	if err != nil {
		return e, err
	}
	for _, x := range todas {
		if x.ID == id {
			return x, nil
		}
	}
	return e, errNaoEncontrou
}

// ---------- handlers ----------

func (a *app) responderErro(w http.ResponseWriter, err error) {
	var entrada erroDeEntrada
	switch {
	case errors.Is(err, errNaoEncontrou):
		erroJSON(w, http.StatusNotFound, "não encontrado")
	case errors.As(err, &entrada):
		erroJSON(w, http.StatusBadRequest, entrada.Error())
	default:
		erroInterno(w, err)
	}
}

func lerCorpo(r *http.Request, destino any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 20<<20))
	if err := dec.Decode(destino); err != nil {
		return erroDeEntrada{fmt.Errorf("corpo inválido: %w", err)}
	}
	return nil
}

func idDaRota(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errNaoEncontrou
	}
	return id, nil
}

func (a *app) handleListarTrilhas(w http.ResponseWriter, r *http.Request) {
	lista, err := a.listarTrilhas(r.Context())
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, lista)
}

func (a *app) handleCriarTrilha(w http.ResponseWriter, r *http.Request) {
	var t trilha
	if err := lerCorpo(r, &t); err != nil {
		a.responderErro(w, err)
		return
	}
	id, err := a.criarTrilha(r.Context(), t)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	nova, err := a.trilhaCompleta(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, nova)
}

func (a *app) handleTrilha(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	t, err := a.trilhaCompleta(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, t)
}

func (a *app) handleEditarTrilha(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var t trilha
	if err := lerCorpo(r, &t); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := validarTrilha(&t); err != nil {
		a.responderErro(w, erroDeEntrada{err})
		return
	}
	res, err := a.db.ExecContext(r.Context(), `UPDATE trilhas SET titulo = ?, objetivo = ?, descricao = ?, recorte = ?,
		instrucoes = ?, nivel_partida = ?, nivel_alvo = ?, profundidade = ?, atualizado_em = ? WHERE id = ?`,
		t.Titulo, t.Objetivo, t.Descricao, t.Recorte, t.Instrucoes, t.NivelPartida, t.NivelAlvo, t.Profundidade, agora(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if err := a.exportar(r.Context(), id, "dados da trilha", nil); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleTrilha(w, r)
}

func (a *app) handleExcluirTrilha(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	slug := a.slugDaTrilha(r.Context(), id)
	if slug == "" {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM trilhas WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.conteudo.apagarPasta(slug+": trilha excluída", "trilhas/"+slug); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleEstadoTrilha(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Estado string `json:"estado"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if !contem(estadosTrilha, corpo.Estado) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("estado desconhecido: %q", corpo.Estado)})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE trilhas SET estado = ?, atualizado_em = ? WHERE id = ?`, corpo.Estado, agora(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleTrilha(w, r)
}

func (a *app) handleCriarModulo(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var m modulo
	if err := lerCorpo(r, &m); err != nil {
		a.responderErro(w, err)
		return
	}
	if m.Titulo = strings.TrimSpace(m.Titulo); m.Titulo == "" {
		a.responderErro(w, erroDeEntrada{errors.New("o módulo precisa de um título")})
		return
	}
	if a.slugDaTrilha(r.Context(), id) == "" {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `INSERT INTO modulos (trilha_id, ordem, titulo, descricao)
		VALUES (?, (SELECT COALESCE(MAX(ordem), -1) + 1 FROM modulos WHERE trilha_id = ?), ?, ?)`,
		id, id, m.Titulo, m.Descricao); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), id)
	if err := a.exportar(r.Context(), id, "módulo "+m.Titulo, nil); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleTrilha(w, r)
}

func (a *app) trilhaDoModulo(ctx context.Context, moduloID int64) (int64, error) {
	var t int64
	err := a.db.QueryRowContext(ctx, `SELECT trilha_id FROM modulos WHERE id = ?`, moduloID).Scan(&t)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNaoEncontrou
	}
	return t, err
}

func (a *app) handleEditarModulo(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var m modulo
	if err := lerCorpo(r, &m); err != nil {
		a.responderErro(w, err)
		return
	}
	if m.Titulo = strings.TrimSpace(m.Titulo); m.Titulo == "" {
		a.responderErro(w, erroDeEntrada{errors.New("o módulo precisa de um título")})
		return
	}
	trilhaID, err := a.trilhaDoModulo(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE modulos SET titulo = ?, descricao = ? WHERE id = ?`, m.Titulo, m.Descricao, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), trilhaID)
	if err := a.exportar(r.Context(), trilhaID, "módulo "+m.Titulo, nil); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleExcluirModulo(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	trilhaID, err := a.trilhaDoModulo(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var n int
	a.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM etapas WHERE modulo_id = ?`, id).Scan(&n)
	if n > 0 {
		a.responderErro(w, erroDeEntrada{errors.New("mova ou exclua as etapas do módulo antes")})
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM modulos WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), trilhaID)
	if err := a.exportar(r.Context(), trilhaID, "módulo excluído", nil); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// corpoEtapa: a lição vem como ponteiro para distinguir "não mexer" (ausente) de "apagar" ("").
type corpoEtapa struct {
	etapa
	Licao *string `json:"licao"`
}

func (a *app) handleCriarEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var c corpoEtapa
	if err := lerCorpo(r, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	novo, err := a.criarEtapa(r.Context(), id, c.etapa, licaoOuNada(c.Licao))
	if err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.etapaPorID(r.Context(), novo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusCreated, e)
}

// licaoOuNada: lição vazia não cria arquivo.
func licaoOuNada(l *string) *string {
	if l == nil || strings.TrimSpace(*l) == "" {
		return nil
	}
	return l
}

func (a *app) handleEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if e.TemLicao {
		texto, err := a.conteudo.ler(caminhoLicao(a.slugDaTrilha(r.Context(), e.TrilhaID), e.Chave))
		if err == nil {
			e.Licao = &texto
		}
	}
	if vistos, err := a.vistosDaEtapa(r.Context(), e); err == nil {
		e.TrechosVistos = vistos
	}
	responderJSON(w, http.StatusOK, e)
}

func (a *app) handleEditarEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	atual, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var c corpoEtapa
	if err := lerCorpo(r, &c); err != nil {
		a.responderErro(w, err)
		return
	}
	e := c.etapa
	e.ID, e.TrilhaID, e.Chave = atual.ID, atual.TrilhaID, atual.Chave
	if err := validarEtapa(&e); err != nil {
		a.responderErro(w, erroDeEntrada{err})
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	if err := a.salvarEtapa(r.Context(), tx, e); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), e.TrilhaID)
	extras := map[string]*string{}
	if c.Licao != nil {
		extras[caminhoLicao(a.slugDaTrilha(r.Context(), e.TrilhaID), e.Chave)] = licaoOuNada(c.Licao)
	}
	if err := a.exportar(r.Context(), e.TrilhaID, "etapa "+e.Chave, extras); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleEtapa(w, r)
}

func (a *app) handleExcluirEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	e, err := a.etapaPorID(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM etapas WHERE id = ?`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), e.TrilhaID)
	slug := a.slugDaTrilha(r.Context(), e.TrilhaID)
	if err := a.exportar(r.Context(), e.TrilhaID, "etapa "+e.Chave+" excluída", map[string]*string{
		caminhoLicao(slug, e.Chave): nil, caminhoEstudo(slug, tipoExercicios, e.Chave): nil, caminhoEstudo(slug, tipoSimulado, e.Chave): nil,
	}); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleStatusEtapa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Status string `json:"status"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if !contem(statusEtapa, corpo.Status) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("status desconhecido: %q", corpo.Status)})
		return
	}
	if err := a.mudarStatusEtapa(r.Context(), id, corpo.Status); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleEtapa(w, r)
}

// handleOrdem recebe a ordem completa: [{id do módulo, etapas: [ids]}]. Etapas podem mudar de módulo.
func (a *app) handleOrdem(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Modulos []struct {
			ID     int64   `json:"id"`
			Etapas []int64 `json:"etapas"`
		} `json:"modulos"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	var totalModulos, totalEtapas int
	tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM modulos WHERE trilha_id = ?`, id).Scan(&totalModulos)
	tx.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM etapas WHERE trilha_id = ?`, id).Scan(&totalEtapas)
	vistas := 0
	for i, m := range corpo.Modulos {
		res, err := tx.ExecContext(r.Context(), `UPDATE modulos SET ordem = ? WHERE id = ? AND trilha_id = ?`, i, m.ID, id)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			a.responderErro(w, erroDeEntrada{fmt.Errorf("módulo %d não é desta trilha", m.ID)})
			return
		}
		for j, e := range m.Etapas {
			res, err := tx.ExecContext(r.Context(), `UPDATE etapas SET modulo_id = ?, ordem = ? WHERE id = ? AND trilha_id = ?`, m.ID, j, e, id)
			if err != nil {
				a.responderErro(w, err)
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				a.responderErro(w, erroDeEntrada{fmt.Errorf("etapa %d não é desta trilha", e)})
				return
			}
			vistas++
		}
	}
	if len(corpo.Modulos) != totalModulos || vistas != totalEtapas {
		a.responderErro(w, erroDeEntrada{errors.New("a ordem precisa trazer todos os módulos e todas as etapas")})
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.tocarTrilha(r.Context(), id)
	if err := a.exportar(r.Context(), id, "nova ordem", nil); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleTrilha(w, r)
}
