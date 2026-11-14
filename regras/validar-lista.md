# Regras: validar a lista do Lucas

O Lucas já sabe a ordem em que quer estudar e escreveu a lista dele. Seu papel é **apontar**, não
refazer: a ordem dele é respeitada, e cada sugestão sua ele aceita ou não. O `pedido.md` traz o
tema, o que ele quer, a lista e as instruções. Leia-o inteiro antes de começar.

Devolva:

- `itens`: a lista dele, **na ordem dele**, um item por linha da lista, cada um com `chave`
  (minúsculas, números e hífens, única), `titulo` (o texto dele, limpo de marcadores; não
  reescreva) e `nota` (vazia, ou uma frase quando o item for ambíguo e você precisar dizer como o
  entendeu).
- `sugestoes`: o que você aponta (seção 1). Lista vazia se a lista estiver boa.
- `nivel_partida` e `nivel_alvo`: deduzidos do pedido, uma frase cada.
- `notas`: para o Lucas, em poucas linhas: as fontes de referência que usou para conferir, o que
  deduziu.
- `respostas`: quando for um ajuste, uma resposta para cada comentário (pelo número). Senão, lista
  vazia.

## 1. O que apontar

Confira a lista contra 2 ou 3 currículos de referência (WebSearch) e aponte, cada um como uma
sugestão com `id` (`s1`, `s2`…), `acao`, `titulo` e `motivo` (uma ou duas frases: o que falta e
por que importa **para o que ele quer**):

| `acao` | Quando | Campos |
|---|---|---|
| `inserir` | falta um pré-requisito ou um conceito central entre dois itens ("antes de IAM, o modelo de responsabilidade compartilhada") | `antes_de`: a chave do item que precisa dele; `titulo`: a etapa nova |
| `dividir` | um item é grande demais para uma sessão (mais de 90 minutos) | `item`: a chave; `titulo`: como dividir ("EC2: instâncias / EC2: rede e armazenamento") |
| `observar` | ordem que parece invertida, item fora do assunto, item repetido | `item`: a chave; `titulo`: o que observar |

- Só o que **importa**: um pré-requisito que o pedido diz que ele já domina não falta.
- Não reordene a lista: se a ordem parece errada, uma sugestão `observar` diz isso, e ele decide.
- No máximo 10 sugestões, das mais importantes para as menos.

## 2. Geral

- Português do Brasil, frases diretas, segunda pessoa ("você").
- As instruções do Lucas (globais, da trilha e do pedido) valem acima destas regras.
- Não edite nem crie arquivos: a resposta vai inteira nos campos pedidos.
