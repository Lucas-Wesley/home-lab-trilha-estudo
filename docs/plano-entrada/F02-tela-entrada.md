# Fase 2 — Tela Entrada (sem IA)

## Objetivo

O Lucas abre a Trilha, toca em **Entrada**, cola um link do YouTube (ou compartilha do app do
YouTube no celular) e vê os vídeos chegando: lendo, prontos, com erro. Abre um e vê a transcrição
com os minutos e os capítulos.

## Leia antes

- `README.md` desta pasta e o arquivo da fase 1 (a API que esta tela usa).
- `frontend/src/App.vue` (o topo tem **Hoje** e **Trilhas**; a Entrada entra como terceiro item),
  `frontend/src/rotas.js` e `rotas.test.js` (rotas por `#/…`), `frontend/src/api.js`,
  `frontend/src/Pagina.vue`, `frontend/src/style.css` (tokens: toda cor, raio e tamanho vem dali).
- O design system: `../hub-home-lab-web/docs/DESIGN-SYSTEM.md`.
- Compartilhar no Android: veja como o Streaming faz (`../streaming/frontend/vite.config.js`,
  `share_target` com `action: '/compartilhar'`, e `linkCompartilhado` em
  `../streaming/frontend/src/meusVideos.js`). Na Trilha o PWA é configurado em `frontend/vite.config.js`.

## O que fazer

1. **Rotas**: `#/entrada` (a lista) e `#/entrada/<id>` (a fonte). Atualize `rotas.js` e o teste.
2. **Topo**: item "Entrada" ao lado de Hoje e Trilhas, com um número discreto quando há fontes
   prontas que ainda não viraram fragmentos (por enquanto: fontes `pronta`).
3. **Tela `frontend/src/telas/Entrada.vue`** (ou `frontend/src/Entrada.vue`, siga onde ficam as
   outras telas):
   - Campo "Link do YouTube (vídeo ou playlist)" + botão "Colar" (área de transferência) + "Mandar".
   - Lista das fontes: miniatura (`https://i.ytimg.com/vi/<video_id>/mqdefault.jpg`), título, canal,
     duração, estado (`na fila`, `lendo…`, `pronta`, `erro` com a mensagem e "Tentar de novo"),
     a playlist quando veio de uma. Atualiza sozinha a cada 3 s enquanto houver fonte `na_fila` ou
     `lendo`.
   - Apagar fonte (com confirmação).
4. **Tela da fonte** (`#/entrada/<id>`): título, canal, link "Abrir no YouTube", capítulos (lista
   clicável que abre o YouTube no minuto: `https://youtu.be/<id>?t=<segundos>`), e a transcrição em
   bloco recolhível, com os `[mm:ss]` clicáveis do mesmo jeito. Deixe um espaço vazio com o título
   "Fragmentos" e o texto "Na próxima versão" (a fase 3 preenche).
5. **Compartilhar do celular**: `share_target` no manifesto do PWA (`/compartilhar`, GET, com
   `title`, `text`, `url`). A rota `/compartilhar` acha o link do YouTube no meio do texto (o app do
   YouTube manda "Título https://youtu.be/…"), manda para `POST /api/entrada` e abre `#/entrada`
   com um aviso "Na Entrada". Função pura `linkDoYouTube(texto)` em `frontend/src/entrada.js`.
   O servidor Go precisa servir `/compartilhar` com o `index.html` (veja como o Streaming faz em
   `../streaming/backend/server.go`, procure `compartilhar`).

## Testes obrigatórios

- `frontend/src/entrada.test.js`: `linkDoYouTube` acha o link em textos do tipo que o app do YouTube
  manda (3 casos), devolve vazio sem link; uma função `segundosDoMinuto("1:02:05")` (ou reuse
  `tempoParaSegundos` de `videoaula.js`).
- `rotas.test.js` com as rotas novas.
- Go: `GET /compartilhar` devolve o `index.html`.
- Olhe a tela no navegador em largura de celular (360 px): nada passa da largura (sem rolagem
  horizontal). Se houver testes de navegador no repo, acrescente um; se não houver, descreva no
  "Feito" o que você olhou.

## Não fazer

- Não criar fragmentos nem botões de IA (fase 3).
- Não inventar cor ou tamanho fora dos tokens do `style.css`.

