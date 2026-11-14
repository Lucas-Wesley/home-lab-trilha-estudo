import { describe, expect, it } from 'vitest'
import { diferenca, resumoDiferenca } from './diff.js'

describe('diferenca', () => {
  it('textos iguais: um trecho igual', () => {
    expect(diferenca('a\nb', 'a\nb')).toEqual([{ tipo: 'igual', linhas: ['a', 'b'] }])
  })
  it('linha trocada vira menos e mais, no lugar', () => {
    expect(diferenca('a\nb\nc', 'a\nB\nc')).toEqual([
      { tipo: 'igual', linhas: ['a'] },
      { tipo: 'menos', linhas: ['b'] },
      { tipo: 'mais', linhas: ['B'] },
      { tipo: 'igual', linhas: ['c'] },
    ])
  })
  it('acréscimo no fim e remoção no começo', () => {
    const d = diferenca('x\na\nb', 'a\nb\nc\nd')
    expect(d[0]).toEqual({ tipo: 'menos', linhas: ['x'] })
    expect(d.at(-1)).toEqual({ tipo: 'mais', linhas: ['c', 'd'] })
    expect(resumoDiferenca(d)).toEqual({ mais: 2, menos: 1 })
  })
  it('texto vazio', () => {
    expect(resumoDiferenca(diferenca('', 'a\nb'))).toEqual({ mais: 2, menos: 1 })
  })
})
