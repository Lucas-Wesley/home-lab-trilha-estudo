// Verificar material (T53): o que a tela da etapa calcula sobre a verificação.

const chave = (m) => `${m.tipo}|${m.referencia}|${m.localizador}`

// mudouDesde: um material verificado não é mais o que está na etapa (tirou, trocou ou mudou o trecho).
export function mudouDesde(verificacao, materiais) {
  if (!verificacao) return false
  const atuais = materiais || []
  return (verificacao.materiais || []).some((m) => !atuais[m.indice] || chave(atuais[m.indice]) !== chave(m))
}

// naoVerificados: os materiais da etapa que esta verificação não leu (novos, ou de fora do pedido).
export function naoVerificados(verificacao, materiais) {
  if (!verificacao) return []
  const lidos = new Set((verificacao.materiais || []).map(chave))
  return (materiais || []).filter((m) => !lidos.has(chave(m)))
}

// pedidoDasFaltas: o pedido do "Sugerir material" para complementar o que a verificação achou.
export function pedidoDasFaltas(laudo) {
  const faltas = (laudo?.faltas || []).filter((f) => f.ponto)
  const rasos = (laudo?.topicos || []).filter((t) => t.situacao !== 'coberto' && !faltas.some((f) => f.ponto === t.topico))
  if (!faltas.length && !rasos.length) return ''
  const linhas = ['Complementar o material atual da etapa, que não cobre:']
  for (const f of faltas) linhas.push(`- ${f.ponto}${f.onde_achar ? ` (fonte sugerida: ${f.onde_achar})` : ''}`)
  for (const t of rasos) linhas.push(`- ${t.topico} (${t.situacao === 'raso' ? 'está raso' : 'ausente'})`)
  linhas.push('Prefira fontes primárias (documentação oficial, RFC, paper, livro de referência).')
  return linhas.join('\n')
}

// emAndamento: a verificação ainda espera a transcrição ou a IA.
export const emAndamento = (v) => v?.estado === 'aguardando' || v?.estado === 'verificando'

// fase: o que dizer enquanto espera.
export function fase(v) {
  if (v?.estado === 'aguardando') {
    const n = (v.materiais || []).filter((m) => m.texto === 'transcrevendo').length
    return `Transcrevendo ${n === 1 ? 'o vídeo' : `${n} vídeos`} na Entrada; a verificação segue sozinha quando terminar.`
  }
  if (v?.estado === 'verificando') return v.tarefa?.estado === 'rodando' ? 'A IA está lendo o material…' : 'Na fila da IA…'
  return ''
}
