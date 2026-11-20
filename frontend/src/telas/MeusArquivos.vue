<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AbasPagina from '../AbasPagina.vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link, linkPasta } from '../rotas.js'
import { acharPasta, aulasPedidas, montarArvore, oQueFalta, tamanho } from '../pastas.js'
import { segundosParaTempo } from '../videoaula.js'
import { MODOS, modoRecomendado, pacoteDoCurso, resumoDoPacote } from '../cursoParaTrilha.js'
import { aulasDaPasta, referenciaDaAula, referenciaDoCurso, selecaoCursos as marcadas, tituloDoCurso } from '../cursos.js'

// Meus arquivos (T41): os cursos do Telegram que estão no Streaming, em curso › módulo › aula
// (a árvore é a do Streaming, S55 de lá — pastas.js copiado). A Trilha só chama a API dele:
// baixar (uma aula, um módulo ou o curso) e apagar (lixeira); nada baixa sozinho.
const props = defineProps({
  pasta: { type: String, default: '' }, // o caminho da rota (#/entrada/arquivos/<caminho>)
})

const ABAS = [
  { id: 'videos', nome: 'Vídeos' },
  { id: 'arquivos', nome: 'Meus arquivos' },
]
const mudarAba = (id) => {
  if (id === 'videos') location.hash = '#/entrada'
}

const dados = ref(null)
const erro = ref('')
const aviso = ref('')
const ligado = ref(true)
const baixandoAulas = ref(false)

const arvore = computed(() => montarArvore(dados.value?.videos || [], dados.value?.salvos || []))
const pastaAtual = computed(() => (props.pasta ? acharPasta(arvore.value, props.pasta) : arvore.value))
const caminhoAtual = computed(() => {
  const nomes = props.pasta.split('/').filter(Boolean)
  return nomes.map((nome, i) => ({ nome, caminho: nomes.slice(0, i + 1).join('/') }))
})
const pedidas = computed(() => aulasPedidas(dados.value?.pedidos || []))
const estaPedida = (a) => pedidas.value.has(a.chave) || pedidas.value.has(`${a.link}#*`)

// O que falta baixar nesta pasta (e nas subpastas), sem o que já está na fila: um pedido por
// grupo do Telegram.
const faltaAqui = computed(() =>
  pastaAtual.value
    ? oQueFalta(pastaAtual.value)
        .map((p) => {
          const mensagens = p.mensagens.filter((m) => !pedidas.value.has(`${p.link}#${m}`) && !pedidas.value.has(`${p.link}#*`))
          return { ...p, mensagens }
        })
        .filter((p) => p.mensagens.length)
    : [],
)
const aulasQueFaltam = computed(() => faltaAqui.value.reduce((soma, p) => soma + p.mensagens.length, 0))
const pesoDaFalta = computed(() => faltaAqui.value.reduce((soma, p) => soma + p.tamanho, 0))
const naFilaAqui = computed(() => (pastaAtual.value ? pastaAtual.value.total - pastaAtual.value.baixadas - aulasQueFaltam.value : 0))
const livre = computed(() => dados.value?.livre || 0)
const naoCabe = computed(() => livre.value > 0 && pesoDaFalta.value > livre.value)
const andando = computed(() => (dados.value?.pedidos || []).some((p) => p.estado === 'na fila' || p.estado === 'baixando'))
const telegramInstalado = computed(() => dados.value?.telegram?.instalado !== false)

const quantasAulas = (n) => `${n} ${n === 1 ? 'aula' : 'aulas'}`

function resumoDaPasta(p) {
  const partes = [quantasAulas(p.total)]
  if (p.baixadas && p.baixadas < p.total) partes.push(`${p.baixadas} ${p.baixadas === 1 ? 'baixada' : 'baixadas'}`)
  if (p.baixadas === 0 && p.total) partes.push('nada baixado')
  if (p.tamanho) partes.push(`${tamanho(p.tamanho)} no servidor`)
  if (p.falta) partes.push(`faltam ${tamanho(p.falta)}`)
  return partes.join(' · ')
}

