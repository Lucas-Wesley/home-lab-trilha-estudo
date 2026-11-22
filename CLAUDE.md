# Trilha de Estudo

App do **ecossistema de apps pessoais do Lucas**. Trilhas de estudo **profundas, personalizadas e
em ordem**: o Lucas diz o que quer aprender, a IA delimita o assunto com perguntas adequadas ao tipo
de assunto, monta ou valida a sequência de etapas, prepara cada lição na profundidade pedida e
acompanha meta, ritmo e domínio.

**Comece lendo:**
- [`docs/DECISOES.md`](docs/DECISOES.md) — **a fonte da verdade.** Vocabulário, fluxos, decisões
  (T1–T60), fases, riscos e o que está em aberto.
- [`docs/PEDAGOGIA.md`](docs/PEDAGOGIA.md) — como a trilha ensina (entrevista por perfil de domínio,
  profundidade em 5 níveis, molde da lição).
- [`docs/DESIGN-SYSTEM.md`](docs/DESIGN-SYSTEM.md) — aponta para o design system do **Hub**, que é a
  fonte de todas as apps desde 18/11/2026 (o design nasceu aqui, T14).

## Em uma frase
Trilha → módulos → etapas; tudo que a IA decide é **proposta** que o Lucas comenta (geral ou
ancorado num trecho), pede ajuste ou aprova (T12, T13); lição gerada sob demanda (T7).

## Contexto do ecossistema
App **independente** (banco + API próprios) que conversa com os outros **por API**: Biblioteca
(livros, capítulos, espinha dorsal), Acervo (Temas e Referências como material), Flashcards
(`POST /api/textos`), Streaming (futuro). Visão geral no repo **Hub**
(`github.com/Lucas-Wesley/hub-home-lab-web`, pasta `docs/`).

## Ambiente
Homelab **LAN-only**: VM `apps` (`192.168.1.20`) no Proxmox do M920q, Docker Compose + Caddy, nome
`trilha.internal`. A conta do Claude vem do Hub (`CLAUDE_TOKEN_ARQUIVO`, H11 de lá).

## Rodar (sempre em containers, T22; nada de Go ou Node no Windows)
- `docker compose -f compose.dev.yaml up` → API em `:8080` e Vite em `:5173` (`PORTA=8093` se a
  8080 estiver ocupada no Windows).
- Testes: `docker compose -f compose.dev.yaml run --rm api go test ./...` e
  `docker compose -f compose.dev.yaml run --rm web npm test`.
- Imagem de produção local: `PORTA=8093 docker compose up -d --build`.
- Revisar uma trilha: `scripts/revisar_trilha.py` num container Python (ver `docs/CRIACAO-DE-TRILHAS.md`).
- **IA sem token** (testar propostas sem gastar a assinatura): `CLAUDE_BIN=/falso/claude-falso.mjs`
  com `scripts/` montado em `/falso`; ele lê o `pedido.md` e devolve uma lição de teste. Com token
  de verdade: `CLAUDE_CODE_OAUTH_TOKEN` no `.env` (fora do Git); no homelab, vem do Hub.
- Deploy: o Lucas faz do Mac (`./deploy.sh`), a partir da `main` no GitHub.

## IA (fase 3)
- `backend/ia.go` chama `claude -p` com a regra do tipo de tarefa, `--json-schema` e só ferramentas
  de leitura e web; `fila.go` roda uma tarefa por vez; `propostas.go` tem propostas, comentários e a
  montagem do `pedido.md` (etapa, lugar na trilha, material, instruções globais → trilha → pedido).
- Regras da IA em `regras/` (vão para `/regras` na imagem; as editadas em Ajustes ficam em
  `/data/regras`). Mudou o que a IA recebe ou devolve, atualize a regra e os testes de `propostas_test.go`.

## Criação com IA (fase 4, T25)
`backend/criacao.go`: a trilha nasce rascunho e cada passo é uma proposta dela (sem etapa):
entrevista → mapa → plano (entrada A) ou validação → plano (entrada B). Aprovar o plano importa o
pacote na própria trilha. Regras em `regras/entrevista.md`, `mapa.md`, `validar-lista.md` e
`plano.md` (+ `criar-trilha.md`). Tela: `frontend/src/telas/Criacao.vue` (`#/trilhas/<id>/criar`).
O `scripts/claude-falso.mjs` responde também aos quatro passos.

