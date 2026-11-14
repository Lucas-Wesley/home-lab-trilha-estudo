<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Leitor from './Leitor.vue'
import { api } from './api.js'
import { diferenca, resumoDiferenca } from './diff.js'

// A lição feita pela IA (DECISOES.md §5): pedido antes, proposta versionada, comentários gerais,
// numa seção ou num trecho selecionado, "Pedir ajuste" (próxima versão, com a resposta da IA a
// cada comentário), mudanças entre versões, Aprovar e Descartar.
const props = defineProps({
  etapaId: { type: Number, required: true },
  licaoAtual: { type: String, default: '' },
  baseDocs: { type: String, default: '' },
})
const emit = defineEmits(['aprovada', 'aberta'])

const lista = ref([])
const erro = ref('')
const ocupado = ref(false)

async function carregar() {
  try {
    lista.value = await api(`/api/etapas/${props.etapaId}/propostas`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}

// Enquanto uma versão gera, a tela pergunta a cada poucos segundos.
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
const comConteudo = computed(() => lista.value.filter((p) => p.conteudo))
const aberta = computed(() => !!(gerando.value || aguardando.value || falhou.value))
watch(aberta, (v) => emit('aberta', v), { immediate: true })

// Versão em leitura: a que espera o Lucas, ou outra escolhida no histórico. Sem nada aberto
// (lição aprovada), as versões ficam recolhidas atrás de "Ver as versões".
const escolhida = ref(0)
const vista = computed(() => lista.value.find((p) => p.id === escolhida.value) || aguardando.value || null)
watch(aguardando, (p) => {
  escolhida.value = p ? p.id : 0
})
const conteudo = computed(() => vista.value?.conteudo || {})
const editavel = computed(() => vista.value && vista.value.estado === 'aguardando' && !gerando.value)

// O que mudou: contra a versão anterior, ou contra a lição em vigor num "pedir mudança".
const textoBase = computed(() => {
  const p = vista.value
  if (!p) return ''
  if (p.anterior_id) return lista.value.find((x) => x.id === p.anterior_id)?.conteudo?.licao || ''
  return props.licaoAtual
})
const modo = ref('texto')
const trechos = computed(() => (textoBase.value && conteudo.value.licao ? diferenca(textoBase.value, conteudo.value.licao) : []))
const resumo = computed(() => resumoDiferenca(trechos.value))
const linksRuins = computed(() => (conteudo.value.links || []).filter((l) => l.situacao !== 'ok'))

// ---------- pedido (antes) ----------
const pedido = ref('')
const abrirPedido = ref(false)
async function pedir(texto = pedido.value) {
  ocupado.value = true
  erro.value = ''
  try {
    await api(`/api/etapas/${props.etapaId}/licao`, { metodo: 'POST', corpo: { pedido: texto } })
    pedido.value = ''
    abrirPedido.value = false
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

async function cancelar() {
  if (!gerando.value?.tarefa) return
  await acao(() => api(`/api/tarefas/${gerando.value.tarefa.id}/cancelar`, { metodo: 'POST' }))
}

async function acao(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    const r = await fn()
    await carregar()
    return r ?? true
  } catch (e) {
    erro.value = e.message
    return false
  } finally {
    ocupado.value = false
  }
}

// ---------- comentários (depois) ----------
const titulos = ref([])
const comentario = ref({ texto: '', ancora: '', citacao: '' })
const campoComentario = ref(null)
const caixaLicao = ref(null)

const nomeAncora = (ancora) => {
  if (!ancora) return 'Geral'
  const id = ancora.replace(/^secao:/, '')
  return 'Seção "' + (titulos.value.find((h) => h.id === id)?.texto || id) + '"'
}

// Texto selecionado na lição vira o trecho do comentário, já com a seção onde ele está.
function aoSelecionar() {
  if (!editavel.value) return
  const sel = window.getSelection()
  const texto = sel ? sel.toString().trim() : ''
  if (!texto || !caixaLicao.value?.contains(sel.anchorNode)) return
  comentario.value.citacao = texto.slice(0, 2000)
  let no = sel.anchorNode.nodeType === 1 ? sel.anchorNode : sel.anchorNode.parentElement
  while (no && no.parentElement && !no.parentElement.classList.contains('leitura')) no = no.parentElement
  for (let el = no; el; el = el.previousElementSibling) {
    if (/^H[23]$/.test(el.tagName) && el.id) {
      comentario.value.ancora = 'secao:' + el.id
      break
    }
  }
}
async function irParaComentario() {
  await nextTick()
  campoComentario.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  campoComentario.value?.focus({ preventScroll: true })
}

async function comentar() {
  const ok = await acao(() => api(`/api/propostas/${vista.value.id}/comentarios`, { metodo: 'POST', corpo: comentario.value }))
  if (ok) comentario.value = { texto: '', ancora: '', citacao: '' }
}
const tirarComentario = (c) => acao(() => api(`/api/comentarios/${c.id}`, { metodo: 'DELETE' }))

const meus = computed(() => (vista.value?.comentarios || []).filter((c) => c.autor === 'lucas'))
// Respostas da IA, com o comentário (da versão anterior) a que cada uma responde.
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
  if (await acao(() => api(`/api/propostas/${vista.value.id}/aprovar`, { metodo: 'POST' }))) {
    escolhida.value = 0
    emit('aprovada')
  }
}
async function descartar(p) {
  if (!confirm(`Descartar a proposta v${p.versao}?`)) return
  await acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
}

const ESTADOS = {
  gerando: { nome: 'gerando', cor: 'azul' },
  aguardando: { nome: 'aguardando você', cor: 'ambar' },
  aprovada: { nome: 'aprovada', cor: 'verde' },
  descartada: { nome: 'descartada', cor: '' },
  substituida: { nome: 'substituída', cor: '' },
  falhou: { nome: 'falhou', cor: 'vermelho' },
}
const ESTADO_TAREFA = { na_fila: 'na fila', rodando: 'escrevendo', concluida: 'pronta', falhou: 'falhou', cancelada: 'cancelada' }

defineExpose({ abrir: () => (abrirPedido.value = true) })
</script>

<template>
  <div class="proposta pilha">
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <!-- Pedido (antes): sempre com texto livre. -->
    <div v-if="!aberta && (!licaoAtual || abrirPedido)" class="cartao pilha">
      <template v-if="!aberta">
        <label class="rotulo" for="pedido-licao">{{ licaoAtual ? 'O que mudar na lição?' : 'Pedido para a IA (opcional)' }}</label>
        <textarea
          id="pedido-licao"
          v-model="pedido"
          class="campo"
          rows="3"
          :placeholder="licaoAtual ? 'Ex.: acrescente um diagrama do caminho do pacote; a parte de CGNAT ficou rasa.' : 'Ex.: compare com o IPv6; use exemplos da minha rede; mais curta.'"
        />
        <p class="pequeno sem-margem">Vai junto com as instruções globais (Ajustes) e as da trilha. A IA lê o material da etapa e a lição dos pré-requisitos; a lição fica como proposta até você aprovar.</p>
        <div class="linha">
          <button class="botao primario" :disabled="ocupado || (licaoAtual && !pedido.trim())" @click="pedir()">{{ licaoAtual ? 'Pedir mudança' : 'Gerar lição' }}</button>
          <button v-if="licaoAtual" class="botao discreto" @click="abrirPedido = false">Cancelar</button>
        </div>
      </template>
    </div>

    <!-- Gerando -->
    <div v-if="gerando" class="cartao linha entre">
      <div>
        <strong>Versão {{ gerando.versao }}: {{ ESTADO_TAREFA[gerando.tarefa?.estado] || 'na fila' }}…</strong>
        <div class="suave">Leva alguns minutos; pode sair desta tela.<template v-if="gerando.pedido"> Pedido: “{{ gerando.pedido }}”</template></div>
      </div>
      <button class="botao discreto" :disabled="ocupado" @click="cancelar">Cancelar</button>
    </div>

    <!-- Falhou -->
    <div v-if="falhou && !gerando" class="cartao pilha">
      <p class="aviso sem-margem">A versão {{ falhou.versao }} não saiu: {{ falhou.erro || 'erro sem mensagem' }}</p>
      <div class="linha">
        <button class="botao" :disabled="ocupado" @click="pedir(falhou.pedido)">Tentar de novo</button>
        <button class="botao discreto" :disabled="ocupado" @click="descartar(falhou)">Descartar</button>
      </div>
    </div>

    <div v-if="!aberta && !vista && comConteudo.length" class="linha">
      <button class="botao discreto" @click="escolhida = comConteudo.at(-1).id">Ver as versões da IA ({{ comConteudo.length }})</button>
    </div>

    <!-- A proposta -->
    <template v-if="vista && vista.conteudo">
      <div class="linha entre cabeca">
        <div class="linha">
          <strong>Proposta v{{ vista.versao }}</strong>
          <span class="etiqueta" :class="ESTADOS[vista.estado].cor">{{ ESTADOS[vista.estado].nome }}</span>
        </div>
        <div class="linha versoes" aria-label="Versões">
          <template v-if="comConteudo.length > 1">
            <button v-for="p in comConteudo" :key="p.id" class="botao discreto" :class="{ ativa: p.id === vista.id }" @click="escolhida = p.id">v{{ p.versao }}</button>
          </template>
          <button v-if="!aguardando" class="botao discreto" @click="escolhida = 0">Fechar</button>
        </div>
      </div>
      <p v-if="vista.pedido" class="suave sem-margem">Pedido: “{{ vista.pedido }}”</p>

      <div v-if="conteudo.notas" class="cartao cheio">
        <span class="rotulo">Notas da IA</span>
        <p class="sem-margem pre">{{ conteudo.notas }}</p>
      </div>
      <div v-if="respostas.length" class="cartao cheio pilha">
        <span class="rotulo">Respostas aos seus comentários</span>
        <div v-for="r in respostas" :key="r.id" class="resposta">
          <div v-if="r.original" class="suave">
            <span class="fraco">{{ nomeAncora(r.original.ancora) }}:</span> “{{ r.original.texto }}”
          </div>
          <div>{{ r.texto }}</div>
        </div>
      </div>
      <p v-if="linksRuins.length" class="aviso neutro sem-margem">
        Links conferidos pelo app:
        <template v-for="(l, i) in linksRuins" :key="l.url">
          <span v-if="i">; </span><a :href="l.url" target="_blank" rel="noopener noreferrer">{{ l.url }}</a>
          {{ l.situacao === 'quebrado' ? 'quebrado' : 'não confirmado (pode bloquear robôs; abra para ver)' }}<template v-if="l.detalhe"> ({{ l.detalhe }})</template>
        </template>
      </p>

      <div v-if="trechos.length" class="linha">
        <button class="botao discreto" :class="{ ativa: modo === 'texto' }" @click="modo = 'texto'">Texto</button>
        <button class="botao discreto" :class="{ ativa: modo === 'mudancas' }" @click="modo = 'mudancas'">
          Mudanças <span class="fraco">+{{ resumo.mais }} −{{ resumo.menos }}</span>
        </button>
      </div>

      <div v-if="modo === 'mudancas' && trechos.length" class="diff" aria-label="Mudanças desde a versão anterior">
        <template v-for="(t, i) in trechos" :key="i">
          <pre v-if="t.tipo !== 'igual'" :class="t.tipo">{{ t.linhas.join('\n') }}</pre>
          <details v-else-if="t.linhas.length > 6" class="igual">
            <summary>{{ t.linhas.length }} linhas iguais</summary>
            <pre>{{ t.linhas.join('\n') }}</pre>
          </details>
          <pre v-else class="igual">{{ t.linhas.join('\n') }}</pre>
        </template>
      </div>
      <div v-else ref="caixaLicao" class="licao-proposta" @mouseup="aoSelecionar" @touchend="aoSelecionar">
        <Leitor :texto="conteudo.licao" :base-docs="baseDocs" @titulos="titulos = $event" />
      </div>

      <!-- Comentários (depois) -->
      <template v-if="editavel">
        <button v-if="comentario.citacao" class="botao comentar-trecho" @click="irParaComentario">Comentar o trecho selecionado</button>
        <div class="cartao pilha">
          <span class="rotulo">Comentários para o ajuste</span>
          <ul v-if="meus.length" class="comentarios">
            <li v-for="c in meus" :key="c.id">
              <div class="fraco">{{ nomeAncora(c.ancora) }}</div>
              <blockquote v-if="c.citacao">{{ c.citacao }}</blockquote>
              <div class="linha entre"><span>{{ c.texto }}</span><button class="botao discreto" :disabled="ocupado" @click="tirarComentario(c)">Tirar</button></div>
            </li>
          </ul>
          <form class="pilha" @submit.prevent="comentar">
            <select v-model="comentario.ancora" class="campo" aria-label="Onde">
              <option value="">Geral (a lição toda)</option>
              <option v-for="h in titulos" :key="h.id" :value="'secao:' + h.id">{{ h.nivel === 3 ? '— ' : '' }}{{ h.texto }}</option>
            </select>
            <div v-if="comentario.citacao" class="citacao linha entre">
              <blockquote>{{ comentario.citacao }}</blockquote>
              <button type="button" class="botao discreto" @click="comentario.citacao = ''">Tirar o trecho</button>
            </div>
            <p v-else class="pequeno sem-margem">Para comentar um trecho, selecione o texto na lição.</p>
            <textarea ref="campoComentario" v-model="comentario.texto" class="campo" rows="2" placeholder="Ex.: este parágrafo está errado; aprofunde aqui; faltou o caso do IPv6." />
            <div><button class="botao" type="submit" :disabled="ocupado || !comentario.texto.trim()">Comentar</button></div>
          </form>
        </div>

        <div class="cartao pilha">
          <label class="rotulo" for="pedido-ajuste">Pedido para o ajuste (opcional)</label>
          <textarea id="pedido-ajuste" v-model="pedidoAjuste" class="campo" rows="2" placeholder="Ex.: no geral, mais curta e com mais exemplos." />
          <div class="linha entre">
            <div class="linha">
              <button class="botao primario" :disabled="ocupado" @click="aprovar">Aprovar</button>
              <button class="botao" :disabled="ocupado || (!meus.length && !pedidoAjuste.trim())" @click="ajustar">Pedir ajuste</button>
            </div>
            <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
          </div>
          <p class="pequeno sem-margem">Aprovar grava esta versão como a lição da etapa (a anterior fica no histórico do git). Pedir ajuste manda os comentários e o pedido e gera a versão {{ vista.versao + 1 }}.</p>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.sem-margem { margin: 0; }
.pre { white-space: pre-wrap; }
.cabeca { margin-top: var(--e2); }
.versoes .botao { height: 26px; padding: 0 8px; }
.botao.ativa { background: var(--superficie-2); color: var(--texto); }
.resposta { display: grid; gap: 2px; padding-left: var(--e3); border-left: 2px solid var(--borda-forte); font-size: 14.5px; }
.licao-proposta { border: 1px solid var(--borda); border-radius: var(--raio); padding: var(--e2) var(--e5) var(--e4); }
.licao-proposta :deep(h1:first-child) { display: none; }
.comentarios { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); font-size: 14.5px; }
blockquote { margin: 2px 0; padding: 2px var(--e3); border-left: 2px solid var(--ambar); color: var(--texto-suave); font-size: 13.5px; white-space: pre-wrap; max-height: 7.5em; overflow: auto; }
.citacao blockquote { flex: 1; }
.comentar-trecho { position: sticky; bottom: calc(var(--e4) + 64px); align-self: center; z-index: 5; box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2); }
.diff { border: 1px solid var(--borda); border-radius: var(--raio); overflow: hidden; font-size: 12.5px; }
.diff pre { margin: 0; padding: 2px var(--e3); white-space: pre-wrap; word-break: break-word; font-family: var(--fonte-mono); }
.diff .mais { background: var(--verde-fundo); color: var(--verde); }
.diff .menos { background: var(--vermelho-fundo); color: var(--vermelho); text-decoration: line-through; }
.diff .igual { color: var(--texto-suave); }
.diff details summary { padding: 2px var(--e3); cursor: pointer; color: var(--texto-fraco); font-size: 12px; }
@media (max-width: 599px) {
  .licao-proposta { padding: var(--e1) var(--e3) var(--e3); }
}
@media (min-width: 900px) {
  .comentar-trecho { bottom: var(--e4); }
}
</style>
