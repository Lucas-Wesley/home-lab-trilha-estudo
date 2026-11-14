import { api } from './api.js'
import { slugDoArquivo } from './slug.js'

// Envia arquivos .md como documentos da trilha. O nome do arquivo vira o nome do documento (é ele
// que os links entre documentos usam); mandar de novo um arquivo com o mesmo nome substitui.
export async function enviarDocumentos(trilhaId, arquivos) {
  const erros = []
  let enviados = 0
  for (const f of arquivos) {
    try {
      const nome = slugDoArquivo(f.name)
      if (!nome) throw new Error('nome de arquivo sem letras ou números')
      await api(`/api/trilhas/${trilhaId}/documentos/${nome}`, { metodo: 'PUT', corpo: { conteudo: await f.text() } })
      enviados++
    } catch (e) {
      erros.push(`${f.name}: ${e.message}`)
    }
  }
  return { enviados, erros }
}
