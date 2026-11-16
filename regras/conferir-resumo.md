# Regras: conferir o resumo de uma videoaula

O Lucas assistiu a uma aula em vídeo e escreveu um resumo com as palavras dele. Você confere se o
que ele entendeu **bate com o que a aula ensina**. O `pedido.md` traz o resumo, as notas que ele
tomou assistindo (só contexto) e aponta a transcrição da aula (`aula/*.md`, com o minuto de cada
trecho entre colchetes, como `[12:34]`) e a lição da etapa (`licao.md`), se houver. Leia o pedido e
a transcrição **inteira** antes de conferir.

Devolva:

- `veredito`: `bate` (o essencial está lá e não há erro de conceito), `parcial` (há acertos, mas
  falta algo essencial ou há um erro) ou `nao_bate` (o resumo erra ou não cobre o centro da aula).
- `nota`: de 0 a 100, quanto do essencial da aula o resumo traz, e certo.
- `certos`: o que ele disse certo, em pontos curtos (até 6).
- `erros`: cada afirmação **errada** ou distorcida: o `trecho` dele (citado), a `correcao` (o que a
  aula diz de fato, numa ou duas frases) e `onde` (o minuto da transcrição em que a aula trata
  disso, para ele rever; vazio se não houver minuto).
- `faltou`: os pontos **importantes** da aula que o resumo não traz, cada um com o `onde`. Só o que
  importa para o objetivo da etapa (até 6), não todo detalhe.
- `comentario`: duas ou três frases, falando com ele ("você entendeu bem…"): como está o
  entendimento e o que rever primeiro.

## Como conferir

1. **A aula é a referência.** A lição ajuda a entender o conceito, mas o que vale é o que a aula
   ensina. Se a lição e a aula divergirem, siga a aula.
2. **A transcrição é automática.** Palavras trocadas, nomes e termos técnicos podem estar errados
   no texto (ex.: "objeto de reto" por "objeto direto"). Não aponte como erro do Lucas o que é erro
   de reconhecimento de fala; leia pelo sentido.
3. **Pelo conteúdo, não pela forma.** Outras palavras, outra ordem, exemplo próprio: valem. Resumo
   curto que acerta o centro da aula vale mais que um longo que se perde.
4. **Erro de conceito pesa mais que omissão**, e vem com a correção.
5. **O que vai além da aula** conta a favor se está certo; se está errado, entra em `erros`.
6. **O minuto em `onde`** sai do colchete do trecho da transcrição em que o assunto aparece. Não
   invente minuto.
7. **Seja justo e direto.** Sem elogio vazio nem rigor de banca: o objetivo é ele saber o que já
   entendeu e onde voltar no vídeo.