async function carregar() {
  try {
    dados.value = await api('/api/arquivos')
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 502 && /configurado/.test(e.message) ? 'O Streaming não está configurado neste servidor.' : e.message
  }
}

// ---------- processar (T44): a aula do Telegram vira fonte da Entrada ----------
// fontesDaEntrada: ref telegram: → fonte — as já processadas mostram "na Entrada".
const fontesDaEntrada = ref({})
async function buscarEntrada() {
  try {
    const lista = await api('/api/entrada')
    const mapa = {}
    for (const f of lista || []) if (f.tipo === 'telegram') mapa[f.video_id] = f
    fontesDaEntrada.value = mapa
  } catch { /* sem a lista, as aulas mostram "Processar" — o POST deduplica */ }
}
const fonteDaAula = (a) => fontesDaEntrada.value[referenciaDaAula(a)]

const dialogoProcessar = ref(null) // {aulas, titulo}
const fragmentarDepois = ref(true)
const mandandoProcessar = ref(false)
const erroProcessar = ref('')
const tempoDeAudio = computed(() => (dialogoProcessar.value?.aulas || []).reduce((s, a) => s + (a.duracao || 0), 0))

const abrirProcessarPasta = (p) => {
  dialogoProcessar.value = { aulas: aulasDaPasta(p).filter(referenciaDaAula), titulo: `a pasta "${p.nome}"` }
  fragmentarDepois.value = true
  erroProcessar.value = ''
}
const abrirProcessarAula = (a) => {
  dialogoProcessar.value = { aulas: [a], titulo: `a aula "${a.nome}"` }
  fragmentarDepois.value = true
  erroProcessar.value = ''
}

// Processar: a Trilha baixa (se preciso) e transcreve no Whisper — sem crédito do Claude.
// As fontes aparecem na aba Vídeos da Entrada.
async function processar() {
  mandandoProcessar.value = true
  erroProcessar.value = ''
  try {
    const refs = dialogoProcessar.value.aulas.map(referenciaDaAula).filter(Boolean)
    const fontes = await api('/api/entrada/telegram', { metodo: 'POST', corpo: { refs, fragmentar: fragmentarDepois.value } })
    dialogoProcessar.value = null
    aviso.value = `${fontes.length === 1 ? '1 aula' : `${fontes.length} aulas`} na Entrada: a transcrição é pelo Whisper e cada fonte ${fragmentarDepois.value ? 'já vai fragmentar' : 'fica pronta'} ao terminar.`
    await buscarEntrada()
  } catch (e) {
    erroProcessar.value = e.message
  } finally {
    mandandoProcessar.value = false
  }
}

// Baixar aulas do catálogo: um pedido por grupo, só com as mensagens que faltam. Cada aula vai
// para a pasta do curso e do módulo dela (o Streaming lê o caminho no texto da mensagem).
async function baixarAulas(lista) {
  baixandoAulas.value = true
  aviso.value = ''
  try {
    for (const p of lista) {
      await api('/api/arquivos/baixar', { metodo: 'POST', corpo: { link: p.link, grupo: p.grupo, mensagens: p.mensagens } })
    }
    aviso.value = `${quantasAulas(lista.reduce((s, p) => s + p.mensagens.length, 0))} na fila: o servidor baixa uma por vez e cada uma aparece aqui ao terminar.`
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    baixandoAulas.value = false
  }
}

const baixarAula = (a) => baixarAulas([{ link: a.link, grupo: a.grupo, mensagens: [a.mensagem] }])

