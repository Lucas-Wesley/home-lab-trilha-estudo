# Pedagogia: como a trilha ensina

> Os princípios que viram funcionalidade. Cada um diz **de onde vem** e **como entra no app**.
> Decisões em [DECISOES.md](DECISOES.md).

---

## 1. Princípios

| Princípio | De onde vem | Como entra no app |
|---|---|---|
| **Mapa de pré-requisitos** | Khan Academy, Math Academy (grafo de conhecimento) | Cada etapa tem pré-requisitos. A tela mostra uma lista ordenada; por baixo é um grafo, o que permite pular e inserir reforço sem quebrar a ordem |
| **Planejamento reverso** (*backward design*) | Wiggins & McTighe | O plano começa pelo "ao final você será capaz de…" e só depois define as etapas |
| **Objetivos com Bloom** | Taxonomia de Bloom revisada | Cada etapa tem um verbo-alvo (lembrar, entender, aplicar, analisar, avaliar, criar), que define a profundidade e o tipo de checagem |
| **Nível real ≠ nível declarado** | testes de nivelamento | Mini-teste opcional, à escolha do Lucas (T9) |
| **Domínio antes de avançar** (*mastery learning*) | Bloom | Critério por etapa; errou muito, a IA **propõe** etapas de reforço ("Pedir reforço", com o diagnóstico; fase 7). Pular é sempre permitido |
| **Prática de recuperação** | ciência cognitiva | "Teste seu entendimento" antes da resposta; exercícios e simulado |
| **Repetição espaçada** | FSRS / Anki | Flashcards de cada etapa vão ao app Flashcards; etapas concluídas voltam com perguntas rápidas em 1, 3, 7, 14, 30, 60 e 120 dias (acertou, sobe o degrau; errou, recomeça; fase 7) |
| **Intercalar** | ciência cognitiva | O simulado intercalado do painel mistura perguntas das etapas já estudadas, as fracas primeiro (fase 7) |
| **Aprender em camadas** | `metodologia.md` do estudo do homelab | O molde de toda lição (§4) |
| **Explicar com as próprias palavras** | técnica de Feynman | Futuro: o Lucas explica, a IA aponta o que faltou |

---

## 2. A entrevista: perguntas certas para cada tipo de assunto

Antes de planejar, a IA precisa fechar os **eixos** do perfil de domínio. Ela pula o que o pedido
inicial já respondeu e oferece "não sei, sugira" em cada pergunta. Os perfis são **regras
editáveis** (`regras/perfis/*.md`), não código.

| Perfil | Eixos que a IA precisa fechar |
|---|---|
| **História** | período/século; recorte (evento, figura, tema, linha do tempo geral); escala (nacional, regional, mundial); ângulo (político, social, econômico, cultural); profundidade |
| **Tecnologia** | objetivo (trabalho, certificação, projeto, curiosidade); nível atual; o que já usa; teoria × fundamentos internos; ambiente disponível |
| **Língua** | área (gramática, ortografia, redação, literatura, sintaxe); objetivo (escrever melhor, entender a norma, prova); nível; foco normativo × descritivo |
| **Ciências** | área e sub-área; base matemática disponível; conceitual × quantitativo; profundidade |
| **Genérico** | a IA deduz os eixos pela pré-pesquisa e diz quais escolheu |

Exemplo (História do Brasil, sem mais nada no pedido): "Que período: Colônia, Império, República
Velha, Era Vargas, Ditadura, Nova República, ou a linha do tempo inteira?" → "Um evento, uma
figura, um tema que atravessa períodos (escravidão, industrialização) ou o contexto geral?" →
"Que ângulo pesa mais?".

---

## 3. Profundidade: do panorama ao deep dive

Regulada **na trilha**, com **sobrescrita por etapa**.

| Nível | O que a lição cobre |
|---|---|
| **1. Panorama** | o que é, para que serve, onde se encaixa; 1–2 páginas |
| **2. Essencial** | conceitos principais + exemplos; o necessário para usar e conversar sobre o assunto |
| **3. Sólido** | o padrão da metodologia de camadas: fundamentos, principal, ao redor, aplicação, teste |
| **4. Aprofundado** | + casos-limite, trade-offs, erros comuns, alternativas comparadas, fontes primárias |
| **5. Deep dive** | + **origens** (por que surgiu, que problema resolvia, como evoluiu), a base completa embaixo, especificações/RFCs, debates em aberto |

**Conceitos anteriores** (o formato da aba Estudar do Acervo: `base` ou `vizinho`, por que ajuda,
explicação curta, fontes conferidas, link para Tema do Acervo):

1. **Dentro da lição:** sempre, como seção "Conceitos anteriores".
2. **Na estrutura (níveis 4–5):** um conceito `base` grande demais para um parágrafo vira **etapa de
   base** inserida antes — "da base até a superfície". O `vizinho` fica como leitura complementar,
   sem bloquear.

---

## 4. O molde da lição (camadas)

Herdado da `metodologia.md` do estudo do homelab. A profundidade decide quais seções entram e o
tamanho de cada uma.

| Seção | Conteúdo | Níveis |
|---|---|---|
| Cabeçalho | objetivo (Bloom), pré-requisitos, tempo estimado, materiais | todos |
| Conceitos anteriores | base e vizinho | todos (curto em 1–2) |
| Fundamentos | o que precisa saber antes, do zero | 2–5 |
| Conceitos principais | o centro da etapa, de forma geral | todos |
| Conceitos ao redor | vizinhos, confusões comuns, alternativas | 3–5 |
| Origens e evolução | por que existe, como chegou aqui | 5 (e 4, curto) |
| Como tudo se conecta | diagrama ou mapa | 3–5 |
| Aplicação | no contexto do Lucas (instruções da trilha) | todos |
| Teste seu entendimento | perguntas com resposta escondida | todos |
| Flashcards sugeridos | pares pergunta/resposta | todos |
| Referências | com rótulo `[oficial]` `[padrão]` `[livro]` `[curso]` `[artigo]` `[vídeo]` | todos |
| Para aprofundar | caminhos opcionais | 3–5 |

Cada conceito leva a prioridade 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade.
Regra de parada: descer nos fundamentos até algo que o Lucas já domina; dali, um link basta.

---

## 5. Meta, ritmo e tempo

- Tempo estimado por etapa = leitura da lição (palavras ÷ ritmo de leitura) + exercícios + simulado.
- O cronograma distribui as etapas pelos dias de estudo da meta.
- O app mede o tempo real (como as estatísticas da Biblioteca, R31 de lá) e corrige as estimativas.
- Atrasou: o app **pergunta** se recalcula (e aceita um pedido junto, como toda ação, T13).
