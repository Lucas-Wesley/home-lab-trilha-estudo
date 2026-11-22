package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Anexos das aulas (T56): os PDFs que a aula da plataforma traz (o livro digital, os guias, as
// tabelas para imprimir) ficam na própria Trilha, ao lado do vídeo da aula, e entram na etapa
// como material `arquivo`. Quem baixa é a página da aula, na extensão do Chrome (com o login do
// Lucas: o link da Cademí é /area/arquivo/anexo?… e redireciona para um endereço assinado); a
// Trilha só recebe o arquivo, confere que é PDF e guarda.

const esquemaAnexos = `
CREATE TABLE IF NOT EXISTS anexos_aula (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	aula_id   INTEGER NOT NULL REFERENCES aulas_plataforma(id) ON DELETE CASCADE,
	nome      TEXT NOT NULL,          -- o nome do arquivo na plataforma ("E-BOOK CDPS .pdf")
	url       TEXT NOT NULL DEFAULT '',
	arquivo   TEXT NOT NULL,          -- relativo a CURSOS_DIR
	tamanho   INTEGER NOT NULL DEFAULT 0,
	criado_em TEXT NOT NULL,
	UNIQUE (aula_id, nome)
);
`

const limiteAnexo = 300 << 20 // 300 MB: os PDFs da escola têm de 6 a 9 MB

type anexoAula struct {
	ID      int64  `json:"id"`
	Nome    string `json:"nome"`
	Tamanho int64  `json:"tamanho"`
	Arquivo string `json:"-"`
}

func refAnexo(id int64) string { return fmt.Sprintf("/api/cursos/anexos/%d", id) }

// anexosDasAulas: os anexos de cada aula (por id da aula).
func (a *app) anexosDasAulas(ctx context.Context) (map[int64][]anexoAula, error) {
	linhas, err := a.db.QueryContext(ctx, `SELECT id, aula_id, nome, tamanho, arquivo FROM anexos_aula ORDER BY aula_id, id`)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	por := map[int64][]anexoAula{}
	for linhas.Next() {
		var x anexoAula
		var aula int64
		if err := linhas.Scan(&x.ID, &aula, &x.Nome, &x.Tamanho, &x.Arquivo); err != nil {
			return nil, err
		}
		por[aula] = append(por[aula], x)
	}
	return por, linhas.Err()
}

func (a *app) anexosDaAula(ctx context.Context, aula int64) ([]anexoAula, error) {
	por, err := a.anexosDasAulas(ctx)
	if err != nil {
		return nil, err
	}
	if por[aula] == nil {
		return []anexoAula{}, nil
	}
	return por[aula], nil
}

// nomeDoAnexo: o nome do arquivo no disco: o da aula, " - ", o do anexo, sempre .pdf.
func nomeDoAnexo(au aulaCurso, nome string) string {
	base := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(nome), filepath.Ext(nome)))
	return nomeDePasta(au.Titulo) + " - " + nomeDePasta(base) + ".pdf"
}

