// Confere telas do app num celular emulado (390×844, toque): título, rolagem lateral e erros do
// console. O Chrome do Windows não abre janela com menos de ~500 px, e um iframe numa página file://
// dá falso branco; por isso a conferência roda num Chromium de verdade, num container:
//
//   docker run --rm --add-host=host.docker.internal:host-gateway -v "$PWD/scripts:/w" -w /w \
//     -e BASE=http://host.docker.internal:8093/ node:24-alpine sh -c \
//     "apk add --no-cache chromium >/dev/null && npm i -s puppeteer-core@24 >/dev/null && node conferir_celular.mjs"
//
// Sai com 1 se alguma tela tiver erro, ficar sem título ou rolar para o lado. Prints em /w/cel-*.png.
import puppeteer from 'puppeteer-core'

const base = process.env.BASE || 'http://host.docker.internal:8080/'
const telas = (process.env.TELAS || '#/hoje,#/trilhas,#/trilhas/1,#/trilhas/1/etapas/1,#/documentos,#/design').split(',')

const navegador = await puppeteer.launch({ executablePath: '/usr/bin/chromium-browser', args: ['--no-sandbox'] })
let falhou = false
for (const [i, hash] of telas.entries()) {
  const p = await navegador.newPage()
  await p.emulate({
    viewport: { width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
    userAgent: 'Mozilla/5.0 (Linux; Android 14) Mobile',
  })
  const erros = []
  p.on('pageerror', (e) => erros.push(e.message))
  p.on('console', (m) => m.type() === 'error' && erros.push(m.text()))
  await p.goto(base + hash, { waitUntil: 'networkidle0' })
  await new Promise((r) => setTimeout(r, 500))
  const titulo = await p.$eval('h1', (e) => e.textContent.trim()).catch(() => '')
  const largura = await p.evaluate(() => document.documentElement.scrollWidth)
  await p.screenshot({ path: `/w/cel-${i}.png` })
  const ok = titulo && largura <= 390 && !erros.length
  falhou ||= !ok
  console.log(`${ok ? 'ok   ' : 'FALHA'} ${hash} | título: ${titulo || '(nenhum)'} | largura: ${largura} | erros: ${erros.join(' || ') || 'nenhum'}`)
}
await navegador.close()
process.exit(falhou ? 1 : 0)
