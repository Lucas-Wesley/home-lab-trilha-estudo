# Fase 9 — Encaixar cursos numa trilha pelos títulos; juntar cursos do mesmo assunto

## Objetivo

1. Em **Meus arquivos**, numa pasta (curso ou módulo): **"Encaixar numa trilha"** → escolhe a
   trilha → a IA lê **só os títulos** das aulas (curso › módulo › aula, com a duração), **sem baixar
   nem transcrever**, e propõe em que etapa cada aula (ou grupo de aulas) entra e que etapas novas
   faltam. É o encaixe de sempre (T32), com um material novo, `curso`. Barato: são só títulos.
2. Aula com nome que não diz nada ("Aula 3", "Parte 2") a IA **não chuta**: lista como "precisa
   ouvir" nas notas, e a tela oferece **"Processar estas aulas"** (fase 10).
3. **Juntar cursos do mesmo assunto**: marcar **várias pastas** (ex.: "Elemar JR › 4. DDD do Jeito
   Certo" e "MBA › 009 - DDD") e encaixar todas **num encaixe só** (a IA tira as repetições e põe em
   ordem; a mesma etapa pode ganhar uma aula principal e outra de apoio). Ou **"Criar trilha a partir
   destes cursos"**: cria a trilha vazia com o nome escolhido e faz o mesmo encaixe nela (a fase 4
   garantiu que encaixe em trilha vazia funciona).

## Leia antes

- `README.md`; fases 4, 7 e 8.
- `backend/encaixe.go` (pedido, esquema, `conteudoEncaixe`, `limparEncaixe`, `aprovarEncaixe`),
  `regras/encaixe.md`, `backend/material.go` (`arquivosDoMaterial`).
- `frontend/src/pastas.js` e `MeusArquivos.vue` (fase 7); a referência `telegram:` da fase 8.

## O que fazer

### 9.1 Material `curso`

- `tipo = "curso"`, `titulo` = os nomes das pastas (`"DDD do Jeito Certo + 009 - DDD"`),
  `referencia` = uma ou mais pastas, cada uma como `telegram:<link do grupo>::<caminho da pasta na
  árvore>` (o caminho como em `pastas.js`, com `/`), separadas por ` | `. Funções puras
  `lerCursos(referencia)` / `escreverCursos(lista)` com teste. `localizador` vazio.
- Em `arquivosDoMaterial`, o caso `curso`: lê a árvore no Streaming (`GET /api/meus-videos`, a mesma
  montagem de árvore da fase 7, feita em Go: escreva `arvoreDosArquivos(...)` no servidor seguindo a
  mesma regra de `pastas.js`, com teste) e escreve o **sumário**: para cada pasta, os níveis como
  títulos (`##`, `###`) e cada aula como `- A12 · <nome da aula> (12 min)`. O número `A<n>` é
  **único no material todo** (conta seguido entre os cursos).
- Guarde o **mapa** dos números: `conteudoEncaixe` ganha `Aulas map[string]string
  json:"aulas,omitempty"` (`"A12": "telegram:https://t.me/c/1#45"`), preenchido em `gerarEncaixe` com
  o mesmo número que foi para o pedido. (A árvore pode mudar entre pedir e aprovar; o mapa guardado
  é o que vale.)
- Acrescente `curso` a `tiposEncaixe`.

### 9.2 Regra

Em `regras/encaixe.md`, uma seção **"Curso em vídeo pelos títulos"**:

- O texto é só o sumário do curso (sem transcrição). O `localizador` de cada encaixe e tópico novo é
  uma lista de aulas pelos números: `A12`, `A12–A15`, `A3, A7`. Nunca invente número.
- Encaixe pelo que o **título** diz com segurança; aula com título que não diz o conteúdo
  (genérico, "Parte 2", "Aula 3", "Live") **não encaixa**: liste em `notas` numa linha começando com
  `Precisam ser ouvidas:` seguida dos números.
- Vários cursos do mesmo assunto: a mesma etapa pode receber aulas de cursos diferentes — a melhor
  como `principal`, as outras como `apoio`; não crie etapas repetidas para o mesmo conceito dito por
  dois cursos.
- Trilha vazia: monte os `novos` seguindo a ordem que faz sentido para aprender (não
  necessariamente a do curso), em módulos.

### 9.3 Aprovar

Em `aprovarEncaixe`, quando o material é `curso`: cada encaixe/tópico novo aprovado vira, na etapa,
**um material `streaming` principal ou de apoio por aula** (o `papel` do encaixe), com `referencia`
= `telegram:...#<mensagem>` do mapa e `titulo` = nome da aula, na ordem dos números (partes). Número
que não está no mapa: ignore e registre nas notas (o servidor já faz isso com o que tira).
`limparEncaixe` tira encaixe cujo localizador não tem nenhum número válido.

### 9.4 Tela

- Em **Meus arquivos**: caixa de marcar em cada pasta (para escolher várias); com uma ou mais
  marcadas, aparecem **"Encaixar numa trilha"** (escolhe a trilha, pedido opcional) e **"Criar trilha
  a partir destes cursos"** (nome da trilha, pedido opcional).
- Depois de pedir: "Encaixe pedido na trilha X" com link para a aba "Encaixar material" da trilha
  (onde a proposta aparece e é aprovada, como sempre).
