# Fase 4 — Para que trilhas vai cada fragmento, e em que etapa

## Objetivo

Com os fragmentos confirmados (fase 3):

1. A IA **sugere o destino** de cada fragmento: uma ou mais trilhas existentes, uma trilha nova
   (com um nome) ou nenhuma. Uma chamada por fonte, olhando só os títulos/resumos dos fragmentos e um
   resumo de cada trilha (barato, `modelo_entrada`).
2. O Lucas confirma, tira, acrescenta trilhas à mão (pode ser mais de uma), cria uma trilha nova ou
   marca "sem destino".
3. **"Mandar para as trilhas"**: para cada trilha escolhida, um **encaixe** (o que já existe, T32)
   com os fragmentos que vão para ela. O encaixe diz a etapa de cada fragmento ou propõe etapas
   novas no lugar certo da ordem. O Lucas aprova o encaixe na trilha, como já faz hoje.

## Leia antes

- `README.md`; fases 1 a 3.
- `backend/encaixe.go` inteiro (pedido, esquema, `materialDoEncaixe`, `limparEncaixe`,
  `aprovarEncaixe`, `inserirEtapa`, `lugarNoModulo`) e `regras/encaixe.md`.
- `backend/material.go`: `arquivosDoMaterial` (cada tipo de material escreve o seu texto na pasta da
  tarefa) e `textoDoAcervo` (exemplo de um tipo).
- `backend/trilhas.go`: `material` (struct), `tiposMaterial`/validação de tipos, `criarTrilha`.
- `frontend/src/EncaixarMaterial.vue` (como a proposta de encaixe aparece e é aprovada).

## O que fazer

### 4.1 Tipo de material novo: `youtube`

- `material.tipo = "youtube"`, `referencia` = `https://www.youtube.com/watch?v=<id>` (sempre esse
  formato), `localizador` = um ou mais intervalos `mm:ss–mm:ss` separados por `; ` (ex.:
  `06:00–10:00` ou `06:00–10:00; 25:10–31:00`). Aceite também hífen comum (`06:00-10:00`).
- Acrescente `youtube` aos tipos aceitos (trilha, importação, encaixe: `tiposEncaixe`).
- Em `arquivosDoMaterial`, o caso `youtube`: acha a fonte pela `url` (fontes do tipo `youtube` **ou**
  `acervo` com a mesma url), lê `entrada/<id>/transcricao.md` e escreve **só os trechos** dos
  intervalos, cada um com um cabeçalho `## 06:00–10:00 · <título do fragmento, se houver um com esse
  intervalo> ` seguido do resumo do fragmento e das linhas `[mm:ss]` daquele intervalo. Sem fonte:
  erro "o vídeo não está na Entrada". Função pura `trechoDaTranscricao(transcricao, inicio, fim)`
  com teste.
- Funções puras `lerIntervalos(localizador) ([][2]int, error)` e `escreverIntervalos(...)` com teste.

### 4.2 A tarefa `destino`

- Em `fragmentos`, estados novos: `sem_destino` e `enviado`. Coluna nova `sem_destino_em TEXT
  NOT NULL DEFAULT ''` (a fase 6 usa). Tabela nova:

```sql
CREATE TABLE IF NOT EXISTS fragmento_destinos (
	fragmento_id INTEGER NOT NULL REFERENCES fragmentos(id) ON DELETE CASCADE,
	trilha_id    INTEGER NOT NULL REFERENCES trilhas(id) ON DELETE CASCADE,
	proposta_id  INTEGER REFERENCES propostas(id) ON DELETE SET NULL, -- o encaixe criado
	criado_em    TEXT NOT NULL,
	PRIMARY KEY (fragmento_id, trilha_id)
);
```

- Em `fontes`, coluna `tarefa_destino INTEGER` (a tarefa de sugerir destino).
- `POST /api/entrada/{id}/destino` (pedido opcional): enfileira a tarefa `destino` para os fragmentos
  `confirmado` da fonte (os `opiniao` vão também; a IA costuma dizer "nenhuma").
- Executor: `pedido.md` com cabeçalho `# Pedido: destino`, os fragmentos (número, tempo, título,
  resumo, assuntos, tipo) e **as trilhas** (todas que não estão `concluida`): id, título, objetivo,
  e uma linha por módulo com o título do módulo e os títulos das etapas (corte em ~40 etapas por
  trilha, com "…"). Regra `regras/destino.md`, modelo `modeloPara(ctx, "destino")` (= `modelo_entrada`).
- Esquema:

