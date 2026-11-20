import { describe, expect, it } from 'vitest'
import { ehAulaDoTelegram, linkNoStreaming, marcaDeTempo, momentosDasNotas, segundosParaTempo, tempoEstimado, tempoParaSegundos, videosDaEtapa } from './videoaula.js'

describe('videoaula', () => {
  it('pega só os vídeos principais', () => {
    const ms = [
      { tipo: 'documento', papel: 'principal', referencia: 'aula-01' },
      { tipo: 'streaming', papel: 'principal', referencia: 'https://s/api/video/proprio/3.mp4' },
      { tipo: 'streaming', papel: 'apoio', referencia: 'https://s/api/video/proprio/4.mp4' },
    ]
    expect(videosDaEtapa(ms).map((m) => m.referencia)).toEqual(['https://s/api/video/proprio/3.mp4'])
  })
  it('liga o arquivo ao player do Streaming', () => {
    expect(linkNoStreaming('https://streaming.internal/api/video/proprio/57.mp4')).toBe('https://streaming.internal/#/assistir/proprio:57')
    expect(linkNoStreaming('https://exemplo.org/v')).toBe('https://exemplo.org/v')
  })
  it('converte tempos', () => {
    expect(segundosParaTempo(75)).toBe('01:15')
    expect(segundosParaTempo(3725)).toBe('1:02:05')
    expect(tempoParaSegundos('01:15')).toBe(75)
    expect(tempoParaSegundos('1:02:05')).toBe(3725)
    expect(tempoParaSegundos('abc')).toBe(null)
  })
  it('reconhece a aula do Telegram', () => {
    expect(ehAulaDoTelegram('telegram:https://t.me/c/1#42')).toBe(true)
    expect(ehAulaDoTelegram('telegram:https://t.me/grupo#7')).toBe(true)
    expect(ehAulaDoTelegram('https://s/api/video/proprio/3.mp4')).toBe(false)
    expect(ehAulaDoTelegram('telegram:https://exemplo.org/x#1')).toBe(false)
    expect(ehAulaDoTelegram('telegram:sem-mensagem')).toBe(false)
    expect(ehAulaDoTelegram('')).toBe(false)
  })
  it('estima o tempo do download a 4 MB/s', () => {
    expect(tempoEstimado(230 * 1024 * 1024)).toBe('~1 min')
    expect(tempoEstimado(0)).toBe('~1 min')
    expect(tempoEstimado(2 * 1024 * 1024 * 1024)).toBe('~9 min')
  })
  it('acha os momentos marcados nas notas', () => {
    expect(marcaDeTempo(75, 1, 1)).toBe('[01:15]')
    expect(marcaDeTempo(75, 2, 2)).toBe('[Parte 2 · 01:15]')
    const ms = momentosDasNotas('[01:15] substantivo concreto\nnada\n[Parte 2 · 1:02:05] coletivo')
    expect(ms).toEqual([
      { parte: 1, tempo: '01:15', segundos: 75, texto: 'substantivo concreto' },
      { parte: 2, tempo: '1:02:05', segundos: 3725, texto: 'coletivo' },
    ])
  })
})
