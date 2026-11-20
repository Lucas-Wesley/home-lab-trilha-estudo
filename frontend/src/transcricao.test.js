import { describe, expect, it } from 'vitest'
import { documentoDaTranscricao, etapaComTranscricao, slugDaTranscricao, temTranscricao } from './transcricao.js'

const etapa = {
  chave: 'm1-02', titulo: 'Modelo OSI', objetivo: '', bloom: 'entender', profundidade: 0, tipo: 'normal', minutos: 20,
  criterio: '', requer: [3], materiais: [{ tipo: 'web', titulo: 'Aula', referencia: 'https://x', papel: 'principal' }],
}

describe('transcrição colada', () => {
  it('nome do documento pela chave', () => {
    expect(slugDaTranscricao('m1-02')).toBe('transcricao-m1-02')
  })
  it('documento com título', () => {
    expect(documentoDaTranscricao('Modelo OSI', '  oi\n')).toBe('# Transcrição: Modelo OSI\n\noi\n')
  })
  it('põe a transcrição primeiro, uma vez só, sem mexer na lição', () => {
    const c1 = etapaComTranscricao(etapa, 'Modelo OSI')
    expect(c1.materiais.map((m) => m.tipo)).toEqual(['documento', 'web'])
    expect(c1.materiais[0]).toMatchObject({ referencia: 'transcricao-m1-02', papel: 'principal' })
    expect('licao' in c1).toBe(false)
    const c2 = etapaComTranscricao({ ...etapa, materiais: c1.materiais }, 'Modelo OSI')
    expect(c2.materiais).toHaveLength(2)
    expect(temTranscricao({ ...etapa, materiais: c1.materiais })).toBe(true)
    expect(temTranscricao(etapa)).toBe(false)
  })
})
