<script setup>
// Videoaula (T30): o vídeo principal da etapa no topo, como numa aula, e as notas que o Lucas toma
// assistindo (salvas sozinhas; "Marcar o minuto" escreve [mm:ss] e cada marca volta ao ponto).
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import BarraTrecho from './BarraTrecho.vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { aulaDaPlataforma, ehAulaDoTelegram, estadoDaAulaBaixada, linkNoStreaming, marcaDeTempo, momentosDasNotas, segundosParaTempo, tempoEstimado } from './videoaula.js'
import { chaveDoTrecho, estadoDoTrecho, intervaloDoMaterial } from './trechoYoutube.js'
import { tamanho } from './pastas.js'

const props = defineProps({
  videos: { type: Array, required: true },
  etapaId: { type: Number, required: true },
  vistos: { type: Array, default: () => [] }, // [{referencia, localizador, visto_em}]
  proxima: { type: String, default: '' }, // link da próxima etapa, se houver
})
const emit = defineEmits(['mudou'])

const parte = ref(0) // índice do vídeo tocando
const video = ref(null)
const area = ref(null)
const notas = ref('')
const estado = ref('') // '', 'salvando', 'salvo', 'erro'
const erroVideo = ref('')
const VELOCIDADES = [1, 1.25, 1.5, 1.75, 2]
const velocidade = ref(lerLocal('videoaula:velocidade', 1))

const atual = computed(() => props.videos[parte.value])
const momentos = computed(() => momentosDasNotas(notas.value))

// ---------- trecho de aula do Telegram (T44) ----------
// Material streaming com ref telegram: e localizador "mm:ss–mm:ss": começa no início do
// trecho e mostra a mesma barra/aviso do TrechoYouTube (o comum está em BarraTrecho.vue).
const trecho = computed(() => intervaloDoMaterial(atual.value))
const posicao = ref(0)
const estadoTrecho = computed(() => (trecho.value ? estadoDoTrecho(posicao.value, trecho.value.inicio, trecho.value.fim) : null))
const avisoVisto = ref(false)
const erroVisto = ref('')
const vistosLocal = reactive(new Set())
watch(() => props.vistos, (vs) => {
  for (const v of vs || []) vistosLocal.add(`${v.referencia}|${v.localizador}`)
}, { immediate: true })

let tocouPedido = false
async function tocou() {
  if (tocouPedido) return
  tocouPedido = true
  try {
    await api(`/api/etapas/${props.etapaId}/trechos/visto`, { metodo: 'POST', corpo: { tocou: true } })
    emit('mudou')
  } catch { /* sem servidor: só não marca */ }
}
const pedidosVisto = new Set()
async function marcarVisto() {
  const t = trecho.value
  const chave = chaveDoTrecho(t)
  if (pedidosVisto.has(chave) || vistosLocal.has(chave)) return
  pedidosVisto.add(chave)
  erroVisto.value = ''
  try {
    await api(`/api/etapas/${props.etapaId}/trechos/visto`, {
      metodo: 'POST',
      corpo: { referencia: t.referencia, localizador: t.localizador },
    })
    vistosLocal.add(chave)
    avisoVisto.value = true
    emit('mudou')
  } catch (err) {
    erroVisto.value = err.message || 'Não marcou: tente de novo.'
  } finally {
    pedidosVisto.delete(chave)
  }
}
function voltarAoTrecho() {
  if (!video.value || !trecho.value) return
  video.value.currentTime = trecho.value.inicio
  video.value.play().catch(() => {})
}

// ---------- aulas do Telegram (T42) ----------
// infos[i]: onde a parte i está — baixada (com a url do vídeo), na_fila, no_telegram, carregando
// ou erro. Um material streaming comum já nasce "baixada" (a referência é a própria url).
// T55: a aula baixada pela Trilha (trilha montada de um curso) pergunta à Entrada › Cursos se já baixou.
const precisaPerguntar = (ref) => ehAulaDoTelegram(ref) || aulaDaPlataforma(ref) > 0
const infos = ref(props.videos.map((m) => ({ estado: precisaPerguntar(m.referencia) ? 'carregando' : 'baixada', url: m.referencia })))
const infoDoAtual = computed(() => infos.value[parte.value] || { estado: 'carregando' })
const srcAtual = computed(() => infoDoAtual.value.url || atual.value?.referencia)
const faltando = computed(() => infos.value.filter((i) => i.estado === 'no_telegram').length)

