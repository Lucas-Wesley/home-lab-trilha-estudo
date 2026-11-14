import { expect, it } from 'vitest'
import { hrefMaterial, localizadorDeCapitulos, localizadorDePaginas } from './material.js'

it('link de cada tipo de material', () => {
  expect(hrefMaterial({ tipo: 'web', referencia: 'https://x.org' }, 1)).toBe('https://x.org')
  expect(hrefMaterial({ tipo: 'documento', referencia: '05-wan', localizador: '23-nat' }, 2)).toBe('#/trilhas/2/documentos/05-wan#23-nat')
  expect(hrefMaterial({ tipo: 'livro', referencia: 'abc', localizador: 'cap. 3' }, 1)).toBe('/api/material/abrir?tipo=livro&referencia=abc&localizador=cap.+3')
  expect(hrefMaterial({ tipo: 'acervo', referencia: 'temas/redes', localizador: '' }, 1)).toBe('/api/material/abrir?tipo=acervo&referencia=temas%2Fredes')
  expect(hrefMaterial({ tipo: 'streaming', referencia: '' }, 1)).toBe('')
})

it('capítulos e páginas viram o localizador que o servidor lê', () => {
  expect(localizadorDeCapitulos([5, 1, 2, 3])).toBe('caps. 1–3, 5')
  expect(localizadorDeCapitulos([4])).toBe('cap. 4')
  expect(localizadorDeCapitulos([7, 8])).toBe('caps. 7, 8')
  expect(localizadorDeCapitulos([])).toBe('')
  expect(localizadorDePaginas(45, 60)).toBe('p. 45–60')
  expect(localizadorDePaginas(12, 12)).toBe('p. 12')
  expect(localizadorDePaginas(0, 0)).toBe('')
})
