// Onde as aulas do Telegram já estão (Meus arquivos): GET /api/arquivos/usos devolve cada
// material telegram: com a trilha e a etapa, e os encaixes de cursos esperando aprovação. Aqui
// isso vira o que a tela mostra na aula ("na trilha X · etapa") e na pasta ("12 de 129 aulas na
// trilha X", "virou a trilha X", "encaixe esperando aprovação em Y").
import { aulasDaPasta, linkDoGrupo, referenciaDaAula } from './cursos.js'

// usosPorAula: a lista do servidor → Map ref → [{trilha_id, trilha, etapa_chave, etapa}].
export function usosPorAula(usos) {
  const mapa = new Map()
  for (const u of usos?.aulas || []) {
    if (!mapa.has(u.ref)) mapa.set(u.ref, [])
    mapa.get(u.ref).push(u)
  }
  return mapa
}

// trilhasDaPasta: quantas aulas da pasta (e subpastas) estão em cada trilha, da que tem mais para
// a que tem menos. `inteira` = todas as aulas do Telegram da pasta estão nessa trilha.
export function trilhasDaPasta(pasta, mapa) {
  const aulas = aulasDaPasta(pasta).map(referenciaDaAula).filter(Boolean)
  const porTrilha = new Map()
  for (const ref of aulas) {
    const vistas = new Set()
    for (const u of mapa.get(ref) || []) {
      if (vistas.has(u.trilha_id)) continue
      vistas.add(u.trilha_id)
      const t = porTrilha.get(u.trilha_id) || { trilha_id: u.trilha_id, trilha: u.trilha, aulas: 0 }
      t.aulas++
      porTrilha.set(u.trilha_id, t)
    }
  }
  return [...porTrilha.values()]
    .map((t) => ({ ...t, inteira: aulas.length > 0 && t.aulas === aulas.length }))
    .sort((a, b) => b.aulas - a.aulas || a.trilha.localeCompare(b.trilha))
}

// encaixesDaPasta: encaixes pendentes que pegam esta pasta — a própria, uma pasta de dentro dela
// ou a pasta que a contém (o encaixe de "MBA" pega o módulo "009 - DDD").
export function encaixesDaPasta(pasta, encaixes) {
  const link = linkDoGrupo(pasta)
  if (!link) return []
  const minha = `telegram:${link}::${pasta.caminho.join('/')}`
  const pega = (outra) => outra === minha || outra.startsWith(minha + '/') || minha.startsWith(outra + '/')
  const vistas = new Set()
  return (encaixes || []).filter((e) => {
    if (!e.pastas?.some(pega) || vistas.has(e.trilha_id)) return false
    vistas.add(e.trilha_id)
    return true
  })
}
