package main

import (
	"database/sql"
	"strings"

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
	for _, m := range migracoes {
		if _, err := db.Exec(m); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

// migracoes: colunas novas em tabelas que já existiam (a coluna repetida é ignorada).
var migracoes = []string{
	// Fase 6 (T27): o material escolhido na entrada C ("a partir de material").
	`ALTER TABLE criacoes ADD COLUMN material TEXT NOT NULL DEFAULT '[]'`,
	// Fase 7 (T28): o foco de uma etapa (o que a lição deve trabalhar; a etapa de reforço nasce
	// com ele) e o mini-teste de nível da criação (T9).
	`ALTER TABLE etapas ADD COLUMN foco TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE criacoes ADD COLUMN nivelamento TEXT NOT NULL DEFAULT ''`,
	// T34: o encaixe mandado a um tópico só (chave da etapa; vazio = a trilha inteira).
	`ALTER TABLE encaixes ADD COLUMN foco TEXT NOT NULL DEFAULT ''`,
	// T37: a fonte guarda a tarefa de fragmentar e o pedido do Lucas para a IA.
	`ALTER TABLE fontes ADD COLUMN tarefa_id INTEGER REFERENCES tarefas(id) ON DELETE SET NULL`,
	`ALTER TABLE fontes ADD COLUMN pedido TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE fontes ADD COLUMN notas TEXT NOT NULL DEFAULT ''`, // T37: as notas da IA sobre a divisão (não é erro)
	// T38: "sem destino" guarda quando foi marcado (a fase 6 o expira em 30 dias) e a fonte guarda
	// a tarefa de sugerir trilhas (separada da de fragmentar).
	`ALTER TABLE fragmentos ADD COLUMN sem_destino_em TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE fontes ADD COLUMN tarefa_destino INTEGER REFERENCES tarefas(id) ON DELETE SET NULL`,
	`ALTER TABLE fontes ADD COLUMN pedido_destino TEXT NOT NULL DEFAULT ''`, // o pedido do Lucas à tarefa destino
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

-- Fase 3 (T4, T13). Tudo o que a IA produz vira uma proposta versionada que espera o Lucas
-- (DECISOES.md §5). O alvo é a etapa pela chave (sobrevive a uma nova importação da trilha).
-- estado: gerando · aguardando · aprovada · descartada · substituida (veio uma versão nova) ·
-- falhou. conteudo: o JSON devolvido pela IA (ex.: {"licao": "...", "notas": "..."}).
CREATE TABLE IF NOT EXISTS propostas (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	trilha_id     INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave   TEXT NOT NULL DEFAULT '',
	tipo          TEXT NOT NULL,
	versao        INTEGER NOT NULL,
	anterior_id   INTEGER REFERENCES propostas(id) ON DELETE SET NULL,
	estado        TEXT NOT NULL,
	pedido        TEXT NOT NULL DEFAULT '',
	conteudo      TEXT NOT NULL DEFAULT '',
	erro          TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS propostas_alvo ON propostas (trilha_id, etapa_chave, tipo, versao);

-- Comentários numa proposta (§5.3): gerais (ancora vazia) ou ancorados numa seção
-- ("secao:<id do título>") ou num trecho selecionado (citacao). autor: lucas · ia; a resposta da
-- IA a um comentário fica na versão seguinte, com responde_a apontando para ele.
CREATE TABLE IF NOT EXISTS comentarios (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	proposta_id INTEGER NOT NULL REFERENCES propostas(id) ON DELETE CASCADE,
	ancora      TEXT NOT NULL DEFAULT '',
	citacao     TEXT NOT NULL DEFAULT '',
	autor       TEXT NOT NULL,
	texto       TEXT NOT NULL,
	responde_a  INTEGER REFERENCES comentarios(id) ON DELETE SET NULL,
	criado_em   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS comentarios_proposta ON comentarios (proposta_id, id);

-- Fila (T4): uma chamada ao Claude por vez. estado: na_fila · rodando · concluida · falhou ·
-- cancelada.
CREATE TABLE IF NOT EXISTS tarefas (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	tipo         TEXT NOT NULL,
	proposta_id  INTEGER REFERENCES propostas(id) ON DELETE CASCADE,
	titulo       TEXT NOT NULL DEFAULT '',
	estado       TEXT NOT NULL,
	erro         TEXT NOT NULL DEFAULT '',
	modelo       TEXT NOT NULL DEFAULT '',
	criado_em    TEXT NOT NULL,
	iniciado_em  TEXT NOT NULL DEFAULT '',
	terminado_em TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS tarefas_estado ON tarefas (estado, id);

-- Fase 4 (T25): a criação com IA de uma trilha (que nasce como rascunho). entrada: ia (entrevista
-- → mapa → plano) · lista (validação → plano). entrevista: rapida · padrao · profunda.
-- passo: entrevista · mapa · validacao · plano · pronta. rodadas: JSON das rodadas da entrevista
-- (perguntas e respostas); escolhas: JSON do recorte do mapa ou das sugestões aceitas.
-- Fase 5 (T26): cada vez que o Lucas faz os exercícios ou o simulado de uma etapa (pela chave,
-- como as propostas). tipo: exercicios (corrigidos na hora) · simulado (a IA corrige na fila).
-- estado: corrigida · corrigindo · falhou. nota de 0 a 100.
CREATE TABLE IF NOT EXISTS tentativas (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	trilha_id     INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave   TEXT NOT NULL,
	tipo          TEXT NOT NULL,
	estado        TEXT NOT NULL,
	nota          INTEGER NOT NULL DEFAULT 0,
	acertos       INTEGER NOT NULL DEFAULT 0,
	total         INTEGER NOT NULL DEFAULT 0,
	respostas     TEXT NOT NULL DEFAULT '[]',
	resultado     TEXT NOT NULL DEFAULT '',
	erro          TEXT NOT NULL DEFAULT '',
	tarefa_id     INTEGER REFERENCES tarefas(id) ON DELETE SET NULL,
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS tentativas_etapa ON tentativas (trilha_id, etapa_chave, id);
CREATE INDEX IF NOT EXISTS tentativas_tarefa ON tentativas (tarefa_id);

-- Fase 5: cada lição mandada ao Flashcards (POST /api/textos de lá); geracao_id = o pedido lá.
CREATE TABLE IF NOT EXISTS envios_flashcards (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	trilha_id   INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave TEXT NOT NULL,
	geracao_id  INTEGER NOT NULL DEFAULT 0,
	erro        TEXT NOT NULL DEFAULT '',
	criado_em   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS criacoes (
	trilha_id     INTEGER PRIMARY KEY REFERENCES trilhas(id) ON DELETE CASCADE,
	entrada       TEXT NOT NULL,
	tema          TEXT NOT NULL,
	pedido        TEXT NOT NULL DEFAULT '',
	entrevista    TEXT NOT NULL DEFAULT '',
	lista         TEXT NOT NULL DEFAULT '',
	passo         TEXT NOT NULL,
	rodadas       TEXT NOT NULL DEFAULT '[]',
	escolhas      TEXT NOT NULL DEFAULT '{}',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);

-- Fase 7 (T28): revisão espaçada das etapas concluídas, pela chave (como as tentativas). nivel =
-- o degrau na escada de intervalos (dominio.go); proxima = o dia da próxima revisão (AAAA-MM-DD,
-- no fuso do Lucas); ultima_nota = -1 enquanto não houve revisão.
CREATE TABLE IF NOT EXISTS revisoes (
	trilha_id     INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave   TEXT NOT NULL,
	nivel         INTEGER NOT NULL DEFAULT 0,
	proxima       TEXT NOT NULL,
	ultima        TEXT NOT NULL DEFAULT '',
	ultima_nota   INTEGER NOT NULL DEFAULT -1,
	PRIMARY KEY (trilha_id, etapa_chave)
);

-- T32 (encaixar material): o material de cada proposta de encaixe (a 1ª versão; os ajustes apontam para ela).
CREATE TABLE IF NOT EXISTS encaixes (
	proposta_id INTEGER PRIMARY KEY REFERENCES propostas(id) ON DELETE CASCADE,
	material    TEXT NOT NULL
);

-- T30 (videoaula): as notas que o Lucas toma assistindo e o resumo dele, pela chave da etapa.
CREATE TABLE IF NOT EXISTS anotacoes (
	trilha_id     INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave   TEXT NOT NULL,
	notas         TEXT NOT NULL DEFAULT '',
	resumo        TEXT NOT NULL DEFAULT '',
	atualizado_em TEXT NOT NULL,
	PRIMARY KEY (trilha_id, etapa_chave)
);

-- Entrada (T35): vídeos que chegaram para virar fragmentos nas trilhas.
CREATE TABLE IF NOT EXISTS fontes (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	tipo        TEXT NOT NULL,              -- youtube · acervo · telegram
	url         TEXT NOT NULL DEFAULT '',   -- o link do vídeo (youtube: https://www.youtube.com/watch?v=ID)
	video_id    TEXT NOT NULL DEFAULT '',   -- youtube: o id; acervo: o id da Referência; telegram: link#mensagem
	titulo      TEXT NOT NULL DEFAULT '',
	canal       TEXT NOT NULL DEFAULT '',
	duracao     INTEGER NOT NULL DEFAULT 0, -- segundos
	idioma      TEXT NOT NULL DEFAULT '',
	capitulos   TEXT NOT NULL DEFAULT '[]', -- [{"inicio":0,"fim":120,"titulo":"…"}] do autor do vídeo
	estado      TEXT NOT NULL DEFAULT 'na_fila', -- na_fila · lendo · pronta · erro (as fases 3+ acrescentam)
	erro        TEXT NOT NULL DEFAULT '',
	playlist    TEXT NOT NULL DEFAULT '',   -- o título da playlist de onde veio, se veio de uma
	criado_em   TEXT NOT NULL,
	UNIQUE (tipo, video_id)
);

-- Entrada (T37): os trechos por assunto de uma fonte — é neles que as trilhas se apoiam.
-- estado: sugerido (a IA propôs) · confirmado (o Lucas conferiu) · descartado. sugestao: fase 4.
CREATE TABLE IF NOT EXISTS fragmentos (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	fonte_id      INTEGER NOT NULL REFERENCES fontes(id) ON DELETE CASCADE,
	ordem         INTEGER NOT NULL,
	inicio        INTEGER NOT NULL,               -- segundos
	fim           INTEGER NOT NULL,               -- segundos (fim > inicio)
	titulo        TEXT NOT NULL,
	resumo        TEXT NOT NULL DEFAULT '',       -- 2 a 4 frases: o que o trecho ensina
	assuntos      TEXT NOT NULL DEFAULT '[]',     -- ["TLS", "certificados"]
	tipo          TEXT NOT NULL DEFAULT 'conteudo', -- conteudo · opiniao · fora
	estado        TEXT NOT NULL DEFAULT 'sugerido',
	sugestao      TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS fragmentos_fonte ON fragmentos (fonte_id, ordem);

-- Entrada (T38): as trilhas escolhidas para cada fragmento (pela IA, que já marca, ou pelo
-- Lucas). proposta_id é o encaixe criado no "Mandar para as trilhas"; NULL = ainda não enviado.
CREATE TABLE IF NOT EXISTS fragmento_destinos (
	fragmento_id INTEGER NOT NULL REFERENCES fragmentos(id) ON DELETE CASCADE,
	trilha_id    INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	proposta_id  INTEGER REFERENCES propostas(id) ON DELETE SET NULL,
	criado_em    TEXT NOT NULL,
	PRIMARY KEY (fragmento_id, trilha_id)
);

-- Entrada (T39): cada intervalo principal do vídeo que o Lucas viu até o fim na etapa.
CREATE TABLE IF NOT EXISTS trechos_vistos (
	trilha_id   INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave TEXT NOT NULL,
	referencia  TEXT NOT NULL, -- a url do vídeo
	localizador TEXT NOT NULL, -- o intervalo
	visto_em    TEXT NOT NULL,
	PRIMARY KEY (trilha_id, etapa_chave, referencia, localizador)
);
`
