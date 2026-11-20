<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import EscolherMaterial from '../EscolherMaterial.vue'
import AbasPagina from '../AbasPagina.vue'
import DominioEtapa from '../DominioEtapa.vue'
import EncaixarMaterial from '../EncaixarMaterial.vue'
import EstudoEtapa from '../EstudoEtapa.vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import PropostaLicao from '../PropostaLicao.vue'
import PropostaMaterial from '../PropostaMaterial.vue'
import ResumoEtapa from '../ResumoEtapa.vue'
import RevisarRoteiro from '../RevisarRoteiro.vue'
import TrechoYouTube from '../TrechoYouTube.vue'
import Videoaula from '../Videoaula.vue'
import { trechos } from '../trechoYoutube.js'
import { videosDaEtapa } from '../videoaula.js'
import { api } from '../api.js'
import { abreEmOutraAba, hrefMaterial } from '../material.js'
import { link } from '../rotas.js'
import { textoComCodigo } from '../markdown.js'
import { BLOOM, PAPEIS, PROFUNDIDADES, STATUS, TIPOS_MATERIAL, duracao } from '../rotulos.js'

const TIPOS_ETAPA = { base: 'Etapa de base', reforco: 'Reforço', projeto: 'Projeto do módulo' }

const props = defineProps({
  trilhaId: { type: Number, required: true },
  etapaId: { type: Number, required: true },
  ancora: { type: String, default: '' },
})

const t = ref(null)
const e = ref(null)
const erro = ref('')
const aviso = ref('')
const titulos = ref([])
const proposta = ref(null)
const propostaAberta = ref(false)
// Fazer exercícios ou simulado começa a etapa: o status muda no servidor.
const versaoEstudo = ref(0)
async function recarregarEtapa() {
  versaoEstudo.value++
  e.value = await api(`/api/etapas/${props.etapaId}`)
  t.value = await api(`/api/trilhas/${props.trilhaId}`)
}
async function aoAprovar() {
  e.value = await api(`/api/etapas/${props.etapaId}`)
  t.value = await api(`/api/trilhas/${props.trilhaId}`)
}

async function carregar() {
  try {
    ;[t.value, e.value] = await Promise.all([api(`/api/trilhas/${props.trilhaId}`), api(`/api/etapas/${props.etapaId}`)])
  } catch (err) {
    erro.value = err.status === 404 ? 'Etapa não encontrada.' : err.message
  }
}
onMounted(carregar)

const todas = computed(() => (t.value ? t.value.modulos.flatMap((m) => m.etapas.map((x) => ({ ...x, modulo: m.titulo }))) : []))
const porId = computed(() => Object.fromEntries(todas.value.map((x) => [x.id, x])))
const posicao = computed(() => todas.value.findIndex((x) => x.id === props.etapaId))
const anterior = computed(() => todas.value[posicao.value - 1])
const proxima = computed(() => todas.value[posicao.value + 1])
const modulo = computed(() => todas.value[posicao.value]?.modulo || '')
const profundidade = computed(() => e.value?.profundidade || t.value?.profundidade || 3)
const doModulo = computed(() => todas.value.filter((x) => x.modulo === modulo.value))
const posicaoNoModulo = computed(() => doModulo.value.findIndex((x) => x.id === props.etapaId) + 1)
const totalNoModulo = computed(() => doModulo.value.length)

