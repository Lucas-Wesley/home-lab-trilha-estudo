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
import { encaixesDaPasta, trilhasDaPasta, usosPorAula } from '../usos.js'

// Meus arquivos (T41): os cursos do Telegram que estão no Streaming, em curso › módulo › aula
// (a árvore é a do Streaming, S55 de lá — pastas.js copiado). A Trilha só chama a API dele:
// baixar (uma aula, um módulo ou o curso) e apagar (lixeira); nada baixa sozinho.
const props = defineProps({
  pasta: { type: String, default: '' }, // o caminho da rota (#/entrada/arquivos/<caminho>)
})

const ABAS = [
  { id: 'videos', nome: 'Vídeos' },
  { id: 'arquivos', nome: 'Meus arquivos' },
  { id: 'cursos', nome: 'Cursos' },
]
const mudarAba = (id) => {
  if (id === 'videos') location.hash = '#/entrada'
  if (id === 'cursos') location.hash = '#/entrada/cursos'
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
  if (p.tamanho) partes.push(`${tamanho(p.tamanho)} no servidor`)
  if (p.falta) partes.push(`faltam ${tamanho(p.falta)}`)
  return partes.join(' · ')
}

// O selo do que já está no servidor, em destaque na pasta e na aula: tudo, parte ou nada.
function seloDaPasta(p) {
  if (!p.total) return null
  if (p.baixadas >= p.total) return { classe: 'tudo', texto: 'Baixada' }
  if (p.baixadas > 0) return { classe: 'parte', texto: `${p.baixadas} de ${p.total} baixadas` }
  return { classe: 'nada', texto: 'Só no Telegram' }
}
function seloDaAula(a) {
  if (a.video) return { classe: 'tudo', texto: 'Baixada' }
  if (estaPedida(a)) return { classe: 'parte', texto: 'Na fila' }
  return { classe: 'nada', texto: 'Só no Telegram' }
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

// Onde as aulas já estão: "na trilha X" na aula e na pasta, e os encaixes esperando aprovação.
const usos = ref({ aulas: [], encaixes: [] })
async function buscarUsos() {
  try {
    usos.value = await api('/api/arquivos/usos')
  } catch { /* sem os usos, a tela só não mostra onde cada aula está */ }
}
const mapaDeUsos = computed(() => usosPorAula(usos.value))
const usosDaAula = (a) => {
  const vistas = new Set()
  return (mapaDeUsos.value.get(referenciaDaAula(a)) || []).filter((u) => !vistas.has(u.trilha_id) && vistas.add(u.trilha_id))
}
const trilhasDe = (p) => trilhasDaPasta(p, mapaDeUsos.value)
const encaixesDe = (p) => encaixesDaPasta(p, usos.value.encaixes)
const linkTrilha = (id) => link('trilhas', String(id))
const pctBaixado = (p) => (p.total ? Math.round((p.baixadas / p.total) * 100) : 0)

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
const materialDe = (pastas) => {
  if (!pastas.length) return null
  const referencia = referenciaDoCurso(pastas)
  if (!referencia) return null
  return { tipo: 'curso', titulo: tituloDoCurso(pastas), referencia }
}
const materialDoCurso = computed(() => materialDe(pastasMarcadas.value))
// As pastas do encaixe que está sendo pedido: as marcadas, ou só a pasta aberta ("Encaixar esta
// pasta numa trilha", sem precisar marcar).
const pastasDoEncaixe = ref([])
const limparMarcadas = () => { marcadas.value = new Set() }

// "Encaixar numa trilha": seletor de trilha + pedido opcional.
const encaixando = ref(false)
const trilhas = ref([])
// Um curso pode ir a várias trilhas (ex.: o MBA em Banco de dados, DDD e Observabilidade): um
// encaixe por trilha, e cada um só pega as aulas do recorte daquela trilha.
const trilhasEscolhidas = ref(new Set())
const alternarTrilha = (id) => {
  const s = new Set(trilhasEscolhidas.value)
  s.has(id) ? s.delete(id) : s.add(id)
  trilhasEscolhidas.value = s
}
const pedidoCurso = ref('')
const mandandoEncaixe = ref(false)
const erroEncaixe = ref('')
// "Criar trilha a partir destes cursos": nome da trilha + pedido opcional.
const criandoDosCursos = ref(false)
const tituloTrilhaCursos = ref('')
const pedidoCursoNovo = ref('')
// Aviso depois de pedir: "Encaixe pedido na trilha X" com link para a aba Encaixar.
const avisoEncaixe = ref([]) // [{id, titulo}]

async function abrirEncaixar(pastas = pastasMarcadas.value) {
  pastasDoEncaixe.value = pastas
  erroEncaixe.value = ''
  pedidoCurso.value = ''
  try {
    trilhas.value = await api('/api/trilhas')
  } catch (e) {
    erroEncaixe.value = e.message
  }
  trilhasEscolhidas.value = new Set()
  encaixando.value = true
}

// O material "curso" vai pelo encaixe de sempre — a IA lê só os títulos das aulas.
async function pedirEncaixe(lista, pedido) {
  const material = materialDe(pastasDoEncaixe.value)
  if (!material) {
    erroEncaixe.value = 'Uma das pastas marcadas não é de um grupo do Telegram.'
    return false
  }
  const feitos = []
  try {
    for (const t of lista) {
      await api(`/api/trilhas/${t.id}/encaixar`, { metodo: 'POST', corpo: { material, pedido } })
      feitos.push({ id: t.id, titulo: t.titulo })
    }
  } finally {
    avisoEncaixe.value = feitos
  }
  if (pastasDoEncaixe.value === pastasMarcadas.value) marcadas.value = new Set()
  await buscarUsos()
  return true
}

async function encaixarCursos() {
  const lista = trilhas.value.filter((x) => trilhasEscolhidas.value.has(x.id))
  if (!lista.length) return
  mandandoEncaixe.value = true
  erroEncaixe.value = ''
  try {
    if (await pedirEncaixe(lista, pedidoCurso.value)) encaixando.value = false
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
    if (await pedirEncaixe([t], pedidoCursoNovo.value)) criandoDosCursos.value = false
  } catch (e) {
    erroEncaixe.value = e.message
  } finally {
    mandandoEncaixe.value = false
  }
}

function abrirCriarDosCursos(pastas = pastasMarcadas.value) {
  pastasDoEncaixe.value = pastas
  tituloTrilhaCursos.value = tituloDoCurso(pastas)
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
  await Promise.all([buscarEntrada(), buscarUsos()])
})
onBeforeUnmount(() => clearInterval(relogioDaFila))
</script>

