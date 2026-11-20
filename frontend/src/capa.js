// A capa de cada trilha (T47), gerada do título: uma sigla curta, a cor (matiz) e o desenho do
// fundo. Sem imagem guardada: o mesmo título dá sempre a mesma capa.

// Assuntos conhecidos: a palavra no título (sem acento, minúscula) → sigla e cor.
const ASSUNTOS = [
  [/\bddd\b|domain.driven/, 'DDD', 265],
  [/cloud practitioner|clf-c0/, 'CLF', 30],
  [/\baws\b/, 'AWS', 32],
  [/\bgolang\b|\bgo\b/, 'Go', 192],
  [/\blinux\b/, '$_', 140],
  [/home ?lab/, 'Lab', 170],
  [/\bredes?\b/, 'Net', 210],
  [/algoritmo|estruturas de dados/, 'O(n)', 330],
  [/system design/, 'SD', 15],
  [/ingles|english/, 'EN', 355],
  [/gramatica|portugues/, 'Aa', 48],
  [/conhecimento de base|fundamentos/, '01', 95],
  [/banco de dados|\bsql\b/, 'DB', 225],
  [/docker|kubernetes/, 'K8s', 200],
  [/observabilidade/, 'Obs', 285],
  [/arquitetura/, 'Arq', 250],
  [/\btdd\b|testes?/, 'TDD', 120],
  [/solid|design patterns/, 'OO', 300],
]

const PALAVRAS_VAZIAS = new Set(['de', 'da', 'do', 'das', 'dos', 'e', 'para', 'a', 'o', 'em', 'na', 'no', 'com', 'ao', 'zero'])

const semAcento = (s) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

function hash(s) {
  let h = 0
  for (const c of s) h = (h * 31 + c.codePointAt(0)) >>> 0
  return h
}

export const DESENHOS = ['pontos', 'arcos', 'diagonais', 'barras']

export function capaDaTrilha(titulo = '') {
  const t = semAcento(titulo)
  const h = hash(t)
  const desenho = DESENHOS[h % DESENHOS.length]
  for (const [regra, sigla, matiz] of ASSUNTOS) {
    if (regra.test(t)) return { sigla, matiz, desenho }
  }
  // Fora dos conhecidos: as iniciais das duas primeiras palavras que dizem algo.
  const palavras = titulo.split(/[\s:()–-]+/).filter((p) => p && !PALAVRAS_VAZIAS.has(semAcento(p)))
  const sigla = palavras.slice(0, 2).map((p) => p[0].toUpperCase()).join('') || '·'
  return { sigla, matiz: h % 360, desenho }
}