// ---------- abas ----------
const contagem = ref({ exercicios: 0, simulado: 0 })
const abas = computed(() => [
  { id: 'licao', nome: aulaEmVideo.value ? 'Texto da aula' : e.value?.tipo === 'projeto' ? 'Enunciado' : 'Lição', feito: !!e.value?.licao },
  { id: 'resumo', nome: 'Meu resumo' },
  { id: 'exercicios', nome: 'Exercícios', conta: contagem.value.exercicios },
  { id: 'simulado', nome: 'Simulado', conta: contagem.value.simulado },
  { id: 'flashcards', nome: 'Flashcards' },
  { id: 'dominio', nome: 'Domínio' },
  { id: 'fontes', nome: 'Fontes', conta: e.value?.materiais?.length || 0 },
])
const IDS_ABAS = ['licao', 'resumo', 'exercicios', 'simulado', 'flashcards', 'dominio', 'fontes']
// A âncora do endereço escolhe a aba (#exercicios) ou um título da lição (#por-que-existe).
const abaDaAncora = (a) => (a === 'material' ? 'fontes' : IDS_ABAS.includes(a) ? a : 'licao')
const aba = ref(abaDaAncora(props.ancora))
const ancoraDaLicao = computed(() => (IDS_ABAS.includes(props.ancora) || props.ancora === 'material' ? '' : props.ancora))
watch(() => props.ancora, (a) => (aba.value = abaDaAncora(a)))
watch(() => props.etapaId, () => (aba.value = abaDaAncora(props.ancora)))
const barraAbas = ref(null)
async function trocarAba(id) {
  aba.value = id
  history.replaceState(null, '', `#/trilhas/${props.trilhaId}/etapas/${props.etapaId}${id === 'licao' ? '' : '#' + id}`)
  await nextTick()
  // Trocou de aba lá embaixo: volta para o começo dela.
  const barra = barraAbas.value?.barra
  if (barra && barra.getBoundingClientRect().top < 0) barra.scrollIntoView({ block: 'start' })
}

const materiaisPorPapel = computed(() => {
  const grupos = { principal: [], apoio: [], aprofundar: [] }
  for (const m of e.value?.materiais || []) (grupos[m.papel] || grupos.apoio).push(m)
  return Object.entries(grupos).filter(([, lista]) => lista.length)
})

// Videoaula (T30) e trecho do YouTube (T39): o vídeo principal é a aula e vem no topo.
const videos = computed(() => videosDaEtapa(e.value?.materiais))
const trechosYt = computed(() => trechos(e.value?.materiais))
const aulaEmVideo = computed(() => videos.value.length > 0 || trechosYt.value.length > 0)
const player = ref(null)
const playerYt = ref(null)
const irNoVideo = (parte, segundos) => (player.value || playerYt.value)?.ir(parte, segundos)

const baseDocs = computed(() => `#/trilhas/${props.trilhaId}/documentos/`)

const sugerir = ref(null)

// Revisar o tópico com IA (T34): aplicada a revisão, a etapa pode ter mudado ou saído da trilha.
const revisar = ref(null)
const revisaoAberta = ref(false)
async function aposRevisao() {
  try {
    await recarregarEtapa()
  } catch (err) {
    if (err.status === 404) location.hash = link('trilhas', props.trilhaId)
    else aviso.value = err.message
  }
}
const sugestaoAberta = ref(false)

async function mudarStatus(status) {
  aviso.value = ''
  try {
    e.value = await api(`/api/etapas/${props.etapaId}/status`, { metodo: 'POST', corpo: { status } })
    t.value = await api(`/api/trilhas/${props.trilhaId}`)
    if (status === 'concluida' && proxima.value) location.hash = link('trilhas', props.trilhaId, 'etapas', proxima.value.id)
  } catch (err) {
    aviso.value = err.message
  }
}

