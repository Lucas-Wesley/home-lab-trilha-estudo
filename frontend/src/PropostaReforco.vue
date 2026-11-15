<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { BLOOM, duracao } from './rotulos.js'

// Reforço (fase 7, T28): numa etapa em que o Lucas errou, a IA diagnostica e propõe de uma a três
// etapas novas (antes ou depois desta). O mesmo ciclo das outras propostas (pedido, comentário
// geral ou numa etapa, ajuste); aprovar insere na trilha só as marcadas.
const props = defineProps({
  etapaId: { type: Number, required: true },
  temErros: { type: Boolean, default: false },
})
const emit = defineEmits(['aprovada', 'aberta'])

const lista = ref([])
const abrirPedido = ref(false)
const erro = ref('')
const ocupado = ref(false)
async function carregar() {
  try {
    lista.value = await api(`/api/etapas/${props.etapaId}/propostas?tipo=reforco`)
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
const aprovadas = computed(() => lista.value.filter((p) => p.estado === 'aprovada'))
const aberta = computed(() => !!(gerando.value || aguardando.value || falhou.value || abrirPedido.value))
watch(aberta, (v) => emit('aberta', v))
const vista = computed(() => aguardando.value || null)
const itens = computed(() => vista.value?.conteudo?.etapas || [])
const editavel = computed(() => vista.value && !gerando.value)

const marcados = ref(new Set())
watch(
  () => vista.value?.id,
  () => (marcados.value = new Set(itens.value.map((_, i) => i))),
  { immediate: true },
)
function alternar(i) {
  const s = new Set(marcados.value)
  s.has(i) ? s.delete(i) : s.add(i)
  marcados.value = s
}

const pedido = ref('')
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
async function pedir(texto = pedido.value) {
  if (await acao(() => api(`/api/etapas/${props.etapaId}/reforco`, { metodo: 'POST', corpo: { pedido: texto } }))) {
    pedido.value = ''
    abrirPedido.value = false
  }
}
const cancelar = () => gerando.value?.tarefa && acao(() => api(`/api/tarefas/${gerando.value.tarefa.id}/cancelar`, { metodo: 'POST' }))

const comentario = ref({ texto: '', ancora: '' })
const nomeAncora = (ancora) => (ancora ? `Etapa ${ancora.replace(/^etapa:/, '')}` : 'Geral')
function comentarEm(i) {
  comentario.value.ancora = `etapa:${i + 1}`
  document.getElementById('comentario-reforco')?.focus()
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

const pedidoAjuste = ref('')
async function ajustar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/ajustar`, { metodo: 'POST', corpo: { pedido: pedidoAjuste.value } }))) pedidoAjuste.value = ''
}
async function aprovar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/aprovar`, { metodo: 'POST', corpo: { aceitos: [...marcados.value].sort((a, b) => a - b) } }))) emit('aprovada')
}
async function descartar(p) {
  if (confirm(`Descartar a proposta v${p.versao}?`)) await acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
}
const ESTADO_TAREFA = { na_fila: 'na fila', rodando: 'pensando', concluida: 'pronta', falhou: 'falhou', cancelada: 'cancelada' }

defineExpose({ abrir: () => (abrirPedido.value = true) })
</script>

