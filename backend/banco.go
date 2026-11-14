package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func abrirBanco(caminho string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", caminho+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// O esquema cresce fase a fase (DECISOES.md §9). Na fase 1 só existem os ajustes: as instruções
// globais para a IA (T13) e as preferências da interface (tema).
const esquema = `
CREATE TABLE IF NOT EXISTS ajustes (
	chave         TEXT PRIMARY KEY,
	valor         TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
`
