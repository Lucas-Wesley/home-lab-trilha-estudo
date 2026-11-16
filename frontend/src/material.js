import { linkNoStreaming } from './videoaula.js'

// Material de apoio (fase 6, T27): o link de cada tipo e o localizador de um livro.

// hrefMaterial: web abre a página; documento abre dentro da trilha; livro e Acervo passam pelo
// servidor, que acha o ponto do livro e redireciona para a Biblioteca ou o Acervo.
export function hrefMaterial(m, trilhaId) {
  if (m.tipo === 'web') return m.referencia
  if (m.tipo === 'streaming') return linkNoStreaming(m.referencia)
  if (m.tipo === 'documento') return `#/trilhas/${trilhaId}/documentos/${encodeURIComponent(m.referencia)}${m.localizador ? '#' + m.localizador : ''}`
  if ((m.tipo === 'livro' || m.tipo === 'acervo') && m.referencia) {
    const q = new URLSearchParams({ tipo: m.tipo, referencia: m.referencia })
    if (m.localizador) q.set('localizador', m.localizador)
    return `/api/material/abrir?${q}`
  }
  return ''
}

// abreEmOutraAba: o que sai da Trilha (web, Biblioteca, Acervo).
export const abreEmOutraAba = (m) => m.tipo !== 'documento'

// localizadorDeCapitulos: [1, 2, 3, 5] → "caps. 1–3, 5"; [4] → "cap. 4"; [] → "".
export function localizadorDeCapitulos(lista) {
  const ns = [...new Set(lista)].sort((a, b) => a - b)
  if (!ns.length) return ''
  const faixas = []
  for (const n of ns) {
    const ultima = faixas.at(-1)
    if (ultima && n === ultima[1] + 1) ultima[1] = n
    else faixas.push([n, n])
  }
  const texto = faixas.map(([a, b]) => (a === b ? `${a}` : b === a + 1 ? `${a}, ${b}` : `${a}–${b}`)).join(', ')
  return `${ns.length === 1 ? 'cap.' : 'caps.'} ${texto}`
}

// localizadorDePaginas: (45, 60) → "p. 45–60"; (12, 12) → "p. 12".
export function localizadorDePaginas(de, ate) {
  if (!de) return ''
  return !ate || ate === de ? `p. ${de}` : `p. ${de}–${ate}`
}
