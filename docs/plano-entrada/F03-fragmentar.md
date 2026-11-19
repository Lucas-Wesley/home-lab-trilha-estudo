# Fase 3 — A IA divide a fonte em fragmentos

## Objetivo

Uma fonte `pronta` (fase 1) vira **fragmentos**: trechos por assunto, com início, fim, título,
resumo, assuntos e tipo. É a **única** vez que a IA lê a transcrição inteira (D3). O Lucas vê os
fragmentos, ajusta os tempos, junta, separa ou descarta, e **confirma**.

## Leia antes

- `README.md` desta pasta; fases 1 e 2.
- `backend/fila.go` (a fila do Claude: `enfileirar`, `executores`, uma tarefa por vez).
- `backend/ia.go`: `chamar`/`pedidoClaude` (pasta da tarefa, regra, `esquema` JSON, `modelo`,
  `ferramentas`), `dadosDaResposta`, `lerRegra`, `regraConhecida`, `modeloPara` e os ajustes de
  modelo (`modelo_licao` etc.).
- **Tarefa sem proposta**: a correção do simulado (`backend/estudo.go`, `corrigirSimulado`) acha o
  que corrigir pela coluna `tentativas.tarefa_id`. Faça igual: a fonte guarda o id da tarefa.
- `backend/videoaula.go`: `conferirResumo` é outra tarefa sem proposta (bom modelo para copiar).
- `scripts/claude-falso.mjs`: reconhece o pedido pelo cabeçalho `# Pedido: …` e devolve dados de
  teste. Você vai acrescentar o caso novo.
- `frontend/src/Ajustes.vue` (página Modelos: os modelos recomendados).

## O que fazer

### 3.1 Banco

```sql
CREATE TABLE IF NOT EXISTS fragmentos (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	fonte_id    INTEGER NOT NULL REFERENCES fontes(id) ON DELETE CASCADE,
	ordem       INTEGER NOT NULL,
	inicio      INTEGER NOT NULL,          -- segundos
	fim         INTEGER NOT NULL,          -- segundos (fim > inicio)
	titulo      TEXT NOT NULL,
	resumo      TEXT NOT NULL DEFAULT '',  -- 2 a 4 frases: o que o trecho ensina
	assuntos    TEXT NOT NULL DEFAULT '[]',-- ["TLS", "certificados"]
	tipo        TEXT NOT NULL DEFAULT 'conteudo', -- conteudo · opiniao · fora
	estado      TEXT NOT NULL DEFAULT 'sugerido', -- sugerido · confirmado · descartado (a fase 4 acrescenta)
	sugestao    TEXT NOT NULL DEFAULT '',  -- fase 4
	criado_em   TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
```

E em `fontes` (pela lista de `ALTER TABLE`): `tarefa_id INTEGER` (a tarefa de fragmentar),
`pedido TEXT NOT NULL DEFAULT ''` (o pedido do Lucas para a IA). Estados novos da fonte:
`fragmentando` (tarefa na fila ou rodando) e `fragmentada` (há fragmentos para revisar ou já
confirmados). Se a tarefa falhar, a fonte volta para `pronta` com o `erro` preenchido.

### 3.2 Modelo novo em Ajustes

`modelo_entrada` ("Entrada: fragmentar e sugerir trilhas"), recomendado **Haiku**. Acrescente em
`ajustes` (`ia.go`), em `modeloPara` (tipos `fragmentar` e, na fase 4, `destino`; vazio = o padrão)
e na página Modelos de `Ajustes.vue`, com "(recomendado)" e no botão "Usar os recomendados".

### 3.3 A tarefa `fragmentar`

- `POST /api/entrada/{id}/fragmentar` com `{"pedido": "..."}` (opcional, texto livre do Lucas, ex.:
  "ignore a parte de perguntas do chat"). Só para fonte `pronta` ou `fragmentada`. Enfileira a tarefa
  `fragmentar` (título "Fragmentar: <título do vídeo>"), guarda `tarefa_id` e `pedido`, fonte vai a
  `fragmentando`. Fragmentar de novo **apaga os fragmentos `sugerido`** e mantém os já
  confirmados (manda-os no pedido para a IA não repetir).
- Executor `a.fragmentar(ctx, t)`: acha a fonte por `tarefa_id`; escreve na pasta da tarefa o
  `pedido.md` (cabeçalho `# Pedido: fragmentar`, título, canal, duração, capítulos do autor, o pedido
  do Lucas, os fragmentos já confirmados) e a `transcricao.md`; chama o Claude com a regra
  `fragmentar.md`, o esquema abaixo, o modelo `modeloPara(ctx, "fragmentar")` e só `Read`.
- Esquema da resposta:

```json
{"type":"object","properties":{
 "fragmentos":{"type":"array","items":{"type":"object","properties":{
   "inicio":{"type":"string","description":"mm:ss ou h:mm:ss, como na transcrição"},
   "fim":{"type":"string","description":"mm:ss ou h:mm:ss"},
   "titulo":{"type":"string"},
   "resumo":{"type":"string","description":"2 a 4 frases: o que o trecho ensina"},
   "assuntos":{"type":"array","items":{"type":"string"}},
   "tipo":{"type":"string","enum":["conteudo","opiniao","fora"]}},
   "required":["inicio","fim","titulo","resumo","assuntos","tipo"]}},
 "notas":{"type":"string"}},
 "required":["fragmentos","notas"]}
```

