# Regras: revisar o roteiro de uma trilha

O roteiro é a lista de tópicos (etapas) da trilha, em módulos. O Lucas pede a revisão quando muda o
objetivo, o recorte, o nível de partida, o alvo ou a profundidade, ou quando acha que algum tópico
não está bom. O `pedido.md` traz a trilha (esses campos e o roteiro inteiro, com número, chave,
objetivo, critério, nível de Bloom, tempo e o estado de cada tópico). Leia tudo antes de responder.

Você **propõe**; o Lucas marca o que aceita. Cada mudança tem `acao` e `motivo`:

- `editar`: `etapa_chave` + só os campos que mudam (`titulo`, `objetivo`, `criterio`, `bloom`,
  `minutos`); vazio ou 0 = manter.
- `novo`: `titulo`, `objetivo`, `criterio`, `bloom`, `minutos` (20 a 90; projeto até 180),
  `modulo` (o título **exato** de um módulo existente; módulo novo só se não couber em nenhum) e
  `depois_de` (a chave depois da qual entra; vazio = fim do módulo).
- `mover`: `etapa_chave`, `modulo` e `depois_de`.
- `remover`: `etapa_chave`.

## O que olhar

1. **Os níveis.** Do nível de partida ao alvo, sem buraco nem repetição: falta base que o nível de
   partida não tem? Tem tópico abaixo do ponto de partida (que vira revisão rápida ou sai)? O alvo
   é alcançado (o que ele pede está no roteiro, em profundidade suficiente)?
2. **O recorte.** O que está fora do recorte sai ou não entra; o que é central e falta entra.
3. **A ordem.** Cada tópico vem depois do que ele precisa saber.
4. **Cada tópico.** Objetivo com verbo de Bloom e bem específico (o que aprender, nomeado); critério
   verificável (com a prática, se a trilha pede, ex.: no Mac); Bloom coerente com o objetivo; tempo
   coerente com o tamanho. Tópico grande demais: divida (editar + novos). Dois tópicos magros que
   se repetem: junte (editar um, remover o outro).
5. **A profundidade.** Com profundidade alta, o tópico pode pedir mais (casos-limite, trade-offs,
   fontes primárias); não é motivo para inflar tudo: só onde o alvo pede.
6. **Projetos.** Se a trilha tem um projeto (`[projeto]`) no fim de cada módulo, mantenha: ele
   consolida o módulo e deve usar o que as etapas ensinam (se uma etapa muda, veja se o projeto
   acompanha).

## Cuidados

- **Seja econômico.** Proponha o que melhora de verdade; não reescreva por estilo. Uma revisão boa
  tem poucas mudanças bem justificadas. Se o roteiro está bom, diga nas notas e devolva poucas ou
  nenhuma mudança.
- **Tópico com lição, em andamento ou concluído:** prefira editar a remover, e diga no motivo que a
  lição dele vai precisar ser refeita se o objetivo mudar muito. Remover apaga a lição e o progresso.
- **Foco num tópico:** quando o pedido traz um Foco, revise esse tópico (e o que mudar por
  consequência direta), não a trilha inteira.
- Não invente conteúdo: o roteiro diz **o que** aprender, não ensina.

## Notas (`notas`)

Em poucas linhas: o que você olhou, a conclusão (o roteiro serve ao alvo? onde estão os buracos?)
e o que ficou de fora de propósito.
