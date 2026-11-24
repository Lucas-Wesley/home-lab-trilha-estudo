# Plano: sugestões de artigo para o blog (aba Projeto)

Plano de 23/11/2026. Implementado em 24/11/2026 (T62); só a fase 4 fica para depois.

## O que é
No item do Projeto (0.1, 0.2…), um quadro **Blog** que orienta o Lucas sobre o que escrever a partir
do item. **Quem escreve é o Lucas, tudo.** A IA só sugere temas e tópicos e não redige o artigo, nem
um trecho dele.

Os artigos:
- **Independentes do projeto.** Nada de "no meu projeto estou fazendo X". O item é a matéria-prima,
  e o artigo fica de pé sozinho.
- **Não são tutorial.** São concisos, concretos e sólidos, com começo, meio e fim. Juntam teoria e
  prática e, de preferência, vários assuntos do item num texto com substância. Exemplo do 0.1:
  "Bases sólidas: Go, Git e GitHub Actions" em vez de "como usar o pipe no terminal".
- **Atemporais quando der**, e de aprofundamento. O assunto pode ser básico (o 0.1 é), mas tratado
  a fundo.

## As duas ações
**A. Sugerir artigos.** A IA lê o item inteiro: o que implementar, o pronto, os tópicos, o estudo
(trilhas e etapas, e se já foram estudadas), o "Consultar" e o caderno (decisões, erros, o que
aprendi). Devolve de 1 a 5 sugestões.

**B. Tópicos para um título.** O Lucas escreve o título que quer (e, se quiser, uma linha do que quer
dizer). A IA analisa o título e o que o item já tem e devolve os tópicos que o artigo não pode deixar
de ter. Exemplo: "GitHub Actions e o movimento ágil" → o que todo artigo com esse título precisa
cobrir. Cada sugestão da A tem um "Pedir os tópicos deste", que abre a B já preenchida.

## As engrenagens (valem para as duas)
| Engrenagem | Opções (padrão em negrito) |
|---|---|
| Horizonte | **atemporal** · atual (versões e ferramentas de agora) · núcleo atemporal + uma seção datada |
| Nível do leitor | iniciante · intermediário · **aprofundamento** |
| Formato | **a IA escolhe** · ensaio de fundamentos (o porquê de uma base) · como funciona por dentro (a anatomia de um mecanismo: o `go.mod`, o exit code) · decisão e trade-offs (monorepo x repositórios, com o critério) · mapa de um assunto (como as peças se encaixam) · erros e armadilhas (o que quebra e por quê) · princípio aplicado (uma ideia, vários casos) · opinião fundamentada (uma tese defendida) · a história de uma ideia (por que surgiu, o que resolveu) |
| Abrangência | **reunir os assuntos do item** · um assunto por artigo · as duas coisas |
| Tamanho | curto (~1.000 palavras) · **médio (~2.000)** · longo (3.000+) |
| Código | **pouco, só o que prova o ponto** · nenhum · com exemplo trabalhado |
| Quantas sugestões | 1 a 5 (**3**), só na A |
| Pedido livre | texto: o que a IA deve levar em conta |

As engrenagens ficam recolhidas, com o resumo numa linha ("atemporal · aprofundamento · a IA escolhe
o formato · médio"); a última escolha fica guardada no navegador.

## O que volta
**Cada sugestão (A):**
- o título e mais duas alternativas;
- a **tese** em uma frase: o que o artigo afirma;
- para quem é e o que o leitor leva;
- o formato e por que é atemporal (ou o que nele envelhece);
- o **arco**: começo, meio e fim, uma frase cada;
- os tópicos, obrigatórios e opcionais, cada um com o porquê;
- o **exemplo central**, o caso concreto que sustenta o texto;
- o que deixar de fora (para não virar apostila);
- as fontes primárias para conferir;
- a base no item: quais trilhas e etapas alimentam o artigo, com "já estudado" ou "a estudar".

**Os tópicos de um título (B):**
- a tese que o título sugere (e um título melhor, se o dado for vago);
- os tópicos obrigatórios, cada um com o porquê e a situação: já estudado (trilha › etapa) ou
  não estudado (onde estudar);
- os opcionais, o que fica de fora e o arco;
- as armadilhas: o que os artigos sobre o tema costumam errar ou repetir;
- as perguntas que o leitor vai ter.

Nas sugestões, o Lucas marca **guardar** (a ideia fica no topo do quadro), **descartar** ou
**escrevi** (com o link do post).

## Como funciona por dentro
- `backend/blog.go`: `POST /api/projeto/itens/{id}/blog` (`{modo: sugerir|topicos, titulo, nota,
  opcoes, pedido}`) cria o pedido e enfileira a tarefa `blog` (fila do Claude, uma por vez, sem
  proposta: nada é aplicado em lugar nenhum). `GET .../blog` lista as do item. `PUT
  /api/blog/{id}/sugestoes/{n}` marca guardada, descartada ou escrita.
- **Tabela `blog_pedidos`:** item, modo, título, nota, opções, pedido, estado (`gerando`, `pronta`,
  `falhou`), tarefa, resultado (JSON) e as marcações. Fica no banco da Trilha, fora do GitHub, como
  todo conteúdo.
- **O `pedido.md`** sai dos dados do projeto que já estão em memória (o item do ROADMAP + o caderno)
  e das etapas ligadas pelo `casar`: título, objetivo, se foi concluída e o texto da lição aprovada,
  quando há. Leva também as sugestões guardadas e escritas de todos os itens, para não repetir tema e
  para sugerir continuação.
- **Regra `regras/blog.md`:** os princípios acima, o esquema JSON de cada modo, a proibição de
  redigir parágrafos e o veto ao tutorial e ao "no meu projeto". Usa o `modelo_licao` e só a
  ferramenta `Read`.
- O `scripts/claude-falso.mjs` responde a "# Pedido: artigos do blog" e testes em `blog_test.go`
  (os dois modos, a falha, a limpeza do JSON da IA) e na `blog.js` do front.
- **Tela:** `BlogItem.vue` no item, logo abaixo de "Caderno do item". Em cima, as ideias guardadas;
  depois os botões "Sugerir artigos" e o campo "Tenho um título" → "Pedir tópicos"; depois as
  engrenagens e o histórico das últimas 5.

## Fases
1. Backend, regra, falso e testes.
2. O quadro na aba Projeto.
3. Deploy e o primeiro uso real no 0.1.
4. Depois: ler os artigos publicados do repositório do blog (títulos e teses do frontmatter) para
   não repetir assunto e sugerir continuações. O blog está zerado desde 23/11/2026, então isso só
   faz sentido quando houver artigos.

**Fora do escopo** (só se o Lucas pedir): a IA conferir o rascunho dele, e escrever qualquer texto
do artigo.
