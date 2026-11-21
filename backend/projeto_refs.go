package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Projeto de estudo (T52): as referências do roadmap ("T13 › Concorrência › Goroutines",
// "T24 › 10 Multithreading", "T22 cap. 3, 4 e 8", "T36 › módulos 1 a 3"…) ligadas às trilhas do
// banco. Casa pelo título (exato, prefixo por palavra, trecho ou todas as palavras) ou pelo número
// do módulo/etapa/capítulo; o que não casar fica como texto.

// referenciaProjeto: um alvo de estudo já procurado no banco.
type referenciaProjeto struct {
	Texto        string `json:"texto"` // como veio do roadmap (limpo)
	Tipo         string `json:"tipo"`  // trilha · modulo · etapa · "" (não casou)
	Parcial      bool   `json:"parcial,omitempty"`
	TrilhaID     int64  `json:"trilha_id,omitempty"`
	TrilhaTitulo string `json:"trilha_titulo,omitempty"`
	ModuloID     int64  `json:"modulo_id,omitempty"`
	EtapaID      int64  `json:"etapa_id,omitempty"`
	Caminho      string `json:"caminho,omitempty"` // "T13 › Concorrência › Goroutines e o modelo…"
	Status       string `json:"status,omitempty"`  // da etapa; "concluida" no módulo/trilha completos
	Concluidas   int    `json:"concluidas,omitempty"`
	Total        int    `json:"total,omitempty"`
	concluidas   []string
}

// segmento de um caminho: texto, ou número (módulo/etapa "10", "M1", "módulos 1 a 3").
type segmentoRef struct {
	Texto  string
	Num    int  // -1 = sem número
	Modulo bool // "M1", "módulo(s) N", "capítulos N": só módulo (com a posição como reserva)
}

type alvoRef struct {
	Texto   string
	Trilha  int64
	Cap     int // > 0: "cap. N"
	Inteira bool
	Caminho []segmentoRef
}

var (
	reTrilhaRef  = regexp.MustCompile(`\bT(\d+)\b`)
	reFimFrase   = regexp.MustCompile(`\.\s+\p{Lu}`)
	reNumerosRef = regexp.MustCompile(`(\d+)(?:\s+a\s+(\d+))?`)
	reSegM       = regexp.MustCompile(`^M(\d+)\b\s*(.*)$`)
	reSegModulos = regexp.MustCompile(`(?i)^(m[oó]dulos?|cap[ií]tulos?)\s+(.*)$`)
	reCapRef     = regexp.MustCompile(`(?i)^(?:caps?\.?|cap[ií]tulos?)\s*(.*)$`)
	reNumTitulo  = regexp.MustCompile(`^(?:modulo\s+)?(\d+)\b`)
	reCapTitulo  = regexp.MustCompile(`^cap\s+(\d+)\b`)
	reBaseTitulo = regexp.MustCompile(`^(?:modulo\s+|cap\s+)?\d+\s+`)
)

// limparRef tira negrito, código, parênteses e a frase explicativa que vem depois do ponto.
func limparRef(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	for {
		novo := reParenteses.ReplaceAllString(s, "")
		if novo == s {
			break
		}
		s = novo
	}
	if loc := reFimFrase.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, " ,", ",")
	s = strings.ReplaceAll(s, " ;", ";")
	return strings.TrimSpace(s)
}

func numerosDe(s string) []int {
	var nums []int
	for _, m := range reNumerosRef.FindAllStringSubmatch(s, -1) {
		de, _ := strconv.Atoi(m[1])
		ate := de
		if m[2] != "" {
			ate, _ = strconv.Atoi(m[2])
		}
		if ate < de || ate-de > 200 {
			ate = de
		}
		for n := de; n <= ate; n++ {
			nums = append(nums, n)
		}
	}
	return nums
}

func comecaComDigito(s string) bool {
	s = strings.TrimSpace(s)
	return s != "" && s[0] >= '0' && s[0] <= '9'
}

func lerSegmento(s string) []segmentoRef {
	s = strings.TrimSpace(s)
	if m := reSegM.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return []segmentoRef{{Texto: strings.TrimSpace(m[2]), Num: n, Modulo: true}}
	}
	if m := reSegModulos.FindStringSubmatch(s); m != nil {
		var segs []segmentoRef
		for _, n := range numerosDe(m[2]) {
			segs = append(segs, segmentoRef{Num: n, Modulo: true})
		}
		if len(segs) > 0 {
			return segs
		}
	}
	if comecaComDigito(s) {
		// "10 e 33", "002 a 004", "04 e 09 APIs", "05 Context": os números do começo.
		i := 0
		for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == ' ' || s[i] == ',' ||
			strings.HasPrefix(s[i:], "e ") || strings.HasPrefix(s[i:], "a ")) {
			if strings.HasPrefix(s[i:], "e ") || strings.HasPrefix(s[i:], "a ") {
				// só é conector se vier um número depois
				if !comecaComDigito(s[i+2:]) {
					break
				}
				i += 2
				continue
			}
			i++
		}
		texto := strings.TrimSpace(s[i:])
		var segs []segmentoRef
		for _, n := range numerosDe(s[:i]) {
			segs = append(segs, segmentoRef{Texto: texto, Num: n})
		}
		if len(segs) > 0 {
			return segs
		}
	}
	return []segmentoRef{{Texto: s, Num: -1}}
}

