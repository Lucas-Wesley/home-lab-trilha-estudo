#!/usr/bin/env node
// Um "claude -p" de mentira, para testar o fluxo de propostas sem gastar a assinatura nem precisar
// de token (CLAUDE_BIN=/caminho/claude-falso.mjs). Lê o pedido.md da pasta da tarefa como o Claude
// leria e devolve, no mesmo formato do --output-format json, uma lição curta que mostra o que
// recebeu: o título da etapa, o pedido e os comentários a responder.
import { existsSync, readFileSync } from 'node:fs'

await new Promise((r) => setTimeout(r, Number(process.env.CLAUDE_FALSO_ESPERA || 4000)))

const pedido = readFileSync('pedido.md', 'utf8')
const titulo = (pedido.match(/^# Pedido: lição da etapa "(.*)"$/m) || [])[1] || 'Etapa'
const deste = (pedido.match(/^- Deste pedido: (.*)$/m) || [])[1] || ''
const comentarios = [...pedido.matchAll(/^- Comentário (\d+)(?: \(em [^)]*\))?(?:, sobre o trecho: «[\s\S]*?»)?: (.*)$/gm)].map((m) => ({ id: Number(m[1]), texto: m[2] }))
const anterior = existsSync('versao-anterior.md') ? readFileSync('versao-anterior.md', 'utf8') : ''
const versao = anterior ? (anterior.match(/Versão de teste (\d+)/) || [, 1])[1] * 1 + 1 : 1

const licao = `# ${titulo}

> **Objetivo:** lição de teste, escrita pelo Claude de mentira (scripts/claude-falso.mjs).
> Prioridade: 🔴 precisa saber · 🟡 bom saber · ⚪ curiosidade

## Conceitos anteriores

<details><summary>Endereço IP · base</summary>

Explicação curta de teste.

</details>

## O que esta versão recebeu

🔴 Versão de teste ${versao}. Pedido: ${deste}.

${comentarios.map((c) => `- Ajuste feito para o comentário ${c.id}: ${c.texto}`).join('\n') || 'Sem comentários.'}

## Teste seu entendimento

1. Por que esta lição existe?

<details><summary>Resposta</summary>

Para testar o fluxo de propostas.

</details>

## Referências

- [oficial] [RFC 1918](https://www.rfc-editor.org/rfc/rfc1918)
- [artigo] [Um link inventado](https://exemplo-que-nao-existe.invalid/pagina)
`

const saida = {
  type: 'result',
  is_error: false,
  result: '',
  structured_output: {
    licao,
    notas: 'Lição de teste. Nenhum fato foi conferido.',
    respostas: comentarios.map((c) => ({ comentario: c.id, resposta: `Mudei a seção "O que esta versão recebeu" para atender: ${c.texto}` })),
  },
}
process.stdout.write(JSON.stringify(saida))
