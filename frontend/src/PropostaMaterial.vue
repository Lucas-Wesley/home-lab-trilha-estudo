<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import { abreEmOutraAba, hrefMaterial } from './material.js'
import { PAPEIS, TIPOS_MATERIAL } from './rotulos.js'

// "Sugerir material" (fase 6, T27): a IA procura na web, nos livros da Biblioteca, nos Temas do
// Acervo e nos documentos da trilha; o servidor confere os links e tira o que não existe. O mesmo
// ciclo das outras propostas (pedido, comentário geral ou num material, ajuste), e aprovar põe na
// etapa só os que o Lucas marcou.
const props = defineProps({
  etapaId: { type: Number, required: true },
  trilhaId: { type: Number, required: true },
})
const emit = defineEmits(['aprovada', 'aberta'])

const lista = ref([])
const abrirPedido = ref(false)
const erro = ref('')
const ocupado = ref(false)
async function carregar() {
  try {
    lista.value = await api(`/api/etapas/${props.etapaId}/propostas?tipo=materiais`)
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
const aberta = computed(() => !!(gerando.value || aguardando.value || falhou.value || abrirPedido.value))
watch(aberta, (v) => emit('aberta', v))
const vista = computed(() => aguardando.value || null)
const itens = computed(() => vista.value?.conteudo?.materiais || [])
const editavel = computed(() => vista.value && !gerando.value)

// Marcados para entrar: todos, menos o link quebrado.
const marcados = ref(new Set())
watch(
  () => vista.value?.id,
  () => (marcados.value = new Set(itens.value.map((m, i) => (m.situacao === 'quebrado' ? -1 : i)).filter((i) => i >= 0))),
  { immediate: true },
)
function alternar(i) {
  const s = new Set(marcados.value)
  s.has(i) ? s.delete(i) : s.add(i)
  marcados.value = s
}

// ---------- pedido ----------
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
  if (await acao(() => api(`/api/etapas/${props.etapaId}/materiais`, { metodo: 'POST', corpo: { pedido: texto } }))) {
    pedido.value = ''
    abrirPedido.value = false
  }
}
const cancelar = () => gerando.value?.tarefa && acao(() => api(`/api/tarefas/${gerando.value.tarefa.id}/cancelar`, { metodo: 'POST' }))

// ---------- comentários ----------
const comentario = ref({ texto: '', ancora: '' })
const nomeAncora = (ancora) => (ancora ? `Material ${ancora.replace(/^material:/, '')}` : 'Geral')
function comentarEm(i) {
  comentario.value.ancora = `material:${i + 1}`
  document.getElementById('comentario-materiais')?.focus()
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
  if (await acao(() => api(`/api/propostas/${vista.value.id}/aprovar`, { metodo: 'POST', corpo: { aceitos: [...marcados.value].sort((a, b) => a - b) } }))) emit('aprovada')
}
async function descartar(p) {
  if (confirm(`Descartar a proposta v${p.versao}?`)) await acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
}
const ESTADO_TAREFA = { na_fila: 'na fila', rodando: 'procurando', concluida: 'pronta', falhou: 'falhou', cancelada: 'cancelada' }
const SITUACAO = { quebrado: ['quebrado', 'vermelho'], nao_confirmado: ['não confirmado', 'ambar'] }

// pedir: o "Procurar material para o que falta" da verificação (T53) chega com o pedido pronto.
defineExpose({ abrir: () => (abrirPedido.value = true), pedir })
</script>

<template>
  <div v-if="aberta || erro" class="pilha proposta-material">
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <div v-if="abrirPedido && !gerando && !vista" class="cartao pilha">
      <label class="rotulo" for="pedido-materiais">O que procurar? <span class="fraco">(opcional)</span></label>
      <textarea id="pedido-materiais" v-model="pedido" class="campo" rows="2" placeholder="Ex.: um capítulo de livro que explique melhor; vídeos curtos; a documentação oficial do OpenWrt." />
      <p class="pequeno sem-margem">A IA procura na web, nos seus livros da Biblioteca, nos Temas do Acervo e nos documentos da trilha. O servidor confere os links e tira o que não existe; você marca o que entra.</p>
      <div class="linha">
        <button class="botao primario" :disabled="ocupado" @click="pedir()">Procurar material</button>
        <button class="botao discreto" @click="abrirPedido = false">Cancelar</button>
      </div>
    </div>

    <div v-if="gerando" class="cartao linha entre">
      <div>
        <strong>Material v{{ gerando.versao }}: {{ ESTADO_TAREFA[gerando.tarefa?.estado] || 'na fila' }}…</strong>
        <div class="suave">Costuma levar alguns minutos (a IA abre cada página); pode sair desta tela.</div>
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
        <strong>Material sugerido v{{ vista.versao }}: {{ itens.length }}</strong>
        <span class="etiqueta ambar">aguardando você</span>
      </div>
      <p v-if="vista.pedido" class="suave sem-margem">Pedido: “{{ vista.pedido }}”</p>
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
        <li v-for="(m, i) in itens" :key="i" class="cartao">
          <label class="linha marcar">
            <input type="checkbox" :checked="marcados.has(i)" :disabled="!editavel" @change="alternar(i)" />
            <span class="pilha corpo">
              <span class="linha cabeca">
                <span class="etiqueta">{{ TIPOS_MATERIAL[m.tipo] }}</span>
                <span class="etiqueta">{{ PAPEIS[m.papel] }}</span>
                <span v-if="SITUACAO[m.situacao]" class="etiqueta" :class="SITUACAO[m.situacao][1]">{{ SITUACAO[m.situacao][0] }}</span>
              </span>
              <span>
                <a v-if="hrefMaterial(m, trilhaId)" :href="hrefMaterial(m, trilhaId)" :target="abreEmOutraAba(m) ? '_blank' : undefined" rel="noopener noreferrer" @click.stop>{{ m.titulo }}</a>
                <span v-else>{{ m.titulo }}</span>
                <span v-if="m.localizador" class="fraco"> · {{ m.localizador }}</span>
              </span>
              <span v-if="m.motivo" class="pequeno">{{ m.motivo }}</span>
              <span v-if="m.nota" class="suave nota">Nota: {{ m.nota }}</span>
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
              <option v-for="(m, i) in itens" :key="i" :value="`material:${i + 1}`">{{ nomeAncora(`material:${i + 1}`) }}: {{ m.titulo }}</option>
            </select>
            <textarea id="comentario-materiais" v-model="comentario.texto" class="campo" rows="2" placeholder="Ex.: este é raso; prefiro algo em português; ache o capítulo certo do livro." />
            <div><button class="botao" type="submit" :disabled="ocupado || !comentario.texto.trim()">Comentar</button></div>
          </form>
        </div>
        <div class="cartao pilha">
          <label class="rotulo" for="ajuste-materiais">Pedido para o ajuste (opcional)</label>
          <textarea id="ajuste-materiais" v-model="pedidoAjuste" class="campo" rows="2" placeholder="Ex.: procure mais em livros." />
          <div class="linha entre">
            <div class="linha">
              <button class="botao primario" :disabled="ocupado || !marcados.size" @click="aprovar">Pôr {{ marcados.size }} na etapa</button>
              <button class="botao" :disabled="ocupado || (!meus.length && !pedidoAjuste.trim())" @click="ajustar">Pedir ajuste</button>
            </div>
            <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
          </div>
          <p class="pequeno sem-margem">Os marcados entram no fim do material da etapa (dá para reordenar em Editar). Pedir ajuste gera a versão {{ vista.versao + 1 }}.</p>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.proposta-material { margin-top: var(--e4); }
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
