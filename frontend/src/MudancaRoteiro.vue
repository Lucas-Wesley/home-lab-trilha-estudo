<script setup>
// Uma mudança proposta no roteiro (T34): editar (o antes e o depois de cada campo), novo, mover ou
// remover. Usada na revisão do roteiro e nas revisões do encaixe.
import { computed } from 'vue'
import { link } from './rotas.js'
import { BLOOM } from './rotulos.js'

const props = defineProps({
  m: { type: Object, required: true },
  trilhaId: { type: Number, required: true },
})

const ACOES = { editar: 'Editar', novo: 'Novo', mover: 'Mover', remover: 'Remover' }
const COR = { editar: 'azul', novo: 'verde', mover: '', remover: 'vermelho' }
const campos = computed(() => {
  const m = props.m
  const a = m.atual || {}
  if (m.acao !== 'editar') return []
  return [
    m.titulo && { nome: 'Título', antes: a.titulo, depois: m.titulo },
    m.objetivo && { nome: 'Objetivo', antes: a.objetivo, depois: m.objetivo },
    m.criterio && { nome: 'Critério', antes: a.criterio, depois: m.criterio },
    m.bloom && { nome: 'Bloom', antes: BLOOM[a.bloom] || '—', depois: BLOOM[m.bloom] },
    m.minutos && { nome: 'Tempo', antes: `${a.minutos} min`, depois: `${m.minutos} min` },
  ].filter(Boolean)
})
const onde = computed(() => `em "${props.m.modulo}"${props.m.modulo_novo ? ' (módulo novo)' : ''}${props.m.depois_de ? `, depois de \`${props.m.depois_de}\`` : ', no fim do módulo'}`)
</script>

<template>
  <span class="mudanca">
    <span class="topo">
      <span class="etiqueta" :class="COR[m.acao]">{{ ACOES[m.acao] }}</span>
      <template v-if="m.acao === 'novo'">
        <strong>{{ m.titulo }}</strong>
        <span class="fraco">{{ onde }} · {{ BLOOM[m.bloom] }} · {{ m.minutos }} min</span>
      </template>
      <template v-else>
        <strong class="num">{{ m.numero }}</strong>
        <a :href="link('trilhas', trilhaId, 'etapas', m.atual?.id)" target="_blank">{{ m.atual?.titulo }}</a>
        <span v-if="m.acao === 'mover'" class="fraco">→ {{ onde }}</span>
      </template>
    </span>
    <span v-for="c in campos" :key="c.nome" class="campo-mudado">
      <span class="rotulo-campo">{{ c.nome }}</span>
      <span class="antes">{{ c.antes || '—' }}</span>
      <span class="depois">{{ c.depois }}</span>
    </span>
    <template v-if="m.acao === 'novo'">
      <span class="bloco">{{ m.objetivo }}</span>
      <span v-if="m.criterio" class="bloco fraco">Para concluir: {{ m.criterio }}</span>
    </template>
    <span v-if="m.acao === 'remover' && (m.atual?.tem_licao || ['concluida', 'em_andamento'].includes(m.atual?.status))" class="bloco alerta">
      Atenção: esta etapa {{ m.atual?.tem_licao ? 'tem lição' : '' }}{{ m.atual?.tem_licao && m.atual?.status !== 'disponivel' ? ' e ' : '' }}{{ { concluida: 'já foi concluída', em_andamento: 'está em andamento' }[m.atual?.status] || '' }}; remover apaga tudo dela.
    </span>
    <span v-if="m.acao === 'editar' && m.atual?.tem_licao && (m.objetivo || m.titulo)" class="bloco fraco">A etapa já tem lição: com o objetivo novo, vale pedir mudança na lição.</span>
    <span class="bloco fraco motivo">{{ m.motivo }}</span>
  </span>
</template>

<style scoped>
.mudanca { display: grid; gap: 4px; min-width: 0; }
.topo { display: flex; flex-wrap: wrap; align-items: baseline; gap: 6px var(--e2); }
.num { font-variant-numeric: tabular-nums; }
.campo-mudado { display: grid; grid-template-columns: 72px 1fr; gap: 0 var(--e2); font-size: 13.5px; line-height: 1.5; }
.rotulo-campo { grid-row: span 2; color: var(--texto-fraco); }
.antes { color: var(--texto-fraco); text-decoration: line-through; text-decoration-color: var(--borda-forte); }
.depois { color: var(--texto); }
.bloco { display: block; font-size: 13.5px; line-height: 1.5; color: var(--texto-suave); }
.motivo { font-style: italic; }
.alerta { color: var(--vermelho); }
@media (max-width: 599px) {
  .campo-mudado { grid-template-columns: 1fr; }
  .rotulo-campo { grid-row: auto; }
}
</style>
