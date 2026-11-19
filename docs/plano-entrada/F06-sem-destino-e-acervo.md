# Fase 6 — "Sem destino" some em 30 dias; trazer vídeos do Acervo

## Objetivo

1. Fragmento marcado **sem destino** e não mandado a nenhuma trilha some sozinho depois de **30
   dias** (D6). A tela avisa quantos dias faltam.
2. **Trazer do Acervo**: uma Referência de vídeo do Acervo (que já tem a transcrição com os minutos)
   entra na Entrada como fonte **sem baixar de novo** e sem IA para ler a legenda (D7). É assim que
   os 6 vídeos do Akita ("Introdução a Redes", partes 1 a 6) entram no fluxo novo.

## Leia antes

- `README.md`; fases 1, 3 e 4.
- `backend/material.go`: como a Trilha já fala com o Acervo (`a.integra.acervo`, `textoDoAcervo`,
  `handleReferenciasAcervo` → `GET /api/referencias` de lá).
- No Acervo (só leitura, **não mude o Acervo**): `GET /api/referencias/{id}` devolve a Referência
  (`url`, `video_id`, `titulo`, `canal`, `duracao`, `tipo: "video"`) e `textos.cru`, que é a
  transcrição **com `[mm:ss]` a cada ~30 s** (formato de `legendaParaCru` em `acervo/backend/youtube.go`).

## O que fazer

### 6.1 Sem destino

- Uma limpeza que roda ao iniciar o app e depois a cada 6 horas (goroutine com `time.Ticker`, que
  para com o contexto): apaga os fragmentos `sem_destino` com `sem_destino_em` mais antigo que 30
  dias. Fonte que ficar sem nenhum fragmento e sem nada enviado continua (o Lucas apaga se quiser).
- Mandar um fragmento sem destino a uma trilha (fase 4) limpa o `sem_destino_em`.
- Na tela: "sem destino · some em 12 dias".
- O prazo como constante (`diasSemDestino = 30`) e a limpeza como função testável que recebe o
  "agora".

### 6.2 Trazer do Acervo

- `POST /api/entrada/acervo` `{"referencia":"62eb810dfd"}`: lê a Referência no Acervo; só aceita
  `tipo: "video"` com `url` do YouTube e `textos.cru` não vazio. Cria a fonte `tipo = "acervo"`,
  `video_id` = id da Referência, `url` = a url do YouTube normalizada
  (`https://www.youtube.com/watch?v=<id>`), título, canal, duração (converter `"46:58"` em segundos),
  grava a transcrição (`textos.cru`) em `entrada/<id>/transcricao.md` e já deixa a fonte `pronta`.
  A mesma Referência de novo devolve a fonte existente. Se já houver uma fonte `youtube` com a mesma
  url, devolve essa (não duplica o vídeo).
- Na tela da Entrada: botão discreto **"Trazer do Acervo"** que lista as Referências de vídeo do
  Acervo (`GET /api/acervo/referencias`, que já existe; filtre `tipo === "video"`) com busca por
  título, e marca as que já estão na Entrada.
- A transcrição do Acervo tem blocos de ~30 s: a regra `fragmentar.md` já pede o início "uns
  segundos antes"; não precisa mudar nada, só citar na decisão que a precisão é de ~30 s nessas
  fontes.

### 6.3 Os vídeos do Akita (depois de publicado, quem faz é o Claude, não você)

Só para você saber o porquê da 6.2: depois desta fase publicada, o Claude vai trazer as 6
Referências do Akita, fragmentar, mandar para a trilha 8 e **descartar** os encaixes antigos
(propostas 1629–1634 da trilha 8). **Você não faz isso**: não mexa em dados de produção.

## Testes obrigatórios

- Limpeza: fragmento sem destino há 31 dias some, há 29 fica, enviado nunca some.
- `POST /api/entrada/acervo` com um Acervo falso (`httptest`, como em `material_test.go`): cria a
  fonte pronta com a transcrição; Referência que não é vídeo → `400`; repetida → a mesma fonte; com
  fonte `youtube` da mesma url → a existente.
- `"46:58"` → 2818; `"1:02:03"` → 3723.

## Não fazer

- Não mudar o Acervo. Não mexer em dados de produção.

## Pronto quando

Testes passam; decisão **T40 — Sem destino em 30 dias e fontes trazidas do Acervo**.

Commit: `Entrada (T40): fragmento sem destino some em 30 dias; trazer vídeos do Acervo para a Entrada pela transcrição de lá`

## Dúvidas

## Feito
