// Cursos do Telegram na trilha (T43): um material "curso" é uma ou mais pastas de Meus arquivos.
// A referência é "telegram:<link do grupo>::<caminho na árvore>", várias separadas por " | ".
// A IA encaixa só pelos títulos das aulas, numeradas A1, A2… no sumário — sem baixar nada.
import { ref } from 'vue'

// As pastas marcadas ("caminho" → true) ficam no módulo: a tela remonta a cada troca de pasta
// e a seleção tem que sobreviver — marcar cursos de grupos diferentes exige navegar entre eles.
export const selecaoCursos = ref(new Set())

// linkDoGrupo: o link do Telegram das aulas da pasta (desce na árvore até achar uma; pasta só de
// vídeos do disco não é curso do Telegram).
export function linkDoGrupo(pasta) {
  for (const a of pasta?.aulas || []) {
    if (a.link) return a.link
  }
  for (const f of pasta?.pastas || []) {
    const l = linkDoGrupo(f)
    if (l) return l
  }
  return ''
}

// escreverCursos: [{link, caminho}] → "telegram:<link>::<caminho> | telegram:<link2>::<caminho2>".
export function escreverCursos(lista) {
  return lista.map((c) => `telegram:${c.link}::${c.caminho}`).join(' | ')
}

// referenciaDoCurso: as pastas marcadas → a referência do material "curso". null se alguma pasta
// não é de um grupo do Telegram.
export function referenciaDoCurso(pastas) {
  const lista = []
  for (const p of pastas) {
    const link = linkDoGrupo(p)
    if (!link) return null
    lista.push({ link, caminho: p.caminho.join('/') })
  }
  return escreverCursos(lista)
}

// tituloDoCurso: "DDD do Jeito Certo + 009 - DDD".
export function tituloDoCurso(pastas) {
  return pastas.map((p) => p.nome).join(' + ')
}

// aulasDoLocalizador: "A12", "A12–A15", "A3, A7" → [12,13,14,15,3,7] na ordem, sem repetir
// (a mesma leitura do backend: números A<n>, faixas com hífen ou travessão, listas com vírgula).
export function aulasDoLocalizador(localizador) {
  const nums = []
  const visto = new Set()
  const add = (n) => {
    if (!visto.has(n)) {
      visto.add(n)
      nums.push(n)
    }
  }
  for (const tok of String(localizador || '').split(/[\s,;.]+/)) {
    const up = tok.toUpperCase()
    if (!up.startsWith('A')) continue // todo número de aula vem com o prefixo A
    const faixa = /^A(\d+)[-–]A?(\d+)$/.exec(up)
    if (faixa) {
      const de = +faixa[1]
      const ate = +faixa[2]
      if (de > 0 && ate >= de) {
        for (let n = de; n <= ate; n++) add(n)
        continue
      }
    }
    const um = /^A(\d+)$/.exec(up)
    if (um) add(+um[1])
  }
  return nums
}

// nomesDasAulas: "A1–A2" + o mapa de nomes da proposta → "Aula 1 · Aula 2" (os números sem nome
// conhecido aparecem como "A<n>").
export function nomesDasAulas(localizador, nomes = {}) {
  return aulasDoLocalizador(localizador)
    .map((n) => nomes[`A${n}`] || `A${n}`)
    .join(' · ')
}

// precisamOuvir: a linha "Precisam ser ouvidas:" das notas → os números listados nela.
export function precisamOuvir(notas) {
  for (const linha of String(notas || '').split('\n')) {
    const m = /^\s*(?:[-*]\s*)?Precisam ser ouvidas?:\s*(.+)$/i.exec(linha)
    if (m) return aulasDoLocalizador(m[1])
  }
  return []
}

// ---------- processar (T44): a aula do Telegram vira fonte da Entrada ----------

// aulasDaPasta: todas as aulas da pasta e das subpastas, na ordem da árvore.
export function aulasDaPasta(pasta) {
  const lista = [...(pasta?.aulas || [])]
  for (const sub of pasta?.pastas || []) lista.push(...aulasDaPasta(sub))
  return lista
}

// referenciaDaAula: a ref estável "telegram:<link do grupo>#<mensagem>" — só aulas do
// catálogo têm (aula só do disco, sem link, não dá para processar).
export function referenciaDaAula(a) {
  return a?.link && a?.mensagem ? `telegram:${a.link}#${a.mensagem}` : ''
}
