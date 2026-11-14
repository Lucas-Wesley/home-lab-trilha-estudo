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
| **Concluir** | pelo critério da etapa (ex.: exercícios ≥ 80%) **ou** à mão; pular é sempre permitido |

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
| **Biblioteca** | escolher livro → capítulos (`GET /api/books/{id}/chapters`); texto para a IA (`/text?chapters=`); abrir a leitura no ponto | ✅ API já existe (R33 de lá) |
| **Acervo** | escolher Tema ou Referência; texto como fonte; link de volta | ✅ `GET /api/temas` e `/api/arquivos/…` existem; confirmar o uso |
| **PDF próprio** | upload **para a Biblioteca**, e a Trilha liga (ganha busca, capítulos e leitor de graça) | ✅ |
| **Web** | a IA busca; o **servidor confere se cada URL abre** antes de salvar | ✅ |
| **Docs próprios** (ex.: `estudo/` do homelab) | arquivos `.md` **da trilha** (T23), adicionados à mão na criação (qualquer entrada) ou depois, na página da trilha | ✅ |
| **Streaming** | botão "Vincular pasta de Meus vídeos" e área no material da etapa, **visíveis e desativados** ("em breve") | 🔜 quando o Streaming tiver subpastas e legendas |

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
| **5. Estudar a etapa** | exercícios, simulado, critério de conclusão, envio ao Flashcards | |
| **6. Material de apoio** | Biblioteca (com a entrada C), Acervo, web, PDF; área do Streaming desativada | |
| **7. Domínio e revisão** | revisão espaçada das etapas, simulado intercalado, reforço, pontos fracos, painel | |
| **Depois** | Streaming de verdade; ferramentas práticas por trilha | |

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
- [ ] Mini-teste de nível (T9), opcional na criação.
- [ ] Aplicar nos documentos de `estudo/` as três correções propostas pela trilha piloto
      ([CRIACAO-DE-TRILHAS.md](CRIACAO-DE-TRILHAS.md)).
- [ ] Endpoints de leitura do Acervo que a Trilha vai usar (confirmar na fase 6).
