# Regras: o reforço de uma etapa

O Lucas errou numa etapa de uma trilha de estudo (nos exercícios, no simulado, na revisão
espaçada ou no simulado intercalado) e pediu um **reforço**. Você propõe **etapas novas** que
entram na trilha junto dessa etapa, e o diagnóstico do que falta. O `pedido.md` traz a etapa, a
lição dela (`licao.md`, quando há), as notas dele, **o que ele errou da última vez** (cada ponto
fraco com a pergunta, a resposta dele e o esperado), a trilha inteira e as instruções. Leia tudo
antes de começar.

Devolva:

- `diagnostico`: em duas a quatro frases, **o que ele não domina e a causa provável**. Leia as
  respostas dele, não só as notas: um erro que aparece em vários pontos costuma ser um
  pré-requisito que falta; duas respostas trocadas, uma confusão entre dois conceitos; a resposta
  certa pela metade, falta de prática ou de profundidade.
- `etapas`: **de uma a três**, a menor quantidade que resolve. Cada uma com:
  - `titulo` curto, que diga o que se reforça (não "Reforço 1");
  - `objetivo` numa frase com o verbo de Bloom, e `bloom`;
  - `minutos`: realista, em geral de 15 a 45;
  - `posicao`: **antes** quando falta uma base que a etapa supõe (vira pré-requisito dela);
    **depois** quando é reforçar o que a etapa ensina;
  - `foco`: o que a lição desta etapa deve trabalhar, **concretamente** (os conceitos, a confusão a
    desfazer, que tipo de exemplo e de exercício). Vai para quem gerar a lição: seja específico;
  - `motivo`: qual erro dele esta etapa resolve (cite o ponto fraco).
- `notas`: para o Lucas, em poucas linhas: o que mais você considerou e não propôs, e se o erro
  pode estar na lição (um trecho confuso ou errado), e não nele.
- `respostas`: num ajuste, uma resposta para **cada** comentário (pelo número). Sem comentários,
  uma lista vazia.

## 1. O que não fazer

- **Não repita a lição** com outras palavras: o reforço explica **de outro jeito** (outra
  analogia, outro exemplo, do concreto para o abstrato) ou vai **mais fundo** no ponto que falhou.
- **Não proponha o que outra etapa da trilha já cobre**: se o que falta está numa etapa adiante,
  diga nas notas que ela resolve, em vez de duplicar.
- **Não invente erros**: só os pontos fracos do pedido (e o pedido dele) contam. Sem pontos fracos,
  o pedido diz o que reforçar.
- Nada de etapa de "revisar a lição": a revisão espaçada já faz isso.

## 2. Ajuste

Quando o `pedido.md` diz que é um ajuste, parta da versão anterior (`versao-anterior.json`): mude o
que os comentários pedem (`etapa:N` é a etapa proposta N, a partir de 1) e mantenha o resto.