```json
{"type":"object","properties":{
 "destinos":{"type":"array","items":{"type":"object","properties":{
   "fragmento":{"type":"integer","description":"o número do fragmento no pedido"},
   "trilhas":{"type":"array","items":{"type":"object","properties":{
      "trilha_id":{"type":"integer"},"motivo":{"type":"string"}},"required":["trilha_id","motivo"]}},
   "trilha_nova":{"type":"string","description":"nome de uma trilha nova, se nenhuma serve e o assunto merece; senão vazio"},
   "motivo_nenhuma":{"type":"string","description":"se não vai a lugar nenhum (opinião, fora do tema), por quê; senão vazio"}},
   "required":["fragmento","trilhas","trilha_nova","motivo_nenhuma"]}},
 "notas":{"type":"string"}},
 "required":["destinos","notas"]}
```

- O servidor tira trilha que não existe e grava a sugestão em `fragmentos.sugestao` (JSON).

### 4.3 A regra `regras/destino.md` (nova)

Diga à IA: o mesmo fragmento pode ir a **várias** trilhas quando serve de verdade a cada uma
(ex.: um trecho sobre S3 vai para "AWS para engenharia de software" e para "System Design"); não
espalhe por generosidade. Existe a trilha **"Conhecimento de base"** para conhecimentos soltos de
tecnologia que não pertencem a nenhuma trilha grande (o que é UTF-8, diferença entre sistemas de
arquivos…): use-a quando o assunto é técnico e útil, mas não cabe nas outras. `trilha_nova` só
quando o assunto é grande e recorrente o bastante para virar trilha (não para um tópico solto).
`opiniao` e `fora` costumam ir a nenhuma.

### 4.4 Escolher e mandar (API)

| Rota | Faz |
|---|---|
| `PUT /api/fragmentos/{id}/destinos` | `{"trilhas":[8,12]}` ou `{"sem_destino":true}`: o que o Lucas escolheu (substitui a escolha anterior que ainda não foi enviada). `sem_destino` grava `sem_destino_em`. |
| `POST /api/entrada/{id}/enviar` | Para cada trilha escolhida em algum fragmento da fonte: **um** encaixe (a função de `handleEncaixar`, reaproveitada, sem duplicar código) com `material = {tipo: "youtube", titulo: "<título do vídeo> (<canal>)", referencia: <url>, localizador: <os intervalos dos fragmentos dessa trilha>}` e um pedido que diz "São N fragmentos deste vídeo; encaixe cada um pelo seu intervalo". Grava o `proposta_id` em `fragmento_destinos` e o fragmento vira `enviado`. Responde a lista de encaixes criados (trilha, proposta). |
| `POST /api/trilhas` | já existe: a tela usa para "Trilha nova" (cria vazia, `rascunho` ou `ativa`, com o título). |

### 4.5 O encaixe com fragmentos

- `regras/encaixe.md`: acrescente uma seção **"Vídeo em fragmentos"**: quando o material é do tipo
  `youtube`, o texto traz um bloco por fragmento; o `localizador` de cada encaixe ou tópico novo é
  **exatamente um** desses intervalos (ou um pedaço dele), nunca um tempo inventado; um fragmento
  pode encaixar em mais de uma etapa da mesma trilha só se ensina as duas de verdade.
- **Trilha vazia** (nova, sem módulos): o encaixe tem de funcionar e propor só `novos`, com módulos
  novos. Teste isso; se `limparEncaixe` ou `inserirEtapa` não aceitarem trilha sem módulo, conserte.
- `limparEncaixe`: para `youtube`, tira o encaixe cujo localizador não cai dentro de um dos
  intervalos mandados.
- Ao aprovar, o material vai para a etapa com o `localizador` de um intervalo só e `papel` como
  a IA disse (`principal` = o trecho é a aula da etapa).

### 4.6 Tela

Na página da fonte, depois de confirmar os fragmentos:

- **"Sugerir trilhas"** (pedido opcional) → estado da tarefa.
- Em cada fragmento: as trilhas sugeridas já marcadas (com o motivo ao passar o dedo/mouse), um
  seletor "+ trilha" com todas as trilhas, "Trilha nova" (pede o nome, sugere o da IA, cria e marca)
  e "Sem destino".
- **"Mandar para as trilhas (N encaixes)"**. Depois de mandar, cada fragmento mostra para onde foi
  e o estado do encaixe (aguardando aprovação, aprovado, descartado), com link para a aba "Encaixar
  material" da trilha (onde o Lucas aprova).
- Na lista da Entrada, cada fonte mostra um resumo: "5 fragmentos · 3 enviados · 1 sem destino".

