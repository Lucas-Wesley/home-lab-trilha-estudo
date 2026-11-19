// Nomes em português para os valores da API.
export const STATUS = {
  disponivel: { nome: 'A fazer', cor: '' },
  em_andamento: { nome: 'Em andamento', cor: 'azul' },
  concluida: { nome: 'Concluída', cor: 'verde' },
  pulada: { nome: 'Pulada', cor: '' },
}

export const ESTADOS = {
  rascunho: { nome: 'Rascunho', cor: 'ambar' },
  ativa: { nome: 'Ativa', cor: 'azul' },
  pausada: { nome: 'Pausada', cor: '' },
  concluida: { nome: 'Concluída', cor: 'verde' },
}

export const PROFUNDIDADES = {
  1: 'Panorama',
  2: 'Essencial',
  3: 'Sólido',
  4: 'Aprofundado',
  5: 'Deep dive',
}

export const BLOOM = {
  lembrar: 'Lembrar',
  entender: 'Entender',
  aplicar: 'Aplicar',
  analisar: 'Analisar',
  avaliar: 'Avaliar',
  criar: 'Criar',
}

export const ESTADOS_FONTE = {
  na_fila: { nome: 'na fila', cor: '' },
  lendo: { nome: 'lendo…', cor: 'azul' },
  pronta: { nome: 'pronta', cor: 'verde' },
  fragmentando: { nome: 'fragmentando…', cor: 'azul' },
  fragmentada: { nome: 'fragmentada', cor: 'verde' },
  erro: { nome: 'erro', cor: 'vermelho' },
}

// T37: os tipos de fragmento (opinião e fora do tema em tom mais fraco).
export const TIPOS_FRAGMENTO = {
  conteudo: { nome: 'Conteúdo', cor: 'azul' },
  opiniao: { nome: 'Opinião', cor: '' },
  fora: { nome: 'Fora do tema', cor: '' },
}

export const TIPOS_MATERIAL = {
  documento: 'Documento',
  web: 'Web',
  livro: 'Livro',
  acervo: 'Acervo',
  streaming: 'Streaming',
}

export const PAPEIS = {
  principal: 'Fonte principal',
  apoio: 'Apoio',
  aprofundar: 'Para aprofundar',
}

export const DIAS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']

export function duracao(minutos) {
  if (!minutos) return '—'
  const h = Math.floor(minutos / 60)
  const m = minutos % 60
  if (!h) return `${m} min`
  return m ? `${h} h ${m} min` : `${h} h`
}

export function dataCurta(iso) {
  if (!iso) return ''
  const [a, m, d] = iso.split('-').map(Number)
  return new Date(a, m - 1, d).toLocaleDateString('pt-BR', { weekday: 'short', day: 'numeric', month: 'short' })
}

export function percentual(p) {
  if (!p || !p.minutos_total) return 0
  return Math.round((100 * p.minutos_feitos) / p.minutos_total)
}
