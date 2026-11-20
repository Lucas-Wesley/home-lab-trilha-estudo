// cursoParaTrilha: monta o pacote da trilha "igual ao curso" do Telegram (T42) a partir da pasta da
// árvore de Meus arquivos (pastas.js) — sem IA, sem lição escrita e sem baixar nada (o download de
// cada aula é por clique na etapa, quando ela entra). Dois modos:
//   por_aula  — cada aula é uma etapa; cada pasta com aulas é um módulo (o nome é o caminho a
//               partir da pasta escolhida, com " › ").
//   por_pasta — cada pasta com aulas é uma etapa (as aulas entram como partes em ordem); o módulo
//               é a pasta mãe dela (filha direta da escolhida → a própria escolhida).
import { slugDoArquivo } from './slug.js'

export const MODOS = {
  por_aula: 'Uma etapa por aula',
  por_pasta: 'Uma etapa por pasta',
}

// As pastas com aulas: a escolhida e as descendentes, na ordem da árvore.
function* pastasComAulas(pasta) {
  if (pasta.aulas?.length) yield pasta
  for (const p of pasta.pastas || []) yield* pastasComAulas(p)
}

// O caminho dela a partir da escolhida: [] = a própria escolhida.
const caminhoRel = (pasta, raiz) => (pasta.caminho || []).slice((raiz.caminho || []).length)

const minutosDasAulas = (aulas) => Math.ceil(aulas.reduce((s, a) => s + (a.duracao || 0), 0) / 60)

// A referência da aula do Telegram é a mensagem (estável quando baixada, apagada e baixada de
// novo); uma aula que só existe no disco aponta direto para o vídeo.
function materialDaAula(a, linkStreaming) {
  const referencia = a.link
    ? `telegram:${a.link}#${a.mensagem}`
    : `${linkStreaming}/api/video/proprio/${a.video?.id}.mp4`
  return { tipo: 'streaming', papel: 'principal', titulo: a.nome, referencia }
}

// pacoteDoCurso(pasta, modo, titulo) → o pacote de POST /api/importar (estado "ativa").
export function pacoteDoCurso(pasta, modo, titulo, linkStreaming = '') {
  const chaves = new Set()
  const chaveUnica = (base) => {
    const raiz = slugDoArquivo(base) || 'aula'
    let chave = raiz
    for (let i = 2; chaves.has(chave); i++) chave = `${raiz}-${i}`
    chaves.add(chave)
    return chave
  }

  const tituloTrilha = (titulo || '').trim() || pasta.nome
  const modulos = []

  if (modo === 'por_pasta') {
    const porTitulo = new Map()
    for (const p of pastasComAulas(pasta)) {
      const rel = caminhoRel(p, pasta)
      const tituloModulo = rel.length > 1 ? rel.slice(0, -1).join(' › ') : pasta.nome
      if (!porTitulo.has(tituloModulo)) porTitulo.set(tituloModulo, { titulo: tituloModulo, etapas: [] })
      porTitulo.get(tituloModulo).etapas.push({
        chave: chaveUnica((rel.length ? rel : [p.nome]).join('-')),
        titulo: p.nome,
        objetivo: `Assistir às aulas de ${p.nome}`,
        tipo: 'normal',
        minutos: minutosDasAulas(p.aulas),
        materiais: p.aulas.map((a) => materialDaAula(a, linkStreaming)),
      })
    }
    modulos.push(...porTitulo.values())
  } else {
    for (const p of pastasComAulas(pasta)) {
      const rel = caminhoRel(p, pasta)
      modulos.push({
        titulo: rel.length ? rel.join(' › ') : pasta.nome,
        etapas: p.aulas.map((a) => ({
          chave: chaveUnica([...rel, a.nome].join('-')),
          titulo: a.nome,
          objetivo: `Assistir à aula: ${a.nome}`,
          tipo: 'normal',
          minutos: minutosDasAulas([a]),
          materiais: [materialDaAula(a, linkStreaming)],
        })),
      })
    }
  }

  let grupo = pasta.caminho?.[0] || pasta.nome
  for (const p of pastasComAulas(pasta)) for (const a of p.aulas) if (a.grupo) { grupo = a.grupo; break }
  const descricao = `Curso do Telegram ${grupo}` +
    ((pasta.caminho?.length || 0) > 1 ? `, pasta ${pasta.caminho.slice(1).join(' › ')}` : '') + '.'

  return {
    trilha: {
      titulo: tituloTrilha,
      objetivo: `Assistir às aulas de ${tituloTrilha}`,
      descricao,
      estado: 'ativa',
      modulos,
    },
  }
}

// resumoDoPacote: "3 módulos · 9 etapas · 2 h" para as opções da caixa "Criar trilha".
export function resumoDoPacote(pacote) {
  const modulos = pacote.trilha.modulos.length
  let etapas = 0
  let minutos = 0
  for (const m of pacote.trilha.modulos) {
    etapas += m.etapas.length
    for (const e of m.etapas) minutos += e.minutos || 0
  }
  const tempo = minutos >= 60 ? `${Math.round(minutos / 60)} h` : `${minutos} min`
  const pl = (n, s, p) => (n === 1 ? `1 ${s}` : `${n} ${p}`)
  return { modulos, etapas, minutos, tempo, texto: `${pl(modulos, 'módulo', 'módulos')} · ${pl(etapas, 'etapa', 'etapas')} · ${tempo}` }
}

// modoRecomendado: média das aulas abaixo de 15 min → uma etapa por pasta rende mais.
export function modoRecomendado(pasta) {
  let n = 0
  let segundos = 0
  for (const p of pastasComAulas(pasta)) {
    for (const a of p.aulas) {
      n++
      segundos += a.duracao || 0
    }
  }
  return n && segundos / n < 15 * 60 ? 'por_pasta' : 'por_aula'
}
