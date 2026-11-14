# Criação de trilhas: histórico e aprendizado

> O **método** está em [`regras/criar-trilha.md`](../regras/criar-trilha.md) (o que fazer, os
> critérios C1–C15 e as lições aprendidas). Este arquivo guarda **como cada trilha foi criada**, para
> a próxima sair melhor. O conteúdo das trilhas e o registro detalhado de cada rodada ficam no
> conteúdo do homelab (fora do GitHub); aqui fica só o que generaliza.

## Ferramentas

- `scripts/revisar_trilha.py`: o revisor automático. Recebe o pacote de importação (o mesmo JSON de
  `POST /api/importar`) e um `escopo.json` (documentos do recorte e seções excluídas, com motivo).
  Mede a leitura de cada seção citada, confere âncoras, links internos dos documentos e das lições,
  testa os links externos e aplica C2–C4, C8–C12, C14 e C15. Roda num container:

  ```sh
  docker run --rm -v "$PWD:/w" -w /w python:3.12-alpine \
      python -I scripts/revisar_trilha.py pacote.json escopo.json
  ```

- Os critérios manuais (C1, C5–C7, C13) são conferidos lendo, e o resultado vai para o registro da
  rodada.

## Trilha 1: Home lab: rede (14/11/2026)

**Entrada:** os documentos de estudo do homelab (01 OpenWrt, 02 Wi-Fi, 03 redes separadas e
firewall, 05 WAN e SQM, a seção de IP fixo do 06) e o projeto da rede. Profundidade 3 (sólido).

**Referências de ordem e cobertura:** os cinco domínios do CompTIA Network+ (N10-009) e a ordem do
Kurose & Ross (*Computer Networking*). Fatos conferidos em fonte primária: RFCs 826, 1918, 2131,
3022, 3927, 6598, 7766 e 8290; o `wireless-regdb` (regras de rádio do Brasil); documentação do
OpenWrt.

| Rodada | Etapas | Revisor | Leitura manual | O que mudou |
|---|---|---|---|---|
| v1 | 21 em 5 módulos, 8,6 h | 14 falhas (links internos quebrados nos documentos) | NAT usado antes de ensinado; critérios que não medem o nível; IP fixo no módulo errado; falta diagnóstico (24% do Network+); etapas magras (1–2 min de leitura declaradas como 20–30) | — |
| v2 | 15 em 5 módulos, 6,7 h | 1 falha (verbo do objetivo fora do nível) | termo `br-lan` usado antes da etapa que explica bridge; citação a um diagnóstico que não era material; IPv6 sem lugar no recorte | etapas juntadas; 4 lições próprias (roteamento, DNS, NAT com errata, diagnóstico); módulo Diagnóstico; testes e práticas dos documentos viram critério e material; links do glossário e do índice corrigidos |
| v3 | 15 em 5 módulos, 6,7 h | **passou** (3 avisos registrados) | **passou** | ordem do módulo de fundamentos; objetivo do rádio; material de contexto; recorte com IPv6 |

**Achados no material de origem (para o dono decidir):**

- Errata: o documento 05 (§2.3) diz que o 03 tratou de NAT entre as redes internas; o 03 não trata
  de NAT, e entre as redes internas não há NAT. A trilha mostra a errata e ensina o correto.
- Complemento: o 02 explica os canais 1, 6 e 11 sem dizer que no Brasil o regulatório permite até o
  13 (a escolha continua certa; falta o porquê).
- Nuance: o 03 associa o REJECT ao "connection refused" do 01, cuja causa era porta sem programa;
  o sintoma é o mesmo porque o REJECT responde ao TCP com reset.
- Corrigido direto (mecânico): 14 links do glossário e do índice apontavam para âncoras sem o hífen
  final que o 🔴 no fim do título produz.

**Generalização** (virou lição em `regras/criar-trilha.md` §6): ordem da prática ≠ ordem do
conceito; "citado" ≠ "ensinado"; regra de rádio depende do país; bloqueio de robô depende do
cliente; etapa magra infla o tempo; critério mede o nível do objetivo; reler a ordem depois de
mexer; práticas e testes do material são material; errata visível.
