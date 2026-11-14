<script setup>
import { ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { enviarDocumentos } from './documentos.js'
import { link } from './rotas.js'

// Documentos próprios da trilha (.md): lista, adicionar e excluir. Eles são leitura das etapas
// (material do tipo "documento") e os links entre eles funcionam dentro da trilha.
const props = defineProps({ trilha: { type: Object, required: true } })
const emit = defineEmits(['mudou'])

const aviso = ref('')
const enviando = ref(false)

async function adicionar(evento) {
  const arquivos = [...evento.target.files]
  evento.target.value = ''
  if (!arquivos.length) return
  enviando.value = true
  const { enviados, erros } = await enviarDocumentos(props.trilha.id, arquivos)
  enviando.value = false
  aviso.value = [...erros, enviados ? `${enviados} documento(s) adicionado(s).` : ''].filter(Boolean).join(' · ')
  emit('mudou')
}

async function excluir(d) {
  if (!confirm(`Excluir o documento "${d.titulo}"? Etapas que apontam para ele ficam com o link quebrado. O histórico fica no git do conteúdo.`)) return
  try {
    await api(`/api/trilhas/${props.trilha.id}/documentos/${d.slug}`, { metodo: 'DELETE' })
    aviso.value = ''
    emit('mudou')
  } catch (e) {
    aviso.value = e.message
  }
}
</script>

<template>
  <div class="pilha">
    <table v-if="trilha.documentos?.length" class="tabela">
      <tbody>
        <tr v-for="d in trilha.documentos" :key="d.slug">
          <td>
            <a class="titulo" :href="link('trilhas', trilha.id, 'documentos', d.slug)">{{ d.titulo }}</a>
            <div class="fraco"><code>{{ d.slug }}</code></div>
          </td>
          <td class="direita">
            <button class="botao discreto" @click="excluir(d)">Excluir</button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="suave sem-margem">Nenhum documento. Se você tem material próprio em Markdown sobre o assunto, adicione aqui: as etapas podem usar como leitura.</p>
    <div class="linha">
      <label class="botao" :class="{ desligado: enviando }">
        <Icone nome="mais" /> {{ enviando ? 'Enviando…' : 'Adicionar documentos .md' }}
        <input type="file" accept=".md,text/markdown" multiple class="so-leitor" :disabled="enviando" @change="adicionar" />
      </label>
      <span class="pequeno">Mandar de novo um arquivo com o mesmo nome substitui o documento.</span>
    </div>
    <p v-if="aviso" class="aviso neutro sem-margem">{{ aviso }}</p>
  </div>
</template>

<style scoped>
label.botao { cursor: pointer; }
label.desligado { opacity: 0.6; pointer-events: none; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
</style>
