# Roteamento e gateway: entregar direto ou mandar para o roteador

> Pré-requisitos: camadas, MAC e IP, sub-rede. · Profundidade: sólido.
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

## A pergunta que todo aparelho faz

🔴 Antes de mandar **qualquer** pacote, o PC (ou o celular, ou a lâmpada) responde a uma pergunta só:
**o destino está na minha sub-rede?**

- **Sim** → entrega **direto**: descobre o MAC do destino pelo ARP e manda o quadro para ele.
- **Não** → entrega para o **gateway** (o roteador): descobre o MAC **do gateway** pelo ARP e manda
  o quadro para ele. O endereço **IP** de destino continua sendo o do destino final; só o **MAC**
  de destino é o do roteador.

Para responder, o aparelho aplica a máscara ao próprio IP e ao IP de destino e compara o resultado
(a "parte da rede"):

```
PC: 192.168.1.50/24        destino 192.168.1.10 (o M920q)
    rede do PC  = 192.168.1.0
    rede do destino = 192.168.1.0   → mesma rede → entrega direto

PC: 192.168.1.50/24        destino 1.1.1.1
    rede do PC  = 192.168.1.0
    rede do destino = 1.1.1.0       → outra rede → manda para o gateway 192.168.1.1
```

É por isso que o gateway precisa estar **dentro** da sub-rede do aparelho: senão, ele não teria
como descobrir o MAC do gateway (o ARP só funciona no mesmo enlace).

## A tabela de rotas

🔴 A decisão acima é um caso particular de algo mais geral: todo aparelho com IP tem uma **tabela de
rotas**, uma lista de "para tal faixa, saia por tal caminho". Duas linhas aparecem em quase todo
aparelho de casa:

| Destino | Caminho | De onde veio |
|---|---|---|
| `192.168.1.0/24` | direto, pela placa de rede (sem gateway) | criada sozinha ao receber o IP |
| `0.0.0.0/0` (a **rota padrão**, *default*) | via `192.168.1.1` | o gateway que o DHCP entregou |

Quando mais de uma linha serve, vence a **mais específica** (o prefixo mais longo: `/24` ganha de
`/0`). A rota padrão é a que "sobra": tudo o que não casa com nada mais vai para o gateway.

🟡 O **roteador** tem a mesma tabela, só que maior: uma linha por rede que ele conhece (Principal,
IoT, Convidados) e a rota padrão apontando para a operadora.

```
roteador (ip route, resumido):
  default via <gateway da operadora> dev wan        ← tudo o que não é de casa
  192.168.1.0/24  dev br-lan   src 192.168.1.1      ← rede Principal
  192.168.20.0/24 dev br-iot   src 192.168.20.1     ← rede IoT
  192.168.30.0/24 dev br-guest src 192.168.30.1     ← rede Convidados
```

## Roteamento entre as redes de casa

🔴 Quando o PC (Principal) fala com uma lâmpada (IoT), o roteador **roteia** de uma rede para a
outra: recebe pela `br-lan`, consulta a tabela, entrega pela `br-iot`. **Não há tradução de
endereço** nesse caminho: a lâmpada vê o IP verdadeiro do PC. O que decide se o pacote pode passar
é o **firewall** (a etapa das zonas). A tradução de endereço (NAT) só acontece na saída para a
internet (a etapa de NAT).

## TTL: por que um pacote não roda para sempre

🟡 Todo pacote IP carrega um contador, o **TTL** (*time to live*). Cada roteador que o encaminha
diminui o TTL em 1; quem recebe um pacote com TTL esgotado o descarta e avisa a origem (uma
mensagem ICMP). Isso impede que um erro de rotas deixe pacotes circulando para sempre.

⚪ O `tracert` (Windows) usa exatamente isso: manda pacotes com TTL 1, 2, 3… e anota quem avisou
em cada passo. O resultado é a lista dos roteadores do caminho.

## Na prática: como observar

- No PC: `route print` (Windows) mostra a tabela; procure a linha `0.0.0.0 … 192.168.1.1`.
- No PC: `ipconfig /all` mostra o **Gateway Padrão** que o DHCP entregou.
- No roteador (SSH): `ip route` mostra as rotas da casa; `ip route get 1.1.1.1` mostra qual linha
  seria usada para um destino.
- `tracert 1.1.1.1`: o primeiro passo é sempre o seu roteador (`192.168.1.1`).

## Teste seu entendimento

1. O PC `192.168.1.50/24` vai mandar um pacote para `192.168.20.31`. Qual MAC vai no quadro?
2. Por que a rota padrão não "engole" o tráfego para `192.168.1.10`, se `0.0.0.0/0` casa com tudo?
3. Quando o PC fala com a lâmpada, a lâmpada vê qual IP de origem? Por quê?
4. Um aparelho recebeu IP `192.168.1.77/24` e gateway `192.168.2.1`. O que acontece quando ele
   tenta abrir um site?

<details><summary>Respostas</summary>

1. O MAC do **roteador** (`192.168.1.1`): o destino está em outra sub-rede, então o quadro vai para
   o gateway. O IP de destino continua `192.168.20.31`.
2. Porque vence a rota mais específica: `192.168.1.0/24` é mais específica que `0.0.0.0/0`.
3. O IP verdadeiro do PC (`192.168.1.50`): entre as redes de casa há roteamento, sem NAT.
4. Não consegue: o gateway não está na sub-rede dele, então não há como descobrir o MAC do gateway
   pelo ARP. Ele fica sem saída para fora da própria rede.

</details>

## Referências

- [padrão] RFC 1812, *Requirements for IP Version 4 Routers* (encaminhamento, TTL):
  https://www.rfc-editor.org/rfc/rfc1812
- [padrão] RFC 4632, *CIDR* (prefixos e a regra do mais específico):
  https://www.rfc-editor.org/rfc/rfc4632
- [livro] Kurose & Ross, *Computer Networking*, cap. 4 (camada de rede):
  https://gaia.cs.umass.edu/kurose_ross/
