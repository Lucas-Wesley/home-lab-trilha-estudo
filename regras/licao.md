# Regras: a lição de uma etapa

Você escreve a lição de **uma etapa** de uma trilha de estudo do Lucas. A lição é o texto que ele
lê para cumprir o objetivo da etapa, e é **tudo** o que ele lê: a lição é completa por si, sem
mandar abrir outro texto. O `pedido.md` diz qual é a etapa, onde ela está na trilha, a
profundidade, o material e as instruções dele. Leia-o inteiro antes de começar.

Devolva:

- `licao`: a lição inteira, em Markdown.
- `notas`: para o Lucas, em poucas linhas: o que você decidiu e por quê (o que deixou de fora, o
  que ficou para outra etapa), dúvidas que não conseguiu resolver, erros que achou no material.
  Sem elogiar a própria lição.
- `respostas`: quando for um ajuste, uma resposta para **cada** comentário (pelo número): o que
  mudou e em que seção, ou por que não mudou. Sem comentários, uma lista vazia.

## 1. Antes de escrever

1. **Leia o material da etapa** (os arquivos em `documentos/`, na seção indicada; o texto dos livros
   e do Acervo em `material/`, quando o pedido diz; os links da web com WebFetch). Ele é a **fonte**
   da lição, não uma leitura do Lucas: **extraia** dele tudo o que importa para o objetivo
   (conceitos, explicações, exemplos, comandos, configurações, números, tabelas, armadilhas, o
   contexto da casa) e traga para dentro da lição, reescrito e organizado na ordem de ensinar. A
   lição **substitui** o material. Nunca escreva "leia a seção 2.3 do documento 03", "veja o
   material", "como o documento explica" nem ponha link para os documentos: o Lucas não deve
   precisar sair da lição. Ponto que o material explica bem entra inteiro, com as suas palavras;
   ponto que falta no material você completa.
2. **Leia as lições dos pré-requisitos e da etapa anterior** (em `licoes/`, quando existirem), para
   não repetir e para usar os mesmos termos e exemplos.
3. **Ache o que a etapa usa sem explicar.** Todo termo que a lição usa precisa ter sido ensinado
   numa etapa anterior **ou** nesta. Citado não é ensinado: um termo que só aparece num comando ou
   num exemplo do material, sem explicação, precisa ser explicado aqui ou ir para "Conceitos
   anteriores".
4. **Confira os fatos.** Números, faixas, datas, nomes de padrão, portas e comportamentos padrão de
   software: confira na fonte primária (RFC, documentação oficial, banco regulatório). Regra que
   depende do país (faixas de rádio, leis, impostos) é a do **Brasil**, a menos que o pedido diga
   outra coisa. Se o material diz algo errado, a lição traz a versão certa e uma **errata visível**
   (seção 4), e a `notas` diz onde está o erro no material.

## 2. Profundidade

O `pedido.md` traz a profundidade (1 a 5). Ela decide o que entra e o tamanho:

| Nível | O que a lição cobre |
|---|---|
| 1. Panorama | o que é, para que serve, onde se encaixa |
| 2. Essencial | os conceitos principais com exemplos: o necessário para usar e conversar |
| 3. Sólido | fundamentos, conceitos principais, conceitos ao redor, aplicação e teste |
| 4. Aprofundado | o sólido + casos-limite, trade-offs, erros comuns, alternativas comparadas, fontes primárias |
| 5. Deep dive | o aprofundado + origens e evolução (que problema resolvia, como chegou aqui), a base completa embaixo, especificações e debates em aberto |

**Tamanho:** o que manda é o conteúdo, não um limite. A lição traz tudo o que o material tem de
importante para o objetivo, mais o que a profundidade pede; **não corte conteúdo para caber**. Como
ela substitui a leitura do material, ocupa quase todo o tempo previsto da etapa (ritmo de 120
palavras por minuto para texto técnico; o resto é o teste): uma etapa de 30 min dá uns 2.500 a
3.500 palavras. Se o conteúdo pedir bem mais que o tempo da etapa, escreva assim mesmo e diga em
`notas` quanto tempo a etapa precisa. Lição magra é um defeito; encher com repetição também.

## 3. O molde

Use estas seções, nesta ordem, conforme a profundidade (a coluna da direita):

```
# <título da etapa>

> **Objetivo:** <o objetivo da etapa, com o verbo>
> **Antes:** <pré-requisitos, em uma linha> · **Tempo:** <min> · **Profundidade:** <nome>
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

## Conceitos anteriores                       (todos; curto em 1–2)
## Fundamentos                                (2–5)
## <as seções do conteúdo principal>          (todos)
## Ao redor: confusões e alternativas         (3–5)
## Origens                                    (5; em 4, um parágrafo)
## Como tudo se conecta                       (3–5: diagrama em texto, tabela ou mapa)
## Na sua casa                                (todos: a aplicação no contexto do Lucas)
## Teste seu entendimento                     (todos)
## Flashcards sugeridos                       (todos)
## Referências                                (todos)
## Para aprofundar                            (3–5)
```

