# Trilha de Estudo — Plano do Projeto

> Fonte da verdade do projeto. Registro das decisões a partir de **14/11/2026**.
> Ao retomar em outra máquina, leia este arquivo inteiro antes de qualquer coisa.
> Complementos: [PEDAGOGIA.md](PEDAGOGIA.md) (como a trilha ensina) e
> [DESIGN-SYSTEM.md](DESIGN-SYSTEM.md) (como a interface é).

---

## 1. O que é

Um app para **estudar a fundo, em ordem e com progresso**. O Lucas diz o que quer aprender (AWS,
Língua Portuguesa, História do Brasil, a rede do homelab…), a IA ajuda a **delimitar** o assunto com
perguntas certas para aquele tipo de assunto, monta (ou valida) uma **sequência de etapas**, prepara
a **lição** de cada etapa na profundidade pedida e acompanha **meta, ritmo e domínio**.

| | Acervo | **Trilha de Estudo** | Biblioteca |
|---|---|---|---|
| Pergunta | "O que eu sei?" | **"Em que ordem eu aprendo isto, e quanto já domino?"** | "O que estou lendo?" |
| Unidade | Tema | **Trilha → Etapa** | Livro |
| Na Trilha, vira | material de apoio | — | material de apoio **ou** espinha dorsal |

**Não é** o Acervo (curiosidades e Temas soltos que crescem com as Referências). A Trilha **liga**
o que está nos outros apps e produz o próprio conteúdo de estudo, sempre com um objetivo e uma ordem.

Por enquanto é **só teoria**: lição, exercícios teóricos e simulado. A prática específica de cada
trilha (terminal simulado para Linux, laboratórios) é pedida depois, **uma trilha por vez**, com o
Lucas definindo cada cenário.

---

## 2. Vocabulário

