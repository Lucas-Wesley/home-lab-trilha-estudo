import { describe, expect, it } from 'vitest'
import { linhasDaTranscricao, linkCompartilhado, linkDoYouTube, linkNoMinuto } from './entrada.js'
import { tempoParaSegundos } from './videoaula.js'

describe('linkDoYouTube', () => {
  it('acha o link no texto do compartilhar do YouTube', () => {
    expect(linkDoYouTube('Como funciona o Docker https://youtu.be/abc123XYZ_-?si=xyz')).toBe(
      'https://youtu.be/abc123XYZ_-?si=xyz',
    )
    expect(linkDoYouTube('Aula nova https://www.youtube.com/watch?v=abc123XYZ_-&t=30 legal')).toBe(
      'https://www.youtube.com/watch?v=abc123XYZ_-&t=30',
    )
    expect(linkDoYouTube('Curso inteiro: https://youtube.com/playlist?list=PL123abc_-XYZ')).toBe(
      'https://youtube.com/playlist?list=PL123abc_-XYZ',
    )
  })

  it('completa o https:// quando o texto vem sem o esquema (o servidor exige)', () => {
    expect(linkDoYouTube('youtu.be/abc123XYZ_-')).toBe('https://youtu.be/abc123XYZ_-')
    expect(linkDoYouTube('olha: youtube.com/watch?v=abc123XYZ_-')).toBe('https://youtube.com/watch?v=abc123XYZ_-')
  })

  it('devolve "" quando não tem link do YouTube', () => {
    expect(linkDoYouTube('')).toBe('')
    expect(linkDoYouTube('só texto, sem link')).toBe('')
    expect(linkDoYouTube('https://example.com/video')).toBe('')
  })

  it('ignora pontuação colada no fim do link', () => {
    expect(linkDoYouTube('(veja https://youtu.be/abc123XYZ_-).')).toBe('https://youtu.be/abc123XYZ_-')
  })
})

describe('linkCompartilhado', () => {
  it('junta url, text e title do share_target', () => {
    expect(linkCompartilhado('?title=V%C3%ADdeo&text=olha')).toBe('')
    expect(linkCompartilhado('?title=Aula&text=Veja%20https%3A%2F%2Fyoutu.be%2Fabc123XYZ_-')).toBe(
      'https://youtu.be/abc123XYZ_-',
    )
    expect(linkCompartilhado('?url=https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3Dabc123XYZ_-')).toBe(
      'https://www.youtube.com/watch?v=abc123XYZ_-',
    )
  })
})

describe('linhasDaTranscricao', () => {
  it('marca os segundos das linhas [mm:ss] e [h:mm:ss]', () => {
    const linhas = linhasDaTranscricao('# Título\n\n[00:30] primeiro\n[1:02:05] depois')
    expect(linhas[2]).toEqual({ marca: '00:30', segundos: 30, texto: 'primeiro' })
    expect(linhas[3]).toEqual({ marca: '1:02:05', segundos: 3725, texto: 'depois' })
    expect(linhas[0].segundos).toBeNull()
  })
})

describe('linkNoMinuto', () => {
  it('monta o link do YouTube no segundo', () => {
    expect(linkNoMinuto('abc123XYZ_-', 30)).toBe('https://youtu.be/abc123XYZ_-?t=30')
    expect(linkNoMinuto('abc123XYZ_-', 3725.8)).toBe('https://youtu.be/abc123XYZ_-?t=3725')
  })
})

describe('tempoParaSegundos', () => {
  it('converte mm:ss e h:mm:ss', () => {
    expect(tempoParaSegundos('1:02:05')).toBe(3725)
    expect(tempoParaSegundos('00:30')).toBe(30)
  })
})
