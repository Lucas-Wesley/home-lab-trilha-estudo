# Fase 10 — Processar aulas do Telegram (transcrever e fragmentar como um vídeo do YouTube)

## Objetivo

Para aulas que precisam ser **ouvidas** (título que não diz nada, ou quando o Lucas quer o conteúdo
a fundo), **"Processar"** faz: baixa a aula (se não estiver baixada), **transcreve pelo Whisper do
Mac** (sem crédito do Claude) e cria uma **fonte** `telegram` na Entrada. Daí em diante é o mesmo
caminho do YouTube: fragmentar (fase 3), destino (fase 4), encaixe. Na etapa, o trecho de uma aula do
Telegram toca pelo player do Streaming, no minuto do trecho.

## Leia antes

- `README.md`; fases 1 (Whisper e o formato `[mm:ss]` da transcrição), 3, 4, 5, 8 e 9.
- `backend/youtube.go` da fase 1 (a parte do Whisper: reaproveite a função que manda áudio ao
  `WHISPER_URL` e monta a transcrição com os minutos).
- `frontend/src/Videoaula.vue` (toca o vídeo do Streaming; já sabe começar em um minuto pelas notas).

## O que fazer

### 10.1 Processar

- `POST /api/entrada/telegram` `{"refs": ["telegram:https://t.me/c/1#45", ...]}` (uma ou várias
  aulas; da tela de Meus arquivos — pasta inteira ou aula — e do botão "Processar estas aulas" da
  fase 9). Para cada uma, cria a fonte `tipo = "telegram"`, `video_id` = a ref, `titulo` = nome da
  aula, `canal` = o caminho (curso › módulo), estado `na_fila`. A mesma ref de novo devolve a fonte
  existente.
- A fila da fase 1 (a do yt-dlp, **não** a do Claude) ganha o caso `telegram`:
  1. pergunta o estado da aula (`GET /api/arquivos/aula`, fase 8); `no_telegram` → pede o download
     (`POST /api/arquivos/aula/baixar`) e deixa a fonte em `baixando` (estado novo); a fila passa para
     a próxima fonte e volta a esta depois (não fique parado esperando um download de 1 GB);
  2. `baixada` → extrai o áudio com o ffmpeg direto da url do vídeo no Streaming (rede interna:
     troque `STREAMING_LINK` por `STREAMING_URL` na url), em mono 16 kHz (`-vn -ac 1 -ar 16000`),
     em pedaços de 10 minutos se o Whisper precisar (veja como o Acervo corta podcasts: `acervo/backend/podcast.go`);
  3. manda ao `WHISPER_URL` e monta `entrada/<id>/transcricao.md` com `[mm:ss]` (somando o começo de
     cada pedaço);
  4. fonte `pronta`. O áudio é apagado.
- Sem `WHISPER_URL`: `erro` "o Whisper não está configurado". Whisper fora do ar: `erro` com
  "Tentar de novo".
- Opção na tela ao processar: **"Dividir em fragmentos quando terminar"** (marcada por padrão):
  quando a fonte fica `pronta`, enfileira a tarefa `fragmentar` (fase 3) sozinha.

### 10.2 Trecho de aula do Telegram na etapa

- No encaixe (fase 4), uma fonte `telegram` vira material `tipo = "streaming"`, `referencia` = a ref
  `telegram:...#<mensagem>`, `localizador` = o intervalo `mm:ss–mm:ss` (em vez do material
  `youtube`). `arquivosDoMaterial` com `streaming` + `telegram:` + intervalo lê o trecho da
  transcrição da fonte (como o `youtube` da fase 4).
- Na `Videoaula.vue`: material `streaming` com `localizador` de intervalo → começa no início do
  trecho e mostra a mesma barra/aviso de fim do `TrechoYouTube` (fase 5). **Extraia** para um lugar
  comum o que for igual (a barra, `estadoDoTrecho`, o aviso), sem duplicar. O "visto" e o progresso
  usam a mesma rota da fase 5 (`/trechos/visto`).