## Estudar a etapa (fase 5, T26)
`backend/estudo.go`: exercícios e simulado são propostas da etapa (tipos `exercicios` e `simulado`,
gabarito escondido na revisão) que saem da lição aprovada; aprovados, vão para
`trilhas/<slug>/exercicios|simulado/<chave>.json`. Exercícios corrigidos pelo servidor; simulado
corrigido pela IA na fila (tarefa `correcao`, ligada à `tentativas`, sem proposta). Critério: nota
mínima de Ajustes. A lição vai ao Flashcards (`FLASHCARDS_URL`, `POST /api/textos` com
`origem: "trilha"`): mudar esse envio é mudar o Flashcards também. Telas: `EstudoEtapa.vue` e
`PropostaEstudo.vue`, na página da etapa.

## Material de apoio (fase 6, T27)
`backend/material.go`: livro da Biblioteca (`referencia` = id, `localizador` = `cap. N`/`p. A–B`)
e Tema/Referência do Acervo são guardados por referência; o texto é lido pelas APIs de lá na hora de
chamar a IA e vai para `material/` na pasta da tarefa (lição e espinha dorsal). Links por
`GET /api/material/abrir` (redireciona ao ponto do livro). `backend/sugerir.go`: "Sugerir material"
(proposta `materiais`, regra `materiais.md`). Entrada C ("a partir de material") em `criacao.go`: a
espinha dorsal é uma validação com a regra `espinha.md`. Telas: `EscolherMaterial.vue` (seletor, com
"Enviar PDF" para a Biblioteca) e `PropostaMaterial.vue`. Mudar uma API da Biblioteca ou do Acervo
que a Trilha usa é mudar a Trilha também (`BIBLIOTECA_URL`, `ACERVO_URL`).

## Domínio e revisão (fase 7, T28)
`backend/dominio.go`: revisão espaçada (tabela `revisoes`; escada 1–120 dias; perguntas dos
exercícios, sem IA), simulado intercalado (tentativa `intercalado` com `etapa_chave` vazia; a
correção em `estudo.go` junta as lições em `licoes/`), pontos fracos (contagem por conceito) e o
painel. `backend/reforco.go`: proposta `reforco` (regra `reforco.md`) que insere etapas do tipo
`reforco` com `foco`. `backend/nivelamento.go`: mini-teste de nível (T9) na criação, sem proposta
(`criacoes.nivelamento`, regra `nivelamento.md`). Telas: `Painel.vue`, `Revisao.vue`,
`DominioEtapa.vue`, `PropostaReforco.vue`, `MiniTeste.vue`; revisões do dia no Hoje.

## A lição substitui o material (T29)
O material da etapa é a **fonte** da lição, não leitura do Lucas: a lição extrai tudo o que importa
e não manda abrir documento (sem "leia a seção X", sem link para ele); o material só aparece em
"Referências" e, na tela, em "Fontes", recolhido no fim. Vale para as regras `licao.md`,
`plano.md` e `criar-trilha.md` e para o pedido em `propostas.go`.

## Videoaula (T30)
`backend/videoaula.go`: material `streaming` principal toca no topo da etapa (`Videoaula.vue`, com as
notas); `ResumoEtapa.vue` no fim da lição; "Conferir com a aula" = tarefa `conferir_resumo` (regra
`conferir-resumo.md`) que lê os documentos da etapa (a transcrição) e a lição. Notas e resumo na
tabela `anotacoes`; cada conferência é uma `tentativas` do tipo `resumo`. O vídeo vem direto do
Streaming (`/api/video/proprio/<id>.mp4`): mudar essa rota lá é mudar a Trilha também.

## Modelos (T31)
`modeloPara` (`ia.go`): lição e "sugerir material" → `modelo_licao`; criação, reforço e mini-teste →
`modelo_criacao`; exercícios e simulado → `modelo_estudo`; correção do simulado e conferência do
resumo → `modelo_correcao` (vazio = o dos exercícios). Recomendados na tela `Ajustes.vue` (página
Modelos): Sonnet, Opus, Haiku, Haiku.

