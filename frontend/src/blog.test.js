import { describe, expect, it } from 'vitest'
import { emAndamento, fase, ideias, limparOpcoes, opcoesPadrao, resumoOpcoes } from './blog.js'

describe('opções', () => {
  it('padrão: atemporal, aprofundamento, a IA escolhe, 3 sugestões', () => {
    expect(opcoesPadrao()).toEqual({ quantas: 3, horizonte: 'atemporal', nivel: 'aprofundamento', formato: 'livre', abrangencia: 'reunir', tamanho: 'medio', codigo: 'pouco' })
  })
  it('limpa o que veio do navegador', () => {
    const o = limparOpcoes({ horizonte: 'atual', formato: 'poema', quantas: 9 })
    expect(o.horizonte).toBe('atual')
    expect(o.formato).toBe('livre')
    expect(o.quantas).toBe(3)
    expect(limparOpcoes('lixo')).toEqual(opcoesPadrao())
  })
  it('resumo numa linha', () => {
    expect(resumoOpcoes(opcoesPadrao())).toBe('atemporal · aprofundamento · a IA escolhe · reunir os assuntos do item · médio · 3 sugestões')
    expect(resumoOpcoes({ ...opcoesPadrao(), codigo: 'nenhum', quantas: 1 }, 'topicos')).toBe('atemporal · aprofundamento · a IA escolhe · médio · nenhum')
  })
})

describe('ideias', () => {
  const pedidos = [
    { id: 2, modo: 'topicos', titulo: 'Meu título', estado: 'pronta', resultado: { tese: 'T2' }, marcas: { 0: { estado: 'escrita', link: 'https://x', em: '2026-11-24' } } },
    {
      id: 1, modo: 'sugerir', estado: 'pronta',
      resultado: { sugestoes: [{ titulo: 'A', tese: 'TA' }, { titulo: 'B', tese: 'TB' }] },
      marcas: { 0: { estado: 'guardada', em: '2026-11-23' }, 1: { estado: 'descartada', em: '2026-11-25' }, 7: { estado: 'guardada', em: '2026-11-23' } },
    },
    { id: 3, modo: 'sugerir', estado: 'gerando', marcas: {} },
  ]
  it('guardadas e escritas, as mais novas primeiro; descartadas e inexistentes fora', () => {
    expect(ideias(pedidos)).toEqual([
      { pedido: 2, n: 0, estado: 'escrita', link: 'https://x', em: '2026-11-24', titulo: 'Meu título', tese: 'T2' },
      { pedido: 1, n: 0, estado: 'guardada', link: '', em: '2026-11-23', titulo: 'A', tese: 'TA' },
    ])
  })
})

describe('andamento', () => {
  it('fase', () => {
    expect(emAndamento({ estado: 'gerando' })).toBe(true)
    expect(fase({ estado: 'gerando', modo: 'topicos', tarefa: { estado: 'rodando' } })).toMatch(/os tópicos/)
    expect(fase({ estado: 'gerando', tarefa: { estado: 'na_fila' } })).toBe('Na fila da IA…')
    expect(fase({ estado: 'pronta' })).toBe('')
  })
})
