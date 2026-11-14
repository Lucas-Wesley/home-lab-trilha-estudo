# Revisão automática: Home lab: rede

- Etapas: 21 · módulos: 5 · tempo declarado: 515 min (8.6 h)
- Links conferidos: 11
- **Resultado: 14 falha(s)**, 4 aviso(s)

## Falhas

- **C10** `00-indice` liga para `06-proxmox-instalacao#41-reserva-de-ip-ip-fora-do-pool--static-lease`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#21-modem--roteador-duas-funções-que-moravam-no-mesmo-aparelho`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#22-carrier-o-tom-de-discagem-de-uma-porta-de-rede`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#31-nat-duplo-o-problema-que-o-modo-bridge-evita`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#32-ip-público--cgnat`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#33-bufferbloat-fila-grande-latência-alta`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#34-sqm-e-o-algoritmo-cake`, âncora que não existe
- **C10** `glossario` liga para `05-claro-bridge-wan-sqm#41-interface-lógica--dispositivo-físico--o-bug-que-encontramos`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#33-boot-por-pendrive-uefi-e-ordem-de-boot`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#22-bare-metal--em-cima-de-um-so-hypervisor-tipo-1--tipo-2`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#32-vm--container-lxc-as-duas-formas-de-isolar`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#31-o-que-é-o-proxmox-ve`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#34-repositórios-apt-enterprise--no-subscription`, âncora que não existe
- **C10** `glossario` liga para `06-proxmox-instalacao#44-kernel-e-por-que-o-reboot`, âncora que não existe

## Avisos

- **C10** `metodologia` liga para `PLANO.md`, que não está no pacote
- **C11** https://www.cloudflare.com/learning/network-layer/what-is-a-subnet/ → 403 (proteção anti-robô; não verificável aqui)
- **C11** https://www.cloudflare.com/learning/dns/what-is-dns/ → 403 (proteção anti-robô; não verificável aqui)
- **C11** https://www.cloudflare.com/learning/network-layer/what-is-routing/ → 403 (proteção anti-robô; não verificável aqui)

## Etapas

| Módulo | Etapa | Bloom | Min | Palavras | Leitura medida |
|---|---|---|---|---|---|
| Fundamentos | `camadas` | entender | 25 | 177 | ~11 |
| Fundamentos | `mac-ip-arp` | entender | 25 | 117 | ~11 |
| Fundamentos | `portas-tcp-udp` | entender | 20 | 153 | ~11 |
| Fundamentos | `ip-sub-rede-dhcp` | aplicar | 30 | 128 | ~11 |
| Fundamentos | `dns` | entender | 25 | 0 | ~10 |
| O roteador | `roteamento-gateway` | entender | 25 | 0 | ~10 |
| O roteador | `openwrt-luci` | entender | 20 | 116 | ~11 |
| O roteador | `portas-fisicas-logicas` | entender | 20 | 89 | ~11 |
| Wi-Fi | `radio-canais` | entender | 25 | 376 | ~13 |
| Wi-Fi | `ssid-seguranca` | entender | 30 | 462 | ~14 |
| Wi-Fi | `sinal` | entender | 20 | 203 | ~12 |
| Redes separadas e firewall | `bridge-interface-dhcp` | entender | 35 | 814 | ~17 |
| Redes separadas e firewall | `zonas` | entender | 25 | 292 | ~12 |
| Redes separadas e firewall | `estado-reject` | entender | 20 | 164 | ~11 |
| Redes separadas e firewall | `excecoes` | aplicar | 25 | 229 | ~12 |
| Redes separadas e firewall | `vida-de-um-pacote` | analisar | 25 | 122 | ~11 |
| Internet | `modem-bridge` | entender | 20 | 198 | ~12 |
| Internet | `nat-cgnat` | entender | 30 | 386 | ~13 |
| Internet | `bufferbloat-sqm` | analisar | 30 | 263 | ~12 |
| Internet | `interface-device` | analisar | 20 | 214 | ~12 |
| Internet | `ip-fixo` | aplicar | 20 | 247 | ~12 |
