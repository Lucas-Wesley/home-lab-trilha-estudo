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