- O servidor **confere** antes de gravar: tempos válidos, `fim > inicio`, dentro da duração, em
  ordem, sem sobreposição (se dois se sobrepõem, o fim do primeiro vira o início do segundo); tira o
  que não passa e diz em `erro`/notas o que tirou (como `limparEncaixe` faz no encaixe). Grava como
  `sugerido`, fonte vai a `fragmentada`.

### 3.4 A regra `regras/fragmentar.md` (nova; acrescente em `regraConhecida`)

Escreva a regra em português, no estilo de `regras/encaixe.md`. Ela tem que dizer:

- Você recebe a transcrição de **um** vídeo com `[mm:ss]` e divide em **trechos por assunto** para
  estudo. Leia o pedido e a transcrição inteiros antes.
- Um fragmento = **um assunto** que dá para estudar sozinho. Vídeo curto sobre uma coisa só = **um
  fragmento** com o vídeo todo. Live que pula de assunto = vários.
- **Assunto que volta mais tarde**: um fragmento só, no trecho principal (o mais longo/completo);
  não picote em vários (decisão do Lucas).
- Mínimo de ~2 minutos por fragmento, salvo quando o assunto é mesmo curto e importante.
- `inicio` uns segundos **antes** de o assunto começar (o começo da frase que o introduz); `fim`
  quando o assunto acaba de verdade. Use os capítulos do autor como pista, não como verdade.
- `tipo`: `conteudo` (ensina algo técnico), `opiniao` (opinião, desabafo, história pessoal, recados,
  propaganda, perguntas do chat sem conteúdo), `fora` (assunto técnico, mas fora do tema do vídeo).
  Trechos de abertura, patrocínio e encerramento podem ficar de fora (não precisa cobrir 100% do vídeo).
- `titulo` curto e específico ("TLS 1.3: handshake em 1 RTT"), `resumo` dizendo **o que se aprende**,
  `assuntos` com 1 a 5 termos que ajudem a achar a trilha (ex.: "TCP", "AWS S3", "DDD agregados").
- Responda os comentários do Lucas, se houver (como nas outras regras).

### 3.5 Revisar e confirmar (API e tela)

| Rota | Faz |
|---|---|
| `GET /api/entrada/{id}` | passa a trazer `fragmentos` e a `tarefa` (estado, erro). |
| `PUT /api/fragmentos/{id}` | edita `inicio`, `fim` (aceita `mm:ss`), `titulo`, `resumo`, `assuntos`, `tipo`. Valida como em 3.3. |
| `POST /api/fragmentos/{id}/dividir` | `{"em":"12:30"}`: vira dois, o segundo com título "(continuação)". |
| `POST /api/fragmentos/{id}/juntar` | junta com o **seguinte** (início do primeiro, fim do segundo, títulos com " + "). |
| `POST /api/fragmentos/{id}/descartar` | estado `descartado` (some da lista, com "mostrar descartados"). |
| `POST /api/entrada/{id}/confirmar` | todos os `sugerido` da fonte viram `confirmado`. |

Na tela da fonte (fase 2), o espaço "Fragmentos" ganha:

- botão **"Dividir em fragmentos"** (com campo de pedido opcional), estado da tarefa (na fila,
  rodando, falhou com a mensagem e "Tentar de novo");
- a lista dos fragmentos: tempo `06:00–10:00` (clicável: abre o YouTube no início), duração, título,
  resumo, assuntos, etiqueta do tipo (`opinião` e `fora do tema` em tom mais fraco);
- editar (tempos e textos), dividir, juntar com o próximo, descartar;
- **"Confirmar fragmentos"** e **"Dividir de novo"** (com pedido).

## Testes obrigatórios

- Go: a tarefa `fragmentar` com o Claude falso → fragmentos gravados como `sugerido`, fonte
  `fragmentada`; a limpeza (fim antes do início, fora da duração, sobreposição); fragmentar de novo
  mantém os confirmados; editar/dividir/juntar/descartar/confirmar; tarefa que falha devolve a fonte
  a `pronta` com o erro.
- `modeloPara(ctx, "fragmentar")` usa `modelo_entrada`.
- `scripts/claude-falso.mjs`: o caso `# Pedido: fragmentar` devolve 3 fragmentos tirados dos
  `[mm:ss]` da transcrição (um `opiniao`).
- `regraConhecida("fragmentar.md")`.
- Frontend: teste da função que converte e valida tempos (`mm:ss` ↔ segundos), se você criar uma
  nova (prefira reusar `tempoParaSegundos`/`segundosParaTempo` de `videoaula.js`).

## Não fazer

- Não mandar fragmento para trilha (fase 4). Não usar o modelo da lição aqui.

## Pronto quando