## Pronto quando

Testes passam, decisão **T36 — Tela Entrada e compartilhar do celular** em `docs/DECISOES.md`.

Commit: `Entrada (T36): tela da Entrada (colar link, lista, transcrição com os minutos) e compartilhar do celular`

## Dúvidas

## Feito

- Rotas `#/entrada` e `#/entrada/<id>` em `rotas.js` (`SECOES`) com teste novo em `rotas.test.js`.
- Aba **Entrada** na navegação (desktop e celular, terceiro item; nav do celular passou a 3
  colunas), com um contador discreto das fontes `pronta` (`buscarProntas` no `App.vue`, ao abrir e
  ao trocar de seção) e ícone `entrada` novo em `Icone.vue`.
- `frontend/src/entrada.js`: `linkDoYouTube` acha o link do YouTube no meio do texto e completa
  `https://` quando vem sem esquema (`videoDoLink` só aceita com ele); `linkCompartilhado` junta
  `title`/`text`/`url` do share_target; `linkNoMinuto` monta `youtu.be/<id>?t=<s>`;
  `linhasDaTranscricao` marca os `[mm:ss]`/`[h:mm:ss]` com o segundo (reusa `tempoParaSegundos`).
- `telas/Entrada.vue` decide a tela; `telas/ListaEntrada.vue` tem o campo com "Colar" +
  "Mandar", a lista com miniatura `mqdefault.jpg` (ícone de vídeo quando a miniatura falha),
  título, canal, duração, playlist e estado (`na fila`, `lendo…`, `pronta`, `erro` com a mensagem e
  "Tentar de novo"), apagar com `confirm()` e autoatualização a cada 3 s enquanto houver `na_fila`
  ou `lendo`.
- `telas/Fonte.vue`: título, canal, duração, "Abrir no YouTube", capítulos clicáveis para o minuto,
  transcrição em `<details>` com os `[mm:ss]` clicáveis (mostra só os blocos `[mm:ss]`; o
  cabeçalho do arquivo repete o que a tela já tem), estado `erro` com a mensagem e "Tentar de
  novo", e a seção "Fragmentos" com "Na próxima versão".
- Compartilhar: `share_target` no manifesto (`/compartilhar`, GET, `title`/`text`/`url`); o
  fallback do app já devolve o `index.html` nesse caminho (teste `TestCompartilharDevolveOApp`
  trava isso); o `App.vue` lê o link, manda para `POST /api/entrada` e abre `#/entrada` com o aviso
  "Na Entrada" (mostrado pela lista; sem link, o aviso diz que não achou).
- `ESTADOS_FONTE` em `rotulos.js` dá nome e cor aos estados.

Arquivos novos: `frontend/src/entrada.js`, `frontend/src/entrada.test.js`,
`frontend/src/telas/Entrada.vue`, `frontend/src/telas/ListaEntrada.vue`,
`frontend/src/telas/Fonte.vue`. Mudados: `frontend/src/rotas.js`, `frontend/src/rotas.test.js`,
`frontend/src/rotulos.js`, `frontend/src/App.vue`, `frontend/src/Icone.vue`,
`frontend/vite.config.js`, `backend/entrada_test.go`, `CLAUDE.md`, `docs/DECISOES.md` (T36).

Testes rodados:
- `cd backend && go vet ./... && go test ./...` → ok (inclui `TestCompartilharDevolveOApp`).
- `cd frontend && npm test` → 7 arquivos, 38 testes, ok.
- `cd frontend && npm run build` → ok; `share_target` conferido no `manifest.webmanifest` gerado.
- Navegador em 360 px (app real, servidor local): lista, formulário, card de erro e nav sem
  rolagem horizontal (`scrollWidth` = 360); o fluxo `/compartilhar?text=Veja youtu.be/<id>` (sem
  esquema) criou a fonte e abriu `#/entrada` com o aviso "Na Entrada". Não há teste de navegador no
  repo — a conferência foi manual e está descrita aqui.

Não feito: nada de fragmentos ou IA (fases 3+). O aviso "Na Entrada" fica até sair da tela da
Entrada; o contador de prontas atualiza ao trocar de seção, não em tempo real fora da aba.
