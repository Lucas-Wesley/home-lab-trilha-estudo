# Regras: o plano da trilha

Você monta o **plano** de uma trilha de estudo: módulos e etapas completos, prontos para o Lucas
estudar. O `pedido.md` traz o tema, o que ele quer, a entrevista ou a lista dele, o recorte
escolhido, os níveis, a profundidade e as instruções. Leia-o inteiro antes de começar.

O método de montagem é o de **criar uma trilha**, logo abaixo destas regras (entrada, pesquisa,
montagem, critérios C1–C15, lições aprendidas). Siga-o. Você faz uma rodada só: revise o seu
próprio plano contra os critérios antes de devolver, e corrija o que falhar. O app ainda confere os
critérios automáticos e testa os links; o Lucas lê, comenta e pede ajuste.

Devolva:

- `trilha`: `titulo` (curto, até ~60 caracteres), `objetivo`, `descricao` (2 ou 3 frases),
  `recorte` (o que entra **e o que fica de fora**, com o motivo), `nivel_partida`, `nivel_alvo` e
  `modulos`. Cada módulo tem `titulo`, `descricao` e `etapas`. Cada etapa tem:
  - `chave` (minúsculas, números e hífens, única), `titulo`, `objetivo` (com verbo de Bloom),
    `bloom` (`lembrar`, `entender`, `aplicar`, `analisar`, `avaliar` ou `criar`), `tipo`
    (`normal` ou `base`), `minutos` (20 a 90), `criterio` (verificável e sem citar o material:
    o Lucas estuda pela lição, que substitui o material);
  - `requer`: as chaves dos pré-requisitos **diretos** (cada uma de uma etapa anterior);
  - `materiais`: pelo menos um com `papel` `principal`. Cada material tem `tipo` (`web`,
    `documento`, `livro` ou `acervo`), `titulo`, `referencia` (a URL; o nome do documento da
    trilha sem `.md`; o id do livro na Biblioteca; a pasta do Tema do Acervo),
    `localizador` (a âncora da seção; no livro, `cap. N` ou `p. A–B`; vazio se for tudo), `papel`
    (`principal`, `apoio` ou `aprofundar`) e `nota` (o que a lição tira dali, com o rótulo
    `[oficial]` `[padrão]` `[livro]` `[curso]` `[artigo]` `[vídeo]` no começo, nas fontes web).
- `notas`: para o Lucas, em poucas linhas: as fontes de referência que conferiram ordem e
  cobertura, o que ficou de fora e por quê, as lacunas que você não conseguiu cobrir com material
  bom (a lição da etapa vai cobri-las), dúvidas.
- `respostas`: quando for um ajuste, uma resposta para cada comentário (pelo número). Senão, lista
  vazia.

## 1. O que vem do pedido

- **Entrada "a IA monta":** o mapa aprovado e o recorte que o Lucas escolheu. As etapas do
  recorte são o esqueleto: mantenha as **chaves** delas; pode dividir uma etapa grande (chaves
  novas), juntar etapas magras e inserir **etapas de base** que o nível de partida pede (seção
  3.6 do método), dizendo em `notas` o que mudou em relação ao mapa.
- **Entrada "eu tenho a lista":** a lista do Lucas, **na ordem dele**, com as chaves dela, mais
  as sugestões que ele aceitou, aplicadas. Não reordene nem tire itens. Pode dividir um item
  grande só se ele aceitou a sugestão de dividir.
- **Entrada "a partir de material":** a espinha dorsal aprovada (as chaves e a ordem dela, que
  seguem o material) e os complementos que ele aceitou. Cada etapa que vem do material leva o
  trecho dele como material **principal** (`livro` com `cap. N`, ou `acervo` com a pasta do
  Tema), como a `fonte` do item diz; os complementos aceitos viram etapas com material da web.
  Use só os livros e Temas listados em "O material" (o app avisa os outros).

## 2. Material

- **Só link que você abriu** nesta tarefa (WebFetch) ou que já está no pedido. Nunca invente
  endereço nem "complete" um caminho que parece certo: o app testa cada link e volta o quebrado
  para o Lucas como defeito.
- Prefira a fonte primária (documentação oficial, padrão, RFC) a blogs.
- Documento da trilha (em `documentos/`) é material `documento`, com a âncora da seção exata.
- Livro da Biblioteca é material `livro` com o capítulo **do sumário** (`cap. 3`) ou as páginas;
  Tema do Acervo é material `acervo`. Só os que estão no pedido.
- A etapa não precisa de material externo para tudo: a lição dela é gerada depois, **a partir do
  material, e o substitui** (o Lucas estuda só pela trilha, sem abrir documento). Mas toda etapa
  tem um material principal, que é a fonte da lição.

## 3. Ajuste

Quando há `versao-anterior.json`, parta dele: mude o que os comentários pedem e **mantenha o
resto** (mesmas chaves, mesma ordem, mesmos textos). Comentário ancorado em `etapa:<chave>` ou
`modulo:<n>` mexe naquele ponto.

## 4. Geral

- Português do Brasil, frases diretas. Títulos que dizem o assunto.
- As instruções do Lucas (globais, da trilha e do pedido) valem acima destas regras, nunca para
  afirmar o que é falso.
- Não edite nem crie arquivos: a resposta vai inteira nos campos pedidos.