## Encaixar material (T32)
`backend/encaixe.go`: `POST /api/trilhas/{id}/encaixar` (material + pedido) cria uma proposta
`encaixe` com chave `@encaixe-<n>` (uma por material) e guarda o material em `encaixes`;
`GET /api/trilhas/{id}/encaixes` lista; aprovar recebe `{encaixes:[i], novos:[i]}`. Regra
`encaixe.md`. Tela: `EncaixarMaterial.vue` na página da trilha (usa `EscolherMaterial` com `sem-papel`).

## Projeto do módulo (T33)
Etapa `tipo = projeto` (última do módulo): o pedido em `propostas.go` lista as etapas do módulo e a
regra `licao.md` (seção 3.1) usa o molde de enunciado de projeto no lugar do da lição.

## Revisar o roteiro e encaixar no tópico (T34)
`backend/roteiro.go`: `POST /api/trilhas/{id}/roteiro` (pedido + `etapa_chave` opcional) cria a
proposta `roteiro` (chave `@roteiro` ou `@roteiro:<etapa>`); `GET .../roteiro?etapa=` devolve a
última; aprovar recebe `{mudancas:[i]}` (editar, novo, mover, remover; `inserirEtapa` e
`lugarNoModulo` são usados também pelo encaixe). O encaixe aceita `etapa_chave` (foco, coluna
`encaixes.foco`, filtro `?etapa=`) e devolve `revisoes` (aprovar: `revisoes:[i]`). Regras
`roteiro.md` e `encaixe.md`. Telas: `RevisarRoteiro.vue`, `MudancaRoteiro.vue`; as páginas da
trilha e da etapa usam `AbasPagina.vue`.

## Entrada (T35–T44)
`backend/entrada.go` e `backend/youtube.go` (código copiado do Acervo, sem chamá-lo):
`POST /api/entrada` com um link de vídeo do YouTube cria a **fonte** (tabela `fontes`); com uma
playlist, uma por vídeo (`--flat-playlist`). Uma fila própria do yt-dlp — separada da do Claude,
sem gastar crédito, uma leitura por vez com 15 s de pausa entre vídeos (`lendo` volta a `na_fila`
ao iniciar) — lê as informações e a legenda (`YTDLP_BIN`, `YTDLP_ARGS`; blocos de 10 s e os
capítulos do autor no cabeçalho) e grava a transcrição em `entrada/<id>/transcricao.md` no
conteúdo. Vídeo sem legenda vai ao Whisper (`WHISPER_URL`); sem ele, a fonte fica em `erro` e
`POST /api/entrada/{id}/tentar` tenta de novo. `DELETE` apaga a fonte e a pasta dela.

Telas: aba **Entrada** na navegação (com o número de fontes `pronta`), `telas/Entrada.vue` dividida
em `ListaEntrada.vue` (`#/entrada`: colar/mandar link, lista com miniatura, estado e apagar;
autoatualiza a cada 3 s enquanto lê) e `Fonte.vue` (`#/entrada/<id>`: capítulos e transcrição
recolhível com os `[mm:ss]` clicáveis, `linkNoMinuto`). O compartilhar do celular cai em
`/compartilhar` (share_target do manifesto, devolvido com o `index.html` pelo fallback do app);
`linkDoYouTube`/`linkCompartilhado` (`frontend/src/entrada.js`) acham o link no texto e **completam
`https://` quando falta o esquema** — `videoDoLink` só aceita com ele. Encaixar é a fase 4 do
[plano-entrada](docs/plano-entrada/README.md).

**Fragmentar** (`backend/fragmentos.go`, T37): `POST /api/entrada/{id}/fragmentar` enfileira a
tarefa `fragmentar` (sem proposta; `fontes.tarefa_id` liga de volta). A IA lê a transcrição
**uma vez** e devolve os trechos por assunto (regra `regras/fragmentar.md`, só `Read`, modelo
`modelo_entrada`); `limparFragmentos` confere tempos, ordem, duração e sobreposição antes de
gravar como `sugerido`. Na página da fonte o Lucas edita (`PUT /api/fragmentos/{id}`, tempos em
`mm:ss`), divide (`/dividir`), junta (`/juntar`), descarta (`/descartar`) e confirma
(`POST /api/entrada/{id}/confirmar`). Refazer apaga os sugeridos e mantém os confirmados; a
tarefa que falha devolve a fonte a `pronta` com o erro.