// alternativas: "Goroutines; Channels e select" e "06, 17 SQLC, 18 UOW" (vírgula só separa quando
// as duas partes começam com número).
func alternativas(s string) []string {
	var saida []string
	for _, parte := range strings.Split(s, ";") {
		var atual []string
		for _, p := range strings.Split(parte, ",") {
			if len(atual) > 0 && comecaComDigito(p) && comecaComDigito(atual[0]) {
				saida = append(saida, strings.TrimSpace(strings.Join(atual, ",")))
				atual = nil
			}
			atual = append(atual, p)
		}
		if t := strings.TrimSpace(strings.Join(atual, ",")); t != "" {
			saida = append(saida, t)
		}
	}
	return saida
}

// lerReferencias quebra um texto do roadmap nos alvos de estudo.
func lerReferencias(texto string) []alvoRef {
	s := limparRef(texto)
	locs := reTrilhaRef.FindAllStringSubmatchIndex(s, -1)
	var alvos []alvoRef
	for i, loc := range locs {
		fim := len(s)
		if i+1 < len(locs) {
			fim = locs[i+1][0]
		}
		pedaco := s[loc[0]:fim]
		for {
			antes := pedaco
			pedaco = strings.TrimRight(pedaco, " ·;,—–-")
			pedaco = strings.TrimSuffix(pedaco, " e")
			pedaco = strings.TrimSuffix(pedaco, " ou")
			if pedaco == antes {
				break
			}
		}
		n, _ := strconv.ParseInt(s[loc[2]:loc[3]], 10, 64)
		resto := strings.TrimSpace(pedaco[loc[1]-loc[0]:])
		base := alvoRef{Trilha: n}
		switch {
		case strings.HasPrefix(resto, "›") || strings.HasPrefix(resto, ">"):
			partes := strings.FieldsFunc(resto, func(r rune) bool { return r == '›' || r == '>' })
			var caminho []segmentoRef
			for _, p := range partes[:max(0, len(partes)-1)] {
				if segs := lerSegmento(p); len(segs) > 0 {
					caminho = append(caminho, segs[0])
				}
			}
			if len(partes) == 0 {
				alvos = append(alvos, alvoRef{Texto: pedaco, Trilha: n, Inteira: true})
				continue
			}
			prefixo := fmt.Sprintf("T%d › ", n)
			for _, p := range partes[:len(partes)-1] {
				prefixo += strings.TrimSpace(p) + " › "
			}
			for _, alt := range alternativas(partes[len(partes)-1]) {
				for _, seg := range lerSegmento(alt) {
					a := base
					a.Texto = prefixo + alt
					a.Caminho = append(append([]segmentoRef{}, caminho...), seg)
					alvos = append(alvos, a)
				}
			}
		case reCapRef.MatchString(resto) && len(numerosDe(resto)) > 0:
			for _, c := range numerosDe(resto) {
				a := base
				a.Texto = fmt.Sprintf("T%d cap. %d", n, c)
				a.Cap = c
				alvos = append(alvos, a)
			}
		default:
			alvos = append(alvos, alvoRef{Texto: fmt.Sprintf("T%d", n), Trilha: n, Inteira: true})
		}
	}
	return alvos
}

// ---------- catálogo das trilhas ----------

type etapaCat struct {
	ID          int64
	Titulo      string
	Status      string
	Num, Cap    int
	norma, base string
	mod         *moduloCat
}

type moduloCat struct {
	ID          int64
	Titulo      string
	Num         int
	Posicao     int // 1, 2, 3…
	norma, base string
	Etapas      []*etapaCat
	tri         *trilhaCat
}

type trilhaCat struct {
	ID      int64
	Titulo  string
	Modulos []*moduloCat
	Etapas  []*etapaCat // na ordem de estudo
}

type catalogoTrilhas map[int64]*trilhaCat

func numeroDoTitulo(re *regexp.Regexp, n string) int {
	if m := re.FindStringSubmatch(n); m != nil {
		v, _ := strconv.Atoi(m[1])
		return v
	}
	return -1
}