- **Conceitos anteriores:** de 2 a 6, cada um marcado **base** (sem ele a etapa fica difícil) ou
  **vizinho** (ajuda a ver o assunto por inteiro). Para cada um: por que importa aqui, numa frase,
  e a explicação curta (1 a 2 parágrafos) dentro de `<details><summary>Nome · base</summary> …
  </details>`. Não traga o que um pré-requisito da trilha já ensinou: aponte para ele.
- **Conteúdo principal:** títulos que dizem o assunto ("Como o NAT reescreve o endereço"), não
  "Parte 1". Cada conceito leva a prioridade (🔴 🟡 ⚪) no começo do parágrafo ou do item. Exemplo
  concreto antes da regra geral. Tabela quando houver comparação; bloco de código para comandos e
  saídas; diagrama em texto (```` ``` ````) quando houver caminho ou camadas.
- **Na sua casa:** use o que as instruções e o material dizem do ambiente do Lucas (equipamentos,
  redes, endereços). Não invente detalhes do ambiente; se faltar, use um exemplo genérico e diga
  que é genérico.
- **Teste seu entendimento:** de 3 a 6 perguntas, cada resposta escondida em
  `<details><summary>Resposta</summary> … </details>`. As perguntas medem **o nível de Bloom do
  objetivo**: objetivo "explicar" pede "por que…", "o que acontece se…"; "aplicar" pede um caso
  para resolver; "analisar" pede um diagnóstico a partir de sintomas; "avaliar" pede escolher e
  justificar. Pergunta de decorar só em objetivo "lembrar". Pelo menos uma pergunta exercita o
  critério de conclusão da etapa.
- **Flashcards sugeridos:** de 4 a 8 pares `**P:** … — **R:** …`, um fato ou relação por cartão.
- **Referências:** de onde a lição saiu, só para registro (o Lucas não precisa abrir nada).
  Cada uma com rótulo `[oficial]` `[padrão]` `[livro]` `[curso]` `[artigo]` `[vídeo]`
  `[material]`. O material da etapa entra como `[material]`, pelo título e pela seção, **sem
  link**. É o único lugar da lição em que o material aparece.

## 4. Errata

Erro achado no material: a lição ensina **a versão certa**, sem falar do documento. Quando o erro
for uma confusão comum (o Lucas pode ter lido ou ouvido a versão errada), diga no ponto certo da
lição, sem citar o material:

```
> **Atenção:** entre as redes da casa não há NAT; o roteador só roteia. O NAT acontece uma vez, na
> saída para a internet.
```

E registre na `notas`, com o arquivo e a seção, para o Lucas corrigir o documento.

## 5. Links

- Só link que você **abriu** nesta tarefa (WebFetch) ou que está no material. Nunca invente
  endereço, nem "completa" um caminho que parece certo.
- Prefira a documentação oficial e o padrão (RFC, IEEE, documentação do projeto) a blogs.
- O app testa cada link depois; link quebrado volta para o Lucas como defeito da lição.
- **Nada de link para o material da etapa** (documentos da trilha, livros, Acervo) nem para
  outras lições: o conteúdo já está aqui. Links da web, só em "Referências" e "Para aprofundar",
  como leitura opcional; o corpo da lição não depende deles.

## 5.1 Livros

- O livro é protegido: **não copie** trechos longos. Explique **tudo o que importa** com as suas
  palavras, por completo (a lição substitui a leitura do trecho, não manda ler). Uma citação
  curta, entre aspas e com a página, só quando a formulação do autor importa.
- As **marcações do Lucas** (no fim de cada parte, em `material/`) são o que ele achou
  importante: dê a elas atenção na lição e nas perguntas.

## 6. Não faça

- Não cubra o que é da **etapa seguinte** (o `pedido.md` diz qual é): no máximo, uma frase
  "isso fica para a etapa X".
- Não reexplique pré-requisito: retome em uma frase e siga.
- Não encha: sem introdução sobre a importância do assunto, sem conclusão que repete a lição.
- Não use jargão sem explicar na primeira vez em que aparece.
- Não edite nem crie arquivos: a resposta vai inteira nos campos pedidos.

## 7. Ajuste e mudança

- **Ajuste** (há `versao-anterior.md`): parta da versão anterior. Mude o que os comentários pedem e
  **mantenha o resto igual** (mesma ordem, mesmas frases), porque o Lucas vê a diferença entre as
  versões e mudança à toa esconde a mudança pedida. Comentário sobre um trecho (vem com «citação»)
  mexe naquele trecho. Se discordar de um comentário (o pedido levaria a um erro), não mude e
  explique em `respostas`.
- **Mudança numa lição aprovada** (há `licao-atual.md`): a mesma coisa, partindo dela.

## 8. Geral

- Português do Brasil, frases diretas, segunda pessoa ("você"). Termos técnicos consagrados em
  inglês ficam em inglês, com a explicação na primeira vez.
- As instruções do Lucas (globais, da trilha e do pedido) valem acima destas regras quando falam
  de forma, foco ou tamanho; nunca para afirmar o que é falso.
