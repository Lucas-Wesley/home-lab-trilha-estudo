<script setup>
import { computed } from 'vue'
import Icone from './Icone.vue'
import { linkReferencia, progressoReferencia, semTrilha, textoReferencia } from './projeto.js'

// Uma referência do roadmap do Projeto de estudo (T52): link para a etapa/módulo/trilha quando
// casou, com o ✓ de concluída; sem par, só o texto.
const props = defineProps({ r: { type: Object, required: true } })

const href = computed(() => linkReferencia(props.r))
const feita = computed(() => props.r.status === 'concluida')
const progresso = computed(() => progressoReferencia(props.r))
const dica = computed(() => {
  if (!props.r.tipo) return 'Não achei nas trilhas'
  if (props.r.parcial) return `Achei só até aqui: ${props.r.caminho}`
  return feita.value ? `${props.r.caminho} · concluída` : props.r.caminho
})
</script>

<template>
  <a v-if="href" class="ref" :class="{ feita, parcial: r.parcial }" :href="href" :title="dica">
    <Icone v-if="feita" nome="check" :tamanho="14" class="marca-feita" />
    <span>{{ texto }}</span>
    <span v-if="progresso" class="conta">{{ progresso }}</span>
  </a>
  <span v-else class="ref sem-par" :title="dica">{{ curto ? semTrilha(r.texto) || r.texto : r.texto }}</span>
</template>

<style scoped>
.ref {
  display: inline-flex;
  align-items: baseline;
  gap: var(--e1);
  max-width: 100%;
  color: var(--link);
  text-decoration: none;
  overflow-wrap: anywhere;
}
.ref:hover span:first-of-type { text-decoration: underline; }
.ref.feita { color: var(--verde); }
.ref.parcial { color: var(--texto-suave); text-decoration: underline dotted; }
.ref.sem-par { color: var(--texto-suave); }
.marca-feita { flex: none; align-self: center; }
.conta { font-size: 12px; font-variant-numeric: tabular-nums; color: var(--texto-fraco); }
</style>
