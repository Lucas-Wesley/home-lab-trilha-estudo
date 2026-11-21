import { describe, expect, it } from 'vitest'
import { itensPorId, linkReferencia, progressoReferencia, resumoFase, rotuloSituacao, textoReferencia } from './projeto.js'

describe('projeto de estudo', () => {
  it('liga a referência à etapa ou à trilha', () => {
    expect(linkReferencia({ tipo: 'etapa', trilha_id: 13, etapa_id: 40 })).toBe('#/trilhas/13/etapas/40')
    expect(linkReferencia({ tipo: 'modulo', trilha_id: 24, modulo_id: 9 })).toBe('#/trilhas/24')
    expect(linkReferencia({ tipo: '', texto: 'T99 › Nada' })).toBe('')
  })

  it('mostra o caminho real só quando casou inteiro', () => {
    expect(textoReferencia({ tipo: 'etapa', caminho: 'T13 › Concorrência › Goroutines e o modelo', texto: 'T13 › Concorrência › Goroutines' }))
      .toBe('T13 › Concorrência › Goroutines e o modelo')
    expect(textoReferencia({ tipo: 'modulo', parcial: true, caminho: 'T18 › Módulo 2', texto: 'T18 › M2 › DAS' })).toBe('T18 › M2 › DAS')
    expect(textoReferencia({ tipo: '', texto: 'T99' })).toBe('T99')
  })

  it('conta o progresso de módulo e trilha', () => {
    expect(progressoReferencia({ tipo: 'modulo', concluidas: 2, total: 5 })).toBe('2/5')
    expect(progressoReferencia({ tipo: 'etapa', status: 'concluida' })).toBe('')
    expect(progressoReferencia({ tipo: 'trilha', total: 0 })).toBe('')
  })

  it('dá o nome da situação com a fase do ciclo', () => {
    expect(rotuloSituacao({ situacao: 'em_andamento', estado: 'caminhos' })).toEqual({ nome: 'Em andamento · caminhos', cor: 'azul' })
    expect(rotuloSituacao({ situacao: 'feito', estado: 'feito' }).nome).toBe('Feito')
    expect(rotuloSituacao({}).nome).toBe('A fazer')
  })

  it('junta os itens e resume a fase', () => {
    const dados = { fases: [{ itens: [{ id: '0.1', situacao: 'feito' }, { id: '0.2', situacao: 'a_fazer' }] }, { itens: [{ id: 'R1' }] }] }
    expect(Object.keys(itensPorId(dados))).toEqual(['0.1', '0.2', 'R1'])
    expect(resumoFase(dados.fases[0])).toEqual({ feitos: 1, total: 2 })
  })
})