func (c catalogoTrilhas) trilha(id int64, titulo string) *trilhaCat {
	t := c[id]
	if t == nil {
		t = &trilhaCat{ID: id, Titulo: titulo}
		c[id] = t
	}
	return t
}

func (t *trilhaCat) modulo(id int64, titulo string) *moduloCat {
	n := normalizar(titulo)
	m := &moduloCat{ID: id, Titulo: titulo, norma: n, base: reBaseTitulo.ReplaceAllString(n, ""),
		Num: numeroDoTitulo(reNumTitulo, n), Posicao: len(t.Modulos) + 1, tri: t}
	t.Modulos = append(t.Modulos, m)
	return m
}

func (m *moduloCat) etapa(id int64, titulo, status string) *etapaCat {
	n := normalizar(titulo)
	e := &etapaCat{ID: id, Titulo: titulo, Status: status, norma: n, base: reBaseTitulo.ReplaceAllString(n, ""),
		Num: numeroDoTitulo(reNumTitulo, n), Cap: numeroDoTitulo(reCapTitulo, n), mod: m}
	m.Etapas = append(m.Etapas, e)
	m.tri.Etapas = append(m.tri.Etapas, e)
	return e
}

// catalogo lê todas as trilhas, módulos e etapas (só o que o casamento usa) em três consultas.
func (a *app) catalogo(ctx context.Context) (catalogoTrilhas, error) {
	c := catalogoTrilhas{}
	rows, err := a.db.QueryContext(ctx, `SELECT id, titulo FROM trilhas`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var titulo string
		if err := rows.Scan(&id, &titulo); err != nil {
			rows.Close()
			return nil, err
		}
		c.trilha(id, titulo)
	}
	rows.Close()
	mods := map[int64]*moduloCat{}
	rows, err = a.db.QueryContext(ctx, `SELECT id, trilha_id, titulo FROM modulos ORDER BY trilha_id, ordem, id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, tid int64
		var titulo string
		if err := rows.Scan(&id, &tid, &titulo); err != nil {
			rows.Close()
			return nil, err
		}
		if t := c[tid]; t != nil {
			mods[id] = t.modulo(id, titulo)
		}
	}
	rows.Close()
	rows, err = a.db.QueryContext(ctx, `SELECT e.id, e.modulo_id, e.titulo, e.status FROM etapas e
		JOIN modulos m ON m.id = e.modulo_id ORDER BY e.trilha_id, m.ordem, m.id, e.ordem, e.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, mid int64
		var titulo, status string
		if err := rows.Scan(&id, &mid, &titulo, &status); err != nil {
			return nil, err
		}
		if m := mods[mid]; m != nil {
			m.etapa(id, titulo, status)
		}
	}
	return c, rows.Err()
}

// ---------- casamento ----------

var palavrasVazias = map[string]bool{"e": true, "de": true, "da": true, "do": true, "das": true, "dos": true,
	"o": true, "a": true, "os": true, "as": true, "em": true, "no": true, "na": true, "x": true,
	"com": true, "para": true, "por": true, "um": true, "uma": true}

// notaTitulo: 4 igual · 3 começa com · 2 contém (palavras inteiras) · 1 todas as palavras · 0 não.
func notaTitulo(ref, norma, base string) int {
	r := normalizar(ref)
	if r == "" {
		return 0
	}
	melhor := 0
	for _, t := range []string{norma, base} {
		switch {
		case t == r:
			return 4
		case strings.HasPrefix(t, r+" "):
			melhor = max(melhor, 3)
		case strings.Contains(" "+t+" ", " "+r+" "):
			melhor = max(melhor, 2)
		}
	}
	if melhor > 0 {
		return melhor
	}
	palavras := map[string]bool{}
	for _, p := range strings.Fields(norma) {
		palavras[p] = true
	}
	significativas := 0
	for _, p := range strings.Fields(r) {
		if palavrasVazias[p] {
			continue
		}
		significativas++
		if !palavras[p] {
			return 0
		}
	}
	if significativas == 0 {
		return 0
	}
	return 1
}

func melhorModulo(t *trilhaCat, texto string) (*moduloCat, int) {
	var achado *moduloCat
	nota := 0
	for _, m := range t.Modulos {
		if n := notaTitulo(texto, m.norma, m.base); n > nota {
			achado, nota = m, n
		}
	}
	return achado, nota
}

func melhorEtapa(etapas []*etapaCat, texto string) (*etapaCat, int) {
	var achada *etapaCat
	nota := 0
	for _, e := range etapas {
		if n := notaTitulo(texto, e.norma, e.base); n > nota {
			achada, nota = e, n
		}
	}
	return achada, nota
}