// ---------- edição ----------
const edicao = ref(null)
function editar() {
  escolhendo.value = false
  const x = e.value
  edicao.value = {
    titulo: x.titulo,
    objetivo: x.objetivo,
    bloom: x.bloom,
    profundidade: x.profundidade,
    tipo: x.tipo,
    minutos: x.minutos,
    criterio: x.criterio,
    foco: x.foco || '',
    requer: [...x.requer],
    materiais: x.materiais.map((m) => ({ ...m })),
    licao: x.licao || '',
  }
}
const escolhendo = ref(false)
function adicionarMaterial(m) {
  edicao.value.materiais.push(m)
}
function moverMaterial(i, delta) {
  const lista = edicao.value.materiais
  const j = i + delta
  if (j < 0 || j >= lista.length) return
  ;[lista[i], lista[j]] = [lista[j], lista[i]]
}
async function salvar() {
  aviso.value = ''
  try {
    e.value = await api(`/api/etapas/${props.etapaId}`, { metodo: 'PUT', corpo: { ...edicao.value, minutos: Number(edicao.value.minutos) || 0 } })
    t.value = await api(`/api/trilhas/${props.trilhaId}`)
    edicao.value = null
  } catch (err) {
    aviso.value = err.message
  }
}
async function excluir() {
  if (!confirm(`Excluir a etapa "${e.value.titulo}" e a lição dela?`)) return
  try {
    await api(`/api/etapas/${props.etapaId}`, { metodo: 'DELETE' })
    location.hash = link('trilhas', props.trilhaId)
  } catch (err) {
    aviso.value = err.message
  }
}
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas v-if="t" :trilha="t" :etapa-atual="etapaId" /></template>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-else-if="t && e">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('trilhas')">Trilhas</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <a :href="link('trilhas', t.id)">{{ t.titulo }}</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <span>{{ modulo }}</span>
      </nav>

      <!-- Leitura -->
      <template v-if="!edicao">
        <header class="cabecalho">
          <div>
            <h1>{{ e.titulo }}</h1>
            <p class="contexto">
              {{ modulo }} · etapa {{ posicaoNoModulo }} de {{ totalNoModulo }}
              <span v-if="TIPOS_ETAPA[e.tipo]" class="etiqueta ambar">{{ TIPOS_ETAPA[e.tipo] }}</span>
            </p>
          </div>
          <div class="linha acoes-topo">
            <button v-if="!revisaoAberta" class="botao discreto" @click="revisar?.abrir()">Revisar com IA</button>
            <button class="botao discreto" @click="editar">Editar</button>
          </div>
        </header>
        <p v-if="e.objetivo" class="objetivo" v-html="textoComCodigo(e.objetivo)" />

        <!-- Progresso: o estado, o que falta para concluir e as ações, num cartão só. -->
        <div class="situacao">
          <div class="situacao-texto">
            <div class="linha">
              <span class="rotulo sem-margem">Para concluir</span>
              <span class="etiqueta" :class="STATUS[e.status].cor">{{ STATUS[e.status].nome }}</span>
            </div>
            <p v-if="e.criterio" class="criterio" v-html="textoComCodigo(e.criterio)" />
            <p v-if="e.requer.length" class="antes">
              <span class="fraco">Antes:</span>
              <template v-for="(r, i) in e.requer" :key="r">
                <a :href="link('trilhas', t.id, 'etapas', r)">{{ porId[r]?.titulo }}</a><span v-if="porId[r]?.status === 'concluida'" class="feito" title="Concluída"> ✓</span><span v-if="i < e.requer.length - 1">, </span>
              </template>
            </p>
          </div>
          <div class="situacao-acoes">
            <button v-if="e.status === 'disponivel'" class="botao" @click="mudarStatus('em_andamento')">Começar</button>
            <button v-if="e.status !== 'concluida'" class="botao primario" @click="mudarStatus('concluida')">Concluir</button>
            <button v-if="e.status === 'disponivel' || e.status === 'em_andamento'" class="botao discreto" @click="mudarStatus('pulada')">Pular</button>
            <button v-if="e.status === 'concluida' || e.status === 'pulada'" class="botao discreto" @click="mudarStatus('disponivel')">Reabrir</button>
          </div>
        </div>
        <p v-if="aviso" class="aviso topo">{{ aviso }}</p>
        <RevisarRoteiro ref="revisar" :key="`roteiro-${e.id}`" :trilha="t" :etapa="e" @aberta="revisaoAberta = $event" @mudou="aposRevisao" />

        <Videoaula
          v-if="videos.length"
          ref="player"
          :key="`video-${e.id}`"
          :videos="videos"
          :etapa-id="e.id"
          :vistos="e.trechos_vistos"
          :proxima="proxima ? link('trilhas', t.id, 'etapas', proxima.id) : ''"
          class="topo"
          @mudou="recarregarEtapa"
        />
        <TrechoYouTube
          v-if="trechosYt.length"
          ref="playerYt"
          :key="`yt-${e.id}`"
          :trechos="trechosYt"
          :etapa-id="e.id"
          :vistos="e.trechos_vistos"
          :proxima="proxima ? link('trilhas', t.id, 'etapas', proxima.id) : ''"
          class="topo"
          @mudou="recarregarEtapa"
        />

        <!-- Abas: cada parte da etapa no seu lugar (fica no endereço: dá para voltar direto nela). -->
        <AbasPagina ref="barraAbas" class="abas-etapa" :abas="abas" :model-value="aba" rotulo="Partes da etapa" @update:model-value="trocarAba" />

        <div class="conteudo-aba">
          <section v-show="aba === 'licao'" id="licao">
            <div class="linha entre cabeca-aba">
              <span class="pequeno">{{ e.licao ? `${PROFUNDIDADES[profundidade]} · cerca de ${duracao(e.minutos)}` : '' }}</span>
              <button v-if="e.licao" class="botao discreto" @click="proposta?.abrir()">Pedir mudança</button>
            </div>
            <p v-if="!e.licao && !propostaAberta && trechosYt.length" class="suave">A aula desta etapa é o trecho do vídeo. Se quiser, a IA pode escrever a lição a partir da transcrição dele, como proposta. <button class="botao discreto" @click="proposta?.abrir()">Gerar lição escrita (opcional)</button></p>
            <p v-else-if="!e.licao && !propostaAberta" class="suave">Esta etapa ainda não tem {{ e.tipo === 'projeto' ? 'enunciado' : 'lição' }}. A IA escreve na profundidade da etapa ({{ PROFUNDIDADES[profundidade] }}), a partir das fontes, trazendo para dentro tudo o que importa; chega como proposta, você comenta e aprova. Também dá para escrever à mão em Editar.</p>
            <PropostaLicao ref="proposta" :key="e.id" :etapa-id="e.id" :licao-atual="e.licao || ''" :base-docs="baseDocs" @aprovada="aoAprovar" @aberta="propostaAberta = $event" />
            <Leitor v-if="e.licao" class="licao-etapa" :texto="e.licao" :ancora="ancoraDaLicao" :base-docs="baseDocs" @titulos="titulos = $event" />
          </section>

          <div v-show="aba === 'resumo'">
            <ResumoEtapa :key="`resumo-${e.id}`" :etapa-id="e.id" :tem-video="aulaEmVideo" @ir="irNoVideo" />
          </div>

          <EstudoEtapa
            v-show="['exercicios', 'simulado', 'flashcards'].includes(aba)"
            :key="`estudo-${e.id}-${!!e.licao}`"
            :etapa="e"
            :trilha="t"
            :aba="aba"
            @concluir="mudarStatus('concluida')"
            @mudou="recarregarEtapa"
            @contagem="contagem = $event"
          />

          <div v-show="aba === 'dominio'">
            <DominioEtapa :key="`dominio-${e.id}-${e.status}-${versaoEstudo}`" :etapa="e" :trilha-id="t.id" @mudou="recarregarEtapa" />
          </div>

          <section v-show="aba === 'fontes'" id="material">
            <div class="linha entre cabeca-aba">
              <span class="suave">De onde a lição sai. A lição já traz o que importa: não precisa abrir nada daqui.</span>
              <button v-if="!sugestaoAberta" class="botao discreto" @click="sugerir?.abrir()">Sugerir material</button>
            </div>
            <p v-if="!materiaisPorPapel.length" class="suave">Sem fonte ainda. A IA pode sugerir (web, os seus livros, o Acervo e os documentos da trilha), encaixe um material seu logo abaixo, ou adicione em Editar.</p>
            <div v-for="[papel, lista] in materiaisPorPapel" :key="papel" class="grupo-material">
              <span class="rotulo">{{ PAPEIS[papel] }}</span>
              <ul class="materiais">
                <li v-for="(m, i) in lista" :key="i">
                  <span class="etiqueta">{{ TIPOS_MATERIAL[m.tipo] }}</span>
                  <div>
                    <a v-if="hrefMaterial(m, t.id)" :href="hrefMaterial(m, t.id)" :target="abreEmOutraAba(m) ? '_blank' : undefined" :rel="abreEmOutraAba(m) ? 'noopener noreferrer' : undefined">{{ m.titulo }}</a>
                    <span v-else>{{ m.titulo }}</span>
                    <span v-if="m.localizador && m.tipo !== 'documento'" class="fraco"> · {{ m.localizador }}</span>
                    <div v-if="m.nota" class="suave nota">{{ m.nota }}</div>
                  </div>
                </li>
              </ul>
            </div>
            <PropostaMaterial ref="sugerir" :key="`material-${e.id}`" :etapa-id="e.id" :trilha-id="t.id" @aprovada="recarregarEtapa" @aberta="sugestaoAberta = $event" />
            <div class="encaixar-aqui">
              <span class="rotulo">Encaixar material neste tópico</span>
              <EncaixarMaterial :key="`encaixe-${e.id}`" :trilha="t" :etapa="e" @mudou="recarregarEtapa" />
            </div>
          </section>
        </div>

        <nav class="vizinhas" aria-label="Etapas vizinhas">
          <a v-if="anterior" class="cartao" :href="link('trilhas', t.id, 'etapas', anterior.id)">
            <span class="pequeno">Anterior</span>
            <strong>{{ anterior.titulo }}</strong>
          </a>
          <span v-else />
          <a v-if="proxima" class="cartao direita" :href="link('trilhas', t.id, 'etapas', proxima.id)">
            <span class="pequeno">Próxima</span>
            <strong>{{ proxima.titulo }}</strong>
          </a>
        </nav>
      </template>

      <!-- Edição -->
      <form v-else class="pilha" @submit.prevent="salvar">
        <h1>Editar etapa</h1>
        <div>
          <label class="rotulo" for="e-titulo">Título</label>
          <input id="e-titulo" v-model="edicao.titulo" class="campo" required />
        </div>
        <div>
          <label class="rotulo" for="e-obj">Objetivo <span class="fraco">(com um verbo: explicar, configurar, comparar…)</span></label>
          <textarea id="e-obj" v-model="edicao.objetivo" class="campo" rows="2" />
        </div>
        <div class="campos">
          <div>
            <label class="rotulo" for="e-bloom">Nível (Bloom)</label>
            <select id="e-bloom" v-model="edicao.bloom" class="campo">
              <option value="">—</option>
              <option v-for="(nome, v) in BLOOM" :key="v" :value="v">{{ nome }}</option>
            </select>
          </div>
          <div>
            <label class="rotulo" for="e-prof">Profundidade</label>
            <select id="e-prof" v-model.number="edicao.profundidade" class="campo">
              <option :value="0">A da trilha ({{ PROFUNDIDADES[t.profundidade] }})</option>
              <option v-for="(nome, n) in PROFUNDIDADES" :key="n" :value="Number(n)">{{ n }} · {{ nome }}</option>
            </select>
          </div>
          <div>
            <label class="rotulo" for="e-tipo">Tipo</label>
            <select id="e-tipo" v-model="edicao.tipo" class="campo">
              <option value="normal">Normal</option>
              <option value="base">Etapa de base</option>
              <option value="reforco">Reforço</option>
              <option value="projeto">Projeto do módulo</option>
            </select>
          </div>
          <div>
            <label class="rotulo" for="e-min">Minutos</label>
            <input id="e-min" v-model.number="edicao.minutos" class="campo" type="number" min="0" step="5" />
          </div>
        </div>
        <div>
          <label class="rotulo" for="e-crit">Critério de conclusão</label>
          <input id="e-crit" v-model="edicao.criterio" class="campo" placeholder="Ex.: responder às 5 perguntas sem olhar" />
        </div>
        <div>
          <label class="rotulo" for="e-foco">Foco da lição <span class="fraco">(opcional; vai para a IA ao gerar a lição)</span></label>
          <textarea id="e-foco" v-model="edicao.foco" class="campo" rows="2" placeholder="Ex.: a diferença entre NAT e CGNAT, com a tabela de tradução do OpenWrt" />
        </div>

        <fieldset class="caixa">
          <legend class="rotulo">Pré-requisitos</legend>
          <label v-for="x in todas.filter((x) => x.id !== etapaId)" :key="x.id" class="opcao">
            <input v-model="edicao.requer" type="checkbox" :value="x.id" />
            <span>{{ x.titulo }} <span class="fraco">· {{ x.modulo }}</span></span>
          </label>
        </fieldset>

        <fieldset class="caixa pilha">
          <legend class="rotulo">Material</legend>
          <div v-for="(m, i) in edicao.materiais" :key="i" class="material-edicao">
            <div class="campos">
              <select v-model="m.tipo" class="campo" aria-label="Tipo">
                <option v-for="(nome, v) in TIPOS_MATERIAL" :key="v" :value="v">{{ nome }}</option>
              </select>
              <select v-model="m.papel" class="campo" aria-label="Papel">
                <option v-for="(nome, v) in PAPEIS" :key="v" :value="v">{{ nome }}</option>
              </select>
            </div>
            <input v-model="m.titulo" class="campo" placeholder="Título" />
            <input
              v-model="m.referencia"
              class="campo"
              :list="m.tipo === 'documento' ? 'documentos-da-trilha' : undefined"
              :placeholder="{ web: 'https://…', documento: 'documento da trilha (ex.: 03-redes-separadas-firewall)', livro: 'id do livro na Biblioteca', acervo: 'pasta do Tema (temas/…) ou id da Referência', streaming: 'vídeo do Streaming: https://streaming.internal/api/video/proprio/57.mp4' }[m.tipo] || 'Referência'"
            />
            <input v-model="m.localizador" class="campo" :placeholder="{ documento: 'âncora da seção (opcional)', livro: 'cap. 3, caps. 2–4 ou p. 45–60 (vazio = o livro todo)', acervo: 'arquivo do Tema (opcional)' }[m.tipo] || 'cap., página, tempo (opcional)'" />
            <input v-model="m.nota" class="campo" placeholder="Nota: o que ler e por quê (opcional)" />
            <div class="linha">
              <button type="button" class="botao discreto icone" aria-label="Subir" @click="moverMaterial(i, -1)">↑</button>
              <button type="button" class="botao discreto icone" aria-label="Descer" @click="moverMaterial(i, 1)">↓</button>
              <button type="button" class="botao discreto" @click="edicao.materiais.splice(i, 1)">Remover</button>
            </div>
          </div>
          <EscolherMaterial v-if="escolhendo" :documentos="t.documentos || []" @escolher="adicionarMaterial" @fechar="escolhendo = false" />
          <div v-else><button type="button" class="botao" @click="escolhendo = true">Adicionar material</button></div>
          <datalist id="documentos-da-trilha">
            <option v-for="d in t.documentos || []" :key="d.slug" :value="d.slug">{{ d.titulo }}</option>
          </datalist>
        </fieldset>

        <div>
          <label class="rotulo" for="e-licao">Lição (Markdown)</label>
          <textarea id="e-licao" v-model="edicao.licao" class="campo licao" spellcheck="true" />
        </div>

        <p v-if="aviso" class="aviso">{{ aviso }}</p>
        <div class="linha entre">
          <div class="linha">
            <button class="botao primario" type="submit">Salvar</button>
            <button class="botao discreto" type="button" @click="edicao = null">Cancelar</button>
          </div>
          <button class="botao discreto perigo" type="button" @click="excluir">Excluir a etapa</button>
        </div>
      </form>
    </template>

    <template #lateral>
      <ul v-if="titulos.length && !edicao && aba === 'licao'" class="sumario">
        <li v-for="h in titulos" :key="h.id" :class="{ sub: h.nivel === 3 }">
          <a :href="`#/trilhas/${trilhaId}/etapas/${etapaId}#${h.id}`">{{ h.texto }}</a>
        </li>
      </ul>
    </template>
  </Pagina>
