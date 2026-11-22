<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { textoComCodigo } from './markdown.js'
import { TIPOS_MATERIAL } from './rotulos.js'
import { emAndamento, fase, mudouDesde, naoVerificados, pedidoDasFaltas } from './verificacao.js'

// Verificar material (T53): a IA lê o material da etapa (o vídeo é transcrito antes, na Entrada, se
// precisar) e diz o que a etapa exige, o que o material cobre, o quanto cobre e o que falta, com a
// fonte primária. Não muda a etapa; o que falta vira um "Sugerir material" com o pedido escrito.
const props = defineProps({
  etapaId: { type: Number, required: true },
  materiais: { type: Array, default: () => [] },
})
const emit = defineEmits(['sugerir'])

const lista = ref([])
const erro = ref('')
const ocupado = ref(false)
const pedido = ref('')
const comPedido = ref(false)

async function carregar() {
  try {
    lista.value = await api(`/api/etapas/${props.etapaId}/verificacoes`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
const ultima = computed(() => lista.value[0] || null)
const anteriores = computed(() => lista.value.slice(1).filter((v) => v.estado === 'pronta'))
const andando = computed(() => emAndamento(ultima.value))
let relogio = 0
watch(andando, (sim) => {
  clearInterval(relogio)
  if (sim) relogio = setInterval(carregar, 4000)
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

async function verificar(indices) {
  ocupado.value = true
  erro.value = ''
  try {
    const corpo = { pedido: pedido.value }
    if (indices?.length) corpo.materiais = indices
    await api(`/api/etapas/${props.etapaId}/verificar`, { metodo: 'POST', corpo })
    pedido.value = ''
    comPedido.value = false
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}
defineExpose({ verificar })

const laudo = computed(() => (ultima.value?.estado === 'pronta' ? ultima.value.resultado : null))
const mudou = computed(() => mudouDesde(ultima.value, props.materiais))
const faltando = computed(() => (laudo.value ? naoVerificados(ultima.value, props.materiais) : []))
const pedidoComplemento = computed(() => pedidoDasFaltas(laudo.value))
const tituloDo = (v, n) => v?.materiais?.[n - 1]?.titulo || `Material ${n}`

const VEREDITOS = {
  suficiente: { nome: 'Basta para estudar', cor: 'verde' },
  complementar: { nome: 'Precisa complementar', cor: 'ambar' },
  insuficiente: { nome: 'Não basta', cor: 'vermelho' },
}
const SITUACOES = { coberto: ['coberto', 'verde'], raso: ['raso', 'ambar'], ausente: ['ausente', 'vermelho'] }
const PROFUNDIDADES = { abaixo: 'abaixo da profundidade da etapa', na_medida: 'na profundidade da etapa', acima: 'acima da profundidade da etapa' }
const TEXTOS = { pronto: 'texto pronto', web: 'a IA abre a página', transcrevendo: 'transcrevendo na Entrada…', sem_texto: 'sem texto' }
const quando = (iso) => (iso ? new Date(iso).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' }) : '')
</script>

<template>
  <div class="verificar pilha">
    <div class="linha entre">
      <span class="rotulo">Este material basta?</span>
      <div class="linha">
        <button v-if="!comPedido && !andando" class="botao discreto" @click="comPedido = true">Com um pedido</button>
        <button class="botao" :disabled="ocupado || andando || !materiais.length" @click="verificar()">
          {{ andando ? 'Verificando…' : ultima ? 'Verificar de novo' : 'Verificar material' }}
        </button>
      </div>
    </div>
    <p v-if="!ultima && !andando" class="suave pequeno sem-margem">
      A IA lê cada material (o vídeo é transcrito antes, se ainda não foi) e diz o quanto ele ensina desta etapa, o que fica raso e o que falta, com a fonte para complementar.
    </p>
    <textarea v-if="comPedido" v-model="pedido" class="campo" rows="2" placeholder="O que conferir com mais cuidado (opcional). Ex.: se cobre o caso do IPv6." />
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <div v-if="andando" class="cartao pilha">
      <p class="sem-margem">{{ fase(ultima) }}</p>
      <ul class="itens">
        <li v-for="m in ultima.materiais" :key="m.indice">
          {{ m.titulo }} <span class="fraco">· {{ TEXTOS[m.texto] }}<template v-if="m.aviso">: {{ m.aviso }}</template></span>
        </li>
      </ul>
    </div>

    <p v-else-if="ultima?.estado === 'falhou'" class="aviso sem-margem">A verificação falhou: {{ ultima.erro }}. Tente de novo.</p>

    <div v-if="laudo" class="cartao cheio pilha laudo">
      <div class="linha">
        <span class="etiqueta" :class="VEREDITOS[laudo.veredito]?.cor">{{ VEREDITOS[laudo.veredito]?.nome }}</span>
        <span class="pequeno">Cobre cerca de <strong>{{ laudo.cobertura }}%</strong> da etapa · {{ quando(ultima.atualizado_em) }}</span>
      </div>
      <div class="medidor" role="img" :aria-label="`Cobertura de ${laudo.cobertura}%`">
        <span :class="VEREDITOS[laudo.veredito]?.cor" :style="{ width: `${laudo.cobertura}%` }" />
      </div>
      <p v-if="mudou" class="aviso neutro sem-margem">O material mudou depois desta verificação: verifique de novo.</p>
      <p class="sem-margem" v-html="textoComCodigo(laudo.resumo)" />

      <div v-if="laudo.topicos?.length">
        <span class="rotulo">O que a etapa pede</span>
        <ul class="topicos">
          <li v-for="(x, i) in laudo.topicos" :key="i">
            <span class="etiqueta" :class="SITUACOES[x.situacao]?.[1]">{{ SITUACOES[x.situacao]?.[0] }}</span>
            <div>
              <span v-html="textoComCodigo(x.topico)" />
              <span v-if="x.onde" class="fraco"> · {{ x.onde }}</span>
              <div v-if="x.comentario" class="suave pequeno" v-html="textoComCodigo(x.comentario)" />
            </div>
          </li>
        </ul>
      </div>

      <div v-if="laudo.faltas?.length">
        <span class="rotulo">Para complementar</span>
        <ul class="itens">
          <li v-for="(x, i) in laudo.faltas" :key="i">
            <span v-html="textoComCodigo(x.ponto)" />
            <div v-if="x.onde_achar" class="suave pequeno">Onde achar: <span v-html="textoComCodigo(x.onde_achar)" /></div>
          </li>
        </ul>
      </div>

      <div v-if="laudo.materiais?.length">
        <span class="rotulo">Cada material</span>
        <ul class="itens">
          <li v-for="x in laudo.materiais" :key="x.material">
            <strong>{{ tituloDo(ultima, x.material) }}</strong>
            <span class="fraco"> · {{ x.lido ? `${x.cobertura}% sozinho, ${PROFUNDIDADES[x.profundidade]}` : 'não foi lido' }}</span>
            <div v-if="x.comentario" class="pequeno" v-html="textoComCodigo(x.comentario)" />
            <div v-if="x.fora" class="suave pequeno">Fora da etapa (dá para pular): <span v-html="textoComCodigo(x.fora)" /></div>
          </li>
        </ul>
      </div>

      <p v-if="faltando.length" class="suave pequeno sem-margem">
        Não verificado: {{ faltando.map((m) => `${m.titulo} (${TIPOS_MATERIAL[m.tipo] || m.tipo})`).join(', ') }}.
      </p>
      <div v-if="pedidoComplemento && laudo.veredito !== 'suficiente'">
        <button class="botao" @click="emit('sugerir', pedidoComplemento)">Procurar material para o que falta</button>
      </div>
    </div>

    <details v-if="anteriores.length" class="anteriores">
      <summary class="pequeno">Verificações anteriores ({{ anteriores.length }})</summary>
      <ul class="itens">
        <li v-for="v in anteriores" :key="v.id" class="pequeno">
          {{ quando(v.atualizado_em) }} · {{ VEREDITOS[v.resultado?.veredito]?.nome }} · {{ v.resultado?.cobertura }}% ·
          {{ v.materiais.map((m) => m.titulo).join(', ') }}
        </li>
      </ul>
    </details>
  </div>
</template>

<style scoped>
.verificar { margin-top: var(--e5); padding-top: var(--e4); border-top: 1px solid var(--borda); }
.entre { justify-content: space-between; }
.sem-margem { margin: 0; }
.itens { margin: 0; padding-left: var(--e5); display: grid; gap: var(--e2); font-size: 15px; line-height: 1.55; }
.topicos { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e2); font-size: 15px; line-height: 1.5; }
.topicos li { display: grid; grid-template-columns: 76px 1fr; gap: var(--e3); align-items: start; }
.topicos .etiqueta { justify-self: start; }
.medidor { height: 8px; border-radius: 999px; background: var(--superficie-2); overflow: hidden; }
.medidor span { display: block; height: 100%; border-radius: inherit; background: var(--azul); }
.medidor span.verde { background: var(--verde); }
.medidor span.ambar { background: var(--ambar); }
.medidor span.vermelho { background: var(--vermelho); }
.anteriores summary { cursor: pointer; color: var(--texto-suave); }
.anteriores .itens { margin-top: var(--e2); }
</style>
