<script setup>
import { inject, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../api.js'
import { linkDoYouTube } from '../entrada.js'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'
import { ESTADOS_FONTE } from '../rotulos.js'
import { segundosParaTempo } from '../videoaula.js'

// Aviso do compartilhar do celular (/compartilhar): o App manda o link e deixa o recado aqui.
const avisoCompartilhar = inject('avisoCompartilhar', ref(''))

const fontes = ref([])
const campo = ref('')
const mandando = ref(false)
const erro = ref('')
const semCapa = ref({})
let timer = null

function ligarTimer() {
  if (timer) clearInterval(timer)
  timer = fontes.value.some((f) => ['na_fila', 'lendo', 'fragmentando'].includes(f.estado)) ? setInterval(buscar, 3000) : null
}

async function buscar() {
  try {
    fontes.value = await api('/api/entrada')
    ligarTimer()
  } catch (e) {
    erro.value = e.message
  }
}

async function colar() {
  try {
    campo.value = await navigator.clipboard.readText()
  } catch {
    // Sem permissão ou área de transferência vazia: o Lucas digita à mão.
  }
}

async function mandar() {
  const url = linkDoYouTube(campo.value)
  if (!url) {
    erro.value = 'Não achei um link do YouTube nesse texto.'
    return
  }
  mandando.value = true
  erro.value = ''
  try {
    await api('/api/entrada', { metodo: 'POST', corpo: { url } })
    campo.value = ''
    await buscar()
  } catch (e) {
    erro.value = e.message
  } finally {
    mandando.value = false
  }
}

async function tentar(f) {
  await api(`/api/entrada/${f.id}/tentar`, { metodo: 'POST' })
  await buscar()
}

async function apagar(f) {
  if (!confirm(`Apagar "${f.titulo || f.url}" da Entrada?`)) return
  await api(`/api/entrada/${f.id}`, { metodo: 'DELETE' })
  await buscar()
}

const capa = (f) => `https://i.ytimg.com/vi/${f.video_id}/mqdefault.jpg`
const duracao = (f) => (f.duracao ? segundosParaTempo(f.duracao) : '')
const estado = (f) => ESTADOS_FONTE[f.estado] || { nome: f.estado, cor: '' }

onMounted(buscar)
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  avisoCompartilhar.value = ''
})
</script>

<template>
  <Pagina>
  <div class="pilha">
    <header class="cabecalho">
      <div>
        <h1>Entrada</h1>
        <p class="pequeno">Vídeos e playlists do YouTube que ainda não viraram fragmentos de uma trilha.</p>
      </div>
    </header>

    <p v-if="avisoCompartilhar" class="aviso neutro">{{ avisoCompartilhar }}</p>

    <form @submit.prevent="mandar">
      <label class="rotulo" for="link-yt">Link do YouTube (vídeo ou playlist)</label>
      <div class="linha entrada-form">
        <input id="link-yt" v-model="campo" class="campo" placeholder="https://youtu.be/…" inputmode="url" />
        <button type="button" class="botao" @click="colar">Colar</button>
        <button class="botao primario" :disabled="mandando">Mandar</button>
      </div>
      <p class="ajuda">Só a legenda (ou o Whisper, quando não tem) é lida — nada de baixar o vídeo.</p>
    </form>

    <p v-if="erro" class="aviso">{{ erro }}</p>

    <p v-if="!fontes.length" class="fraco">Nenhuma fonte ainda.</p>

    <div v-for="f in fontes" :key="f.id" class="cartao fonte">
      <a class="fonte-info" :href="link('entrada', String(f.id))">
        <img v-if="!semCapa[f.id]" class="capa" :src="capa(f)" alt="" loading="lazy" @error="semCapa[f.id] = true" />
        <div v-else class="capa capa-vazia"><Icone nome="video" :tamanho="20" /></div>
        <div class="fonte-texto">
          <strong>{{ f.titulo || 'Lendo o vídeo…' }}</strong>
          <span class="pequeno">{{ [f.canal, duracao(f)].filter(Boolean).join(' · ') }}</span>
          <span v-if="f.playlist" class="pequeno">da playlist {{ f.playlist }}</span>
        </div>
      </a>
      <div class="fonte-lado">
        <span class="etiqueta" :class="estado(f).cor">{{ estado(f).nome }}</span>
        <button class="botao discreto icone" title="Apagar a fonte" aria-label="Apagar a fonte" @click="apagar(f)">
          <Icone nome="fechar" />
        </button>
      </div>
      <div v-if="f.estado === 'erro'" class="fonte-erro">
        <p class="aviso">{{ f.erro }}</p>
        <button class="botao" @click="tentar(f)">Tentar de novo</button>
      </div>
    </div>
  </div>
  </Pagina>
</template>

<style scoped>
.entrada-form .campo {
  flex: 1;
  min-width: 0;
}
.fonte {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: var(--e3);
}
.fonte-info {
  display: flex;
  gap: var(--e3);
  min-width: 0;
  color: inherit;
  text-decoration: none;
}
.capa {
  flex: none;
  width: 112px;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  border-radius: var(--raio-p);
  background: var(--superficie-2);
}
.capa-vazia {
  display: grid;
  place-items: center;
  color: var(--texto-fraco);
}
.fonte-texto {
  display: grid;
  gap: 2px;
  min-width: 0;
  align-content: start;
}
.fonte-texto strong {
  font-size: 15px;
  line-height: 1.35;
  font-weight: 600;
}
.fonte-lado {
  display: grid;
  justify-items: end;
  gap: var(--e2);
}
.fonte-erro {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: var(--e3);
  flex-wrap: wrap;
}
.fonte-erro .aviso {
  margin: 0;
}
@media (max-width: 520px) {
  .capa {
    width: 88px;
  }
}
</style>
