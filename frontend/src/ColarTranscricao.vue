<script setup>
import { ref } from 'vue'
import { api } from './api.js'
import { documentoDaTranscricao, etapaComTranscricao, slugDaTranscricao, temTranscricao } from './transcricao.js'

// Colar a transcrição da aula (T49): salva como documento da trilha, liga à etapa como fonte
// principal e avisa a etapa, que pede o texto da aula à IA (proposta, como sempre).
const props = defineProps({
  trilhaId: { type: [Number, String], required: true },
  etapa: { type: Object, required: true },
})
const emit = defineEmits(['salva'])

const aberto = ref(false)
const texto = ref('')
const gerar = ref(true)
const salvando = ref(false)
const erro = ref('')

async function salvar() {
  if (!texto.value.trim()) return
  salvando.value = true
  erro.value = ''
  try {
    const e = props.etapa
    await api(`/api/trilhas/${props.trilhaId}/documentos/${slugDaTranscricao(e.chave)}`, {
      metodo: 'PUT',
      corpo: { conteudo: documentoDaTranscricao(e.titulo, texto.value) },
    })
    const nova = await api(`/api/etapas/${e.id}`, { metodo: 'PUT', corpo: etapaComTranscricao(e, e.titulo) })
    texto.value = ''
    aberto.value = false
    emit('salva', { etapa: nova, gerar: gerar.value })
  } catch (err) {
    erro.value = err.message
  } finally {
    salvando.value = false
  }
}
</script>

<template>
  <div class="colar">
    <button v-if="!aberto" class="botao discreto" type="button" @click="aberto = true">
      {{ temTranscricao(etapa) ? 'Trocar a transcrição da aula' : 'Colar a transcrição da aula' }}
    </button>
    <form v-else class="pilha" @submit.prevent="salvar">
      <label class="rotulo" for="transcricao">Transcrição da aula</label>
      <textarea id="transcricao" v-model="texto" class="campo texto" placeholder="Cole aqui a transcrição (ou o texto) da aula." spellcheck="false" />
      <label class="marcar"><input v-model="gerar" type="checkbox" /> Pedir o texto da aula à IA (organizado e resumido, chega como proposta)</label>
      <div class="linha">
        <button class="botao primario" :disabled="salvando || !texto.trim()">{{ salvando ? 'Salvando…' : 'Salvar' }}</button>
        <button class="botao discreto" type="button" @click="aberto = false">Cancelar</button>
      </div>
      <p class="pequeno">Fica nos documentos da trilha e entra nas fontes da etapa. Colar de novo substitui.</p>
      <p v-if="erro" class="aviso">{{ erro }}</p>
    </form>
  </div>
</template>

<style scoped>
.colar { margin: var(--e3) 0; }
.texto { min-height: 240px; font-size: 13.5px; }
.marcar { display: flex; gap: var(--e2); align-items: center; font-size: 14px; }
</style>
