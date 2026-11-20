// Trecho do YouTube na etapa (T39): o trecho principal É a aula — começa no início, a barra
// mostra o trecho e, no fim, avisa sem parar o vídeo. Funções puras (testadas em
// trechoYoutube.test.js); o player e as notas ficam em TrechoYouTube.vue.
import { segundosParaTempo, tempoParaSegundos } from './videoaula.js'

const RE_VIDEO = [
  /[?&]v=([\w-]{6,})/, // watch?v= (e m.youtube.com)
  /youtu\.be\/([\w-]{6,})/, // youtu.be/
  /youtube\.com\/(?:shorts|live|embed)\/([\w-]{6,})/, // shorts, live, embed
]

// idDoVideo: o id do vídeo num link do YouTube (watch, youtu.be, shorts, live, embed); '' se não acha.
export function idDoVideo(url) {
  for (const re of RE_VIDEO) {
    const m = re.exec(url || '')
    if (m) return m[1]
  }
  return ''
}

// intervalos: "13:33–18:25; 25:10–31:00" (travessão, travessão longo ou hífen) → [[813, 1105], …].
export function intervalos(localizador) {
  const lista = []
  for (const parte of String(localizador || '').split(';')) {
    const lados = parte.trim().split(/[-–—]/)
    if (lados.length !== 2) continue
    const inicio = tempoParaSegundos(lados[0].trim())
    const fim = tempoParaSegundos(lados[1].trim())
    if (inicio === null || fim === null || fim <= inicio) continue
    lista.push([inicio, fim])
  }
  return lista
}

// escreverIntervalo: [813, 1105] → "13:33–18:25" (o formato canônico, com travessão).
export function escreverIntervalo(inicio, fim) {
  return segundosParaTempo(inicio) + '–' + segundosParaTempo(fim)
}

// trechos: cada intervalo de um material `youtube` principal vira uma parte da aula, já com
// início/fim em segundos, o id do vídeo e o localizador canônico (a chave do "visto").
export function trechos(materiais) {
  const lista = []
  for (const m of materiais || []) {
    if (m.tipo !== 'youtube' || m.papel !== 'principal' || !m.referencia) continue
    for (const [inicio, fim] of intervalos(m.localizador)) {
      lista.push({
        videoId: idDoVideo(m.referencia),
        referencia: m.referencia,
        localizador: escreverIntervalo(inicio, fim),
        inicio,
        fim,
        titulo: m.titulo,
      })
    }
  }
  return lista
}

// intervaloDoMaterial: o trecho de um material streaming do Telegram (T44) — a ref é a
// aula estável e o localizador é "mm:ss–mm:ss". Sem intervalo, é uma videoaula comum.
export function intervaloDoMaterial(m) {
  if (!m || !/^telegram:https:\/\/t\.me\/.+#\d+$/.test(m.referencia || '')) return null
  const [iv] = intervalos(m.localizador)
  return iv ? { inicio: iv[0], fim: iv[1], localizador: escreverIntervalo(iv[0], iv[1]), referencia: m.referencia } : null
}

// pctDoTrecho: a posição do vídeo como porcentagem da janela do trecho (0–100).
export function pctDoTrecho(posicao, inicio, fim) {
  if (!fim || fim <= inicio) return 0
  return Math.min(100, Math.max(0, ((posicao - inicio) / (fim - inicio)) * 100))
}

// estadoDoTrecho: onde o vídeo está em relação ao trecho ("passou" aos fim − 2 s, a régua do
// aviso; "faltam" o que falta para o fim, sem negativo).
export function estadoDoTrecho(posicao, inicio, fim) {
  const p = Number(posicao) || 0
  return {
    antes: p < inicio,
    dentro: p >= inicio && p < fim,
    passou: p >= fim - 2,
    faltam: Math.max(0, Math.ceil(fim - p)),
  }
}

// linkNoMinuto: abre o YouTube no segundo certo (https://youtu.be/<id>?t=<segundos>).
export function linkNoMinuto(id, segundos) {
  return `https://youtu.be/${id}?t=${Math.max(0, Math.floor(segundos || 0))}`
}

// chaveDoTrecho: como um trecho se identifica no "visto" (referencia + localizador canônico).
export function chaveDoTrecho(t) {
  return `${t.referencia}|${t.localizador}`
}

// segundosDosTrechos: a duração somada dos trechos principais (a "duração" na estrutura).
export function segundosDosTrechos(materiais) {
  return trechos(materiais).reduce((s, t) => s + (t.fim - t.inicio), 0)
}
