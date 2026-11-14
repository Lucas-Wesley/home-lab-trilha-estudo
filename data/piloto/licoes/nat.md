# NAT: a casa inteira sai por um único IP

> Pré-requisitos: roteamento e gateway; firewall com estado. · Profundidade: sólido.
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

> **Errata do documento 05.** A seção 2.3 do [05](05-claro-bridge-wan-sqm.md#23-nat-revisitado-agora-do-lado-de-fora-)
> diz que o documento 03 tratou do "NAT entre as suas redes internas". O 03 não trata de NAT, e
> entre as redes internas (Principal, IoT e Convidados) **não há NAT**: o roteador só roteia e o
> firewall decide. O NAT acontece **uma vez**, na saída para a internet. Esta lição é a primeira
> explicação de NAT da trilha; o 05 vem depois, como aplicação.

## O problema que o NAT resolve

🔴 O IPv4 tem cerca de 4,3 bilhões de endereços, poucos para o número de aparelhos do mundo. A
solução foi reservar faixas **privadas**, que qualquer casa ou empresa pode usar por dentro, mas
que **não andam pela internet**:

| Faixa privada (RFC 1918) | Tamanho |
|---|---|
| `10.0.0.0/8` | 16 milhões de endereços |
| `172.16.0.0/12` | 1 milhão |
| `192.168.0.0/16` | 65 mil (a da casa: `192.168.1/20/30.x`) |

Um pacote com origem `192.168.1.50` não teria como receber resposta da internet: milhões de casas
usam esse mesmo endereço. Alguém precisa trocar o endereço privado por um **público** na saída e
desfazer a troca na volta. Esse alguém é o roteador, e a troca é o **NAT** (*Network Address
Translation*).

## Como funciona (NAT com portas)

🔴 O roteador da casa usa a variante mais comum, que traduz **endereço e porta** (chamada de NAPT,
ou *masquerade* no Linux). Na saída, ele troca a **origem** do pacote pelo IP da WAN e anota a troca
numa tabela:

```
saindo:   192.168.1.50:51000  →  1.1.1.1:443
              │ o roteador troca a origem e anota
              ▼
          187.65.178.135:51000 →  1.1.1.1:443

tabela de traduções (resumida):
  interno               externo                destino
  192.168.1.50:51000    187.65.178.135:51000   1.1.1.1:443
  192.168.20.31:40100   187.65.178.135:40100   (servidor da lâmpada):8883
```

Na volta, a resposta chega para `187.65.178.135:51000`; o roteador consulta a tabela e devolve
para `192.168.1.50:51000`. Se duas máquinas usarem a mesma porta de origem, o roteador troca
também a porta, para cada conversa continuar única.

🔴 **A tabela é a mesma do firewall com estado** (o *conntrack*): a conversa que saiu é lembrada, e
a resposta volta. Uma conexão que **começa lá fora** não tem linha na tabela, então o roteador não
sabe para qual aparelho mandar e a descarta. Para servir algo para a internet é preciso uma regra
explícita, o **redirecionamento de porta** (*port forwarding*): "o que chegar na porta X da WAN vai
para tal IP interno".

## Onde o NAT está ligado na casa

🔴 No OpenWrt, o NAT é uma opção da **zona** do firewall: **Masquerading**, ligada na zona `wan` e
desligada nas outras. Por isso:

- Principal → internet, IoT → internet, Convidados → internet: **com NAT** (saem pela `wan`).
- Principal → IoT: **sem NAT** (roteamento entre redes de casa; a lâmpada vê o IP real do PC).

## Os vizinhos do NAT

🟡 Três situações aparecem na etapa seguinte (WAN), todas consequência do que está acima:

- **NAT duplo:** dois roteadores em sequência, cada um com o seu NAT. O redirecionamento de porta
  precisa ser feito nos dois, e alguns serviços quebram.
- **CGNAT:** a operadora faz um NAT dela, com vários clientes atrás de um IP público. A faixa
  reservada para isso é `100.64.0.0/10` (RFC 6598). Atrás de CGNAT, nenhum redirecionamento de
  porta na casa adianta.
- **IPv6:** com endereços de sobra, cada aparelho pode ter endereço público próprio, e o NAT deixa de
  ser necessário (o firewall continua necessário).

## Na prática: como observar

- No roteador, LuCI: **Network → Firewall**, coluna **Masquerading** de cada zona.
- No roteador (SSH): `cat /proc/net/nf_conntrack | head` mostra conversas em andamento, com o
  endereço interno e o traduzido. (Se o arquivo não existir na sua versão, o pacote
  `conntrack` traz o comando `conntrack -L`.)
- Compare o IP da WAN (LuCI, Status) com o que um site de "qual é o meu IP" mostra: iguais
  indicam que não há outro NAT no caminho.

## Teste seu entendimento

1. Por que um pacote com origem `192.168.1.50` não poderia sair para a internet sem tradução?
2. O que acontece quando alguém da internet tenta abrir uma conexão para o IP da WAN, sem nenhum
   redirecionamento de porta configurado?
3. O Home Assistant (Principal) comanda uma lâmpada (IoT). Há NAT nesse caminho? Por quê?
4. Qual a relação entre a tabela de NAT e o firewall com estado?

<details><summary>Respostas</summary>

1. Porque endereços privados (RFC 1918) não são roteados na internet e se repetem em milhões de
   redes: a resposta não teria como voltar.
2. Não há linha na tabela para aquela conexão, então o roteador não sabe para qual aparelho
   mandar: ela é descartada (ou rejeitada).
3. Não. Entre as redes de casa o roteador só roteia; o NAT (masquerading) está ligado apenas na
   zona `wan`.
4. São a mesma tabela de conexões (conntrack): ela permite tanto deixar a resposta voltar (estado)
   quanto desfazer a tradução de endereço e porta (NAT).

</details>

## Referências

- [padrão] RFC 1918, *Address Allocation for Private Internets*: https://www.rfc-editor.org/rfc/rfc1918
- [padrão] RFC 3022, *Traditional IP Network Address Translator* (NAT e NAPT): https://www.rfc-editor.org/rfc/rfc3022
- [padrão] RFC 6598, *Shared Address Space* (a faixa do CGNAT): https://www.rfc-editor.org/rfc/rfc6598
- [oficial] OpenWrt, configuração do firewall (opção `masq` da zona): https://openwrt.org/docs/guide-user/firewall/firewall_configuration