<template>
  <div class="pilha proposta-reforco">
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <div v-if="!aberta" class="linha">
      <button class="botao" @click="abrirPedido = true">Pedir reforço</button>
      <span v-if="aprovadas.length" class="pequeno">{{ aprovadas.length }} reforço(s) aprovado(s) antes.</span>
    </div>

    <div v-if="abrirPedido && !gerando && !vista" class="cartao pilha">
      <label class="rotulo" for="pedido-reforco">O que reforçar? <span class="fraco">{{ temErros ? '(opcional: a IA já recebe os seus erros)' : '(diga o que não ficou claro)' }}</span></label>
      <textarea id="pedido-reforco" v-model="pedido" class="campo" rows="2" placeholder="Ex.: não entendi a diferença entre NAT e CGNAT; quero mais prática." />
      <p class="pequeno sem-margem">A IA lê a lição, as suas notas e o que você errou da última vez, diz a causa provável e propõe de uma a três etapas novas (antes ou depois desta). Você marca as que entram.</p>
      <div class="linha">
        <button class="botao primario" :disabled="ocupado || (!temErros && !pedido.trim())" @click="pedir()">Pedir à IA</button>
        <button class="botao discreto" @click="abrirPedido = false">Cancelar</button>
      </div>
    </div>

    <div v-if="gerando" class="cartao linha entre">
      <div>
        <strong>Reforço v{{ gerando.versao }}: {{ ESTADO_TAREFA[gerando.tarefa?.estado] || 'na fila' }}…</strong>
        <div class="suave">Pode sair desta tela.</div>
      </div>
      <button class="botao discreto" :disabled="ocupado" @click="cancelar">Cancelar</button>
    </div>

    <div v-if="falhou && !gerando && !vista" class="cartao pilha">
      <p class="aviso sem-margem">A versão {{ falhou.versao }} não saiu: {{ falhou.erro || 'erro sem mensagem' }}</p>
      <div class="linha">
        <button class="botao" :disabled="ocupado" @click="pedir(falhou.pedido)">Tentar de novo</button>
        <button class="botao discreto" :disabled="ocupado" @click="descartar(falhou)">Descartar</button>
      </div>
    </div>

    <template v-if="vista && vista.conteudo">
      <div class="linha">
        <strong>Reforço v{{ vista.versao }}: {{ itens.length }} etapa(s)</strong>
        <span class="etiqueta ambar">aguardando você</span>
      </div>
      <p v-if="vista.pedido" class="suave sem-margem">Pedido: “{{ vista.pedido }}”</p>
      <div class="cartao cheio">
        <span class="rotulo">Diagnóstico</span>
        <p class="sem-margem pre">{{ vista.conteudo.diagnostico }}</p>
      </div>
      <div v-if="vista.conteudo.notas" class="cartao cheio">
        <span class="rotulo">Notas da IA</span>
        <p class="sem-margem pre">{{ vista.conteudo.notas }}</p>
      </div>
      <div v-if="respostas.length" class="cartao cheio pilha">
        <span class="rotulo">Respostas aos seus comentários</span>
        <div v-for="r in respostas" :key="r.id" class="resposta">
          <div v-if="r.original" class="suave"><span class="fraco">{{ nomeAncora(r.original.ancora) }}:</span> “{{ r.original.texto }}”</div>
          <div>{{ r.texto }}</div>
        </div>
      </div>

      <ol class="sugeridos">
        <li v-for="(x, i) in itens" :key="i" class="cartao">
          <label class="linha marcar">
            <input type="checkbox" :checked="marcados.has(i)" :disabled="!editavel" @change="alternar(i)" />
            <span class="pilha corpo">
              <span class="linha cabeca">
                <span class="etiqueta">{{ x.posicao === 'antes' ? 'antes desta' : 'depois desta' }}</span>
                <span v-if="BLOOM[x.bloom]" class="etiqueta">{{ BLOOM[x.bloom] }}</span>
                <span class="etiqueta">{{ duracao(x.minutos) }}</span>
              </span>
              <strong>{{ x.titulo }}</strong>
              <span v-if="x.objetivo">{{ x.objetivo }}</span>
              <span class="pequeno"><span class="fraco">Foco da lição:</span> {{ x.foco }}</span>
              <span v-if="x.motivo" class="suave nota">Por quê: {{ x.motivo }}</span>
            </span>
          </label>
          <div v-if="editavel" class="direita"><button class="botao discreto" @click="comentarEm(i)">Comentar</button></div>
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
              <option v-for="(x, i) in itens" :key="i" :value="`etapa:${i + 1}`">{{ nomeAncora(`etapa:${i + 1}`) }}: {{ x.titulo }}</option>
            </select>
            <textarea id="comentario-reforco" v-model="comentario.texto" class="campo" rows="2" placeholder="Ex.: esta já sei; junte as duas; mais prática com o OpenWrt." />
            <div><button class="botao" type="submit" :disabled="ocupado || !comentario.texto.trim()">Comentar</button></div>
          </form>
        </div>
        <div class="cartao pilha">
          <label class="rotulo" for="ajuste-reforco">Pedido para o ajuste (opcional)</label>
          <textarea id="ajuste-reforco" v-model="pedidoAjuste" class="campo" rows="2" />
          <div class="linha entre">
            <div class="linha">
              <button class="botao primario" :disabled="ocupado || !marcados.size" @click="aprovar">Pôr {{ marcados.size }} na trilha</button>
              <button class="botao" :disabled="ocupado || (!meus.length && !pedidoAjuste.trim())" @click="ajustar">Pedir ajuste</button>
            </div>
            <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
          </div>
          <p class="pequeno sem-margem">As marcadas entram neste módulo, logo antes ou logo depois desta etapa, sem lição (gere quando chegar nelas). Se a trilha tem meta, recalcule o cronograma depois.</p>
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
.sugeridos { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.marcar { align-items: flex-start; flex-wrap: nowrap; cursor: pointer; }
.marcar input { margin-top: 4px; flex: none; }
.corpo { gap: 4px; font-size: 15px; line-height: 1.5; min-width: 0; overflow-wrap: anywhere; }
.cabeca { gap: 4px; }
.nota { font-size: 14px; }
.direita { display: flex; justify-content: flex-end; }
.comentarios { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e2); font-size: 14.5px; }
</style>
