import { describe, expect, it } from 'vitest'
import { agruparAulas, temDownload } from './aulasCurso.js'

const aula = (id, plataforma, curso, modulo, estado = 'pronta', tamanho = 100) => ({ id, plataforma, curso, modulo, titulo: `A${id}`, estado, tamanho })

describe('agruparAulas', () => {
  it('agrupa em plataforma › curso › módulo, na ordem em que foram pedidas', () => {
    const g = agruparAulas(
      [aula(3, 'linuxtips', 'System Design', 'Módulo 1'), aula(1, 'branas', 'Clean Code', 'Aula 6'), aula(2, 'branas', 'Clean Code', 'Aula 6', 'baixando'), aula(4, 'branas', 'Clean Code', 'Aula 2')],
      { branas: 'Branas.io', linuxtips: 'LINUXtips' },
    )
    expect(g.map((p) => p.nome)).toEqual(['Branas.io', 'LINUXtips'])
    const cc = g[0].cursos[0]
    expect(cc.modulos.map((m) => m.nome)).toEqual(['Aula 6', 'Aula 2'])
    expect(cc.modulos[0].aulas.map((a) => a.id)).toEqual([1, 2])
    expect([cc.total, cc.baixadas, cc.tamanho]).toEqual([3, 2, 200])
  })
  it('sabe quando há download andando', () => {
    expect(temDownload([aula(1, 'b', 'c', '', 'pronta')])).toBe(false)
    expect(temDownload([aula(1, 'b', 'c', '', 'na_fila')])).toBe(true)
  })
})

describe('T55', () => {
  it('ordem do curso, aula só de texto e a trilha do curso', () => {
    const aulas = [
      { id: 5, plataforma: 'enzoneto', curso: 'Praças', modulo: 'Bases', titulo: 'B', ordem: 2, estado: 'texto', tamanho: 0 },
      { id: 9, plataforma: 'enzoneto', curso: 'Praças', modulo: 'Bases', titulo: 'A', ordem: 1, estado: 'pronta', tamanho: 10 },
      { id: 1, plataforma: 'enzoneto', curso: 'Praças', modulo: 'Bases', titulo: 'Sem ordem', ordem: 0, estado: 'na_fila', tamanho: 0 },
    ]
    const [p] = agruparAulas(aulas, { enzoneto: 'Chef Enzo Neto' }, [{ plataforma: 'enzoneto', curso: 'Praças', trilha_id: 7 }])
    const c = p.cursos[0]
    expect(p.nome).toEqual('Chef Enzo Neto')
    expect(c.trilha).toEqual(7)
    expect(c.modulos[0].aulas.map((a) => a.titulo)).toEqual(['A', 'B', 'Sem ordem'])
    expect(c.soTexto).toEqual(1)
    expect(c.baixadas).toEqual(1)
    expect(agruparAulas(aulas, {})[0].cursos[0].trilha).toEqual(0)
  })
})
