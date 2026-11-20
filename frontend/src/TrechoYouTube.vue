<script setup>
// Trecho do YouTube na etapa (T39): o trecho É a aula — o player embutido começa no início, a
// barra mostra o trecho e, no fim (fim − 2 s), avisa "Trecho visto ✓" sem parar o vídeo e grava
// o visto no servidor. As notas com o minuto são as mesmas da Videoaula (T30).
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import BarraTrecho from './BarraTrecho.vue'
import { chaveDoTrecho, estadoDoTrecho, linkNoMinuto } from './trechoYoutube.js'
import { marcaDeTempo, momentosDasNotas, segundosParaTempo } from './videoaula.js'

const props = defineProps({
  trechos: { type: Array, required: true }, // de trechos(materiais): um por parte
  etapaId: { type: Number, required: true },
  vistos: { type: Array, default: () => [] }, // [{referencia, localizador, visto_em}]
  proxima: { type: String, default: '' }, // link da próxima etapa, se houver
})
const emit = defineEmits(['mudou'])

const parte = ref(0)
const raiz = ref(null)
const moldura = ref(null)
const posicao = ref(props.trechos[0]?.inicio ?? 0)
const semEmbed = ref(false)
const aviso = ref(false)
const erroVisto = ref('')
const vistosLocal = reactive(new Set())
watch(() => props.vistos, (vs) => {
  for (const v of vs || []) vistosLocal.add(`${v.referencia}|${v.localizador}`)
}, { immediate: true })

const atual = computed(() => props.trechos[parte.value])
const estado = computed(() => estadoDoTrecho(posicao.value, atual.value?.inicio ?? 0, atual.value?.fim ?? 0))
const visto = computed(() => vistosLocal.has(chaveDoTrecho(atual.value || {})))

// ---------- a IFrame API do YouTube (carrega uma vez para o app inteiro) ----------
let pedidoApi = null
function apiDoYouTube() {
  if (window.YT?.Player) return Promise.resolve(window.YT)
  if (!pedidoApi) {
    pedidoApi = new Promise((resolve, reject) => {
      const antes = window.onYouTubeIframeAPIReady
      window.onYouTubeIframeAPIReady = () => { antes?.(); resolve(window.YT) }
      const s = document.createElement('script')
      s.src = 'https://www.youtube.com/iframe_api'
      s.onerror = () => reject(new Error('a API do YouTube não carregou (sem internet?)'))
      document.head.appendChild(s)
    })
  }
  return pedidoApi
}

// ---------- o player ----------
let player = null
let relogio = null
let playerId = 0
async function montar() {
  semEmbed.value = false
  await nextTick() // o div novo da parte (ou o fim do semEmbed) precisa estar no DOM
  const t = atual.value
  if (!t || !moldura.value) return
  const eu = ++playerId
  try {
    const YT = await apiDoYouTube()
    if (eu !== playerId) return // trocou de parte enquanto carregava
    player = new YT.Player(moldura.value, {
      videoId: t.videoId,
      playerVars: { start: t.inicio, rel: 0, modestbranding: 1, playsinline: 1 },
      events: { onStateChange: aoMudar, onError: aoErro },
    })
  } catch {
    semEmbed.value = true
  }
}
function aoMudar(ev) {
  if (ev.data === window.YT?.PlayerState?.PLAYING) {
    tocou()
    if (!relogio) {
      posicao.value = player?.getCurrentTime?.() || 0
      relogio = setInterval(andar, 500)
    }
  } else {
    clearInterval(relogio)
    relogio = null
  }
}
function andar() {
  posicao.value = player?.getCurrentTime?.() || 0
  if (estado.value.passou && !visto.value) marcarVisto()
}
function aoErro(ev) {
  // 101/150: o dono do vídeo não deixa embutir; o cartão leva ao YouTube no minuto.
  if (ev.data !== 101 && ev.data !== 150) return
  clearInterval(relogio)
  relogio = null
  player?.destroy?.()
  player = null
  semEmbed.value = true
}
watch(parte, async () => {
  clearInterval(relogio)
  relogio = null
  aviso.value = false
  posicao.value = atual.value?.inicio || 0
  player?.destroy?.()
  player = null
  await montar()
})
onMounted(montar)
onBeforeUnmount(() => {
  clearInterval(relogio)
  player?.destroy?.()
  if (relogioNotas) salvar()
})

function voltar() {
  player?.seekTo?.(atual.value.inicio, true)
  player?.playVideo?.()
}