- **Trilha:** o objetivo já recortado ("AWS Cloud: serviços essenciais, do iniciante ao
  intermediário"). Tem nível de partida, nível alvo, profundidade, meta de ritmo e as **instruções
  da trilha** (seção 5).
- **Mapa do domínio:** o assunto inteiro, ordenado e com pré-requisitos (Língua Portuguesa do
  alfabeto à análise sintática). A trilha é um **recorte** do mapa.
- **Módulo:** agrupamento de etapas ("Fundamentos", "Armazenamento").
- **Etapa:** a unidade de estudo. Tem objetivo (verbo de Bloom), pré-requisitos, profundidade,
  tempo estimado, materiais, lição, critério de conclusão e status
  (`bloqueada` · `disponivel` · `em_andamento` · `concluida` · `pulada`).
- **Etapa de base:** etapa inserida antes de outra para cobrir um pré-requisito que não cabe num
  parágrafo (ver profundidade, [PEDAGOGIA.md](PEDAGOGIA.md) §3).
- **Lição:** o texto da etapa, em Markdown, gerado **quando o Lucas chega nela**, no molde da
  metodologia de camadas (fundamentos → principal → ao redor → aplicação; 🔴🟡⚪; teste; flashcards;
  referências).
- **Material de apoio:** livro/capítulo (Biblioteca), Tema/Referência (Acervo), PDF, link da web,
  pasta de vídeos (Streaming, futuro), documento próprio. Sempre com **fonte + localizador**
  ("cap. 3", "p. 45", "aula 7", "12:34").
- **Proposta:** tudo que a IA decide (mapa, plano, lição, reordenação, reforço) nasce como proposta,
  com versão, e só vale depois que o Lucas aprova (seção 5).
- **Pedido:** o texto livre do Lucas que acompanha qualquer ação da IA, antes ou depois dela.
- **Perfil de domínio:** o tipo de assunto (história, tecnologia, língua, ciências, genérico). Define
  **que perguntas a IA faz** e **que formato a etapa tem**.

---

## 3. Criar uma trilha: quatro entradas, um fim

```
 A. "A IA monta"          B. "Eu tenho a lista"     C. "A partir de material"     D. Manual
    tema + pedido            EC2 → S3 → IAM…           livro / PDF / Temas do        (sem IA)
        │                        │                     Acervo / docs próprios
        ▼                        ▼                          │
  pré-pesquisa             IA valida: aponta           IA lê o sumário → espinha
  entrevista adaptativa    pré-requisitos faltando,    dorsal; marca capítulos
  mapa completo            respeita a ordem dada       rasos para a profundidade
  Lucas escolhe o recorte                              pedida e propõe complementos
        └────────────────────────┴──────────────┬───────────┘
                                                ▼
              Configuração: nível (autodeclarado OU mini-teste), profundidade,
              meta/prazo, tamanho da entrevista, instruções da trilha
                                                ▼
              PROPOSTA do plano → Lucas comenta, pede ajustes, aprova  (seção 5)
                                                ▼
              Cronograma gerado → trilha ativa
```

- **Toda entrada começa com texto livre**, nunca só com um botão: o tema **e** um campo
  "O que você quer com isto?" (foco, o que já sabe, o que evitar, fontes preferidas). O que estiver
  ali a IA não pergunta de novo.
- **Pré-pesquisa:** antes de perguntar, a IA levanta o tamanho e a estrutura do assunto (com busca na
  web), para fazer perguntas informadas.
- **Entrevista adaptativa:** perguntas de múltipla escolha + "outro" + "não sei, sugira", escolhidas
  pelos **eixos do perfil de domínio** ([PEDAGOGIA.md](PEDAGOGIA.md) §2). Tamanho configurável:
  **rápida** (2–3), **padrão** (5–8, em 1–2 rodadas) ou **profunda** (até ~15, em rodadas). Cada
  rodada também aceita um comentário livre.
- **Mapa completo:** "esta é a ordem completa; de onde até onde você quer?". O Lucas marca o recorte.
- **Nível de partida:** o Lucas escolhe **autodeclarar** ou fazer um **mini-teste** (8–12 perguntas
  espalhadas pelo mapa, que sugerem onde começar).
- **Entrada B:** a ordem do Lucas é respeitada; a IA só **aponta** o que falta ("antes de IAM, vale o
  modelo de responsabilidade compartilhada"), e cada sugestão é aceita ou não.
- **Entrada C:** um livro denso vira a **linha cronológica** da trilha (capítulos → etapas, na ordem
  do livro). A IA compara cada capítulo com a profundidade pedida e, onde o livro é raso, propõe
  **complementos** (etapa de base, ou lição que aprofunda o capítulo). Serve também para docs
  próprios (a trilha piloto nasce assim, seção 10).

---

## 4. Estudar uma etapa

| Parte | O que é |
|---|---|
| **Lição** | gerada sob demanda, na profundidade da etapa, **ancorada nos materiais de apoio** (que vão para a IA como fonte) |
| **Conceitos anteriores** | base (pré-requisito) e vizinho (contexto), no formato da regra `estudar-anteriores.md` do Acervo; com link para um Tema do Acervo quando existir |
| **Exercícios** | resposta certa (escolha / completar), corrigidos **na hora**, sem gastar a assinatura |
| **Simulado** | perguntas abertas sobre o porquê, corrigidas pela IA; o banco cresce com "Gerar mais" (com foco) |
| **Flashcards** | o texto vai a `POST /api/textos` do Flashcards (a IA e as regras dos cartões ficam lá) |
| **Concluir** | pelo critério da etapa (exercícios ≥ a nota mínima de Ajustes, 80% por padrão: a etapa sugere concluir) **ou** à mão; pular é sempre permitido |
| **Revisar** (fase 7) | a etapa concluída volta em 1, 3, 7, 14, 30, 60 e 120 dias com até 5 exercícios dela (os errados antes primeiro), sem IA; sem exercícios, ele se avalia. No Hoje e no painel |
| **Reforço** (fase 7) | "Pedir reforço": a IA lê os erros e propõe de 1 a 3 etapas novas, antes ou depois desta, com o foco da lição de cada uma |

A tela **Hoje** mostra, ao abrir o app, o que há para estudar em cada trilha ativa, o que está
atrasado e o progresso. **Sem notificação** (por ora).

---

## 5. Pedidos e comentários: o Lucas fala com a IA em toda decisão

Pedido do Lucas em 14/11/2026: **toda vez que a IA decide algo, ele pode escrever** — antes, para
dirigir, e depois, para corrigir ("foque em tal coisa", "faltou tal coisa", "refaça esta parte",
"está raso aqui"). Não basta um botão "sugira": o texto livre é sempre possível e é o caminho
principal para ser preciso.

### 5.1 Três níveis de instrução (do mais permanente ao mais pontual)

| Nível | Onde | Vale para | Exemplo |
|---|---|---|---|
| **Global** | Ajustes | toda chamada à IA, em todas as trilhas | "Português do Brasil; fontes em inglês podem; nada de jargão sem explicar." |
| **Da trilha** | configuração da trilha, editável a qualquer hora | toda chamada daquela trilha | "Exemplos sempre com o homelab (OpenWrt, Proxmox)." |
| **Do pedido** | junto de cada ação | só aquela ação | "Nesta lição, compare com o IPv6." |

Os três vão juntos para a IA, nesta ordem, e o mais específico prevalece em caso de conflito.

### 5.2 Antes: o pedido acompanha a ação

Toda ação de IA abre com um **campo de texto** (opcional, mas sempre visível) acima do botão:
criar trilha, montar mapa, validar lista, gerar lição, gerar exercícios, "gerar mais", reforço,
recalcular cronograma. O pedido fica guardado com a proposta que ele gerou.

### 5.3 Depois: a proposta recebe comentários

Tudo que a IA produz vira uma **proposta versionada** (`v1`, `v2`…), que fica **aguardando** até o
Lucas decidir. Na tela da proposta:

- **Comentário geral** ("refaça focando em segurança").
- **Comentários ancorados** num ponto: um módulo, uma etapa, uma seção da lição ou um **trecho
  selecionado** do texto ("este parágrafo está errado", "aprofunde aqui").
- Ações: **Aprovar** · **Pedir ajuste** (manda os comentários para a IA e gera a próxima versão) ·
  **Descartar**.
- A nova versão mostra o **diff** contra a anterior e, para cada comentário, a **resposta da IA**
  (o que mudou, ou por que não mudou). O histórico fica como uma conversa da proposta.

Depois de aprovada, a coisa continua aberta a mudança: **"Pedir mudança"** numa trilha ativa, numa
etapa ou numa lição abre uma proposta nova a partir da versão em vigor. O progresso já feito não se
perde: etapas concluídas ficam concluídas, mesmo se a ordem mudar.

```
 pedido ──► IA ──► proposta v1 ──► comentários ──► Pedir ajuste ──► IA ──► proposta v2 ──► Aprovar
   ▲                                    (gerais e ancorados)                                 │
   └──────────────────────── "Pedir mudança" depois de aprovada ◄────────────────────────────┘
```

### 5.4 Modelo de dados (esboço)

- `propostas`: `id`, `alvo` (`trilha:<id>`, `etapa:<id>`, `licao:<id>`…), `tipo` (`mapa`, `plano`,
  `licao`, `reforco`…), `versao`, `estado` (`gerando` · `aguardando` · `aprovada` · `descartada` ·
  `substituida`), `pedido`, `conteudo` (JSON ou caminho do Markdown), `tarefa_id`, datas.
- `comentarios`: `id`, `proposta_id`, `ancora` (vazio = geral; `modulo:2`, `etapa:5`,
  `secao:2.1`, `trecho:<início>-<fim>` com o texto citado), `autor` (`lucas` | `ia`), `texto`,
  `resolvido`, `criado_em`.

Futuro: ditar o pedido ou o comentário por voz (Whisper, como o "foco falado" do Acervo, D51 de lá).

---

## 6. Material de apoio

| Origem | Como entra | Situação |
|---|---|---|
| **Biblioteca** | escolher livro → capítulos (`GET /api/books/{id}/chapters`); texto para a IA (`/text?chapters=`); abrir a leitura no ponto | ✅ fase 6 (T27): capítulos ou páginas; o texto vai para a IA com as marcações do Lucas; o link abre o leitor no capítulo |
| **Acervo** | escolher Tema ou Referência; texto como fonte; link de volta | ✅ fase 6 (T27): `GET /api/temas`, `/api/pacotes/{tema}` (o Tema inteiro), `/api/referencias` e `/api/referencias/{id}` (o texto organizado) |
| **PDF próprio** | upload **para a Biblioteca**, e a Trilha liga (ganha busca, capítulos e leitor de graça) | ✅ fase 6: "Enviar PDF" no seletor de material passa o arquivo direto para `POST /api/books` de lá |
| **Web** | a IA busca; o **servidor confere se cada URL abre** antes de salvar | ✅ fase 6: "Sugerir material" (proposta da etapa; link quebrado chega marcado e desmarcado) |
| **Docs próprios** (ex.: `estudo/` do homelab) | arquivos `.md` **da trilha** (T23), adicionados à mão na criação (qualquer entrada) ou depois, na página da trilha | ✅ |
| **Streaming** | botão "Vincular pasta de Meus vídeos" e área no material da etapa, **visíveis e desativados** ("em breve") | 🔜 quando o Streaming tiver subpastas e legendas (o botão e a aba "Streaming · em breve" estão na tela desde a fase 6) |

Regra: a lição **referencia** livros protegidos ("leia o cap. 3, p. 45–60") e explica com as
próprias palavras; não copia o livro.

---

## 7. Decisões tomadas

| # | Decisão | Por quê |
|---|---|---|
| T1 | **App independente** do ecossistema (banco + API próprios), repo `home-lab-trilha-estudo` | Padrão do ecossistema; o plano de trilhas do Acervo (`docs/TRILHAS.md` de lá) passa para cá. |
| T2 | **Stack igual à das outras apps:** Go (biblioteca padrão, um pacote) + SQLite (`modernc.org/sqlite`) + Vue 3 em JavaScript + Vite + `vite-plugin-pwa`; um binário com o front embutido | Mesmo esqueleto já provado no Acervo e na Biblioteca. |
| T3 | **Conteúdo fora do GitHub:** lições, estrutura das trilhas e propostas ficam no **git local do homelab** (`/srv/data/trilha`). Só o código vai para o GitHub | Mesmo critério do Acervo. |
| T4 | **IA = Claude Code headless** (`claude -p`), com a conta ativa do Hub (`CLAUDE_TOKEN_ARQUIVO`, H11 de lá) e **fila** (uma tarefa por vez) | Reaproveita a assinatura e o padrão do Acervo. |
| T5 | **Modelo por tipo de tarefa**, ajustável em Ajustes: Opus para plano e deep dive, Sonnet para lição/exercícios/simulado, Haiku para tarefas pequenas. `CLAUDE_MODEL` vazio = padrão do Claude Code | O `--model` aceita `opus`/`sonnet`/`haiku` ou o nome completo. No Pro o Opus gasta o limite mais rápido. |
| T6 | **Regras em `regras/`, editáveis em Ajustes** (como a D25 do Acervo) | A qualidade mora nas regras; ajustá-las sem deploy. |
| T7 | **Lição gerada sob demanda**; o plano é gerado inteiro | Custo menor; a lição se adapta ao que o Lucas já mostrou saber. |
| T8 | **Profundidade em 5 níveis** (Panorama → Deep dive), na trilha e por etapa; conceitos anteriores dentro da lição e, nos níveis 4–5, como etapas de base | Pedido do Lucas: do superficial ao deep dive, da base à superfície. [PEDAGOGIA.md](PEDAGOGIA.md) §3. |
| T9 | **Nível de partida: autodeclarado ou mini-teste**, à escolha | Pedido do Lucas. |
| T10 | **Meta e prazo sem notificação:** dias da semana + minutos por dia, ou data final (o app calcula); tela **Hoje**; atrasou, o app **pergunta** se recalcula | Pedido do Lucas; modelo dos planos de leitura da Biblioteca (R35). |
| T11 | **Só teoria por enquanto**; ferramentas práticas por trilha depois, uma a uma | Pedido do Lucas. Arquitetura deixa um "tipo de atividade" por etapa para plugar um componente. |
| T12 | **Tudo que a IA decide é proposta**; o Lucas aprova | Mesmo princípio do Acervo (a IA sugere, o Lucas decide). |
| T13 | **Pedidos e comentários em toda decisão da IA**: instruções em três níveis, pedido antes, comentários gerais e ancorados depois, versões com diff e resposta da IA a cada comentário (seção 5) | Pedido do Lucas em 14/11/2026: poder dirigir e corrigir a IA com precisão, não só apertar "sugerir". |
| T14 | **Design system próprio, inspirado na Claude Platform** (Docs, Console e Cookbook), em [DESIGN-SYSTEM.md](DESIGN-SYSTEM.md). **Não usa** o design system compartilhado das outras apps | Pedido do Lucas em 14/11/2026: minimalista, botões pequenos, fonte de leitura boa. A Trilha é a primeira; depois o Lucas leva o mesmo design às outras apps. |
| T15 | **PDF próprio sobe para a Biblioteca** e a Trilha só liga | Um lugar para livros; ganha busca, capítulos e leitor. |
| T16 | **Streaming: só o lugar na interface** (botão/área desativados) até ele ter subpastas e legendas | Pedido do Lucas. |
| T17 | **Infra:** VM `apps` (`192.168.1.20`), Docker Compose, Caddy, nome `trilha.internal`; LAN-only; PWA com lições offline | Igual às outras apps. |
| T18 | **Trilha piloto: "Home lab: rede"**, a partir dos docs de `estudo/` (entrada C) | Conteúdo já existe; testa a entrada a partir de material. |
| T19 | **Onde mora cada coisa (fase 2):** estrutura das trilhas, meta, cronograma e progresso no **SQLite**; lições (`trilhas/<slug>/licoes/<chave>.md`) e documentos (`trilhas/<slug>/documentos/<doc>.md`, T23) em **Markdown no git local** (`/data/conteudo`), com um commit por mudança; a estrutura também é exportada em `trilhas/<slug>/trilha.json` a cada mudança | Consultas do Hoje e do cronograma precisam do banco; o texto precisa de histórico, diff e backup simples. O `trilha.json` dá o histórico da estrutura sem depender do banco. |
| T20 | **Criar trilha é um processo iterativo com critérios** (C1–C15): versão → revisão automática (`scripts/revisar_trilha.py`) + leitura manual → correção → nova rodada, até passar. Método em `regras/criar-trilha.md` (vira a regra da IA na fase 4); histórico e lições em `docs/CRIACAO-DE-TRILHAS.md` | Pedido do Lucas em 14/11/2026: criar, analisar, revisar contra critérios e refazer, documentando para a próxima trilha não repetir os erros. |
| T21 | **Importação por pacote JSON** (`POST /api/importar`): trilha + documentos + lições num arquivo; com `substituir`, a nova versão troca a estrutura e **mantém o progresso** das etapas com a mesma `chave`. A ordem dos pré-requisitos é conferida (requisito sempre antes) | Uma trilha feita fora do app (a piloto, e na fase 4 a resposta da IA) entra inteira; refazer a trilha não apaga o que já foi estudado. |
| T22 | **Desenvolvimento 100% em containers** (`compose.dev.yaml`: Go e Node em imagens oficiais; testes e build também); nada instalado no Windows | Pedido do Lucas em 14/11/2026. |
| T23 | **Documentos pertencem à trilha**, não há seção Documentos: ficam em `trilhas/<slug>/documentos/`, aparecem na página da trilha (seção e navegação lateral) e saem junto quando ela é excluída. Entram **à mão**, em qualquer forma de criar a trilha (A, B, C ou D) ou depois. Links entre eles (`02-wifi.md#canais`) abrem dentro da trilha. O topo do app tem só **Hoje** e **Trilhas**; a vitrine do design system continua em `#/design`, sem aba | Pedido do Lucas em 14/11/2026: os documentos que existiam eram só da trilha de rede. |
| T24 | **Fase 3 como foi feita:** a proposta aponta para a etapa pela **chave** (sobrevive a uma nova importação); a lição da IA traz os **conceitos anteriores dentro dela** (seção com `base`/`vizinho` em `<details>`), não num arquivo à parte; a IA só lê (material da trilha, lições dos pré-requisitos) e pesquisa na web, e devolve JSON (`licao`, `notas`, `respostas` a cada comentário); o servidor **testa os links** da lição (quebrado × não confirmado, lição 4 de `regras/criar-trilha.md`); comentário ancorado numa **seção** ou num **trecho selecionado**; mudanças entre versões por linha; aprovar grava a lição no git do conteúdo. Regra em `regras/licao.md`, editável em **Ajustes** (ícone no topo), junto com as instruções globais, o modelo da lição e a fila | Fatia fina da §5 que já serve à trilha piloto: o tipo "lição" exercita todo o ciclo pedido → proposta → comentários → ajuste → aprovação, que as fases 4 e 5 reaproveitam. |
| T25 | **Fase 4 como foi feita:** a trilha nasce **rascunho** já no começo (tabela `criacoes` guarda tema, pedido, tamanho da entrevista, lista, passo, rodadas e escolhas), e cada passo da IA é uma **proposta da trilha** (sem etapa): **A** = `entrevista` (1 a 4 rodadas conforme o tamanho; o Lucas responde com opções, "outro", "não sei, sugira" e um comentário; a IA faz a pré-pesquisa na 1ª rodada e diz quando basta) → `mapa` (o assunto inteiro; o Lucas marca o recorte, confirma objetivo, níveis e profundidade) → `plano`; **B** = `validacao` (a lista dele, na ordem dele; sugestões `inserir`/`dividir`/`observar` que ele aceita uma a uma) → `plano`. O **plano é um pacote de importação** (T21): aprovar importa na própria trilha em criação. O servidor roda a **revisão automática** dos critérios A (C2, C3, C4, C9, C10, C11, C15) e a validação do pacote (grave = não deixa aprovar). Mapa, validação e plano aceitam comentários (gerais ou ancorados em `etapa:<chave>`/`modulo:<n>`) e ajuste, como a lição. Os **perfis de domínio** ficam numa regra só (`regras/entrevista.md`), não em `regras/perfis/`. Regras novas: `entrevista.md`, `mapa.md`, `validar-lista.md`, `plano.md` (o plano leva junto `criar-trilha.md`); um **modelo da criação** em Ajustes. **Mini-teste de nível (T9) ainda não**: o nível é o que a IA deduz da entrevista, e o Lucas confirma | A fatia que deixa criar uma trilha nova inteira sem montar à mão, reaproveitando proposta, fila, importação e o método da trilha piloto. Uma rodada de revisão só (a da IA + a automática + a leitura do Lucas), em vez das três rodadas da piloto: o "Pedir ajuste" é a rodada seguinte. |
| T26 | **Fase 5 como foi feita:** **exercícios** (`escolha` com uma alternativa certa e `completar` com as respostas aceitas) e **perguntas do simulado** (abertas, com o gabarito em pontos, os essenciais marcados) são **propostas da etapa** como a lição (pedido, comentário geral ou em `questao:N`, ajuste, aprovar), e saem **só da lição aprovada** (sem lição, recusa). Na revisão, o **gabarito fica escondido** ("Ver o gabarito"), para aprovar sem estragar o estudo. Aprovados, ficam no git do conteúdo: `trilhas/<slug>/exercicios/<chave>.json` (um conjunto novo **substitui**) e `.../simulado/<chave>.json` (o **banco cresce**: "Gerar mais" com foco; a IA recebe o banco para não repetir, e o servidor tira repetidas). **Exercícios** são corrigidos **pelo servidor, na hora** (completar ignora caixa, acento, espaços e pontuação nas pontas); **simulado**: o Lucas responde às perguntas que quiser e a **correção vai para a fila** (nota 0–100, certo/parcial/errado, comentário e o que faltou por resposta, mais um resumo; regra `corrigir-simulado.md`), sem virar proposta, porque não muda conteúdo. Cada vez fica em `tentativas` (pela chave da etapa); fazer exercícios ou simulado **começa a etapa**. **Critério automático = nota mínima nos exercícios** (Ajustes, padrão 80%): atingida, a etapa **sugere** concluir (botão), sem concluir sozinha; o critério em texto do plano continua visível; o simulado não entra no critério. **Flashcards:** a lição vai a `POST /api/textos` do Flashcards com `origem: "trilha"` (F44 de lá), o baralho escolhido (ou um novo, sugerido com o nome da trilha), cartões de pergunta ou de leitura, um foco opcional, o link da etapa e o link de cada seção (`##`) da lição; cada envio fica anotado com o link da proposta lá. Regras novas `exercicios.md`, `simulado.md`, `corrigir-simulado.md`; **modelo do estudo** e **nota mínima** em Ajustes | Pedido do Lucas ("pode fazer a fase 5"). Proposta para o que a IA escreve (T12) e correção direta para o que só avalia. A sugestão de concluir, em vez de concluir sozinha, deixa a decisão com o Lucas (ele pode querer fazer o simulado antes). |
| T27 | **Fase 6 como foi feita:** o material de outros apps é guardado **por referência** (a Trilha não copia o texto): `livro` = id na Biblioteca + `cap. N`/`caps. 2, 4–6`/`p. 45–60` (a numeração é a do **sumário da Biblioteca**, a que abre o leitor no ponto); `acervo` = pasta do Tema (`temas/…`, com um arquivo dele opcional) ou id de uma Referência. Na hora de chamar a IA, o servidor lê o texto pelas APIs de lá e escreve em `material/` na pasta da tarefa (livro sem trecho = só o sumário; falhou = o pedido diz por quê e a IA segue sem ele); as **marcações do Lucas** no trecho vão junto como o que ele achou importante. O link de livro e Acervo passa por `GET /api/material/abrir`, que acha a âncora (capítulo ou página) e redireciona. Telas: **seletor de material** na edição da etapa (Livro com capa, busca, capítulos com tamanho e marcações, ou páginas; Acervo com Temas e Referências; Documento; Web; Streaming desativado) e **"Enviar PDF"**, que sobe para a Biblioteca e já abre o sumário. **"Sugerir material"** é uma proposta da etapa (tipo `materiais`, regra `materiais.md`): a IA procura na web, nos livros da Biblioteca (recebe a lista), nos Temas do Acervo e nos documentos; o servidor **tira o que não existe** (livro, Tema ou documento fora da lista, repetido) e **confere os links** (quebrado chega desmarcado); aprovar põe na etapa só os marcados. **Entrada C** ("a partir de material"): livros (inteiros ou capítulos) e Temas na ordem escolhida, mais os documentos; o 1º passo é a **espinha dorsal** (proposta do tipo validação, com outra regra, `espinha.md`, e cada item com a `fonte`): a IA recebe o sumário e o texto dos capítulos (até 60 mil caracteres por capítulo, 1,5 milhão no total), monta a sequência na ordem do material e aponta o que é **raso para a profundidade** (`observar`) e os complementos (`inserir`), que o Lucas aceita um a um; o plano recebe só os sumários e põe o trecho como material principal de cada etapa (a revisão avisa livro ou Tema fora do material escolhido, C10). Endereços: `BIBLIOTECA_URL`/`ACERVO_URL` (rede interna) e `BIBLIOTECA_LINK`/`ACERVO_LINK` (links da tela) | Pedido do Lucas ("pode fazer a fase 6"). Referência em vez de cópia: o livro é protegido (§6) e o texto continua num lugar só, com as marcações sempre atuais. A espinha reaproveita a tela e a aprovação da validação da lista (B): a diferença está no que a IA lê e na regra. |
| T28 | **Fase 7 como foi feita:** **revisão espaçada** (tabela `revisoes`, pela chave da etapa): toda etapa concluída ganha a revisão um dia depois de concluir; a escada é 1, 3, 7, 14, 30, 60 e 120 dias. A revisão sorteia **até 5 exercícios** dela (os que ele errou da última vez primeiro), corrigidos pelo servidor, **sem IA**; nota ≥ mínima sobe um degrau, ≥ 50% repete o degrau, abaixo volta ao primeiro. Sem exercícios, ele tenta lembrar das seções da lição e se avalia (lembrei bem · mais ou menos · esqueci). Aparece no **Hoje** (todas as trilhas ativas), na página da etapa e no painel; cada vez é uma tentativa `revisao`. **Simulado intercalado** (no painel): sorteia de 3 a 10 perguntas dos bancos do simulado das etapas concluídas ou em andamento, uma etapa por vez e **as fracas primeiro**, misturadas (a etapa só aparece na correção); a IA corrige na fila como o simulado da etapa (tentativa `intercalado` da trilha, com a lição de cada etapa em `licoes/`). **Pontos fracos**, sem IA: por etapa e conceito, juntando exercícios, simulado, revisão e intercalado, é fraco o que ele **errou da última vez que apareceu** (certo depois, sai); a etapa fica fraca com exercícios abaixo da mínima, simulado abaixo de 60, última revisão abaixo da mínima ou um ponto fraco. **Reforço** (proposta `reforco` da etapa, regra `reforco.md`): a IA recebe a lição, as notas e cada ponto fraco com a pergunta, a resposta dele e o esperado, e devolve o diagnóstico e de 1 a 3 etapas (`antes` ou `depois`, com o **foco** da lição de cada uma); aprovar insere as marcadas no módulo, junto da etapa, do tipo `reforco`, com o material dela como apoio (a de antes herda os pré-requisitos e, se a etapa não foi feita, vira pré-requisito dela; a de depois tem a etapa como pré-requisito). Sem erros registrados, o pedido é obrigatório. A etapa ganhou o campo **foco** (editável, vai para o pedido da lição). **Painel** (`#/trilhas/<id>/painel`): progresso, média dos exercícios, revisões, etapas fracas e pontos fracos (com "Pedir reforço"), o intercalado e as notas de cada etapa. **Mini-teste de nível** (T9): no mapa ou na lista/espinha, a IA escreve de 8 a 12 perguntas de escolha espalhadas pelos itens (regra `nivelamento.md`; tarefa sem proposta, guardada em `criacoes.nivelamento`); o servidor corrige, diz o que ele já domina, onde começar e o nível (60% num nível conta como sabe); "Usar a sugestão" preenche o nível e tira do recorte o que ele domina, para ele conferir; o resultado vai para o pedido do plano. Reforço e mini-teste usam o modelo da criação; o intercalado, o do estudo | Pedido do Lucas ("pode fazer a fase 7 e todas as outras"). A revisão sem IA não gasta a assinatura (pode ser diária); o reforço e o mini-teste são pedidos por ele, nunca disparados sozinhos. O mini-teste não vira proposta: aprovar as perguntas antes estragaria o teste. Streaming e ferramentas práticas ficaram de fora pelas próprias decisões (T16, T11). |
| T29 | **A lição substitui o material.** O material da etapa (documentos, livros, Acervo, web) é a **fonte** da lição, não leitura do Lucas: a IA extrai dele tudo o que importa e traz para dentro da lição, sem "leia a seção X" nem link para o documento; o material só aparece em "Referências", sem link. O tamanho segue o conteúdo (a lição ocupa quase todo o tempo da etapa; se precisar de mais, a IA diz em `notas`). Erro do material vira a versão certa na lição e uma nota para o Lucas, sem citar o documento. Critérios de conclusão não citam o material. Na página da etapa, a lição vem primeiro e o material vira **"Fontes"**, recolhido no fim. A trilha piloto foi refeita assim (critérios reescritos; lições geradas de novo) | Pedido do Lucas (15/11/2026): "o documento que eu coloquei é apenas como referência para extrair conteúdo… pela trilha eu já quero ter tudo", sem ir e voltar entre páginas. Vale também para livros: a lição explica tudo com as palavras dela, sem copiar trechos longos (§6). |
| T30 | **Videoaula: numa trilha de curso em vídeo, o vídeo é o principal.** O material `streaming` com papel `principal` (`referencia` = o arquivo no Streaming, `…/api/video/proprio/<id>.mp4`) toca **no topo da etapa** como numa aula (partes, velocidade, lembra onde parou, "abrir no Streaming"); embaixo, **Minhas notas** (salvas sozinhas; "Marcar o minuto" ou Ctrl/⌘+Enter escreve `[mm:ss]` e cada marca volta ao ponto). A lição vira **"Texto da aula"** e a transcrição (documento, papel `apoio`) é a fonte dela. No fim da lição de **toda** etapa, **Meu resumo**; "Conferir com a aula" põe na fila a tarefa `conferir_resumo` (regra `conferir-resumo.md`), que compara o resumo com a transcrição (ou a lição) e devolve veredito, nota, o que está certo, os erros com a correção e o que faltou, com o minuto para rever. Guardado na tabela `anotacoes` (notas e resumo, pela chave) e nas `tentativas` do tipo `resumo` (fora do domínio e do critério). Substitui o "em breve" da T16 para os vídeos próprios do Streaming. | Pedido do Lucas em 16/11/2026 para a trilha de gramática (curso do Telegram): assistir na própria trilha, anotar e validar o próprio entendimento contra a transcrição. |
| T31 | **Modelo por tipo de tarefa, com o recomendado; Ajustes em páginas.** Quatro escolhas em Ajustes → Modelos: **Lição** (Sonnet), **Criação de trilhas** (Opus), **Exercícios e simulado** (Haiku) e **Correção e conferência** (Haiku; vazio = o dos exercícios), cada uma com "(recomendado)" na lista e o botão "Usar os recomendados". A recomendação vem de um teste em 17/11/2026 com 3 lições da trilha AWS: Haiku e Sonnet geraram exercícios e simulados de mesmo nível (gabaritos certos, perguntas por cenário) e corrigiram o simulado com o mesmo critério; o Haiku gasta bem menos do limite da conta. Medição nas sessões do Claude Code: uma lição custa ~40 mil tokens de saída e 12 turnos; exercícios ou simulado, ~8 mil. Ajustes virou quatro páginas (`#/ajustes/geral`, `modelos`, `regras`, `fila`), na navegação lateral e em abas no celular. | Pedido do Lucas em 17/11/2026, depois de ver o gasto das trilhas 4 e 5: saber que modelo é usado, poder trocar e a tela de Ajustes menos confusa. |
| T32 | **Encaixar material: a trilha montada aos poucos.** Primeiro o roteiro (só os tópicos, cada etapa com o que aprender e como praticar), depois o conteúdo, material por material. Na página da trilha, **Encaixar material**: o Lucas escolhe um livro ou PDF da Biblioteca (livro inteiro: vai o sumário e a IA encaixa por capítulo), uma Referência ou Tema do Acervo (que já transcreve vídeos do YouTube, Reels do Instagram e páginas), um documento da trilha ou um link, mais um pedido opcional. A IA (regra `encaixe.md`, modelo da lição) devolve uma proposta da trilha (`tipo = encaixe`, chave própria por material, para vários na fila): os **encaixes** (etapa, trecho do material, papel, nota) e os **tópicos novos** que o material traz e a trilha não tem (módulo existente ou novo, depois de qual etapa). O Lucas marca o que aceita, pede ajuste ou descarta; aprovar põe o material nas etapas marcadas e cria as etapas novas, com o material como principal. O material fica na tabela `encaixes` (os ajustes apontam para a 1ª versão). As etapas ganham número (3.2) na estrutura. | Pedido do Lucas em 17/11/2026 para a trilha de redes: montar o roteiro primeiro e depois mandar livros, PDFs e vídeos para a IA dizer "isso entra no 3.2", inclusive usando livros bons como referência de tópicos que o roteiro deixou passar. |
| T33 | **Projeto do módulo.** Etapa do tipo `projeto` (além de `normal`, `base` e `reforco`), a última do módulo: não ensina assunto novo, consolida o que as etapas do módulo ensinaram num trabalho que roda no Mac (e no homelab quando couber). O pedido da lição diz que é projeto e lista as etapas do módulo; a regra `licao.md` (seção 3.1) troca o molde da lição pelo enunciado (o projeto, o que precisa, requisitos, marcos, como verificar, onde costuma travar, ir além, gabarito escondido). Na tela: etiqueta "Projeto" e "Enunciado do projeto" no lugar de "Lição". | Pedido do Lucas em 17/11/2026: na trilha de redes, cada módulo termina com um projeto para consolidar o módulo. |
| T34 | **Revisar o roteiro e encaixar no tópico; trilha e etapa em abas.** "Revisar o roteiro" (aba da trilha) e "Revisar com IA" (na etapa): proposta `roteiro` (chave `@roteiro` ou `@roteiro:<etapa>`, regra `roteiro.md`, modelo da criação) com mudanças `editar` (só os campos que mudam), `novo`, `mover` e `remover`, contra o objetivo, o recorte, os níveis e a profundidade; o Lucas marca (remover vem desmarcado, com aviso quando a etapa tem lição ou progresso) e aplica. O encaixe ganhou o **foco** num tópico (aba Fontes da etapa; coluna `encaixes.foco`) e as **revisões** dos tópicos que o material mostra que deviam mudar (`revisoes`, só `editar`). As páginas da trilha (Estrutura, Revisar roteiro, Encaixar material, Documentos, Meta e cronograma, Sobre) e da etapa (Lição, Meu resumo, Exercícios, Simulado, Flashcards, Domínio, Fontes) ficam em abas (`AbasPagina.vue`), com a aba no endereço. | Pedido do Lucas em 18/11/2026: ao mudar os níveis e a profundidade da trilha, poder revisar os tópicos; encaixar material num tópico, com a opção de revisar o tópico; e as páginas organizadas em abas, "para saber onde começa uma coisa e termina outra". |
| T35 | **Entrada: fontes do YouTube, sem IA (fase 1 do plano-entrada).** A área nova se chama **Entrada**: o Lucas manda um link de vídeo ou de playlist do YouTube e cada vídeo vira uma **fonte** (tabela `fontes`, `UNIQUE(tipo, video_id)` — mandar o mesmo vídeo devolve a existente; playlist cria uma fonte por vídeo), com título, canal, duração, idioma e os **capítulos do autor**. A **transcrição com marcas [mm:ss]** vai para o conteúdo (`entrada/<id>/transcricao.md`), montada da legenda pelo **yt-dlp na própria Trilha** (código copiado do Acervo, `youtube.go`, sem chamar o Acervo), com blocos de **10 s** em vez de 30 — o minuto do fragmento depende disso. A legenda segue a ordem do Acervo (manual no idioma do vídeo → automática `-orig` → manual em português → qualquer manual → automática no idioma); **sem legenda**, o áudio vai ao **Whisper** (`WHISPER_URL`, API da OpenAI com `verbose_json`) e, sem ele configurado, a fonte fica em `erro` com a razão. A leitura roda numa **fila própria**, separada da do Claude (não gasta crédito): uma por vez, **15 s de pausa entre vídeos** (o YouTube bloqueia quem pede muito seguido) e `lendo` volta a `na_fila` ao iniciar. Rotas: `POST/GET /api/entrada`, `GET /api/entrada/{id}` (com a transcrição), `POST .../tentar`, `DELETE .../{id}` (apaga a pasta no conteúdo). yt-dlp e ffmpeg entram na imagem como no Acervo (`ARG YTDLP_EM` no build do deploy). A tela é a fase 2; fragmentar e encaixar, as fases 3 e 4 | O vídeo deixa de passar pelo Acervo (que gastava IA duas vezes): a Trilha baixa só a legenda, e a IA vai dividir a fonte em fragmentos por assunto (fase 3). |
| T36 | **Tela da Entrada e compartilhar do celular (fase 2 do plano-entrada).** Aba **Entrada** na navegação (terceiro item, com um número discreto das fontes `pronta` que ainda não viraram fragmentos), nas rotas `#/entrada` e `#/entrada/<id>`. A lista tem o campo "Link do YouTube (vídeo ou playlist)" com "Colar" (área de transferência) e "Mandar" (`POST /api/entrada`) e cada fonte com miniatura (`i.ytimg.com/vi/<id>/mqdefault.jpg`), título, canal, duração, playlist e estado (`na fila`, `lendo…`, `pronta`, `erro` com a mensagem e "Tentar de novo"), apagar com confirmação e autoatualização a cada 3 s enquanto algo lê. A página da fonte mostra os **capítulos clicáveis** (`youtu.be/<id>?t=<segundos>`) e a **transcrição recolhível** com os `[mm:ss]` abrindo o YouTube no minuto, e o lugar vazio dos fragmentos ("Na próxima versão"). O **share_target** do manifesto (`/compartilhar`, GET, `title`/`text`/`url`) recebe o compartilhar do Android; `linkDoYouTube` (`frontend/src/entrada.js`) acha o link no meio do texto e **completa `https://` quando o texto vem sem esquema** (`videoDoLink` só aceita com ele); o link vai para `POST /api/entrada` e o app abre em `#/entrada` com o aviso "Na Entrada". | Pedido do Lucas: colar um link ou compartilhar do celular já deve bastar para o vídeo entrar na Trilha. |
| T37 | **Fragmentos da Entrada (fase 3 do plano-entrada).** A fonte `pronta` vira **fragmentos**: trechos por assunto com início, fim (segundos), título, resumo, assuntos e tipo (`conteudo` · `opiniao` · `fora`; tabela `fragmentos`, `ON DELETE CASCADE` da fonte). É a **única vez que a IA lê a transcrição inteira** (D3 do plano): a tarefa `fragmentar` (fila, sem proposta — a fonte guarda `tarefa_id` e o `pedido` do Lucas) monta `pedido.md` (`# Pedido: fragmentar` + dados do vídeo, capítulos, pedido e os já confirmados) e `transcricao.md` na pasta da tarefa e chama o Claude só com `Read`, a regra **`fragmentar.md`** e o esquema JSON. O servidor **confere antes de gravar** (`limparFragmentos`): tempos válidos, `fim > inicio`, dentro da duração (fim que passa fecha no fim do vídeo), em ordem e sem sobreposição (o fim do primeiro vira o início do segundo); o que não passa vai para `fontes.erro`/notas. Grava como **`sugerido`**; fonte vai a `fragmentada` (ou, na falha da tarefa, volta a `pronta` com o erro — ou a `fragmentada`, se já tinha fragmentos). Fragmentar de novo apaga os `sugerido` e **mantém os `confirmado`** (vão no pedido para a IA não repetir). O Lucas revisa na página da fonte: edita tempos e textos (`PUT /api/fragmentos/{id}`, aceita `mm:ss`), divide (`.../dividir`, `{"em":"mm:ss"}` — o segundo vira "(continuação)"), junta com o seguinte (`.../juntar`, títulos com " + "), descarta (some, com "mostrar descartados") e confirma tudo (`POST .../confirmar`: `sugerido` → `confirmado`). Modelo próprio: **`modelo_entrada`** ("Entrada: fragmentar e sugerir trilhas", recomendado Haiku; `modeloPara` tipos `fragmentar` e, na fase 4, `destino`). `GET /api/entrada/{id}` passa a trazer `fragmentos` e `tarefa`. Encaixar os fragmentos nas trilhas é a fase 4. | Uma leitura da IA por fonte: o trecho certo já vem com minuto — a fase 4 só sugere o destino, sem ler a transcrição de novo. |
| T38 | **Destino dos fragmentos e encaixe de vídeo em fragmentos (fase 4 do plano-entrada).** Com os fragmentos confirmados, a tarefa **`destino`** (fila, sem proposta; a fonte guarda `tarefa_destino_id` e `pedido_destino`, e as notas da IA vão para `fontes.notas`, nunca `erro`) compara resumo/assuntos de cada fragmento — inclusive `opiniao`/`fora` — com as trilhas não concluídas (título, objetivo, módulos, todas as etapas e tópicos, até 300 por trilha), **sem ler a transcrição de novo**, e grava `fragmentos.sugestao` (`regras/destino.md`, modelo `modelo_entrada`, só `Read`). Um fragmento pode ir para **várias trilhas** ou para **nenhuma** (`sem_destino_em`); `fragmento_destinos` guarda as escolhidas (`sugerido` → `confirmado`, `proposta_id` quando vira encaixe) e `PUT /api/fragmentos/{id}/destinos` troca a seleção não enviada ou marca "Sem destino". `POST /api/entrada/{id}/enviar` cria **um encaixe por trilha** com o material **`youtube`** novo (`referencia` = URL, `localizador` = todos os intervalos escolhidos, `lerIntervalos`/`escreverIntervalos` com `mm:ss`, `h:mm:ss`, hífen e travessão) e pedido "São N fragmentos deste vídeo; encaixe cada um pelo seu intervalo" — aprovando, cada intervalo vira uma entrada de material `principal` na etapa. O pedido ao Claude leva **só os trechos dos intervalos** (`trechoDaTranscricao`, cabeçalho `## início–fim · título` + resumo do fragmento); `limparEncaixe` joga fora encaixe cujo localizador sai dos intervalos mandados; trilha vazia recebe módulo/etapas novas na aprovação (`criarEncaixe` extraído do `handleEncaixar`, sem duplicar). Na fonte: seção "Destino" com "Sugerir trilhas"/"Sugerir de novo", motivo de cada trilha, "+ trilha", "Trilha nova", "Sem destino", "Mandar para as trilhas" e os links "Foi para" → `#/trilhas/<id>#encaixar`; a lista resume "N fragmentos · M enviados · K sem destino". Tocar o trecho na etapa é a fase 5. | Um encaixe por trilha gasta uma chamada de IA por trilha (não por fragmento) e o material `youtube` evita a segunda transcrição/reescrita do Acervo — o custo duplo que motivou a Entrada. |
| T39 | **O trecho do YouTube é a aula da etapa (fase 5 do plano-entrada).** O material `youtube` `principal` toca embutido no topo da etapa (`TrechoYouTube.vue`, IFrame API carregada sob demanda, `start` = início do intervalo): a barra cobre só o trecho (0% = início, 100% = fim) com o "faltam mm:ss", "Voltar ao início do trecho" e "Abrir no YouTube em \<início\>" (`youtu.be/<id>?t=`). Um relógio de 500 ms, só com o vídeo tocando, segue a posição: **antes** do trecho o estado é mostrado sem mexer no vídeo; ao passar do **fim − 2 s** pela primeira vez, o aviso "Trecho visto ✓ · Continuar assistindo · Ir para a próxima etapa" aparece **sem parar o vídeo** e `POST /api/etapas/{id}/trechos/visto` grava o intervalo (uma vez por trecho; o vídeo pode ter vários, virando "partes"). Embed recusado (erros 101/150) ou sem internet: cartão com miniatura, intervalo, "Abrir no YouTube" e "Marquei como visto". O `{"tocou":true}` do mesmo endpoint conta o primeiro sinal da aula: `disponivel` → `em_andamento`. Cada "visto" é uma linha de **`trechos_vistos`** (`trilha_id`, `etapa_chave`, `referencia`, `localizador` canônico) e volta no `GET /api/etapas/{id}` (`trechos_vistos`) para marcar o ✓ das partes; o intervalo tem que ser exato do material principal — pedaço inventado leva 400. Todos os trechos vistos **sem lição aprovada**: a etapa conclui sozinha; **com lição**, vale o critério de sempre (exercícios/simulado). Ambos passam por `mudarStatusEtapa` — a mesma função extraída do `handleStatusEtapa`. As notas são as da Videoaula (mesma tabela `anotacoes`, "Marcar o minuto" com o tempo do player, marcas navegáveis, `ResumoEtapa` continua ligado). A aba "Texto da aula" avisa "A aula desta etapa é o trecho do vídeo" e "Gerar lição escrita (opcional)" segue disponível; na estrutura da trilha, etapa com trecho e sem minutos mostra ícone de vídeo + duração do trecho. `trechoYoutube.js` tem as funções puras testadas (`idDoVideo`, `trechos`, `estadoDoTrecho`, `linkNoMinuto`). | Ver o vídeo no lugar certo é a aula: a etapa anda e conclui pelo que o Lucas fez, sem mais uma chamada de IA nem um formulário de conclusão. |
| T40 | **"Sem destino" expira em 30 dias e vídeos entram na Entrada pelo Acervo (fase 6 do plano-entrada).** `apagarSemDestino(agora)` (`limpeza.go`) apaga fragmentos `sem_destino` com `sem_destino_em` há mais de **30 dias** (`diasSemDestino`); mandar a um destino limpa o carimbo, então fragmento enviado nunca morre. A função recebe o "agora" para testar 31 e 29 dias sem esperar; `limparSemDestino` é a goroutine do app — roda na partida e a cada **6 horas** num `time.Ticker`, e para quando o contexto do `main` cancela. Fonte que fica sem fragmento **não** é apagada (o Lucas decide); a tela mostra "sem destino · some em N dias" (`diasSemDestino` em `entrada.js`, arredonda para cima). **`POST /api/entrada/acervo`** (`acervo.go`, corpo `{"referencia":"<id>"}`) lê `GET /api/referencias/{id}` do Acervo e cria a fonte `tipo = "acervo"` já **`pronta`**: só aceita `tipo = "video"` com URL de YouTube e `textos.cru` não vazio (o resto → 400); o `video_id` guarda o **id da Referência** (deduplica por `tipo`+`video_id`) e a URL normalizada leva o id do vídeo, que é de onde a tela tira miniatura e links no minuto. `textos.cru` vira `entrada/<id>/transcricao.md` — sem baixar legenda, sem yt-dlp, sem IA (blocos `[mm:ss]` a cada ~30 s nessas fontes; a regra `fragmentar.md` já pede o início "uns segundos antes"). Repetir a Referência devolve a mesma fonte; se o vídeo já entrou direto do YouTube (mesma URL normalizada), devolve a fonte `youtube` existente em vez de duplicar. Na Entrada, "Trazer do Acervo" abre um painel (`/api/acervo/referencias`, só `tipo = "video"`) com busca por título, etiqueta "na Entrada" nas já trazidas e clique que importa e abre a fonte; some se o Acervo não está ligado (`ligados.acervo`). | Um vídeo já transcrito no Acervo não paga nem download nem IA de novo — a transcrição pronta é a fonte; e a lista de "sem destino" se limpa sozinha sem nunca apagar o que já virou aula. |
| T41 | **Meus arquivos: os cursos do Telegram do Streaming dentro da Entrada (fase 7 do plano-entrada).** O Streaming continua o dono do Telegram (sessão do tdl, downloads, lixeira — S42, S54–S56 de lá); a Trilha **só repassa a API** e nada baixa sozinho. `backend/arquivos.go`: `GET /api/arquivos` devolve o `/api/meus-videos` dele como veio + `link_streaming`; `POST /api/arquivos/baixar` `{link,grupo,mensagens}` vira `{link,tudo:true,pasta:grupo,mensagens}` em `/api/meus-videos/telegram` (400 sem mensagens ou com link que não é `https://t.me/`); `POST /api/arquivos/apagar-pasta` e `DELETE /api/arquivos/videos/{id}` repassam o corpo/rota e a resposta; erro do Streaming volta com a mensagem dele (502 quando não responde). Envs `STREAMING_URL` (rede interna, no `compose.prod.yaml`) e `STREAMING_LINK` (o endereço que o navegador abre); `GET /api/integracoes` ganhou `streaming` — sem URL, a aba avisa que não está configurado. A aba **"Meus arquivos"** (`#/entrada/arquivos[/<caminho>]`, `AbasPagina` com "Vídeos") junta disco e catálogo pela `pastas.js` **copiada do Streaming** (S55: `montarArvore`, `oQueFalta`, `aulasPedidas`, +`tamanho`): cartões de grupo/curso com "N aulas · X baixadas · Y GB no servidor · faltam Z GB", caminho clicável, "Baixar tudo/o que falta (N aulas · GB)" (um pedido por grupo, só o que falta e não está na fila), "Apagar as baixadas" com confirmação, aulas com duração/tamanho — baixada abre "Assistir" em `STREAMING_LINK/#/assistir/proprio:<id>` e tem lixeira, faltante tem "Baixar", na fila/baixando tem etiqueta; `livre` aparece no topo e "Baixar" avisa e trava quando não cabe; enquanto houver pedido `na fila`/`baixando`, atualiza a cada 5 s. O cabeçalho da pasta guarda o lugar vazio das ações das fases 8–10. | As aulas de curso entram na Trilha pela mesma fonte que o Lucas já gerencia no Streaming — sem duplicar download, sessão ou lixeira entre os apps. |
| T42 | **A trilha nasce igual ao curso do Telegram e a aula baixa por clique na etapa (fase 8 do plano-entrada).** "Criar trilha igual a este curso" no topo da pasta de Meus arquivos abre um diálogo com título (o da pasta) e dois modos com a contagem na hora: **uma etapa por aula** (cada aula é uma etapa; cada pasta com aulas é um módulo, nome = caminho a partir da pasta com " › "; aulas soltas → módulo da pasta escolhida) ou **uma etapa por pasta** (cada pasta com aulas é uma etapa de partes em ordem; o módulo é a pasta mãe — filha direta da escolhida → a própria). `cursoParaTrilha.js` (`pacoteDoCurso`, `resumoDoPacote`, `modoRecomendado` — média < 15 min → por pasta) monta o pacote puro, que entra pelo `POST /api/importar` de sempre; o `estado` do pacote passou a valer na criação (a trilha nasce `ativa`). **Sem IA, sem lição escrita e sem baixar nada**: cada aula vira material `streaming` `principal` com referência estável `telegram:<link do grupo>#<mensagem>` (o id do vídeo muda a cada baixar/apagar; a mensagem não). Na etapa, a `Videoaula` pergunta `GET /api/arquivos/aula?ref=` ao servidor, que devolve `baixada` (com `url` `…/api/video/proprio/<id>.mp4`), `na_fila` ou `no_telegram` (o Streaming responde 404 com "aula não encontrada no Streaming"); a faltante mostra "Esta aula ainda não está no servidor" com tamanho + `tempoEstimado` (~4 MB/s) e "Baixar esta aula" (ou "Baixar as N partes") → `POST /api/arquivos/aula/baixar` repassa só aquela mensagem; `na_fila` mostra "Baixando…" e sonda a cada 5 s até virar o vídeo. A conclusão é a da videoaula de sempre (T30). O `aviso` do `/api/meus-videos` (S57 de lá — disco ilegível, lista do banco) aparece no topo de Meus arquivos como aviso de erro. | A trilha congela a estrutura do curso sem gastar um download: a mensagem é a identidade estável da aula, e o clique explícito decide quando (e se) ela ocupa o disco. |
| T43 | **Encaixar cursos do Telegram numa trilha pelos títulos das aulas, e juntar cursos do mesmo assunto num encaixe só (fase 9 do plano-entrada).** Cada pasta de Meus arquivos ganha uma caixa de marcar; a seleção `selecaoCursos` fica em `frontend/src/cursos.js` (escopo de módulo) porque a tela remonta a cada troca de pasta — marcar cursos de grupos diferentes exige navegar. Com uma ou mais pastas: **"Encaixar numa trilha"** (seletor de trilha + pedido opcional) ou **"Criar trilha a partir destes cursos"** (nome + pedido; `POST /api/trilhas` cria a trilha vazia em rascunho e o mesmo encaixe roda nela); depois, "Encaixe pedido na trilha X" com link para `#/trilhas/<id>#encaixar`. O material **`curso`** tem `referencia` = `telegram:<link do grupo>::<caminho da pasta>` por pasta, separadas por ` | ` (`lerCursos`/`escreverCursos`), `titulo` = nomes com " + ", `localizador` vazio. No pedido, `backend/cursos.go` remonta a árvore em Go (`arvoreDosArquivos` — mesma montagem do `pastas.js`: salvos `tudo` na ordem das mensagens, dedup do disco, pastas só de disco por último em ordem alfabética) e escreve `material/01-curso.md`: `##`/`###` por nível e `- A<n> · <nome> (N min)`, números seguidos entre os cursos. A proposta guarda `aulas` (`A<n>` → `telegram:<link>#<msg>`) e `nomes` (`A<n>` → nome) — **o mapa gravado é o que vale na aprovação**, mesmo se a árvore mudar. A regra `encaixe.md` ("Curso em vídeo pelos títulos") manda a IA localizar só por números, não chutar título genérico (vai para `Precisam ser ouvidas:` nas notas) e não duplicar conceito entre cursos. `limparEncaixe` tira encaixe/tópico cujo localizador não tem `A<n>` (`lerAulas`: números, faixas com hífen/travessão, listas — exige o prefixo `A`, então "cap. 4" não conta); `aprovarEncaixe` expande cada número em um material `streaming` (referência do mapa, título = nome da aula, `localizador` = `A<n>`, papel do encaixe; `principal` nos tópicos novos), na ordem, e registra nas notas os números fora do mapa. Na proposta, o localizador mostra os **nomes** das aulas pelo mapa `nomes`, e "Precisam ser ouvidas" vira uma lista com "Processar estas aulas" **desativado** até a fase 10. Sem baixar nem transcrever — a IA só vê títulos e durações. | O encaixe por títulos custa uma chamada de IA e zero bytes de download: o número `A<n>` é a ponte estável entre o que a IA leu e a mensagem do Telegram que vira material — e o mapa gravado na proposta torna a aprovação imune a mudanças na árvore entre pedir e aprovar. |
| T44 | **Processar aulas do Telegram na Entrada: a Trilha pede o download, transcreve no Whisper do Mac e fragmenta como um vídeo do YouTube (fase 10 do plano-entrada).** `POST /api/entrada/telegram` `{refs, fragmentar}` cria uma fonte `telegram` por aula (`video_id` = a ref estável `telegram:<link>#<msg>`, título e `canal` = grupo › curso › módulo do catálogo do Streaming; ref repetida devolve a mesma fonte). Na fila da Entrada (`lerTelegram`), a fonte pergunta o estado da aula: `baixada` → transcreve; `no_telegram` → `POST /api/arquivos/aula/baixar` (só aquela mensagem) e a fonte fica **`baixando` — a fila segue para a próxima** e uma sonda (15 s) a promove de volta à `na_fila`; `na_fila` (pedido já aberto) idem. `baixada` → estado `transcrevendo`: o **ffmpeg (`FFMPEG_BIN`) lê o vídeo direto da url interna do Streaming** (`STREAMING_URL/api/video/proprio/<id>.mp4`, não o link do navegador), mono 16 kHz FLAC em pedaços de 10 min (como o `podcast.go` do Acervo), um `POST` por pedaço no Whisper (`WHISPER_URL`, nunca o Claude), somando o começo de cada pedaço nos `[mm:ss]` — `entrada/<id>/transcricao.md`, áudio apagado ao sair. Sem `WHISPER_URL` → erro "o Whisper não está configurado"; falha do Whisper → a fonte vai a `erro` com "Tentar de novo" (`/tentar` recomeça). `fragmentar` = coluna da fonte: com ela, `pronta` → `pedirFragmentar` automático. No encaixe, a fonte vira material **`streaming` com `referencia` telegram: e `localizador` `mm:ss–mm:ss`** (`enviar` em `destinos.go`); `arquivosDoMaterial` lê só o trecho da transcrição (`trechosDoVideo` acha a fonte pelo `video_id`); `trechosPrincipais` aceita o streaming+telegram → o "visto" e a conclusão da etapa são os do trecho do YouTube (T39). Na tela: **"Processar na Entrada"** na pasta (todas as aulas dela, com a contagem e o tempo de áudio) e em cada aula de Meus arquivos, diálogo com "Dividir em fragmentos quando terminar" marcado; a processada mostra **"na Entrada"** com link para a fonte (mapa `/api/entrada` por `video_id`); lista da Entrada com ícone `telegram` e os estados `baixando`/`transcrevendo`; "Processar estas aulas" da proposta `curso` (fase 9) manda as refs do mapa `aulas` ao mesmo endpoint. O comum entre `Videoaula` e `TrechoYouTube` virou `BarraTrecho.vue` + `estadoDoTrecho`/`pctDoTrecho`/`intervaloDoMaterial`/`chaveDoTrecho` em `trechoYoutube.js`: o material streaming com intervalo toca na `Videoaula` a partir do início do trecho, com a mesma barra e o mesmo aviso "Trecho visto". | A aula do Telegram ganha o mesmo caminho do YouTube sem baixar nada antes do clique: a sonda desacopla o download da fila de leitura, e a referência estável `telegram:` faz a fonte, o material e o "visto" sobreviverem a um novo download. |
| T45 | **Design v4 do Hub na Trilha e Meus arquivos diz onde cada aula já está (20/11/2026).** A Trilha copiou a quarta versão do design system do Hub (H16 de lá): sem barra no topo (a marca abre a lateral, o tema fica no pé dela; no celular, barra no pé com Menu), cabeçalho numa linha (`.topo-pagina`), página até 1200 px com painel lateral (`.duas-colunas`) e quadros com rótulo. A Entrada seguiu: na aba Vídeos o campo do link fica na linha do título e "Do Acervo" vai para o painel; em Meus arquivos a raiz é uma grade de cartões (curso, resumo, barra do que está baixado) e a pasta tem a lista ao centro (pastas e aulas em quadros) e, ao lado, "Esta pasta" (números), "Nas trilhas" e "O que fazer com ela" (criar igual, encaixar só esta pasta sem marcar, processar, baixar, apagar, com uma linha explicando cada um). As pastas marcadas aparecem numa barra presa no pé, perto de onde se marca. `GET /api/arquivos/usos` (`usos.go`) devolve cada material `telegram:` com trilha e etapa e os encaixes de curso `gerando`/`aguardando`; `usos.js` (`usosPorAula`, `trilhasDaPasta`, `encaixesDaPasta`) mostra na aula a trilha (link para a etapa), na pasta "virou a trilha X" (todas as aulas nela) ou "N em X", e "encaixe esperando" (âmbar) até a aprovação. |
| T46 | **Modo automático da Entrada e o mesmo curso em várias trilhas (20/11/2026).** O vídeo que chega com `automatico: true` (a extensão do Chrome, o colar da Entrada e o compartilhar do celular) é lido, **dividido** (Haiku) e já recebe a **sugestão de trilhas** (Haiku, só os resumos) sem clique: a sugestão roda sobre os fragmentos *sugeridos* e as trilhas marcadas ficam com eles quando o Lucas confirma. O **encaixe** (Sonnet) continua só depois do "Mandar para as trilhas" e vira proposta (T12): é a parte cara e a que mexe na trilha. Vídeo de mais de **3 h** não é dividido sozinho (a divisão lê a transcrição inteira; um curso de 13 h são ~200 mil tokens): fica pronto com uma nota. A lista da Entrada mostra "→ trilhas". Em Meus arquivos, "Encaixar em trilhas" aceita várias trilhas de uma vez (um encaixe por trilha; cada um só pega as aulas do recorte daquela trilha) e o selo do download (baixada · parte · só no Telegram) fica em destaque. |
---

## 8. Relação com os outros apps

- **Acervo:** a aba Estudar de cada Tema continua lá. As **fases 2–5 do `docs/TRILHAS.md` do Acervo
  passam para este app**; a **fase 1** de lá (aulas do Telegram → Referências) continua lá e é útil
  aqui (as aulas organizadas viram material de apoio). As perguntas em aberto de lá ficam
  respondidas aqui: critério de conclusão **por etapa**; trilha **com prazo e meta**; capítulo de
  livro **pode** ser a única fonte de uma etapa. *(Pendente: a nota no `TRILHAS.md` do Acervo.)*
- **Biblioteca:** material de apoio e espinha dorsal (entrada C). Upload de PDF vai para lá.
- **Flashcards:** recebe o texto da etapa por `POST /api/textos`.
- **Streaming:** futuro (T16).
- **Hub:** ganha o app na tela Aplicações e na arquitetura. *(Pendente.)*

---

## 9. Fases

| Fase | Entrega | Estado |
|---|---|---|
| **0. Documentação** | este arquivo, PEDAGOGIA, DESIGN-SYSTEM, `CLAUDE.md` | ✅ 14/11/2026 |
| **1. Esqueleto** | Go + Vue + PWA + **design system novo**, compose, Dockerfile, deploy; `/api/health`; vitrine do design em `#/design` | ✅ 14/11/2026 (no ar em https://trilha.internal desde 14/11/2026) |
| **2. Trilha manual** | criar trilha, módulos e etapas à mão; ler lições em Markdown; progresso; **meta, cronograma e tela Hoje**; documentos e importação por pacote; **trilha piloto criada com o processo iterativo** (T20) | ✅ 14/11/2026 (trilha piloto importada no homelab em 14/11/2026) |
| **3. IA básica** | fila + Claude + regras editáveis; **propostas com pedidos e comentários**; lição sob demanda com profundidade e conceitos anteriores (T24) | ✅ 14/11/2026 (primeira lição com o Claude de verdade gerada no homelab em 14/11/2026, a da etapa 1 da piloto: 83 s, ~2.800 palavras, no molde, links ok) |
| **4. Criação com IA** | entradas A e B: pré-pesquisa, entrevista, mapa, recorte, nível, aprovação | ✅ 14/11/2026 (T25; testada com o Claude de mentira; falta a primeira trilha com o Claude de verdade e o mini-teste de nível) |
| **5. Estudar a etapa** | exercícios, simulado, critério de conclusão, envio ao Flashcards | ✅ 14/11/2026 (T26; testada com o Claude de verdade numa trilha de teste, apagada depois: 9 exercícios em 30 s, 5 perguntas em 20 s, correção de 3 respostas em 15 s com notas 95/35/5 que separaram certo, incompleto e erro de conceito; o envio ao Flashcards foi testado com um Flashcards de mentira e a ligação real conferida pela lista de baralhos) |
| **6. Material de apoio** | Biblioteca (com a entrada C), Acervo, web, PDF; área do Streaming desativada | ✅ 14/11/2026 (T27; testada com o Claude de mentira e uma Biblioteca e um Acervo de mentira, de ponta a ponta e no celular; falta usar com o Claude de verdade e os apps de verdade) |
| **7. Domínio e revisão** | revisão espaçada das etapas, simulado intercalado, reforço, pontos fracos, painel; e o mini-teste de nível (T9) | ✅ 15/11/2026 (T28; testada com o Claude de mentira, de ponta a ponta e no celular; falta usar com o Claude de verdade) |
| **Depois** | Streaming de verdade (T16: espera legendas nos vídeos de "Meus vídeos"; as pastas já existem, S42 de lá); ferramentas práticas por trilha (T11: só quando o Lucas pedir, uma trilha por vez); explicar com as próprias palavras (Feynman) | |

A trilha piloto é usada **de verdade** a partir da fase 3; os ajustes vêm do uso (validar à mão
antes de automatizar, como no Acervo).

---

## 10. Trilha piloto: "Home lab: rede"

- **Material:** `estudo/` (01 OpenWrt, 02 Wi-Fi, 03 redes separadas e firewall, 05 Claro bridge/WAN/SQM,
  a parte de IP fixo do 06) e `casa-inteligente-rede.md`.
- **Recorte:** só rede (firmware do roteador, Wi-Fi, bridges, VLAN/redes separadas, DHCP, firewall,
  WAN, NAT/CGNAT, SQM, IP fixo). Aplicações e containers ficam fora.
- **Em aberto:** entram também as partes de rede que estão no Hub (DNS do AdGuard, `*.internal`,
  Caddy com CA própria, Tailscale)?
- **Feita (v3, 14/11/2026):** 15 etapas em 5 módulos (Fundamentos, Wi-Fi, Redes separadas e
  firewall, Internet, Diagnóstico), 6,7 h; 4 lições próprias onde o material tinha lacunas
  (roteamento, DNS, NAT, diagnóstico). Três rodadas de revisão; o histórico está em
  [CRIACAO-DE-TRILHAS.md](CRIACAO-DE-TRILHAS.md).

A documentação do homelab **continua sendo escrita em `estudo/`**, como antes.

---

## 11. Riscos (honestidade)

1. **Construir a plataforma em vez de estudar.** Fatias finas; a trilha piloto em uso desde a fase 3.
2. **A IA erra ordem, curadoria e fatos.** Tudo é proposta com comentários; materiais confiáveis
   ancoram a lição; fontes citadas e conferidas.
3. **Links inventados.** O servidor testa cada URL antes de salvar.
4. **Livro inteiro não cabe na IA.** Trabalhar por sumário e capítulo.
5. **Medir domínio é aproximado.** O progresso mostra sinais (notas, exercícios, cartões), não um
   número mágico.
6. **Limite da assinatura.** Lição sob demanda, cache, modelo por tarefa.

---

## 12. Em aberto

- [ ] Partes de rede do Hub entram na trilha piloto? (§10)
- [ ] Nota no `TRILHAS.md` do Acervo e entrada do app no Hub (§8).
- [x] Entrada `trilha.internal` (e https://192.168.1.20:8452) no Caddy do homelab e primeiro
      deploy, do Mac, em 14/11/2026. Trilha piloto importada; os arquivos de criação dela ficam em
      `/srv/data/trilha/piloto` na VM (fora do GitHub, com backup diário).
- [ ] Primeiras lições com o Claude de verdade na trilha piloto; ajustar `regras/licao.md` pelo
      que sair (validar à mão antes de automatizar). *A da etapa 1 ("Camadas e endereços") está
      gerada e aguardando a leitura do Lucas (14/11/2026). As notas dela apontam dois pontos do
      documento 01: o flashcard da seção 9 diz "4 camadas" e a tabela da 2.1 tem 5; a seção 2.1
      põe a falta de IP pelo DHCP na "camada 3" (o DHCP é aplicação sobre UDP).*
- [ ] Primeira trilha criada com a IA de verdade (fase 4), para ajustar `entrevista.md`, `mapa.md`
      e `plano.md` pelo que sair.
- [x] Mini-teste de nível (T9), opcional na criação (fase 7, T28).
- [ ] Usar a fase 7 de verdade: as primeiras revisões da trilha piloto, um simulado intercalado e um
      reforço e um mini-teste com o Claude de verdade, para ajustar `reforco.md` e `nivelamento.md`;
      ver se a escada de intervalos (1, 3, 7, 14, 30, 60, 120 dias) e o "60% conta como sabe" do
      mini-teste fazem sentido.
- [ ] Exercícios e simulado da trilha piloto no uso real (fase 5), para ajustar `exercicios.md`,
      `simulado.md` e `corrigir-simulado.md`; ver se a nota mínima de 80% faz sentido e mandar a
      primeira lição ao Flashcards de verdade.
- [ ] Aplicar nos documentos de `estudo/` as três correções propostas pela trilha piloto
      ([CRIACAO-DE-TRILHAS.md](CRIACAO-DE-TRILHAS.md)).
- [x] Endpoints de leitura do Acervo que a Trilha usa (fase 6, T27): `GET /api/temas`,
      `/api/pacotes/{tema}`, `/api/referencias` e `/api/referencias/{id}`. Da Biblioteca:
      `GET /api/books`, `/api/books/{id}/chapters`, `/text`, `/anchor`, `/cover` e `POST /api/books`.
- [ ] Primeira trilha a partir de material com o Claude de verdade (um livro da Biblioteca), e
      o primeiro "Sugerir material", para ajustar `espinha.md` e `materiais.md`; ver se o limite de
      60 mil caracteres por capítulo na espinha basta para a IA julgar o que é raso.
