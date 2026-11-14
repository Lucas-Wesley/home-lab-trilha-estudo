# Trilha de Estudo

App do **ecossistema de apps pessoais do Lucas**. Trilhas de estudo **profundas, personalizadas e
em ordem**: o Lucas diz o que quer aprender, a IA delimita o assunto com perguntas adequadas ao tipo
de assunto, monta ou valida a sequência de etapas, prepara cada lição na profundidade pedida e
acompanha meta, ritmo e domínio.

**Comece lendo:**
- [`docs/DECISOES.md`](docs/DECISOES.md) — **a fonte da verdade.** Vocabulário, fluxos, decisões
  (T1–T26), fases, riscos e o que está em aberto.
- [`docs/PEDAGOGIA.md`](docs/PEDAGOGIA.md) — como a trilha ensina (entrevista por perfil de domínio,
  profundidade em 5 níveis, molde da lição).
- [`docs/DESIGN-SYSTEM.md`](docs/DESIGN-SYSTEM.md) — o design **próprio** desta app (estilo Claude
  Platform). **Não** usar o design system compartilhado das outras apps (T14).

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
- Toda cor, raio e tamanho vem de um token de `style.css`; mudou o design, atualizar
  `docs/DESIGN-SYSTEM.md` e a vitrine `#/design` (sem aba; só pelo endereço).
- Só teoria por enquanto; ferramentas práticas por trilha só quando o Lucas pedir (T11).
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
