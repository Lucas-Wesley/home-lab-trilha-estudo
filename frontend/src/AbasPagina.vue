<script setup>
// Abas de uma página (etapa, trilha): barra em pílula presa no topo ao rolar, para trocar de
// parte de qualquer lugar. Cada aba: { id, nome, conta?, feito? }.
import { ref } from 'vue'

defineProps({
  abas: { type: Array, required: true },
  modelValue: { type: String, required: true },
  rotulo: { type: String, default: 'Partes da página' },
})
const emit = defineEmits(['update:modelValue'])
const barra = ref(null)
defineExpose({ barra })
</script>

<template>
  <nav ref="barra" class="abas-pagina" role="tablist" :aria-label="rotulo">
    <button
      v-for="a in abas"
      :key="a.id"
      role="tab"
      class="aba"
      :class="{ ativa: modelValue === a.id }"
      :aria-selected="modelValue === a.id"
      @click="emit('update:modelValue', a.id)"
    >
      {{ a.nome }}<span v-if="a.conta" class="conta">{{ a.conta }}</span><span v-else-if="a.feito" class="feito" aria-label="pronta">✓</span>
    </button>
  </nav>
</template>

<style scoped>
.abas-pagina {
  position: sticky;
  top: var(--barra);
  z-index: 5;
  display: flex;
  gap: 2px;
  padding: var(--e1);
  overflow-x: auto;
  scrollbar-width: none;
  background: var(--superficie);
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  scroll-margin-top: var(--barra);
}
.abas-pagina::-webkit-scrollbar { display: none; }
.aba {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  padding: 7px 11px;
  border: 0;
  border-radius: var(--raio-p);
  background: transparent;
  color: var(--texto-suave);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
}
.aba:hover { color: var(--texto); background: var(--superficie-2); }
.aba.ativa { background: var(--fundo); color: var(--texto); box-shadow: 0 0 0 1px var(--borda); }
.conta { min-width: 20px; padding: 0 6px; border-radius: 999px; background: var(--superficie-2); color: var(--texto-fraco); font-size: 12px; line-height: 18px; text-align: center; }
.aba.ativa .conta { background: var(--superficie); }
.feito { color: var(--verde); font-size: 12px; }
</style>