## Testes obrigatórios

- `lerIntervalos`/`trechoDaTranscricao` (vários formatos, `h:mm:ss`, intervalo fora da duração).
- Tarefa `destino` com o Claude falso (acrescente `# Pedido: destino` no `claude-falso.mjs`, que
  devolve a primeira trilha do pedido para cada fragmento `conteudo` e nenhuma para `opiniao`).
- `PUT .../destinos` e `POST .../enviar`: dois fragmentos para a trilha A e um para A e B → **dois**
  encaixes (A com dois intervalos, B com um); `fragmento_destinos` preenchido.
- O pedido do encaixe (`montarPedidoEncaixe`) com material `youtube` traz só os trechos dos
  intervalos, com os títulos dos fragmentos.
- Encaixe numa trilha **vazia** com o Claude falso: aprovar cria módulo e etapas com o material
  `youtube` principal.
- Atualize os testes de `encaixe_test.go`/`propostas_test.go` que mudarem.

## Não fazer

- Não tocar o vídeo na etapa (fase 5). Não aprovar encaixe sozinho (o Lucas aprova).
- Não criar um encaixe por fragmento: é **um por trilha**, com todos os fragmentos dela (custo).

## Pronto quando

Testes passam; decisão **T38 — Destino dos fragmentos e encaixe de vídeo em fragmentos**; regras
`destino.md` e `encaixe.md` atualizadas; `CLAUDE.md` atualizado.

Commit: `Entrada (T38): a IA sugere as trilhas de cada fragmento; um encaixe por trilha com os trechos do vídeo (material youtube)`

## Dúvidas

## Feito

**Feito em 19/11/2026.**

- Banco (`backend/banco.go`): tabela `fragmento_destinos` (`fragmento_id`, `trilha_id`,
  `proposta_id` nulo, `criado_em`, PK composta) e os `ALTER TABLE` — `fragmentos.sem_destino_em`
  e `fontes.tarefa_destino`/`pedido_destino` (a tarefa de destino e o pedido do Lucas ficam
  separados dos de fragmentar; as notas da IA continuam em `fontes.notas`, nunca em `erro`).
- Material `youtube` (`backend/destinos.go`, `material.go`, `trilhas.go`): novo tipo aceito no
  encaixe e na trilha (`tiposEncaixe`, `tiposMaterial`). `lerIntervalos`/`escreverIntervalos`
  leem/gravam `mm:ss–mm:ss; ...` (também `h:mm:ss`, hífen comum e travessão; rejeita formato
  errado e fim ≤ início) e `trechoDaTranscricao` corta a transcrição em blocos `[mm:ss]` dentro
  do intervalo (fim que passa da duração é aceito). `trechosDoVideo` acha a fonte pela URL
  (`youtube`/`acervo`), lê `entrada/<id>/transcricao.md` e monta uma seção `## início–fim ·
  título do fragmento` + resumo por intervalo; sem fonte, devolve "o vídeo não está na Entrada".
- Tarefa `destino` (`destinos.go`, executor `sugerirDestinos` na fila): `POST
  /api/entrada/{id}/destino` (fonte `fragmentada`, aceita `{"pedido"}`) escreve `pedido.md`
  (`# Pedido: destino` + vídeo, pedido, fragmentos confirmados — inclusive `opiniao`/`fora` — e
  as trilhas não concluídas com título, objetivo, módulos e até ~40 etapas, com aviso de corte)
  e chama o Claude com `regras/destino.md` (nova), só `Read` e `modelo_entrada` (`modeloPara`
  ganhou `destino`). A resposta vira `fragmentos.sugestao`; trilhas sugeridas entram como
  `fragmento_destinos` no estado `sugerido`. Falha da tarefa vai para a fonte como erro; as
  notas da IA vão para `notas`.
- Escolher e mandar (`destinos.go`, rotas em `servidor.go`): `PUT /api/fragmentos/{id}/destinos`
  recebe `{"trilhas":[ids]}` (valida que existem, troca só os destinos não enviados, marca
  `confirmado` e limpa `sem_destino_em`) ou `{"sem_destino":true}` (marca `sem_destino` com o
  carimbo); devolve o fragmento com `destinos` (trilha + proposta + estado do encaixe) e
  `sugestao`. `POST /api/entrada/{id}/enviar` agrupa os confirmados por trilha, cria **um
  encaixe por trilha** (material `youtube` com título/canal da fonte, URL e todos os intervalos
  escolhidos; pedido "São N fragmentos deste vídeo; encaixe cada um pelo seu intervalo."),
  guarda `proposta_id` nos destinos e marca `enviado` só quem não tem destino pendente; devolve
  `[{trilha_id, titulo, proposta_id}]`. `GET /api/entrada/{id}` passa a trazer `destinos`/
  `sugestao`/`sem_destino_em` nos fragmentos, `tarefa_destino` e a lista ganha
  `resumo_fragmentos` ("3 fragmentos · 2 enviados").
