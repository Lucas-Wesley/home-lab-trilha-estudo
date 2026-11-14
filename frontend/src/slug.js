// Nome de documento a partir do nome do arquivo, no mesmo formato que o servidor aceita
// ("03-Redes separadas.md" → "03-redes-separadas").
export function slugDoArquivo(nome) {
  return nome
    .replace(/\.md$/i, '')
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}
