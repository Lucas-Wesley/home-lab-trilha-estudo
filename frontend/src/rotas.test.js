import { describe, expect, it } from 'vitest'
import { lerRota, link } from './rotas.js'

describe('lerRota', () => {
  it('abre em Hoje sem hash', () => {
    expect(lerRota('')).toEqual({ secao: 'hoje', partes: [], ancora: '' })
  })

  it('lê a seção e as partes', () => {
    expect(lerRota('#/trilhas/home-lab-rede/3')).toEqual({ secao: 'trilhas', partes: ['home-lab-rede', '3'], ancora: '' })
  })

  it('separa a âncora de uma seção da página', () => {
    expect(lerRota('#/design#botoes')).toEqual({ secao: 'design', partes: [], ancora: 'botoes' })
  })

  it('seção desconhecida volta para Hoje', () => {
    expect(lerRota('#/nada/x').secao).toBe('hoje')
  })

  it('decodifica acentos', () => {
    expect(lerRota('#/trilhas/hist%C3%B3ria').partes).toEqual(['história'])
  })

  it('não quebra com "%" malformado', () => {
    expect(lerRota('#/trilhas/x%E2').secao).toBe('trilhas')
  })
})

describe('link', () => {
  it('monta o hash codificado', () => {
    expect(link('trilhas', 'história')).toBe('#/trilhas/hist%C3%B3ria')
  })
})
