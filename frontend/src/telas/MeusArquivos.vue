<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AbasPagina from '../AbasPagina.vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { linkPasta } from '../rotas.js'
import { acharPasta, aulasPedidas, montarArvore, oQueFalta, tamanho } from '../pastas.js'
import { segundosParaTempo } from '../videoaula.js'
import { MODOS, modoRecomendado, pacoteDoCurso, resumoDoPacote } from '../cursoParaTrilha.js'

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

      <!-- Raiz: um cartão por grupo/curso. -->
      <ul v-if="!pasta && arvore.pastas.length" class="lista-arvore">
        <li v-for="g in arvore.pastas" :key="g.nome">
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
            <!-- T42: criar a trilha igual ao curso (Encaixar e Processar entram nas próximas fases). -->
            <div class="acoes-curso">
              <button v-if="pastaAtual.total" class="botao" @click="abrirCriarTrilha">Criar trilha igual a este curso</button>
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
            <li v-for="g in pastaAtual.pastas" :key="g.nome">
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
</style>