</template>

<style scoped>
.cabecalho { display: flex; justify-content: space-between; align-items: flex-start; gap: var(--e4); }
.cabecalho h1 { font-size: 28px; }
.contexto { display: flex; align-items: center; gap: var(--e2); flex-wrap: wrap; margin: var(--e2) 0 0; font-size: 14px; color: var(--texto-fraco); }
.objetivo { font-size: 16px; line-height: 1.6; color: var(--texto-suave); margin: var(--e4) 0 0; max-width: 72ch; }
.objetivo :deep(code), .criterio :deep(code) { font-size: 0.86em; padding: 1px 5px; background: var(--superficie-2); border: 1px solid var(--borda); border-radius: var(--raio-e); }
.topo { margin-top: var(--e4); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }

/* Cartão de progresso */
.situacao { padding: var(--e4) var(--e5); background: var(--superficie); border: 1px solid var(--borda); border-radius: var(--raio); display: flex; gap: var(--e5); align-items: center; justify-content: space-between; margin-top: var(--e5); }
.situacao-texto { display: grid; gap: var(--e2); min-width: 0; }
.situacao-acoes { display: flex; gap: var(--e2); flex: none; }
.criterio { margin: 0; font-size: 14.5px; line-height: 1.55; }
.antes { margin: 0; font-size: 13.5px; line-height: 1.5; }
.feito { color: var(--verde); }

