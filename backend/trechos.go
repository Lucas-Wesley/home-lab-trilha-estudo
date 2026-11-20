package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// Entrada, fase 5 (T39): o trecho do YouTube como aula da etapa — ver o intervalo até o fim
// grava o "visto"; com todos vistos e a etapa sem lição, ela conclui sozinha.

// trechoVisto: um intervalo principal do vídeo já visto até o fim (vai no GET da etapa).
type trechoVisto struct {
	Referencia  string `json:"referencia"`
	Localizador string `json:"localizador"`
	VistoEm     string `json:"visto_em"`
}

// mudarStatusEtapa: o mesmo UPDATE do handleStatusEtapa — quem mais conclui/começa a etapa
// (os trechos vistos) usa a mesma função.
func (a *app) mudarStatusEtapa(ctx context.Context, etapaID int64, status string) error {
	agora := agora()
	var iniciado, concluido string
	switch status {
	case "em_andamento":
		iniciado = agora
	case "concluida", "pulada":
		iniciado, concluido = agora, agora
	}
	res, err := a.db.ExecContext(ctx, `UPDATE etapas SET status = ?,
		iniciado_em = CASE WHEN ? = '' THEN '' WHEN iniciado_em = '' THEN ? ELSE iniciado_em END,
		concluido_em = ? WHERE id = ?`, status, iniciado, iniciado, concluido, etapaID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNaoEncontrou
	}
	return nil
}

// trechosPrincipais: os materiais de vídeo em trecho com papel `principal` da etapa —
// `youtube` (T39) e `streaming` com ref telegram: (T44: a aula toca no minuto do trecho).
func trechosPrincipais(e etapa) []material {
	var lista []material
	for _, m := range e.Materiais {
		if m.Papel != "principal" || m.Referencia == "" {
			continue
		}
		if m.Tipo == "youtube" || (m.Tipo == "streaming" && strings.HasPrefix(m.Referencia, "telegram:")) {
			lista = append(lista, m)
		}
	}
	return lista
}

// intervaloExato: o localizador mandado é um dos intervalos do material (exato — um pedaço
// inventado do intervalo não conta como trecho visto).
func intervaloExato(m material, localizador string) bool {
	mandados, err := lerIntervalos(m.Localizador)
	if err != nil {
		return false
	}
	xs, err := lerIntervalos(localizador)
	if err != nil || len(xs) != 1 {
		return false
	}
	for _, x := range mandados {
		if xs[0] == x {
			return true
		}
	}
	return false
}

// vistosDaEtapa: os trechos já vistos da etapa (para o GET marcar o ✓ nas partes).
func (a *app) vistosDaEtapa(ctx context.Context, e etapa) ([]trechoVisto, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT referencia, localizador, visto_em
		FROM trechos_vistos WHERE trilha_id = ? AND etapa_chave = ? ORDER BY visto_em`, e.TrilhaID, e.Chave)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	vistos := []trechoVisto{}
	for rows.Next() {
		var v trechoVisto
		if err := rows.Scan(&v.Referencia, &v.Localizador, &v.VistoEm); err != nil {
			return nil, err
		}
		vistos = append(vistos, v)
	}
	return vistos, rows.Err()
}

// totalDeTrechos: quantos intervalos principais a etapa tem (cada um é um "visto" esperado).
func totalDeTrechos(e etapa) int {
	total := 0
	for _, m := range trechosPrincipais(e) {
		if xs, err := lerIntervalos(m.Localizador); err == nil {
			total += len(xs)
		}
	}
	return total
}

// handleTrechoVisto: {"referencia","localizador"} marca o trecho visto; {"tocou":true} só diz
// que o vídeo começou — os dois põem a etapa `disponivel` em `em_andamento` no primeiro sinal.
func (a *app) handleTrechoVisto(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Referencia  string `json:"referencia"`
		Localizador string `json:"localizador"`
		Tocou       bool   `json:"tocou"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	e, err := a.etapaPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}

	if corpo.Tocou {
		// Só o andamento: tocar já é começar a etapa.
	} else {
		// O trecho tem que ser um intervalo exato de um material youtube principal da etapa.
		xs, err := lerIntervalos(corpo.Localizador)
		ok := false
		if err == nil && len(xs) == 1 {
			for _, m := range trechosPrincipais(e) {
				if m.Referencia == corpo.Referencia && intervaloExato(m, corpo.Localizador) {
					ok = true
					break
				}
			}
		}
		if !ok {
			a.responderErro(w, erroDeEntrada{errors.New("este trecho não é um material principal desta etapa")})
			return
		}
		if _, err := a.db.ExecContext(ctx, `INSERT OR IGNORE INTO trechos_vistos
			(trilha_id, etapa_chave, referencia, localizador, visto_em) VALUES (?, ?, ?, ?, ?)`,
			e.TrilhaID, e.Chave, corpo.Referencia, escreverIntervalos(xs), agora()); err != nil {
			a.responderErro(w, err)
			return
		}
	}

	if e.Status == "disponivel" {
		if err := a.mudarStatusEtapa(ctx, id, "em_andamento"); err != nil {
			a.responderErro(w, err)
			return
		}
	}

	// Todos os trechos vistos e sem lição aprovada (com lição, vale o critério de sempre): a
	// etapa conclui sozinha, pela mesma função do "Concluir".
	if !corpo.Tocou && !e.TemLicao {
		var vistos int
		if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trechos_vistos
			WHERE trilha_id = ? AND etapa_chave = ?`, e.TrilhaID, e.Chave).Scan(&vistos); err != nil {
			a.responderErro(w, err)
			return
		}
		if total := totalDeTrechos(e); total > 0 && vistos >= total {
			if err := a.mudarStatusEtapa(ctx, id, "concluida"); err != nil {
				a.responderErro(w, err)
				return
			}
		}
	}
	a.handleEtapa(w, r)
}
