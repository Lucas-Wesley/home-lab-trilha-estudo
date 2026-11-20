#!/usr/bin/env node
// Um "claude -p" de mentira, para testar o fluxo de propostas sem gastar a assinatura nem precisar
// de token (CLAUDE_BIN=/caminho/claude-falso.mjs). Lê o pedido.md da pasta da tarefa como o Claude
// leria e devolve, no mesmo formato do --output-format json, uma lição curta que mostra o que
// recebeu: o título da etapa, o pedido e os comentários a responder.
import { existsSync, readFileSync, readdirSync } from 'node:fs'

await new Promise((r) => setTimeout(r, Number(process.env.CLAUDE_FALSO_ESPERA || 4000)))

const pedido = readFileSync('pedido.md', 'utf8')
const comentariosDoPedido = [...pedido.matchAll(/^- Comentário (\d+)(?: \(em [^)]*\))?(?:, sobre o trecho: «[\s\S]*?»)?: (.*)$/gm)].map((m) => ({ id: Number(m[1]), texto: m[2] }))
const responder = (dados) => {
  const respostas = comentariosDoPedido.map((c) => ({ comentario: c.id, resposta: `Ajuste de teste para: ${c.texto}` }))
  process.stdout.write(JSON.stringify({ type: 'result', is_error: false, result: '', structured_output: { ...dados, respostas } }))
  process.exit(0)
}