Testes passam; decisão **T37 — Fragmentos da Entrada** e regra `fragmentar.md` documentadas; o
`CLAUDE.md` cita a tarefa e a regra.

Commit: `Entrada (T37): a IA divide a fonte em fragmentos por assunto (regra fragmentar.md, modelo_entrada); o Lucas ajusta e confirma`

## Dúvidas

## Feito

**Feito em 19/11/2026.**

- Banco (`backend/banco.go`): tabela `fragmentos` (com `sugestao`, para a fase 4) e os `ALTER
  TABLE` de `fontes` — `tarefa_id` (a tarefa de fragmentar) e `pedido` (o pedido do Lucas).
- Tarefa `fragmentar` (`backend/fragmentos.go`, registrada em `servidor.go`/`ligarIA`):
  `POST /api/entrada/{id}/fragmentar` aceita `{"pedido"}` opcional, só para fonte `pronta` ou
  `fragmentada`; enfileira "Fragmentar: <título>", grava `tarefa_id`/`pedido` e a fonte vai a
  `fragmentando`. O executor acha a fonte por `tarefa_id`, escreve `pedido.md`
  (`# Pedido: fragmentar` + vídeo, canal, duração, capítulos, pedido do Lucas, confirmados) e
  `transcricao.md` na pasta da tarefa e chama o Claude com `regras/fragmentar.md`, o esquema do
  plano, só `Read` e o modelo da tarefa (`modelo_entrada`). `limparFragmentos` confere antes de
  gravar: tempos legíveis, `fim > inicio`, dentro da duração (fim que passa fecha no fim do
  vídeo), ordem e sem sobreposição (o fim do primeiro vira o início do segundo); o que saiu vai
  para `fontes.erro` junto das `notas` da IA. Grava como `sugerido`; fonte a `fragmentada`.
  Fragmentar de novo apaga os `sugerido`, mantém os `confirmado` (listados no pedido para a IA
  não repetir) e reordena tudo pelo início.
- Falha/cancelamento: `aoTerminarTarefa` (`propostas.go`) devolve a fonte a `pronta` — ou a
  `fragmentada`, se já tinha fragmentos — com o erro da tarefa.
- Revisão (`fragmentos.go` + `Fonte.vue`): `GET /api/entrada/{id}` traz `fragmentos` e `tarefa`;
  `PUT /api/fragmentos/{id}` (tempos em `mm:ss` ou segundos, validados como na limpeza, com o
  cruzamento entre irmãos proibido), `POST .../dividir` (`{"em":"mm:ss"}`, o segundo é
  "(continuação)"), `.../juntar` (com o seguinte, títulos com " + "), `.../descartar` e
  `POST /api/entrada/{id}/confirmar`. Na tela: campo de pedido + "Dividir em
  fragmentos"/"Dividir de novo", estado da tarefa (na fila, dividindo, falhou com a mensagem e
  "Tentar de novo" pelo mesmo botão), a lista com `mm:ss–mm:ss` clicável, duração, etiqueta do
  tipo (`opinião`/`fora do tema` em tom fraco) e do estado, edição inline, dividir, juntar,
  descartar (com "mostrar descartados") e "Confirmar N fragmentos". `ESTADOS_FONTE` ganhou
  `fragmentando`/`fragmentada` e a lista da Entrada também autoatualiza durante a divisão.
- Modelo: `modelo_entrada` ("Entrada: fragmentar e sugerir trilhas") em `ajustes`,
  `chavesAjustes`, `modeloPara` (`fragmentar`; `destino` entra na fase 4) e a página Modelos de
  `Ajustes.vue`, com `(recomendado)` Haiku e no "Usar os recomendados".
- `scripts/claude-falso.mjs`: o caso `# Pedido: fragmentar` devolve 3 fragmentos tirados das
  marcas `[mm:ss]` da `transcricao.md` (o do meio `opiniao`).

**Testes** (`backend/fragmentos_test.go`): a tarefa com o Claude falso → 3 `sugerido`, fonte
`fragmentada`, `tarefa` no GET, `pedido.md`/`transcricao.md` conferidos; a limpeza (fim < inicio,
fora da duração, sem título, sobreposição, tipo inventado); re-fragmentar mantendo confirmados;
editar/dividir/juntar/descartar/confirmar (com os erros de validação); tarefa falha → fonte
`pronta` com o erro; `modeloPara(ctx, "fragmentar")` = `modelo_entrada`;
`regraConhecida("fragmentar.md")`. Tempos do front: reusei `tempoParaSegundos`/
`segundosParaTempo` (já testados em `entrada.test.js`/`videoaula.test.js`), sem função nova.

**Rodado e passou:** `cd backend && go vet ./... && go test ./...` (ok); `cd frontend && npm
test` (38 ok) `&& npm run build` (ok). Conferido no navegador a 360 px com o app de verdade
(yt-dlp e Claude falsos): fragmentar → `fragmentada` com 3 fragmentos, cartões, edição e
"Confirmar", sem rolagem horizontal.

**Ficou de fora:** mandar fragmento para trilha e as sugestões de destino (fase 4); a coluna
`sugestao` já está na tabela esperando.
