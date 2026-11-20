# Regras: sugerir as trilhas de cada fragmento

Um vídeo da Entrada foi dividido em fragmentos por assunto, e o Lucas confirmou a divisão. O
`pedido.md` traz os fragmentos (número, tempo, título, resumo, assuntos e tipo) e as trilhas dele
(id, título, objetivo e os tópicos de cada módulo). Para cada fragmento, diga a que trilha(s) ele
vai — é sugestão, o Lucas confere e pode mudar.

## Como escolher

- Leia o resumo e os assuntos do fragmento e compare com o objetivo e os tópicos de cada trilha.
- O mesmo fragmento pode ir a **várias** trilhas quando serve de verdade a cada uma (ex.: um trecho
  sobre S3 vai para "AWS para engenharia de software" e para "System Design"); não espalhe por
  generosidade — servir de verdade é ensinar o que a trilha pede.
- Existe a trilha **"Conhecimento de base"** para conhecimentos soltos de tecnologia que não
  pertencem a nenhuma trilha grande (o que é UTF-8, diferença entre sistemas de arquivos…): use-a
  quando o assunto é técnico e útil, mas não cabe nas outras.
- Antes de dizer que nenhuma trilha serve, **procure o assunto nos tópicos de todas as trilhas**
  (ex.: VPN pode estar no módulo de segurança de uma trilha de redes).
- `trilha_nova` só quando o assunto é grande e recorrente o bastante para virar trilha (não para
  um tópico solto). É só o nome sugerido; o Lucas cria se quiser.
- Fragmentos `opiniao` e `fora` costumam ir a **nenhuma**: use `motivo_nenhuma` dizendo por quê
  ("é a opinião do autor, não aula").

## O que devolver

- `destinos`: uma entrada por fragmento, com `fragmento` (o número do pedido), `trilhas` (os
  `trilha_id` do pedido, cada um com um `motivo` curto para o Lucas), `trilha_nova` e
  `motivo_nenhuma` (vazios quando não se aplicam).
- `notas`: **no máximo duas frases curtas**, só sobre as escolhas (ex.: "os três primeiros são do
  mesmo assunto e foram juntos para a trilha X"). Vazio se não tiver nada a dizer.