- Encaixe (`backend/encaixe.go`, `propostas.go`): `handleEncaixar` extraído — `criarEncaixe`
  recebe trilha/material/pedido/ctx e devolve a proposta, usado pelo handler e pelo enviar (sem
  duplicar). `montarPedidoEncaixe` escreve os trechos `youtube` em `material.md` (via
  `arquivosDoMaterial`); `limparEncaixe` rejeita encaixe cujo localizador não cabe nos
  intervalos mandados (`lerIntervalos` no material e no encaixe). Aprovar insere cada
  `localizador` como entrada de material `principal` na etapa; trilha vazia vale para `youtube`
  (o encaixe propõe módulos/etapas novos, criados pelo título na aprovação via
  `inserirEtapa`/`moduloOuNovo`). Regra `regras/encaixe.md` ganhou a seção "Vídeo
  em fragmentos" (localizador dentro dos intervalos, não inventar tempos, mesmo fragmento em
  duas etapas só se ensinar as duas, `principal` = o trecho é a aula).
- Tela (`telas/Fonte.vue`, `rotulos.js`, `ListaEntrada.vue`): seção "Destino" após confirmar —
  "Sugerir trilhas"/"Sugerir de novo" com pedido e estado da tarefa ("Tentar de novo"), e em
  cada fragmento os chips "Foi para" (link `#/trilhas/<id>#encaixar` com o estado do encaixe),
  trilhas sugeridas com o motivo da IA, "+ trilha" para escolher outra, "Trilha nova" (cria e já
  seleciona), "Sem destino" e a nota quando a IA não viu destino. "Mandar para as trilhas (N
  encaixes)" agrupa por trilha. `ESTADOS_ENCAIXE`/`rotulos` para "aguardando aprovação" etc. A
  lista da Entrada mostra o `resumo_fragmentos`. `scripts/claude-falso.mjs`: casos `# Pedido:
  destino` (primeira trilha do pedido para `conteudo`, nenhuma para `opiniao`) e o encaixe com
  material `youtube` (uma etapa por intervalo).
- Conserto à parte: `.cabecalho` (style.css) ganhou `flex-wrap: wrap` — com barra de rolagem no
  celular, título de palavra longa + etiqueta esticavam a grade da página e geravam 7 px de
  rolagem horizontal na fonte (e em qualquer cabeçalho apertado).

**Testes** (`backend/destino_test.go`): `lerIntervalos`/`escreverIntervalos` (formatos, erros,
`h:mm:ss`) e `trechoDaTranscricao` (trecho certo, cabeçalhos, fora da duração); tarefa `destino`
com o `iaFalsa`/`claude-falso` → fragmentos de conteúdo com a primeira trilha, opinião sem, erros
em `erro` e notas em `notas`; `PUT .../destinos` (várias trilhas, sem destino, trilha que não
existe, destino já enviado intocado) e `POST .../enviar` (dois fragmentos na trilha A + um em A e
B → **dois** encaixes, `fragmento_destinos` com `proposta_id`, `enviado` só nos com destino);
pedido do encaixe `youtube` com só os intervalos e os títulos dos fragmentos; encaixe em trilha
**vazia** → aprovar cria módulo e etapas com material `youtube` `principal`; `limparEncaixe`
descartando localizador fora dos intervalos; `modeloPara(ctx, "destino")` = `modelo_entrada`.

**Rodado e passou:** `cd backend && go vet ./... && go test ./...` (ok); `cd frontend && npm
test` (38 ok) `&& npm run build` (ok). Conferido no navegador a 360 px com o app de verdade
(yt-dlp e Claude falsos): fragmentar → confirmar → sugerir destinos (conteúdo → Redes, opinião
sem) → "+ trilha" com duas trilhas → "Mandar" criou 2 encaixes `aguardando` com os links certos;
sem rolagem horizontal depois do `flex-wrap` no `.cabecalho`.

**Ficou de fora:** tocar o trecho na etapa e a contagem de enviados na trilha (fase 5); o
`sem_destino` volta a ser considerado em futuras sugestões só se o Lucas pedir de novo.
