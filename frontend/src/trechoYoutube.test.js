import { describe, expect, it } from 'vitest'
import { chaveDoTrecho, estadoDoTrecho, idDoVideo, intervalos, linkNoMinuto, segundosDosTrechos, trechos } from './trechoYoutube.js'

describe('trechoYoutube', () => {
  it('acha o id do vídeo nos formatos do YouTube', () => {
    expect(idDoVideo('https://www.youtube.com/watch?v=EOmzo5d0F9Y')).toBe('EOmzo5d0F9Y')
    expect(idDoVideo('https://www.youtube.com/watch?v=EOmzo5d0F9Y&t=30s')).toBe('EOmzo5d0F9Y')
    expect(idDoVideo('https://youtu.be/EOmzo5d0F9Y')).toBe('EOmzo5d0F9Y')
    expect(idDoVideo('https://m.youtube.com/watch?v=EOmzo5d0F9Y')).toBe('EOmzo5d0F9Y')
    expect(idDoVideo('https://www.youtube.com/shorts/EOmzo5d0F9Y')).toBe('EOmzo5d0F9Y')
    expect(idDoVideo('https://vimeo.com/123')).toBe('')
  })
  it('lê intervalos em mm:ss, h:mm:ss, hífen e travessão', () => {
    expect(intervalos('13:33–18:25')).toEqual([[813, 1105]])
    expect(intervalos('06:00-10:00; 25:10–31:00')).toEqual([[360, 600], [1510, 1860]])
    expect(intervalos('1:02:05—1:03:00')).toEqual([[3725, 3780]])
    expect(intervalos('10:00–06:00')).toEqual([]) // fim antes do início não vale
    expect(intervalos('')).toEqual([])
  })
  it('devolve os trechos dos materiais youtube principais', () => {
    const ms = [
      { tipo: 'youtube', papel: 'principal', referencia: 'https://www.youtube.com/watch?v=EOmzo5d0F9Y', localizador: '13:33–18:25', titulo: 'TLS' },
      { tipo: 'youtube', papel: 'apoio', referencia: 'https://youtu.be/XXXXXXXXXXX', localizador: '00:00–01:00' },
      { tipo: 'livro', papel: 'principal', referencia: '3', localizador: 'cap. 2' },
      { tipo: 'youtube', papel: 'principal', referencia: 'https://youtu.be/BBBBBBBBBBB', localizador: '00:10–00:40; 00:50–01:00', titulo: 'Duas partes' },
    ]
    expect(trechos(ms)).toEqual([
      { videoId: 'EOmzo5d0F9Y', referencia: ms[0].referencia, localizador: '13:33–18:25', inicio: 813, fim: 1105, titulo: 'TLS' },
      { videoId: 'BBBBBBBBBBB', referencia: ms[3].referencia, localizador: '00:10–00:40', inicio: 10, fim: 40, titulo: 'Duas partes' },
      { videoId: 'BBBBBBBBBBB', referencia: ms[3].referencia, localizador: '00:50–01:00', inicio: 50, fim: 60, titulo: 'Duas partes' },
    ])
  })
  it('diz onde o vídeo está em relação ao trecho', () => {
    expect(estadoDoTrecho(300, 360, 600)).toEqual({ antes: true, dentro: false, passou: false, faltam: 300 })
    expect(estadoDoTrecho(500, 360, 600)).toEqual({ antes: false, dentro: true, passou: false, faltam: 100 })
    expect(estadoDoTrecho(599, 360, 600)).toEqual({ antes: false, dentro: true, passou: true, faltam: 1 }) // fim − 2 s
    expect(estadoDoTrecho(700, 360, 600)).toEqual({ antes: false, dentro: false, passou: true, faltam: 0 })
  })
  it('monta o link no minuto e a chave do visto', () => {
    expect(linkNoMinuto('EOmzo5d0F9Y', 813)).toBe('https://youtu.be/EOmzo5d0F9Y?t=813')
    const t = trechos([{ tipo: 'youtube', papel: 'principal', referencia: 'https://youtu.be/A', localizador: '01:00–02:00' }])[0]
    expect(chaveDoTrecho(t)).toBe('https://youtu.be/A|01:00–02:00')
  })
  it('soma a duração dos trechos (a duração na estrutura)', () => {
    expect(segundosDosTrechos([
      { tipo: 'youtube', papel: 'principal', referencia: 'https://youtu.be/A', localizador: '01:00–02:00; 03:00–04:00' },
    ])).toBe(120)
  })
})
