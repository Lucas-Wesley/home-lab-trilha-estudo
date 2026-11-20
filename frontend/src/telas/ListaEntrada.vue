<script setup>
import { computed, inject, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../api.js'
import { linkDoYouTube } from '../entrada.js'
import AbasPagina from '../AbasPagina.vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'
import { ESTADOS_FONTE } from '../rotulos.js'
import { idDoVideo } from '../trechoYoutube.js'
import { segundosParaTempo } from '../videoaula.js'

// Aviso do compartilhar do celular (/compartilhar): o App manda o link e deixa o recado aqui.
const avisoCompartilhar = inject('avisoCompartilhar', ref(''))

// Abas da Entrada (T41): os vídeos do YouTube aqui; os cursos do Telegram em "Meus arquivos".
const abas = [
  { id: 'videos', nome: 'Vídeos' },
  { id: 'arquivos', nome: 'Meus arquivos' },
  { id: 'cursos', nome: 'Cursos' }, // T50: as aulas da Branas.io e da LINUXtips
]
const mudarAba = (id) => {
  if (id !== 'videos') location.hash = `#/entrada/${id}`
}

const fontes = ref([])
const campo = ref('')
const mandando = ref(false)
const erro = ref('')
const semCapa = ref({})
let timer = null

function ligarTimer() {
  if (timer) clearInterval(timer)
  timer = fontes.value.some((f) => ['na_fila', 'lendo', 'baixando', 'transcrevendo', 'fragmentando'].includes(f.estado)) ? setInterval(buscar, 3000) : null
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
    // T46: divide e sugere as trilhas sozinho (vídeo de até 3 h).
    await api('/api/entrada', { metodo: 'POST', corpo: { url, automatico: true } })
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

// A fonte vinda do Acervo tem video_id = id da Referência — a miniatura sai do link do vídeo.
const capa = (f) => `https://i.ytimg.com/vi/${idDoVideo(f.url) || f.video_id}/mqdefault.jpg`
const duracao = (f) => (f.duracao ? segundosParaTempo(f.duracao) : '')
const estado = (f) => ESTADOS_FONTE[f.estado] || { nome: f.estado, cor: '' }

// Trazer do Acervo (T40): as Referências de vídeo de lá entram com a transcrição, sem baixar.
const ligados = ref({})
const acervoAberto = ref(false)
const referencias = ref(null)
const buscaAcervo = ref('')
const trazendo = ref('')
const erroAcervo = ref('')

async function abrirAcervo() {
  acervoAberto.value = !acervoAberto.value
  if (acervoAberto.value && referencias.value === null) {
    try {
      referencias.value = (await api('/api/acervo/referencias')).filter((r) => r.tipo === 'video')
    } catch (e) {
      erroAcervo.value = e.message
      referencias.value = []
    }
  }
}
const referenciasFiltradas = computed(() => {
  const b = buscaAcervo.value.trim().toLowerCase()
  return (referencias.value || []).filter((r) => !b || r.titulo.toLowerCase().includes(b))
})
// Já está na Entrada? A fonte do Acervo tem video_id = id da Referência; a do YouTube, o id do vídeo.
function naEntrada(r) {
  const vid = idDoVideo(r.url)
  return fontes.value.some((f) => f.video_id === r.id || (vid && f.video_id === vid))
}
async function trazer(r) {
  trazendo.value = r.id
  erroAcervo.value = ''
  try {
    const f = await api('/api/entrada/acervo', { metodo: 'POST', corpo: { referencia: r.id } })
    location.hash = link('entrada', String(f.id))
  } catch (e) {
    erroAcervo.value = e.message
  } finally {
    trazendo.value = ''
  }
}

onMounted(async () => {
  buscar()
  try {
    ligados.value = await api('/api/integracoes')
  } catch {
    // Sem a lista de integrações, o botão do Acervo só não aparece.
  }
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  avisoCompartilhar.value = ''
})
</script>

<template>
  <Pagina>
    <template #topo>
      <div class="topo-pagina">
        <h1>Entrada</h1>
        <form class="meio entrada-form" @submit.prevent="mandar">
          <label class="so-leitor" for="link-yt">Link do YouTube (vídeo ou playlist)</label>
          <input id="link-yt" v-model="campo" class="campo" placeholder="Link do YouTube: vídeo ou playlist" inputmode="url" />
          <button type="button" class="botao" @click="colar">Colar</button>
          <button class="botao primario" :disabled="mandando">Mandar</button>
        </form>
      </div>
      <AbasPagina class="abas-entrada" :abas="abas" model-value="videos" rotulo="Partes da Entrada" @update:model-value="mudarAba" />
    </template>

    <p class="explicacao">
      Vídeos e playlists do YouTube que ainda não viraram fragmentos de uma trilha. Só a legenda
      (ou o Whisper, quando não tem) é lida, sem baixar o vídeo. Depois a IA divide por assunto e
      sugere as trilhas sozinha; você confirma e manda. Vídeo de mais de 3 h espera você pedir.
    </p>
    <p v-if="avisoCompartilhar" class="aviso neutro">{{ avisoCompartilhar }}</p>

    <p v-if="erro" class="aviso">{{ erro }}</p>

    <p v-if="!fontes.length" class="fraco">Nenhuma fonte ainda.</p>

    <div class="fontes">
    <div v-for="f in fontes" :key="f.id" class="cartao fonte">
      <a class="fonte-info" :href="link('entrada', String(f.id))">
        <div v-if="f.tipo === 'telegram'" class="capa capa-vazia"><Icone nome="telegram" :tamanho="20" /></div>
        <img v-else-if="!semCapa[f.id]" class="capa" :src="capa(f)" alt="" loading="lazy" @error="semCapa[f.id] = true" />
        <div v-else class="capa capa-vazia"><Icone nome="video" :tamanho="20" /></div>
        <div class="fonte-texto">
          <strong>{{ f.titulo || 'Lendo o vídeo…' }}</strong>
          <span class="pequeno">{{ [f.canal, duracao(f)].filter(Boolean).join(' · ') }}</span>
          <span v-if="f.playlist" class="pequeno">da playlist {{ f.playlist }}</span>
          <span v-if="f.resumo_fragmentos" class="pequeno">{{ f.resumo_fragmentos }}</span>
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

    <template v-if="ligados.acervo" #lateral>
      <div class="quadro trazer-acervo">
      <div class="quadro-topo"><h2 class="rotulo-secao">Do Acervo</h2></div>
      <p class="pequeno fraco">Vídeos que já estão transcritos no Acervo entram prontos, sem ler de novo.</p>
      <button class="botao discreto" type="button" @click="abrirAcervo">
        <Icone nome="livro" :tamanho="15" /> Trazer do Acervo
      </button>
      <template v-if="acervoAberto">
        <input v-model="buscaAcervo" class="campo" placeholder="Buscar vídeo no Acervo…" aria-label="Buscar vídeo no Acervo" />
        <p v-if="erroAcervo" class="aviso">{{ erroAcervo }}</p>
        <p v-if="referencias === null" class="fraco">Lendo o Acervo…</p>
        <p v-else-if="!referenciasFiltradas.length" class="fraco">Nenhum vídeo{{ buscaAcervo ? ' com esse nome' : '' }} no Acervo.</p>
        <ul v-else class="lista-acervo">
          <li v-for="r in referenciasFiltradas" :key="r.id">
            <button class="item" type="button" :disabled="!!trazendo" @click="trazer(r)">
              <span v-if="naEntrada(r)" class="etiqueta">na Entrada</span>
              {{ trazendo === r.id ? 'Trazendo… ' : '' }}{{ r.titulo }}
            </button>
          </li>
        </ul>
      </template>
    </div>
    </template>
  </Pagina>
</template>

<style scoped>
.abas-entrada { margin-bottom: var(--e5); }
.entrada-form { display: flex; gap: var(--e2); }
.entrada-form .campo {
  flex: 1;
  min-width: 0;
}
.fontes { display: grid; gap: var(--e3); grid-template-columns: repeat(auto-fill, minmax(420px, 1fr)); align-items: start; }
@media (max-width: 699px) { .fontes { grid-template-columns: 1fr; } }
.trazer-acervo {
  display: grid;
  gap: var(--e3);
  justify-items: start;
}
.trazer-acervo .campo {
  width: 100%;
}
.lista-acervo {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 2px;
  max-height: 300px;
  overflow-y: auto;
  width: 100%;
}
.item {
  display: block;
  width: 100%;
  text-align: left;
  padding: var(--e2) var(--e3);
  border-radius: var(--raio-p);
  font-size: 14px;
}
.item:hover {
  background: var(--ativo);
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
