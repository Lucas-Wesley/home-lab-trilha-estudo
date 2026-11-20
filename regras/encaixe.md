# Regras: encaixar um material na trilha

O Lucas monta a trilha aos poucos: primeiro o roteiro (os tópicos), depois o conteúdo, que ele
manda material por material (livros, PDFs, vídeos já transcritos pelo Acervo, artigos). Você lê
**um** material e diz **onde ele entra** na trilha e **que tópicos ele traz que a trilha não tem**.
O `pedido.md` traz a trilha inteira, na ordem, com o número (3.2), a chave, o objetivo de cada etapa
e o material que ela já tem; e diz onde está o texto do material. Leia o pedido e o material
inteiros antes de responder.

## Encaixes (`encaixes`)

Para cada etapa que o material **ensina de verdade**, um encaixe:

- `etapa_chave`: a chave da etapa, exatamente como no pedido.
- `localizador`: **a parte** do material que serve a ela. Livro: `cap. N`, `caps. 2–4` ou `p. 45–60`
  (pelo sumário; um capítulo pode servir a várias etapas, e uma etapa pode ter vários capítulos).
  Vídeo ou texto do Acervo: o trecho, pelo título da seção ou pelos minutos, se houver. Vazio só
  quando o material inteiro é sobre a etapa.
- `papel`: `principal` quando essa parte cobre o centro do objetivo da etapa; `apoio` quando cobre
  parte dele ou de outro ângulo; `aprofundar` quando vai além do que a etapa pede.
- `nota`: o que essa parte traz para a etapa, em uma ou duas frases, para o Lucas (fica junto do material).
- `motivo`: por que encaixa ali (só na proposta).

Seja **preciso, não generoso**: menção de passagem não é encaixe. Um capítulo de introdução que
cita TCP não encaixa na etapa de TCP; o capítulo que ensina TCP, sim.

## Tópicos novos (`novos`)

Quando o material ensina algo **importante para o objetivo da trilha** que **nenhuma etapa cobre**
(nem por outro nome), proponha um tópico novo:

- `titulo` e `objetivo` como os da trilha: o objetivo começa com um verbo de Bloom e lista o que
  aprender, bem específico.
- `modulo`: o título **exato** de um módulo existente onde ele cabe; só invente módulo novo se o
  assunto não cabe em nenhum.
- `depois_de`: a chave da etapa depois da qual ele entra (a ordem importa: o que ele precisa saber
  vem antes); vazio = no fim do módulo.
- `localizador`, `nota` e `motivo` como nos encaixes; `bloom` e `minutos` (20 a 90).

Não proponha tópico que já existe com outro nome, nem detalhe pequeno que cabe dentro de uma etapa
(isso é encaixe, com nota). Respeite o recorte da trilha: o que ele deixa de fora não vira tópico.

## Vídeo em fragmentos

Quando o material é do tipo `youtube`, ele veio da Entrada dividido em fragmentos e o texto traz
**um bloco por fragmento** (`## início–fim · título`, com o resumo e as falas daquele trecho).
Valem, além do que está acima:

- O `localizador` de cada encaixe ou tópico novo é **exatamente um** desses intervalos (ou um
  pedaço dele, `mm:ss–mm:ss`), nunca um tempo inventado.
- Um fragmento pode encaixar em mais de uma etapa da mesma trilha **só se ensina as duas de
  verdade**; não repita o mesmo trecho por proximidade.
- `papel: principal` quando o trecho é a aula da etapa (o Lucas assiste ao fragmento no lugar de
  ler); `apoio` quando é complemento.

## Revisões dos tópicos (`revisoes`)

Quando o material mostra que um tópico **existente** devia mudar (o objetivo deixa de fora algo
central que o material ensina e que é do recorte, o critério não mede o que importa, o título engana,
o tempo está muito fora), proponha a revisão: `etapa_chave`, só os campos que mudam (`titulo`,
`objetivo`, `criterio`, `bloom`, `minutos`; vazio ou 0 = manter) e o `motivo`. Poucas e certeiras:
o material ser mais completo que a etapa não é motivo (isso é encaixe com papel `aprofundar`).

Se o pedido tem um **Foco** (o Lucas mandou o material a um tópico), olhe primeiro para ele: o
encaixe nele com o trecho certo e, se for o caso, a revisão dele.

## Notas (`notas`)

Em poucas linhas: o que o material é, para que parte da trilha ele serve mais, o que dele ficou de
fora e por quê. Se o texto não veio inteiro (só o sumário, vídeo sem transcrição), diga e avise
como melhorar (ex.: mandar o vídeo ao Acervo, que transcreve; indicar os capítulos).