// ir: a marca das notas ou o "onde" da conferência levam ao ponto (parte começa em 1).
function ir(numeroParte, segundos) {
  const i = Math.min(Math.max((numeroParte || 1) - 1, 0), props.trechos.length - 1)
  raiz.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  if (i !== parte.value) {
    parte.value = i
    const esperar = setInterval(() => {
      if (player?.seekTo) {
        clearInterval(esperar)
        player.seekTo(segundos, true)
        player.playVideo?.()
      }
    }, 300)
    setTimeout(() => clearInterval(esperar), 8000)
    return
  }
  player?.seekTo?.(segundos, true)
  player?.playVideo?.()
}
defineExpose({ ir })

// ---------- visto e andamento ----------
let tocouPedido = false
async function tocou() {
  if (tocouPedido) return
  tocouPedido = true
  try {
    await api(`/api/etapas/${props.etapaId}/trechos/visto`, { metodo: 'POST', corpo: { tocou: true } })
    emit('mudou')
  } catch { /* sem internet ou servidor: só não marca */ }
}
const pedidos = new Set()
async function marcarVisto(manual = false) {
  const t = atual.value
  const chave = chaveDoTrecho(t)
  if (pedidos.has(chave) || vistosLocal.has(chave)) return
  pedidos.add(chave)
  erroVisto.value = ''
  try {
    await api(`/api/etapas/${props.etapaId}/trechos/visto`, {
      metodo: 'POST',
      corpo: { referencia: t.referencia, localizador: t.localizador },
    })
    vistosLocal.add(chave)
    aviso.value = !manual
    emit('mudou')
  } catch (err) {
    erroVisto.value = err.message || 'Não marcou: tente de novo.'
  } finally {
    pedidos.delete(chave)
  }
}

// ---------- notas (as mesmas da Videoaula, com o tempo deste player) ----------
const area = ref(null)
const notas = ref('')
const estadoNotas = ref('')
const momentos = computed(() => momentosDasNotas(notas.value))
let carregadas = false
async function carregar() {
  try {
    const an = await api(`/api/etapas/${props.etapaId}/anotacoes`)
    notas.value = an.notas
  } catch {
    estadoNotas.value = 'erro'
  }
  carregadas = true
}
onMounted(carregar)
let relogioNotas = null
watch(notas, () => {
  if (!carregadas) return
  estadoNotas.value = 'salvando'
  clearTimeout(relogioNotas)
  relogioNotas = setTimeout(salvar, 900)
})
let salvo = ''
async function salvar() {
  clearTimeout(relogioNotas)
  relogioNotas = null
  if (notas.value === salvo && estadoNotas.value !== 'salvando') return
  try {
    const texto = notas.value
    await api(`/api/etapas/${props.etapaId}/anotacoes`, { metodo: 'PUT', corpo: { notas: texto } })
    salvo = texto
    if (notas.value === texto) estadoNotas.value = 'salvo'
  } catch {
    estadoNotas.value = 'erro'
  }
}
function marcar() {
  const t = player?.getCurrentTime?.() || posicao.value || 0
  const marca = marcaDeTempo(t, parte.value + 1, props.trechos.length) + ' '
  const el = area.value
  const ini = el ? el.selectionStart : notas.value.length
  const fim = el ? el.selectionEnd : notas.value.length
  const antes = notas.value.slice(0, ini)
  const quebra = antes && !antes.endsWith('\n') ? '\n' : ''
  notas.value = antes + quebra + marca + notas.value.slice(fim)
  const pos = (antes + quebra + marca).length
  requestAnimationFrame(() => {
    el?.focus()
    el?.setSelectionRange(pos, pos)
  })
}
function teclas(ev) {
  if ((ev.ctrlKey || ev.metaKey) && ev.key === 'Enter') {
    ev.preventDefault()
    marcar()
  }
}
</script>

