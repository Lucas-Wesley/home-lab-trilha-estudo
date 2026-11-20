<script setup>
import { computed } from 'vue'
import ListaEntrada from './ListaEntrada.vue'
import Fonte from './Fonte.vue'
import MeusArquivos from './MeusArquivos.vue'
import CursosPlataforma from './CursosPlataforma.vue'

// #/entrada · #/entrada/3 · #/entrada/arquivos/<caminho da pasta> (T41) · #/entrada/cursos (T50)
const props = defineProps({ rota: { type: Object, required: true } })
const tela = computed(() => {
  const [primeira, ...resto] = props.rota.partes
  if (primeira === 'cursos') return { comp: CursosPlataforma, props: {} }
  if (primeira === 'arquivos') return { comp: MeusArquivos, props: { pasta: resto.join('/') } }
  if (primeira) return { comp: Fonte, props: { fonteId: Number(primeira) } }
  return { comp: ListaEntrada, props: {} }
})
</script>

<template>
  <component :is="tela.comp" v-bind="tela.props" :key="props.rota.partes.join('/')" />
</template>
