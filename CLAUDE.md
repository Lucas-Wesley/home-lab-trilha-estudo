# Trilha de Estudo

App do **ecossistema de apps pessoais do Lucas**. Trilhas de estudo **profundas, personalizadas e
em ordem**: o Lucas diz o que quer aprender, a IA delimita o assunto com perguntas adequadas ao tipo
de assunto, monta ou valida a sequência de etapas, prepara cada lição na profundidade pedida e
acompanha meta, ritmo e domínio.

**Comece lendo:**
- [`docs/DECISOES.md`](docs/DECISOES.md) — **a fonte da verdade.** Vocabulário, fluxos, decisões
  (T1–T18), fases, riscos e o que está em aberto.
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

## Rodar
- `make dev-api` (API em `:8080`, dados em `./data`) e `make dev-web` (Vite com proxy de `/api`).
- `make test` roda os testes do Go e do front.
- `docker compose up --build` sobe a imagem de produção localmente.

## Regras e convenções (IMPORTANTE)
- **Conteúdo (lições, trilhas, propostas) NÃO sobe pro GitHub** — fica no git local do homelab
  (`/srv/data/trilha`). Só o código vai para este repositório.
- **A IA propõe, o Lucas decide.** Toda ação de IA aceita um **pedido** em texto livre, e toda
  proposta aceita **comentários**. Nunca aplicar mudança da IA sem aprovação.
- Toda chamada ao Claude passa pela **fila** (uma tarefa por vez).
- Toda cor, raio e tamanho vem de um token de `style.css`; mudou o design, atualizar
  `docs/DESIGN-SYSTEM.md` e a vitrine `#/design`.
- Só teoria por enquanto; ferramentas práticas por trilha só quando o Lucas pedir (T11).
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
