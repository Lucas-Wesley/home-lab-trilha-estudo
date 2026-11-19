# Fase 5 — O trecho do YouTube toca na etapa

## Objetivo

Na etapa que tem um material `youtube` com papel `principal`, o vídeo **é a aula** (D5):

- O vídeo do YouTube aparece embutido no topo da etapa e **começa no início do trecho**.
- Uma barra mostra o trecho ("06:00 → 10:00, faltam 2:30") e onde o vídeo está.
- Ao chegar no fim do trecho, aparece o aviso **"Trecho visto ✓"** com "Ir para a próxima etapa", e
  o vídeo **continua tocando** (não para).
- Ver até o fim marca o trecho como visto; com todos os trechos principais da etapa vistos, a etapa
  é concluída sozinha (se não tiver outro critério, veja abaixo).
- As notas com o minuto (as de hoje, T30) funcionam também com o YouTube.

## Leia antes

- `README.md`; fase 4 (o material `youtube` e `lerIntervalos`).
- `frontend/src/Videoaula.vue` e `frontend/src/videoaula.js` (+ teste): o player do Streaming na
  etapa, as partes, "Marcar o minuto" e as notas. **Reaproveite** o máximo (notas, marcar minuto).
- A página da etapa (procure onde `Videoaula` é usado; as abas vêm de `AbasPagina.vue`).
- `backend/trilhas.go`: `handleStatusEtapa` e o que acontece ao concluir uma etapa (revisão espaçada
  em `dominio.go`, cronograma). Para concluir, **chame a mesma função** que esse handler usa.
- API do player do YouTube (IFrame Player API): `https://www.youtube.com/iframe_api`,
  `new YT.Player(el, { videoId, playerVars: { start, rel: 0, modestbranding: 1, playsinline: 1 } })`,
  `getCurrentTime()`, `seekTo()`, eventos `onReady`, `onStateChange`, `onError` (códigos 101 e 150 =
  o dono do vídeo não deixa embutir).

## O que fazer

### 5.1 Componente `frontend/src/TrechoYouTube.vue`

- Recebe a lista de trechos principais da etapa (cada um: id do vídeo, início, fim, título do
  material, id do material) — mais de um trecho = partes, como na `Videoaula`.
- Carrega a IFrame API uma vez (script em `<head>` só quando a etapa precisa) e cria o player com
  `start` = início. Use `https://www.youtube-nocookie.com` como host se funcionar; se não, o normal.
- A cada 500 ms (só enquanto toca), lê `getCurrentTime()`:
  - barra do trecho: início, fim, posição; "faltam m:ss";
  - passou do **fim − 2 s** pela primeira vez → aviso "Trecho visto ✓ · Continuar assistindo ·
    Próxima etapa" e chama a API de "visto" (uma vez);
  - antes do início (o Lucas voltou): só mostra, não força nada.
- Botões: "Voltar ao início do trecho", "Abrir no YouTube em 6:00" (`https://youtu.be/<id>?t=360`).
- **Vídeo que não deixa embutir** (`onError` 101/150) ou sem internet: no lugar do player, um cartão
  com a miniatura, o trecho e "Abrir no YouTube em 6:00", e um botão "Marquei como visto" (o Lucas
  viu lá fora).
- "Marcar o minuto" das notas: usa `getCurrentTime()` deste player (com várias partes, a parte).
- Funções puras em `frontend/src/trechoYoutube.js` (com teste): `idDoVideo(url)`, `trechos(materiais)`
  (os `youtube` principais, já com início/fim em segundos), `estadoDoTrecho(posicao, inicio, fim)` →
  `{ antes, dentro, passou, faltam }`, `linkNoMinuto(id, segundos)`.

### 5.2 Visto e progresso (servidor)

```sql
CREATE TABLE IF NOT EXISTS trechos_vistos (
	trilha_id   INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	etapa_chave TEXT NOT NULL,
	referencia  TEXT NOT NULL, -- a url do vídeo
	localizador TEXT NOT NULL, -- o intervalo
	visto_em    TEXT NOT NULL,
	PRIMARY KEY (trilha_id, etapa_chave, referencia, localizador)
);
```

- `POST /api/etapas/{id}/trechos/visto` `{"referencia","localizador"}`: grava. Etapa `disponivel`
  vira `em_andamento` no primeiro trecho. Se **todos** os trechos `youtube` principais da etapa
  estão vistos **e** a etapa não tem lição aprovada (com lição, vale o critério de sempre: os
  exercícios), a etapa vira `concluida` pela mesma função do `handleStatusEtapa`.
- `GET /api/etapas/{id}` passa a trazer os trechos vistos (para a tela marcar ✓ nas partes).
- Tocar (o primeiro `onStateChange` = tocando) também marca a etapa `em_andamento` (pode ser na
  mesma rota com `{"tocou":true}` ou numa rota pequena; escolha uma e documente).

### 5.3 A página da etapa

- Com trecho `youtube` principal: o `TrechoYouTube` no topo, como a `Videoaula`; a aba "Lição" diz
  "A aula desta etapa é o trecho do vídeo" e deixa **"Gerar lição escrita (opcional)"** como botão
  secundário (o que já existe).
- Sem trecho `youtube`: nada muda.
- Na estrutura da trilha (lista de etapas), a etapa com trecho mostra um ícone de vídeo e a duração
  do trecho no lugar dos minutos, se os minutos estiverem vazios.

### 5.4 Caddy / cabeçalhos

Confira se o servidor ou o Caddy mandam algum cabeçalho que bloqueie iframe de terceiros
(`Content-Security-Policy` com `frame-src`) ou `Referrer-Policy: no-referrer` (o YouTube recusa o
embed sem referrer: erro 153). Se o Go mandar, ajuste para `Referrer-Policy:
strict-origin-when-cross-origin`. **Não** edite o Caddy (é outro repo): se o problema for lá,
escreva em "Dúvidas".

## Testes obrigatórios

- `trechoYoutube.test.js`: as funções puras (link `watch?v=`, `youtu.be`, intervalo com hora,
  `estadoDoTrecho` nas três situações).
- Go: `POST .../trechos/visto` marca, coloca em andamento, conclui com todos vistos e sem lição; com
  lição aprovada não conclui; trecho de material que não existe na etapa → `400`.
- Olhe no navegador (pode ser `npm run dev` com uma etapa de teste importada por `POST /api/importar`
  com um material `youtube` de um vídeo público qualquer): toca no minuto certo, a barra anda, o
  aviso aparece no fim, "Abrir no YouTube" abre no minuto. Diga no "Feito" o que conferiu.

## Não fazer

- Não baixar vídeo do YouTube. Não parar o vídeo no fim do trecho.
- Não mexer na `Videoaula` do Streaming além de extrair o que for comum (se extrair, mantenha os
  testes dela passando).

## Pronto quando

Testes passam; decisão **T39 — Trecho do YouTube como aula da etapa**; `CLAUDE.md` atualizado.

Commit: `Entrada (T39): o trecho do YouTube toca na etapa como aula, com aviso no fim do trecho e progresso`

## Dúvidas

## Feito