### 10.3 Tela

- Em **Meus arquivos**: **"Processar"** na pasta (todas as aulas dela, com a contagem e o tempo de
  áudio) e em cada aula; aulas já processadas mostram "na Entrada" com link para a fonte.
- Na lista da Entrada, as fontes `telegram` com um ícone próprio e os estados `baixando`,
  `transcrevendo`.
- O botão "Processar estas aulas" da fase 9 passa a funcionar.

## Testes obrigatórios

- Go, com Streaming falso, Whisper falso (`httptest`) e um ffmpeg falso (script em `testdata/`, como
  o yt-dlp falso): aula baixada → fonte `pronta` com a transcrição e os minutos somados entre os
  pedaços; aula no Telegram → pede o download, fonte `baixando`, a fila segue e volta; sem Whisper →
  erro; "fragmentar quando terminar" enfileira a tarefa.
- Encaixe de fonte `telegram`: material `streaming` com `telegram:` e intervalo; o pedido traz só o
  trecho.
- `videoaula.test.js`/`trechoYoutube.test.js`: o que foi extraído para o comum continua testado.

## Não fazer

- Não usar o Claude para transcrever. Não baixar nada sem o clique em "Processar".
- Não mudar o Streaming nem o Whisper do Mac.

## Pronto quando

Testes passam; decisão **T44 — Processar aulas do Telegram (Whisper → fragmentos)**; `CLAUDE.md`
atualizado; e uma revisão final: o `README.md` desta pasta com cada fase marcada como feita.

Commit: `Entrada (T44): processar aulas do Telegram (baixa, transcreve no Whisper do Mac e fragmenta como um vídeo do YouTube)`

## Dúvidas

## Feito

**Backend** (`backend/telegram.go` + fiação em `servidor.go`, `main.go`, `banco.go`, `entrada.go`)
- `POST /api/entrada/telegram` `{"refs": [...], "fragmentar": true}` valida cada ref
  `telegram:<link>#<msg>` (`referenciaTelegram`), acha a aula no catálogo do Streaming e cria a
  fonte `tipo = "telegram"` (`video_id` = a ref, `titulo` = nome da aula, `canal` = grupo › curso ›
  módulo, `duracao`, `estado = "na_fila"`). Ref repetida devolve a fonte existente (e honra o
  `fragmentar` novo se ainda está em processamento). `fontes` ganhou a coluna `fragmentar`.
- `lerTelegram` na fila da Entrada: `no_telegram` → `baixarAulaStreaming` pede só aquela mensagem e
  devolve estado `baixando` — a fila passa para a próxima fonte e a sonda `promoverBaixando` (a
  cada `sondaTelegram` = 15 s; 0 nos testes) a traz de volta à `na_fila`; `na_fila` idem; `baixada`
  → `transcrevendo` e transcreve. `recuperarEntrada` devolve `transcrevendo` à fila ao reiniciar.
- `transcreverAulaTelegram`: ffmpeg (`FFMPEG_BIN`, padrão `ffmpeg`) lê o vídeo da url **interna**
  `STREAMING_URL/api/video/proprio/<id>.mp4`, `-vn -ac 1 -ar 16000 -f segment -segment_time 600
  -c:a flac` (pedaços de 10 min, como o `podcast.go` do Acervo); um `pedirTranscricao` por pedaço
  (`WHISPER_URL`, `verbose_json`), somando `i * 600` nos `[mm:ss]`; escreve
  `entrada/<id>/transcricao.md` e apaga o temporário. Sem `WHISPER_URL` → "o Whisper não está
  configurado"; falha do Whisper → fonte `erro` com "Tentar de novo" (`/tentar` já existia).
