import { describe, expect, it } from 'vitest'
import { agruparPorTrilha, candidatosDoItem, itensPorId, semTrilha, situacaoEstudo, linkReferencia, progressoReferencia, resumoFase, rotuloSituacao, textoReferencia } from './projeto.js'

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

  it('tira o número da trilha do caminho', () => {
    expect(semTrilha('T13 › Fundamentos › Por que Go')).toBe('Fundamentos › Por que Go')
    expect(semTrilha('T40')).toBe('')
  })

  it('junta o estudar antes e a 7.1 por trilha, sem repetir', () => {
    const git = { tipo: 'modulo', trilha_id: 40, trilha_titulo: 'Git, GitHub e CI/CD', modulo_id: 359, caminho: 'T40 › Git no dia a dia' }
    const item = {
      estudo: [
        { texto: 'T13 › A', refs: [{ tipo: 'etapa', trilha_id: 13, trilha_titulo: 'Golang', etapa_id: 1, caminho: 'T13 › A' }] },
        { texto: 'T40 › Git', refs: [git] },
        { texto: 'T99 › Sem par', refs: [] },
      ],
      trilhas_71: { texto: '…', refs: [git, { tipo: 'modulo', trilha_id: 24, trilha_titulo: 'Go Expert', modulo_id: 212, caminho: 'T24 › 01' }] },
      livros_71: { texto: 'T21 cap. 1', refs: [{ tipo: 'etapa', trilha_id: 21, etapa_id: 9, caminho: 'T21 › Cap. 1' }] },
    }
    const grupos = candidatosDoItem(item)
    expect(grupos.map((g) => g.titulo)).toEqual(['Golang', 'Git, GitHub e CI/CD', 'T99', 'Go Expert'])
    expect(grupos[1].refs).toHaveLength(1)
    expect(agruparPorTrilha([{ texto: 'livre' }])[0].titulo).toBe('Outras fontes')
  })

  it('pinta a situação da fonte escolhida', () => {
    expect(situacaoEstudo('Já sabia')).toBe('verde')
    expect(situacaoEstudo('estudado')).toBe('verde')
    expect(situacaoEstudo('a estudar')).toBe('')
  })
})