**Destino e encaixe** (`backend/destinos.go`, T38): `POST /api/entrada/{id}/destino` enfileira a
tarefa `destino` (a fonte guarda `tarefa_destino_id`/`pedido_destino`; as notas da IA ficam em
`fontes.notas`), que compara resumo/assuntos de cada fragmento confirmado com as trilhas não
concluídas (regra `regras/destino.md`, `modelo_entrada`) e grava `fragmentos.sugestao` — um
fragmento pode ir para várias trilhas ou para nenhuma (`sem_destino_em`). `fragmento_destinos`
guarda a seleção e o `proposta_id`; `PUT /api/fragmentos/{id}/destinos` troca a seleção não
enviada ou marca "Sem destino". `POST /api/entrada/{id}/enviar` cria **um encaixe por trilha**
com o material **`youtube`** (`referencia` = URL, `localizador` = todos os intervalos;
`lerIntervalos`/`trechoDaTranscricao`) pelo `criarEncaixe` extraído do `handleEncaixar`; o pedido
só leva os trechos escolhidos, `limparEncaixe` descarta encaixe fora dos intervalos e aprovar
cria/atualiza etapas com um material `principal` por intervalo. Na lista da Entrada cada fonte
resume "N fragmentos · M enviados · K sem destino".

**O trecho na etapa** (`backend/trechos.go`, `TrechoYouTube.vue`, T39): o material `youtube`
`principal` é a aula — toca embutido no topo da etapa (IFrame API, `start` no início do
intervalo), a barra cobre o trecho e, ao passar do fim − 2 s sem parar o vídeo,
`POST /api/etapas/{id}/trechos/visto` grava o "visto" em `trechos_vistos` (intervalo exato do
material — pedaço inventado leva 400); `{"tocou":true}` no mesmo endpoint conta o primeiro sinal
(`disponivel` → `em_andamento`). Todos os trechos vistos **sem lição aprovada**: a etapa conclui
sozinha pela `mudarStatusEtapa` (extraída do `handleStatusEtapa`); com lição, vale o critério de
sempre. O `GET /api/etapas/{id}` devolve `trechos_vistos` para o ✓ das partes. Vídeo sem embed
(erros 101/150) ou sem internet: cartão com miniatura e "Marquei como visto". As notas são as da
Videoaula (`anotacoes`, "Marcar o minuto" com o tempo do player). Na estrutura, etapa com trecho
e sem minutos mostra ícone de vídeo + duração do trecho.

**"Sem destino" e Trazer do Acervo** (`backend/limpeza.go`, `backend/acervo.go`, T40): fragmento
`sem_destino` morre sozinho em **30 dias** — `apagarSemDestino(agora)` (testável, recebe o relógio)
apaga quem passou do prazo e a goroutine `limparSemDestino` roda na partida e a cada 6 h num
`time.Ticker` que para com o contexto do `main`; dar destino limpa `sem_destino_em`, então o
enviado nunca morre, e a fonte sem fragmentos fica (a tela avisa "sem destino · some em N dias").
`POST /api/entrada/acervo` `{"referencia":"<id>"}` traz um vídeo do **Acervo** já transcrito: lê
`GET /api/referencias/{id}`, só aceita `tipo = "video"` com URL de YouTube e `textos.cru` (o resto
→ 400), grava o cru em `entrada/<id>/transcricao.md` e a fonte nasce `tipo = "acervo"` **pronta** —
sem yt-dlp e sem IA; o `video_id` é o id da Referência (o id do vídeo sai da URL normalizada na
tela). Repetir a Referência devolve a mesma fonte; se o vídeo já entrou pelo YouTube, devolve a
fonte `youtube` existente. Na lista, "Trazer do Acervo" abre o painel com os vídeos de lá, busca
por título e "na Entrada" nas já trazidas (some sem `ligados.acervo`).

