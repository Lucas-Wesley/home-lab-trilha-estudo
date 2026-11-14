# Registro de criação: Home lab: rede

> Como esta trilha foi montada, rodada por rodada (método: `regras/criar-trilha.md`). Os relatórios
> automáticos estão no fim; aqui ficam a leitura manual e as decisões.

## Entrada

- **Material:** documentos 01, 02, 03, 05, a seção 4.1 do 06 e o projeto da rede (casa inteligente).
- **Inventário:** os documentos seguem a ordem da prática (o dia a dia da montagem), não a dos
  conceitos. Mapa de cobertura (menções × explicações): DNS aparece 13 vezes no 03, todas como regra
  de firewall, nenhuma explicando a resolução; roteamento e rota padrão quase não aparecem; NAT é
  explicado pela primeira vez no 05 como "revisitado"; VLAN não é usada na casa.
- **Referências de ordem e cobertura:** CompTIA Network+ N10-009 (conceitos 23%, implementação 20%,
  operação 19%, segurança 14%, diagnóstico 24%) e a ordem do Kurose & Ross.

## Rodada 1 (v1: 21 etapas, 8,6 h)

**Revisor:** 14 falhas, todas de links internos (glossário e índice apontando para âncoras sem o
hífen final que o 🔴 no fim do título produz); 3 avisos (páginas da Cloudflare bloqueiam scripts).

**Leitura manual:**
- C5: a etapa de NAT começava por "NAT, revisitado"; NAT nunca tinha sido ensinado. Ao conferir, o
  próprio documento 05 §2.3 está errado (diz que o 03 tratou de NAT entre redes internas).
- C2: critérios que não mediam o nível (objetivo "aplicar" com critério "listar"; critério sobre
  Carrier numa etapa sobre modo bridge).
- C6: IP fixo no módulo Internet, longe do DHCP.
- C7: faltava diagnóstico (24% do Network+, e é o nível alvo).
- C14: as etapas tinham 1 a 2 minutos de leitura e estavam declaradas com 20 a 30. O critério do
  revisor era frouxo demais e deixou passar; foi apertado.

**Decisões:** juntar etapas (21 → 15); escrever lições próprias para roteamento, DNS, NAT (com a
errata) e diagnóstico; criar o módulo Diagnóstico; usar as seções "Na prática" e "Teste seu
entendimento" como material e critério; corrigir os 14 links nos documentos de origem (mecânico,
avisado).

## Rodada 2 (v2: 15 etapas, 6,7 h)

**Revisor:** 1 falha (o objetivo do rádio usava "escolher", verbo de avaliar, numa etapa de aplicar).

**Leitura manual:**
- C5: a lição de roteamento mostra `br-lan`/`br-iot` na tabela de rotas, e a etapa que explica
  bridge vinha depois dela.
- C5: o 05 §3.3 cita "o diagnóstico original"; ele não estava como material.
- C7: IPv6 aparece como contexto e não estava no recorte.
- C13, conferido em fonte primária: faixa do CGNAT (RFC 6598); canais 1 a 13 e DFS do Brasil
  (`wireless-regdb`, country BR); DHCP UDP 67 (RFC 2131); DNS TCP e UDP 53 (RFC 7766); faixas
  privadas (RFC 1918); 169.254/16 (RFC 3927); fw4 sobre nftables (OpenWrt 25.12). O 02 está certo
  sobre 1, 6 e 11, mas incompleto (no Brasil vão até o 13): virou nota na etapa.

**Decisões:** reordenar os fundamentos (o roteador por dentro antes do roteamento); objetivo do
rádio com "configurar"; o diagnóstico de outubro como apoio; IPv6 declarado no recorte.

## Rodada 3 (v3)

**Revisor:** passou, com 3 avisos aceitos (o link para o plano do Flashcards, que é outro projeto, e
as duas páginas da Cloudflare, que existem segundo o índice de busca e são só apoio).
**Leitura manual:** passou.

## Correções propostas para os documentos de origem

1. 05 §2.3: trocar "No doc 03 vimos o NAT entre as suas redes internas" por uma frase que diga que
   entre as redes internas não há NAT, só roteamento e firewall; o NAT acontece na saída para a WAN.
2. 02 §3.2: acrescentar que no Brasil os canais vão de 1 a 13, e por que 1, 6 e 11 continuam a
   escolha segura.
3. 03 §4.4: dizer que o "connection refused" do 01 veio de porta sem programa, e que o REJECT do
   OpenWrt produz o mesmo sintoma.

---

# Relatórios do revisor
