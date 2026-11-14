import { describe, expect, it } from 'vitest'
import { ancora, reescreverLink, renderizar, textoComCodigo } from './markdown.js'

describe('textoComCodigo', () => {
  it('destaca código e escapa o resto', () => {
    expect(textoComCodigo('use `route print` <b>já</b>')).toBe('use <code>route print</code> &lt;b&gt;já&lt;/b&gt;')
  })
})

describe('ancora (igual ao GitHub)', () => {
  it('tira pontuação e troca espaços por hífens', () => {
    expect(ancora('4.1 Reserva de IP: IP fora do pool × static lease')).toBe('41-reserva-de-ip-ip-fora-do-pool--static-lease')
  })
  it('mantém acentos', () => {
    expect(ancora('2.2 MAC e IP: dois endereços para dois problemas')).toBe('22-mac-e-ip-dois-endereços-para-dois-problemas')
  })
})

describe('reescreverLink', () => {
  it('documento relativo vira rota do app, com a âncora', () => {
    expect(reescreverLink('06-proxmox-instalacao.md#41-reserva').href).toBe('#/documentos/06-proxmox-instalacao#41-reserva')
    expect(reescreverLink('../casa-inteligente-rede.md').href).toBe('#/documentos/casa-inteligente-rede')
  })
  it('externo abre fora', () => {
    expect(reescreverLink('https://openwrt.org/docs')).toEqual({ href: 'https://openwrt.org/docs', externo: true })
  })
  it('âncora da própria página', () => {
    expect(reescreverLink('#zonas')).toEqual({ href: '#zonas', ancoraLocal: 'zonas' })
  })
})

describe('renderizar', () => {
  it('dá id aos títulos e lista h2/h3', () => {
    const { html, titulos } = renderizar('# Doc\n\n## Zonas e regras\n\n### Input\n\n## Zonas e regras')
    expect(html).toContain('<h2 id="zonas-e-regras">')
    expect(titulos.map((t) => t.id)).toEqual(['zonas-e-regras', 'input', 'zonas-e-regras-1'])
  })
  it('limpa HTML perigoso e mantém details', () => {
    const { html } = renderizar('<details><summary>Resposta</summary>42</details>\n\n<img src=x onerror=alert(1)>')
    expect(html).toContain('<details>')
    expect(html).not.toContain('onerror')
  })
  it('embrulha tabelas para rolar', () => {
    expect(renderizar('| a |\n|---|\n| b |').html).toContain('<div class="tabela-rolavel"><table>')
  })
  it('link para documento vira rota', () => {
    expect(renderizar('[ver](02-wifi.md#canais)').html).toContain('href="#/documentos/02-wifi#canais"')
  })
})
