import { slugDoArquivo } from './slug.js'

// Transcrição colada (T49): a aula de um curso de fora (sem o vídeo) vira um documento da trilha
// ligado à etapa como fonte principal; a lição (o "texto da aula") sai dele, como proposta.

export const slugDaTranscricao = (chave) => slugDoArquivo(`transcricao-${chave}`)

// O documento: título para a lista de documentos + o texto como veio.
export function documentoDaTranscricao(titulo, texto) {
  return `# Transcrição: ${titulo}\n\n${texto.trim()}\n`
}

// O corpo do PUT da etapa com a transcrição entre as fontes (uma vez só, a primeira da lista).
// Sem "licao": o servidor mantém a que já existe.
export function etapaComTranscricao(e, titulo) {
  const slug = slugDaTranscricao(e.chave)
  const outros = (e.materiais || []).filter((m) => !(m.tipo === 'documento' && m.referencia === slug))
  return {
    titulo: e.titulo,
    objetivo: e.objetivo,
    bloom: e.bloom,
    profundidade: e.profundidade,
    tipo: e.tipo,
    minutos: e.minutos,
    criterio: e.criterio,
    foco: e.foco || '',
    requer: [...(e.requer || [])],
    materiais: [
      { tipo: 'documento', titulo: `Transcrição: ${titulo}`, referencia: slug, localizador: '', papel: 'principal', nota: 'Transcrição colada da aula.' },
      ...outros,
    ],
  }
}

export const temTranscricao = (e) => (e?.materiais || []).some((m) => m.tipo === 'documento' && m.referencia === slugDaTranscricao(e.chave))
