import { link } from './rotas.js'

// Projeto de estudo (T52): o que a tela usa das respostas de /api/projeto.

// Situação do item no roadmap.
export const SITUACOES = {
  a_fazer: { nome: 'A fazer', cor: '' },
  em_andamento: { nome: 'Em andamento', cor: 'azul' },
  feito: { nome: 'Feito', cor: 'verde' },
}

// rotuloSituacao: "Em andamento · caminhos" quando o caderno diz a fase do ciclo.
export function rotuloSituacao(item) {
  const s = SITUACOES[item?.situacao] || SITUACOES.a_fazer
  const fase = item?.situacao === 'em_andamento' && item.estado ? ` · ${item.estado}` : ''
  return { nome: s.nome + fase, cor: s.cor }
}

// linkReferencia: a etapa abre a etapa; módulo e trilha abrem a trilha. Sem par, sem link.
export function linkReferencia(r) {
  if (!r?.tipo || !r.trilha_id) return ''
  if (r.tipo === 'etapa' && r.etapa_id) return link('trilhas', r.trilha_id, 'etapas', r.etapa_id)
  return link('trilhas', r.trilha_id)
}

// textoReferencia: o caminho real quando casou inteiro; senão o texto do roadmap.
export function textoReferencia(r) {
  if (r?.tipo && !r.parcial && r.caminho) return r.caminho
  return r?.texto || ''
}

// textoDoLink: o texto do link da referência; curto tira o "Tn ›" (a lista já agrupa por trilha).
export function textoDoLink(r, curto = false) {
  const t = textoReferencia(r)
  return curto ? semTrilha(t) || t : t
}

// progressoReferencia: "3/5" para módulo e trilha; vazio para etapa.
export function progressoReferencia(r) {
  if (!r || r.tipo === 'etapa' || !r.total) return ''
  return `${r.concluidas || 0}/${r.total}`
}

// itensPorId junta os itens de todas as fases.
export function itensPorId(dados) {
  const mapa = {}
  for (const f of dados?.fases || []) for (const it of f.itens || []) mapa[it.id] = it
  return mapa
}

export function resumoFase(fase) {
  const itens = fase?.itens || []
  return { feitos: itens.filter((i) => i.situacao === 'feito').length, total: itens.length }
}

// copiarTexto: a área de transferência; sem a API (contexto sem HTTPS), o jeito antigo.
export async function copiarTexto(texto) {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(texto)
      return true
    }
  } catch {
    // cai no jeito antigo
  }
  const campo = document.createElement('textarea')
  campo.value = texto
  campo.setAttribute('readonly', '')
  campo.style.position = 'fixed'
  campo.style.opacity = '0'
  document.body.appendChild(campo)
  campo.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  campo.remove()
  return ok
}

// semTrilha: o caminho sem o "T13 › " do começo (o nome da trilha já está no grupo).
export function semTrilha(texto) {
  return (texto || '').replace(/^T\d+\s*(?:›\s*|$)/, '').trim()
}

// numeroTrilha: 13 em "T13 › …"; 0 sem número.
function numeroTrilha(r) {
  if (r?.trilha_id) return r.trilha_id
  const m = /\bT(\d+)\b/.exec(r?.texto || '')
  return m ? Number(m[1]) : 0
}

// agruparPorTrilha: as referências agrupadas pela trilha, na ordem em que aparecem, sem repetir.
export function agruparPorTrilha(refs) {
  const grupos = new Map()
  const vistos = new Set()
  for (const r of refs) {
    const chave = `${r.tipo || ''}:${r.etapa_id || r.modulo_id || ''}:${textoReferencia(r)}`
    if (vistos.has(chave)) continue
    vistos.add(chave)
    const n = numeroTrilha(r)
    if (!grupos.has(n)) grupos.set(n, { id: n, titulo: r.trilha_titulo || (n ? `T${n}` : 'Outras fontes'), refs: [] })
    const g = grupos.get(n)
    if (!g.titulo || g.titulo === `T${n}`) g.titulo = r.trilha_titulo || g.titulo
    g.refs.push(r)
  }
  return [...grupos.values()]
}

// candidatosDoItem: o "Estudar antes" e as trilhas da 7.1 do roadmap numa lista só, por trilha.
// Os capítulos de livro ficam no quadro lateral; por isso a linha de livros da 7.1 não entra.
export function candidatosDoItem(item) {
  const refs = []
  for (const l of [...(item?.estudo || []), item?.trilhas_71].filter(Boolean)) {
    if (l.refs?.length) refs.push(...l.refs)
    else if (l.texto) refs.push({ texto: l.texto })
  }
  return agruparPorTrilha(refs)
}

// situacaoEstudo: a cor da situação de uma fonte escolhida ("estudado", "já sabia", "a estudar").
export function situacaoEstudo(texto) {
  const n = (texto || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase()
  if (n.includes('ja sabia') || n.includes('estudado') || n.includes('feito')) return 'verde'
  if (n.includes('estudando')) return 'azul'
  return ''
}
