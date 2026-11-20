package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Onde as aulas do Telegram já estão (Meus arquivos mostra "na trilha X" na aula e na pasta):
// os materiais com referência telegram: (trilha igual ao curso, encaixe pelos títulos, trecho de
// aula processada) e os encaixes de cursos ainda esperando a aprovação do Lucas.

type usoDeAula struct {
	Ref        string `json:"ref"` // telegram:<link>#<mensagem>
	TrilhaID   int64  `json:"trilha_id"`
	Trilha     string `json:"trilha"`
	EtapaID    int64  `json:"etapa_id"`
	EtapaChave string `json:"etapa_chave"`
	Etapa      string `json:"etapa"`
}

type encaixePendente struct {
	Pastas   []string `json:"pastas"` // telegram:<link>::<caminho>, como no material curso
	TrilhaID int64    `json:"trilha_id"`
	Trilha   string   `json:"trilha"`
	Estado   string   `json:"estado"` // gerando · aguardando
}

type usosDosArquivos struct {
	Aulas    []usoDeAula       `json:"aulas"`
	Encaixes []encaixePendente `json:"encaixes"`
}

func (a *app) handleUsosArquivos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := usosDosArquivos{Aulas: []usoDeAula{}, Encaixes: []encaixePendente{}}
	rows, err := a.db.QueryContext(ctx, `
		SELECT m.referencia, t.id, t.titulo, e.id, e.chave, e.titulo
		FROM materiais m JOIN etapas e ON e.id = m.etapa_id JOIN trilhas t ON t.id = e.trilha_id
		WHERE m.referencia LIKE 'telegram:%'
		ORDER BY t.id, e.ordem`)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer rows.Close()
	vistos := map[string]bool{}
	for rows.Next() {
		var x usoDeAula
		if err := rows.Scan(&x.Ref, &x.TrilhaID, &x.Trilha, &x.EtapaID, &x.EtapaChave, &x.Etapa); err != nil {
			erroInterno(w, err)
			return
		}
		// A mesma aula em vários trechos da mesma etapa conta uma vez.
		chave := x.Ref + "\x00" + x.EtapaChave + "\x00" + x.Trilha
		if vistos[chave] {
			continue
		}
		vistos[chave] = true
		u.Aulas = append(u.Aulas, x)
	}
	if err := rows.Err(); err != nil {
		erroInterno(w, err)
		return
	}

	pend, err := a.db.QueryContext(ctx, `
		SELECT x.material, t.id, t.titulo, p.estado
		FROM encaixes x JOIN propostas p ON p.id = x.proposta_id JOIN trilhas t ON t.id = p.trilha_id
		WHERE p.estado IN (?, ?)
		ORDER BY p.id`, propGerando, propAguardando)
	if err != nil {
		erroInterno(w, err)
		return
	}
	defer pend.Close()
	for pend.Next() {
		var bruto string
		var x encaixePendente
		if err := pend.Scan(&bruto, &x.TrilhaID, &x.Trilha, &x.Estado); err != nil {
			erroInterno(w, err)
			return
		}
		var m material
		if json.Unmarshal([]byte(bruto), &m) != nil || m.Tipo != "curso" {
			continue
		}
		for _, p := range strings.Split(m.Referencia, " | ") {
			if p = strings.TrimSpace(p); p != "" {
				x.Pastas = append(x.Pastas, p)
			}
		}
		u.Encaixes = append(u.Encaixes, x)
	}
	if err := pend.Err(); err != nil {
		erroInterno(w, err)
		return
	}
	responderJSON(w, http.StatusOK, u)
}
