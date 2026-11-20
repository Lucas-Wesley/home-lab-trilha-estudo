import { describe, expect, it } from 'vitest'
import { encaixesDaPasta, trilhasDaPasta, usosPorAula } from './usos.js'

const L = 'https://t.me/c/1'
const aula = (m) => ({ nome: `Aula ${m}`, link: L, mensagem: m })
const pasta = (nome, caminho, aulas = [], pastas = []) => ({ nome, caminho, aulas, pastas })

const mod1 = pasta('Módulo 1', ['Curso', 'Módulo 1'], [aula(1), aula(2)])
const mod2 = pasta('Módulo 2', ['Curso', 'Módulo 2'], [aula(3)])
const curso = pasta('Curso', ['Curso'], [], [mod1, mod2])

const usos = {
  aulas: [
    { ref: `telegram:${L}#1`, trilha_id: 1, trilha: 'Igual ao curso', etapa_chave: 'a1', etapa: 'Aula 1' },
    { ref: `telegram:${L}#2`, trilha_id: 1, trilha: 'Igual ao curso', etapa_chave: 'a2', etapa: 'Aula 2' },
    { ref: `telegram:${L}#3`, trilha_id: 1, trilha: 'Igual ao curso', etapa_chave: 'a3', etapa: 'Aula 3' },
    { ref: `telegram:${L}#2`, trilha_id: 2, trilha: 'System Design', etapa_chave: 'x', etapa: 'Cache' },
    { ref: `telegram:${L}#2`, trilha_id: 2, trilha: 'System Design', etapa_chave: 'y', etapa: 'Filas' },
  ],
  encaixes: [{ pastas: [`telegram:${L}::Curso/Módulo 2`], trilha_id: 3, trilha: 'DDD', estado: 'aguardando' }],
}

describe('usos das aulas nas trilhas', () => {
  const mapa = usosPorAula(usos)

  it('usosPorAula junta por referência', () => {
    expect(mapa.get(`telegram:${L}#2`)).toHaveLength(3)
    expect(mapa.get(`telegram:${L}#9`)).toBeUndefined()
  })

  it('trilhasDaPasta conta cada aula uma vez por trilha e diz quando a pasta inteira está nela', () => {
    expect(trilhasDaPasta(curso, mapa)).toEqual([
      { trilha_id: 1, trilha: 'Igual ao curso', aulas: 3, inteira: true },
      { trilha_id: 2, trilha: 'System Design', aulas: 1, inteira: false },
    ])
    expect(trilhasDaPasta(mod2, mapa)).toEqual([{ trilha_id: 1, trilha: 'Igual ao curso', aulas: 1, inteira: true }])
    expect(trilhasDaPasta(pasta('Vazia', ['Vazia']), mapa)).toEqual([])
  })

  it('encaixesDaPasta pega a própria pasta, a de dentro e a de fora', () => {
    expect(encaixesDaPasta(mod2, usos.encaixes).map((e) => e.trilha)).toEqual(['DDD'])
    expect(encaixesDaPasta(curso, usos.encaixes).map((e) => e.trilha)).toEqual(['DDD'])
    expect(encaixesDaPasta(mod1, usos.encaixes)).toEqual([])
  })
})
