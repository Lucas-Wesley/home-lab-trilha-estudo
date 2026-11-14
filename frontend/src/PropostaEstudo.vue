<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { textoComCodigo } from './markdown.js'

// Exercícios e perguntas do simulado feitos pela IA (fase 5, T26): o mesmo ciclo da lição
// (pedido, proposta versionada, comentários gerais ou numa questão, ajuste, aprovar), com o
// gabarito escondido para não estragar o estudo: o Lucas abre só se quiser conferir.
const props = defineProps({
  etapaId: { type: Number, required: true },
  tipo: { type: String, required: true }, // exercicios · simulado
  temAtual: { type: Boolean, default: false }, // a etapa já tem exercícios ou perguntas aprovados
})
const emit = defineEmits(['aprovada', 'aberta'])

const ehExercicios = computed(() => props.tipo === 'exercicios')
const nome = computed(() => (ehExercicios.value ? 'exercícios' : 'perguntas do simulado'))

const lista = ref([])
const erro = ref('')
const ocupado = ref(false)

async function carregar() {
  try {
    lista.value = await api(`/api/etapas/${props.etapaId}/propostas?tipo=${props.tipo}`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
let relogio = 0
const gerando = computed(() => lista.value.find((p) => p.estado === 'gerando'))
watch(gerando, (g) => {
  clearInterval(relogio)
  if (g) relogio = setInterval(carregar, 4000)
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

const aguardando = computed(() => lista.value.find((p) => p.estado === 'aguardando'))
const ultima = computed(() => lista.value.at(-1))
const falhou = computed(() => (ultima.value?.estado === 'falhou' ? ultima.value : null))
const aberta = computed(() => !!(gerando.value || aguardando.value || falhou.value))
watch(aberta, (v) => emit('aberta', v), { immediate: true })

const vista = computed(() => aguardando.value || null)
const conteudo = computed(() => vista.value?.conteudo || {})
const itens = computed(() => (ehExercicios.value ? conteudo.value.exercicios : conteudo.value.perguntas) || [])
const editavel = computed(() => vista.value && !gerando.value)

// ---------- pedido ----------
const pedido = ref('')
const abrirPedido = ref(false)
async function pedir(texto = pedido.value) {
  if (await acao(() => api(`/api/etapas/${props.etapaId}/${props.tipo}`, { metodo: 'POST', corpo: { pedido: texto } }))) {
    pedido.value = ''
    abrirPedido.value = false
  }
}
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
const cancelar = () => gerando.value?.tarefa && acao(() => api(`/api/tarefas/${gerando.value.tarefa.id}/cancelar`, { metodo: 'POST' }))

// ---------- comentários ----------
const comentario = ref({ texto: '', ancora: '' })
const nomeAncora = (ancora) => (ancora ? `${ehExercicios.value ? 'Exercício' : 'Pergunta'} ${ancora.replace(/^questao:/, '')}` : 'Geral')
function comentarQuestao(i) {
  comentario.value.ancora = `questao:${i + 1}`
  document.getElementById(`comentario-${props.tipo}`)?.focus()
}
async function comentar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/comentarios`, { metodo: 'POST', corpo: comentario.value }))) comentario.value = { texto: '', ancora: '' }
}
const tirarComentario = (c) => acao(() => api(`/api/comentarios/${c.id}`, { metodo: 'DELETE' }))
const meus = computed(() => (vista.value?.comentarios || []).filter((c) => c.autor === 'lucas'))
const respostas = computed(() => {
  const todos = Object.fromEntries(lista.value.flatMap((p) => p.comentarios).map((c) => [c.id, c]))
  return (vista.value?.comentarios || []).filter((c) => c.autor === 'ia').map((c) => ({ ...c, original: todos[c.responde_a] }))
})

// ---------- decidir ----------
const pedidoAjuste = ref('')
async function ajustar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/ajustar`, { metodo: 'POST', corpo: { pedido: pedidoAjuste.value } }))) pedidoAjuste.value = ''
}
async function aprovar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/aprovar`, { metodo: 'POST' }))) emit('aprovada')
}
async function descartar(p) {
  if (confirm(`Descartar a proposta v${p.versao}?`)) await acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
}
const ESTADO_TAREFA = { na_fila: 'na fila', rodando: 'escrevendo', concluida: 'pronta', falhou: 'falhou', cancelada: 'cancelada' }
const letra = (i) => String.fromCharCode(65 + i)

defineExpose({ abrir: () => (abrirPedido.value = true) })
</script>

<template>
  <div class="pilha">
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <div v-if="!aberta && (!temAtual || abrirPedido)" class="cartao pilha">
      <label class="rotulo" :for="`pedido-${tipo}`">
        {{ temAtual ? (ehExercicios ? 'O que mudar nos exercícios?' : 'Foco das perguntas novas (opcional)') : 'Pedido para a IA (opcional)' }}
      </label>
      <textarea
        :id="`pedido-${tipo}`"
        v-model="pedido"
        class="campo"
        rows="2"
        :placeholder="ehExercicios ? 'Ex.: mais casos práticos com a minha rede; menos definição.' : 'Ex.: perguntas sobre o que dá errado quando o NAT falha.'"
      />
      <p class="pequeno sem-margem">
        <template v-if="ehExercicios">A IA tira {{ temAtual ? 'um conjunto novo, que substitui o atual,' : 'os exercícios' }} da lição aprovada. Eles chegam como proposta, com o gabarito escondido.</template>
        <template v-else>A IA tira as perguntas da lição aprovada{{ temAtual ? ', sem repetir as do banco' : '' }}. Elas chegam como proposta e, aprovadas, entram no banco do simulado.</template>
      </p>
      <div class="linha">
        <button class="botao primario" :disabled="ocupado || (temAtual && ehExercicios && !pedido.trim())" @click="pedir()">
          {{ ehExercicios ? (temAtual ? 'Pedir exercícios novos' : 'Gerar exercícios') : temAtual ? 'Gerar mais perguntas' : 'Gerar perguntas' }}
        </button>
        <button v-if="temAtual" class="botao discreto" @click="abrirPedido = false">Cancelar</button>
      </div>
    </div>

    <div v-if="gerando" class="cartao linha entre">
      <div>
        <strong>{{ ehExercicios ? 'Exercícios' : 'Perguntas' }} v{{ gerando.versao }}: {{ ESTADO_TAREFA[gerando.tarefa?.estado] || 'na fila' }}…</strong>
        <div class="suave">Costuma levar um ou dois minutos; pode sair desta tela.</div>
      </div>
      <button class="botao discreto" :disabled="ocupado" @click="cancelar">Cancelar</button>
    </div>

    <div v-if="falhou && !gerando" class="cartao pilha">
      <p class="aviso sem-margem">A versão {{ falhou.versao }} não saiu: {{ falhou.erro || 'erro sem mensagem' }}</p>
      <div class="linha">
        <button class="botao" :disabled="ocupado" @click="pedir(falhou.pedido)">Tentar de novo</button>
        <button class="botao discreto" :disabled="ocupado" @click="descartar(falhou)">Descartar</button>
      </div>
    </div>

    <template v-if="vista && vista.conteudo">
      <div class="linha">
        <strong>Proposta v{{ vista.versao }}: {{ itens.length }} {{ nome }}</strong>
        <span class="etiqueta ambar">aguardando você</span>
      </div>
      <p v-if="vista.pedido" class="suave sem-margem">Pedido: “{{ vista.pedido }}”</p>
      <div v-if="conteudo.notas" class="cartao cheio">
        <span class="rotulo">Notas da IA</span>
        <p class="sem-margem pre">{{ conteudo.notas }}</p>
      </div>
      <div v-if="respostas.length" class="cartao cheio pilha">
        <span class="rotulo">Respostas aos seus comentários</span>
        <div v-for="r in respostas" :key="r.id" class="resposta">
          <div v-if="r.original" class="suave"><span class="fraco">{{ nomeAncora(r.original.ancora) }}:</span> “{{ r.original.texto }}”</div>
          <div>{{ r.texto }}</div>
        </div>
      </div>

      <ol class="questoes">
        <li v-for="(q, i) in itens" :key="i" class="cartao pilha">
          <div class="linha entre topo">
            <span class="pequeno">{{ q.conceito }}<template v-if="ehExercicios"> · {{ q.tipo === 'escolha' ? 'escolha' : 'completar' }}</template></span>
            <button v-if="editavel" class="botao discreto" @click="comentarQuestao(i)">Comentar</button>
          </div>
          <div class="enunciado" v-html="textoComCodigo(ehExercicios ? q.enunciado : q.pergunta)" />
          <ul v-if="q.alternativas" class="alternativas">
            <li v-for="(alt, j) in q.alternativas" :key="j"><span class="fraco">{{ letra(j) }})</span> <span v-html="textoComCodigo(alt)" /></li>
          </ul>
          <details>
            <summary>Ver o gabarito</summary>
            <div class="gabarito pilha">
              <template v-if="ehExercicios">
                <div><strong>Certa:</strong> {{ q.tipo === 'escolha' ? `${letra(q.correta)}) ${q.alternativas[q.correta]}` : q.aceitas.join(' · ') }}</div>
                <div class="pre">{{ q.explicacao }}</div>
              </template>
              <div v-else class="pre">{{ q.esperado }}</div>
            </div>
          </details>
        </li>
      </ol>

      <template v-if="editavel">
        <div class="cartao pilha">
          <span class="rotulo">Comentários para o ajuste</span>
          <ul v-if="meus.length" class="comentarios">
            <li v-for="c in meus" :key="c.id" class="linha entre">
              <span><span class="fraco">{{ nomeAncora(c.ancora) }}:</span> {{ c.texto }}</span>
              <button class="botao discreto" :disabled="ocupado" @click="tirarComentario(c)">Tirar</button>
            </li>
          </ul>
          <form class="pilha" @submit.prevent="comentar">
            <select v-model="comentario.ancora" class="campo" aria-label="Onde">
              <option value="">Geral</option>
              <option v-for="(q, i) in itens" :key="i" :value="`questao:${i + 1}`">{{ nomeAncora(`questao:${i + 1}`) }}</option>
            </select>
            <textarea :id="`comentario-${tipo}`" v-model="comentario.texto" class="campo" rows="2" placeholder="Ex.: a alternativa C também está certa; fácil demais; troque por um caso prático." />
            <div><button class="botao" type="submit" :disabled="ocupado || !comentario.texto.trim()">Comentar</button></div>
          </form>
        </div>
        <div class="cartao pilha">
          <label class="rotulo" :for="`ajuste-${tipo}`">Pedido para o ajuste (opcional)</label>
          <textarea :id="`ajuste-${tipo}`" v-model="pedidoAjuste" class="campo" rows="2" placeholder="Ex.: no geral, mais difíceis." />
          <div class="linha entre">
            <div class="linha">
              <button class="botao primario" :disabled="ocupado" @click="aprovar">Aprovar</button>
              <button class="botao" :disabled="ocupado || (!meus.length && !pedidoAjuste.trim())" @click="ajustar">Pedir ajuste</button>
            </div>
            <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
          </div>
          <p class="pequeno sem-margem">
            {{ ehExercicios ? 'Aprovar troca os exercícios da etapa por estes.' : 'Aprovar põe estas perguntas no banco do simulado.' }}
            Pedir ajuste gera a versão {{ vista.versao + 1 }}.
          </p>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.pre { white-space: pre-wrap; }
.cartao p { display: block; overflow: visible; }
.resposta { display: grid; gap: 2px; padding-left: var(--e3); border-left: 2px solid var(--borda-forte); font-size: 14.5px; }
.questoes { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.topo { align-items: center; }
.enunciado { font-size: 15px; line-height: 1.55; }
.alternativas { list-style: none; margin: 0; padding: 0; display: grid; gap: 4px; font-size: 14.5px; }
details summary { cursor: pointer; color: var(--texto-fraco); font-size: 13px; }
.gabarito { margin-top: var(--e2); font-size: 14px; padding-left: var(--e3); border-left: 2px solid var(--verde); }
.comentarios { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e2); font-size: 14.5px; }
</style>
