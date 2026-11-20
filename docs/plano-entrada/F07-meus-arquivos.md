# Fase 7 — Meus arquivos: os cursos do Telegram dentro da Trilha (sem IA)

## Objetivo

Na Entrada, uma segunda aba **"Meus arquivos"** mostra os cursos do Telegram que estão no
**Streaming**, já organizados em **curso › módulo › aula** (o Streaming faz isso desde a S55 dele),
com o que está baixado no servidor (SSD) e o que está só no Telegram. Daqui o Lucas **baixa** (um
curso, um módulo ou uma aula) e **apaga** (lixeira) sem abrir o Streaming. O Streaming continua o
dono do Telegram (D8): a Trilha só chama a API dele. Nada baixa sozinho (D9).

## Leia antes

- `README.md`.
- No Streaming (só leitura, **não mude o Streaming**):
  - `../streaming/docs/DECISOES.md`, decisões **S42**, **S54**, **S55**, **S56**.
  - `../streaming/frontend/src/pastas.js` e `pastas.test.js`: **a árvore** (junta os vídeos do disco
    com o catálogo dos grupos salvos). Você vai **copiar este arquivo e o teste** para a Trilha.
  - `../streaming/frontend/src/MeusVideos.vue`: como a tela de lá mostra a árvore, "Baixar o que
    falta", "Apagar as baixadas", "na fila".
  - A API de lá:
    - `GET /api/meus-videos` → `{videos, salvos, pedidos, livre, telegram:{instalado}}`. Cada vídeo:
      `id, pasta ("Grupo/Curso/Módulo"), nome, tamanho, duracao, posicao`. Cada salvo (`tudo: true`):
      `link, pasta, titulo, videos:[{mensagem, nome, caminho:[curso, módulo…], tamanho, duracao, ja, video}]`
      (`video` = id do vídeo baixado). Cada pedido: `id, link, estado (na fila, baixando, pronto, erro), mensagens, videos, total, baixado, previsto, mensagem`.
    - `POST /api/meus-videos/telegram` `{link, tudo: true, pasta, mensagens: [..]}` → baixa só essas aulas, cada uma para a pasta do curso e do módulo.
    - `POST /api/meus-videos/apagar-pasta` `{pasta}` → apaga as baixadas da pasta e subpastas.
    - `DELETE /api/meus-videos/{id}` → apaga um vídeo.
    - O vídeo toca em `GET /api/video/proprio/<id>.mp4`; o player de lá abre em `#/assistir/proprio:<id>`.
- Na Trilha: `backend/integracoes.go` (`chamarApp`), `backend/material.go` (como a Biblioteca e o
  Acervo são chamados e configurados por variável de ambiente), `GET /api/integracoes`,
  `frontend/src/videoaula.js` (`linkNoStreaming`).

## O que fazer

### 7.1 Configuração

- Variáveis novas: `STREAMING_URL` (rede interna, `http://streaming:8080`) e `STREAMING_LINK` (o
  endereço que o navegador abre, `https://streaming.internal`). Sem `STREAMING_URL`, a aba "Meus
  arquivos" mostra "O Streaming não está configurado".
- Acrescente as duas em `compose.prod.yaml` (com comentário) e em `GET /api/integracoes`.

### 7.2 API da Trilha (repasse ao Streaming, em `backend/arquivos.go`)

| Rota da Trilha | Chama no Streaming |
|---|---|
| `GET /api/arquivos` | `GET /api/meus-videos`, devolve **como veio** + `link_streaming` (= `STREAMING_LINK`). |
| `POST /api/arquivos/baixar` `{link, grupo, mensagens}` | `POST /api/meus-videos/telegram` com `{link, tudo:true, pasta: grupo, mensagens}`. Valida: mensagens não vazias, link do Telegram (`https://t.me/...`). |
| `POST /api/arquivos/apagar-pasta` `{pasta}` | `POST /api/meus-videos/apagar-pasta`. |
| `DELETE /api/arquivos/videos/{id}` | `DELETE /api/meus-videos/{id}`. |

Erros do Streaming voltam com a mensagem de lá (status 502 quando ele não responde).

### 7.3 Tela

- Na Entrada, abas (use `AbasPagina.vue`): **"Vídeos"** (o que já existe) e **"Meus arquivos"**.
  Endereço: `#/entrada/arquivos` e `#/entrada/arquivos/<caminho da pasta>` (o caminho com `/`,
  codificado). Atualize `rotas.js` e o teste.
- Copie `pastas.js` e `pastas.test.js` do Streaming para `frontend/src/` (o teste tem de passar aqui
  também; ajuste só o que for preciso e diga no comentário do topo que veio do Streaming, S55).
