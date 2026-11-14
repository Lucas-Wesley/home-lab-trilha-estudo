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
  const base = '#/trilhas/3/documentos/'
  it('documento relativo vira rota do documento na trilha, com a âncora', () => {
    expect(reescreverLink('06-proxmox-instalacao.md#41-reserva', base).href).toBe('#/trilhas/3/documentos/06-proxmox-instalacao#41-reserva')
    expect(reescreverLink('../casa-inteligente-rede.md', base).href).toBe('#/trilhas/3/documentos/casa-inteligente-rede')
  })
  it('sem trilha, o link fica como veio', () => {
    expect(reescreverLink('02-wifi.md').href).toBe('02-wifi.md')
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
    expect(renderizar('[ver](02-wifi.md#canais)', { baseDocs: '#/trilhas/1/documentos/' }).html).toContain('href="#/trilhas/1/documentos/02-wifi#canais"')
  })
})
