import { describe, expect, it } from 'vitest'
import { DESENHOS, capaDaTrilha } from './capa.js'

describe('capaDaTrilha', () => {
  it('reconhece os assuntos das trilhas', () => {
    expect(capaDaTrilha('DDD (Domain-Driven Design)').sigla).toBe('DDD')
    expect(capaDaTrilha('AWS Cloud Practitioner (CLF-C02)').sigla).toBe('CLF')
    expect(capaDaTrilha('AWS para engenharia de software').sigla).toBe('AWS')
    expect(capaDaTrilha('Golang do zero ao backend em produção').sigla).toBe('Go')
    expect(capaDaTrilha('Gramática normativa da língua portuguesa').sigla).toBe('Aa')
    expect(capaDaTrilha('Inglês para Devs').sigla).toBe('EN')
    expect(capaDaTrilha('Home lab: rede').sigla).toBe('Lab')
    expect(capaDaTrilha('Redes de computadores para engenharia de software').sigla).toBe('Net')
  })
  it('fora dos conhecidos, as iniciais; o mesmo título dá a mesma capa', () => {
    const c = capaDaTrilha('Culinária japonesa')
    expect(c.sigla).toBe('CJ')
    expect(capaDaTrilha('Culinária japonesa')).toEqual(c)
    expect(DESENHOS).toContain(c.desenho)
    expect(c.matiz).toBeGreaterThanOrEqual(0)
    expect(capaDaTrilha('').sigla).toBe('·')
  })
})
