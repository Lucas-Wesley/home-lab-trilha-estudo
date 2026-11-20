<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../api.js'
import { diasSemDestino, linhasDaTranscricao, linkNoMinuto } from '../entrada.js'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'
import { ESTADOS_ENCAIXE, ESTADOS_FONTE, TIPOS_FRAGMENTO } from '../rotulos.js'
import { idDoVideo } from '../trechoYoutube.js'
import { segundosParaTempo } from '../videoaula.js'

const props = defineProps({ fonteId: { type: Number, required: true } })

const fonte = ref(null)
const erro = ref('')
const tentando = ref(false)
let timer = null

// A fonte do Acervo tem video_id = id da Referência — o id do vídeo do YouTube sai do link.
// A do Telegram não tem vídeo do YouTube: os links "no minuto" ficam sem alvo (T44).
const videoId = computed(() => (fonte.value?.tipo === 'telegram' ? '' : fonte.value ? idDoVideo(fonte.value.url) || fonte.value.video_id : ''))

// T37: a divisão em fragmentos.
const pedido = ref('')
const ocupado = ref(false)
const editando = ref(0)
const edicao = ref({})
const mostrarDescartados = ref(false)
let pedidoIniciado = false

// T38: o destino dos fragmentos confirmados.
const trilhas = ref([])
const pedidoDestino = ref('')
let pedidoDestinoIniciado = false

const destinoOcupando = computed(() => ['na_fila', 'rodando'].includes(fonte.value?.tarefa_destino?.estado))

function ligarTimer() {
  if (timer) clearInterval(timer)
  timer =
    fonte.value && (['na_fila', 'lendo', 'baixando', 'transcrevendo', 'fragmentando'].includes(fonte.value.estado) || destinoOcupando.value)
      ? setInterval(buscar, 3000)
      : null
}

async function buscar() {
  try {
    fonte.value = await api(`/api/entrada/${props.fonteId}`)
    if (!pedidoIniciado && fonte.value.pedido) {
      pedido.value = fonte.value.pedido
      pedidoIniciado = true
    }
    if (!pedidoDestinoIniciado && fonte.value.pedido_destino) {
      pedidoDestino.value = fonte.value.pedido_destino
      pedidoDestinoIniciado = true
    }
    if (fonte.value.estado === 'fragmentada' && !trilhas.value.length) {
      trilhas.value = await api('/api/trilhas')
    }
    ligarTimer()
  } catch (e) {
    erro.value = e.message
  }
}

async function tentar() {
  tentando.value = true
  try {
    await api(`/api/entrada/${props.fonteId}/tentar`, { metodo: 'POST' })
    await buscar()
  } catch (e) {
    erro.value = e.message
  } finally {
    tentando.value = false
  }
}

const estado = computed(() => ESTADOS_FONTE[fonte.value?.estado] || { nome: fonte.value?.estado, cor: '' })

// Só os blocos [mm:ss] entram na transcrição mostrada (o cabeçalho do arquivo repete o que a tela já
// tem); sem nenhum bloco, mostram-se as linhas como vieram.
const blocos = computed(() => {
  const linhas = linhasDaTranscricao(fonte.value?.transcricao || '')
  const marcadas = linhas.filter((l) => l.marca)
  return marcadas.length ? marcadas : linhas.filter((l) => l.texto.trim())
})

// ---------- fragmentos ----------

const fragmentos = computed(() => fonte.value?.fragmentos || [])
const normais = computed(() => fragmentos.value.filter((f) => f.estado !== 'descartado'))
const visiveis = computed(() => (mostrarDescartados.value ? fragmentos.value : normais.value))
const sugeridos = computed(() => fragmentos.value.filter((f) => f.estado === 'sugerido'))
const haDescartados = computed(() => fragmentos.value.some((f) => f.estado === 'descartado'))

const tipo = (fr) => TIPOS_FRAGMENTO[fr.tipo] || { nome: fr.tipo, cor: '' }
const temProximo = (fr) => {
  const i = normais.value.findIndex((x) => x.id === fr.id)
  return i >= 0 && i < normais.value.length - 1
}

