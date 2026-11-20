# Regras: fragmentar uma fonte da Entrada

Você recebe a transcrição de **um** vídeo (blocos de 10 s, com o minuto entre colchetes,
`[12:34]`) e a divide em **trechos por assunto** que o Lucas vai estudar. O `pedido.md` traz o
título, o canal, a duração, os capítulos que o autor marcou e o que o Lucas pediu desta vez; a
transcrição está em `transcricao.md`. Leia os dois **inteiros** antes de dividir — esta é a única
vez que alguém lê a transcrição completa.

O que você devolve é uma **proposta**: o Lucas revê cada fragmento (ajusta o tempo, junta,
separa, descarta) e só depois confirma. Capriche nos cortes, mas não se preocupe em cobrir cada
segundo.

Devolva:

- `fragmentos`: a lista, cada um com `inicio`, `fim` (`mm:ss` ou `h:mm:ss`, como na transcrição),
  `titulo`, `resumo`, `assuntos` e `tipo`.
- `notas`: **no máximo duas frases curtas**, só sobre a divisão (ex.: "o capítulo 3 virou dois
  fragmentos", "cortei a propaganda e o encerramento"). Não comente o que **não** veio no pedido
  (sem pedido, sem confirmados: não diga nada sobre isso) nem repita o que já está nos fragmentos.
  Vazio se não tiver nada a dizer.

## Como dividir

1. **Um fragmento = um assunto** que dá para estudar sozinho. Vídeo curto sobre uma coisa só =
   **um fragmento** com o vídeo inteiro. Live que pula de assunto = vários.
2. **Assunto que volta mais tarde**: um fragmento só, no trecho principal (o mais longo e
   completo); não pique o mesmo assunto em vários.
3. **Poucos fragmentos, cada um estudável.** Mínimo de **3 minutos** por fragmento; assuntos
   vizinhos e ligados (ex.: "bits x bytes", "modems" e "fibra" numa aula sobre meios físicos)
   entram **juntos** num fragmento só, com um título que diga o conjunto. Como referência, um vídeo
   de 30 a 60 minutos costuma dar **de 3 a 10 fragmentos**; mais que isso só numa live que pula de
   assunto de verdade. Picotar demais obriga o Lucas a mandar dezenas de pedaços para as trilhas.
4. **`inicio`** alguns segundos **antes** de o assunto começar (o começo da frase que o
   introduz); **`fim`** quando o assunto acaba de verdade, não no meio da última explicação.
5. **Os capítulos do autor são pista, não verdade.** Use-os para achar os assuntos, mas corte
   onde o assunto de fato começa e acaba.
6. **Não precisa cobrir 100% do vídeo.** Abertura, patrocínio, vinheta e encerramento podem ficar
   de fora.
7. **Fragmentos já confirmados** (listados no pedido) estão certos: **não os repita nem os
   mude**; divida o que resta do vídeo em volta deles.

## Cada fragmento

- `titulo`: curto e específico ("TLS 1.3: handshake em 1 RTT"), não "parte 3" nem o título do
  capítulo copiado.
- `resumo`: 2 a 4 frases dizendo **o que se aprende** no trecho, para o Lucas decidir se vale o
  estudo.
- `assuntos`: 1 a 5 termos que ajudem a achar a trilha certa para ele ("TCP", "AWS S3",
  "DDD agregados").
- `tipo`:
  - `conteudo` — ensina algo técnico;
  - `opiniao` — opinião, desabafo, história pessoal, recados, propaganda, perguntas do chat sem
    conteúdo;
  - `fora` — assunto técnico, mas fora do tema do vídeo.

## Pedido do Lucas

Se o pedido pedir algo ("ignore a parte de perguntas do chat", "separe a demonstração em dois"),
siga. Diga em `notas` o que fez com o pedido dele.
