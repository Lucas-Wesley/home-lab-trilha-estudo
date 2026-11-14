// Rotas no hash (#/trilhas/aws), como nas outras apps: o "voltar" do celular volta para a tela
// anterior e o servidor só precisa servir o index.html.
export const SECOES = ['hoje', 'trilhas', 'documentos', 'design']

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
