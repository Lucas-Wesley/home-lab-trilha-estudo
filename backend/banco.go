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
	// Um escritor por vez: o SQLite serializa de qualquer jeito, e assim as transações não
	// esbarram umas nas outras.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// O esquema cresce fase a fase (DECISOES.md §9).
//
// Fase 2 (T19): a estrutura das trilhas e o progresso ficam aqui; o texto das lições e os
// documentos ficam em arquivos no conteúdo (conteudo.go), versionados no git local.
const esquema = `
CREATE TABLE IF NOT EXISTS ajustes (
	chave         TEXT PRIMARY KEY,
	valor         TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);

-- estado: rascunho (montando) · ativa (no Hoje) · pausada · concluida.
-- Meta (T10): dias da semana (0 = domingo, "1,2,3,4,5") e minutos por dia, ou a data final
-- (o app calcula os minutos). Profundidade 1 (panorama) a 5 (deep dive), PEDAGOGIA.md §3.
CREATE TABLE IF NOT EXISTS trilhas (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	slug          TEXT NOT NULL UNIQUE,
	titulo        TEXT NOT NULL,
	objetivo      TEXT NOT NULL DEFAULT '',
	descricao     TEXT NOT NULL DEFAULT '',
	recorte       TEXT NOT NULL DEFAULT '',
	instrucoes    TEXT NOT NULL DEFAULT '',
	nivel_partida TEXT NOT NULL DEFAULT '',
	nivel_alvo    TEXT NOT NULL DEFAULT '',
	profundidade  INTEGER NOT NULL DEFAULT 3,
	estado        TEXT NOT NULL DEFAULT 'rascunho',
	meta_dias     TEXT NOT NULL DEFAULT '',
	meta_minutos  INTEGER NOT NULL DEFAULT 0,
	meta_inicio   TEXT NOT NULL DEFAULT '',
	meta_fim      TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS modulos (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	trilha_id INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	ordem     INTEGER NOT NULL,
	titulo    TEXT NOT NULL,
	descricao TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS modulos_trilha ON modulos (trilha_id, ordem);

-- chave: identidade estável da etapa dentro da trilha (sobrevive a uma nova importação, que
-- preserva o progresso das etapas com a mesma chave). tipo: normal · base (etapa de base, T8).
-- profundidade 0 = a da trilha. status: disponivel · em_andamento · concluida · pulada.
CREATE TABLE IF NOT EXISTS etapas (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	trilha_id    INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	modulo_id    INTEGER NOT NULL REFERENCES modulos(id) ON DELETE CASCADE,
	ordem        INTEGER NOT NULL,
	chave        TEXT NOT NULL,
	titulo       TEXT NOT NULL,
	objetivo     TEXT NOT NULL DEFAULT '',
	bloom        TEXT NOT NULL DEFAULT '',
	profundidade INTEGER NOT NULL DEFAULT 0,
	tipo         TEXT NOT NULL DEFAULT 'normal',
	minutos      INTEGER NOT NULL DEFAULT 0,
	criterio     TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'disponivel',
	iniciado_em  TEXT NOT NULL DEFAULT '',
	concluido_em TEXT NOT NULL DEFAULT '',
	UNIQUE (trilha_id, chave)
);
CREATE INDEX IF NOT EXISTS etapas_modulo ON etapas (modulo_id, ordem);

CREATE TABLE IF NOT EXISTS etapa_requisitos (
	etapa_id INTEGER NOT NULL REFERENCES etapas(id) ON DELETE CASCADE,
	requer_id INTEGER NOT NULL REFERENCES etapas(id) ON DELETE CASCADE,
	PRIMARY KEY (etapa_id, requer_id)
);

-- Material de apoio (DECISOES.md §6). tipo: documento (referencia = slug do documento) · web
-- (URL) · livro · acervo · streaming. localizador: âncora, "cap. 3", "p. 45", "12:34".
-- papel: principal (a leitura da etapa) · apoio · aprofundar.
CREATE TABLE IF NOT EXISTS materiais (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	etapa_id    INTEGER NOT NULL REFERENCES etapas(id) ON DELETE CASCADE,
	ordem       INTEGER NOT NULL,
	tipo        TEXT NOT NULL,
	titulo      TEXT NOT NULL,
	referencia  TEXT NOT NULL DEFAULT '',
	localizador TEXT NOT NULL DEFAULT '',
	papel       TEXT NOT NULL DEFAULT 'apoio',
	nota        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS materiais_etapa ON materiais (etapa_id, ordem);

-- Cronograma (T10): gerado ao definir a meta e ao recalcular; uma linha por etapa e dia.
CREATE TABLE IF NOT EXISTS cronograma (
	trilha_id INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_id  INTEGER NOT NULL REFERENCES etapas(id) ON DELETE CASCADE,
	data      TEXT NOT NULL,
	minutos   INTEGER NOT NULL,
	PRIMARY KEY (etapa_id, data)
);
CREATE INDEX IF NOT EXISTS cronograma_data ON cronograma (trilha_id, data);
`
