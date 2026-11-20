// Videoaula (T30): o vídeo principal da etapa, as notas com o minuto e o resumo.

// videosDaEtapa: os materiais `streaming` principais, na ordem da etapa (o curso em vídeo).
export function videosDaEtapa(materiais) {
  return (materiais || []).filter((m) => m.tipo === 'streaming' && m.papel === 'principal' && m.referencia)
}

// linkNoStreaming: o arquivo (…/api/video/proprio/57.mp4) vira o player do Streaming (…/#/assistir/proprio:57).
export function linkNoStreaming(referencia) {
  const m = /^(https?:\/\/[^/]+)\/api\/video\/(\w+)\/(\d+)\.\w+$/.exec(referencia || '')
  return m ? `${m[1]}/#/assistir/${m[2]}:${m[3]}` : referencia || ''
}

// ehAulaDoTelegram: referencia "telegram:<link do grupo>#<mensagem>" (T42) — a aula do curso do
// Telegram; estável quando baixada, apagada ou baixada de novo (o id no Streaming muda; a
// mensagem, não).
export const ehAulaDoTelegram = (ref) => /^telegram:https:\/\/t\.me\/.+#\d+$/.test(ref || '')

// tempoEstimado: quanto o download da aula leva, a ~4 MB/s (o ritmo medido do Telegram no
// Streaming, S56 de lá) — "~1 min", "~7 min".
export function tempoEstimado(bytes) {
  const min = Math.max(1, Math.ceil((bytes || 0) / (4 * 1024 * 1024) / 60))
  return `~${min} min`
}

// segundosParaTempo: 75 → "01:15"; 3725 → "1:02:05".
export function segundosParaTempo(s) {
  s = Math.max(0, Math.floor(s || 0))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = String(s % 60).padStart(2, '0')
  return h ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${String(m).padStart(2, '0')}:${ss}`
}

// tempoParaSegundos: "01:15" → 75; "1:02:05" → 3725; o resto → null.
export function tempoParaSegundos(t) {
  const m = /^(?:(\d+):)?(\d{1,2}):(\d{2})$/.exec(String(t || '').trim())
  if (!m) return null
  return Number(m[1] || 0) * 3600 + Number(m[2]) * 60 + Number(m[3])
}

// marcaDeTempo: o que "Marcar o minuto" escreve nas notas. Com mais de um vídeo, diz a parte.
export function marcaDeTempo(segundos, parte, partes) {
  const t = segundosParaTempo(segundos)
  return partes > 1 ? `[Parte ${parte} · ${t}]` : `[${t}]`
}

const reMarca = /\[(?:Parte (\d+) · )?((?:\d+:)?\d{1,2}:\d{2})\]\s*([^\n[]*)/g

// momentosDasNotas: cada [mm:ss] (ou [Parte N · mm:ss]) das notas, com o texto que vem depois.
export function momentosDasNotas(texto) {
  const lista = []
  for (const m of String(texto || '').matchAll(reMarca)) {
    lista.push({ parte: Number(m[1] || 1), tempo: m[2], segundos: tempoParaSegundos(m[2]), texto: m[3].trim() })
  }
  return lista
}

export const VEREDITOS = {
  bate: { nome: 'Bate com a aula', cor: 'verde' },
  parcial: { nome: 'Bate em parte', cor: 'ambar' },
  nao_bate: { nome: 'Não bate', cor: 'vermelho' },
}
