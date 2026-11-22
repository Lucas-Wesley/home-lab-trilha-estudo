// Cursos de plataformas (T50, T55): as aulas baixadas das plataformas, agrupadas para a aba "Cursos"
// da Entrada em plataforma › curso › módulo, na ordem do curso (a que a extensão mandou; sem ela, a
// ordem em que foram pedidas). Cada curso diz a trilha dele, se já virou uma.

export const ESTADOS_AULA = {
  na_fila: { nome: 'na fila', cor: '' },
  baixando: { nome: 'baixando…', cor: 'azul' },
  pronta: { nome: 'baixada', cor: 'verde' },
  erro: { nome: 'erro', cor: 'vermelho' },
  texto: { nome: 'só texto', cor: 'cinza' },
}

export function agruparAulas(aulas, plataformas = {}, trilhas = []) {
  const grupos = []
  const achar = (lista, chave, novo) => {
    let g = lista.find((x) => x.chave === chave)
    if (!g) lista.push((g = novo()))
    return g
  }
  const ordem = (a) => (a.ordem > 0 ? a.ordem : Number.MAX_SAFE_INTEGER)
  for (const a of [...aulas].sort((x, y) => ordem(x) - ordem(y) || x.id - y.id)) {
    const p = achar(grupos, a.plataforma, () => ({ chave: a.plataforma, nome: plataformas[a.plataforma] || a.plataforma, cursos: [] }))
    const c = achar(p.cursos, a.curso, () => ({
      chave: a.curso,
      nome: a.curso,
      plataforma: a.plataforma,
      trilha: trilhas.find((t) => t.plataforma === a.plataforma && t.curso === a.curso)?.trilha_id || 0,
      modulos: [],
      tamanho: 0,
      baixadas: 0,
      soTexto: 0,
      total: 0,
    }))
    const m = achar(c.modulos, a.modulo, () => ({ chave: a.modulo, nome: a.modulo, aulas: [] }))
    m.aulas.push(a)
    c.total++
    if (a.estado === 'texto') c.soTexto++
    if (a.estado === 'pronta') {
      c.baixadas++
      c.tamanho += a.tamanho
    }
  }
  return grupos
}

// Há download andando: a tela pergunta de novo a cada poucos segundos.
export const temDownload = (aulas) => aulas.some((a) => a.estado === 'na_fila' || a.estado === 'baixando')
