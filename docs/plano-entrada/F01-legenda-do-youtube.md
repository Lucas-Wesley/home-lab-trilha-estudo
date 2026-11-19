# Fase 1 — A Trilha lê vídeos do YouTube (sem IA)

## Objetivo

Receber um link do YouTube (vídeo ou playlist) e guardar cada vídeo como uma **fonte**: título,
canal, duração, capítulos do autor e a **transcrição com os minutos**, lida da legenda pelo yt-dlp.
Nada de IA nesta fase. A tela é a fase 2; aqui é só o servidor e a API.

## Leia antes

- `README.md` desta pasta (regras de trabalho).
- `acervo/backend/youtube.go` (no repo do Acervo, pasta vizinha `../acervo/`): é o código que já faz
  isso no Acervo. Você vai **copiar e adaptar**, não chamar o Acervo.
  - `videoDoLink`, `videoDoYouTube`: tiram o id do vídeo do link.
  - `escolhaDeLegenda`: escolhe a legenda (manual antes da automática; idioma do vídeo).
  - `baixarLegenda`: `yt-dlp -J --skip-download --no-playlist <link>` (informações) e depois
    `yt-dlp --skip-download --no-playlist --write-subs|--write-auto-subs --sub-langs <l> --sub-format json3 -o <pasta>/legenda <link>`.
  - `legendaParaCru`: transforma o json3 em texto com `[mm:ss]` no começo de cada bloco.
  - `transcreverVideo`: vídeo sem legenda → baixa só o áudio e manda ao Whisper.
- `acervo/backend/testdata/yt-dlp-falso`: o yt-dlp falso dos testes de lá (copie a ideia).
- `acervo/Dockerfile`, linhas do `yt-dlp` e do `ffmpeg` (como instalar na imagem).
- Na Trilha: `backend/banco.go` (tabelas e a lista de `ALTER TABLE`), `backend/conteudo.go`
  (`gravar`, `ler`: o conteúdo fica no git local, fora do GitHub), `backend/servidor.go` (rotas),
  `backend/fila.go` (como uma fila roda em segundo plano).

## O que fazer

### 1.1 Imagem

No `Dockerfile` da Trilha (estágio final, `node:24-alpine`), instale `python3`, `py3-pip`, `ffmpeg` e
o `yt-dlp` do mesmo jeito que o Acervo (`pip install --break-system-packages -U "yt-dlp[default]"`,
com o `ARG YTDLP_EM` para o deploy sempre pegar a versão nova). Comente em português, como lá.

### 1.2 Banco

Tabela nova em `banco.go`:

```sql
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
```

A **transcrição não fica no banco**: vai para o conteúdo (git local) em
`entrada/<id>/transcricao.md`, pelo `a.conteudo.gravar(...)`. O mesmo vídeo duas vezes não duplica
(o `UNIQUE`): devolva a fonte que já existe.

### 1.3 Ler o vídeo (arquivo novo `backend/youtube.go`)

Copie do Acervo e adapte (nomes em português, sem as partes de Instagram e de Referência):

- `videoDoLink(bruto) (id string, err)`: aceita `youtube.com/watch?v=`, `youtu.be/`, `/shorts/`,
  `/live/`, `m.youtube.com`, com ou sem `&t=`.
- `ehPlaylist(bruto) bool`: link com `list=` e **sem** `v=`, ou `youtube.com/playlist?list=`.
  (Link com `v=` e `list=` é um vídeo só: o Lucas compartilhou um vídeo que estava numa playlist.)
- `a.lerVideo(ctx, fonte)`: roda `yt-dlp -J`, guarda título, canal, duração, idioma e **os
  capítulos** (`chapters` do JSON: `start_time`, `end_time`, `title`); baixa a legenda em json3 e
  monta a transcrição.
- **Blocos de 10 s** (no Acervo são 30 s): `const blocoMs = 10_000`. O minuto do começo de um
  fragmento depende disso.
- O cabeçalho da transcrição: título, canal, link, duração e, se houver, a lista de capítulos
  (`- 00:00 Introdução`). Depois, um bloco por linha: `[mm:ss] texto` (ou `[h:mm:ss]`).
