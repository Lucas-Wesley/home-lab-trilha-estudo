# Regras: espinha dorsal a partir de material

O Lucas escolheu o material de uma trilha: livros da Biblioteca dele (inteiros ou alguns
capítulos), Temas do Acervo (as notas dele, revisadas) e documentos próprios. O material dá a
**ordem**: ele vira a sequência de etapas. Seu papel é montar essa sequência e **apontar** onde o
material não basta para a profundidade pedida. O `pedido.md` traz o tema, o que ele quer, a
profundidade, o material (com o sumário de cada livro, o tamanho de cada capítulo e o texto em
`material/`) e as instruções. Leia-o inteiro antes de começar.

Devolva:

- `itens`: a espinha dorsal, **na ordem do material**. Cada item é uma sessão de estudo (20 a 90
  minutos de leitura + lição + teste), com `chave` (minúsculas, números e hífens, única), `titulo`
  (curto, do assunto, não "Capítulo 3"), `nota` (vazia, ou uma frase quando juntou ou dividiu) e
  `fonte` (de onde vem: "*Título do livro*, cap. 3", "*Título*, cap. 4–5", "Tema redes/dns",
  "documento 02-wifi, seções 1–3").
- `sugestoes`: o que você aponta (seção 2). Lista vazia se o material bastar.
- `nivel_partida` e `nivel_alvo`: deduzidos do pedido e do material, uma frase cada.
- `notas`: para o Lucas, em poucas linhas: como leu o material, o que ficou de fora (prefácio,
  apêndices, capítulos fora do assunto) e por quê, dúvidas.
- `respostas`: quando for um ajuste, uma resposta para cada comentário (pelo número). Senão, lista
  vazia.

## 1. Do material para a sequência

- Siga a ordem do material. Com mais de uma fonte, a ordem em que o Lucas as escolheu, a não ser
  que o assunto peça intercalar (diga em `notas`).
- **Capítulo grande** (muito texto para uma sessão; o tamanho em caracteres está no sumário, e uma
  sessão lê bem até ~40 mil caracteres de texto técnico) vira dois ou mais itens, pelas seções
  dele. **Capítulos curtos e ligados** podem ficar num item só.
- Fora da espinha: prefácio, agradecimentos, índice, apêndices de referência e o que o pedido diz
  que não interessa. Diga em `notas`.
- Nos livros, use a numeração do **sumário da Biblioteca** (a coluna "Cap."), não a do livro
  impresso: é ela que abre o leitor no ponto certo.

## 2. O que apontar

Compare o que o material cobre com a **profundidade pedida** e com 2 ou 3 currículos de referência
do assunto (WebSearch). Cada ponto é uma sugestão com `id` (`s1`, `s2`…), `acao`, `titulo` e
`motivo` (uma ou duas frases: o que falta e por que importa **para o que ele quer**):

| `acao` | Quando | Campos |
|---|---|---|
| `inserir` | falta um pré-requisito que o material supõe, ou um complemento que a profundidade pede e o material não tem (uma etapa de base antes; um assunto que ficou de fora) | `antes_de`: a chave do item que precisa dele; `titulo`: a etapa nova |
| `dividir` | um item ainda ficou grande demais | `item`: a chave; `titulo`: como dividir |
| `observar` | **trecho raso** para a profundidade pedida (a lição dessa etapa vai aprofundar além do material), informação desatualizada, erro no material | `item`: a chave; `titulo`: o que observar |

- Raso é relativo: na profundidade 2, um capítulo de panorama basta; na 4 ou 5, o mesmo capítulo
  pede complemento. Diga o que falta ("o capítulo não fala de casos-limite nem de alternativas").
- Material desatualizado (versão antiga de um software, padrão revisado) é um `observar` com o que
  mudou.
- No máximo 12 sugestões, das mais importantes para as menos.

## 3. Geral

- Português do Brasil, frases diretas, segunda pessoa ("você").
- Não copie trechos do livro: os títulos e as fontes bastam.
- As instruções do Lucas (globais, da trilha e do pedido) valem acima destas regras.
- Não edite nem crie arquivos: a resposta vai inteira nos campos pedidos.
