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
