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

**Feito em 20/11/2026.**

- Player (`frontend/src/TrechoYouTube.vue` + `trechoYoutube.js`, ambos novos): `trechos(materiais)`
  devolve um trecho por intervalo de material `youtube` `principal` (id do vídeo por
  `idDoVideo`, início/fim em segundos, título, `referencia` + `localizador` canônico — o
  material não tem id próprio, a chave do "visto" é essa dupla). A IFrame API carrega uma vez, só
  quando a etapa a usa; o player nasce com `start` = início, `rel: 0`, `modestbranding`,
  `playsinline` (host `youtube.com` — o `youtube-nocookie.com` não atende à IFrame API, que é
  quem dá `getCurrentTime`). Relógio de 500 ms só com o vídeo tocando: a barra cobre o trecho
  (0 % = início, 100 % = fim) com "faltam mm:ss"; antes do início o estado é mostrado sem mexer
  no vídeo; ao passar do fim − 2 s pela primeira vez vem o aviso "Trecho visto ✓ · Continuar
  assistindo · Ir para a próxima etapa" **sem parar o vídeo** e o POST de visto. Com mais de um
  trecho, "Parte N" com ✓ nos já vistos. Erro 101/150 ou API que não carrega: cartão com
  miniatura, intervalo, "Abrir no YouTube em <início>" e "Marquei como visto". Botões "Voltar ao
  início do trecho" e "Abrir no YouTube em <início>" (`youtu.be/<id>?t=`). As notas são as da
  Videoaula (mesmos endpoints `anotacoes`, debounce de 900 ms, "Marcar o minuto" e Ctrl/⌘+Enter
  com `getCurrentTime` deste player, marcas clicáveis que levam à parte e ao minuto).
- Servidor (`backend/trechos.go`, `banco.go`, `trilhas.go`, `servidor.go`): tabela
  `trechos_vistos` como no plano; `POST /api/etapas/{id}/trechos/visto` com
  `{"referencia","localizador"}` exige intervalo **exato** de material `youtube` `principal`
  (pedaço inventado, material apoio ou outro vídeo → 400) e grava o localizador canônico
  (`escreverIntervalos`), uma vez por trecho (`INSERT OR IGNORE`); `{"tocou":true}` na mesma
  rota é o "começou a tocar" — os dois põem `disponivel` em `em_andamento` no primeiro sinal.
  Todos os trechos principais vistos e sem lição aprovada (`TemLicao`): a etapa conclui pela
  **`mudarStatusEtapa`**, a função extraída do `handleStatusEtapa` (que passou a usá-la — mesmo
  UPDATE de status/`iniciado_em`/`concluido_em`); com lição, o critério de sempre segue valendo.
  `GET /api/etapas/{id}` devolve `trechos_vistos` para o ✓ das partes.
- Página (`telas/Etapa.vue`, `telas/Trilha.vue`): `TrechoYouTube` no topo quando há trecho
  principal (ao lado da `Videoaula`, sem tocar nela — testes dela intactos); a aba "Lição" fica
  "Texto da aula" e, sem lição, avisa "A aula desta etapa é o trecho do vídeo" com "Gerar lição
  escrita (opcional)"; o `ResumoEtapa`/`ir` das notas e da conferência falam com o player do
  YouTube também. Na estrutura, etapa com trecho e `minutos` vazio mostra ícone de vídeo +
  duração do trecho ("5 min") no lugar de "—".
- Cabeçalhos (5.4): o servidor Go não manda `Content-Security-Policy` nem `Referrer-Policy` —
  nada a ajustar aqui; o Caddy não foi tocado (outro repo).

**Testes** (`backend/trecho_test.go`, `frontend/src/trechoYoutube.test.js`): JS — `watch?v=`,
`youtu.be`, intervalo com hora, `estadoDoTrecho` antes/dentro/depois e `linkNoMinuto` (6
testes). Go — `{"tocou":true}` põe `em_andamento` com `iniciado_em`; o primeiro "visto" grava e
volta no GET; repetido não duplica; todos vistos sem lição → `concluida` com `concluido_em`;
intervalo fora do material, outro vídeo e pedaço do intervalo → 400; com lição gravada, o visto
não conclui.

**Rodado e passou:** `cd backend && go vet ./... && go test ./...` (ok); `cd frontend && npm
test` (44 ok, 6 novos) `&& npm run build` (ok). Navegador a 360 px com o binário embutido de
verdade, trilha importada por `POST /api/importar` com o material do exemplo
(`EOmzo5d0F9Y`, `13:33–18:25`): o iframe abriu com `start=813`, a barra andou ("faltam 04:44"),
o `tocou` pôs a etapa `em_andamento`; adiantando a 18:17, o aviso "Trecho visto ✓ · Continuar
assistindo" apareceu ao cruzar o fim − 2 s, o visto foi gravado (`13:33–18:25` canônico), a
etapa virou `concluida` e o vídeo **continuou tocando**. "Abrir no YouTube em 13:33" →
`youtu.be/EOmzo5d0F9Y?t=813`; "Marcar o minuto" escreveu `[13:33]`; a estrutura mostrou ícone +
"5 min"; `scrollWidth = clientWidth` (sem rolagem horizontal) nas duas telas.

**Ficou de fora:** o host `youtube-nocookie.com` (a IFrame API, necessária ao relógio e ao seek,
só embute `youtube.com`); posição parada enquanto o vídeo está pausado (o relógio é só com o
vídeo tocando, como pedido); vários trechos/partes foram testados nas funções e na API, mas só
um trecho foi olhado no navegador.