let parou = false
async function resolver(i) {
  if (parou) return
  const m = props.videos[i]
  const idCurso = aulaDaPlataforma(m.referencia)
  if (idCurso) {
    let aula = null
    try {
      aula = await api(`/api/cursos/aulas/${idCurso}`)
    } catch (e) {
      if (e.status !== 404) {
        if (!parou) infos.value[i] = { estado: 'erro', mensagem: e.message || 'Não deu para perguntar à Trilha.' }
        return
      }
    }
    if (parou) return
    infos.value[i] = estadoDaAulaBaixada(aula, m.referencia)
    if (infos.value[i].estado === 'baixando_curso') setTimeout(() => infos.value[i]?.estado === 'baixando_curso' && resolver(i), 5000)
    return
  }
  if (!ehAulaDoTelegram(m.referencia)) return // já nasceu "baixada"
  try {
    const a = await api(`/api/arquivos/aula?ref=${encodeURIComponent(m.referencia)}`)
    if (parou) return
    infos.value[i] = { ...a }
    if (a.estado === 'na_fila') setTimeout(() => infos.value[i]?.estado === 'na_fila' && resolver(i), 5000)
  } catch (e) {
    if (!parou) infos.value[i] = { estado: 'erro', mensagem: e.message || 'Não deu para perguntar ao Streaming.' }
  }
}
onMounted(() => props.videos.forEach((_, i) => infos.value[i].estado === 'carregando' && resolver(i)))

async function tentarCurso(i) {
  const id = infos.value[i]?.id
  infos.value[i] = { estado: 'carregando' }
  try {
    await api(`/api/cursos/aulas/${id}/tentar`, { metodo: 'POST' })
  } catch { /* a resposta de resolver diz o que houve */ }
  resolver(i)
}

async function baixar(i) {
  const m = props.videos[i]
  infos.value[i] = { ...infos.value[i], baixando: true }
  try {
    await api('/api/arquivos/aula/baixar', { metodo: 'POST', corpo: { ref: m.referencia } })
    infos.value[i] = { estado: 'na_fila' }
    setTimeout(() => infos.value[i]?.estado === 'na_fila' && resolver(i), 5000)
  } catch (e) {
    infos.value[i] = { estado: 'erro', mensagem: e.message || 'O pedido de download falhou.' }
  }
}
function baixarTodas() {
  props.videos.forEach((_, i) => infos.value[i]?.estado === 'no_telegram' && baixar(i))
}

function lerLocal(chave, padrao) {
  try {
    const v = localStorage.getItem(chave)
    return v === null ? padrao : JSON.parse(v)
  } catch {
    return padrao
  }
}
function gravarLocal(chave, valor) {
  try {
    localStorage.setItem(chave, JSON.stringify(valor))
  } catch {
    /* sem armazenamento: só não lembra */
  }
}

// ---------- vídeo ----------
let pendenteIr = null
function aoCarregar() {
  const v = video.value
  v.playbackRate = velocidade.value
  if (pendenteIr !== null) {
    v.currentTime = pendenteIr
    pendenteIr = null
    v.play().catch(() => {})
    return
  }
  if (trecho.value) {
    // Material de trecho: começa no início dele (a posição salva serve só quando já está
    // dentro do trecho — fora dele não faz sentido voltar).
    const onde = lerLocal(`videoaula:${atual.value.referencia}`, 0)
    v.currentTime = onde >= trecho.value.inicio && onde < trecho.value.fim ? onde : trecho.value.inicio
    posicao.value = v.currentTime
    return
  }
  const onde = lerLocal(`videoaula:${atual.value.referencia}`, 0)
  if (onde > 5 && onde < v.duration - 10) v.currentTime = onde
}
let ultimoGuardado = 0
function aoAndar() {
  const t = video.value?.currentTime || 0
  posicao.value = t
  if (trecho.value) {
    tocou()
    if (estadoTrecho.value?.passou && !vistosLocal.has(chaveDoTrecho(trecho.value))) marcarVisto()
  }
  if (Math.abs(t - ultimoGuardado) >= 5) {
    ultimoGuardado = t
    gravarLocal(`videoaula:${atual.value.referencia}`, Math.floor(t))
  }
}
function aoErro() {
  erroVideo.value = 'O vídeo não abriu. Confira se o Streaming está no ar, ou abra direto nele.'
}
function mudarVelocidade(v) {
  velocidade.value = v
  gravarLocal('videoaula:velocidade', v)
  if (video.value) video.value.playbackRate = v
}
watch(parte, () => {
  erroVideo.value = ''
  erroVisto.value = ''
  avisoVisto.value = false
  posicao.value = 0
  ultimoGuardado = 0
})

// ir: a marca das notas ou o "onde" da conferência levam ao ponto (parte começa em 1).
function ir(numeroParte, segundos) {
  const i = Math.min(Math.max((numeroParte || 1) - 1, 0), props.videos.length - 1)
  video.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  if (i !== parte.value) {
    pendenteIr = segundos
    parte.value = i
    return
  }
  video.value.currentTime = segundos
  video.value.play().catch(() => {})
}
defineExpose({ ir })

