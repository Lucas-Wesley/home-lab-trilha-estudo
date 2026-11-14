# DNS: de um nome para um IP

> Pré-requisitos: portas, TCP e UDP. · Profundidade: sólido.
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

## Para que serve

🔴 A rede entrega pacotes por **IP**; as pessoas usam **nomes** (`openwrt.org`). O **DNS**
(*Domain Name System*) é o serviço que traduz um no outro. Sem ele, um site abre pelo IP, mas não
pelo nome: é exatamente o sintoma de uma rede IoT sem a regra de DNS (etapa das exceções).

## O caminho de uma pergunta

🔴 Quatro papéis aparecem numa resolução:

| Papel | O que faz | Na casa |
|---|---|---|
| **Resolver do aparelho** (*stub*) | pergunta, e só | o sistema do PC, do celular, da lâmpada |
| **Resolver recursivo** | faz o trabalho de achar a resposta e guarda em cache | o roteador (o `dnsmasq` do OpenWrt), que repassa ao resolver da operadora ou a um público |
| **Servidores raiz e de TLD** | dizem quem responde por `.org`, `.br`… | na internet |
| **Servidor autoritativo** | tem a resposta oficial do domínio | o servidor DNS do próprio site |

```
PC ──"qual o IP de openwrt.org?"──► roteador 192.168.1.1 (resolver da casa)
                                       │ tem em cache? responde na hora
                                       │ não tem? pergunta adiante:
                                       ▼
                         resolver da operadora / público
                         → raiz: "quem cuida de .org?"
                         → TLD .org: "quem cuida de openwrt.org?"
                         → autoritativo de openwrt.org: "o IP é …"
PC ◄──────────── resposta (e o roteador guarda no cache) ─────────────┘
```

🔴 **Como o PC sabe a quem perguntar?** Pelo **DHCP**: junto com o IP e o gateway, o roteador
entrega o endereço do servidor DNS, que na casa é o próprio roteador (`192.168.1.1`,
`192.168.20.1`, `192.168.30.1`, um em cada rede). Por isso, numa rede com *Input = reject*, a
pergunta de DNS precisa de uma exceção para chegar ao roteador.

## Cache e TTL

🟡 Cada resposta vem com um **TTL** (em segundos): por quanto tempo ela pode ser guardada. Enquanto
vale, o roteador responde do cache, sem sair de casa: a segunda visita ao mesmo site é mais rápida.
(Este TTL do DNS é outro conceito, sem relação com o TTL do pacote IP.)

## Porta e protocolo

🔴 O DNS usa a **porta 53**, normalmente em **UDP** (pergunta e resposta cabem num pacote, sem o
custo de abrir conexão). O **TCP** na porta 53 também é usado, por exemplo quando a resposta é grande
demais, e todo servidor precisa aceitá-lo. Por isso a regra de DNS da casa libera **TCP e UDP 53**.

## Tipos de registro mais comuns

🟡

| Tipo | Guarda | Exemplo |
|---|---|---|
| `A` | endereço IPv4 | `openwrt.org → 1.2.3.4` (ilustrativo) |
| `AAAA` | endereço IPv6 | |
| `CNAME` | "este nome é apelido daquele" | `www → site principal` |
| `PTR` | o caminho inverso: IP → nome | usado em diagnóstico e logs |

## Na prática: como observar

- `nslookup openwrt.org`: mostra quem respondeu (deve ser o roteador) e o IP.
- `nslookup openwrt.org 1.1.1.1`: pergunta direto a um resolver público, pulando o roteador. Se
  esta funciona e a de cima não, o problema está no DNS da casa.
- `ipconfig /all`: a linha **Servidores DNS** mostra o que o DHCP entregou.

## Para onde isto vai

⚪ O plano da casa inclui um **AdGuard Home** no M920q como DNS da rede, bloqueando anúncios e
rastreadores por nome. Ele assume o papel de resolver recursivo; o resto do caminho continua igual.

## Teste seu entendimento

1. Na rede de convidados, sem a regra de DNS, o que funciona e o que não funciona?
2. Por que a segunda visita a um site costuma resolver mais rápido?
3. O `nslookup site 1.1.1.1` funciona e o `nslookup site` não. Onde está o problema?
4. Por que a regra de DNS libera TCP **e** UDP?

<details><summary>Respostas</summary>

1. O aparelho recebe IP (há regra de DHCP) e alcança sites **pelo IP**, mas não **pelo nome**: a
   pergunta de DNS ao roteador é rejeitada.
2. Porque o resolver da casa guardou a resposta em cache, durante o TTL dela.
3. No DNS da casa (o roteador, ou a regra que deixa o aparelho falar com ele): a internet e o
   resolver público estão funcionando.
4. Porque o DNS usa normalmente UDP, mas também TCP (respostas grandes), e os servidores precisam
   aceitar os dois.

</details>

## Referências

- [padrão] RFC 1034, *Domain Names: Concepts and Facilities*: https://www.rfc-editor.org/rfc/rfc1034
- [padrão] RFC 7766, *DNS Transport over TCP* (o TCP é obrigatório): https://www.rfc-editor.org/rfc/rfc7766
- [oficial] OpenWrt, DNS e DHCP (`dnsmasq`): https://openwrt.org/docs/guide-user/base-system/dhcp
- [artigo] Cloudflare, *What is DNS?*: https://www.cloudflare.com/learning/dns/what-is-dns/
