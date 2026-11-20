import { describe, expect, it } from 'vitest'
import { pacoteDoCurso, resumoDoPacote, modoRecomendado } from './cursoParaTrilha.js'

// A árvore inventada do plano: grupo › curso › dois módulos com três aulas + uma aula solta.
const LINK = 'https://t.me/c/1'
const aula = (mensagem, nome, duracao = 600) => ({
  chave: `${LINK}#${mensagem}`, nome, link: LINK, grupo: 'Grupo', mensagem, duracao, tamanho: 100,
})
const curso = {
  nome: 'Curso X',
  caminho: ['Grupo', 'Curso X'],
  aulas: [aula(10, 'Aula solta', 900)],
  pastas: [
    { nome: 'Módulo 1', caminho: ['Grupo', 'Curso X', 'Módulo 1'], aulas: [aula(1, 'A1'), aula(2, 'A2')], pastas: [] },
    { nome: 'Módulo 2', caminho: ['Grupo', 'Curso X', 'Módulo 2'], aulas: [aula(3, 'A3', 300)], pastas: [] },
  ],
}

describe('cursoParaTrilha', () => {
  it('uma etapa por aula: cada aula é uma etapa, cada pasta com aulas um módulo', () => {
    const p = pacoteDoCurso(curso, 'por_aula', 'Meu curso')
    expect(p.trilha.estado).toBe('ativa')
    expect(p.trilha.titulo).toBe('Meu curso')
    expect(p.trilha.modulos.map((m) => m.titulo)).toEqual(['Curso X', 'Módulo 1', 'Módulo 2'])
    const etapas = p.trilha.modulos.flatMap((m) => m.etapas)
    expect(etapas.map((e) => e.titulo)).toEqual(['Aula solta', 'A1', 'A2', 'A3'])
    expect(etapas[0].materiais).toEqual([
      { tipo: 'streaming', papel: 'principal', titulo: 'Aula solta', referencia: 'telegram:https://t.me/c/1#10' },
    ])
    expect(etapas[0].minutos).toBe(15)
    expect(etapas[0].tipo).toBe('normal')
    expect(etapas[0].objetivo).toBe('Assistir à aula: Aula solta')
    expect(new Set(etapas.map((e) => e.chave)).size).toBe(4)
    expect(resumoDoPacote(p).texto).toBe('3 módulos · 4 etapas · 40 min')
  })

  it('uma etapa por pasta: cada pasta com aulas é uma etapa; o módulo é a pasta mãe', () => {
    const p = pacoteDoCurso(curso, 'por_pasta', '')
    expect(p.trilha.titulo).toBe('Curso X')
    // Filhas diretas da escolhida e aulas soltas → módulo da própria pasta escolhida.
    expect(p.trilha.modulos.map((m) => m.titulo)).toEqual(['Curso X'])
    const etapas = p.trilha.modulos[0].etapas
    expect(etapas.map((e) => e.titulo)).toEqual(['Curso X', 'Módulo 1', 'Módulo 2'])
    expect(etapas[1].materiais.map((m) => m.referencia)).toEqual([
      'telegram:https://t.me/c/1#1', 'telegram:https://t.me/c/1#2',
    ])
    expect(etapas[1].minutos).toBe(20)
    expect(etapas[1].objetivo).toBe('Assistir às aulas de Módulo 1')
    expect(resumoDoPacote(p).etapas).toBe(3)
  })

  it('pasta aninhada vira módulo pelo caminho', () => {
    const fundo = {
      nome: 'Curso X', caminho: ['Grupo', 'Curso X'], aulas: [], pastas: [
        { nome: 'M1', caminho: ['Grupo', 'Curso X', 'M1'], aulas: [], pastas: [
          { nome: 'Semana 1', caminho: ['Grupo', 'Curso X', 'M1', 'Semana 1'], aulas: [aula(5, 'S1')], pastas: [] },
        ] },
      ],
    }
    const p = pacoteDoCurso(fundo, 'por_pasta', '')
    expect(p.trilha.modulos.map((m) => m.titulo)).toEqual(['M1'])
    expect(p.trilha.modulos[0].etapas[0].titulo).toBe('Semana 1')
    const p2 = pacoteDoCurso(fundo, 'por_aula', '')
    expect(p2.trilha.modulos.map((m) => m.titulo)).toEqual(['M1 › Semana 1'])
  })

  it('recomenda por pasta quando a média é curta, por aula quando é longa', () => {
    expect(modoRecomendado(curso)).toBe('por_pasta') // média de 10 min
    const longo = { nome: 'L', caminho: ['L'], aulas: [aula(1, 'A', 30 * 60), aula(2, 'B', 40 * 60)], pastas: [] }
    expect(modoRecomendado(longo)).toBe('por_aula')
  })
})
