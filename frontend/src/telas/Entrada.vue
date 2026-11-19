<script setup>
import { computed } from 'vue'
import ListaEntrada from './ListaEntrada.vue'
import Fonte from './Fonte.vue'

// #/entrada · #/entrada/3
const props = defineProps({ rota: { type: Object, required: true } })
const tela = computed(() => {
  const id = props.rota.partes[0]
  if (id) return { comp: Fonte, props: { fonteId: Number(id) } }
  return { comp: ListaEntrada, props: {} }
})
</script>

<template>
  <component :is="tela.comp" v-bind="tela.props" :key="props.rota.partes.join('/')" />
</template>