<template>
  <section id="trecho" ref="raiz" class="trecho">
    <div v-if="trechos.length > 1" class="abas partes" role="tablist" aria-label="Partes da aula">
      <button v-for="(t, i) in trechos" :key="chaveDoTrecho(t)" class="aba" role="tab" :aria-selected="i === parte" :class="{ ativa: i === parte }" @click="parte = i">
        Parte {{ i + 1 }} <span v-if="vistosLocal.has(chaveDoTrecho(t))" class="feito">✓</span>
      </button>
    </div>

    <!-- O player embutido; se o vídeo não deixar embutir, um cartão leva ao YouTube no minuto.
         A API do YouTube troca o div pelo iframe — a :key repõe um div limpo a cada parte. -->
    <template v-if="!semEmbed">
      <div class="moldura"><div ref="moldura" :key="parte" class="yt" /></div>
      <BarraTrecho :titulo="atual.titulo" :inicio="atual.inicio" :fim="atual.fim" :posicao="posicao" :aviso="aviso" :proxima="proxima" @continuar="aviso = false">
        <button class="botao discreto" type="button" @click="voltar">Voltar ao início do trecho</button>
        <a class="botao discreto" :href="linkNoMinuto(atual.videoId, atual.inicio)" target="_blank" rel="noopener noreferrer">
          <Icone nome="video" :tamanho="16" /> Abrir no YouTube em {{ segundosParaTempo(atual.inicio) }}
        </a>
      </BarraTrecho>
    </template>
    <div v-else class="cartao sem-embed">
      <img :src="`https://i.ytimg.com/vi/${atual.videoId}/mqdefault.jpg`" alt="" loading="lazy" @error="(e) => (e.target.style.display = 'none')" />
      <div>
        <strong>{{ atual.titulo || 'Vídeo do YouTube' }}</strong>
        <p class="suave">Este vídeo não deixa embutir aqui. Assista no YouTube — ele já abre no começo do trecho ({{ segundosParaTempo(atual.inicio) }} → {{ segundosParaTempo(atual.fim) }}).</p>
        <div class="linha">
          <a class="botao" :href="linkNoMinuto(atual.videoId, atual.inicio)" target="_blank" rel="noopener noreferrer">Abrir no YouTube em {{ segundosParaTempo(atual.inicio) }}</a>
          <button v-if="!visto" class="botao discreto" type="button" @click="marcarVisto(true)">Marquei como visto</button>
          <span v-else class="etiqueta verde">Trecho visto ✓</span>
        </div>
      </div>
    </div>

    <p v-if="erroVisto" class="aviso">{{ erroVisto }}</p>

    <div class="notas">
      <div class="linha entre">
        <label class="rotulo sem-margem" for="notas-trecho">Minhas notas</label>
        <span class="pequeno">{{ { salvando: 'Salvando…', salvo: 'Salvo', erro: 'Não salvou: tente de novo' }[estadoNotas] || '' }}</span>
      </div>
      <textarea
        id="notas-trecho"
        ref="area"
        v-model="notas"
        class="campo"
        rows="6"
        spellcheck="true"
        placeholder="Anote enquanto assiste. “Marcar o minuto” (ou Ctrl/⌘ + Enter) põe o tempo do vídeo na nota, e a marca leva de volta àquele ponto."
        @keydown="teclas"
        @blur="relogioNotas && salvar()"
      />
      <div class="linha">
        <button class="botao" type="button" @click="marcar">Marcar o minuto</button>
        <span class="pequeno">{{ segundosParaTempo(posicao) }}</span>
      </div>
      <ul v-if="momentos.length" class="momentos">
        <li v-for="(m, i) in momentos" :key="i">
          <button class="botao discreto tempo" type="button" @click="ir(m.parte, m.segundos)">
            <template v-if="trechos.length > 1">P{{ m.parte }} · </template>{{ m.tempo }}
          </button>
          <span class="suave">{{ m.texto || '—' }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.trecho { margin-top: var(--e5); }
.partes { margin-bottom: var(--e3); }
.aba { padding: 6px var(--e3); border-radius: var(--raio-p); font-size: 14px; color: var(--texto-suave); }
.aba.ativa { background: var(--ativo); color: var(--texto); }
.feito { color: var(--verde); }
.moldura { background: var(--video); border-radius: var(--raio); overflow: hidden; border: 1px solid var(--borda); }
.moldura .yt { display: block; width: 100%; aspect-ratio: 16 / 9; background: var(--video); }
.moldura .yt :deep(iframe) { width: 100%; height: 100%; display: block; border: 0; }
.sem-embed { display: flex; gap: var(--e3); align-items: flex-start; }
.sem-embed img { width: 168px; border-radius: var(--raio-p); flex: none; }
.sem-embed p { margin: var(--e1) 0 var(--e3); }
.notas { margin-top: var(--e5); display: grid; gap: var(--e2); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.momentos { list-style: none; margin: var(--e2) 0 0; padding: 0; display: grid; gap: var(--e1); }
.momentos li { display: flex; gap: var(--e2); align-items: baseline; font-size: 14px; }
.tempo { font-family: var(--fonte-mono); font-size: 12.5px; padding: 0 var(--e2); flex: none; }
@media (max-width: 560px) { .sem-embed { flex-direction: column; } .sem-embed img { width: 100%; } }
</style>
