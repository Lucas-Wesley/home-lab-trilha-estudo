<script setup>
import { computed } from 'vue'
import ListaTrilhas from './ListaTrilhas.vue'
import NovaTrilha from './NovaTrilha.vue'
import Trilha from './Trilha.vue'
import Etapa from './Etapa.vue'

// #/trilhas · #/trilhas/nova · #/trilhas/3 · #/trilhas/3/etapas/12
const props = defineProps({ rota: { type: Object, required: true } })
const partes = computed(() => props.rota.partes)
const tela = computed(() => {
  const [a, b, c] = partes.value
  if (!a) return { comp: ListaTrilhas, props: {} }
  if (a === 'nova') return { comp: NovaTrilha, props: {} }
  if (b === 'etapas' && c) return { comp: Etapa, props: { trilhaId: Number(a), etapaId: Number(c), ancora: props.rota.ancora } }
  return { comp: Trilha, props: { trilhaId: Number(a), ancora: props.rota.ancora } }
})
</script>

<template>
  <component :is="tela.comp" v-bind="tela.props" :key="partes.join('/')" />
</template>
