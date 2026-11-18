<script setup>
// Revisar o roteiro com IA (T34): a IA relê os tópicos (da trilha inteira ou de um tópico só)
// contra o objetivo, o recorte, os níveis e a profundidade, e propõe mudanças. O Lucas marca o
// que aceita, pede ajuste ou descarta. Num tópico (etapa), fica recolhido até abrir().
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import MudancaRoteiro from './MudancaRoteiro.vue'
import { api } from './api.js'
import { dataCurta } from './rotulos.js'

const props = defineProps({
  trilha: { type: Object, required: true },
  etapa: { type: Object, default: null }, // revisão de um tópico só
})
const emit = defineEmits(['mudou', 'aberta'])

const p = ref(null)
const erro = ref('')
const ocupado = ref(false)
const pedindo = ref(false)
const pedido = ref('')
const marcadas = ref(new Set())
const ajuste = ref('')

const consulta = computed(() => `/api/trilhas/${props.trilha.id}/roteiro${props.etapa ? `?etapa=${encodeURIComponent(props.etapa.chave)}` : ''}`)
async function carregar() {
  try {
    p.value = (await api(consulta.value)).proposta
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
let relogio = 0
const gerando = computed(() => p.value?.estado === 'gerando')
watch(gerando, (g) => {
  clearInterval(relogio)
  if (g) relogio = setInterval(carregar, 5000)
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

// Marcadas no começo: tudo, menos remover (que apaga a etapa).
watch(
  () => p.value?.id,
  () => {
    const ms = p.value?.conteudo?.mudancas || []
    marcadas.value = new Set(ms.map((m, i) => (m.acao === 'remover' ? -1 : i)).filter((i) => i >= 0))
  },
)
function alternar(i) {
  const s = new Set(marcadas.value)
  s.has(i) ? s.delete(i) : s.add(i)
  marcadas.value = s
}

const aberta = computed(() => pedindo.value || ['gerando', 'aguardando', 'falhou'].includes(p.value?.estado))
watch(aberta, (a) => emit('aberta', a), { immediate: true })
function abrir() {
  pedindo.value = true
}
defineExpose({ abrir })

async function acao(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    await fn()
    await carregar()
    return true
  } catch (e) {
    erro.value = e.message
    return false
  } finally {
    ocupado.value = false
  }
}
async function revisar(texto = pedido.value) {
  const corpo = { pedido: texto, ...(props.etapa ? { etapa_chave: props.etapa.chave } : {}) }
  if (await acao(() => api(`/api/trilhas/${props.trilha.id}/roteiro`, { metodo: 'POST', corpo }))) {
    pedindo.value = false
    pedido.value = ''
  }
}
async function aplicar() {
  if (await acao(() => api(`/api/propostas/${p.value.id}/aprovar`, { metodo: 'POST', corpo: { mudancas: [...marcadas.value] } }))) emit('mudou')
}
async function ajustar() {
  if (await acao(() => api(`/api/propostas/${p.value.id}/ajustar`, { metodo: 'POST', corpo: { pedido: ajuste.value } }))) ajuste.value = ''
}
const descartar = () => acao(() => api(`/api/propostas/${p.value.id}/descartar`, { metodo: 'POST' }))
const cancelar = () => p.value?.tarefa && acao(() => api(`/api/tarefas/${p.value.tarefa.id}/cancelar`, { metodo: 'POST' }))
const alvo = computed(() => (props.etapa ? 'este tópico' : 'o roteiro'))
</script>

<template>
  <div v-if="!etapa || aberta" class="pilha revisar" :class="{ caixa: etapa }">
    <div v-if="etapa" class="linha entre">
      <strong>Revisar este tópico com IA</strong>
      <button v-if="pedindo && !['gerando', 'aguardando'].includes(p?.estado)" class="botao discreto" @click="pedindo = false">Fechar</button>
    </div>
    <p v-if="!etapa" class="suave sem-margem">
      A IA relê os {{ trilha.progresso?.total || '' }} tópicos contra o objetivo, o recorte, o nível de partida, o alvo e a profundidade da trilha (os de "Sobre", que você pode ter mudado) e propõe o que editar, criar, mover ou remover. Nada muda até você marcar e aplicar.
    </p>
    <p v-else-if="pedindo" class="suave sem-margem">A IA olha o título, o objetivo, o critério, o nível e o tempo deste tópico (e o material dele, se tiver), contra os níveis da trilha, e propõe mudanças: editar, dividir, mover ou remover.</p>

    <!-- Pedir -->
    <template v-if="!['gerando', 'aguardando'].includes(p?.estado) && (pedindo || !etapa)">
      <div>
        <label class="rotulo" :for="`roteiro-pedido-${etapa?.chave || 'trilha'}`">Pedido para a IA <span class="fraco">(opcional)</span></label>
        <textarea
          :id="`roteiro-pedido-${etapa?.chave || 'trilha'}`"
          v-model="pedido"
          class="campo"
          rows="2"
          :placeholder="etapa ? 'Ex.: acho que está grande demais; quero mais prática no Mac.' : 'Ex.: mudei o nível alvo para SRE; veja se falta observabilidade.'"
        />
      </div>
      <div><button class="botao primario" :disabled="ocupado" @click="revisar()">Revisar {{ alvo }}</button></div>
      <p v-if="p?.estado === 'aprovada'" class="pequeno sem-margem">Última revisão aplicada em {{ dataCurta(p.atualizado_em.slice(0, 10)) }}.</p>
    </template>
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <template v-if="p?.estado === 'gerando'">
      <p class="suave sem-margem">A IA está relendo {{ alvo }}… (leva alguns minutos; a fila faz uma tarefa por vez)</p>
      <div><button class="botao discreto" :disabled="ocupado" @click="cancelar">Cancelar</button></div>
    </template>

    <template v-else-if="p?.estado === 'falhou'">
      <p class="aviso sem-margem">Não deu: {{ p.erro }}</p>
      <div class="linha">
        <button class="botao" :disabled="ocupado" @click="revisar(p.pedido)">Tentar de novo</button>
        <button class="botao discreto" :disabled="ocupado" @click="descartar">Descartar</button>
      </div>
    </template>

    <template v-else-if="p?.estado === 'aguardando' && p.conteudo">
      <div class="linha entre">
        <span class="rotulo sem-margem">Proposta{{ p.versao > 1 ? ` (versão ${p.versao})` : '' }}</span>
        <span v-if="p.pedido" class="pequeno">Pedido: {{ p.pedido }}</span>
      </div>
      <p v-if="p.conteudo.notas" class="suave sem-margem pre">{{ p.conteudo.notas }}</p>
      <p v-if="!p.conteudo.mudancas.length" class="suave sem-margem">Nenhuma mudança: a IA achou que {{ alvo }} está bom.</p>
      <label v-for="(m, i) in p.conteudo.mudancas" :key="i" class="opcao">
        <input type="checkbox" :checked="marcadas.has(i)" @change="alternar(i)" />
        <MudancaRoteiro :m="m" :trilha-id="trilha.id" />
      </label>
      <div class="linha">
        <button v-if="p.conteudo.mudancas.length" class="botao primario" :disabled="ocupado || !marcadas.size" @click="aplicar">Aplicar as marcadas ({{ marcadas.size }})</button>
        <button class="botao discreto" :disabled="ocupado" @click="descartar">{{ p.conteudo.mudancas.length ? 'Descartar' : 'Fechar' }}</button>
      </div>
      <div class="linha ajuste">
        <input v-model="ajuste" class="campo" placeholder="Pedir ajuste: ex.: não divida a 4.1; o tempo do 5.2 está bom" @keydown.enter="ajuste && ajustar()" />
        <button class="botao" :disabled="ocupado || !ajuste" @click="ajustar">Ajustar</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.revisar.caixa { padding: var(--e4) var(--e5); border: 1px solid var(--borda); border-radius: var(--raio); background: var(--superficie); margin-top: var(--e4); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.pre { white-space: pre-line; }
.opcao { display: flex; gap: var(--e2); align-items: flex-start; padding: var(--e2) 0; border-top: 1px solid var(--borda); font-size: 14.5px; line-height: 1.5; }
.opcao input { margin-top: 5px; flex: none; }
.ajuste .campo { flex: 1 1 260px; }
</style>