// agir roda a ação e atualiza a fonte; ocupado evita clique duplo.
async function agir(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    await fn()
    await buscar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

const fragmentar = () =>
  agir(() => api(`/api/entrada/${props.fonteId}/fragmentar`, { metodo: 'POST', corpo: { pedido: pedido.value } }))
const confirmar = () => agir(() => api(`/api/entrada/${props.fonteId}/confirmar`, { metodo: 'POST' }))
const juntar = (fr) => agir(() => api(`/api/fragmentos/${fr.id}/juntar`, { metodo: 'POST' }))
const descartar = (fr) => {
  if (!confirm(`Descartar "${fr.titulo}"?`)) return
  return agir(() => api(`/api/fragmentos/${fr.id}/descartar`, { metodo: 'POST' }))
}
const dividir = (fr) => {
  const em = prompt(
    `Dividir "${fr.titulo}" em que tempo? (mm:ss)`,
    segundosParaTempo(Math.floor((fr.inicio + fr.fim) / 2)),
  )
  if (!em) return
  return agir(() => api(`/api/fragmentos/${fr.id}/dividir`, { metodo: 'POST', corpo: { em } }))
}

const editar = (fr) => {
  editando.value = fr.id
  edicao.value = {
    inicio: segundosParaTempo(fr.inicio),
    fim: segundosParaTempo(fr.fim),
    titulo: fr.titulo,
    resumo: fr.resumo,
    assuntos: (fr.assuntos || []).join(', '),
    tipo: fr.tipo,
  }
}
const salvarEdicao = (fr) =>
  agir(async () => {
    await api(`/api/fragmentos/${fr.id}`, {
      metodo: 'PUT',
      corpo: {
        ...edicao.value,
        assuntos: edicao.value.assuntos
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean),
      },
    })
    editando.value = 0
  })

// ---------- destino ----------

const comDestino = computed(() => normais.value.filter((f) => ['confirmado', 'sem_destino', 'enviado'].includes(f.estado)))
const confirmados = computed(() => fragmentos.value.filter((f) => f.estado === 'confirmado'))

const sugerirDestinos = () =>
  agir(() => api(`/api/entrada/${props.fonteId}/destino`, { metodo: 'POST', corpo: { pedido: pedidoDestino.value } }))

const marcadas = (fr) => new Set((fr.destinos || []).map((d) => d.trilha_id))
const trilhasDisponiveis = (fr) => trilhas.value.filter((t) => t.estado !== 'concluida' && !marcadas(fr).has(t.id))
const motivoDe = (fr, d) => (fr.sugestao?.trilhas || []).find((s) => s.trilha_id === d.trilha_id)?.motivo || ''
const naoMarcadas = (fr) => (fr.sugestao?.trilhas || []).filter((s) => !marcadas(fr).has(s.trilha_id))

const marcar = (fr, trilhaId, marcar) => {
  const atuais = (fr.destinos || []).map((d) => d.trilha_id)
  const novas = marcar ? [...atuais, trilhaId] : atuais.filter((x) => x !== trilhaId)
  return agir(() => api(`/api/fragmentos/${fr.id}/destinos`, { metodo: 'PUT', corpo: { trilhas: novas } }))
}

const adicionarTrilha = (fr, ev) => {
  const id = Number(ev.target.value)
  ev.target.value = ''
  if (id) marcar(fr, id, true)
}

const trilhaNova = (fr) => {
  const nome = prompt('Nome da trilha nova:', fr.sugestao?.trilha_nova || '')
  if (!nome?.trim()) return
  return agir(async () => {
    const t = await api('/api/trilhas', { metodo: 'POST', corpo: { titulo: nome.trim() } })
    trilhas.value = [...trilhas.value, t]
    await marcarSemAgir(fr, t.id)
  })
}

// marcarSemAgir: o PUT dentro de um agir já aberto (trilha nova cria e marca numa tacada só).
const marcarSemAgir = (fr, trilhaId) => {
  const atuais = (fr.destinos || []).map((d) => d.trilha_id)
  return api(`/api/fragmentos/${fr.id}/destinos`, { metodo: 'PUT', corpo: { trilhas: [...atuais, trilhaId] } })
}

const semDestino = (fr) => agir(() => api(`/api/fragmentos/${fr.id}/destinos`, { metodo: 'PUT', corpo: { sem_destino: true } }))
const voltarDestino = (fr) => agir(() => api(`/api/fragmentos/${fr.id}/destinos`, { metodo: 'PUT', corpo: { trilhas: [] } }))

const encaixesPendentes = computed(() => {
  const ids = new Set()
  for (const fr of confirmados.value) {
    for (const d of fr.destinos || []) {
      if (!d.proposta_id) ids.add(d.trilha_id)
    }
  }
  return ids.size
})
const mandar = () => agir(() => api(`/api/entrada/${props.fonteId}/enviar`, { metodo: 'POST' }))
const linkEncaixe = (d) => `${link('trilhas', String(d.trilha_id))}#encaixar`
const estadoEncaixe = (d) => ESTADOS_ENCAIXE[d.estado] || d.estado || 'escolhida'