- Na proposta de encaixe com material `curso` (`EncaixarMaterial.vue`), o localizador `A12–A15`
  aparece com os **nomes** das aulas (pelo mapa `aulas` da proposta), não só os números. Se as notas
  têm `Precisam ser ouvidas:`, mostre as aulas e um botão **"Processar estas aulas"** (deixe
  desativado com "na próxima versão" até a fase 10).

## Testes obrigatórios

- `lerCursos`/`escreverCursos`; `arvoreDosArquivos` em Go com a mesma árvore inventada do
  `pastas.test.js`.
- O pedido do encaixe `curso` com duas pastas de dois grupos: sumário com títulos e `A1…An`
  seguidos; o mapa guardado na proposta.
- Claude falso: caso de encaixe com material `curso` (devolve um encaixe com `A1–A2` e um novo com
  `A3`, e `Precisam ser ouvidas: A4` nas notas).
- Aprovar: a etapa ganha 2 materiais `streaming` `telegram:` (partes, na ordem); o tópico novo é
  criado com o material da aula `A3`; número inexistente é ignorado.
- Encaixe `curso` numa trilha vazia cria módulos e etapas.

## Não fazer

- Não baixar nem transcrever (fase 10). Não mudar o Streaming.

## Pronto quando

Testes passam; decisão **T43 — Encaixar cursos pelos títulos e juntar cursos**; `regras/encaixe.md`
atualizada.

Commit: `Entrada (T43): encaixar cursos do Telegram numa trilha pelos títulos das aulas, e juntar cursos do mesmo assunto num encaixe só`

## Dúvidas

## Feito

- **Material `curso`** (`backend/cursos.go`, novo): `tipo="curso"` entrou em `tiposEncaixe`;
  `referencia` = `telegram:<link do grupo>::<caminho>` por pasta separadas por ` | `
  (`lerCursos`/`escreverCursos`, puras, com teste); validação recusa referência sem pasta
  Telegram válida; trilha vazia aceita para `curso` (como já era para `youtube`).
- **Árvore no servidor**: `arvoreDosArquivos` replica `pastas.js` (salvos `tudo` na ordem das
  mensagens, `grupo = pasta || titulo`, dedup dos vídeos já no catálogo, pastas só de disco por
  último em ordem alfabética); structs do `/api/meus-videos` ganharam `caminho` (aula salva) e
  `pasta`/`nome` (vídeo do disco). Teste com a mesma árvore inventada do `pastas.test.js`.
- **Sumário**: para cada pasta selecionada, `##`/`###` por nível e `- A<n> · <nome> (N min)` com
  números seguidos entre os cursos; vai para `material/01-curso.md` no pedido. Os mapas
  `A<n>` → `telegram:<link>#<msg>` e `A<n>` → nome ficam em `conteudoEncaixe.aulas`/`.nomes` —
  o mapa gravado vale na aprovação.
- **Regra**: seção "Curso em vídeo pelos títulos" em `regras/encaixe.md` (só o sumário;
  localizadores `A<n>`/faixas/listas; nunca inventar número; título genérico vai para
  `Precisam ser ouvidas:`; juntar cursos sem duplicar; trilha vazia em ordem de aprendizado).
- **Aprovar**: `limparEncaixe` tira encaixe/tópico cujo localizador não tem `A<n>` (`lerAulas`
  exige o prefixo `A` — "cap. 4"/"p. 45–60" não contam); `aprovarEncaixe` expande cada número
  em um material `streaming` (referência do mapa, título = nome da aula, `localizador` = `A<n>`,
  papel do encaixe, `principal` nos novos), na ordem, e anota nas notas os números fora do mapa.
- **Tela** (`MeusArquivos.vue`, `frontend/src/cursos.js` novo): checkbox em cada pasta; seleção
  `selecaoCursos` em escopo de módulo (sobrevive à navegação entre pastas); com marcações,
  "Encaixar numa trilha" (seletor + pedido) e "Criar trilha a partir destes cursos" (nome +
  pedido → `POST /api/trilhas` vazia → mesmo encaixe); "Encaixe pedido na trilha X" com link
  para `#/trilhas/<id>#encaixar`. `Trilha.vue` deixou de esconder a aba Encaixar em trilha vazia.
- **Proposta** (`EncaixarMaterial.vue`): localizador `A<n>` do `curso` mostra os **nomes** das
  aulas pelo mapa `nomes`; a linha `Precisam ser ouvidas:` das notas vira uma lista com
  "Processar estas aulas (na próxima versão)" desativado; `TIPOS_MATERIAL` ganhou `curso`.
- **Testes**: `lerCursos`/`escreverCursos`, `lerAulas`, `arvoreDosArquivos` (árvore do
  `pastas.test.js`), encaixe `curso` com duas pastas de dois grupos (sumário `A1…A4` + mapa na
  proposta; falso com `A1–A2` + novo `A3` + `Precisam ser ouvidas: A4`; aprovar → 2 materiais
  `streaming` em ordem na etapa + tópico novo com a aula A3 + `A9` ignorado e registrado) e
  encaixe em trilha vazia criando módulo/etapas. `cursos.test.js` para os helpers JS.
- **Verificação**: `go vet` + `go test` ok; `npm test` 67 ok; `npm run build` ok. Ao vivo a
  360 px com Streaming e claude falsos: marcou pastas em dois grupos (seleção sobrevive à
  navegação), os dois diálogos, "Encaixe pedido na trilha X", proposta com nomes das aulas e
  "Precisam ser ouvidas", aprovação criando as partes `telegram:` na etapa — sem rolagem
  horizontal. Decisão **T43** registrada.
