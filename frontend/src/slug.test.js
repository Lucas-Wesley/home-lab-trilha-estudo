import { expect, it } from 'vitest'
import { slugDoArquivo } from './slug.js'

it('nome de arquivo vira nome de documento', () => {
  expect(slugDoArquivo('03-redes-separadas-firewall.md')).toBe('03-redes-separadas-firewall')
  expect(slugDoArquivo('Glossário de Redes.MD')).toBe('glossario-de-redes')
  expect(slugDoArquivo('--x__y--.md')).toBe('x-y')
})
