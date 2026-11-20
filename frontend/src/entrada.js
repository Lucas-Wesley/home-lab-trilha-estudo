import { tempoParaSegundos } from './videoaula.js'

// O link do YouTube dentro de um texto: o compartilhar do Android manda "Título https://youtu.be/…"
// e a área de transferência pode ter "youtu.be/…" ou "youtube.com/…" sem o https:// — o servidor só
// aceita o link com o esquema, então ele é acrescentado aqui. Sem link, devolve "".
export function linkDoYouTube(texto) {
  const m = String(texto || '').match(
    /(?:https?:\/\/)?(?:[\w-]+\.)?(?:youtu\.be\/[\w-]+(?:\?[^\s<>"']*)?|youtube\.com\/(?:watch|shorts|live|embed|playlist)[^\s<>"']*)/i,
  )
  if (!m) return ''
  const limpo = m[0].replace(/[.,;:!?)\]]+$/, '')
  return limpo.startsWith('http') ? limpo : `https://${limpo}`
}

// Do endereço do compartilhar (/compartilhar?title=…&text=…&url=…), o link do YouTube que veio.
export function linkCompartilhado(busca) {
  const q = new URLSearchParams(busca)
  return linkDoYouTube([q.get('url'), q.get('text'), q.get('title')].filter(Boolean).join(' '))
}

// O link do ponto do vídeo no YouTube: usado nos capítulos e nos [mm:ss] da transcrição.
export function linkNoMinuto(videoId, segundos) {
  return `https://youtu.be/${videoId}?t=${Math.floor(segundos || 0)}`
}

// As linhas da transcrição: as que começam com [mm:ss] ou [h:mm:ss] ganham marca e o segundo do vídeo.
export function linhasDaTranscricao(texto) {
  return String(texto || '')
    .split('\n')
    .map((linha) => {
      const m = /^\[((?:\d+:)?\d{1,2}:\d{2})\]\s?(.*)$/.exec(linha.trimStart())
      if (m) return { marca: m[1], segundos: tempoParaSegundos(m[1]), texto: m[2] }
      return { marca: '', segundos: null, texto: linha }
    })
}

// O fragmento "sem destino" some 30 dias depois da marca (D6) — quantos dias ainda faltam.
export const DIAS_SEM_DESTINO = 30
export function diasSemDestino(em, agora = Date.now()) {
  const faltam = new Date(em).getTime() + DIAS_SEM_DESTINO * 86400000 - agora
  return Math.max(0, Math.ceil(faltam / 86400000))
}
