import { Marked } from 'marked'
import DOMPurify from 'dompurify'

// Âncora de um título no mesmo formato do GitHub, para os links entre os documentos de estudo
// continuarem funcionando ("4.1 Reserva de IP: IP fora do pool × static lease" →
// "41-reserva-de-ip-ip-fora-do-pool--static-lease").
export function ancora(texto) {
  return texto
    .trim()
    .toLowerCase()
    .replace(/<[^>]*>/g, '')
    .replace(/[^\p{L}\p{N}\s_-]/gu, '')
    .replace(/\s/g, '-')
}

// Link relativo para outro Markdown ("03-redes.md#zonas", "../casa.md") vira a rota do documento
// no app; links externos abrem em outra aba.
export function reescreverLink(href) {
  if (!href) return { href }
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return { href, externo: true }
  const m = href.match(/^(?:\.{1,2}\/)*(?:[\w.-]+\/)*([\w.-]+)\.md(?:#(.*))?$/i)
  if (m) {
    const slug = m[1].toLowerCase()
    return { href: `#/documentos/${encodeURIComponent(slug)}${m[2] ? '#' + m[2] : ''}` }
  }
  if (href.startsWith('#') && !href.startsWith('#/')) return { href, ancoraLocal: href.slice(1) }
  return { href }
}

// textoComCodigo: texto simples (objetivo, critério) com `código` em destaque; todo o resto é escapado.
export function textoComCodigo(texto) {
  const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  return esc(texto || '').replace(/`([^`]+)`/g, '<code>$1</code>')
}

// renderizar devolve o HTML limpo e os títulos (h2 e h3) para o "nesta página".
export function renderizar(texto) {
  const titulos = []
  const usados = new Map()
  const marked = new Marked({
    gfm: true,
    renderer: {
      heading({ tokens, depth }) {
        const html = this.parser.parseInline(tokens)
        const base = ancora(html.replace(/&amp;/g, '&').replace(/&#39;/g, "'").replace(/&quot;/g, '"'))
        const n = usados.get(base) || 0
        usados.set(base, n + 1)
        const id = n ? `${base}-${n}` : base
        if (depth === 2 || depth === 3) titulos.push({ id, nivel: depth, texto: html.replace(/<[^>]*>/g, '') })
        return `<h${depth} id="${id}">${html}</h${depth}>\n`
      },
      link({ href, title, tokens }) {
        const texto = this.parser.parseInline(tokens)
        const r = reescreverLink(href)
        const t = title ? ` title="${title}"` : ''
        if (r.externo) return `<a href="${r.href}"${t} target="_blank" rel="noopener noreferrer">${texto}</a>`
        if (r.ancoraLocal !== undefined) return `<a href="${r.href}" data-ancora="${r.ancoraLocal}"${t}>${texto}</a>`
        return `<a href="${r.href}"${t}>${texto}</a>`
      },
    },
  })
  const html = DOMPurify.sanitize(marked.parse(texto || ''), {
    ADD_ATTR: ['target', 'data-ancora'],
  })
    // tabela larga rola sozinha no celular, sem empurrar a página
    .replace(/<table>/g, '<div class="tabela-rolavel"><table>')
    .replace(/<\/table>/g, '</table></div>')
  return { html, titulos }
}
