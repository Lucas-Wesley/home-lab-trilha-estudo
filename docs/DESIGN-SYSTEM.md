# Design system

A fonte do design system das apps do homelab agora é o **Hub**:
[`hub-home-lab-web/docs/DESIGN-SYSTEM.md`](https://github.com/Lucas-Wesley/hub-home-lab-web/blob/main/docs/DESIGN-SYSTEM.md),
com a vitrine viva em `hub.internal/#/design` e as referências (prints da Claude Platform) em
`hub-home-lab-web/docs/design/referencias/`. Decisão do Lucas em 18/11/2026.

O design nasceu aqui (T14) e a Trilha já copiou a quarta versão do Hub: sem barra no topo (a
marca abre a lateral e o tema fica no pé dela; no celular, a barra do pé abre a gaveta),
cabeçalho da página numa linha (`.topo-pagina`), conteúdo largo até 1200 px com painel lateral
(`.duas-colunas`), ferramentas na linha do conteúdo (`.ferramentas`), números primeiro
(`.numeros-quadro`) e conteúdo agrupado em quadros com rótulo (`.quadro`, `.quadro-topo`,
`.rotulo-secao`). Mudança de design é lá primeiro, depois se copia o `frontend/src/style.css`
daqui dele.

Peças que só a Trilha tem (abas da página em `AbasPagina.vue`, quadro de situação da etapa,
comentários de proposta, navegação de contexto da trilha no `#nav-contexto` da lateral, tokens
`--video`) continuam descritas na vitrine `#/design` daqui até virem para o Hub.