- **Sem legenda nenhuma**: transcrever pelo Whisper como o Acervo faz (`transcreverVideo`): baixa só o
  áudio (`-f bestaudio`), manda ao `WHISPER_URL` (API da OpenAI, `POST /v1/audio/transcriptions`,
  `response_format=verbose_json` para ter os segmentos com tempo) e monta o mesmo formato
  `[mm:ss] texto`. Sem `WHISPER_URL`, a fonte fica em `erro` com "o vídeo não tem legenda e o
  Whisper não está configurado". O áudio não fica guardado.

### 1.4 Playlist

`a.expandirPlaylist(ctx, link)`: `yt-dlp --flat-playlist -J <link>` devolve os vídeos (`entries`:
`id`, `title`). Cria **uma fonte por vídeo** (estado `na_fila`, `playlist` = título da playlist).

### 1.5 A fila do yt-dlp

Uma goroutine (como a fila do Claude, mas **separada** dela: não usa crédito) que pega a fonte mais
antiga em `na_fila`, marca `lendo`, chama `lerVideo` e marca `pronta` ou `erro`. **Uma por vez e
com 15 s de pausa entre vídeos** (o YouTube bloqueia quem pede muito seguido). Ao iniciar o app, o
que estava `lendo` volta para `na_fila`. Um canal `acordar` como o da fila do Claude.

### 1.6 API (em `servidor.go`, arquivo novo `backend/entrada.go`)

| Rota | Faz |
|---|---|
| `POST /api/entrada` | `{"url": "..."}`. Vídeo: cria a fonte (ou devolve a existente). Playlist: cria uma por vídeo. Responde `202` com a lista de fontes criadas. Link que não é do YouTube: `400` com mensagem clara. |
| `GET /api/entrada` | Lista as fontes, da mais nova à mais antiga, sem a transcrição. |
| `GET /api/entrada/{id}` | A fonte com a `transcricao` (texto) e os `capitulos`. |
| `POST /api/entrada/{id}/tentar` | Fonte em `erro` volta para `na_fila`. |
| `DELETE /api/entrada/{id}` | Apaga a fonte e a pasta `entrada/<id>/` do conteúdo. |

## Testes obrigatórios (`backend/entrada_test.go`)

Com um `testdata/yt-dlp-falso` (shell, dados inventados) e a variável `YTDLP_BIN` apontando para ele:

1. `videoDoLink` com 8 formatos de link (incluindo `youtu.be/ID?t=30`, `shorts`, `live`) e 3 que
   não são do YouTube.
2. `ehPlaylist`: `playlist?list=`, `watch?v=X&list=Y` (não é), `watch?list=Y` (é).
3. Legenda json3 inventada → transcrição com `[00:00]`, `[00:10]`… em blocos de 10 s e os capítulos
   no cabeçalho.
4. `POST /api/entrada` com vídeo: a fila lê, a fonte fica `pronta`, `GET /api/entrada/{id}` traz a
   transcrição; mandar o mesmo vídeo de novo não cria outra.
5. `POST /api/entrada` com playlist de 3 vídeos inventados: 3 fontes com o título da playlist.
6. Vídeo sem legenda e sem `WHISPER_URL`: `erro` com a mensagem; com um Whisper falso (`httptest`)
   devolvendo segmentos: transcrição com os minutos.
7. A pausa entre vídeos tem que poder ser zero nos testes (variável ou campo do app).

## Não fazer

- Não chamar o Acervo. Não chamar o Claude. Não criar tela (é a fase 2).
- Não guardar o vídeo nem o áudio.

## Pronto quando

Testes passam (`go vet`, `go test`, `npm test`, `npm run build`), a decisão **T35 — Entrada: fontes
do YouTube** está em `docs/DECISOES.md` e a seção "Entrada (T35)" no `CLAUDE.md`.

Commit: `Entrada (T35): a Trilha lê vídeos e playlists do YouTube (legenda com os minutos, capítulos, Whisper sem legenda)`

## Dúvidas

## Feito
