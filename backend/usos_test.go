package main

import (
	"context"
	"encoding/json"
	"testing"
)

func TestUsosDosArquivos(t *testing.T) {
	c := novoCliente(t)
	var res resultadoImportacao
	c.ok("POST", "/api/importar", map[string]any{
		"trilha": map[string]any{
			"titulo": "Curso X", "objetivo": "Assistir", "estado": "ativa",
			"modulos": []map[string]any{{
				"titulo": "Curso X",
				"etapas": []map[string]any{
					{"chave": "m1", "titulo": "Módulo 1", "objetivo": "Assistir", "tipo": "normal", "minutos": 3,
						"materiais": []map[string]any{
							{"tipo": "streaming", "papel": "principal", "titulo": "Aula 1", "referencia": "telegram:https://t.me/c/1#1"},
							{"tipo": "streaming", "papel": "principal", "titulo": "Aula 1, outro trecho", "referencia": "telegram:https://t.me/c/1#1"},
							{"tipo": "web", "papel": "apoio", "titulo": "Site", "referencia": "https://exemplo.com"},
						}},
				},
			}},
		},
	}, &res)

	// Um encaixe de curso esperando aprovação e um já aprovado (este não conta).
	ctx := context.Background()
	for i, estado := range []string{propAguardando, propAprovada} {
		p, err := c.a.novaProposta(ctx, res.TrilhaID, "@encaixe-"+string(rune('a'+i)), tipoEncaixe, "", 0, "Encaixar")
		if err != nil {
			t.Fatal(err)
		}
		m, _ := json.Marshal(material{Tipo: "curso", Titulo: "A + B", Referencia: "telegram:https://t.me/c/1::Curso X/Módulo 2 | telegram:https://t.me/c/2::Outro"})
		c.a.db.Exec(`INSERT INTO encaixes (proposta_id, material, foco) VALUES (?, ?, '')`, p.ID, string(m))
		c.a.db.Exec(`UPDATE propostas SET estado = ? WHERE id = ?`, estado, p.ID)
	}

	var u usosDosArquivos
	c.ok("GET", "/api/arquivos/usos", nil, &u)
	if len(u.Aulas) != 1 || u.Aulas[0].Ref != "telegram:https://t.me/c/1#1" || u.Aulas[0].Trilha != "Curso X" || u.Aulas[0].EtapaChave != "m1" || u.Aulas[0].Etapa != "Módulo 1" {
		t.Fatalf("aulas: %+v", u.Aulas)
	}
	if len(u.Encaixes) != 1 || len(u.Encaixes[0].Pastas) != 2 || u.Encaixes[0].Pastas[0] != "telegram:https://t.me/c/1::Curso X/Módulo 2" || u.Encaixes[0].Estado != propAguardando {
		t.Fatalf("encaixes: %+v", u.Encaixes)
	}
}