// Criação com IA (fase 4): uma resposta de teste para cada passo.
const passo = (pedido.match(/^# Pedido: (entrevista|mapa do assunto|validação da lista|espinha dorsal do material|plano da trilha)$/m) || [])[1]
if (passo === 'entrevista') {
  const rodada = Number((pedido.match(/rodada (\d+) de no máximo/) || [, 1])[1])
  responder({
    perfil: 'tecnologia', eixos: ['objetivo', 'nível atual'],
    panorama: rodada === 1 ? 'Panorama de teste: o assunto se divide em **fundamentos**, **serviços** e **operação**.' : '',
    perguntas: [
      { id: `q${rodada}a`, texto: `Pergunta de teste ${rodada}A: para que você quer isto?`, por_que: 'muda o foco do mapa', opcoes: ['trabalho', 'certificação', 'curiosidade'], multipla: false },
      { id: `q${rodada}b`, texto: `Pergunta de teste ${rodada}B: que partes interessam?`, por_que: 'muda o recorte', opcoes: ['fundamentos', 'serviços', 'operação'], multipla: true },
    ],
    suficiente: rodada > 1, notas: 'Entrevista de teste (scripts/claude-falso.mjs).',
  })
}
if (passo === 'mapa do assunto') {
  const etapa = (chave, titulo, nivel) => ({ chave, titulo, resumo: `Resumo de teste de ${titulo}.`, nivel })
  responder({
    objetivo: 'Ao final, você será capaz de explicar o assunto de teste.',
    mapa: [
      { titulo: 'Fundamentos', descricao: 'a base', etapas: [etapa('o-que-e', 'O que é', 'basico'), etapa('como-funciona', 'Como funciona', 'basico')] },
      { titulo: 'Serviços', descricao: 'o centro', etapas: [etapa('servico-a', 'Serviço A', 'intermediario'), etapa('servico-b', 'Serviço B', 'intermediario')] },
      { titulo: 'Avançado', descricao: 'fora do recorte sugerido', etapas: [etapa('otimizacao', 'Otimização', 'avancado'), etapa('casos', 'Casos-limite', 'avancado')] },
    ],
    recorte_sugerido: ['o-que-e', 'como-funciona', 'servico-a', 'servico-b'],
    por_que_recorte: 'o avançado fica para depois.', nivel_partida: 'iniciante', nivel_alvo: 'intermediário', notas: 'Mapa de teste.',
  })
}
if (passo === 'validação da lista') {
  const itens = [...pedido.matchAll(/^\d+\. (.*)$/gm)].map((m, i) => ({ chave: `item-${i + 1}`, titulo: m[1], nota: '' }))
  responder({
    itens,
    sugestoes: [
      { id: 's1', acao: 'inserir', antes_de: itens.at(-1)?.chave || '', titulo: 'Pré-requisito de teste', motivo: 'o último item supõe isto.' },
      { id: 's2', acao: 'observar', item: itens[0]?.chave || '', titulo: 'Item amplo', motivo: 'talvez seja grande para uma sessão.' },
    ],
    nivel_partida: 'iniciante', nivel_alvo: 'intermediário', notas: 'Validação de teste.',
  })
}
if (passo === 'espinha dorsal do material') {
  // Fase 6: um item por capítulo do sumário (lido de material/livro-N-sumario.md) ou por Tema.
  const sumarios = (existsSync('material') ? readdirSync('material', { withFileTypes: true }) : []).filter((f) => f.isFile() && /sumario\.md$/.test(f.name))
  const itens = []
  for (const f of sumarios) {
    for (const m of readFileSync(`material/${f.name}`, 'utf8').matchAll(/^\| (\d+) \| ([^|]+) \|/gm)) {
      itens.push({ chave: `cap-${m[1]}`, titulo: m[2].trim(), nota: '', fonte: `cap. ${m[1]}` })
    }
  }
  for (const m of pedido.matchAll(/^- Acervo \*\*(.*)\*\* \(referência `([^`]+)`\)/gm)) itens.push({ chave: `tema-${itens.length + 1}`, titulo: m[1], nota: '', fonte: m[2] })
  if (itens.length < 2) itens.push({ chave: 'extra', titulo: 'Item de teste', nota: '', fonte: 'teste' })
  responder({
    itens,
    sugestoes: [
      { id: 's1', acao: 'inserir', antes_de: itens[0].chave, titulo: 'Base de teste', motivo: 'o material supõe isto.' },
      { id: 's2', acao: 'observar', item: itens.at(-1).chave, titulo: 'Raso para a profundidade', motivo: 'o capítulo é curto.' },
    ],
    nivel_partida: 'iniciante', nivel_alvo: 'intermediário', notas: `Espinha de teste a partir de ${sumarios.length} livro(s).`,
  })
}
if (passo === 'plano da trilha') {
  const chaves = [...pedido.matchAll(/^\s*- (?:✔ )?`([a-z0-9-]+)`/gm)].map((m) => m[1]).slice(0, 4)
  const lista = chaves.length >= 2 ? chaves : ['etapa-1', 'etapa-2']
  const livro = (pedido.match(/^- Livro \*\*.*\*\* \(referência `([^`]+)`\)/m) || [])[1]
  responder({
    trilha: {
      titulo: 'Trilha de teste', objetivo: 'Ao final, você será capaz de explicar o assunto de teste.', descricao: 'Plano de teste.',
      recorte: 'Só o de teste; fica de fora o resto.', nivel_partida: 'iniciante', nivel_alvo: 'intermediário',
      modulos: [{
        titulo: 'Módulo de teste', descricao: '',
        etapas: lista.map((chave, i) => (livro && /^cap-\d+$/.test(chave) ? {
          chave, titulo: `Etapa ${i + 1}`, objetivo: 'Explicar o capítulo de teste.', bloom: 'entender', tipo: 'normal',
          minutos: 45, criterio: 'Responder às perguntas.', requer: i ? [lista[i - 1]] : [],
          materiais: [{ tipo: 'livro', titulo: 'Livro de teste', referencia: livro, localizador: `cap. ${chave.slice(4)}`, papel: 'principal', nota: '' }],
        } : {
          chave, titulo: `Etapa ${i + 1}`, objetivo: 'Explicar a etapa de teste.', bloom: 'entender', tipo: 'normal',
          minutos: i === 0 ? 120 : 30, criterio: 'Responder às perguntas.', requer: i ? [lista[i - 1]] : [],
          materiais: [{ tipo: 'web', titulo: 'RFC 1918', referencia: 'https://www.rfc-editor.org/rfc/rfc1918', localizador: '', papel: 'principal', nota: '[padrão] teste' }],
        })),
      }],
    },
    notas: 'Plano de teste.',
  })
}

// Estudar a etapa (fase 5): exercícios, perguntas do simulado e a correção.
if (/^# Pedido: exercícios da etapa/m.test(pedido)) {
  responder({
    exercicios: [
      { tipo: 'escolha', enunciado: 'Exercício de teste 1: qual é a certa?', alternativas: ['A certa', 'Uma errada plausível', 'Outra errada'], correta: 0, explicacao: 'A primeira é a certa no teste.', conceito: 'conceito A' },
      { tipo: 'escolha', enunciado: 'Exercício de teste 2: e agora?', alternativas: ['Errada', 'Errada também', 'A certa'], correta: 2, explicacao: 'A terceira é a certa.', conceito: 'conceito B' },
      { tipo: 'completar', enunciado: 'Exercício de teste 3: a palavra é ____.', aceitas: ['teste', 'testes'], explicacao: 'A palavra é "teste".', conceito: 'conceito C' },
    ],
    notas: 'Exercícios de teste (scripts/claude-falso.mjs).',
  })
}
if (/^# Pedido: simulado da etapa/m.test(pedido)) {
  const n = (pedido.match(/^\d+\. /gm) || []).length
  responder({
    perguntas: [1, 2].map((i) => ({ pergunta: `Pergunta aberta de teste ${n + i}: por que isto acontece?`, esperado: '- o motivo de teste (essencial)\n- um exemplo', conceito: `conceito ${n + i}` })),
    notas: 'Perguntas de teste.',
  })
}
// Sugerir material (fase 6): um da web, o primeiro livro e o primeiro Tema da lista, e um livro
// inventado (o app tira).
if (/^# Pedido: material da etapa/m.test(pedido)) {
  const livro = existsSync('biblioteca.md') ? (readFileSync('biblioteca.md', 'utf8').match(/^\| `([^`]+)` \| ([^|]+) \|/m) || []) : []
  const tema = existsSync('acervo.md') ? (readFileSync('acervo.md', 'utf8').match(/^- `([^`]+)` — ([^:\n]+)/m) || []) : []
  const materiais = [
    { tipo: 'web', titulo: 'RFC 1918', referencia: 'https://www.rfc-editor.org/rfc/rfc1918', localizador: '', papel: 'apoio', nota: '[padrão] teste', motivo: 'Material de teste da web.' },
    { tipo: 'livro', titulo: 'Livro inventado', referencia: 'nao-existe', localizador: 'cap. 1', papel: 'apoio', nota: '', motivo: 'O app tira.' },
  ]
  if (livro[1]) materiais.push({ tipo: 'livro', titulo: livro[2].trim(), referencia: livro[1], localizador: 'cap. 1', papel: 'principal', nota: '[livro] teste', motivo: 'Livro de teste da Biblioteca.' })
  if (tema[1]) materiais.push({ tipo: 'acervo', titulo: tema[2].trim(), referencia: tema[1], localizador: '', papel: 'aprofundar', nota: '', motivo: 'Tema de teste do Acervo.' })
  responder({ materiais, notas: 'Material de teste (scripts/claude-falso.mjs).' })
}
if (/^# Pedido: correção do simulado/m.test(pedido)) {
  const respostas = [...pedido.matchAll(/^## Pergunta (\d+)(?: \(etapa .*\))?$/gm)].map((m) => Number(m[1]))
  responder({
    correcoes: respostas.map((p) => ({ pergunta: p, nota: p % 2 ? 85 : 50, veredito: p % 2 ? 'certo' : 'parcial', comentario: `Correção de teste da pergunta ${p}.`, faltou: p % 2 ? '' : 'Faltou o exemplo.' })),
    resumo: 'Resumo de teste: revise a seção de teste.',
  })
}

// Fase 7: o reforço (uma etapa antes e uma depois) e o mini-teste de nível.
if (/^# Pedido: reforço da etapa/m.test(pedido)) {
  const etapa = (pedido.match(/^# Pedido: reforço da etapa "(.*)"$/m) || [])[1] || 'Etapa'
  const fracos = [...pedido.matchAll(/^\d+\. \*\*(.*?)\*\*/gm)].map((m) => m[1])
  responder({
    diagnostico: `Diagnóstico de teste: errou ${fracos.length} ponto(s)${fracos.length ? ' (' + fracos.join('; ') + ')' : ''}.`,
    etapas: [
      { titulo: `Base de ${etapa}`, objetivo: 'Entender a base de teste.', bloom: 'entender', minutos: 20, posicao: 'antes', foco: 'Foco de teste: a base.', motivo: 'Motivo de teste (antes).' },
      { titulo: `Prática de ${etapa}`, objetivo: 'Aplicar o conceito de teste.', bloom: 'aplicar', minutos: 30, posicao: 'depois', foco: `Foco de teste: ${fracos[0] || 'o pedido'}.`, motivo: 'Motivo de teste (depois).' },
    ],
    notas: 'Reforço de teste (scripts/claude-falso.mjs).',
  })
}
if (/^# Pedido: mini-teste de nível/m.test(pedido)) {
  const itens = [...pedido.matchAll(/^- `([^`]+)` — /gm)].map((m) => m[1])
  const niveis = ['basico', 'basico', 'basico', 'intermediario', 'intermediario', 'intermediario', 'avancado', 'avancado']
  responder({
    perguntas: niveis.map((nivel, i) => ({
      enunciado: `Pergunta de teste ${i + 1} (${nivel})?`, alternativas: ['Certa', 'Errada A', 'Errada B'], correta: 0,
      explicacao: 'Explicação de teste.', item: itens[Math.min(i, itens.length - 1)] || 'x', nivel,
    })),
  })
}

// Entrada (T37): três fragmentos de teste nos terços das marcas [mm:ss] da transcrição.
if (/^# Pedido: fragmentar/m.test(pedido)) {
  const paraSeg = (t) => { const p = t.split(':').map(Number); return p.length === 3 ? p[0] * 3600 + p[1] * 60 + p[2] : p[0] * 60 + p[1] }
  const paraTempo = (s) => { const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), ss = String(s % 60).padStart(2, '0'); return h ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${String(m).padStart(2, '0')}:${ss}` }
  const transcricao = existsSync('transcricao.md') ? readFileSync('transcricao.md', 'utf8') : ''
  const marcas = [...transcricao.matchAll(/^\[((?:\d+:)?\d{1,2}:\d{2})\]/gm)].map((m) => paraSeg(m[1]))
  const cortes = [0, marcas[Math.floor(marcas.length / 3)] ?? 20, marcas[Math.floor((2 * marcas.length) / 3)] ?? 40, (marcas.at(-1) ?? 50) + 10]
  if (cortes[1] <= 0) cortes[1] = 20
  if (cortes[2] <= cortes[1]) cortes[2] = cortes[1] + 20
  if (cortes[3] <= cortes[2]) cortes[3] = cortes[2] + 20
  responder({
    fragmentos: ['Começo', 'Opinião do autor', 'Fim'].map((titulo, i) => ({
      inicio: paraTempo(cortes[i]), fim: paraTempo(cortes[i + 1]), titulo,
      resumo: `Resumo de teste do fragmento "${titulo}".`, assuntos: ['teste'],
      tipo: i === 1 ? 'opiniao' : 'conteudo',
    })),
    notas: 'Fragmentos de teste (scripts/claude-falso.mjs).',
  })
}

// Entrada (T38): a primeira trilha do pedido para cada fragmento "conteudo" e nenhuma para os
// outros tipos (opinião, fora do tema).
if (/^# Pedido: destino/m.test(pedido)) {
  const trilha = Number((pedido.match(/^- Trilha (\d+) —/m) || [])[1] || 0)
  const destinos = [...pedido.matchAll(/^(\d+)\. \d{1,2}:\d{2}(?::\d{2})?–\d{1,2}:\d{2}(?::\d{2})? — "[^"]+" \((\w+)\)/gm)].map((m) => ({
    fragmento: Number(m[1]),
    trilhas: m[2] === 'conteudo' && trilha ? [{ trilha_id: trilha, motivo: 'Destino de teste.' }] : [],
    trilha_nova: '',
    motivo_nenhuma: m[2] === 'conteudo' ? '' : 'É opinião/fora do tema, não vira aula.',
  }))
  responder({ destinos, notas: 'Destino de teste.' })
}

// Encaixe (T32): um tópico novo por intervalo do localizador ("- Só este trecho: a–b; c–d"), ou
// um encaixe na primeira etapa quando o material não é de trechos (teste do encaixe comum).
if (/^# Pedido: encaixar um material/m.test(pedido)) {
  const trecho = (pedido.match(/^- Só este trecho: (.*)$/m) || [])[1] || ''
  const intervalos = trecho.split(';').map((s) => s.trim()).filter(Boolean)
  const etapa = (pedido.match(/^- \*\*[\d.]+\*\* `([\w-]+)` —/m) || [])[1] || ''
  responder({
    encaixes: etapa ? [{ etapa_chave: etapa, localizador: intervalos[0] || '', papel: 'principal', nota: 'Encaixe de teste.', motivo: 'Teste.' }] : [],
    novos: intervalos.map((iv, i) => ({
      titulo: `Trecho ${iv}`, objetivo: `Explicar o trecho ${iv} de teste.`, bloom: 'entender',
      minutos: 30, modulo: 'Vídeos da Entrada', depois_de: '', localizador: iv,
      nota: 'Trecho de teste.', motivo: 'Teste.',
    })),
    revisoes: [], notas: 'Encaixe de teste.',
  })
}

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
