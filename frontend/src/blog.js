// Blog (T62): as engrenagens e o que o quadro do item calcula sobre os pedidos.

// As engrenagens, na ordem da tela. O primeiro valor de cada uma é o padrão (o mesmo do servidor).
export const ENGRENAGENS = [
  { nome: 'horizonte', rotulo: 'Horizonte', opcoes: [['atemporal', 'Atemporal'], ['atual', 'Atual (com data)'], ['misto', 'Núcleo atemporal + seção datada']] },
  { nome: 'nivel', rotulo: 'Leitor', opcoes: [['aprofundamento', 'Aprofundamento'], ['intermediario', 'Intermediário'], ['iniciante', 'Iniciante']] },
  {
    nome: 'formato',
    rotulo: 'Formato',
    opcoes: [
      ['livre', 'A IA escolhe'],
      ['fundamentos', 'Ensaio de fundamentos'],
      ['por_dentro', 'Como funciona por dentro'],
      ['decisao', 'Decisão e trade-offs'],
      ['mapa', 'Mapa de um assunto'],
      ['armadilhas', 'Erros e armadilhas'],
      ['principio', 'Princípio aplicado'],
      ['opiniao', 'Opinião fundamentada'],
      ['historia', 'A história de uma ideia'],
    ],
  },
  { nome: 'abrangencia', rotulo: 'Abrangência', soSugerir: true, opcoes: [['reunir', 'Reunir os assuntos do item'], ['um', 'Um assunto por artigo'], ['ambos', 'As duas coisas']] },
  { nome: 'tamanho', rotulo: 'Tamanho', opcoes: [['medio', 'Médio (~2.000 palavras)'], ['curto', 'Curto (~1.000)'], ['longo', 'Longo (3.000+)']] },
  { nome: 'codigo', rotulo: 'Código', opcoes: [['pouco', 'Só o que prova o ponto'], ['nenhum', 'Nenhum'], ['exemplo', 'Com exemplo trabalhado']] },
]

export function opcoesPadrao() {
  const o = { quantas: 3 }
  for (const e of ENGRENAGENS) o[e.nome] = e.opcoes[0][0]
  return o
}

// limparOpcoes: o que veio guardado no navegador, só com valores conhecidos.
export function limparOpcoes(salvas) {
  const o = opcoesPadrao()
  if (!salvas || typeof salvas !== 'object') return o
  for (const e of ENGRENAGENS) if (e.opcoes.some(([v]) => v === salvas[e.nome])) o[e.nome] = salvas[e.nome]
  const n = Number(salvas.quantas)
  if (Number.isInteger(n) && n >= 1 && n <= 5) o.quantas = n
  return o
}

const nomeDa = (e, v) => (e.opcoes.find(([x]) => x === v) || [v, v])[1]

// resumoOpcoes: "atemporal · aprofundamento · a IA escolhe · médio": só o que importa numa linha.
export function resumoOpcoes(o, modo = 'sugerir') {
  const partes = []
  for (const e of ENGRENAGENS) {
    if (e.soSugerir && modo !== 'sugerir') continue
    if (e.nome === 'codigo' && o[e.nome] === 'pouco') continue
    const nome = nomeDa(e, o[e.nome]).replace(/ \(.*\)$/, '')
    partes.push(nome.charAt(0).toLowerCase() + nome.slice(1))
  }
  if (modo === 'sugerir') partes.push(`${o.quantas} ${o.quantas === 1 ? 'sugestão' : 'sugestões'}`)
  return partes.join(' · ')
}

// ideias: as sugestões marcadas como guardadas ou escritas, de todos os pedidos do item (as mais novas primeiro).
export function ideias(pedidos) {
  const lista = []
  for (const p of pedidos || []) {
    if (p.estado !== 'pronta') continue
    for (const [n, m] of Object.entries(p.marcas || {})) {
      if (m.estado === 'descartada') continue
      const s = p.modo === 'topicos' ? null : p.resultado?.sugestoes?.[Number(n)]
      if (p.modo !== 'topicos' && !s) continue
      lista.push({
        pedido: p.id, n: Number(n), estado: m.estado, link: m.link || '', em: m.em,
        titulo: s ? s.titulo : p.titulo, tese: s ? s.tese : p.resultado?.tese || '',
      })
    }
  }
  return lista.sort((a, b) => String(b.em).localeCompare(String(a.em)))
}

export const emAndamento = (p) => p?.estado === 'gerando'

export function fase(p) {
  if (!emAndamento(p)) return ''
  const quem = p.modo === 'topicos' ? 'os tópicos' : 'as sugestões'
  return p.tarefa?.estado === 'rodando' ? `A IA está lendo o item e montando ${quem}…` : 'Na fila da IA…'
}

export const SITUACOES_BLOG = {
  estudado: ['já estudado', 'verde'],
  a_estudar: ['a estudar', 'ambar'],
  fora_do_item: ['fora do item', 'azul'],
}
