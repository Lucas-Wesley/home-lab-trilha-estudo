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
