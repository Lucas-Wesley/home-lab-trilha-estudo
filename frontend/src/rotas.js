// Rotas no hash (#/trilhas/aws), como nas outras apps: o "voltar" do celular volta para a tela
// anterior e o servidor só precisa servir o index.html.
// "design" (a vitrine do design system) não tem aba: só se chega pelo endereço. "ajustes" abre
// pelo ícone no topo.
export const SECOES = ['hoje', 'trilhas', 'entrada', 'projeto', 'ajustes', 'design']

// Um "%" malformado no endereço (link copiado pela metade, outra codificação) não pode derrubar o
// app: o trecho fica como veio.
function decodificar(texto) {
  try {
    return decodeURIComponent(texto)
  } catch {
    return texto
  }
}

export function lerRota(hash) {
  const caminho = decodificar(String(hash || '').replace(/^#\/?/, ''))
  const [caminhoSemAncora, ancora = ''] = caminho.split('#')
  const partes = caminhoSemAncora.split('/').filter(Boolean)
  const secao = SECOES.includes(partes[0]) ? partes[0] : 'hoje'
  return { secao, partes: SECOES.includes(partes[0]) ? partes.slice(1) : [], ancora }
}

export function link(secao, ...partes) {
  return '#/' + [secao, ...partes].map(encodeURIComponent).join('/')
}

// linkPasta: "Meus arquivos" (T41) — o caminho "Grupo/Curso" vira
// "#/entrada/arquivos/Grupo/Curso", um nível por parte, cada um codificado.
export function linkPasta(caminho) {
  const niveis = String(caminho || '').split('/').filter(Boolean)
  return link('entrada', 'arquivos', ...niveis)
}