<template>
  <Pagina>
    <template #topo>
      <div class="topo-pagina">
        <h1>Entrada</h1>
        <div class="acoes">
          <span v-if="dados && livre" class="pequeno fraco">{{ tamanho(livre) }} livres no servidor</span>
        </div>
      </div>
      <AbasPagina class="abas-entrada" :abas="ABAS" model-value="arquivos" rotulo="Partes da Entrada" @update:model-value="mudarAba" />
    </template>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <p v-else-if="aviso" class="aviso neutro">{{ aviso }}</p>
    <!-- S57 do Streaming: o disco dos vídeos não pôde ser lido; a lista vem do banco mesmo assim. -->
    <p v-if="dados?.aviso" class="aviso">{{ dados.aviso }}</p>
    <p v-if="dados && !telegramInstalado" class="aviso">O tdl não está instalado no Streaming: dá para ver as aulas, mas não baixar.</p>
    <p v-if="avisoEncaixe.length" class="aviso neutro">
      {{ avisoEncaixe.length === 1 ? 'Encaixe pedido na trilha' : 'Encaixes pedidos nas trilhas' }}
      <template v-for="(t, i) in avisoEncaixe" :key="t.id">
        <a :href="`${linkTrilha(t.id)}#encaixar`">{{ t.titulo }}</a>{{ i < avisoEncaixe.length - 1 ? ', ' : '' }}
      </template>:
      a proposta aparece na aba Encaixar material de cada uma, para você aprovar.
    </p>

    <!-- Raiz: um cartão por curso, lado a lado. -->
    <template v-if="!pasta">
      <p class="explicacao">
        Os cursos do Telegram que estão no Streaming, em curso › módulo › aula. Abra um curso para
        baixar, criar uma trilha igual a ele ou encaixar as aulas numa trilha; marque vários para
        juntá-los num encaixe só.
      </p>
      <p v-if="dados && !arvore.pastas.length" class="fraco">
        Nenhum curso ainda. Os grupos de aulas salvos no Streaming aparecem aqui.
      </p>
      <ul class="cursos">
        <li v-for="g in arvore.pastas" :key="g.nome" class="cartao curso" :class="{ marcada: marcadas.has(chavePasta(g)) }">
          <div class="curso-topo">
            <input type="checkbox" class="marca" :checked="marcadas.has(chavePasta(g))" :aria-label="`Marcar ${g.nome}`" @change="alternarMarcada(g)" />
            <a class="curso-nome" :href="linkPasta(g.caminho.join('/'))">{{ g.nome }}</a>
          </div>
          <div class="curso-baixa">
            <span v-if="seloDaPasta(g)" class="selo" :class="seloDaPasta(g).classe">{{ seloDaPasta(g).texto }}</span>
            <span class="pequeno fraco">{{ resumoDaPasta(g) }}</span>
          </div>
          <div class="progresso" :title="`${pctBaixado(g)}% baixado`"><span :style="{ width: pctBaixado(g) + '%' }" /></div>
          <div v-if="trilhasDe(g).length || encaixesDe(g).length" class="usos">
            <a v-for="t in trilhasDe(g).slice(0, 3)" :key="t.trilha_id" class="etiqueta verde" :href="linkTrilha(t.trilha_id)">
              <Icone nome="trilhas" :tamanho="12" /> {{ t.inteira ? `virou a trilha ${t.trilha}` : `${t.aulas} em ${t.trilha}` }}
            </a>
            <a v-for="e in encaixesDe(g)" :key="'e' + e.trilha_id" class="etiqueta ambar" :href="`${linkTrilha(e.trilha_id)}#encaixar`">
              encaixe esperando em {{ e.trilha }}
            </a>
          </div>
        </li>
      </ul>
    </template>

    <!-- Numa pasta: a lista ao centro e, ao lado, o resumo, onde as aulas já estão e as ações. -->
    <template v-else>
      <nav class="caminho" aria-label="Onde estou">
        <a class="crumb" :href="linkPasta('')"><Icone nome="voltar" :tamanho="14" /> Meus arquivos</a>
        <template v-for="c in caminhoAtual.slice(0, -1)" :key="c.caminho">
          <span class="sep" aria-hidden="true">›</span>
          <a class="crumb" :title="c.nome" :href="linkPasta(c.caminho)">{{ c.nome }}</a>
        </template>
      </nav>

      <p v-if="dados && !pastaAtual" class="fraco">Esta pasta não existe mais.</p>
      <template v-else-if="pastaAtual">
        <h2 class="nome-pasta">{{ pastaAtual.nome }}</h2>
        <div class="pilha-quadros">

        <div v-if="pastaAtual.pastas.length" class="quadro">
          <div class="quadro-topo">
            <h3 class="rotulo-secao">Pastas · {{ pastaAtual.pastas.length }}</h3>
            <span class="pequeno fraco">marque para encaixar várias juntas</span>
          </div>
          <ul class="lista linhas">
            <li v-for="g in pastaAtual.pastas" :key="g.nome" class="linha-item" :class="{ marcada: marcadas.has(chavePasta(g)) }">
              <input type="checkbox" class="marca" :checked="marcadas.has(chavePasta(g))" :aria-label="`Marcar ${g.nome}`" @change="alternarMarcada(g)" />
              <a class="item-principal" :href="linkPasta(g.caminho.join('/'))">
                <Icone nome="pasta" :tamanho="16" class="icone-item" />
                <span class="sobre">
                  <span class="nome">{{ g.nome }}</span>
                  <span class="meta">{{ resumoDaPasta(g) }}</span>
                  <span v-if="g.baixadas && g.baixadas < g.total" class="progresso mini"><span :style="{ width: pctBaixado(g) + '%' }" /></span>
                </span>
              </a>
              <span v-if="seloDaPasta(g)" class="selo" :class="seloDaPasta(g).classe">{{ seloDaPasta(g).texto }}</span>
              <span class="usos">
                <a v-for="t in trilhasDe(g).slice(0, 2)" :key="t.trilha_id" class="etiqueta verde" :href="linkTrilha(t.trilha_id)" :title="`${t.aulas} aulas na trilha ${t.trilha}`">
                  <Icone nome="trilhas" :tamanho="12" /> {{ t.inteira ? t.trilha : `${t.aulas} em ${t.trilha}` }}
                </a>
                <a v-for="e in encaixesDe(g)" :key="'e' + e.trilha_id" class="etiqueta ambar" :href="`${linkTrilha(e.trilha_id)}#encaixar`">esperando · {{ e.trilha }}</a>
              </span>
              <Icone nome="seta-direita" :tamanho="15" class="seta" />
            </li>
          </ul>
        </div>

        <div v-if="pastaAtual.aulas.length" class="quadro">
          <div class="quadro-topo">
            <h3 class="rotulo-secao">Aulas · {{ pastaAtual.aulas.length }}</h3>
          </div>
          <ul class="lista linhas">
            <li v-for="a in pastaAtual.aulas" :key="a.chave" class="linha-item aula" :class="{ falta: !a.video }">
              <span class="icone-aula" :class="{ vazia: !a.video }"><Icone nome="video" :tamanho="15" /></span>
              <span class="sobre">
                <span class="nome">{{ a.nome }}</span>
                <span class="meta">
                  {{ [segundosParaTempo(a.duracao), tamanho(a.video ? a.video.tamanho : a.tamanho)].filter(Boolean).join(' · ') }}
                </span>
              </span>
              <span class="selo" :class="seloDaAula(a).classe">{{ seloDaAula(a).texto }}</span>
              <span class="usos">
                <a v-for="u in usosDaAula(a)" :key="u.trilha_id" class="etiqueta verde" :href="link('trilhas', u.trilha_id, 'etapas', u.etapa_id)" :title="`Etapa ${u.etapa}`">
                  <Icone nome="trilhas" :tamanho="12" /> {{ u.trilha }}
                </a>
                <a v-if="fonteDaAula(a)" class="etiqueta azul" :href="link('entrada', String(fonteDaAula(a).id))">na Entrada</a>
              </span>
              <span class="acoes-aula">
                <button v-if="!fonteDaAula(a) && referenciaDaAula(a)" class="botao discreto" title="Transcrever no Whisper e dividir por assunto" @click="abrirProcessarAula(a)">Processar</button>
                <template v-if="a.video">
                  <a class="botao discreto" :href="linkAssistir(a)" target="_blank" rel="noopener">Assistir</a>
                  <button class="botao discreto icone perigo" aria-label="Apagar do servidor" title="Apagar do servidor" @click="apagarAula(a)">
                    <Icone nome="lixeira" :tamanho="14" />
                  </button>
                </template>
                <button v-else-if="!estaPedida(a)" class="botao" :disabled="baixandoAulas || !telegramInstalado" @click="baixarAula(a)">Baixar</button>
              </span>
            </li>
          </ul>
        </div>
        <p v-if="!pastaAtual.pastas.length && !pastaAtual.aulas.length" class="fraco">Nenhuma aula nesta pasta.</p>
        </div>
      </template>
    </template>

    <template v-if="pasta && pastaAtual" #lateral>
      <div class="quadro">
        <div class="quadro-topo"><h2 class="rotulo-secao">Esta pasta</h2></div>
        <div class="numeros">
          <div><span class="valor">{{ pastaAtual.total }}</span><span class="pequeno fraco">aulas</span></div>
          <div><span class="valor">{{ pastaAtual.baixadas }}</span><span class="pequeno fraco">baixadas</span></div>
          <div><span class="valor">{{ tamanho(pastaAtual.tamanho) || '0' }}</span><span class="pequeno fraco">no servidor</span></div>
        </div>
        <div class="progresso"><span :style="{ width: pctBaixado(pastaAtual) + '%' }" /></div>
        <p class="pequeno fraco">
          <template v-if="pastaAtual.falta">Faltam {{ tamanho(pastaAtual.falta) }} no Telegram.</template>
          <template v-else>Tudo baixado.</template>
          <template v-if="naFilaAqui > 0"> {{ naFilaAqui }} na fila.</template>
        </p>
      </div>

      <div class="quadro">
        <div class="quadro-topo"><h2 class="rotulo-secao">Nas trilhas</h2></div>
        <ul v-if="trilhasDe(pastaAtual).length || encaixesDe(pastaAtual).length" class="lista linhas compacta">
          <li v-for="t in trilhasDe(pastaAtual)" :key="t.trilha_id" class="linha-uso">
            <a :href="linkTrilha(t.trilha_id)">{{ t.trilha }}</a>
            <span class="pequeno fraco">{{ t.inteira ? 'a pasta inteira' : `${t.aulas} de ${aulasDaPasta(pastaAtual).length} aulas` }}</span>
          </li>
          <li v-for="e in encaixesDe(pastaAtual)" :key="'e' + e.trilha_id" class="linha-uso">
            <a :href="`${linkTrilha(e.trilha_id)}#encaixar`">{{ e.trilha }}</a>
            <span class="etiqueta ambar">encaixe esperando aprovação</span>
          </li>
        </ul>
        <p v-else class="pequeno fraco">Nenhuma aula desta pasta está numa trilha ainda.</p>
      </div>

      <div class="quadro acoes-pasta">
        <div class="quadro-topo"><h2 class="rotulo-secao">O que fazer com ela</h2></div>
        <button v-if="pastaAtual.total" class="botao largo" @click="abrirCriarTrilha">
          <Icone nome="trilhas" :tamanho="15" /> Criar trilha igual a este curso
        </button>
        <button class="botao largo" :disabled="!materialDe([pastaAtual])" @click="abrirEncaixar([pastaAtual])">
          <Icone nome="mais" :tamanho="15" /> Encaixar numa trilha
        </button>
        <button v-if="aulasDaPasta(pastaAtual).some(referenciaDaAula)" class="botao largo" @click="abrirProcessarPasta(pastaAtual)">
          <Icone nome="entrada" :tamanho="15" /> Processar na Entrada
        </button>
        <p class="pequeno fraco ajuda-acoes">
          <strong>Criar igual</strong>: a trilha copia a estrutura do curso, sem IA.
          <strong>Encaixar</strong>: a IA lê só os títulos e põe as aulas nas etapas (ou cria etapas novas).
          <strong>Processar</strong>: transcreve as aulas no Whisper e divide por assunto, como um vídeo do YouTube.
        </p>
        <div v-if="aulasQueFaltam || pastaAtual.baixadas" class="separador" />
        <button v-if="aulasQueFaltam" class="botao largo" :disabled="baixandoAulas || !telegramInstalado || naoCabe" @click="baixarAulas(faltaAqui)">
          {{ baixandoAulas ? 'Mandando…' : `Baixar ${aulasQueFaltam === pastaAtual.total ? 'tudo' : 'o que falta'} (${quantasAulas(aulasQueFaltam)} · ${tamanho(pesoDaFalta)})` }}
        </button>
        <p v-if="naoCabe && aulasQueFaltam" class="aviso">
          Não cabe: faltam {{ tamanho(pesoDaFalta) }} e há {{ tamanho(livre) }} livres no servidor. Apague algo antes.
        </p>
        <button v-if="pastaAtual.baixadas" class="botao largo perigo" @click="apagarPasta(pastaAtual)">
          <Icone nome="lixeira" :tamanho="14" /> Apagar as baixadas
        </button>
      </div>
    </template>
  </Pagina>

  <div v-if="pastasMarcadas.length" class="espaco-barra" aria-hidden="true" />
  <!-- Pastas marcadas: a barra fica presa no pé, perto de onde se marca, em qualquer pasta. -->
  <div v-if="pastasMarcadas.length" class="barra-selecao" role="region" aria-label="Pastas marcadas">
    <div class="selecao-texto">
      <strong>{{ pastasMarcadas.length }} {{ pastasMarcadas.length === 1 ? 'pasta marcada' : 'pastas marcadas' }}</strong>
      <span class="pequeno fraco nomes-marcados">{{ tituloDoCurso(pastasMarcadas) }}</span>
      <span v-if="!materialDoCurso" class="pequeno aviso-texto">uma delas não é de um grupo do Telegram</span>
    </div>
    <div class="selecao-acoes">
      <button class="botao primario" :disabled="!materialDoCurso" @click="abrirEncaixar()">Encaixar numa trilha</button>
      <button class="botao" :disabled="!materialDoCurso" @click="abrirCriarDosCursos()">Criar trilha com elas</button>
      <button class="botao discreto icone fechar-selecao" aria-label="Desmarcar todas" title="Desmarcar todas" @click="limparMarcadas"><Icone nome="fechar" :tamanho="15" /></button>
    </div>
  </div>

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
      <h3>Encaixar em trilhas</h3>
      <p class="pequeno">
        A IA lê só os títulos das aulas ({{ tituloDoCurso(pastasDoEncaixe) }}) e diz em que
        etapas elas entram e o que falta. Nada é baixado. Marque quantas trilhas quiser: cada uma
        recebe um encaixe próprio e pega só as aulas do assunto dela.
      </p>
      <fieldset class="escolha-trilhas">
        <legend class="rotulo">Trilhas <span class="fraco">· {{ trilhasEscolhidas.size }} marcadas</span></legend>
        <label v-for="t in trilhas" :key="t.id" class="opcao-trilha">
          <input type="checkbox" :checked="trilhasEscolhidas.has(t.id)" @change="alternarTrilha(t.id)" />
          <span>{{ t.titulo }}</span>
        </label>
      </fieldset>
      <div>
        <label class="rotulo" for="e-pedido">Pedido para a IA <span class="fraco">(opcional)</span></label>
        <textarea id="e-pedido" v-model="pedidoCurso" class="campo" rows="2" placeholder="Ex.: priorize as aulas de DDD tático; a parte de eventos já está coberta." />
      </div>
      <p v-if="erroEncaixe" class="aviso">{{ erroEncaixe }}</p>
      <div class="linha">
        <button class="botao primario" type="submit" :disabled="mandandoEncaixe || !trilhasEscolhidas.size">
          {{ mandandoEncaixe ? 'Pedindo…' : trilhasEscolhidas.size > 1 ? `Pedir ${trilhasEscolhidas.size} encaixes` : 'Pedir o encaixe' }}
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
        ({{ tituloDoCurso(pastasDoEncaixe) }}). Nada é baixado.
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
</template>