- A tela (componente `MeusArquivos.vue`) mostra, como no Streaming:
  - na raiz: um cartão por grupo/curso, com "N aulas · X baixadas · Y GB no servidor · faltam Z GB";
  - numa pasta: o caminho clicável no topo, o resumo, **"Baixar tudo / o que falta (N aulas · GB)"**,
    **"Apagar as baixadas"** (lixeira, com confirmação dizendo quantas e quanto espaço), as subpastas
    e as aulas na ordem;
  - cada aula: nome, duração, tamanho; baixada → "Assistir" (abre o Streaming em
    `STREAMING_LINK/#/assistir/proprio:<id>`, nova aba) e lixeira; não baixada → "Baixar"; na fila
    ou baixando → etiqueta;
  - o espaço livre no servidor (`livre`) no topo da aba, e aviso quando o que vai baixar não cabe;
  - enquanto houver pedido `na fila`/`baixando`, atualiza a cada 5 s.
- Deixe, no topo de cada pasta, um lugar para as ações das próximas fases ("Criar trilha igual a
  este curso", "Encaixar numa trilha", "Processar") — **não** os crie ainda.

## Testes obrigatórios

- Go: as quatro rotas com um Streaming falso (`httptest`): repassam o corpo certo, devolvem o que o
  Streaming devolveu, 502 quando ele está fora, 400 com corpo inválido; sem `STREAMING_URL` → erro
  claro.
- Frontend: `pastas.test.js` copiado passando; teste das rotas novas.
- Olhe no navegador com um Streaming falso ou com o de verdade **só lendo** (não baixe nem apague
  nada de verdade nos testes).

## Não fazer

- Não mudar o Streaming. Não baixar nem apagar nada de verdade.
- Não criar trilha, encaixe ou transcrição (fases 8–10).

## Pronto quando

Testes passam; decisão **T41 — Meus arquivos (cursos do Telegram pelo Streaming)**; `CLAUDE.md` com
a seção e as variáveis novas.

Commit: `Entrada (T41): Meus arquivos — os cursos do Telegram do Streaming em curso › módulo › aula, com baixar e lixeira`

## Dúvidas

## Feito

- **`backend/arquivos.go`** (novo): as quatro rotas de repasse — `GET /api/arquivos` devolve o
  `/api/meus-videos` do Streaming como veio + `link_streaming`; `POST /api/arquivos/baixar`
  `{link,grupo,mensagens}` valida (mensagens não vazias, link `https://t.me/`; senão 400) e vira
  `{link,tudo:true,pasta:grupo,mensagens}` em `POST /api/meus-videos/telegram`;
  `POST /api/arquivos/apagar-pasta` e `DELETE /api/arquivos/videos/{id}` repassam. Erros dele
  voltam com a mensagem dele (`statusOu`: 4xx dele passa, o resto → 502). `integracoes` ganhou
  `streaming`/`streamingLink` (`STREAMING_URL` + `STREAMING_LINK`, comentadas no
  `compose.prod.yaml`); `GET /api/integracoes` devolve `streaming` — sem URL a aba diz que não
  está configurado.
- **Frontend**: `pastas.js` + `pastas.test.js` copiados do Streaming (S55; `tamanho` acrescentado
  — vem do `tamanho.js` de lá). `MeusArquivos.vue` novo: `AbasPagina` "Vídeos · Meus arquivos"
  nas duas telas da Entrada; `#/entrada/arquivos` e `#/entrada/arquivos/<caminho>` (`linkPasta`
  em `rotas.js`, nível por parte codificado); cartões com o resumo (aulas · baixadas · GB ·
  faltam), caminho clicável, "Baixar tudo/o que falta (N aulas · GB)" — um pedido por grupo sem
  o que já está na fila — e "Apagar as baixadas"/lixeira por aula com confirmação; baixada abre
  "Assistir" em `STREAMING_LINK/#/assistir/proprio:<id>` (aba nova); faltante tem "Baixar"; na
  fila/baixando tem etiqueta; `livre` no topo e "Baixar" avisa e trava quando não cabe; sem o
  Telegram instalado avisa e só deixa ver; pedidos na fila/baixando atualizam a cada 5 s. O
  `.acoes-curso` vazio no topo da pasta é o lugar das ações das fases 8–10.
- **Testes**: `arquivos_test.go` com Streaming falso (`httptest`) — repasse de corpo e resposta
  nas quatro rotas, 400 com corpo inválido (sem chamar o Streaming), 502 quando ele está fora e
  a mensagem dele quando responde 4xx; `pastas.test.js` copiado passando; rotas novas em
  `rotas.test.js`. `go vet` + `go test` ok; `npm test` 53 ok; `npm run build` ok.
- **360 px**: com Streaming falso — raiz com os grupos e "4,4 GB livres", pasta com caminho,
  resumo e botões, módulo com aulas (Assistir → `streaming.internal/#/assistir/proprio:7`,
  "na fila" certa), "Baixar o que falta" clicado de ponta a ponta (aviso + "2 na fila"), abas
  trocando — sem rolagem horizontal.