**Meus arquivos** (`backend/arquivos.go`, `telas/MeusArquivos.vue`, T41): a aba
`#/entrada/arquivos[/<caminho>]` mostra os cursos do Telegram do **Streaming** em curso › módulo ›
aula — a árvore é `frontend/src/pastas.js`, **copiada do Streaming** (S55 de lá). O Streaming segue
o dono do Telegram: a Trilha só repassa a API (`GET /api/arquivos` devolve o `/api/meus-videos` dele
+ `link_streaming`; `POST /api/arquivos/baixar` vira o pedido `tudo` de lá só com as mensagens que
faltam; `POST /api/arquivos/apagar-pasta` e `DELETE /api/arquivos/videos/{id}` repassam). Variáveis
novas: `STREAMING_URL` (rede interna; vazio = aba avisa "não configurado") e `STREAMING_LINK` (o
"Assistir" abre `…/#/assistir/proprio:<id>` lá). Enquanto houver pedido na fila/baixando a tela
atualiza a cada 5 s. Mudar `GET /api/meus-videos` no Streaming é mudar a Trilha também; o `aviso`
que ele manda quando o disco não pôde ser lido (S57 de lá) aparece no topo da aba como erro.
`GET /api/arquivos/usos` (`usos.go`, T45) diz onde cada aula já está (materiais `telegram:` com trilha
e etapa, e encaixes de curso pendentes); `usos.js` monta as etiquetas da aula e da pasta.

**Trilha igual ao curso** (`frontend/src/cursoParaTrilha.js`, T42): "Criar trilha igual a este
curso" na pasta de Meus arquivos abre um diálogo com o título (o da pasta) e dois modos com a
contagem na hora — "uma etapa por aula" (aula = etapa; pasta com aulas = módulo) ou "uma etapa por
pasta" (pasta com aulas = etapa de partes; a pasta mãe = módulo), com a recomendação pela média das
aulas (< 15 min → por pasta). `pacoteDoCurso` é puro e o pacote entra pelo `POST /api/importar` —
o `estado` do pacote passou a valer na criação (nasce `ativa`). Sem IA, sem lição e sem baixar
nada: cada aula vira material `streaming` `principal` com a referência estável
`telegram:<link do grupo>#<mensagem>` (o id do vídeo muda a cada baixar/apagar; a mensagem não).
Na etapa, a `Videoaula.vue` pergunta `GET /api/arquivos/aula?ref=` e o servidor diz `baixada` (com
a `url` `…/api/video/proprio/<id>.mp4`), `na_fila` ou `no_telegram`; a faltante mostra
"Esta aula ainda não está no servidor" + tamanho e `tempoEstimado` (~4 MB/s) com
"Baixar esta aula" ("Baixar as N partes" quando falta mais de uma) →
`POST /api/arquivos/aula/baixar` baixa só aquela mensagem; `na_fila` mostra "Baixando…" e sonda a
cada 5 s até o vídeo aparecer. `ehAulaDoTelegram` e `tempoEstimado` ficam em
`frontend/src/videoaula.js`.

**Encaixar cursos pelos títulos** (`backend/cursos.go`, `frontend/src/cursos.js`, T43): cada pasta
de Meus arquivos tem uma caixa de marcar (a seleção `selecaoCursos` vive no módulo — a tela remonta
a cada troca de pasta). Com uma ou mais marcadas aparecem "Encaixar numa trilha" (seletor de trilha
+ pedido) e "Criar trilha a partir destes cursos" (nome + pedido; cria a trilha vazia por
`POST /api/trilhas` e encaixa nela). O material **`curso`** tem `referencia`
`telegram:<link>::<caminho>` por pasta, separadas por ` | ` (`lerCursos`/`escreverCursos`), e
`titulo` = os nomes com " + ". O servidor remonta a árvore em Go (`arvoreDosArquivos`, mesma regra
do `pastas.js`), escreve o sumário (`##`/`###` por nível, `- A<n> · <nome> (N min)`, números
seguidos entre os cursos) e guarda na proposta os mapas `aulas` (`A<n>` → referência da mensagem) e
`nomes` — o mapa gravado vale na aprovação, não uma nova consulta. A regra `encaixe.md` manda a IA
usar só os números (`A12`, `A12–A15`, `A3, A7`), não chutar título genérico (vai para
`Precisam ser ouvidas:` nas notas) e juntar cursos do mesmo assunto sem duplicar etapas.
`limparEncaixe` tira encaixe/tópico sem nenhum `A<n>`; `aprovarEncaixe` expande cada número em um
material `streaming` por aula (ordem dos números, papel do encaixe — `principal` nos tópicos novos)
e registra nas notas os números fora do mapa. Na proposta, o localizador mostra os nomes das aulas
pelo mapa `nomes`, e "Precisam ser ouvidas" lista as aulas com o botão "Processar estas aulas"
(T44). Sem baixar nem transcrever nada.

