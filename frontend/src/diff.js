// Diferença por linha entre duas versões de um texto (maior subsequência comum). Devolve trechos
// { tipo: 'igual' | 'mais' | 'menos', linhas: [...] }, na ordem do texto novo. Trechos iguais
// longos ficam recolhidos pela tela; aqui só se agrupa.
export function diferenca(antes, depois) {
  const a = (antes || '').split('\n')
  const b = (depois || '').split('\n')
  const n = a.length
  const m = b.length
  // tabela[i][j] = tamanho da maior subsequência comum de a[i:] e b[j:]
  const tabela = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      tabela[i][j] = a[i] === b[j] ? tabela[i + 1][j + 1] + 1 : Math.max(tabela[i + 1][j], tabela[i][j + 1])
    }
  }
  const trechos = []
  const por = (tipo, linha) => {
    const ultimo = trechos.at(-1)
    if (ultimo && ultimo.tipo === tipo) ultimo.linhas.push(linha)
    else trechos.push({ tipo, linhas: [linha] })
  }
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      por('igual', a[i])
      i++
      j++
    } else if (tabela[i + 1][j] >= tabela[i][j + 1]) {
      por('menos', a[i++])
    } else {
      por('mais', b[j++])
    }
  }
  while (i < n) por('menos', a[i++])
  while (j < m) por('mais', b[j++])
  return trechos
}

// resumoDiferenca: quantas linhas entraram e saíram.
export function resumoDiferenca(trechos) {
  let mais = 0
  let menos = 0
  for (const t of trechos) {
    if (t.tipo === 'mais') mais += t.linhas.length
    if (t.tipo === 'menos') menos += t.linhas.length
  }
  return { mais, menos }
}
