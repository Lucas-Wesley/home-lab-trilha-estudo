// T43: os helpers do material "curso" — referência, título, números de aula e "Precisam ser
// ouvidas". A árvore de teste é a mesma do pastas.test.js.
import { describe, expect, it } from 'vitest'
import { aulasDaPasta, aulasDoLocalizador, escreverCursos, linkDoGrupo, nomesDasAulas, precisamOuvir, referenciaDaAula, referenciaDoCurso, tituloDoCurso } from './cursos.js'

const aula = (nome, link, chave = '') => ({ nome, link, chave })
const pasta = (nome, caminho, aulas = [], pastas = []) => ({ nome, caminho, aulas, pastas })

describe('referência do material curso', () => {
  const cursoA = pasta('Curso A', ['Grupo', 'Curso A'], [aula('Aula 1', 'https://t.me/c/1')])
  const cursoDDD = pasta('Curso DDD', ['MBA', 'Curso DDD'], [aula('Intro', 'https://t.me/c/2')])

  it('escreveCursos junta as pastas com " | "', () => {
    const ref = escreverCursos([
      { link: 'https://t.me/c/1', caminho: 'Grupo/Curso A' },
      { link: 'https://t.me/c/2', caminho: 'MBA/Curso DDD' },
    ])
    expect(ref).toBe('telegram:https://t.me/c/1::Grupo/Curso A | telegram:https://t.me/c/2::MBA/Curso DDD')
  })

  it('referenciaDoCurso tira o link do grupo das aulas da pasta', () => {
    expect(referenciaDoCurso([cursoA])).toBe('telegram:https://t.me/c/1::Grupo/Curso A')
    expect(referenciaDoCurso([cursoA, cursoDDD])).toBe(
      'telegram:https://t.me/c/1::Grupo/Curso A | telegram:https://t.me/c/2::MBA/Curso DDD',
    )
  })

  it('linkDoGrupo desce nas subpastas; pasta só de disco não tem link', () => {
    const mae = pasta('Mãe', ['G', 'Mãe'], [], [pasta('Filha', ['G', 'Mãe', 'Filha'], [aula('X', 'https://t.me/c/1')])])
    expect(linkDoGrupo(mae)).toBe('https://t.me/c/1')
    const disco = pasta('Viagens', ['Viagens'], [aula('Chile', '')])
    expect(linkDoGrupo(disco)).toBe('')
    expect(referenciaDoCurso([disco])).toBeNull()
  })

  it('tituloDoCurso junta os nomes com " + "', () => {
    expect(tituloDoCurso([cursoA, cursoDDD])).toBe('Curso A + Curso DDD')
  })
})

describe('aulasDoLocalizador', () => {
  it('lê números, faixas e listas', () => {
    expect(aulasDoLocalizador('A12')).toEqual([12])
    expect(aulasDoLocalizador('A12–A15')).toEqual([12, 13, 14, 15])
    expect(aulasDoLocalizador('A3-A4')).toEqual([3, 4])
    expect(aulasDoLocalizador('A3, A7')).toEqual([3, 7])
    expect(aulasDoLocalizador('A2 A2 A5')).toEqual([2, 5])
  })

  it('sem o prefixo A não é aula', () => {
    expect(aulasDoLocalizador('cap. 4')).toEqual([])
    expect(aulasDoLocalizador('p. 45–60')).toEqual([])
    expect(aulasDoLocalizador('')).toEqual([])
  })
})

describe('nomes das aulas e precisam ouvir', () => {
  const nomes = { A1: 'Aula 1', A3: 'DDD tático' }

  it('nomesDasAulas resolve pelo mapa da proposta', () => {
    expect(nomesDasAulas('A1–A2', nomes)).toBe('Aula 1 · A2')
    expect(nomesDasAulas('A3, A9', nomes)).toBe('DDD tático · A9')
  })

  it('precisamOuvir lê a linha das notas', () => {
    expect(precisamOuvir('Títulos bons.\nPrecisam ser ouvidas: A4, A7')).toEqual([4, 7])
    expect(precisamOuvir('sem a linha')).toEqual([])
  })
})

describe('processar (T44)', () => {
  const arvore = pasta(
    'Curso',
    ['G', 'Curso'],
    [aula('A1', 'https://t.me/c/1'), { nome: 'Só disco', chave: 'd1' }],
    [pasta('Módulo', ['G', 'Curso', 'Módulo'], [{ nome: 'A2', link: 'https://t.me/c/1', mensagem: 7, chave: 'a2' }])],
  )

  it('aulasDaPasta desce na árvore na ordem', () => {
    expect(aulasDaPasta(arvore).map((a) => a.nome)).toEqual(['A1', 'Só disco', 'A2'])
  })

  it('referenciaDaAula monta a ref estável; aula sem link/mensagem não processa', () => {
    expect(referenciaDaAula({ nome: 'A2', link: 'https://t.me/c/1', mensagem: 7 })).toBe('telegram:https://t.me/c/1#7')
    expect(referenciaDaAula({ nome: 'Só disco' })).toBe('')
    expect(referenciaDaAula({ link: 'https://t.me/c/1' })).toBe('') // sem mensagem
  })
})
