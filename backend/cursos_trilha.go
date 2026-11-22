package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"unicode/utf8"
)

// O curso da plataforma vira trilha (T55): uma trilha por curso, no grupo da plataforma (a
// escola, ex.: "Chef Enzo Neto"), com os módulos do curso e uma etapa por aula. Sem IA e sem
// lição: a etapa tem o vídeo baixado (material `streaming` principal apontando para
// /api/cursos/aulas/<id>/video) e o texto da página da aula (documento `aula-<id>` da trilha).
// Na primeira vez a trilha nasce pela importação; depois, só ganha o que falta (aulas novas no
// lugar certo, o texto que chegou depois), sem tocar no progresso, nas lições nem no que o
// Lucas mudou à mão.

var travaTrilhaCurso sync.Mutex

func chaveEtapaDaAula(id int64) string { return fmt.Sprintf("aula-%d", id) }
func refVideoDaAula(id int64) string   { return fmt.Sprintf("/api/cursos/aulas/%d/video", id) }

func temVideo(au aulaCurso) bool { return au.Estado != "texto" }

// minutosDaAula: o vídeo arredondado para cima; sem vídeo, a leitura a ~180 palavras por minuto.
func minutosDaAula(au aulaCurso) int {
	if au.Duracao > 0 && temVideo(au) {
		return (au.Duracao + 59) / 60
	}
	if p := len(strings.Fields(au.Texto)); p > 0 {
		return max(2, (p+179)/180)
	}
	return 5
}

func materiaisDaAula(au aulaCurso) []material {
	var ms []material
	if temVideo(au) {
		ms = append(ms, material{Tipo: "streaming", Papel: "principal", Titulo: au.Titulo, Referencia: refVideoDaAula(au.ID)})
	}
	if au.TemTexto {
		papel := "apoio"
		if !temVideo(au) {
			papel = "principal"
		}
		ms = append(ms, material{Tipo: "documento", Papel: papel, Titulo: "Texto da aula", Referencia: chaveEtapaDaAula(au.ID)})
	}
	return ms
}

func etapaDaAula(au aulaCurso) etapa {
	objetivo := "Assistir à aula: " + au.Titulo
	if !temVideo(au) {
		objetivo = "Ler a aula: " + au.Titulo
	}
	return etapa{Chave: chaveEtapaDaAula(au.ID), Titulo: au.Titulo, Objetivo: objetivo, Tipo: "normal",
		Minutos: minutosDaAula(au), Materiais: materiaisDaAula(au)}
}

func documentoDaAula(au aulaCurso) string {
	return "# " + au.Titulo + "\n\n" + strings.TrimSpace(au.Texto) + "\n"
}

func moduloDaAula(au aulaCurso) string {
	if m := strings.TrimSpace(au.Modulo); m != "" {
		return m
	}
	return au.Curso
}

func (a *app) aulasDoCurso(ctx context.Context, plataforma, curso string) ([]aulaCurso, error) {
	linhas, err := a.db.QueryContext(ctx, `SELECT `+colunasAula+` FROM aulas_plataforma WHERE plataforma = ? AND curso = ?
		ORDER BY ordem, id`, plataforma, curso)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	var aulas []aulaCurso
	for linhas.Next() {
		au, err := lerAula(linhas)
		if err != nil {
			return nil, err
		}
		aulas = append(aulas, au)
	}
	return aulas, linhas.Err()
}

// sincronizarTrilhaDoCurso cria a trilha do curso ou acrescenta o que falta; devolve o id dela.
func (a *app) sincronizarTrilhaDoCurso(ctx context.Context, plataforma, curso string) (int64, error) {
	travaTrilhaCurso.Lock()
	defer travaTrilhaCurso.Unlock()
	aulas, err := a.aulasDoCurso(ctx, plataforma, curso)
	if err != nil {
		return 0, err
	}
	if len(aulas) == 0 {
		return 0, errNaoEncontrou
	}
	var id int64
	err = a.db.QueryRowContext(ctx, `SELECT trilha_id FROM cursos_trilha WHERE plataforma = ? AND curso = ?`, plataforma, curso).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return a.criarTrilhaDoCurso(ctx, plataforma, curso, aulas)
	}
	if err != nil {
		return 0, err
	}
	return id, a.completarTrilhaDoCurso(ctx, id, aulas)
}

