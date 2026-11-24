<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import PedidoBlog from './PedidoBlog.vue'
import { api } from './api.js'
import { ENGRENAGENS, emAndamento, fase, ideias, limparOpcoes, resumoOpcoes } from './blog.js'

// Blog (T62): no item do Projeto, a IA orienta o que escrever — "Sugerir artigos" a partir do item,
// ou "Pedir tópicos" para um título do Lucas. Ele escreve tudo; a IA não redige nada.
const props = defineProps({ itemId: { type: String, required: true } })

const GUARDADAS = 'trilha.blog.opcoes'
function lerSalvas() {
  try {
    return limparOpcoes(JSON.parse(localStorage.getItem(GUARDADAS) || 'null'))
  } catch {
    return limparOpcoes(null)
  }
}
const opcoes = ref(lerSalvas())
watch(opcoes, (o) => {
  try {
    localStorage.setItem(GUARDADAS, JSON.stringify(o))
  } catch {
    /* sem armazenamento: vale só nesta tela */
  }
}, { deep: true })

const pedidos = ref([])
const erro = ref('')
const ocupado = ref(false)
const titulo = ref('')
const nota = ref('')
const pedidoLivre = ref('')
const campoTitulo = ref(null)

async function carregar() {
  try {
    pedidos.value = await api(`/api/projeto/itens/${encodeURIComponent(props.itemId)}/blog`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
const ultimo = computed(() => pedidos.value[0] || null)
const andando = computed(() => emAndamento(ultimo.value))
const guardadas = computed(() => ideias(pedidos.value))
// O mais novo aberto; os anteriores (até 5, mais os que têm marca) recolhidos.
const anteriores = computed(() => pedidos.value.slice(1).filter((p) => p.estado === 'pronta'))

let relogio = 0
watch(andando, (sim) => {
  clearInterval(relogio)
  if (sim) relogio = setInterval(carregar, 4000)
})
watch(() => props.itemId, () => {
  pedidos.value = []
  titulo.value = nota.value = ''
  carregar()
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

async function pedir(modo) {
  ocupado.value = true
  erro.value = ''
  try {
    await api(`/api/projeto/itens/${encodeURIComponent(props.itemId)}/blog`, {
      metodo: 'POST', tempo: 45000,
      corpo: { modo, titulo: titulo.value, nota: nota.value, opcoes: opcoes.value, pedido: pedidoLivre.value },
    })
    if (modo === 'topicos') titulo.value = nota.value = ''
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

async function marcar(pedido, n, estado, link) {
  try {
    const novo = await api(`/api/blog/${pedido.id}/marcas/${n}`, { metodo: 'PUT', corpo: { estado, link } })
    pedidos.value = pedidos.value.map((p) => (p.id === novo.id ? novo : p))
  } catch (e) {
    erro.value = e.message
  }
}

function pedirTopicosDe(t, tese) {
  titulo.value = t
  nota.value = tese || ''
  campoTitulo.value?.focus()
  campoTitulo.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

const quando = (iso) => (iso ? new Date(iso).toLocaleDateString('pt-BR', { day: 'numeric', month: 'short' }) : '')
const rotuloPedido = (p) => (p.modo === 'topicos' ? `Tópicos: ${p.titulo}` : `${p.resultado?.sugestoes?.length || 0} sugestões`)
</script>

<template>
  <section class="quadro blog">
    <div class="quadro-topo">
      <h2 class="rotulo-secao">Blog</h2>
      <span class="pequeno">você escreve; a IA só orienta</span>
    </div>

    <div v-if="guardadas.length" class="guardadas">
      <span class="rotulo">Ideias guardadas</span>
      <ul class="lista-guardadas">
        <li v-for="g in guardadas" :key="`${g.pedido}-${g.n}`">
          <div class="linha entre">
            <strong>{{ g.titulo }}</strong>
            <a v-if="g.link" class="etiqueta verde" :href="g.link" target="_blank" rel="noopener noreferrer">escrita ↗</a>
            <span v-else-if="g.estado === 'escrita'" class="etiqueta verde">escrita</span>
          </div>
          <p v-if="g.tese" class="pequeno sem-margem">{{ g.tese }}</p>
        </li>
      </ul>
    </div>

    <div class="caminhos">
      <div class="caminho">
        <p class="sem-margem">A IA lê o item (o que implementar, o estudo, o caderno) e sugere artigos que reúnem o que ele ensina.</p>
        <div>
          <button class="botao primario" :disabled="ocupado || andando" @click="pedir('sugerir')">Sugerir artigos</button>
        </div>
      </div>
      <div class="caminho">
        <label class="rotulo" for="blog-titulo">Tenho um título</label>
        <input id="blog-titulo" ref="campoTitulo" v-model="titulo" class="campo" placeholder="Ex.: GitHub Actions e o movimento ágil" @keyup.enter="titulo.trim() && pedir('topicos')" />
        <textarea v-model="nota" class="campo" rows="2" placeholder="O que você quer dizer com ele (opcional)" />
        <div>
          <button class="botao" :disabled="ocupado || andando || !titulo.trim()" @click="pedir('topicos')">Pedir tópicos</button>
        </div>
      </div>
    </div>

    <details class="engrenagens">
      <summary class="pequeno">Engrenagens · {{ resumoOpcoes(opcoes) }}</summary>
      <div class="campos">
        <label v-for="e in ENGRENAGENS" :key="e.nome">
          <span class="rotulo">{{ e.rotulo }}<template v-if="e.soSugerir"> (só nas sugestões)</template></span>
          <select v-model="opcoes[e.nome]" class="campo">
            <option v-for="[v, nome] in e.opcoes" :key="v" :value="v">{{ nome }}</option>
          </select>
        </label>
        <label>
          <span class="rotulo">Quantas sugestões</span>
          <select v-model.number="opcoes.quantas" class="campo">
            <option v-for="n in 5" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
      </div>
      <label class="pedido-livre">
        <span class="rotulo">Pedido livre (opcional)</span>
        <textarea v-model="pedidoLivre" class="campo" rows="2" placeholder="Ex.: quero algo que conecte CI com cultura de time." />
      </label>
    </details>

    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>
    <div v-if="andando" class="cartao">
      <p class="sem-margem">{{ fase(ultimo) }}</p>
    </div>
    <p v-else-if="ultimo?.estado === 'falhou'" class="aviso sem-margem">O pedido falhou: {{ ultimo.erro }}. Tente de novo.</p>

    <template v-if="ultimo?.estado === 'pronta'">
      <span class="pequeno">{{ rotuloPedido(ultimo) }} · {{ quando(ultimo.atualizado_em) }} · {{ resumoOpcoes(ultimo.opcoes, ultimo.modo) }}</span>
      <PedidoBlog :pedido="ultimo" @marcar="marcar" @topicos="pedirTopicosDe" />
    </template>

    <details v-for="p in anteriores" :key="p.id" class="anterior">
      <summary class="pequeno">{{ quando(p.atualizado_em) }} · {{ rotuloPedido(p) }}</summary>
      <PedidoBlog :pedido="p" @marcar="marcar" @topicos="pedirTopicosDe" />
    </details>
  </section>
</template>

<style scoped>
.blog { display: grid; gap: var(--e4); }
.sem-margem { margin: 0; }
.caminhos { display: grid; gap: var(--e4); grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); }
.caminho { display: grid; gap: var(--e2); align-content: start; font-size: 15px; line-height: 1.5; }
.caminho .rotulo { margin-bottom: 0; }
.engrenagens summary, .anterior summary { cursor: pointer; color: var(--texto-suave); }
.engrenagens .campos { margin-top: var(--e3); }
.pedido-livre { display: block; margin-top: var(--e3); }
.lista-guardadas { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.lista-guardadas strong { font-weight: 600; }
.anterior > .pedido-blog { margin-top: var(--e3); }
</style>
