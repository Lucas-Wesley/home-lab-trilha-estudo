<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import MiniTeste from '../MiniTeste.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { abreEmOutraAba, hrefMaterial } from '../material.js'
import { link } from '../rotas.js'
import { BLOOM, PROFUNDIDADES, TIPOS_MATERIAL, duracao } from '../rotulos.js'

// Criar a trilha com a IA (fase 4, DECISOES.md §3 e T25). Cada passo é uma proposta que espera
// você: a entrevista (responder), o mapa (marcar o recorte), a validação da lista (aceitar
// sugestões) e o plano (comentar, pedir ajuste, aprovar). Aprovar o plano cria a trilha.
const props = defineProps({ trilhaId: { type: Number, required: true } })

const dados = ref(null)
const trilha = ref(null)
const erro = ref('')
const ocupado = ref(false)

async function carregar() {
  try {
    ;[dados.value, trilha.value] = await Promise.all([api(`/api/trilhas/${props.trilhaId}/criacao`), api(`/api/trilhas/${props.trilhaId}`)])
  } catch (e) {
    erro.value = e.status === 404 ? 'Esta trilha não foi criada com a IA.' : e.message
  }
}

const criacao = computed(() => dados.value?.criacao)
const propostas = computed(() => dados.value?.propostas || [])
const ultima = computed(() => propostas.value.at(-1))
const gerando = computed(() => propostas.value.find((p) => p.estado === 'gerando'))
const aguardando = computed(() => propostas.value.find((p) => p.estado === 'aguardando'))
const pronta = computed(() => criacao.value?.passo === 'pronta')
// Nada aberto (descartou, ou falhou): gerar o passo de novo, com um pedido.
const parado = computed(() => criacao.value && !pronta.value && !gerando.value && !aguardando.value)

