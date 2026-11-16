package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// pacote é uma trilha inteira (e os documentos dela) num JSON só. É o formato da trilha
// montada fora do app (a piloto, feita à mão com o processo de regras/criar-trilha.md) e, na fase 4,
// o que a IA vai devolver. Ver docs/CRIACAO-DE-TRILHAS.md.
type pacote struct {
	Substituir bool        `json:"substituir"`
	Documentos []documento `json:"documentos"`
	Trilha     *struct {
		trilha
		Modulos []struct {
			Titulo    string `json:"titulo"`
			Descricao string `json:"descricao"`
			Etapas    []struct {
				etapa
				Requer []string `json:"requer"`
				Licao  string   `json:"licao"`
			} `json:"etapas"`
		} `json:"modulos"`
	} `json:"trilha"`
}

// validarPacote confere chaves, requisitos e ordem antes de tocar no banco.
func validarPacote(p *pacote) error {
	for i := range p.Documentos {
		d := &p.Documentos[i]
		if !slugValido.MatchString(d.Slug) {
			return fmt.Errorf("documento com nome inválido: %q", d.Slug)
		}
		if strings.TrimSpace(d.Conteudo) == "" {
			return fmt.Errorf("documento %q vazio", d.Slug)
		}
	}
	if p.Trilha == nil {
		return errors.New("o pacote precisa de uma trilha (os documentos pertencem a ela)")
	}
	t := p.Trilha
	if err := validarTrilha(&t.trilha); err != nil {
		return err
	}
	if t.Slug == "" {
		t.Slug = slugificar(t.Titulo)
	}
	if !slugValido.MatchString(t.Slug) {
		return fmt.Errorf("slug da trilha inválido: %q", t.Slug)
	}
	posicao := map[string]int{}
	n := 0
	for mi := range t.Modulos {
		m := &t.Modulos[mi]
		if m.Titulo = strings.TrimSpace(m.Titulo); m.Titulo == "" {
			return fmt.Errorf("o módulo %d precisa de um título", mi+1)
		}
		for ei := range m.Etapas {
			e := &m.Etapas[ei]
			if e.Chave == "" {
				e.Chave = slugificar(e.Titulo)
			}
			if !slugValido.MatchString(e.Chave) {
				return fmt.Errorf("chave de etapa inválida: %q", e.Chave)
			}
			if _, repetida := posicao[e.Chave]; repetida {
				return fmt.Errorf("chave de etapa repetida: %q", e.Chave)
			}
			if err := validarEtapa(&e.etapa); err != nil {
				return fmt.Errorf("etapa %q: %w", e.Chave, err)
			}
			posicao[e.Chave] = n
			n++
		}
	}
	// Todo pré-requisito existe e vem antes (o que também impede ciclos).
	for _, m := range t.Modulos {
		for _, e := range m.Etapas {
			for _, r := range e.Requer {
				p, ok := posicao[r]
				if !ok {
					return fmt.Errorf("etapa %q requer %q, que não existe", e.Chave, r)
				}
				if p >= posicao[e.Chave] {
					return fmt.Errorf("etapa %q requer %q, que vem depois dela", e.Chave, r)
				}
			}
		}
	}
	return nil
}