.abas-etapa { margin-top: var(--e6); }
.conteudo-aba { margin-top: var(--e5); min-height: 240px; }
/* Dentro da aba, o nome já está na aba: some o título repetido e o espaço de seção. */
.conteudo-aba :deep(.secao) { margin-top: 0; }
.conteudo-aba :deep(.secao > h2), .conteudo-aba :deep(.secao > .linha > h2), .conteudo-aba :deep(.secao > .cabeca-secao) { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
.conteudo-aba :deep(.secao > .linha.entre) { justify-content: flex-end; }
.conteudo-aba :deep(#resumo > .linha.entre) { justify-content: flex-end; }
.cabeca-aba { margin-bottom: var(--e3); gap: var(--e3); }
.acoes-topo { flex-wrap: nowrap; gap: var(--e1); }
.encaixar-aqui { margin-top: var(--e6); padding-top: var(--e5); border-top: 1px solid var(--borda); }

.grupo-material + .grupo-material { margin-top: var(--e4); }
.materiais { list-style: none; margin: var(--e2) 0 0; padding: 0; display: grid; gap: var(--e3); }
.materiais li { display: flex; gap: var(--e3); align-items: flex-start; font-size: 15px; line-height: 1.5; }
.materiais .etiqueta { flex: none; margin-top: 2px; min-width: 76px; justify-content: center; }
.nota { font-size: 14px; margin-top: 2px; }
.vizinhas { display: grid; grid-template-columns: 1fr 1fr; gap: var(--e3); margin-top: var(--e7); }
.vizinhas .cartao { display: grid; gap: 2px; }
.vizinhas .direita { text-align: right; }
.caixa { border: 1px solid var(--borda); border-radius: var(--raio); padding: var(--e3) var(--e4) var(--e4); margin: 0; }
.caixa legend { padding: 0 var(--e1); margin: 0; }
.opcao { display: flex; gap: var(--e2); align-items: flex-start; padding: 3px 0; font-size: 14px; }
.material-edicao { display: grid; gap: var(--e2); padding-bottom: var(--e3); border-bottom: 1px solid var(--borda); }
.licao { min-height: 360px; font-family: var(--fonte-mono); font-size: 13px; }
/* O título da lição repete o da etapa, que já está no topo da página. */
.licao-etapa :deep(h1:first-child) { display: none; }
/* E o quadro de abertura (objetivo, antes, tempo) repete o topo da página. */
.licao-etapa :deep(h1:first-child + blockquote) { display: none; }
/* Ir a um título pelo sumário não o esconde atrás da barra e das abas presas no topo. */
.licao-etapa :deep(h2), .licao-etapa :deep(h3) { scroll-margin-top: calc(var(--barra) + 64px); }
.perigo { color: var(--vermelho); }
@media (max-width: 699px) {
  .cabecalho h1 { font-size: 24px; }
  .situacao { flex-direction: column; align-items: stretch; gap: var(--e3); }
  .situacao-acoes { flex-wrap: wrap; }
  .vizinhas { grid-template-columns: 1fr; }
}
</style>
