# Diagnóstico: camada por camada, de baixo para cima

> Pré-requisitos: todas as etapas anteriores. · Profundidade: sólido.
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

## O método

🔴 "A internet caiu" quase nunca é a internet. O jeito mais rápido de achar o problema é subir as
camadas **de baixo para cima** e parar na primeira que falha: cada teste só faz sentido se o de
baixo passou.

| # | Pergunta | Camada | Teste no PC (Windows) | Resultado que indica problema |
|---|---|---|---|---|
| 1 | Há sinal no cabo / estou no Wi-Fi certo? | física / enlace | luz da porta; nome da rede conectada | sem luz; SSID errado (ex.: entrou na de convidados) |
| 2 | Recebi um IP? | rede (DHCP) | `ipconfig /all` | endereço `169.254.x.x` = o DHCP não respondeu |
| 3 | Alcanço o roteador? | rede (local) | `ping 192.168.1.1` | sem resposta = problema dentro de casa |
| 4 | Alcanço a internet pelo IP? | rede (rota, NAT, WAN) | `ping 1.1.1.1` | falha aqui com o 3 passando = WAN, operadora |
| 5 | Resolvo nomes? | aplicação (DNS) | `nslookup openwrt.org` | o 4 passa e o 5 não = DNS |
| 6 | O serviço responde? | transporte / aplicação | abrir o site; `Test-NetConnection host -Port 443` | "recusado" (nada escutando ou REJECT) × tempo esgotado (DROP ou caminho) |

🔴 **Por que `169.254.x.x` diz tanto:** quando o DHCP não responde, o Windows inventa um endereço
nessa faixa (reservada para isso, "link-local"). Ver um `169.254` é ver o passo 2 falhar: o problema
está entre o aparelho e o servidor DHCP (cabo, Wi-Fi, ou a regra de DHCP da rede fechada).

## Do lado do roteador

🟡 Quando o PC não basta, os mesmos passos podem ser vistos no roteador (SSH):

| Para ver | Comando |
|---|---|
| interfaces, IPs e estado (`UP`, `NO-CARRIER`) | `ip addr` |
| rotas, inclusive a rota padrão para a WAN | `ip route` |
| quem recebeu IP pelo DHCP | `cat /tmp/dhcp.leases` |
| o que o sistema registrou (DHCP, Wi-Fi, firewall) | `logread`, `logread -f` (acompanhar) |
| as regras do firewall montadas (fw4 usa nftables) | `nft list ruleset` |
| se o SQM está agindo (aparece o `cake` com contadores) | `tc -s qdisc show dev wan` |

## Três casos da própria casa

🔴 Cada um mostra o método funcionando:

1. **Porta WAN ou LAN?** (instalação, documento 01) — cabo ligado (passo 1 ok) mas sem IP (passo 2
   falha): estava na porta errada. Um IP fixo no PC isolou a dúvida.
2. **SQM "ligado" sem efeito** (documento 05) — a configuração foi aceita, mas o teste mostrava
   velocidades acima do limite. Comparar o **esperado** com o **medido** revelou o device errado.
   O `tc -s qdisc` teria mostrado na hora que não havia `cake` na `wan`.
3. **Convidado com IP mas sem sites** (documento 03) — passos 2, 3 (até onde o firewall deixa) e 4
   passam, o 5 falha: falta a regra de DNS.

## Teste seu entendimento

1. O celular de uma visita mostra "conectado, sem internet" e `169.254.10.4`. Qual passo falhou e o
   que você confere primeiro?
2. `ping 1.1.1.1` responde, `nslookup openwrt.org` não. Onde está o problema?
3. Qual a diferença, para quem diagnostica, entre "conexão recusada" e "tempo esgotado"?
4. Como você confirmaria, em um comando, que o SQM está agindo na WAN?

<details><summary>Respostas</summary>

1. O passo 2 (DHCP). Conferir se o SSID está ligado à rede certa e se a regra de DHCP (UDP 67) da
   zona `guest` existe.
2. No DNS: a rede e a WAN funcionam. Conferir o servidor DNS entregue pelo DHCP e a regra de DNS.
3. "Recusado" é uma resposta: chegou-se ao destino e nada escuta ali, ou um REJECT respondeu.
   "Tempo esgotado" é ausência de resposta: DROP, caminho quebrado ou destino desligado.
4. `tc -s qdisc show dev wan`: deve aparecer uma fila `cake` com contadores crescendo.

</details>

## Referências

- [padrão] RFC 3927, endereços link-local IPv4 (`169.254.0.0/16`): https://www.rfc-editor.org/rfc/rfc3927
- [oficial] Microsoft, comando `ipconfig`: https://learn.microsoft.com/windows-server/administration/windows-commands/ipconfig
- [oficial] OpenWrt, firewall (fw4 sobre nftables): https://openwrt.org/docs/guide-user/firewall/overview
