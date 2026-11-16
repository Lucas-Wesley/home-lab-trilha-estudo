<script setup>
// Meu resumo (T30): no fim da lição, o Lucas escreve o que entendeu (salvo sozinho). "Conferir
// com a aula" manda para a IA, que compara com a transcrição: o que bate, o que está errado e o
// que faltou, com o minuto para rever no vídeo.
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { textoComCodigo } from './markdown.js'
import { VEREDITOS, tempoParaSegundos } from './videoaula.js'

const props = defineProps({
  etapaId: { type: Number, required: true },
  temVideo: { type: Boolean, default: false },
})
const emit = defineEmits(['ir'])

const resumo = ref('')
const conferencias = ref([])
const estado = ref('')
const aviso = ref('')
const enviando = ref(false)

const conferindo = computed(() => conferencias.value.find((c) => c.estado === 'corrigindo'))
const ultima = computed(() => conferencias.value.find((c) => c.estado !== 'corrigindo'))
const resultado = computed(() => (ultima.value?.estado === 'corrigida' ? ultima.value.resultado : null))
const palavras = computed(() => (resumo.value.trim() ? resumo.value.trim().split(/\s+/).length : 0))
// O resumo mudou depois da última conferência?
const mudouDesde = computed(() => {
  if (!ultima.value) return true
  return ultima.value.respostas?.resumo !== resumo.value.trim()
})

let carregado = false
let poll = null
async function carregar() {
  try {
    const an = await api(`/api/etapas/${props.etapaId}/anotacoes`)
    if (!carregado) resumo.value = an.resumo
    conferencias.value = an.conferencias
  } catch (err) {
    aviso.value = err.message
  }
  carregado = true
  clearInterval(poll)
  poll = conferindo.value ? setInterval(carregar, 4000) : null
}
onMounted(carregar)

let relogio = null
watch(resumo, () => {
  if (!carregado) return
  estado.value = 'salvando'
  clearTimeout(relogio)
  relogio = setTimeout(salvar, 900)
})
async function salvar() {
  clearTimeout(relogio)
  relogio = null
  try {
    const texto = resumo.value
    await api(`/api/etapas/${props.etapaId}/anotacoes`, { metodo: 'PUT', corpo: { resumo: texto } })
    if (resumo.value === texto) estado.value = 'salvo'
  } catch {
    estado.value = 'erro'
  }
}
onBeforeUnmount(() => {
  clearInterval(poll)
  if (relogio) salvar()
})

async function conferir() {
  aviso.value = ''
  enviando.value = true
  try {
    clearTimeout(relogio)
    relogio = null
    await api(`/api/etapas/${props.etapaId}/resumo/conferir`, { metodo: 'POST', corpo: { resumo: resumo.value } })
    estado.value = 'salvo'
    await carregar()
  } catch (err) {
    aviso.value = err.message
  } finally {
    enviando.value = false
  }
}

// "onde" da IA: "03:10" ou "1:02:05" vira um botão que leva ao ponto do vídeo.
function irPara(onde) {
  const s = tempoParaSegundos(onde)
  if (s !== null) emit('ir', 1, s)
}
const temTempo = (onde) => props.temVideo && tempoParaSegundos(onde) !== null
</script>

<template>
  <section id="resumo" class="secao">
    <div class="linha entre">
      <h2>Meu resumo</h2>
      <span class="pequeno">{{ { salvando: 'Salvando…', salvo: 'Salvo', erro: 'Não salvou: tente de novo' }[estado] || '' }}</span>
    </div>
    <p class="suave">
      Com as suas palavras, o que você entendeu{{ temVideo ? ' da aula' : ' da etapa' }}. Depois, confira: a IA compara com {{ temVideo ? 'a transcrição da aula' : 'a lição' }} e diz o que bate, o que está errado e o que faltou{{ temVideo ? ', com o minuto para rever' : '' }}.
    </p>
    <textarea v-model="resumo" class="campo resumo" rows="9" spellcheck="true" placeholder="Escreva o resumo aqui…" @blur="relogio && salvar()" />
    <div class="linha entre">
      <div class="linha">
        <button class="botao primario" :disabled="enviando || !!conferindo || palavras < 8" @click="conferir">
          {{ conferindo ? 'Conferindo…' : ultima && !mudouDesde ? 'Conferir de novo' : temVideo ? 'Conferir com a aula' : 'Conferir com a lição' }}
        </button>
        <span v-if="conferindo" class="pequeno">A IA está lendo {{ temVideo ? 'a aula' : 'a lição' }} e o seu resumo; leva um ou dois minutos.</span>
      </div>
      <span class="pequeno">{{ palavras }} palavra(s)</span>
    </div>
    <p v-if="aviso" class="aviso">{{ aviso }}</p>
    <p v-if="ultima?.estado === 'falhou'" class="aviso">A conferência falhou: {{ ultima.erro }}. Tente de novo.</p>

    <div v-if="resultado" class="cartao cheio pilha conferencia">
      <div class="linha">
        <span class="etiqueta" :class="VEREDITOS[resultado.veredito]?.cor">{{ VEREDITOS[resultado.veredito]?.nome }}</span>
        <span class="etiqueta">{{ ultima.nota }}/100</span>
        <span v-if="mudouDesde" class="pequeno">do resumo como estava ao conferir</span>
      </div>
      <p class="sem-margem" v-html="textoComCodigo(resultado.comentario)" />
      <div v-if="resultado.erros?.length">
        <span class="rotulo">A corrigir</span>
        <ul class="itens">
          <li v-for="(x, i) in resultado.erros" :key="i">
            <div class="citado">“{{ x.trecho }}”</div>
            <div><span v-html="textoComCodigo(x.correcao)" /> <button v-if="temTempo(x.onde)" class="botao discreto tempo" @click="irPara(x.onde)">{{ x.onde }}</button></div>
          </li>
        </ul>
      </div>
      <div v-if="resultado.faltou?.length">
        <span class="rotulo">Faltou</span>
        <ul class="itens">
          <li v-for="(x, i) in resultado.faltou" :key="i">
            <span v-html="textoComCodigo(x.ponto)" /> <button v-if="temTempo(x.onde)" class="botao discreto tempo" @click="irPara(x.onde)">{{ x.onde }}</button>
          </li>
        </ul>
      </div>
      <div v-if="resultado.certos?.length">
        <span class="rotulo">Certo</span>
        <ul class="itens">
          <li v-for="(x, i) in resultado.certos" :key="i" v-html="textoComCodigo(x)" />
        </ul>
      </div>
    </div>
  </section>
</template>

<style scoped>
.secao { margin-top: var(--e7); }
.secao > .linha:first-child h2 { margin: 0; }
.entre { justify-content: space-between; }
.resumo { margin: var(--e3) 0; min-height: 180px; line-height: 1.6; }
.conferencia { margin-top: var(--e4); }
.sem-margem { margin: 0; }
.itens { margin: 0; padding-left: var(--e5); display: grid; gap: var(--e2); font-size: 15px; line-height: 1.55; }
.citado { color: var(--texto-suave); font-style: italic; }
.tempo { font-family: var(--fonte-mono); font-size: 12.5px; padding: 0 var(--e2); min-height: 24px; }
</style>
