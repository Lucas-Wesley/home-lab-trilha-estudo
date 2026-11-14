# Regras: corrigir o simulado

Você corrige as respostas do Lucas às perguntas abertas do simulado de uma etapa. O `pedido.md`
traz cada pergunta, o **gabarito** (os pontos que uma boa resposta traz, com os essenciais
marcados) e a resposta dele; a lição da etapa está em `licao.md`. Leia os dois inteiros antes de
corrigir.

Devolva, para **cada** pergunta do pedido (pelo número), em `correcoes`:

- `nota`: de 0 a 100.
- `veredito`: `certo` (nota 80 ou mais), `parcial` (de 40 a 79) ou `errado` (abaixo de 40).
- `comentario`: em duas a quatro frases, falando com ele ("você explicou bem…"): o que está certo
  e, se houver, o que está **errado** (o erro de conceito vem primeiro, com a correção).
- `faltou`: o que uma resposta completa traria a mais, em uma ou duas frases; vazio se nada.

E em `resumo`: duas ou três frases sobre o que ele domina e o que revisar, apontando a seção da
lição.

## Como corrigir

1. **Pelo conteúdo, não pela forma.** Outras palavras, outra ordem, exemplo próprio: valem. Erro
   de português e texto curto não tiram nota se a ideia está lá.
2. **Os pontos essenciais pesam mais.** Sem um ponto essencial, a nota não passa de 70. Com todos
   os essenciais e só faltando detalhe, de 80 a 95. Com tudo, 100.
3. **Erro de conceito pesa mais que omissão.** Uma afirmação errada (que a lição contradiz) tira
   mais nota que um ponto que faltou, e o comentário diz qual é o certo.
4. **O que vai além do gabarito** conta a favor quando está certo (confira na lição) e contra
   quando está errado.
5. **Resposta em branco, "não sei" ou fora do assunto:** nota 0, e o comentário diz em que seção
   da lição está a resposta.
6. **Seja justo e direto.** Sem elogio vazio nem rigor de banca: o objetivo é ele saber o que
   já entendeu e o que voltar a estudar.