// A lixeira da pasta: apaga do servidor as aulas baixadas dela e das subpastas. As do Telegram
// continuam no catálogo, para baixar de novo.
async function apagarPasta(p) {
  if (!confirm(`Apagar do servidor as ${p.baixadas} aulas baixadas de "${p.nome}" (${tamanho(p.tamanho)})? As do Telegram continuam aqui para baixar de novo.`)) return
  aviso.value = ''
  try {
    const r = await api('/api/arquivos/apagar-pasta', { metodo: 'POST', corpo: { pasta: p.caminho.join('/') } })
    aviso.value = `${r.apagados} apagadas: ${tamanho(r.liberado)} livres de novo.`
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

async function apagarAula(a) {
  if (!confirm(`Apagar "${a.nome}" do servidor? Não dá para desfazer.`)) return
  aviso.value = ''
  try {
    await api(`/api/arquivos/videos/${a.video.id}`, { metodo: 'DELETE' })
    aviso.value = `"${a.nome}" apagada: continua no Telegram para baixar de novo.`
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

// Assistir: o player é do Streaming — abre lá, numa aba nova.
const linkAssistir = (a) => `${dados.value?.link_streaming || ''}/#/assistir/proprio:${a.video.id}`

// ---------- fase 8 (T42): criar a trilha igual ao curso ----------
const criandoTrilha = ref(false)
const tituloTrilha = ref('')
const modoTrilha = ref('por_pasta')
const mandandoTrilha = ref(false)
const erroTrilha = ref('')
const recomendado = ref('por_pasta')
const resumoDosModos = computed(() =>
  pastaAtual.value
    ? Object.fromEntries(
        Object.keys(MODOS).map((m) => [m, resumoDoPacote(pacoteDoCurso(pastaAtual.value, m, tituloTrilha.value))]),
      )
    : {},
)

function abrirCriarTrilha() {
  tituloTrilha.value = pastaAtual.value.nome
  recomendado.value = modoRecomendado(pastaAtual.value)
  modoTrilha.value = recomendado.value
  erroTrilha.value = ''
  criandoTrilha.value = true
}

// A trilha nasce pela mesma importação de sempre — sem IA, sem lição e sem baixar nada: cada aula
// entra como material streaming "telegram:<link>#<mensagem>" e só baixa por clique na etapa.
async function criarTrilha() {
  mandandoTrilha.value = true
  erroTrilha.value = ''
  try {
    const pacote = pacoteDoCurso(pastaAtual.value, modoTrilha.value, tituloTrilha.value, dados.value?.link_streaming || '')
    const r = await api('/api/importar', { metodo: 'POST', corpo: pacote })
    criandoTrilha.value = false
    location.hash = `#/trilhas/${r.trilha_id}`
  } catch (e) {
    erroTrilha.value = e.message
  } finally {
    mandandoTrilha.value = false
  }
}

const irParaPasta = (caminho) => {
  location.hash = linkPasta(caminho)
}

// ---------- fase 9 (T43): encaixar os cursos marcados pelos títulos das aulas ----------
// Cada pasta ganha uma caixa de marcar; uma ou mais marcadas mostram as duas ações:
// encaixar numa trilha existente ou criar a trilha vazia e encaixar nela — mesmo encaixe.
const chavePasta = (p) => p.caminho.join('/')
const alternarMarcada = (p) => {
  const s = new Set(marcadas.value)
  s.has(chavePasta(p)) ? s.delete(chavePasta(p)) : s.add(chavePasta(p))
  marcadas.value = s
}
// As pastas marcadas, na ordem da árvore (o título do material segue essa ordem).
const pastasMarcadas = computed(() => {
  const lista = []
  const anda = (p) => {
    for (const f of p?.pastas || []) {
      if (marcadas.value.has(chavePasta(f))) lista.push(f)
      anda(f)
    }
  }
  anda(arvore.value)
  return lista
})
const materialDoCurso = computed(() => {
  if (!pastasMarcadas.value.length) return null
  const referencia = referenciaDoCurso(pastasMarcadas.value)
  if (!referencia) return null
  return { tipo: 'curso', titulo: tituloDoCurso(pastasMarcadas.value), referencia }
})

// "Encaixar numa trilha": seletor de trilha + pedido opcional.
const encaixando = ref(false)
const trilhas = ref([])
const trilhaEscolhida = ref('')
const pedidoCurso = ref('')
const mandandoEncaixe = ref(false)
const erroEncaixe = ref('')
// "Criar trilha a partir destes cursos": nome da trilha + pedido opcional.
const criandoDosCursos = ref(false)
const tituloTrilhaCursos = ref('')
const pedidoCursoNovo = ref('')
// Aviso depois de pedir: "Encaixe pedido na trilha X" com link para a aba Encaixar.
const avisoEncaixe = ref(null) // {id, titulo}

async function abrirEncaixar() {
  erroEncaixe.value = ''
  pedidoCurso.value = ''
  try {
    trilhas.value = await api('/api/trilhas')
  } catch (e) {
    erroEncaixe.value = e.message
  }
  trilhaEscolhida.value = ''
  encaixando.value = true
}

// O material "curso" vai pelo encaixe de sempre — a IA lê só os títulos das aulas.
async function pedirEncaixe(trilhaId, trilhaTitulo, pedido) {
  const material = materialDoCurso.value
  if (!material) {
    erroEncaixe.value = 'Uma das pastas marcadas não é de um grupo do Telegram.'
    return false
  }
  await api(`/api/trilhas/${trilhaId}/encaixar`, { metodo: 'POST', corpo: { material, pedido } })
  avisoEncaixe.value = { id: trilhaId, titulo: trilhaTitulo }
  marcadas.value = new Set()
  return true
}

async function encaixarCursos() {
  const t = trilhas.value.find((x) => String(x.id) === trilhaEscolhida.value)
  if (!t) return
  mandandoEncaixe.value = true
  erroEncaixe.value = ''
  try {
    if (await pedirEncaixe(t.id, t.titulo, pedidoCurso.value)) encaixando.value = false
  } catch (e) {
    erroEncaixe.value = e.message
  } finally {
    mandandoEncaixe.value = false
  }
}

// A trilha nasce vazia (rascunho) com o nome escolhido e o mesmo encaixe roda nela.
async function criarTrilhaDosCursos() {
  mandandoEncaixe.value = true
  erroEncaixe.value = ''
  try {
    const t = await api('/api/trilhas', { metodo: 'POST', corpo: { titulo: tituloTrilhaCursos.value.trim() } })
    if (await pedirEncaixe(t.id, t.titulo, pedidoCursoNovo.value)) criandoDosCursos.value = false
  } catch (e) {
    erroEncaixe.value = e.message
  } finally {
    mandandoEncaixe.value = false
  }
}

function abrirCriarDosCursos() {
  tituloTrilhaCursos.value = tituloDoCurso(pastasMarcadas.value)
  pedidoCursoNovo.value = ''
  erroEncaixe.value = ''
  criandoDosCursos.value = true
}

let relogioDaFila
watch(andando, (sim) => {
  clearInterval(relogioDaFila)
  if (sim) relogioDaFila = setInterval(carregar, 5000)
})

onMounted(async () => {
  try {
    ligado.value = (await api('/api/integracoes')).streaming !== false
    if (ligado.value === false) {
      erro.value = 'O Streaming não está configurado neste servidor.'
      return
    }
  } catch {
    /* se /api/integracoes falhar, tenta mesmo assim: o erro de /api/arquivos explica */
  }
  await carregar()
  await buscarEntrada()
})
onBeforeUnmount(() => clearInterval(relogioDaFila))
</script>

<template>
  <Pagina>
    <div class="pilha">
      <AbasPagina :abas="ABAS" model-value="arquivos" rotulo="Partes da Entrada" @update:model-value="mudarAba" />

      <header class="cabecalho">
        <div>
          <h1>Meus arquivos</h1>
          <p class="pequeno">
            Os cursos do Telegram que estão no Streaming, em curso › módulo › aula.
            <template v-if="dados && livre"> {{ tamanho(livre) }} livres no servidor.</template>
          </p>
        </div>
      </header>

      <p v-if="erro" class="aviso">{{ erro }}</p>
      <p v-else-if="aviso" class="aviso neutro">{{ aviso }}</p>
      <!-- S57 do Streaming: o disco dos vídeos não pôde ser lido; a lista vem do banco mesmo assim. -->
      <p v-if="dados?.aviso" class="aviso">{{ dados.aviso }}</p>
      <p v-if="dados && !telegramInstalado" class="aviso">O tdl não está instalado no Streaming: dá para ver as aulas, mas não baixar.</p>
      <p v-if="dados && !dados.videos?.length && !dados.salvos?.length" class="fraco">
        Nenhum curso ainda. Os grupos de aulas salvos no Streaming aparecem aqui.
      </p>

      <!-- T43: com pastas marcadas, encaixar numa trilha ou criar a trilha dos cursos. -->
      <div v-if="pastasMarcadas.length" class="acoes-cursos">
        <span class="pequeno meta">{{ pastasMarcadas.length }} {{ pastasMarcadas.length === 1 ? 'curso marcado' : 'cursos marcados' }}</span>
        <button class="botao primario" :disabled="!materialDoCurso" @click="abrirEncaixar">Encaixar numa trilha</button>
        <button class="botao" :disabled="!materialDoCurso" @click="abrirCriarDosCursos">Criar trilha a partir destes cursos</button>
      </div>
      <p v-if="pastasMarcadas.length && !materialDoCurso" class="aviso">
        Uma das pastas marcadas não é de um grupo do Telegram — só cursos do Telegram entram num encaixe.
      </p>
      <p v-if="avisoEncaixe" class="aviso neutro">
        Encaixe pedido na trilha
        <a class="link" :href="`${link('trilhas', String(avisoEncaixe.id))}#encaixar`">{{ avisoEncaixe.titulo }}</a>
        — a proposta aparece na aba Encaixar material.
      </p>

      <!-- Raiz: um cartão por grupo/curso. -->
      <ul v-if="!pasta && arvore.pastas.length" class="lista-arvore">
        <li v-for="g in arvore.pastas" :key="g.nome" class="linha-pasta">
          <input
            type="checkbox"
            class="marca"
            :checked="marcadas.has(chavePasta(g))"
            :aria-label="`Marcar ${g.nome}`"
            @change="alternarMarcada(g)"
          />
          <button class="item" type="button" @click="irParaPasta(g.caminho.join('/'))">
            <Icone nome="pasta" :tamanho="17" />
            <span class="sobre">
              <span class="nome">{{ g.nome }}</span>
              <span class="meta">{{ resumoDaPasta(g) }}</span>
            </span>
            <Icone nome="seta-direita" :tamanho="15" class="seta" />
          </button>
        </li>
      </ul>

      <!-- Numa pasta: caminho clicável, resumo e as ações. -->
      <template v-if="pasta">
        <nav class="caminho" aria-label="Onde estou">
          <button class="crumb" type="button" @click="irParaPasta('')"><Icone nome="voltar" :tamanho="14" /> Arquivos</button>
          <template v-for="(c, i) in caminhoAtual.slice(0, -1)" :key="c.caminho">
            <span class="sep" aria-hidden="true">›</span>
            <button class="crumb" type="button" :title="c.nome" @click="irParaPasta(c.caminho)">{{ c.nome }}</button>
          </template>
        </nav>

        <p v-if="dados && !pastaAtual" class="fraco">Esta pasta não existe mais.</p>
        <template v-else-if="pastaAtual">
          <div class="cabeca-pasta">
            <h2>{{ pastaAtual.nome }}</h2>
            <p class="pequeno meta">
              {{ resumoDaPasta(pastaAtual) }}<template v-if="naFilaAqui > 0"> · {{ naFilaAqui }} na fila</template>
            </p>
            <!-- T42: criar a trilha igual ao curso · T44: processar aulas na Entrada. -->
            <div class="acoes-curso">
              <button v-if="pastaAtual.total" class="botao" @click="abrirCriarTrilha">Criar trilha igual a este curso</button>
              <button
                v-if="aulasDaPasta(pastaAtual).some(referenciaDaAula)"
                class="botao"
                @click="abrirProcessarPasta(pastaAtual)"
              >
                Processar na Entrada
              </button>
            </div>
            <div v-if="aulasQueFaltam || pastaAtual.baixadas" class="linha">
              <button v-if="aulasQueFaltam" class="botao primario" :disabled="baixandoAulas || !telegramInstalado || naoCabe" @click="baixarAulas(faltaAqui)">
                {{
                  baixandoAulas
                    ? 'Mandando…'
                    : `Baixar ${aulasQueFaltam === pastaAtual.total ? 'tudo' : 'o que falta'} (${quantasAulas(aulasQueFaltam)} · ${tamanho(pesoDaFalta)})`
                }}
              </button>
              <button v-if="pastaAtual.baixadas" class="botao perigo" @click="apagarPasta(pastaAtual)">
                <Icone nome="lixeira" :tamanho="14" /> Apagar as baixadas
              </button>
            </div>
            <p v-if="naoCabe && aulasQueFaltam" class="aviso">
              Não cabe: faltam {{ tamanho(pesoDaFalta) }} e há {{ tamanho(livre) }} livres no servidor. Apague algo antes.
            </p>
          </div>

          <ul v-if="pastaAtual.pastas.length" class="lista-arvore">
            <li v-for="g in pastaAtual.pastas" :key="g.nome" class="linha-pasta">
              <input
                type="checkbox"
                class="marca"
                :checked="marcadas.has(chavePasta(g))"
                :aria-label="`Marcar ${g.nome}`"
                @change="alternarMarcada(g)"
              />
              <button class="item" type="button" @click="irParaPasta(g.caminho.join('/'))">
                <Icone nome="pasta" :tamanho="17" />
                <span class="sobre">
                  <span class="nome">{{ g.nome }}</span>
                  <span class="meta">{{ resumoDaPasta(g) }}</span>
                </span>
                <Icone nome="seta-direita" :tamanho="15" class="seta" />
              </button>
            </li>
          </ul>

          <ul v-if="pastaAtual.aulas.length" class="lista-arvore aulas">
            <li v-for="a in pastaAtual.aulas" :key="a.chave" class="aula" :class="{ falta: !a.video }">
              <template v-if="a.video">
                <span class="icone-aula"><Icone nome="video" :tamanho="16" /></span>
                <span class="sobre">
                  <span class="nome">{{ a.nome }}</span>
                  <span class="meta">{{ [segundosParaTempo(a.duracao), tamanho(a.video.tamanho)].filter(Boolean).join(' · ') }}</span>
                </span>
                <a v-if="fonteDaAula(a)" class="link mini" :href="link('entrada', String(fonteDaAula(a).id))">na Entrada</a>
                <button v-else-if="referenciaDaAula(a)" class="botao discreto mini" @click="abrirProcessarAula(a)">Processar</button>
                <a class="botao discreto mini" :href="linkAssistir(a)" target="_blank" rel="noopener">Assistir</a>
                <button class="botao discreto mini perigo" aria-label="Apagar do servidor" @click="apagarAula(a)">
                  <Icone nome="lixeira" :tamanho="14" />
                </button>
              </template>
              <template v-else>
                <span class="icone-aula vazia"><Icone nome="video" :tamanho="16" /></span>
                <span class="sobre">
                  <span class="nome">{{ a.nome }}</span>
                  <span class="meta">{{ [segundosParaTempo(a.duracao), tamanho(a.tamanho), 'no Telegram'].filter(Boolean).join(' · ') }}</span>
                </span>
                <a v-if="fonteDaAula(a)" class="link mini" :href="link('entrada', String(fonteDaAula(a).id))">na Entrada</a>
                <button v-else-if="referenciaDaAula(a)" class="botao discreto mini" @click="abrirProcessarAula(a)">Processar</button>
                <span v-if="estaPedida(a)" class="etiqueta">na fila</span>
                <button v-else class="botao mini" :disabled="baixandoAulas || !telegramInstalado" @click="baixarAula(a)">Baixar</button>
              </template>
            </li>
          </ul>
          <p v-if="!pastaAtual.pastas.length && !pastaAtual.aulas.length" class="fraco">Nenhuma aula nesta pasta.</p>
        </template>
      </template>

      <!-- Criar a trilha igual ao curso: título, um dos dois modos e a contagem na hora. -->
      <div v-if="criandoTrilha" class="dialogo-fundo" @click.self="criandoTrilha = false">
        <form class="painel pilha" role="dialog" aria-label="Criar trilha igual a este curso" @submit.prevent="criarTrilha">
          <h3>Criar trilha igual a este curso</h3>
          <p class="pequeno">
            A trilha fica com a cara do curso: cada aula do Telegram vira material da etapa e só
            baixa quando você pedir na hora de assistir. Sem lição escrita e sem IA.
          </p>
          <div>
            <label class="rotulo" for="t-titulo">Título da trilha</label>
            <input id="t-titulo" v-model="tituloTrilha" class="campo" type="text" required />
          </div>
          <div class="modos">
            <button
              v-for="(nome, modo) in MODOS"
              :key="modo"
              type="button"
              class="modo"
              :class="{ ativa: modoTrilha === modo }"
              :aria-pressed="modoTrilha === modo"
              @click="modoTrilha = modo"
            >
              <span class="nome-modo">
                {{ nome }}
                <span v-if="modo === recomendado" class="etiqueta">recomendado</span>
              </span>
              <span class="meta">{{ resumoDosModos[modo]?.texto }}</span>
            </button>
          </div>
          <p v-if="erroTrilha" class="aviso">{{ erroTrilha }}</p>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="mandandoTrilha || !tituloTrilha.trim()">
              {{ mandandoTrilha ? 'Criando…' : 'Criar a trilha' }}
            </button>
            <button class="botao discreto" type="button" @click="criandoTrilha = false">Cancelar</button>
          </div>
        </form>
      </div>

      <!-- T43: encaixar os cursos marcados numa trilha existente. -->
      <div v-if="encaixando" class="dialogo-fundo" @click.self="encaixando = false">
        <form class="painel pilha" role="dialog" aria-label="Encaixar cursos numa trilha" @submit.prevent="encaixarCursos">
          <h3>Encaixar numa trilha</h3>
          <p class="pequeno">
            A IA lê só os títulos das aulas ({{ tituloDoCurso(pastasMarcadas) }}) e diz em que
            etapas elas entram e o que falta. Nada é baixado.
          </p>
          <div>
            <label class="rotulo" for="e-trilha">Trilha</label>
            <select id="e-trilha" v-model="trilhaEscolhida" class="campo" required>
              <option value="" disabled>Escolher…</option>
              <option v-for="t in trilhas" :key="t.id" :value="String(t.id)">{{ t.titulo }}</option>
            </select>
          </div>
          <div>
            <label class="rotulo" for="e-pedido">Pedido para a IA <span class="fraco">(opcional)</span></label>
            <textarea id="e-pedido" v-model="pedidoCurso" class="campo" rows="2" placeholder="Ex.: priorize as aulas de DDD tático; a parte de eventos já está coberta." />
          </div>
          <p v-if="erroEncaixe" class="aviso">{{ erroEncaixe }}</p>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="mandandoEncaixe || !trilhaEscolhida">
              {{ mandandoEncaixe ? 'Pedindo…' : 'Pedir o encaixe' }}
            </button>
            <button class="botao discreto" type="button" @click="encaixando = false">Cancelar</button>
          </div>
        </form>
      </div>

      <!-- T43: criar a trilha vazia com o nome escolhido e encaixar os cursos nela. -->
      <div v-if="criandoDosCursos" class="dialogo-fundo" @click.self="criandoDosCursos = false">
        <form class="painel pilha" role="dialog" aria-label="Criar trilha a partir destes cursos" @submit.prevent="criarTrilhaDosCursos">
          <h3>Criar trilha a partir destes cursos</h3>
          <p class="pequeno">
            A trilha nasce vazia e a IA monta os módulos e as etapas pelos títulos das aulas
            ({{ tituloDoCurso(pastasMarcadas) }}). Nada é baixado.
          </p>
          <div>
            <label class="rotulo" for="c-titulo">Nome da trilha</label>
            <input id="c-titulo" v-model="tituloTrilhaCursos" class="campo" type="text" required />
          </div>
          <div>
            <label class="rotulo" for="c-pedido">Pedido para a IA <span class="fraco">(opcional)</span></label>
            <textarea id="c-pedido" v-model="pedidoCursoNovo" class="campo" rows="2" />
          </div>
          <p v-if="erroEncaixe" class="aviso">{{ erroEncaixe }}</p>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="mandandoEncaixe || !tituloTrilhaCursos.trim()">
              {{ mandandoEncaixe ? 'Criando…' : 'Criar e encaixar' }}
            </button>
            <button class="botao discreto" type="button" @click="criandoDosCursos = false">Cancelar</button>
          </div>
        </form>
      </div>

      <!-- T44: processar aulas na Entrada — a Trilha baixa o que precisar e transcreve no Whisper. -->
      <div v-if="dialogoProcessar" class="dialogo-fundo" @click.self="dialogoProcessar = null">
        <form class="painel pilha" role="dialog" aria-label="Processar aulas na Entrada" @submit.prevent="processar">
          <h3>Processar na Entrada</h3>
          <p class="pequeno">
            {{ dialogoProcessar.aulas.length === 1 ? '1 aula' : `${dialogoProcessar.aulas.length} aulas` }}
            <template v-if="tempoDeAudio"> · ~{{ segundosParaTempo(tempoDeAudio) }} de áudio</template>
            — {{ dialogoProcessar.titulo }}. Cada uma vira uma fonte da Entrada: a Trilha pede o
            download ao Streaming se a aula ainda não estiver no servidor e transcreve o áudio no
            Whisper, sem gastar a IA.
          </p>
          <label class="linha marca-linha">
            <input v-model="fragmentarDepois" type="checkbox" class="marca" />
            Dividir em fragmentos quando terminar
          </label>
          <p v-if="erroProcessar" class="aviso">{{ erroProcessar }}</p>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="mandandoProcessar || !dialogoProcessar.aulas.length">
              {{ mandandoProcessar ? 'Mandando…' : 'Processar' }}
            </button>
            <button class="botao discreto" type="button" @click="dialogoProcessar = null">Cancelar</button>
          </div>
        </form>
      </div>
    </div>
  </Pagina>
</template>

<style scoped>
.lista-arvore {
  display: grid;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.item {
  display: flex;
  align-items: center;
  gap: var(--e2);
  width: 100%;
  padding: var(--e2) var(--e1);
  border: 0;
  border-radius: var(--raio-p);
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.item:hover { background: var(--superficie-2); }
.item:hover .nome { color: var(--marca); }
.sobre {
  display: grid;
  flex: 1;
  gap: 2px;
  min-width: 0;
}
.nome { overflow-wrap: anywhere; }
.item .nome { font-weight: 550; }
.seta { flex-shrink: 0; color: var(--texto-fraco); }

.caminho {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px 8px;
}
.crumb {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 6px 0;
  border: 0;
  background: none;
  color: var(--texto-suave);
  font: inherit;
  font-size: 0.9rem;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crumb:hover { color: var(--marca); }
.sep { color: var(--texto-fraco); }

.cabeca-pasta {
  display: grid;
  gap: 6px;
}
.cabeca-pasta h2 {
  margin: 0;
  font-size: 1.15rem;
  overflow-wrap: anywhere;
}
.acoes-curso {
  display: flex;
  flex-wrap: wrap;
  gap: var(--e1);
}

.aula {
  display: flex;
  align-items: center;
  gap: var(--e2);
  padding: var(--e2) var(--e1);
  border-radius: var(--raio-p);
}
.icone-aula {
  display: grid;
  flex-shrink: 0;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--raio-p);
  background: var(--superficie);
  color: var(--marca);
}
.icone-aula.vazia { color: var(--texto-fraco); }
.aula.falta .nome { color: var(--texto-suave); }
.meta { color: var(--texto-suave); font-size: 0.8rem; }

.modos {
  display: grid;
  gap: var(--e1);
}
.modo {
  display: grid;
  gap: 4px;
  padding: var(--e2) var(--e3);
  border: 1px solid var(--borda);
  border-radius: var(--raio-p);
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.modo.ativa { border-color: var(--marca); background: var(--superficie); }
.nome-modo { display: flex; align-items: center; gap: var(--e2); font-weight: 550; }

.linha-pasta {
  display: flex;
  align-items: center;
}
.marca-linha {
  display: flex;
  align-items: center;
  gap: var(--e2);
  font-size: var(--fonte-pequena);
}
.marca {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  margin-left: var(--e1);
  accent-color: var(--marca);
}
.acoes-cursos {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--e2);
}
.acoes-cursos .pequeno { flex-basis: 100%; }
.link { color: var(--marca); }
</style>