onMounted(buscar)
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <Pagina>
  <div class="pilha">
    <nav class="migalhas">
      <a :href="link('entrada')">Entrada</a>
      <Icone nome="seta-direita" :tamanho="12" />
      <span>{{ fonte?.titulo || 'Fonte' }}</span>
    </nav>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <p v-if="!fonte && !erro" class="fraco">Carregando…</p>

    <template v-if="fonte">
      <header class="topo-pagina topo-fonte">
        <div>
          <h1>{{ fonte.titulo || 'Lendo o vídeo…' }}</h1>
          <p class="pequeno contexto">
            <template v-if="fonte.canal">{{ fonte.canal }} · </template>
            <template v-if="fonte.duracao">{{ segundosParaTempo(fonte.duracao) }} · </template>
            <a v-if="fonte.tipo !== 'telegram'" :href="fonte.url" target="_blank" rel="noopener noreferrer">Abrir no YouTube</a>
            <span v-else class="fraco">aula do Telegram</span>
          </p>
        </div>
        <div class="acoes"><span class="etiqueta" :class="estado.cor">{{ estado.nome }}</span></div>
      </header>

      <template v-if="fonte.estado === 'erro'">
        <p class="aviso">{{ fonte.erro }}</p>
        <div><button class="botao" :disabled="tentando" @click="tentar">Tentar de novo</button></div>
      </template>
      <p v-else-if="fonte.estado !== 'pronta' && fonte.estado !== 'fragmentando' && fonte.estado !== 'fragmentada'" class="aviso neutro">
        A fonte está {{ estado.nome }} — a transcrição aparece aqui quando terminar.
      </p>

      <section v-if="fonte.capitulos?.length">
        <h2>Capítulos</h2>
        <ul class="capitulos">
          <li v-for="c in fonte.capitulos" :key="c.inicio">
            <a :href="linkNoMinuto(videoId, c.inicio)" target="_blank" rel="noopener noreferrer">{{
              segundosParaTempo(c.inicio)
            }}</a>
            {{ c.titulo }}
          </li>
        </ul>
      </section>

      <details v-if="fonte.transcricao" class="transcricao">
        <summary><h2>Transcrição</h2></summary>
        <p v-for="(b, i) in blocos" :key="i" class="bloco">
          <a
            v-if="b.marca && videoId"
            class="marca"
            :href="linkNoMinuto(videoId, b.segundos)"
            target="_blank"
            rel="noopener noreferrer"
            >[{{ b.marca }}]</a
          >
          <span v-else-if="b.marca" class="marca">[{{ b.marca }}]</span>
          {{ b.texto }}
        </p>
      </details>

      <section v-if="fonte.estado !== 'na_fila' && fonte.estado !== 'lendo' && fonte.estado !== 'erro'" class="pilha">
        <div class="cabecalho">
          <h2>Fragmentos</h2>
          <span v-if="sugeridos.length" class="etiqueta ambar">{{ sugeridos.length }} para revisar</span>
        </div>

        <p v-if="fonte.estado === 'fragmentando'" class="aviso neutro">
          {{
            fonte.tarefa?.estado === 'na_fila'
              ? 'A divisão em fragmentos está na fila.'
              : 'A IA está dividindo a fonte em fragmentos…'
          }}
        </p>
        <template v-else>
          <p v-if="fonte.tarefa?.estado === 'falhou'" class="aviso">A divisão falhou: {{ fonte.erro }}</p>
          <p v-else-if="fonte.notas" class="aviso neutro">{{ fonte.notas }}</p>
          <form class="linha" @submit.prevent="fragmentar">
            <input v-model="pedido" class="campo fragmentar-pedido" placeholder="Pedido para a IA (opcional)" />
            <button class="botao" :class="{ primario: !fragmentos.length }" :disabled="ocupado">
              {{ fragmentos.length ? 'Dividir de novo' : 'Dividir em fragmentos' }}
            </button>
          </form>
        </template>

        <div v-for="fr in visiveis" :key="fr.id" class="cartao fragmento" :class="{ descartado: fr.estado === 'descartado' }">
          <template v-if="editando !== fr.id">
            <p class="fragmento-topo pequeno">
              <a v-if="videoId" :href="linkNoMinuto(videoId, fr.inicio)" target="_blank" rel="noopener noreferrer"
                >{{ segundosParaTempo(fr.inicio) }}–{{ segundosParaTempo(fr.fim) }}</a
              >
              <template v-else>{{ segundosParaTempo(fr.inicio) }}–{{ segundosParaTempo(fr.fim) }}</template>
              · {{ segundosParaTempo(fr.fim - fr.inicio) }}
              <span class="etiqueta" :class="tipo(fr).cor">{{ tipo(fr).nome }}</span>
              <span v-if="fr.estado === 'confirmado'" class="etiqueta verde">confirmado</span>
              <span v-else-if="fr.estado === 'descartado'" class="etiqueta">descartado</span>
            </p>
            <strong>{{ fr.titulo }}</strong>
            <p v-if="fr.resumo" class="resumo pequeno">{{ fr.resumo }}</p>
            <p v-if="fr.assuntos?.length" class="linha">
              <span v-for="a in fr.assuntos" :key="a" class="etiqueta">{{ a }}</span>
            </p>
            <p v-if="fr.estado !== 'descartado'" class="linha">
              <button class="botao discreto" :disabled="ocupado" @click="editar(fr)">Editar</button>
              <button class="botao discreto" :disabled="ocupado" @click="dividir(fr)">Dividir</button>
              <button v-if="temProximo(fr)" class="botao discreto" :disabled="ocupado" @click="juntar(fr)">Juntar</button>
              <button class="botao discreto" :disabled="ocupado" @click="descartar(fr)">Descartar</button>
            </p>

            <!-- T38: o destino do fragmento (só depois de confirmado). -->
            <div v-if="fr.estado === 'enviado'" class="destinos pequeno">
              <span class="fraco">Foi para:</span>
              <a v-for="d in fr.destinos" :key="d.trilha_id" :href="linkEncaixe(d)" class="etiqueta link-destino"
                >{{ d.titulo }} · {{ estadoEncaixe(d) }}</a
              >
            </div>
            <div v-else-if="fr.estado === 'confirmado' || fr.estado === 'sem_destino'" class="destinos">
              <button
                v-for="d in fr.destinos"
                :key="d.trilha_id"
                class="etiqueta destino-marcado"
                :title="motivoDe(fr, d) || 'Tirar desta trilha'"
                :disabled="ocupado"
                @click="marcar(fr, d.trilha_id, false)"
              >
                {{ d.titulo }} ✕
              </button>
              <button
                v-for="s in naoMarcadas(fr)"
                :key="s.trilha_id"
                class="etiqueta destino-sugerido"
                :title="s.motivo || 'Sugerida pela IA'"
                :disabled="ocupado"
                @click="marcar(fr, s.trilha_id, true)"
              >
                + {{ s.titulo }}
              </button>
              <select v-if="trilhasDisponiveis(fr).length" class="campo seletor-trilha" :disabled="ocupado" @change="adicionarTrilha(fr, $event)">
                <option value="">+ trilha</option>
                <option v-for="t in trilhasDisponiveis(fr)" :key="t.id" :value="t.id">{{ t.titulo }}</option>
              </select>
              <button class="botao discreto mini" :disabled="ocupado" @click="trilhaNova(fr)">Trilha nova</button>
              <button v-if="fr.estado !== 'sem_destino'" class="botao discreto mini" :disabled="ocupado" @click="semDestino(fr)">Sem destino</button>
              <template v-else>
                <span class="etiqueta">sem destino · some em {{ diasSemDestino(fr.sem_destino_em) }} dias</span>
                <button class="botao discreto mini" :disabled="ocupado" @click="voltarDestino(fr)">desfazer</button>
              </template>
              <p v-if="fr.sugestao?.motivo_nenhuma" class="pequeno fraco destino-nota">A IA não viu destino: {{ fr.sugestao.motivo_nenhuma }}</p>
              <p v-else-if="fr.sugestao?.trilha_nova" class="pequeno fraco destino-nota">A IA sugeriu a trilha nova "{{ fr.sugestao.trilha_nova }}".</p>
            </div>
          </template>
          <form v-else class="pilha" @submit.prevent="salvarEdicao(fr)">
            <div class="linha">
              <label class="pequeno"
                >de <input v-model="edicao.inicio" class="campo tempo" placeholder="mm:ss" required
              /></label>
              <label class="pequeno"
                >até <input v-model="edicao.fim" class="campo tempo" placeholder="mm:ss" required
              /></label>
            </div>
            <input v-model="edicao.titulo" class="campo" placeholder="Título" required />
            <textarea v-model="edicao.resumo" class="campo" rows="3" placeholder="Resumo"></textarea>
            <input v-model="edicao.assuntos" class="campo" placeholder="Assuntos, separados por vírgula" />
            <select v-model="edicao.tipo" class="campo">
              <option value="conteudo">Conteúdo</option>
              <option value="opiniao">Opinião</option>
              <option value="fora">Fora do tema</option>
            </select>
            <div class="linha">
              <button class="botao primario" type="submit" :disabled="ocupado">Salvar</button>
              <button class="botao" type="button" @click="editando = 0">Cancelar</button>
            </div>
          </form>
        </div>

        <div v-if="sugeridos.length || haDescartados" class="linha">
          <button v-if="sugeridos.length" class="botao primario" :disabled="ocupado" @click="confirmar">
            Confirmar {{ sugeridos.length }} fragmento{{ sugeridos.length > 1 ? 's' : '' }}
          </button>
          <button v-if="haDescartados" class="botao discreto" @click="mostrarDescartados = !mostrarDescartados">
            {{ mostrarDescartados ? 'Esconder descartados' : 'Mostrar descartados' }}
          </button>
        </div>

        <!-- T38: sugerir trilhas e mandar — depois de confirmar os fragmentos. -->
        <section v-if="comDestino.length" class="pilha destino-secao">
          <div class="cabecalho">
            <h2>Destino</h2>
          </div>
          <p v-if="destinoOcupando" class="aviso neutro">
            {{ fonte.tarefa_destino?.estado === 'na_fila' ? 'A sugestão de trilhas está na fila.' : 'A IA está sugerindo as trilhas…' }}
          </p>
          <template v-else>
            <p v-if="fonte.tarefa_destino?.estado === 'falhou'" class="aviso">A sugestão falhou: {{ fonte.tarefa_destino.erro }}</p>
            <form class="linha" @submit.prevent="sugerirDestinos">
              <input v-model="pedidoDestino" class="campo fragmentar-pedido" placeholder="Pedido para a IA (opcional)" />
              <button class="botao" :class="{ primario: !fonte.tarefa_destino }" :disabled="ocupado || !confirmados.length">
                {{ fonte.tarefa_destino ? 'Sugerir de novo' : 'Sugerir trilhas' }}
              </button>
            </form>
          </template>
          <div v-if="encaixesPendentes" class="linha">
            <button class="botao primario" :disabled="ocupado" @click="mandar">
              Mandar para as trilhas ({{ encaixesPendentes }} encaixe{{ encaixesPendentes > 1 ? 's' : '' }})
            </button>
          </div>
        </section>
      </section>
    </template>
  </div>
  </Pagina>