func moduloPorNumero(t *trilhaCat, n int, posicao bool) *moduloCat {
	for _, m := range t.Modulos {
		if m.Num == n {
			return m
		}
	}
	if posicao && n >= 1 && n <= len(t.Modulos) {
		return t.Modulos[n-1]
	}
	return nil
}

func etapaPorNumero(etapas []*etapaCat, n int) *etapaCat {
	for _, e := range etapas {
		if e.Num == n {
			return e
		}
	}
	return nil
}

func nomeTrilha(t *trilhaCat) string { return fmt.Sprintf("T%d", t.ID) }

func caminhoEtapa(e *etapaCat) string {
	return nomeTrilha(e.mod.tri) + " › " + e.mod.Titulo + " › " + e.Titulo
}

func (r *referenciaProjeto) comEtapa(e *etapaCat) {
	r.Tipo, r.ModuloID, r.EtapaID, r.Status = "etapa", e.mod.ID, e.ID, e.Status
	r.Caminho = caminhoEtapa(e)
	if e.Status == "concluida" {
		r.concluidas = []string{r.Caminho}
	}
}

func (r *referenciaProjeto) contar(etapas []*etapaCat) {
	r.Total, r.Concluidas, r.concluidas = len(etapas), 0, nil
	for _, e := range etapas {
		if e.Status == "concluida" {
			r.Concluidas++
			r.concluidas = append(r.concluidas, caminhoEtapa(e))
		}
	}
	if r.Total > 0 && r.Concluidas == r.Total {
		r.Status = "concluida"
	}
}

func (r *referenciaProjeto) comModulo(m *moduloCat) {
	r.Tipo, r.ModuloID = "modulo", m.ID
	r.Caminho = nomeTrilha(m.tri) + " › " + m.Titulo
	r.contar(m.Etapas)
}

func (r *referenciaProjeto) comTrilha(t *trilhaCat) {
	r.Tipo = "trilha"
	r.Caminho = nomeTrilha(t) + " · " + t.Titulo
	r.contar(t.Etapas)
}

// casar procura um alvo no catálogo. Achou só o nível de cima (o módulo, mas não a etapa) →
// devolve o de cima com Parcial.
func (c catalogoTrilhas) casar(a alvoRef) referenciaProjeto {
	r := referenciaProjeto{Texto: a.Texto}
	t := c[a.Trilha]
	if t == nil {
		return r
	}
	r.TrilhaID, r.TrilhaTitulo = t.ID, t.Titulo
	switch {
	case a.Inteira:
		r.comTrilha(t)
		return r
	case a.Cap > 0:
		for _, e := range t.Etapas {
			if e.Cap == a.Cap {
				r.comEtapa(e)
				return r
			}
		}
		if m := moduloPorNumero(t, a.Cap, true); m != nil {
			r.comModulo(m)
			return r
		}
		r.comTrilha(t)
		r.Parcial = true
		return r
	case len(a.Caminho) == 0:
		r.comTrilha(t)
		return r
	}

	s0 := a.Caminho[0]
	var mod *moduloCat
	var etp *etapaCat
	if s0.Num >= 0 {
		mod = moduloPorNumero(t, s0.Num, s0.Modulo)
		if mod == nil && !s0.Modulo {
			etp = etapaPorNumero(t.Etapas, s0.Num)
		}
	} else {
		m, nm := melhorModulo(t, s0.Texto)
		if len(a.Caminho) == 1 {
			e, ne := melhorEtapa(t.Etapas, s0.Texto)
			if m != nil && nm >= ne {
				mod = m
			} else {
				etp = e
			}
		} else {
			mod = m
		}
	}

	if len(a.Caminho) >= 2 {
		ultimo := a.Caminho[len(a.Caminho)-1]
		etapas := t.Etapas
		if mod != nil {
			etapas = mod.Etapas
		}
		if ultimo.Num >= 0 {
			etp = etapaPorNumero(etapas, ultimo.Num)
		} else {
			etp, _ = melhorEtapa(etapas, ultimo.Texto)
		}
		if etp == nil && mod == nil {
			// o módulo mudou de nome: a etapa ainda pode estar na trilha
			etp, _ = melhorEtapa(t.Etapas, ultimo.Texto)
		}
		if etp == nil && mod != nil {
			r.comModulo(mod)
			r.Parcial = true
			return r
		}
	}

	switch {
	case etp != nil:
		r.comEtapa(etp)
	case mod != nil:
		r.comModulo(mod)
	default:
		r.comTrilha(t)
		r.Parcial = true
	}
	return r
}

func (c catalogoTrilhas) referencias(texto string) []referenciaProjeto {
	refs := []referenciaProjeto{}
	for _, a := range lerReferencias(texto) {
		refs = append(refs, c.casar(a))
	}
	return refs
}