**Processar aulas do Telegram** (`backend/telegram.go`, T44): `POST /api/entrada/telegram`
`{refs, fragmentar}` cria uma fonte `telegram` por aula (`video_id` = a ref estável
`telegram:<link>#<msg>`; repetida devolve a mesma). Na fila, `lerTelegram` pergunta o estado da
aula: `no_telegram` → pede o download ao Streaming (`baixarAulaStreaming`, só aquela mensagem) e a
fonte fica `baixando` — **a fila não espera**: uma sonda (`sondaTelegram`, 15 s) promove as
`baixando` de volta à `na_fila`; `baixada` → `transcrevendo`: o ffmpeg (`FFMPEG_BIN`) lê o vídeo
pela url interna (`STREAMING_URL/api/video/proprio/<id>.mp4`), mono 16 kHz em pedaços de 10 min, um
POST por pedaço no Whisper (`WHISPER_URL` — nunca o Claude), somando o início de cada pedaço nos
`[mm:ss]`; áudio apagado ao sair. `fontes.fragmentar` marcada → `pedirFragmentar` ao ficar `pronta`.
No encaixe a fonte vira material **`streaming` com `referencia` telegram: e `localizador`
`mm:ss–mm:ss`**; `trechosDoVideo` acha a fonte pelo `video_id`; `trechosPrincipais` aceita o
streaming+telegram — "visto" e conclusão são os do trecho do YouTube (T39). Na tela: "Processar na
Entrada" na pasta (contagem + tempo de áudio) e por aula em Meus arquivos, com "Dividir em
fragmentos quando terminar" marcado; a processada mostra "na Entrada" (mapa `/api/entrada` por
`video_id`); a lista tem ícone `telegram` e os estados `baixando`/`transcrevendo`. O que era igual
entre `Videoaula` e `TrechoYouTube` virou `BarraTrecho.vue` + `estadoDoTrecho`/`pctDoTrecho`/
`intervaloDoMaterial`/`chaveDoTrecho` (`trechoYoutube.js`): o trecho do Telegram começa no início
do intervalo, com a mesma barra e o mesmo aviso.

**Cursos de plataformas** (`backend/plataformas.go`, `telas/CursosPlataforma.vue`, T50): a aba
`#/entrada/cursos` lista as aulas da Branas.io e da LINUXtips que a extensão do Chrome mandou
(`POST /api/cursos/estimar` e `POST /api/cursos/aulas`, com o manifesto HLS e o Referer do player).
Uma fila própria baixa uma por vez com o ffmpeg, em vídeo na qualidade máxima, para
`CURSOS_DIR/<plataforma>/<curso>/<módulo>/<aula>.mp4` (no servidor `/srv/cursos`, fora do backup).
Mudar o corpo desses pedidos é mudar a extensão (`home-lab-plugin-web`) também.

**Escola de cursos** (`backend/embeds.go`, `backend/cursos_trilha.go`, T55): a aula da plataforma pode vir com o
`embed` do YouTube/Vimeo (o yt-dlp baixa, com `--referer`) ou sem vídeo (`texto` → estado `texto`), com a `ordem`
no curso e o texto da página. Com `trilha: true` (ou `POST /api/cursos/trilha`), o curso vira trilha ativa no
**grupo** da plataforma (coluna `trilhas.grupo`; a lista mostra o grupo como um cartão, `#/trilhas/grupo/<nome>`):
uma etapa por aula (`aula-<id>`), vídeo `/api/cursos/aulas/<id>/video` e o documento `aula-<id>`. Depois só
completa (nunca reimporta: a importação apagaria lições). A `Videoaula.vue` mostra o download em andamento.
Os PDFs da aula (`backend/anexos.go`, T56) chegam por `POST /api/cursos/aulas/{id}/anexos`, ficam ao lado do vídeo e entram na
etapa como material `arquivo` (`/api/cursos/anexos/<id>`).

