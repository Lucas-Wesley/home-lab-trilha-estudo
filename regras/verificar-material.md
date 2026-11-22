# Regras: verificar se o material cobre a etapa

O Lucas escolheu material para estudar uma etapa (um vídeo, um trecho de vídeo, um capítulo de
livro, uma página, um documento) e quer saber, **antes de estudar**, se esse material ensina de
verdade o que a etapa pede, e o quanto ensina, para decidir se complementa. O `pedido.md` traz a
trilha, a etapa (objetivo, profundidade, critério de domínio), as outras etapas do módulo e cada
material com o arquivo do texto, a transcrição do vídeo ou o endereço da página. **Leia o pedido e
cada material inteiro antes de julgar.** Você não muda nada: devolve um laudo.

Devolva:

- `topicos`: primeiro, o que a etapa **exige** que se aprenda, tirado do título, do objetivo, do
  critério de domínio e da profundidade (de 4 a 10 pontos concretos, não genéricos: "por que o NAT
  quebra conexões de entrada", não "conceitos de NAT"). Para cada ponto, a `situacao` no material:
  `coberto` (ensina, na profundidade pedida), `raso` (menciona ou explica por cima, abaixo da
  profundidade) ou `ausente`; `onde` ("Material 2, 12:40–15:10" ou "Material 1, seção 3.2"; vazio
  se ausente); e um `comentario` de uma frase.
- `materiais`: um por material do pedido, pelo número: `lido` (false se o texto não veio ou a
  página não abriu), `cobertura` (0 a 100, quanto da etapa ele **sozinho** ensina), `profundidade`
  (`abaixo`, `na_medida` ou `acima` da pedida), `comentario` (duas frases: o que ensina bem, o que
  deixa a desejar) e `fora` (o que ele traz que não é desta etapa e dá para pular; vazio se nada).
- `cobertura`: 0 a 100, quanto da etapa os materiais **juntos** ensinam na profundidade pedida.
- `veredito`: `suficiente` (dá para estudar só com isso), `complementar` (o centro está lá, mas
  falta algo que importa) ou `insuficiente` (não ensina o centro da etapa, ou ensina errado).
- `faltas`: o que falta ou está raso e importa, cada ponto com `onde_achar`: a **fonte primária ou
  de referência** que cobre (documentação oficial, RFC, especificação, paper, livro e capítulo). Só
  cite fonte que você sabe que existe; se não souber uma precisa, diga o tipo de fonte ("a
  documentação oficial do PostgreSQL sobre MVCC").
- `resumo`: duas ou três frases falando com ele: dá para estudar só com isso? O que complementar
  primeiro?

## Como julgar

1. **A medida é a etapa, não o assunto.** Conte só o que o objetivo, o critério e a profundidade
   pedem. O que é de outra etapa do módulo (a lista está no pedido) não é falta desta.
2. **Profundidade conta.** Na profundidade 4 ou 5, citar um conceito sem trade-offs, casos-limite
   e alternativas é `raso`. Na 1 ou 2, um panorama correto pode ser `coberto`.
3. **Ensinar não é mencionar.** Um vídeo que diz o nome de um conceito sem explicar o porquê não
   cobre o conceito. Leia o trecho em que ele aparece antes de marcar.
4. **Erro conta contra.** Se o material ensina algo errado ou desatualizado, o ponto não está
   coberto: diga no comentário o que está errado e conte em `faltas`.
5. **A transcrição é automática.** Termos técnicos podem estar trocados ("cubernetes"); leia pelo
   sentido e não culpe o material por erro de reconhecimento de fala. O minuto de `onde` sai do
   colchete da transcrição; não invente minuto.
6. **Material que não veio** (`lido: false`) não cobre nada: não suponha o conteúdo dele pelo
   título. Se todos os outros não bastam, o veredito sai do que foi lido, e o resumo avisa.
7. **Seja exato e econômico.** A nota de cobertura é a sua estimativa honesta, não um elogio: 100
   só quando não falta nada que a etapa pede. Prefira poucas faltas que importam a uma lista longa
   de detalhes.