let relogio = 0
watch(gerando, (g) => {
  clearInterval(relogio)
  if (g) relogio = setInterval(carregar, 4000)
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

async function acao(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    const r = await fn()
    if (r && r.criacao) dados.value = r
    else await carregar()
    return true
  } catch (e) {
    erro.value = e.message
    return false
  } finally {
    ocupado.value = false
  }
}

const PASSOS = {
  ia: [
    { id: 'entrevista', nome: 'Entrevista' },
    { id: 'mapa', nome: 'Mapa e recorte' },
    { id: 'plano', nome: 'Plano' },
    { id: 'pronta', nome: 'Trilha criada' },
  ],
  lista: [
    { id: 'validacao', nome: 'Validação da lista' },
    { id: 'plano', nome: 'Plano' },
    { id: 'pronta', nome: 'Trilha criada' },
  ],
  material: [
    { id: 'validacao', nome: 'Espinha dorsal' },
    { id: 'plano', nome: 'Plano' },
    { id: 'pronta', nome: 'Trilha criada' },
  ],
}
const doMaterial = computed(() => criacao.value?.entrada === 'material')
const passos = computed(() => PASSOS[criacao.value?.entrada] || [])
const indicePasso = computed(() => passos.value.findIndex((p) => p.id === criacao.value?.passo))
const ESTADO_TAREFA = { na_fila: 'na fila', rodando: 'trabalhando', concluida: 'pronta', falhou: 'falhou', cancelada: 'cancelada' }
const OQUE = { entrevista: 'pesquisando o assunto e preparando as perguntas', mapa: 'montando o mapa do assunto', validacao: 'conferindo a sua lista', plano: 'montando o plano da trilha' }

// ---------- gerar de novo e cancelar ----------
const pedidoNovo = ref('')
async function gerarDeNovo() {
  if (await acao(() => api(`/api/trilhas/${props.trilhaId}/criacao/gerar`, { metodo: 'POST', corpo: { pedido: pedidoNovo.value } }))) pedidoNovo.value = ''
}
const cancelar = () => acao(() => api(`/api/tarefas/${gerando.value.tarefa.id}/cancelar`, { metodo: 'POST' }))
async function descartar(p) {
  if (!confirm('Descartar esta proposta? Depois dá para pedir outra, com um pedido.')) return
  await acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
}

const vista = computed(() => aguardando.value)
const conteudo = computed(() => vista.value?.conteudo || {})
const aprovar = (corpo) => acao(() => api(`/api/propostas/${vista.value.id}/aprovar`, { metodo: 'POST', corpo }))

// ---------- entrevista ----------
const respostas = ref({})
const comentarioRodada = ref('')
watch(
  () => (vista.value?.tipo === 'entrevista' ? vista.value.id : 0),
  () => {
    respostas.value = Object.fromEntries((conteudo.value.perguntas || []).map((q) => [q.id, { escolhas: [], outro: '', sugira: false }]))
    comentarioRodada.value = ''
  },
  { immediate: true },
)
function escolher(q, opcao) {
  const r = respostas.value[q.id]
  r.sugira = false
  if (q.multipla) r.escolhas = r.escolhas.includes(opcao) ? r.escolhas.filter((x) => x !== opcao) : [...r.escolhas, opcao]
  else r.escolhas = r.escolhas[0] === opcao ? [] : [opcao]
}
function sugira(q) {
  const r = respostas.value[q.id]
  r.sugira = !r.sugira
  if (r.sugira) r.escolhas = []
}
const rodadaAtual = computed(() => (criacao.value?.rodadas.length || 0) + 1)
const podeOutraRodada = computed(() => rodadaAtual.value < (dados.value?.limite.rodadas || 0) && !conteudo.value.suficiente)
const semResposta = computed(() => Object.values(respostas.value).filter((r) => !r.escolhas.length && !r.outro.trim() && !r.sugira).length)
function responder(proximo) {
  const lista = Object.entries(respostas.value).map(([pergunta, r]) => ({ pergunta, ...r }))
  aprovar({ respostas: lista, comentario: comentarioRodada.value, proximo })
}

// ---------- mapa ----------
const recorte = ref(new Set())
const niveis = ref({ nivel_partida: '', nivel_alvo: '', profundidade: 3, objetivo: '' })
watch(
  () => (vista.value && vista.value.tipo !== 'entrevista' ? vista.value.id : 0),
  (id) => {
    if (!id) return
    const c = conteudo.value
    recorte.value = new Set(c.recorte_sugerido || [])
    niveis.value = {
      nivel_partida: c.nivel_partida || trilha.value?.nivel_partida || '',
      nivel_alvo: c.nivel_alvo || trilha.value?.nivel_alvo || '',
      profundidade: trilha.value?.profundidade || 3,
      objetivo: c.objetivo || '',
    }
    aceitas.value = new Set()
  },
  { immediate: true },
)
function alternar(chave) {
  const s = new Set(recorte.value)
  s.has(chave) ? s.delete(chave) : s.add(chave)
  recorte.value = s
}
function alternarModulo(m) {
  const s = new Set(recorte.value)
  const todas = m.etapas.every((e) => s.has(e.chave))
  m.etapas.forEach((e) => (todas ? s.delete(e.chave) : s.add(e.chave)))
  recorte.value = s
}
const NIVEIS = { basico: 'básico', intermediario: 'intermediário', avancado: 'avançado' }
const ordemDoMapa = computed(() => (conteudo.value.mapa || []).flatMap((m) => m.etapas.map((e) => e.chave)))
// Mini-teste de nível (T9): os títulos dos itens, e "usar a sugestão" (nível; no mapa, tira do
// recorte o que ele já domina). Ele confere antes de aprovar.
const titulosDosItens = computed(() => {
  const c = conteudo.value
  const lista = c.mapa ? c.mapa.flatMap((m) => m.etapas) : c.itens || []
  return Object.fromEntries(lista.map((x) => [x.chave, x.titulo]))
})
function usarNivelamento(r) {
  niveis.value.nivel_partida = r.nivel_sugerido
  if (vista.value?.tipo === 'mapa') {
    const s = new Set(recorte.value)
    r.dominados.forEach((k) => s.delete(k))
    recorte.value = s
  }
}
function aprovarMapa() {
  aprovar({ ...niveis.value, recorte: ordemDoMapa.value.filter((k) => recorte.value.has(k)), pedido: pedidoProximo.value })
}

// ---------- validação ----------
const aceitas = ref(new Set())
function alternarSugestao(id) {
  const s = new Set(aceitas.value)
  s.has(id) ? s.delete(id) : s.add(id)
  aceitas.value = s
}
const ACOES = { inserir: 'Inserir antes', dividir: 'Dividir', observar: 'Observação' }
// Cada sugestão aparece junto do item: "inserir" antes dele, as outras depois.
const listaComSugestoes = computed(() => {
  const c = conteudo.value
  if (!c.itens) return []
  const usadas = new Set()
  const linhas = []
  for (const it of c.itens) {
    for (const s of c.sugestoes.filter((s) => s.acao === 'inserir' && s.antes_de === it.chave)) {
      linhas.push({ sugestao: s })
      usadas.add(s.id)
    }
    linhas.push({ item: it })
    for (const s of c.sugestoes.filter((s) => s.acao !== 'inserir' && s.item === it.chave)) {
      linhas.push({ sugestao: s })
      usadas.add(s.id)
    }
  }
  for (const s of c.sugestoes.filter((s) => !usadas.has(s.id))) linhas.push({ sugestao: s })
  return linhas
})
function aprovarValidacao() {
  aprovar({ ...niveis.value, objetivo: '', aceitas: [...aceitas.value], pedido: pedidoProximo.value })
}

// ---------- plano ----------
const plano = computed(() => conteudo.value.trilha || null)
const avisosGraves = computed(() => (conteudo.value.revisao || []).filter((a) => a.grave))
const avisosGerais = computed(() => (conteudo.value.revisao || []).filter((a) => !a.grave && !a.onde))
const avisosDe = (onde) => (conteudo.value.revisao || []).filter((a) => a.onde === onde)
const etapasDoPlano = computed(() => (plano.value?.modulos || []).flatMap((m) => m.etapas))
const tituloDa = (chave) => etapasDoPlano.value.find((e) => e.chave === chave)?.titulo || chave
const minutosDoPlano = computed(() => etapasDoPlano.value.reduce((s, e) => s + (e.minutos || 0), 0))
const linksRuins = computed(() => (conteudo.value.links || []).filter((l) => l.situacao !== 'ok'))

// ---------- comentários, ajuste (mapa, validação e plano) ----------
const comentario = ref({ texto: '', ancora: '' })
const campoComentario = ref(null)
const ancoras = computed(() => {
  const c = conteudo.value
  if (vista.value?.tipo === 'plano' && plano.value) {
    return plano.value.modulos.flatMap((m, i) => [
      { id: `modulo:${i + 1}`, nome: `Módulo ${i + 1}: ${m.titulo}` },
      ...m.etapas.map((e) => ({ id: `etapa:${e.chave}`, nome: `— ${e.titulo}` })),
    ])
  }
  if (vista.value?.tipo === 'mapa') return (c.mapa || []).flatMap((m) => m.etapas.map((e) => ({ id: `etapa:${e.chave}`, nome: e.titulo })))
  if (vista.value?.tipo === 'validacao') return (c.itens || []).map((e) => ({ id: `etapa:${e.chave}`, nome: e.titulo }))
  return []
})
const nomeAncora = (a) => (a ? ancoras.value.find((x) => x.id === a)?.nome.replace(/^— /, '') || a : 'Geral')
async function comentarEm(ancora) {
  comentario.value.ancora = ancora
  await nextTick()
  campoComentario.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  campoComentario.value?.focus({ preventScroll: true })
}
async function comentar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/comentarios`, { metodo: 'POST', corpo: comentario.value }))) comentario.value = { texto: '', ancora: '' }
}
const tirarComentario = (c) => acao(() => api(`/api/comentarios/${c.id}`, { metodo: 'DELETE' }))
const meus = computed(() => (vista.value?.comentarios || []).filter((c) => c.autor === 'lucas'))
const respostasIA = computed(() => {
  const todos = Object.fromEntries(propostas.value.flatMap((p) => p.comentarios).map((c) => [c.id, c]))
  return (vista.value?.comentarios || []).filter((c) => c.autor === 'ia').map((c) => ({ ...c, original: todos[c.responde_a] }))
})
const pedidoAjuste = ref('')
async function ajustar() {
  if (await acao(() => api(`/api/propostas/${vista.value.id}/ajustar`, { metodo: 'POST', corpo: { pedido: pedidoAjuste.value } }))) pedidoAjuste.value = ''
}
// Pedido que vai junto do próximo passo (ao aprovar o mapa ou a validação).
const pedidoProximo = ref('')

// O que já foi decidido, para reler.
const rodadasFeitas = computed(() =>
  (criacao.value?.rodadas || []).map((r) => ({
    ...r,
    linhas: r.perguntas.map((q) => {
      const x = (r.respostas || []).find((a) => a.pergunta === q.id)
      const partes = x ? [...x.escolhas, x.outro, x.sugira ? 'não sei, sugira' : ''].filter(Boolean) : []
      return { pergunta: q.texto, resposta: partes.join(' · ') || '(sem resposta)' }
    }),
  })),
)
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas v-if="trilha" :trilha="trilha" /></template>

    <nav class="migalhas" aria-label="Você está em">
      <a :href="link('trilhas')">Trilhas</a>
      <Icone nome="seta-direita" :tamanho="14" />
      <a v-if="trilha" :href="link('trilhas', trilhaId)">{{ trilha.titulo }}</a>
      <Icone nome="seta-direita" :tamanho="14" />
      <span>Criar com a IA</span>
    </nav>

    <p v-if="erro && !dados" class="aviso">{{ erro }}</p>
    <div v-if="criacao" class="pilha conteudo">
      <div>
      <h1>{{ criacao.tema }}</h1>
      <p v-if="criacao.pedido" class="lead">{{ criacao.pedido }}</p>

      </div>
      <ol class="passos" aria-label="Passos">
        <li v-for="(p, i) in passos" :key="p.id" :class="{ feito: i < indicePasso || pronta, atual: i === indicePasso && !pronta }">
          <span class="bolinha">{{ i < indicePasso || pronta ? '✓' : i + 1 }}</span>{{ p.nome }}
        </li>
      </ol>

      <p v-if="erro" class="aviso">{{ erro }}</p>

      <details v-if="rodadasFeitas.length && criacao.passo !== 'entrevista'" class="cartao decidido">
        <summary>A entrevista ({{ rodadasFeitas.length }} rodada{{ rodadasFeitas.length > 1 ? 's' : '' }})</summary>
        <div v-for="(r, i) in rodadasFeitas" :key="r.proposta_id" class="pilha">
          <span v-if="rodadasFeitas.length > 1" class="rotulo">Rodada {{ i + 1 }}</span>
          <dl class="ficha">
            <template v-for="l in r.linhas" :key="l.pergunta"><dt>{{ l.pergunta }}</dt><dd>{{ l.resposta }}</dd></template>
            <template v-if="r.comentario"><dt>Comentário</dt><dd>{{ r.comentario }}</dd></template>
          </dl>
        </div>
      </details>

      <!-- Gerando -->
      <div v-if="gerando" class="cartao linha entre">
        <div>
          <strong>A IA está {{ doMaterial && gerando.tipo === 'validacao' ? 'lendo o material e montando a espinha dorsal' : OQUE[gerando.tipo] }}: {{ ESTADO_TAREFA[gerando.tarefa?.estado] || 'na fila' }}…</strong>
          <div class="suave">Leva alguns minutos; pode sair desta tela.<template v-if="gerando.pedido"> Pedido: “{{ gerando.pedido }}”</template></div>
        </div>
        <button class="botao discreto" :disabled="ocupado" @click="cancelar">Cancelar</button>
      </div>

      <!-- Parado: falhou, cancelou ou descartou -->
      <div v-if="parado" class="cartao pilha">
        <p v-if="ultima?.estado === 'falhou'" class="aviso sem-margem">Não saiu: {{ ultima.erro || 'erro sem mensagem' }}</p>
        <p v-else-if="ultima" class="sem-margem">A última proposta foi {{ ultima.estado === 'descartada' ? 'descartada' : 'interrompida' }}. Peça outra, dizendo o que quer diferente.</p>
        <label class="rotulo" for="c-pedido-novo">Pedido para a IA (opcional)</label>
        <textarea id="c-pedido-novo" v-model="pedidoNovo" class="campo" rows="2" placeholder="Ex.: o mapa ficou raso no começo; pergunte sobre o meu objetivo." />
        <div><button class="botao primario" :disabled="ocupado" @click="gerarDeNovo">{{ ultima ? 'Gerar de novo' : 'Começar' }}</button></div>
      </div>

      <!-- Pronta -->
      <div v-if="pronta" class="cartao pilha">
        <strong>A trilha foi criada.</strong>
        <p class="sem-margem">Ela está como rascunho. Na página dela, defina a meta e o ritmo e ative: as lições de cada etapa são geradas quando você chegar nelas.</p>
        <div><a class="botao primario" :href="link('trilhas', trilhaId)">Abrir a trilha</a></div>
      </div>

      <template v-if="vista && vista.conteudo">
        <p v-if="vista.pedido" class="suave sem-margem">Pedido: “{{ vista.pedido }}”</p>
        <div v-if="conteudo.notas" class="cartao cheio">
          <span class="rotulo">Notas da IA</span>
          <p class="sem-margem pre">{{ conteudo.notas }}</p>
        </div>
        <div v-if="respostasIA.length" class="cartao cheio pilha">
          <span class="rotulo">Respostas aos seus comentários</span>
          <div v-for="r in respostasIA" :key="r.id" class="resposta">
            <div v-if="r.original" class="suave"><span class="fraco">{{ nomeAncora(r.original.ancora) }}:</span> “{{ r.original.texto }}”</div>
            <div>{{ r.texto }}</div>
          </div>
        </div>

        <!-- Entrevista -->
        <section v-if="vista.tipo === 'entrevista'" class="pilha">
          <div v-if="conteudo.panorama" class="cartao panorama">
            <span class="rotulo">O assunto</span>
            <Leitor :texto="conteudo.panorama" />
          </div>
          <h2>Rodada {{ rodadaAtual }}<span class="fraco"> de até {{ dados.limite.rodadas }}</span></h2>
          <p v-if="!conteudo.perguntas.length" class="sem-margem">A IA não tem mais perguntas: o que você disse basta para o mapa.</p>
          <fieldset v-for="q in conteudo.perguntas" :key="q.id" class="cartao pergunta pilha">
            <legend class="sem-margem"><strong>{{ q.texto }}</strong></legend>
            <p class="pequeno sem-margem">{{ q.por_que }}<template v-if="q.multipla"> · pode escolher várias</template></p>
            <div class="opcoes">
              <button
                v-for="o in q.opcoes"
                :key="o"
                type="button"
                class="botao opcao"
                :class="{ marcada: respostas[q.id].escolhas.includes(o) }"
                :aria-pressed="respostas[q.id].escolhas.includes(o)"
                @click="escolher(q, o)"
              >{{ o }}</button>
              <button type="button" class="botao opcao discreto" :class="{ marcada: respostas[q.id].sugira }" :aria-pressed="respostas[q.id].sugira" @click="sugira(q)">Não sei, sugira</button>
            </div>
            <input v-model="respostas[q.id].outro" class="campo" :aria-label="`Outra resposta para: ${q.texto}`" placeholder="Outro, ou um detalhe sobre a escolha" />
          </fieldset>
          <div class="cartao pilha">
            <label class="rotulo" for="c-comentario-rodada">Comentário (opcional)</label>
            <textarea id="c-comentario-rodada" v-model="comentarioRodada" class="campo" rows="2" placeholder="Algo que as perguntas não cobriram." />
            <div class="linha">
              <button class="botao primario" :disabled="ocupado" @click="responder('mapa')">Responder e montar o mapa</button>
              <button v-if="podeOutraRodada" class="botao" :disabled="ocupado" @click="responder('rodada')">Responder e ter mais uma rodada</button>
              <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
            </div>
            <p class="pequeno sem-margem">
              <template v-if="semResposta">{{ semResposta }} pergunta{{ semResposta > 1 ? 's' : '' }} sem resposta: a IA decide e diz o que escolheu. </template>
              <template v-if="conteudo.suficiente && conteudo.perguntas.length">A IA acha que, com estas respostas, já dá para montar o mapa.</template>
            </p>
          </div>
        </section>

        <!-- Mapa -->
        <section v-if="vista.tipo === 'mapa'" class="pilha">
          <div>
            <label class="rotulo" for="c-objetivo">Ao final, você será capaz de…</label>
            <textarea id="c-objetivo" v-model="niveis.objetivo" class="campo" rows="2" />
          </div>
          <h2>O mapa do assunto</h2>
          <p class="sem-margem">Marque o que entra na trilha. A IA sugeriu um recorte<template v-if="conteudo.por_que_recorte">: {{ conteudo.por_que_recorte }}</template></p>
          <div v-for="(m, i) in conteudo.mapa" :key="i" class="cartao modulo pilha">
            <label class="linha cabeca-modulo">
              <input type="checkbox" :checked="m.etapas.every((e) => recorte.has(e.chave))" :aria-label="`Marcar o módulo ${m.titulo} inteiro`" @change="alternarModulo(m)" />
              <span><strong>{{ m.titulo }}</strong><span v-if="m.descricao" class="suave"> · {{ m.descricao }}</span></span>
            </label>
            <ul class="etapas-mapa">
              <li v-for="e in m.etapas" :key="e.chave" :class="{ fora: !recorte.has(e.chave) }">
                <label class="linha">
                  <input type="checkbox" :checked="recorte.has(e.chave)" @change="alternar(e.chave)" />
                  <span class="texto-etapa"><strong>{{ e.titulo }}</strong> <span class="etiqueta">{{ NIVEIS[e.nivel] || e.nivel }}</span><br /><span class="suave">{{ e.resumo }}</span></span>
                </label>
                <button class="botao discreto pequeno-botao" @click="comentarEm('etapa:' + e.chave)">Comentar</button>
              </li>
            </ul>
          </div>
          <p class="pequeno sem-margem">{{ recorte.size }} de {{ ordemDoMapa.length }} etapas marcadas.</p>
        </section>

        <!-- Validação da lista -->
        <section v-if="vista.tipo === 'validacao'" class="pilha">
          <h2>{{ doMaterial ? 'A espinha dorsal' : 'A sua lista' }}</h2>
          <p class="sem-margem">{{ doMaterial ? 'A ordem é a do material. Marque os complementos e observações que quer aceitar; os outros ficam de fora.' : 'A ordem é a sua. Marque as sugestões que quer aceitar; as outras ficam de fora.' }}</p>
          <ul v-if="doMaterial && criacao.material.length" class="lista-simples pequeno">
            <li v-for="m in criacao.material" :key="m.tipo + m.referencia + m.localizador">
              <span class="etiqueta">{{ TIPOS_MATERIAL[m.tipo] }}</span>
              <a :href="hrefMaterial(m, trilhaId)" target="_blank" rel="noopener noreferrer">{{ m.titulo }}</a><span v-if="m.localizador" class="fraco"> · {{ m.localizador }}</span>
            </li>
          </ul>
          <ul class="lista-validada">
            <li v-for="(l, i) in listaComSugestoes" :key="i" :class="l.sugestao ? 'sugestao' : 'item'">
              <template v-if="l.item">
                <strong>{{ l.item.titulo }}</strong>
                <span v-if="l.item.fonte" class="fraco"> · {{ l.item.fonte }}</span>
                <span v-if="l.item.nota" class="suave"> · {{ l.item.nota }}</span>
              </template>
              <label v-else class="linha">
                <input type="checkbox" :checked="aceitas.has(l.sugestao.id)" @change="alternarSugestao(l.sugestao.id)" />
                <span><span class="etiqueta azul">{{ ACOES[l.sugestao.acao] }}</span> <strong>{{ l.sugestao.titulo }}</strong><br /><span class="suave">{{ l.sugestao.motivo }}</span></span>
              </label>
            </li>
          </ul>
          <p v-if="!conteudo.sugestoes.length" class="sem-margem">{{ doMaterial ? 'A IA achou o material suficiente para a profundidade pedida.' : 'A IA não viu nada faltando.' }}</p>
        </section>

        <!-- Níveis (mapa e validação) -->
        <div v-if="vista.tipo === 'mapa' || vista.tipo === 'validacao'" class="campos">
          <div>
            <label class="rotulo" for="c-partida">Nível de partida</label>
            <input id="c-partida" v-model="niveis.nivel_partida" class="campo" />
          </div>
          <div>
            <label class="rotulo" for="c-alvo">Nível alvo</label>
            <input id="c-alvo" v-model="niveis.nivel_alvo" class="campo" />
          </div>
          <div>
            <label class="rotulo" for="c-prof">Profundidade</label>
            <select id="c-prof" v-model.number="niveis.profundidade" class="campo">
              <option v-for="(nome, n) in PROFUNDIDADES" :key="n" :value="Number(n)">{{ n }} · {{ nome }}</option>
            </select>
          </div>
        </div>

        <MiniTeste
          v-if="vista.tipo === 'mapa' || vista.tipo === 'validacao'"
          :trilha-id="trilhaId"
          :nivelamento="dados.nivelamento"
          :titulos="titulosDosItens"
          :no-mapa="vista.tipo === 'mapa'"
          @atualizado="dados = $event"
          @usar="usarNivelamento"
        />

        <!-- Plano -->
        <section v-if="vista.tipo === 'plano' && plano" class="pilha">
          <h2>{{ plano.titulo }}</h2>
          <div class="linha etiquetas">
            <span class="etiqueta">{{ etapasDoPlano.length }} etapas</span>
            <span class="etiqueta">{{ duracao(minutosDoPlano) }}</span>
            <span v-if="vista.versao > 1" class="etiqueta">versão {{ vista.versao }}</span>
          </div>
          <dl class="ficha">
            <dt>Objetivo</dt><dd>{{ plano.objetivo }}</dd>
            <template v-if="plano.recorte"><dt>Recorte</dt><dd>{{ plano.recorte }}</dd></template>
            <template v-if="plano.nivel_partida"><dt>Nível de partida</dt><dd>{{ plano.nivel_partida }}</dd></template>
            <template v-if="plano.nivel_alvo"><dt>Nível alvo</dt><dd>{{ plano.nivel_alvo }}</dd></template>
            <template v-if="plano.descricao"><dt>Descrição</dt><dd>{{ plano.descricao }}</dd></template>
          </dl>
          <div v-if="avisosGraves.length" class="aviso">
            <strong>O plano não pode virar trilha assim:</strong>
            <span v-for="a in avisosGraves" :key="a.texto"> {{ a.texto }}.</span>
            Peça um ajuste.
          </div>
          <div v-if="avisosGerais.length || linksRuins.length" class="aviso neutro">
            <div v-for="a in avisosGerais" :key="a.texto"><span class="fraco">{{ a.criterio }}</span> {{ a.texto }}</div>
            <div v-for="l in linksRuins.filter((l) => l.situacao !== 'quebrado')" :key="l.url">
              <a :href="l.url" target="_blank" rel="noopener noreferrer">{{ l.url }}</a> não confirmado (pode bloquear robôs; abra para ver)
            </div>
          </div>
          <div v-for="(m, i) in plano.modulos" :key="i" class="cartao modulo pilha">
            <div class="linha entre">
              <span><strong>{{ i + 1 }}. {{ m.titulo }}</strong><span v-if="m.descricao" class="suave"> · {{ m.descricao }}</span></span>
              <button class="botao discreto pequeno-botao" @click="comentarEm(`modulo:${i + 1}`)">Comentar</button>
            </div>
            <p v-for="a in avisosDe(`modulo:${i + 1}`)" :key="a.texto" class="pequeno aviso-item sem-margem">{{ a.criterio }} · {{ a.texto }}</p>
            <details v-for="e in m.etapas" :key="e.chave" class="etapa-plano">
              <summary>
                <span class="titulo-etapa">{{ e.titulo }}</span>
                <span v-if="e.tipo === 'base'" class="etiqueta">base</span>
                <span class="fraco">{{ e.minutos }} min</span>
                <span v-if="avisosDe(`etapa:${e.chave}`).length" class="etiqueta ambar">{{ avisosDe(`etapa:${e.chave}`).length }}</span>
              </summary>
              <dl class="ficha">
                <dt>Objetivo</dt><dd>{{ e.objetivo }} <span class="fraco">({{ BLOOM[e.bloom] || e.bloom }})</span></dd>
                <template v-if="e.requer?.length"><dt>Antes</dt><dd>{{ e.requer.map(tituloDa).join(' · ') }}</dd></template>
                <template v-if="e.criterio"><dt>Concluir quando</dt><dd>{{ e.criterio }}</dd></template>
                <template v-if="e.materiais?.length">
                  <dt>Material</dt>
                  <dd>
                    <ul class="materiais">
                      <li v-for="mt in e.materiais" :key="mt.referencia + mt.titulo">
                        <span class="fraco">{{ mt.papel }} · </span>
                        <span v-if="mt.tipo === 'livro' || mt.tipo === 'acervo'" class="fraco">{{ TIPOS_MATERIAL[mt.tipo].toLowerCase() }} · </span>
                        <a :href="hrefMaterial(mt, trilhaId)" :target="abreEmOutraAba(mt) ? '_blank' : undefined" rel="noopener noreferrer">{{ mt.titulo }}</a>
                        <span v-if="mt.localizador" class="fraco"> ({{ mt.localizador }})</span>
                        <span v-if="(conteudo.links || []).some((l) => l.url === mt.referencia && l.situacao === 'quebrado')" class="etiqueta vermelho">quebrado</span>
                        <div v-if="mt.nota" class="suave">{{ mt.nota }}</div>
                      </li>
                    </ul>
                  </dd>
                </template>
              </dl>
              <p v-for="a in avisosDe(`etapa:${e.chave}`)" :key="a.texto" class="pequeno aviso-item sem-margem">{{ a.criterio }} · {{ a.texto }}</p>
              <button class="botao discreto pequeno-botao" @click="comentarEm('etapa:' + e.chave)">Comentar esta etapa</button>
            </details>
          </div>
        </section>

        <!-- Comentários, ajuste e decisão (mapa, validação e plano) -->
        <template v-if="vista.tipo !== 'entrevista'">
          <div class="cartao pilha">
            <span class="rotulo">Comentários para o ajuste</span>
            <ul v-if="meus.length" class="comentarios">
              <li v-for="c in meus" :key="c.id">
                <div class="fraco">{{ nomeAncora(c.ancora) }}</div>
                <div class="linha entre"><span>{{ c.texto }}</span><button class="botao discreto" :disabled="ocupado" @click="tirarComentario(c)">Tirar</button></div>
              </li>
            </ul>
            <form class="pilha" @submit.prevent="comentar">
              <select v-model="comentario.ancora" class="campo" aria-label="Onde">
                <option value="">Geral</option>
                <option v-for="a in ancoras" :key="a.id" :value="a.id">{{ a.nome }}</option>
              </select>
              <textarea ref="campoComentario" v-model="comentario.texto" class="campo" rows="2" placeholder="Ex.: falta a parte de custos; esta etapa é grande demais; troque este link." />
              <div><button class="botao" type="submit" :disabled="ocupado || !comentario.texto.trim()">Comentar</button></div>
            </form>
          </div>

          <div class="cartao pilha">
            <label class="rotulo" for="c-pedido-ajuste">Pedido para o ajuste (opcional)</label>
            <textarea id="c-pedido-ajuste" v-model="pedidoAjuste" class="campo" rows="2" placeholder="Ex.: no geral, menos etapas e mais práticas." />
            <template v-if="vista.tipo !== 'plano'">
              <label class="rotulo" for="c-pedido-proximo">Pedido para o plano (opcional, vai junto ao aprovar)</label>
              <textarea id="c-pedido-proximo" v-model="pedidoProximo" class="campo" rows="2" placeholder="Ex.: prefira a documentação oficial; etapas de 30 a 45 minutos." />
            </template>
            <div class="linha entre">
              <div class="linha">
                <button v-if="vista.tipo === 'mapa'" class="botao primario" :disabled="ocupado || !recorte.size" @click="aprovarMapa">Aprovar o recorte e montar o plano</button>
                <button v-else-if="vista.tipo === 'validacao'" class="botao primario" :disabled="ocupado" @click="aprovarValidacao">Aprovar e montar o plano</button>
                <button v-else class="botao primario" :disabled="ocupado || avisosGraves.length > 0" @click="aprovar({})">Aprovar e criar a trilha</button>
                <button class="botao" :disabled="ocupado || (!meus.length && !pedidoAjuste.trim())" @click="ajustar">Pedir ajuste</button>
              </div>
              <button class="botao discreto" :disabled="ocupado" @click="descartar(vista)">Descartar</button>
            </div>
            <p class="pequeno sem-margem">
              Pedir ajuste manda os comentários e o pedido e gera a versão {{ vista.versao + 1 }}, com a resposta da IA a cada comentário.
              <template v-if="vista.tipo === 'plano'"> Aprovar cria as etapas na trilha (ela fica como rascunho até você definir a meta e ativar).</template>
            </p>
          </div>
        </template>
      </template>
    </div>
  </Pagina>
</template>

<style scoped>
.sem-margem { margin: 0; }
.pre { white-space: pre-wrap; }
/* O .cartao p global corta em duas linhas (é para as vitrines); aqui o texto é para ler inteiro. */
.cartao p { display: block; overflow: visible; }
.ficha { display: grid; grid-template-columns: max-content 1fr; gap: var(--e2) var(--e5); margin: 0; font-size: 15px; line-height: 1.6; }
.ficha dt { color: var(--texto-fraco); font-size: 13.5px; padding-top: 2px; }
.ficha dd { margin: 0; }
@media (max-width: 599px) {
  .ficha { grid-template-columns: 1fr; }
  .ficha dd { margin-bottom: var(--e2); }
}
.conteudo h1 { margin: 0; }
.conteudo .lead { margin-top: var(--e2); }
.passos { list-style: none; display: flex; flex-wrap: wrap; gap: var(--e2) var(--e5); margin: 0; padding: 0; font-size: 14px; color: var(--texto-fraco); }
.passos li { display: flex; align-items: center; gap: var(--e2); }
.passos .bolinha { display: inline-grid; place-items: center; width: 22px; height: 22px; border-radius: 50%; border: 1px solid var(--borda-forte); font-size: 12px; }
.passos .atual { color: var(--texto); font-weight: 600; }
.passos .atual .bolinha { border-color: var(--texto); }
.passos .feito { color: var(--texto-suave); }
.passos .feito .bolinha { background: var(--superficie-2); border-color: var(--borda); }
section h2 { margin: var(--e4) 0 0; }
.decidido summary { cursor: pointer; font-weight: 600; }
.decidido[open] summary { margin-bottom: var(--e3); }
.panorama :deep(.leitura) { font-size: 15.5px; }
.pergunta { border: 1px solid var(--borda); margin: 0; }
.pergunta legend { float: left; width: 100%; padding: 0; }
.opcoes { display: flex; flex-wrap: wrap; gap: var(--e2); }
.opcao { height: auto; min-height: 30px; padding: 4px 10px; white-space: normal; text-align: left; }
.opcao.marcada { background: var(--texto); color: var(--fundo); border-color: var(--texto); }
.modulo { padding: var(--e3) var(--e4); }
.cabeca-modulo { cursor: pointer; flex-wrap: nowrap; align-items: flex-start; }
.cabeca-modulo input { margin-top: 4px; }
.etapas-mapa, .lista-validada, .materiais { list-style: none; margin: 0; padding: 0; }
.etapas-mapa li { display: flex; justify-content: space-between; align-items: flex-start; gap: var(--e2); padding: var(--e2) 0; border-top: 1px solid var(--borda); }
.etapas-mapa label, .lista-validada label { flex-wrap: nowrap; align-items: flex-start; cursor: pointer; flex: 1; min-width: 0; }
.etapas-mapa input, .lista-validada input { margin-top: 4px; }
.etapas-mapa .fora .texto-etapa { color: var(--texto-fraco); }
.texto-etapa { font-size: 14.5px; line-height: 1.45; }
.lista-validada li { padding: var(--e2) var(--e3); border-left: 2px solid var(--borda); font-size: 14.5px; }
.lista-validada .sugestao { margin-left: var(--e4); border-left-color: var(--azul); background: var(--superficie); }
.pequeno-botao { height: 26px; padding: 0 8px; font-size: 13px; flex: none; }
.etapa-plano { border-top: 1px solid var(--borda); padding: var(--e2) 0; }
.etapa-plano summary { cursor: pointer; display: flex; flex-wrap: wrap; align-items: center; gap: var(--e2); }
.titulo-etapa { font-weight: 500; }
.lista-simples { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e1); }
.lista-simples li { display: flex; gap: var(--e2); align-items: center; flex-wrap: wrap; }
.etapa-plano .ficha { margin: var(--e2) 0; }
.materiais li { margin-bottom: var(--e1); }
.aviso-item { color: var(--ambar); }
.etiquetas { flex-wrap: wrap; }
.resposta { display: grid; gap: 2px; padding-left: var(--e3); border-left: 2px solid var(--borda-forte); font-size: 14.5px; }
.comentarios { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); font-size: 14.5px; }
</style>
