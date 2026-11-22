<script setup>
import { computed } from 'vue'
import ListaTrilhas from './ListaTrilhas.vue'
import NovaTrilha from './NovaTrilha.vue'
import Trilha from './Trilha.vue'
import Etapa from './Etapa.vue'
import DocumentoTrilha from './DocumentoTrilha.vue'
import Criacao from './Criacao.vue'
import Painel from './Painel.vue'
import Revisao from './Revisao.vue'

// #/trilhas · #/trilhas/grupo/<nome> · #/trilhas/nova · #/trilhas/3 · #/trilhas/3/criar · #/trilhas/3/painel · #/trilhas/3/etapas/12
// · #/trilhas/3/etapas/12/revisao · #/trilhas/3/documentos/02-wifi
const props = defineProps({ rota: { type: Object, required: true } })
const partes = computed(() => props.rota.partes)
const tela = computed(() => {
  const [a, b, c, d] = partes.value
  if (!a) return { comp: ListaTrilhas, props: {} }
  if (a === 'nova') return { comp: NovaTrilha, props: {} }
  if (a === 'grupo') return { comp: ListaTrilhas, props: { grupo: b || '' } }
  if (b === 'criar') return { comp: Criacao, props: { trilhaId: Number(a) } }
  if (b === 'painel') return { comp: Painel, props: { trilhaId: Number(a) } }
  if (b === 'etapas' && c && d === 'revisao') return { comp: Revisao, props: { trilhaId: Number(a), etapaId: Number(c) } }
  if (b === 'documentos' && c) return { comp: DocumentoTrilha, props: { trilhaId: Number(a), doc: c, ancora: props.rota.ancora } }
  if (b === 'etapas' && c) return { comp: Etapa, props: { trilhaId: Number(a), etapaId: Number(c), ancora: props.rota.ancora } }
  return { comp: Trilha, props: { trilhaId: Number(a), ancora: props.rota.ancora } }
})
</script>

<template>
  <component :is="tela.comp" v-bind="tela.props" :key="partes.join('/')" />
</template>