<style scoped>
.abas-entrada { margin-bottom: var(--e5); }

/* ---------- raiz: cartões de curso lado a lado ---------- */
.cursos {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--e3);
  margin: 0;
  padding: 0;
  list-style: none;
}
.curso { display: grid; gap: var(--e2); align-content: start; }
.curso.marcada, .linha-item.marcada { border-color: var(--borda-forte); background: var(--superficie); }
.curso-topo { display: flex; align-items: flex-start; gap: var(--e2); }
.curso-nome {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--texto);
  text-decoration: none;
  overflow-wrap: anywhere;
}
.curso-nome:hover { color: var(--link); }
.curso .pequeno { margin: 0; }

.usos { display: flex; flex-wrap: wrap; gap: var(--e1); }
.curso-baixa { display: flex; flex-wrap: wrap; align-items: center; gap: var(--e1) var(--e2); }

/* O selo do download tem outro formato que as etiquetas de trilha (contorno e bolinha), para não
   se confundir com o verde de "está na trilha". */
.selo {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6px;
  padding: 2px 9px 2px 7px;
  border: 1px solid currentColor;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.5;
  white-space: nowrap;
}
.selo::before { content: ''; width: 8px; height: 8px; border-radius: 50%; background: currentColor; }
.selo.tudo { color: var(--verde); background: var(--verde-fundo); }
.selo.parte { color: var(--azul); background: var(--azul-fundo); }
.selo.parte::before { background: linear-gradient(90deg, currentColor 50%, transparent 50%); box-shadow: inset 0 0 0 1.5px currentColor; }
.selo.nada { color: var(--texto-suave); border-style: dashed; }
.selo.nada::before { background: none; box-shadow: inset 0 0 0 1.5px currentColor; }
.progresso.mini { display: block; max-width: 160px; height: 4px; margin-top: 3px; }