type resultadoImportacao struct {
	Documentos int    `json:"documentos"`
	TrilhaID   int64  `json:"trilha_id,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Etapas     int    `json:"etapas"`
	Mantidas   int    `json:"progresso_mantido"` // etapas cujo status veio da versão anterior
}

func (a *app) importar(ctx context.Context, p pacote) (resultadoImportacao, error) {
	var res resultadoImportacao
	if err := validarPacote(&p); err != nil {
		return res, erroDeEntrada{err}
	}
	t := p.Trilha

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	defer tx.Rollback()

	agora := agora()
	var id int64
	var estado string
	err = tx.QueryRowContext(ctx, `SELECT id, estado FROM trilhas WHERE slug = ?`, t.Slug).Scan(&id, &estado)
	statusAnterior := map[string][3]string{}
	licoesAntigas := []string{}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		r, err := tx.ExecContext(ctx, `INSERT INTO trilhas (slug, titulo, objetivo, descricao, recorte, instrucoes,
			nivel_partida, nivel_alvo, profundidade, estado, criado_em, atualizado_em)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'rascunho', ?, ?)`,
			t.Slug, t.Titulo, t.Objetivo, t.Descricao, t.Recorte, t.Instrucoes, t.NivelPartida, t.NivelAlvo, t.Profundidade, agora, agora)
		if err != nil {
			return res, err
		}
		if id, err = r.LastInsertId(); err != nil {
			return res, err
		}
	case err != nil:
		return res, err
	case !p.Substituir:
		return res, erroDeEntrada{fmt.Errorf("a trilha %q já existe; mande \"substituir\": true para trocar a estrutura (o progresso das etapas com a mesma chave fica)", t.Slug)}
	default:
		rows, err := tx.QueryContext(ctx, `SELECT chave, status, iniciado_em, concluido_em FROM etapas WHERE trilha_id = ?`, id)
		if err != nil {
			return res, err
		}
		for rows.Next() {
			var chave string
			var s [3]string
			if err := rows.Scan(&chave, &s[0], &s[1], &s[2]); err != nil {
				rows.Close()
				return res, err
			}
			statusAnterior[chave] = s
			licoesAntigas = append(licoesAntigas, chave)
		}
		rows.Close()
		if _, err := tx.ExecContext(ctx, `DELETE FROM modulos WHERE trilha_id = ?`, id); err != nil {
			return res, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE trilhas SET titulo = ?, objetivo = ?, descricao = ?, recorte = ?, instrucoes = ?,
			nivel_partida = ?, nivel_alvo = ?, profundidade = ?, atualizado_em = ? WHERE id = ?`,
			t.Titulo, t.Objetivo, t.Descricao, t.Recorte, t.Instrucoes, t.NivelPartida, t.NivelAlvo, t.Profundidade, agora, id); err != nil {
			return res, err
		}
	}

	idDaChave := map[string]int64{}
	licoes := map[string]*string{}
	for mi, m := range t.Modulos {
		r, err := tx.ExecContext(ctx, `INSERT INTO modulos (trilha_id, ordem, titulo, descricao) VALUES (?, ?, ?, ?)`, id, mi, m.Titulo, m.Descricao)
		if err != nil {
			return res, err
		}
		moduloID, err := r.LastInsertId()
		if err != nil {
			return res, err
		}
		for ei, e := range m.Etapas {
			s, manteve := statusAnterior[e.Chave]
			if !manteve {
				s = [3]string{"disponivel", "", ""}
			} else {
				res.Mantidas++
			}
			r, err := tx.ExecContext(ctx, `INSERT INTO etapas (trilha_id, modulo_id, ordem, chave, titulo, status, iniciado_em, concluido_em)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, moduloID, ei, e.Chave, e.Titulo, s[0], s[1], s[2])
			if err != nil {
				return res, err
			}
			et := e.etapa
			if et.ID, err = r.LastInsertId(); err != nil {
				return res, err
			}
			et.TrilhaID = id
			et.Requer = nil
			for _, req := range e.Requer {
				et.Requer = append(et.Requer, idDaChave[req])
			}
			if err := a.salvarEtapa(ctx, tx, et); err != nil {
				return res, err
			}
			idDaChave[e.Chave] = et.ID
			if strings.TrimSpace(e.Licao) != "" {
				texto := e.Licao
				licoes[caminhoLicao(t.Slug, e.Chave)] = &texto
			}
			res.Etapas++
		}
	}
	if err := tx.Commit(); err != nil {
		return res, err
	}
	// Lição de etapa que saiu, ou que ficou sem texto, sai do conteúdo.
	// Exercícios e simulado só saem com a etapa (a nova versão do plano não traz os dois).
	for _, chave := range licoesAntigas {
		if _, fica := licoes[caminhoLicao(t.Slug, chave)]; !fica {
			licoes[caminhoLicao(t.Slug, chave)] = nil
		}
		if _, fica := idDaChave[chave]; !fica {
			licoes[caminhoEstudo(t.Slug, tipoExercicios, chave)] = nil
			licoes[caminhoEstudo(t.Slug, tipoSimulado, chave)] = nil
		}
	}
	for i := range p.Documentos {
		licoes[caminhoDocumento(t.Slug, p.Documentos[i].Slug)] = &p.Documentos[i].Conteudo
		res.Documentos++
	}
	mensagem := "importada"
	if p.Substituir && len(statusAnterior) > 0 {
		mensagem = "nova versão importada"
	}
	if err := a.exportar(ctx, id, mensagem, licoes); err != nil {
		return res, err
	}
	if estado == "ativa" {
		// Sem meta ainda, não há cronograma a refazer (a nova versão já está gravada).
		if err := a.gerarCronograma(ctx, id, hoje()); err != nil && err != errSemDias {
			return res, err
		}
	}
	res.TrilhaID, res.Slug = id, t.Slug
	return res, nil
}

func (a *app) handleImportar(w http.ResponseWriter, r *http.Request) {
	var p pacote
	if err := lerCorpo(r, &p); err != nil {
		a.responderErro(w, err)
		return
	}
	res, err := a.importar(r.Context(), p)
	if err != nil {
		var entrada erroDeEntrada
		if errors.As(err, &entrada) && strings.Contains(err.Error(), "já existe") {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, res)
}
