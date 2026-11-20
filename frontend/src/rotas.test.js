import { describe, expect, it } from 'vitest'
import { lerRota, link, linkPasta } from './rotas.js'

describe('lerRota', () => {
  it('abre em Hoje sem hash', () => {
    expect(lerRota('')).toEqual({ secao: 'hoje', partes: [], ancora: '' })
  })

  it('lê a seção e as partes', () => {
    expect(lerRota('#/trilhas/home-lab-rede/3')).toEqual({ secao: 'trilhas', partes: ['home-lab-rede', '3'], ancora: '' })
  })

  it('lê a entrada e a fonte', () => {
    expect(lerRota('#/entrada')).toEqual({ secao: 'entrada', partes: [], ancora: '' })
    expect(lerRota('#/entrada/12')).toEqual({ secao: 'entrada', partes: ['12'], ancora: '' })
  })

  it('lê Meus arquivos com o caminho da pasta em níveis', () => {
    expect(lerRota('#/entrada/arquivos')).toEqual({ secao: 'entrada', partes: ['arquivos'], ancora: '' })
    expect(lerRota('#/entrada/arquivos/Grupo/Curso%20A').partes).toEqual(['arquivos', 'Grupo', 'Curso A'])
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

  it('linkPasta: um nível por parte, codificado', () => {
    expect(linkPasta('Grupo/Curso A')).toBe('#/entrada/arquivos/Grupo/Curso%20A')
    expect(linkPasta('')).toBe('#/entrada/arquivos')
  })
})