func (a *app) criarTrilhaDoCurso(ctx context.Context, plataforma, curso string, aulas []aulaCurso) (int64, error) {
	escola := plataformasCurso[plataforma]
	slug, err := slugLivre(ctx, a.db, slugificar(escola+" "+curso), `SELECT COUNT(*) FROM trilhas WHERE slug = ?`)
	if err != nil {
		return 0, err
	}
	type etapaPacote struct {
		etapa
	}
	type moduloPacote struct {
		Titulo string        `json:"titulo"`
		Etapas []etapaPacote `json:"etapas"`
	}
	var modulos []*moduloPacote
	porTitulo := map[string]*moduloPacote{}
	docs := []documento{}
	for _, au := range aulas {
		t := moduloDaAula(au)
		m := porTitulo[t]
		if m == nil {
			m = &moduloPacote{Titulo: t}
			porTitulo[t] = m
			modulos = append(modulos, m)
		}
		m.Etapas = append(m.Etapas, etapaPacote{etapaDaAula(au)})
		if au.TemTexto {
			docs = append(docs, documento{Slug: chaveEtapaDaAula(au.ID), Titulo: au.Titulo, Conteudo: documentoDaAula(au)})
		}
	}
	bruto, err := json.Marshal(map[string]any{
		"documentos": docs,
		"trilha": map[string]any{
			"slug": slug, "titulo": curso, "grupo": escola, "estado": "ativa",
			"objetivo":  "Assistir às aulas de " + curso,
			"descricao": fmt.Sprintf("Curso %s, da %s. As aulas vêm da Entrada › Cursos, baixadas pela extensão do Chrome.", curso, escola),
			"modulos":   modulos,
		},
	})
	if err != nil {
		return 0, err
	}
	var p pacote
	if err := json.Unmarshal(bruto, &p); err != nil {
		return 0, err
	}
	res, err := a.importar(ctx, p)
	if err != nil {
		return 0, err
	}
	if _, err := a.db.ExecContext(ctx, `INSERT OR REPLACE INTO cursos_trilha (plataforma, curso, trilha_id) VALUES (?, ?, ?)`,
		plataforma, curso, res.TrilhaID); err != nil {
		return 0, err
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE aulas_plataforma SET na_trilha = 1 WHERE plataforma = ? AND curso = ?`, plataforma, curso); err != nil {
		return 0, err
	}
	return res.TrilhaID, nil
}

// completarTrilhaDoCurso: a aula nova entra no módulo dela, logo depois da aula anterior do curso
// (o módulo novo, logo depois do módulo da aula anterior); a etapa que já existe só ganha o
// material que faltava (o texto que chegou depois). A aula que já entrou na trilha uma vez
// (aulas_plataforma.na_trilha) e cuja etapa o Lucas apagou não volta.
func (a *app) completarTrilhaDoCurso(ctx context.Context, trilhaID int64, aulas []aulaCurso) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	type lugar struct {
		etapa, modulo int64
		ordem         int
	}
	etapas := map[string]lugar{}
	linhas, err := tx.QueryContext(ctx, `SELECT id, modulo_id, ordem, chave FROM etapas WHERE trilha_id = ?`, trilhaID)
	if err != nil {
		return err
	}
	for linhas.Next() {
		var l lugar
		var chave string
		if err := linhas.Scan(&l.etapa, &l.modulo, &l.ordem, &chave); err != nil {
			linhas.Close()
			return err
		}
		etapas[chave] = l
	}
	linhas.Close()
	modulos := map[string]int64{}
	ordemModulo := map[int64]int{}
	linhas, err = tx.QueryContext(ctx, `SELECT id, titulo, ordem FROM modulos WHERE trilha_id = ?`, trilhaID)
	if err != nil {
		return err
	}
	for linhas.Next() {
		var id int64
		var titulo string
		var ordem int
		if err := linhas.Scan(&id, &titulo, &ordem); err != nil {
			linhas.Close()
			return err
		}
		if _, ja := modulos[titulo]; !ja {
			modulos[titulo] = id
		}
		ordemModulo[id] = ordem
	}
	linhas.Close()

	slug := ""
	if err := tx.QueryRowContext(ctx, `SELECT slug FROM trilhas WHERE id = ?`, trilhaID).Scan(&slug); err != nil {
		return err
	}
	jaEntrou := map[int64]bool{}
	linhas, err = tx.QueryContext(ctx, `SELECT id FROM aulas_plataforma WHERE plataforma = ? AND curso = ? AND na_trilha = 1`,
		aulas[0].Plataforma, aulas[0].Curso)
	if err != nil {
		return err
	}
	for linhas.Next() {
		var id int64
		if err := linhas.Scan(&id); err != nil {
			linhas.Close()
			return err
		}
		jaEntrou[id] = true
	}
	linhas.Close()
	docs := map[string]*string{}
	novas := 0
	var anterior *lugar
	for _, au := range aulas {
		chave := chaveEtapaDaAula(au.ID)
		if l, ok := etapas[chave]; ok {
			if err := completarMateriais(ctx, tx, l.etapa, au); err != nil {
				return err
			}
			copia := l
			anterior = &copia
		} else if jaEntrou[au.ID] {
			continue // a etapa foi apagada na trilha
		} else {
			titulo := moduloDaAula(au)
			mod, ok := modulos[titulo]
			pos := 0
			if !ok {
				// Módulo novo: logo depois do módulo da aula anterior (ou no começo).
				om := 0
				if anterior != nil {
					om = ordemModulo[anterior.modulo] + 1
				}
				if _, err := tx.ExecContext(ctx, `UPDATE modulos SET ordem = ordem + 1 WHERE trilha_id = ? AND ordem >= ?`, trilhaID, om); err != nil {
					return err
				}
				for k, v := range ordemModulo {
					if v >= om {
						ordemModulo[k] = v + 1
					}
				}
				r, err := tx.ExecContext(ctx, `INSERT INTO modulos (trilha_id, ordem, titulo) VALUES (?, ?, ?)`, trilhaID, om, titulo)
				if err != nil {
					return err
				}
				if mod, err = r.LastInsertId(); err != nil {
					return err
				}
				modulos[titulo], ordemModulo[mod] = mod, om
			} else if anterior != nil && anterior.modulo == mod {
				pos = anterior.ordem + 1
			}
			if _, err := tx.ExecContext(ctx, `UPDATE etapas SET ordem = ordem + 1 WHERE modulo_id = ? AND ordem >= ?`, mod, pos); err != nil {
				return err
			}
			for k, v := range etapas {
				if v.modulo == mod && v.ordem >= pos {
					v.ordem++
					etapas[k] = v
				}
			}
			et := etapaDaAula(au)
			et.TrilhaID = trilhaID
			r, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo) VALUES (?, ?, ?, ?, ?)`,
				trilhaID, mod, pos, chave, au.Titulo)
			if err != nil {
				return err
			}
			if et.ID, err = r.LastInsertId(); err != nil {
				return err
			}
			if err := a.salvarEtapa(ctx, tx, et); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE aulas_plataforma SET na_trilha = 1 WHERE id = ?`, au.ID); err != nil {
				return err
			}
			l := lugar{et.ID, mod, pos}
			etapas[chave] = l
			anterior = &l
			novas++
		}
		if au.TemTexto {
			caminho := caminhoDocumento(slug, chave)
			texto := documentoDaAula(au)
			if atual, err := a.conteudo.ler(caminho); err != nil || atual != texto {
				docs[caminho] = &texto
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if novas == 0 && len(docs) == 0 {
		return nil
	}
	a.tocarTrilha(ctx, trilhaID)
	mensagem := fmt.Sprintf("curso: %d aula(s) nova(s)", novas)
	if novas == 0 {
		mensagem = "curso: texto das aulas"
	}
	if err := a.exportar(ctx, trilhaID, mensagem, docs); err != nil {
		return err
	}
	if novas > 0 {
		var estado string
		a.db.QueryRowContext(ctx, `SELECT estado FROM trilhas WHERE id = ?`, trilhaID).Scan(&estado)
		if estado == "ativa" {
			if err := a.gerarCronograma(ctx, trilhaID, hoje()); err != nil && err != errSemDias {
				return err
			}
		}
	}
	return nil
}

// completarMateriais: a etapa que já existe ganha o vídeo ou o texto da aula, se ainda não tem.
func completarMateriais(ctx context.Context, tx *sql.Tx, etapaID int64, au aulaCurso) error {
	for _, m := range materiaisDaAula(au) {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM materiais WHERE etapa_id = ? AND tipo = ? AND referencia = ?`,
			etapaID, m.Tipo, m.Referencia).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO materiais (etapa_id, ordem, tipo, titulo, referencia, localizador, papel, nota)
			VALUES (?, (SELECT COALESCE(MAX(ordem), -1) + 1 FROM materiais WHERE etapa_id = ?), ?, ?, ?, '', ?, '')`,
			etapaID, etapaID, m.Tipo, m.Titulo, m.Referencia, m.Papel); err != nil {
			return err
		}
	}
	return nil
}

type trilhaDoCurso struct {
	Plataforma string `json:"plataforma"`
	Curso      string `json:"curso"`
	TrilhaID   int64  `json:"trilha_id"`
}

func (a *app) trilhasDosCursos(ctx context.Context) ([]trilhaDoCurso, error) {
	linhas, err := a.db.QueryContext(ctx, `SELECT plataforma, curso, trilha_id FROM cursos_trilha ORDER BY plataforma, curso`)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	lista := []trilhaDoCurso{}
	for linhas.Next() {
		var t trilhaDoCurso
		if err := linhas.Scan(&t.Plataforma, &t.Curso, &t.TrilhaID); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, linhas.Err()
}

// POST /api/cursos/trilha {plataforma, curso}: cria a trilha do curso (ou acrescenta as aulas que faltam).
func (a *app) handleTrilhaDoCurso(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Plataforma string `json:"plataforma"`
		Curso      string `json:"curso"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	if plataformasCurso[corpo.Plataforma] == "" || strings.TrimSpace(corpo.Curso) == "" || !utf8.ValidString(corpo.Curso) {
		a.responderErro(w, erroDeEntrada{errors.New("mande a plataforma e o curso")})
		return
	}
	id, err := a.sincronizarTrilhaDoCurso(r.Context(), corpo.Plataforma, corpo.Curso)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, map[string]any{"trilha_id": id})
}
