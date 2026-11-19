# Fase 8 — Criar uma trilha igual a um curso do Telegram (sem IA)

## Objetivo

Em **Meus arquivos**, numa pasta de curso (ex.: "Go Expert-Full Cycle-2023", ou "Elemar JR › 4. DDD
do Jeito Certo"), o botão **"Criar trilha igual a este curso"** monta uma trilha com a **mesma
estrutura** do curso, **sem IA e sem baixar nada**. Cada etapa aponta para a(s) aula(s) do Telegram.
Na etapa, a aula que ainda não está no servidor mostra **"Baixar esta aula (230 MB)"**: o Lucas toca,
o Streaming baixa e o vídeo toca ali mesmo quando chega (D9: só por clique).

## Leia antes

- `README.md`; fase 7.
- `backend/importar.go` (o `pacote` do `POST /api/importar`: trilha → módulos → etapas com
  materiais; `validarPacote`) — você vai **montar um pacote** e usar a mesma importação.
- `backend/videoaula.go` e `frontend/src/Videoaula.vue` (o vídeo do Streaming na etapa, com partes:
  vários materiais `streaming` principais, em ordem).
- `frontend/src/pastas.js` (copiado na fase 7): a árvore e `oQueFalta`.
- `frontend/src/slug.js` (chaves das etapas).

## O que fazer

### 8.1 Referência de aula do Telegram

- Material `tipo = "streaming"`, `papel = "principal"`, `titulo` = nome da aula e
  `referencia = "telegram:<link do grupo>#<mensagem>"` (ex.: `telegram:https://t.me/c/123#45` —
  nos testes, só links inventados). Essa referência **não muda** quando a aula é baixada, apagada ou
  baixada de novo (o id no Streaming muda; a mensagem não).
- `GET /api/arquivos/aula?ref=telegram:...` (servidor da Trilha): procura a mensagem nos `salvos` do
  `GET /api/meus-videos` do Streaming e devolve `{estado: "baixada"|"na_fila"|"no_telegram",
  video: <id>, url: STREAMING_LINK/api/video/proprio/<id>.mp4, tamanho, duracao, nome}`. Mensagem que
  não está em nenhum salvo: `404` "aula não encontrada no Streaming".
- `POST /api/arquivos/aula/baixar` `{ref}` → `POST /api/arquivos/baixar` (fase 7) com essa mensagem só.
- Na `Videoaula.vue`: material `streaming` com `referencia` começando por `telegram:` → pergunta o
  estado. `baixada` → toca como hoje (pela `url`). `no_telegram` → cartão "Esta aula ainda não está no
  servidor · 230 MB · ~1 min" + **"Baixar esta aula"** (e, com várias partes, **"Baixar as N partes"**).
  `na_fila` → "Baixando…", pergunta de novo a cada 5 s e toca sozinho quando ficar `baixada`.
  Função pura `ehAulaDoTelegram(ref)` e `tempoEstimado(bytes)` (4 MB/s) em `videoaula.js`, com teste.

### 8.2 Montar a trilha a partir da pasta

Na tela da pasta, **"Criar trilha igual a este curso"** abre um diálogo com:

- **Título** (padrão: o nome da pasta; para pasta de grupo com um curso só, o nome do grupo).
- **Como dividir** (duas opções, com a contagem de cada uma na hora):
  - **"Uma etapa por aula"**: cada pasta que tem aulas vira um **módulo** (título = o caminho a
    partir da pasta escolhida, com ` › `); cada aula vira uma **etapa** com 1 material.
  - **"Uma etapa por pasta"**: cada pasta que tem aulas vira uma **etapa** (as aulas são as partes,
    em ordem); a pasta de cima dela vira o **módulo** (se a pasta de aulas é filha direta da escolhida,
    o módulo é a própria pasta escolhida).
  - Aulas soltas direto na pasta escolhida: um módulo com o nome dela.
  - Mostre algo como: "24 módulos · 300 etapas · 55 h" e "1 módulo · 24 etapas · 55 h".
  - Recomende "uma etapa por pasta" quando as aulas são curtas (média < 15 min) e "uma etapa por
    aula" quando são longas.
- **"Criar"**: monta o **pacote** no frontend (função pura `pacoteDoCurso(pasta, modo, titulo)` em
  `frontend/src/cursoParaTrilha.js`, com teste) e manda para `POST /api/importar`:
  - etapa: `chave` única (slug do caminho + nome, com sufixo se repetir), `titulo` = nome da aula
    (ou da pasta), `objetivo` = `"Assistir à aula: <nome>"` (ou `"Assistir às aulas de <pasta>"`),
    `minutos` = duração somada arredondada para cima, `tipo = "normal"`, sem requisitos (a ordem já
    é a do curso), sem lição;
  - materiais: as aulas da etapa, na ordem, como em 8.1;
  - trilha: `estado = "ativa"`, descrição dizendo de que curso veio (nome do grupo e da pasta).
- Depois de criar, abre a trilha.

### 8.3 Progresso

A aula do Telegram na etapa conta como a `Videoaula` já conta hoje (T30). Não invente regra nova
aqui; se a `Videoaula` não marca nada sozinha, deixe igual e diga no "Feito".

## Testes obrigatórios

- `cursoParaTrilha.test.js`: uma árvore inventada (grupo › curso › 2 módulos com 3 aulas, mais uma
  aula solta) nos dois modos: módulos, etapas, partes, minutos, chaves únicas, referências
  `telegram:`; o pacote passa nas mesmas regras de `validarPacote` (copie as regras que der para
  conferir no JS, ou faça um teste Go que importa um pacote igual).
- Go: `GET /api/arquivos/aula` com um Streaming falso: baixada (com url), na fila (há pedido com a
  mensagem), no Telegram, não encontrada; `POST /api/arquivos/aula/baixar` repassa a mensagem certa.
- Go: importar um pacote com materiais `telegram:` funciona (o tipo `streaming` já é aceito; a
  referência não precisa ser URL).
- `videoaula.test.js`: `ehAulaDoTelegram`, `tempoEstimado`.

## Não fazer

- Não baixar nada ao criar a trilha. Não usar IA. Não mudar o Streaming.

## Pronto quando

Testes passam; decisão **T42 — Trilha igual ao curso do Telegram e aula baixada por clique**.

Commit: `Entrada (T42): criar trilha igual a um curso do Telegram (uma etapa por aula ou por pasta); a aula baixa na etapa, por clique`

## Dúvidas

## Feito