- `pronta` + `fragmentar` → `pedirFragmentar` (a mesma tarefa do botão da página da fonte).
- Encaixe/etapa: `enviar` cria material **`streaming`** (`referencia` = ref telegram:, `localizador`
  = intervalos) para fonte `telegram`; `trechosDoVideo` acha a fonte pelo `video_id` (url só para
  youtube/acervo); `arquivosDoMaterial` lê só o trecho para `streaming`+`telegram:`; `criarEncaixe`
  aceita `streaming` com ref telegram: e trilha vazia; `trechosPrincipais` conta o streaming+telegram
  → "visto" e auto-conclusão idênticos ao do YouTube (T39).

**Frontend**
- `BarraTrecho.vue` novo: o comum da `Videoaula` e do `TrechoYouTube` — título + localizador, barra
  de progresso da janela do trecho, "faltam…", aviso "Trecho visto ✓" e "Ir para a próxima etapa".
  `trechoYoutube.js` ganhou `pctDoTrecho` e `intervaloDoMaterial` (ref telegram: + intervalo →
  `{inicio, fim, localizador, referencia}`); `TrechoYouTube.vue` usa o componente e seus testes
  seguem passando.
- `Videoaula.vue`: material `streaming`+`telegram:` com intervalo toca o vídeo baixado começando no
  início do trecho (posição salva só vale dentro dele), mostra a `BarraTrecho`, "Voltar ao início
  do trecho" e grava o "visto"/`tocou` na rota da fase 5. Aulas sem intervalo seguem videoaula
  comum (T30/T42 inalterados).
- Meus arquivos: "Processar na Entrada" na pasta e "Processar" por aula — diálogo com a contagem,
  `~tempo` de áudio e "Dividir em fragmentos quando terminar" marcado; aulas já processadas
  mostram "na Entrada" com link para a fonte (`aulasDaPasta`/`referenciaDaAula` em `cursos.js`;
  o mapa `ref → fonte` vem do `GET /api/entrada` por `video_id`).
- `EncaixarMaterial.vue`: "Processar estas aulas" manda as refs dos números do bloco ao mesmo
  endpoint (fragmentar ligado) e vira etiqueta "na Entrada — veja a aba Vídeos".
- `ListaEntrada.vue`: ícone `telegram` (novo em `Icone.vue`) no lugar da miniatura, estados
  `baixando`/`transcrevendo` em `rotulos.js` e no timer; `Fonte.vue` esconde os links "no minuto"
  do YouTube para fonte telegram (o `videoId` fica vazio → "aula do Telegram" no lugar de "Abrir no
  YouTube", marcas `[mm:ss]` viram texto, intervalos dos fragmentos idem).

**Testes** — `telegram_test.go` (Streaming + Whisper + `testdata/ffmpeg-falso`, todos falsos):
baixada → `pronta` com transcrição e `[mm:ss]` somados entre pedaços e flags do ffmpeg conferidos;
`no_telegram` → download pedido, fonte `baixando`, a fila segue e a sonda a reprocessa; sem Whisper
→ erro; `fragmentar` enfileira a tarefa ao ficar pronta. `cursos_test.go` ganhou o encaixe de fonte
`telegram` (material `streaming` + intervalo, pedido só com o trecho) e `trecho_test.go` o "visto"
do trecho streaming que conclui a etapa. `cursos.test.js`: `aulasDaPasta`/`referenciaDaAula`;
`trechoYoutube.test.js`: `intervaloDoMaterial`/`pctDoTrecho`. Suítes antigas intactas.

**Verificação** — `go vet` + `go test` ok; `npm test` 71 ok; `npm run build` ok. Ao vivo a 360 px
com Streaming/Whisper/ffmpeg falsos: "3 aulas · ~06:00 de áudio" no diálogo com a caixa marcada, as
três fontes aparecendo na Entrada (`pronta`/`baixando`/`na_fila`), o `[10:00]` somado no 2º pedaço,
"na Entrada" nas aulas, página da fonte sem link YouTube e a etapa com trecho do Telegram
mostrando a `BarraTrecho` igual à do YouTube. Sem rolagem horizontal.
