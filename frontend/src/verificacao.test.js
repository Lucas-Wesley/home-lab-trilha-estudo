import { describe, expect, it } from 'vitest'
import { emAndamento, fase, mudouDesde, naoVerificados, pedidoDasFaltas } from './verificacao.js'

const doc = { tipo: 'documento', referencia: 'a', localizador: '' }
const yt = { tipo: 'youtube', referencia: 'https://youtu.be/x', localizador: '00:10–02:00' }

describe('mudouDesde', () => {
  const v = { materiais: [{ ...doc, indice: 0 }, { ...yt, indice: 1 }] }
  it('igual: não mudou', () => expect(mudouDesde(v, [doc, yt])).toBe(false))
  it('trecho trocado ou material tirado: mudou', () => {
    expect(mudouDesde(v, [doc, { ...yt, localizador: '00:00–01:00' }])).toBe(true)
    expect(mudouDesde(v, [doc])).toBe(true)
  })
  it('material novo não muda o que foi verificado', () => expect(mudouDesde(v, [doc, yt, doc])).toBe(false))
  it('sem verificação', () => expect(mudouDesde(null, [doc])).toBe(false))
})

describe('naoVerificados', () => {
  it('os da etapa que a verificação não leu', () => {
    const novo = { tipo: 'web', referencia: 'https://x', localizador: '' }
    expect(naoVerificados({ materiais: [{ ...yt, indice: 1 }] }, [doc, yt, novo])).toEqual([doc, novo])
    expect(naoVerificados(null, [doc])).toEqual([])
  })
})

describe('pedidoDasFaltas', () => {
  it('faltas com a fonte e tópicos rasos sem repetir', () => {
    const p = pedidoDasFaltas({
      faltas: [{ ponto: 'MVCC', onde_achar: 'Docs do PostgreSQL' }],
      topicos: [{ topico: 'MVCC', situacao: 'ausente' }, { topico: 'Vacuum', situacao: 'raso' }, { topico: 'WAL', situacao: 'coberto' }],
    })
    expect(p).toContain('- MVCC (fonte sugerida: Docs do PostgreSQL)')
    expect(p).toContain('- Vacuum (está raso)')
    expect(p).not.toContain('WAL')
    expect(p.match(/MVCC/g)).toHaveLength(1)
  })
  it('nada falta: vazio', () => expect(pedidoDasFaltas({ faltas: [], topicos: [{ topico: 'x', situacao: 'coberto' }] })).toBe(''))
})

describe('estado', () => {
  it('em andamento e a frase da espera', () => {
    expect(emAndamento({ estado: 'aguardando' })).toBe(true)
    expect(emAndamento({ estado: 'pronta' })).toBe(false)
    expect(fase({ estado: 'aguardando', materiais: [{ texto: 'transcrevendo' }, { texto: 'pronto' }] })).toContain('o vídeo')
    expect(fase({ estado: 'verificando', tarefa: { estado: 'rodando' } })).toContain('lendo')
    expect(fase({ estado: 'verificando', tarefa: { estado: 'na_fila' } })).toContain('fila')
  })
})