.escolha-trilhas { display: grid; gap: 2px; max-height: 40vh; margin: 0; padding: 0; overflow-y: auto; border: 0; }
.opcao-trilha { display: flex; align-items: center; gap: var(--e2); padding: 6px var(--e1); border-radius: var(--raio-p); cursor: pointer; }
.opcao-trilha:hover { background: var(--superficie); }
.usos .etiqueta { gap: 4px; text-decoration: none; white-space: nowrap; max-width: 100%; overflow: hidden; text-overflow: ellipsis; }

/* ---------- numa pasta ---------- */
.caminho { display: flex; flex-wrap: wrap; align-items: center; gap: 2px var(--e2); margin-bottom: var(--e2); }
.crumb {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  font-size: 13px;
  color: var(--texto-suave);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.crumb:hover { color: var(--texto); }
.sep { color: var(--texto-fraco); }
.nome-pasta { margin: 0 0 var(--e4); overflow-wrap: anywhere; }
.pilha-quadros { display: grid; gap: var(--e4); }

.linhas { margin: 0; padding: 0; list-style: none; }
.linha-item {
  display: flex;
  align-items: center;
  gap: var(--e3);
  min-height: 44px;
  padding: var(--e2) var(--e2);
  border-bottom: 1px solid var(--borda);
  border-radius: var(--raio-p);
}
.linha-item:last-child { border-bottom: 0; }
.linha-item:hover { background: var(--superficie); }
.item-principal {
  display: flex;
  flex: 1;
  align-items: center;
  gap: var(--e3);
  min-width: 0;
  color: var(--texto);
  text-decoration: none;
}
.item-principal:hover .nome { color: var(--link); }
.icone-item { flex: none; color: var(--texto-fraco); }
.sobre { display: grid; flex: 1; gap: 2px; min-width: 0; }
.nome { font-size: 14px; font-weight: 550; overflow-wrap: anywhere; }
.meta { font-size: 12.5px; color: var(--texto-suave); }
.linha-item > .usos { flex: 0 1 auto; justify-content: flex-end; max-width: 45%; }
.seta { flex: none; color: var(--texto-fraco); }

.icone-aula {
  display: grid;
  flex: none;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: var(--raio-p);
  background: var(--superficie-2);
  color: var(--texto);
}
.icone-aula.vazia { background: none; border: 1px dashed var(--borda-forte); color: var(--texto-fraco); }
.aula.falta .nome { color: var(--texto-suave); }
.acoes-aula { display: flex; flex: none; align-items: center; gap: 2px; }

/* ---------- lateral ---------- */
.numeros { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--e2); margin-bottom: var(--e3); }
.numeros > div { display: grid; gap: 2px; }
.numeros .valor { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; line-height: 1.2; }
.quadro > .progresso { margin-bottom: var(--e2); }
.quadro > .pequeno { margin: 0; }
.compacta .linha-uso { display: grid; gap: 2px; padding: var(--e2) 0; border-bottom: 1px solid var(--borda); }
.compacta .linha-uso:last-child { border-bottom: 0; }
.linha-uso a { font-weight: 550; color: var(--texto); text-decoration: none; }
.linha-uso a:hover { color: var(--link); }
.acoes-pasta { display: grid; gap: var(--e2); }
.acoes-pasta .quadro-topo { margin-bottom: var(--e1); }
.acoes-pasta .botao { justify-content: flex-start; }
.ajuda-acoes { margin: var(--e1) 0 0; line-height: 1.5; }
.ajuda-acoes strong { color: var(--texto-suave); font-weight: 600; }
.separador { height: 1px; margin: var(--e2) 0; background: var(--borda); }