</template>

<style scoped>
.topo-fonte { margin-bottom: 0; }
.topo-fonte h1 { flex: 1 1 auto; overflow-wrap: anywhere; }
.contexto { margin: var(--e1) 0 0; }
.capitulos {
  list-style: none;
  margin: var(--e3) 0 0;
  padding: 0;
  display: grid;
  gap: var(--e2);
}
.capitulos a {
  font-family: var(--fonte-mono);
  font-size: 13px;
}
.transcricao {
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  padding: var(--e3) var(--e4);
}
.transcricao summary {
  cursor: pointer;
}
.transcricao summary h2 {
  display: inline;
}
.transcricao summary::marker {
  color: var(--texto-fraco);
}
.bloco {
  margin: var(--e3) 0;
  font-size: 14.5px;
  line-height: 1.6;
  color: var(--texto-suave);
}
.bloco .marca {
  font-family: var(--fonte-mono);
  font-size: 12.5px;
}
.fragmentar-pedido {
  flex: 1;
  min-width: 160px;
}
.fragmento-topo a {
  font-family: var(--fonte-mono);
  font-size: 12.5px;
}
.fragmento .resumo {
  color: var(--texto-suave);
  margin: var(--e1) 0 var(--e2);
}
.fragmento .linha {
  margin-top: var(--e2);
}
.fragmento.descartado {
  opacity: 0.55;
}
.tempo {
  width: 88px;
  font-family: var(--fonte-mono);
  margin-left: var(--e1);
}
.destinos {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e2);
  margin-top: var(--e2);
}
.destino-marcado,
.destino-sugerido {
  cursor: pointer;
  border: 1px solid var(--borda);
  font: inherit;
  font-size: 12.5px;
}
.destino-marcado {
  background: var(--superficie-2);
}
.destino-sugerido {
  opacity: 0.65;
}
.link-destino {
  color: inherit;
}
.seletor-trilha {
  width: auto;
  font-size: 12.5px;
  padding: 4px var(--e2);
}
.botao.mini {
  font-size: 12.5px;
  padding: 4px var(--e2);
}
.destino-nota {
  flex-basis: 100%;
  margin: 0;
}
.destino-secao {
  border-top: 1px solid var(--borda);
  padding-top: var(--e3);
}
</style>