## Projeto de estudo (T52)
`backend/projeto.go`, `projeto_roadmap.go` e `projeto_refs.go`: a seção **Projeto** (`#/projeto`,
`telas/Projeto.vue`) mostra o caderno público `Lucas-Wesley/projeto-de-estudo` e **só lê**. O
repositório vem inteiro num tarball (`PROJETO_REPO`, `PROJETO_RAMO`, `PROJETO_TARBALL`), fica em memória
e em `DATA_DIR/projeto/caderno.tar.gz`, e é baixado de novo a cada 10 min ou pelo
`POST /api/projeto/atualizar`; GitHub fora = última cópia com `aviso`. `GET /api/projeto` devolve
estado, livros (T21–T23), fases e itens com as referências ligadas às trilhas (`casar`: número de
módulo/etapa/`Cap. N` ou título normalizado; `parcial` quando só achou o módulo) e o prompt de
`PROMPT.md` já preenchido. Estudo na tela: com a tabela "2. Estudo" no caderno do item
(`estudoDoCaderno` → `estudo_escolhido`), "O que estudar, nesta ordem" no topo e os candidatos recolhidos;
sem ela, "Candidatos ao estudo" (o "Estudar antes" + a 7.1 de trilhas, por trilha, sem repetir:
`candidatosDoItem` em `projeto.js`). A **faixa de leitura** (a 7.3, T54): `lerTabela73` → `montarFaixa` (bloco lido/atual/a ler e o próximo capítulo, pelas etapas concluídas das trilhas T21–T23) → quadro "Faixa de leitura" na visão geral. O parser segue o formato do `ROADMAP.md` (fixture real em
`backend/testdata/projeto/`): mudou o formato lá, atualize a fixture e `projeto_test.go`.

## Verificar material (T53)
`backend/verificar.go`: `POST /api/etapas/{id}/verificar` (`{pedido, materiais:[i]}`; sem `materiais` = todos)
grava a verificação em `verificacoes` com o retrato do material e o estado do texto de cada um
(`pronto`, `web`, `transcrevendo`, `sem_texto` + aviso). Vídeo `youtube` ou `streaming` telegram sem
transcrição entra na Entrada (`criarFonte`/`fonteDaAulaTelegram`, a em erro volta à fila) e a
verificação fica `aguardando`; `avancarVerificacoes` (fim de cada `lerProximaFonte` e o GET
`/api/etapas/{id}/verificacoes`) promove para `verificando` com a tarefa `verificar_material` (regra
`verificar-material.md`, `modelo_licao`). O executor monta o `pedido.md` (etapa, etapas do módulo,
cada material como "### Material N") e grava o laudo em `resultado`; falha → `falhou` em
`aoTerminarTarefa`. `trechosDoVideo` aceita localizador vazio (o vídeo inteiro) e acha a fonte também
pelo id do YouTube (`fonteDoVideo`). Tela: `VerificarMaterial.vue` na aba Fontes (+ `verificacao.js`);
"Procurar material para o que falta" chama `pedir` do `PropostaMaterial`.

## Criar trilhas à mão
Siga [`regras/criar-trilha.md`](regras/criar-trilha.md): versão → revisão (automática + manual,
critérios C1–C15) → correção → nova rodada, até passar; registre cada rodada e acrescente as lições
novas na seção 6 das regras. A trilha entra no app por `POST /api/importar` (pacote JSON).

## Regras e convenções (IMPORTANTE)
- **Conteúdo (lições, trilhas, propostas) NÃO sobe pro GitHub** — fica no git local do homelab
  (`/srv/data/trilha`). Só o código vai para este repositório.
- **A IA propõe, o Lucas decide.** Toda ação de IA aceita um **pedido** em texto livre, e toda
  proposta aceita **comentários**. Nunca aplicar mudança da IA sem aprovação.
- Toda chamada ao Claude passa pela **fila** (uma tarefa por vez).
- Toda cor, raio e tamanho vem de um token de `style.css`. O design system mora no Hub
  (`hub-home-lab-web/docs/DESIGN-SYSTEM.md`, quarta versão desde T45): mudança de design é lá primeiro, depois copia aqui.
- Só teoria por enquanto; ferramentas práticas por trilha só quando o Lucas pedir (T11).
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