/* ---------- caixas de marcar e a barra das marcadas ---------- */
.marca { flex: none; width: 16px; height: 16px; margin: 2px 0 0; accent-color: var(--texto); cursor: pointer; }
.barra-selecao {
  position: fixed;
  left: calc(var(--nav) + var(--e5));
  right: var(--e5);
  bottom: var(--e4);
  z-index: 20;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--e2) var(--e4);
  max-width: var(--pagina);
  margin: 0 auto;
  padding: var(--e3) var(--e4);
  border: 1px solid var(--borda-forte);
  border-radius: var(--raio);
  background: var(--superficie);
  box-shadow: 0 8px 24px rgb(0 0 0 / 0.28);
}
.selecao-texto { display: grid; gap: 2px; min-width: 0; }
.nomes-marcados { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.aviso-texto { color: var(--vermelho); }
.selecao-acoes { display: flex; flex-wrap: wrap; align-items: center; gap: var(--e2); }

/* Espaço no pé para a barra não cobrir a última linha. */
.espaco-barra { height: 96px; }

.modos { display: grid; gap: var(--e1); }
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
.modo.ativa { border-color: var(--borda-forte); background: var(--superficie-2); }
.nome-modo { display: flex; align-items: center; gap: var(--e2); font-weight: 550; }
.marca-linha { display: flex; align-items: center; gap: var(--e2); font-size: 13px; }

@media (max-width: 899px) {
  .barra-selecao { left: var(--e3); right: var(--e3); bottom: calc(64px + env(safe-area-inset-bottom)); }
  .selecao-texto { padding-right: 36px; }
  .fechar-selecao { position: absolute; top: var(--e2); right: var(--e2); }
  .linha-item { flex-wrap: wrap; }
  /* No celular o nome fica com a linha inteira; o selo e as trilhas descem para baixo dele. */
  .linha-item > .item-principal { flex: 1 1 calc(100% - 64px); }
  .aula > .sobre { flex: 1 1 calc(100% - 44px); }
  .linha-item > .selo { order: 5; margin-left: 56px; }
  .aula > .selo { margin-left: 40px; }
  .linha-item > .usos { max-width: none; order: 6; flex: 1 1 auto; justify-content: flex-start; }
  .linha-item > .usos:empty { display: none; }
  .aula .acoes-aula { order: 4; }
}
</style>