// POST /api/cursos/aulas/{id}/anexos (formulário: file, nome, url): guarda o PDF da aula.
// O mesmo nome de novo na mesma aula troca o arquivo (200); o novo é 201.
func (a *app) handleAnexarAula(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	au, err := a.aulaPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if a.cursosDir == "" {
		erroJSON(w, http.StatusServiceUnavailable, "CURSOS_DIR não configurado")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, limiteAnexo+1<<20)
	arq, cab, err := r.FormFile("file")
	if err != nil {
		a.responderErro(w, erroDeEntrada{errors.New("mande o PDF no campo file")})
		return
	}
	defer arq.Close()
	nome := strings.TrimSpace(r.FormValue("nome"))
	if nome == "" {
		nome = cab.Filename
	}
	if nome == "" {
		nome = "anexo.pdf"
	}
	inicio := make([]byte, 5)
	if n, _ := io.ReadFull(arq, inicio); n < 5 || !bytes.Equal(inicio, []byte("%PDF-")) {
		a.responderErro(w, erroDeEntrada{errors.New("o arquivo não é um PDF")})
		return
	}
	rel := filepath.Join(filepath.Dir(caminhoDaAula(au)), nomeDoAnexo(au, nome))
	var existente int64
	var arquivoAntigo string
	a.db.QueryRowContext(ctx, `SELECT id, arquivo FROM anexos_aula WHERE aula_id = ? AND nome = ?`, id, nome).Scan(&existente, &arquivoAntigo)
	if existente > 0 {
		rel = arquivoAntigo
	} else if _, err := os.Stat(filepath.Join(a.cursosDir, rel)); err == nil {
		rel = strings.TrimSuffix(rel, ".pdf") + fmt.Sprintf(" (%d).pdf", id)
	}
	final := filepath.Join(a.cursosDir, rel)
	if err := os.MkdirAll(filepath.Dir(final), 0o755); err != nil {
		a.responderErro(w, err)
		return
	}
	temp := final + ".recebendo"
	defer os.Remove(temp)
	f, err := os.Create(temp)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	tamanho, err := io.Copy(f, io.MultiReader(bytes.NewReader(inicio), io.LimitReader(arq, limiteAnexo)))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := os.Rename(temp, final); err != nil {
		a.responderErro(w, err)
		return
	}
	status := http.StatusCreated
	if existente > 0 {
		status = http.StatusOK
		_, err = a.db.ExecContext(ctx, `UPDATE anexos_aula SET tamanho = ?, url = ? WHERE id = ?`, tamanho, r.FormValue("url"), existente)
	} else {
		var res sql.Result
		res, err = a.db.ExecContext(ctx, `INSERT INTO anexos_aula (aula_id, nome, url, arquivo, tamanho, criado_em) VALUES (?, ?, ?, ?, ?, ?)`,
			id, nome, r.FormValue("url"), rel, tamanho, agora())
		if err == nil {
			existente, err = res.LastInsertId()
		}
	}
	if err != nil {
		a.responderErro(w, err)
		return
	}
	resp := map[string]any{"anexo": anexoAula{ID: existente, Nome: nome, Tamanho: tamanho}}
	// O curso que já é trilha ganha o PDF na etapa da aula.
	var trilha int64
	if a.db.QueryRowContext(ctx, `SELECT trilha_id FROM cursos_trilha WHERE plataforma = ? AND curso = ?`, au.Plataforma, au.Curso).Scan(&trilha) == nil {
		if _, err := a.sincronizarTrilhaDoCurso(ctx, au.Plataforma, au.Curso); err != nil {
			log.Printf("cursos: anexo %d na trilha: %v", existente, err)
			resp["trilha_erro"] = err.Error()
		}
	}
	responderJSON(w, status, resp)
}

// GET /api/cursos/anexos/{id}: o PDF, para abrir no navegador.
func (a *app) handleAnexo(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var nome, arquivo string
	if err := a.db.QueryRowContext(r.Context(), `SELECT nome, arquivo FROM anexos_aula WHERE id = ?`, id).Scan(&nome, &arquivo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errNaoEncontrou
		}
		a.responderErro(w, err)
		return
	}
	arq, err := os.Open(filepath.Join(a.cursosDir, arquivo))
	if err != nil {
		erroJSON(w, http.StatusNotFound, "o PDF não está na pasta (o disco está ligado?)")
		return
	}
	defer arq.Close()
	info, err := arq.Stat()
	if err != nil {
		a.responderErro(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": nome}))
	http.ServeContent(w, r, "", info.ModTime(), arq)
}

// apagarArquivosDosAnexos: ao apagar a aula, os PDFs dela saem do disco (as linhas saem pelo CASCADE).
func (a *app) apagarArquivosDosAnexos(ctx context.Context, aula int64) {
	anexos, err := a.anexosDaAula(ctx, aula)
	if err != nil || a.cursosDir == "" {
		return
	}
	for _, x := range anexos {
		os.Remove(filepath.Join(a.cursosDir, x.Arquivo))
	}
}