// ---------- notas ----------
let carregadas = false
async function carregar() {
  try {
    const an = await api(`/api/etapas/${props.etapaId}/anotacoes`)
    notas.value = an.notas
  } catch {
    estado.value = 'erro'
  }
  carregadas = true
}
onMounted(carregar)

let relogio = null
watch(notas, () => {
  if (!carregadas) return
  estado.value = 'salvando'
  clearTimeout(relogio)
  relogio = setTimeout(salvar, 900)
})
let salvo = ''
async function salvar() {
  clearTimeout(relogio)
  relogio = null
  if (notas.value === salvo && estado.value !== 'salvando') return
  try {
    const texto = notas.value
    await api(`/api/etapas/${props.etapaId}/anotacoes`, { metodo: 'PUT', corpo: { notas: texto } })
    salvo = texto
    if (notas.value === texto) estado.value = 'salvo'
  } catch {
    estado.value = 'erro'
  }
}
onBeforeUnmount(() => {
  parou = true
  if (relogio) salvar()
})

function marcar() {
  if (infoDoAtual.value.estado !== 'baixada') return
  const t = video.value?.currentTime || 0
  const marca = marcaDeTempo(t, parte.value + 1, props.videos.length) + ' '
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
// Atalho dentro das notas: Ctrl/⌘ + Enter marca o minuto.
function teclas(ev) {
  if ((ev.ctrlKey || ev.metaKey) && ev.key === 'Enter') {
    ev.preventDefault()
    marcar()
  }
}
</script>

<template>
  <section id="videoaula" class="videoaula">
    <div v-if="videos.length > 1" class="abas partes" role="tablist" aria-label="Partes da aula">
      <button v-for="(v, i) in videos" :key="v.referencia" class="aba" role="tab" :aria-selected="i === parte" :class="{ ativa: i === parte }" @click="parte = i">
        Parte {{ i + 1 }}
      </button>
    </div>
    <div class="moldura">
      <video
        v-if="infoDoAtual.estado === 'baixada'"
        :key="srcAtual"
        ref="video"
        :src="srcAtual"
        controls
        preload="metadata"
        playsinline
        @loadedmetadata="aoCarregar"
        @timeupdate="aoAndar"
        @pause="aoAndar"
        @error="aoErro"
      />
      <div v-else class="espera">
        <template v-if="infoDoAtual.estado === 'carregando'">
          <p class="suave">Procurando o vídeo…</p>
        </template>
        <template v-else-if="infoDoAtual.estado === 'no_telegram'">
          <p class="titulo-espera">Esta aula ainda não está no servidor</p>
          <p class="suave">
            {{ tamanho(infoDoAtual.tamanho) }}<template v-if="infoDoAtual.duracao"> · {{ segundosParaTempo(infoDoAtual.duracao) }}</template> · {{ tempoEstimado(infoDoAtual.tamanho) }} para baixar
          </p>
          <div class="linha centro">
            <button class="botao" :disabled="infoDoAtual.baixando" @click="baixar(parte)">{{ infoDoAtual.baixando ? 'Pedindo…' : 'Baixar esta aula' }}</button>
            <button v-if="faltando > 1" class="botao discreto" @click="baixarTodas">Baixar as {{ faltando }} partes</button>
          </div>
        </template>
        <template v-else-if="infoDoAtual.estado === 'baixando_curso'">
          <p class="titulo-espera">{{ infoDoAtual.naFila ? 'Na fila de downloads' : `Baixando… ${infoDoAtual.progresso}%` }}</p>
          <p class="suave">A Trilha está baixando esta aula do curso (Entrada › Cursos). O vídeo aparece aqui quando terminar.</p>
        </template>
        <template v-else-if="infoDoAtual.estado === 'sem_video'">
          <p class="titulo-espera">Esta aula não tem vídeo</p>
          <p class="suave">Na plataforma ela é só texto ou material: está nas Fontes da etapa.</p>
        </template>
        <template v-else-if="infoDoAtual.estado === 'erro_curso'">
          <p class="titulo-espera">O download desta aula deu erro</p>
          <p class="suave">{{ infoDoAtual.mensagem }}</p>
          <button class="botao" @click="tentarCurso(parte)">Tentar de novo</button>
        </template>
        <template v-else-if="infoDoAtual.estado === 'na_fila'">
          <p class="titulo-espera">Baixando…</p>
          <p class="suave">A aula está no Telegram: já pedi ao Streaming. O vídeo aparece aqui quando terminar.</p>
        </template>
        <template v-else>
          <p class="titulo-espera">Não deu para perguntar ao Streaming</p>
          <p class="suave">{{ infoDoAtual.mensagem }}</p>
          <button class="botao" @click="resolver(parte)">Tentar de novo</button>
        </template>
      </div>
    </div>
    <BarraTrecho v-if="trecho && infoDoAtual.estado === 'baixada'" :titulo="atual.titulo" :inicio="trecho.inicio" :fim="trecho.fim" :posicao="posicao" :aviso="avisoVisto" :proxima="proxima" @continuar="avisoVisto = false">
      <button class="botao discreto" type="button" @click="voltarAoTrecho">Voltar ao início do trecho</button>
    </BarraTrecho>
    <div class="linha barra">
      <span v-if="!trecho" class="titulo-video">{{ atual.titulo }}</span>
      <span class="espaco" />
      <div v-if="infoDoAtual.estado === 'baixada'" class="linha velocidades" role="group" aria-label="Velocidade">
        <button v-for="v in VELOCIDADES" :key="v" class="botao discreto vel" :class="{ ativa: v === velocidade }" @click="mudarVelocidade(v)">{{ v }}×</button>
      </div>
      <a v-if="infoDoAtual.estado === 'baixada' && !aulaDaPlataforma(atual.referencia)" class="botao discreto" :href="linkNoStreaming(srcAtual)" target="_blank" rel="noopener noreferrer" title="Abrir no Streaming (tela cheia, TV, projetor)">
        <Icone nome="video" :tamanho="16" /> Streaming
      </a>
    </div>
    <p v-if="erroVideo" class="aviso">{{ erroVideo }}</p>
    <p v-if="erroVisto" class="aviso">{{ erroVisto }}</p>

    <div class="notas">
      <div class="linha entre">
        <label class="rotulo sem-margem" for="notas-aula">Minhas notas</label>
        <span class="pequeno">{{ { salvando: 'Salvando…', salvo: 'Salvo', erro: 'Não salvou: tente de novo' }[estado] || '' }}</span>
      </div>
      <textarea
        id="notas-aula"
        ref="area"
        v-model="notas"
        class="campo"
        rows="6"
        spellcheck="true"
        placeholder="Anote enquanto assiste. “Marcar o minuto” (ou Ctrl/⌘ + Enter) põe o tempo do vídeo na nota, e a marca leva de volta àquele ponto."
        @keydown="teclas"
        @blur="relogio && salvar()"
      />
      <div class="linha">
        <button class="botao" type="button" @click="marcar">Marcar o minuto</button>
        <span class="pequeno">{{ segundosParaTempo(video?.currentTime) }}</span>
      </div>
      <ul v-if="momentos.length" class="momentos">
        <li v-for="(m, i) in momentos" :key="i">
          <button class="botao discreto tempo" type="button" @click="ir(m.parte, m.segundos)">
            <template v-if="videos.length > 1">P{{ m.parte }} · </template>{{ m.tempo }}
          </button>
          <span class="suave">{{ m.texto || '—' }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<style scoped>
.videoaula { margin-top: var(--e5); }
.partes { margin-bottom: var(--e3); }
.aba { padding: 6px var(--e3); border-radius: var(--raio-p); font-size: 14px; color: var(--texto-suave); }
.aba.ativa { background: var(--ativo); color: var(--texto); }
.moldura { background: var(--video); border-radius: var(--raio); overflow: hidden; border: 1px solid var(--borda); }
.moldura video { display: block; width: 100%; aspect-ratio: 16 / 9; background: var(--video); }
.espera { aspect-ratio: 16 / 9; display: grid; place-content: center; justify-items: center; gap: var(--e2); padding: var(--e4); text-align: center; }
.titulo-espera { margin: 0; font-weight: 600; }
.centro { justify-content: center; flex-wrap: wrap; }
.barra { margin-top: var(--e2); }
.titulo-video { font-size: 14px; color: var(--texto-suave); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1 1 200px; }
.espaco { flex: 0 0 0; }
.velocidades { gap: 2px; }
.vel { padding: 0 var(--e2); font-size: 13px; font-variant-numeric: tabular-nums; }
.vel.ativa { background: var(--ativo); color: var(--texto); }
.notas { margin-top: var(--e5); display: grid; gap: var(--e2); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.momentos { list-style: none; margin: var(--e2) 0 0; padding: 0; display: grid; gap: var(--e1); }
.momentos li { display: flex; gap: var(--e2); align-items: baseline; font-size: 14px; }
.tempo { font-family: var(--fonte-mono); font-size: 12.5px; padding: 0 var(--e2); flex: none; }
</style>
