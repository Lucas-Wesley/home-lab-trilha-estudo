// Cursos de plataformas (T50): as aulas baixadas da Branas.io e da LINUXtips, agrupadas para a aba
// "Cursos" da Entrada em plataforma › curso › módulo, na ordem em que foram pedidas.

export const ESTADOS_AULA = {
  na_fila: { nome: 'na fila', cor: '' },
  baixando: { nome: 'baixando…', cor: 'azul' },
  pronta: { nome: 'baixada', cor: 'verde' },
  erro: { nome: 'erro', cor: 'vermelho' },
}

export function agruparAulas(aulas, plataformas = {}) {
  const grupos = []
  const achar = (lista, chave, novo) => {
    let g = lista.find((x) => x.chave === chave)
    if (!g) lista.push((g = novo()))
    return g
  }
  for (const a of [...aulas].sort((x, y) => x.id - y.id)) {
    const p = achar(grupos, a.plataforma, () => ({ chave: a.plataforma, nome: plataformas[a.plataforma] || a.plataforma, cursos: [] }))
    const c = achar(p.cursos, a.curso, () => ({ chave: a.curso, nome: a.curso, modulos: [], tamanho: 0, baixadas: 0, total: 0 }))
    const m = achar(c.modulos, a.modulo, () => ({ chave: a.modulo, nome: a.modulo, aulas: [] }))
    m.aulas.push(a)
    c.total++
    if (a.estado === 'pronta') {
      c.baixadas++
      c.tamanho += a.tamanho
    }
  }
  return grupos
}

// Há download andando: a tela pergunta de novo a cada poucos segundos.
export const temDownload = (aulas) => aulas.some((a) => a.estado === 'na_fila' || a.estado === 'baixando')
