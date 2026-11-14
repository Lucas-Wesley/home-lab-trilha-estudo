<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { renderizar } from './markdown.js'

// Texto de leitura em Markdown: títulos com âncora (para o "nesta página") e links entre
// documentos que continuam funcionando dentro do app.
const props = defineProps({
  texto: { type: String, default: '' },
  ancora: { type: String, default: '' },
  // "#/trilhas/3/documentos/": para onde vão os links a outros .md (documentos da trilha)
  baseDocs: { type: String, default: '' },
})
const emit = defineEmits(['titulos'])

const caixa = ref(null)
const resultado = computed(() => renderizar(props.texto, { baseDocs: props.baseDocs }))

function irPara(id, suave = true) {
  const alvo = id && caixa.value?.querySelector(`[id="${CSS.escape(id)}"]`)
  if (alvo) alvo.scrollIntoView({ behavior: suave ? 'smooth' : 'auto', block: 'start' })
}

watch(
  resultado,
  async (r) => {
    emit('titulos', r.titulos)
    await nextTick()
    if (props.ancora) irPara(props.ancora, false)
  },
  { immediate: true },
)
watch(() => props.ancora, (a) => irPara(a))

// Âncora da própria página (#secao) não pode trocar a rota (as rotas ficam no hash).
function aoClicar(evento) {
  const link = evento.target.closest('a[data-ancora]')
  if (!link) return
  evento.preventDefault()
  irPara(link.dataset.ancora)
}

defineExpose({ irPara })
</script>

<template>
  <article ref="caixa" class="leitura" @click="aoClicar" v-html="resultado.html" />
</template>
