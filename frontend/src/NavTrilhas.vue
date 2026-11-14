<script setup>
import { onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { link } from './rotas.js'

// Navegação lateral da seção Trilhas: dentro de uma trilha, os módulos e as etapas (com o status);
// fora, a lista das trilhas.
const props = defineProps({
  trilha: { type: Object, default: null },
  etapaAtual: { type: Number, default: 0 },
  nova: { type: Boolean, default: false },
})

const trilhas = ref([])
onMounted(async () => {
  if (props.trilha) return
  try {
    trilhas.value = await api('/api/trilhas')
  } catch {
    trilhas.value = []
  }
})

const ICONE_STATUS = { concluida: '✓', pulada: '–', em_andamento: '●', disponivel: '' }
</script>

<template>
  <template v-if="trilha">
    <div class="nav-grupo">
      <a class="nav-item voltar" :href="link('trilhas')"><Icone nome="seta-direita" :tamanho="14" class="vira" /> Todas as trilhas</a>
      <a class="nav-item" :class="{ ativo: !etapaAtual }" :href="link('trilhas', trilha.id)"><strong>{{ trilha.titulo }}</strong></a>
    </div>
    <div v-for="(m, i) in trilha.modulos" :key="m.id" class="nav-grupo">
      <span class="nav-titulo">{{ i + 1 }}. {{ m.titulo }}</span>
      <a
        v-for="e in m.etapas"
        :key="e.id"
        class="nav-item"
        :class="{ ativo: etapaAtual === e.id, feita: e.status === 'concluida' || e.status === 'pulada' }"
        :href="link('trilhas', trilha.id, 'etapas', e.id)"
      >
        <span class="marca-status" :class="e.status">{{ ICONE_STATUS[e.status] }}</span>
        <span>{{ e.titulo }}</span>
      </a>
      <p v-if="!m.etapas.length" class="nav-vazio">Sem etapas.</p>
    </div>
  </template>
  <template v-else>
    <div class="nav-grupo">
      <span class="nav-titulo">Trilhas</span>
      <a class="nav-item" :class="{ ativo: !nova }" :href="link('trilhas')">Todas</a>
      <a class="nav-item" :class="{ ativo: nova }" :href="link('trilhas', 'nova')">Nova trilha</a>
    </div>
    <div v-if="trilhas.length" class="nav-grupo">
      <span class="nav-titulo">Suas trilhas</span>
      <a v-for="t in trilhas" :key="t.id" class="nav-item" :href="link('trilhas', t.id)">{{ t.titulo }}</a>
    </div>
  </template>
</template>

<style scoped>
.voltar { color: var(--texto-fraco); font-size: 13px; }
.vira { transform: rotate(180deg); }
.marca-status {
  flex: none;
  display: inline-grid;
  place-items: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--borda-forte);
  border-radius: 50%;
  font-size: 10px;
  line-height: 1;
}
.marca-status.concluida { background: var(--verde-fundo); border-color: var(--verde); color: var(--verde); }
.marca-status.em_andamento { border-color: var(--azul); color: var(--azul); font-size: 8px; }
.marca-status.pulada { color: var(--texto-fraco); }
.nav-item.feita:not(.ativo) { color: var(--texto-fraco); }
.nav-item { align-items: flex-start; }
.nav-item .marca-status { margin-top: 2px; }
</style>
