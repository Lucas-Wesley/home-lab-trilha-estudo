<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import DialogoMeta from '../DialogoMeta.vue'
import DocumentosTrilha from '../DocumentosTrilha.vue'
import FormTrilha from '../FormTrilha.vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { BLOOM, DIAS, ESTADOS, PROFUNDIDADES, STATUS, dataCurta, duracao, percentual } from '../rotulos.js'

const props = defineProps({
  trilhaId: { type: Number, required: true },
  ancora: { type: String, default: '' },
})

const t = ref(null)
const cronograma = ref([])
const erro = ref('')
const aviso = ref('')
// Trilha em criação com a IA (fase 4): a página leva de volta ao passo em que ela está.
const passoCriacao = ref('')

async function carregar() {
  try {
    api(`/api/trilhas/${props.trilhaId}/criacao`).then(
      (r) => (passoCriacao.value = r.criacao.passo),
      () => (passoCriacao.value = ''),
    )
    t.value = await api(`/api/trilhas/${props.trilhaId}`)
    t.value.modulos ||= [] // trilha sem módulo: a API omite a lista
    cronograma.value = t.value.meta.dias.length ? await api(`/api/trilhas/${props.trilhaId}/cronograma`) : []
  } catch (e) {
    erro.value = e.status === 404 ? 'Trilha não encontrada.' : e.message
  }
}
// Chegou com âncora (#/trilhas/3#documentos): vai até a seção depois de carregar.
onMounted(async () => {
  await carregar()
  if (props.ancora) {
    await nextTick()
    document.getElementById(props.ancora)?.scrollIntoView({ block: 'start' })
  }
})

// Toda mudança devolve a trilha atualizada (ou não), e o erro aparece no topo.
async function fazer(acao) {
  aviso.value = ''
  try {
    const r = await acao()
    if (r && r.modulos) t.value = r
    else await carregar()
    return true
  } catch (e) {
    aviso.value = e.message
    return false
  }
}

const hojeIso = (() => {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
})()

const etapas = computed(() => (t.value ? t.value.modulos.flatMap((m) => m.etapas) : []))
const porId = computed(() => Object.fromEntries(etapas.value.map((e) => [e.id, e])))
const restante = computed(() => etapas.value.filter((e) => e.status !== 'concluida' && e.status !== 'pulada').reduce((s, e) => s + (e.minutos || 45), 0))
const previsao = computed(() => cronograma.value.at(-1)?.data || '')
const atrasadas = computed(() => {
  const ultimo = {}
  for (const d of cronograma.value) for (const it of d.itens) ultimo[it.etapa_id] = d.data
  return Object.entries(ultimo).filter(([id, data]) => data < hojeIso && !['concluida', 'pulada'].includes(porId.value[id]?.status)).length
})
const metaTexto = computed(() => {
  const m = t.value?.meta
  if (!m || !m.dias.length) return ''
  const dias = m.dias.map((d) => DIAS[d]).join(', ')
  return m.minutos ? `${dias} · ${m.minutos} min por dia` : `${dias} · até ${dataCurta(m.fim)}`
})

// ---------- dados da trilha ----------
const editando = ref(null)
function abrirEdicao() {
  const { titulo, objetivo, recorte, nivel_partida, nivel_alvo, profundidade, instrucoes, descricao } = t.value
  editando.value = { titulo, objetivo, recorte, nivel_partida, nivel_alvo, profundidade, instrucoes, descricao }
}
async function salvarDados() {
  if (await fazer(() => api(`/api/trilhas/${props.trilhaId}`, { metodo: 'PUT', corpo: editando.value }))) editando.value = null
}
async function excluirTrilha() {
  if (!confirm(`Excluir a trilha "${t.value.titulo}" com todas as etapas e lições? O histórico fica no git do conteúdo.`)) return
  try {
    await api(`/api/trilhas/${props.trilhaId}`, { metodo: 'DELETE' })
    location.hash = link('trilhas')
  } catch (e) {
    aviso.value = e.message
  }
}
const mudarEstado = (estado) => fazer(() => api(`/api/trilhas/${props.trilhaId}/estado`, { metodo: 'POST', corpo: { estado } }))

// ---------- meta ----------
const dialogoMeta = ref(false)
const erroMeta = ref('')
async function salvarMeta(meta) {
  erroMeta.value = ''
  try {
    t.value = await api(`/api/trilhas/${props.trilhaId}/meta`, { metodo: 'PUT', corpo: meta })
    cronograma.value = await api(`/api/trilhas/${props.trilhaId}/cronograma`)
    dialogoMeta.value = false
  } catch (e) {
    erroMeta.value = e.message
  }
}
async function recalcular() {
  aviso.value = ''
  try {
    cronograma.value = await api(`/api/trilhas/${props.trilhaId}/recalcular`, { metodo: 'POST' })
  } catch (e) {
    aviso.value = e.message
  }
}
const verTudo = ref(false)
const diasVisiveis = computed(() => (verTudo.value ? cronograma.value : cronograma.value.filter((d) => d.data >= hojeIso).slice(0, 7)))

// ---------- estrutura ----------
const organizar = ref(false)
const novoModulo = ref('')
const novaEtapa = ref({})

const adicionarModulo = async () => {
  if (!novoModulo.value.trim()) return
  if (await fazer(() => api(`/api/trilhas/${props.trilhaId}/modulos`, { metodo: 'POST', corpo: { titulo: novoModulo.value } }))) novoModulo.value = ''
}
async function adicionarEtapa(m) {
  const titulo = (novaEtapa.value[m.id] || '').trim()
  if (!titulo) return
  if (await fazer(() => api(`/api/modulos/${m.id}/etapas`, { metodo: 'POST', corpo: { titulo } }))) novaEtapa.value[m.id] = ''
}
async function renomearModulo(m) {
  const titulo = prompt('Título do módulo', m.titulo)
  if (!titulo || titulo === m.titulo) return
  await fazer(() => api(`/api/modulos/${m.id}`, { metodo: 'PUT', corpo: { titulo, descricao: m.descricao } }))
}
async function excluirModulo(m) {
  if (!confirm(`Excluir o módulo "${m.titulo}"?`)) return
  await fazer(() => api(`/api/modulos/${m.id}`, { metodo: 'DELETE' }))
}

// Mover = mandar a ordem inteira de novo (o servidor confere que nada sumiu).
function ordemAtual() {
  return t.value.modulos.map((m) => ({ id: m.id, etapas: m.etapas.map((e) => e.id) }))
}
function enviarOrdem(ordem) {
  return fazer(() => api(`/api/trilhas/${props.trilhaId}/ordem`, { metodo: 'PUT', corpo: { modulos: ordem } }))
}
function moverModulo(i, delta) {
  const ordem = ordemAtual()
  const j = i + delta
  if (j < 0 || j >= ordem.length) return
  ;[ordem[i], ordem[j]] = [ordem[j], ordem[i]]
  enviarOrdem(ordem)
}
// Uma etapa sobe/desce dentro do módulo; na ponta, passa para o módulo vizinho.
function moverEtapa(mi, ei, delta) {
  const ordem = ordemAtual()
  const lista = ordem[mi].etapas
  const [id] = lista.splice(ei, 1)
  const destino = ei + delta
  if (destino < 0) {
    if (mi === 0) return
    ordem[mi - 1].etapas.push(id)
  } else if (destino > lista.length) {
    if (mi === ordem.length - 1) return
    ordem[mi + 1].etapas.unshift(id)
  } else {
    lista.splice(destino, 0, id)
  }
  enviarOrdem(ordem)
}

// A próxima etapa a estudar: a primeira, na ordem, que não foi concluída nem pulada. (Marcar
// "requisito pendente" em todas as outras era ruído: no começo, todas têm.)
const proximaId = computed(() => etapas.value.find((e) => e.status !== 'concluida' && e.status !== 'pulada')?.id)

const secoes = [
  { id: 'sobre', nome: 'Sobre a trilha' },
  { id: 'meta', nome: 'Meta e ritmo' },
  { id: 'estrutura', nome: 'Estrutura' },
  { id: 'documentos', nome: 'Documentos' },
  { id: 'cronograma', nome: 'Cronograma' },
]
function irPara(id) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas v-if="t" :trilha="t" /></template>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-else-if="t">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('trilhas')">Trilhas</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <span>{{ t.titulo }}</span>
      </nav>
      <div class="cabecalho">
        <h1>{{ t.titulo }}</h1>
        <div class="linha">
          <a v-if="t.progresso.concluidas || t.progresso.em_andamento" class="botao" :href="link('trilhas', t.id, 'painel')">Painel</a>
          <button class="botao discreto" @click="abrirEdicao">Editar</button>
        </div>
      </div>
      <div class="linha etiquetas-topo">
        <span class="etiqueta" :class="ESTADOS[t.estado].cor">{{ ESTADOS[t.estado].nome }}</span>
        <span class="etiqueta">Profundidade {{ t.profundidade }} · {{ PROFUNDIDADES[t.profundidade] }}</span>
        <span class="etiqueta">{{ t.progresso.total }} etapas · {{ duracao(t.progresso.minutos_total) }}</span>
      </div>
      <p v-if="t.objetivo" class="lead">{{ t.objetivo }}</p>
      <p v-if="aviso" class="aviso topo">{{ aviso }}</p>
      <p v-if="passoCriacao && passoCriacao !== 'pronta'" class="aviso neutro topo">
        Esta trilha está sendo criada com a IA. <a :href="link('trilhas', t.id, 'criar')">Continuar a criação</a>
      </p>

      <!-- Sobre -->
      <section id="sobre" class="secao">
        <h2>Sobre a trilha</h2>
        <dl class="ficha">
          <template v-if="t.recorte"><dt>Recorte</dt><dd>{{ t.recorte }}</dd></template>
          <template v-if="t.nivel_partida"><dt>Nível de partida</dt><dd>{{ t.nivel_partida }}</dd></template>
          <template v-if="t.nivel_alvo"><dt>Nível alvo</dt><dd>{{ t.nivel_alvo }}</dd></template>
          <template v-if="t.descricao"><dt>Descrição</dt><dd class="pre">{{ t.descricao }}</dd></template>
          <template v-if="t.instrucoes"><dt>Instruções da trilha</dt><dd class="pre">{{ t.instrucoes }}</dd></template>
        </dl>
        <p v-if="!t.recorte && !t.nivel_partida && !t.instrucoes" class="suave">Sem detalhes ainda. <a href="" @click.prevent="abrirEdicao">Completar</a></p>
      </section>

      <!-- Meta -->
      <section id="meta" class="secao">
        <h2>Meta e ritmo</h2>
        <div v-if="metaTexto" class="cartao cheio pilha">
          <div class="linha entre">
            <div>
              <strong>{{ metaTexto }}</strong>
              <div class="suave">
                Desde {{ dataCurta(t.meta.inicio) }}<template v-if="previsao"> · previsão de término {{ dataCurta(previsao) }}</template>
              </div>
            </div>
            <div class="linha">
              <button class="botao" @click="dialogoMeta = true">Mudar meta</button>
              <button class="botao" @click="recalcular">Recalcular</button>
            </div>
          </div>
          <p v-if="atrasadas" class="aviso neutro">{{ atrasadas }} etapa(s) passaram do dia previsto. "Recalcular" redistribui o que falta a partir de hoje.</p>
          <div class="linha">
            <button v-if="t.estado === 'ativa'" class="botao discreto" @click="mudarEstado('pausada')">Pausar a trilha</button>
            <button v-else-if="t.estado === 'pausada'" class="botao discreto" @click="mudarEstado('ativa')">Retomar</button>
            <button v-if="t.estado !== 'concluida'" class="botao discreto" @click="mudarEstado('concluida')">Marcar a trilha como concluída</button>
            <button v-else class="botao discreto" @click="mudarEstado('ativa')">Reabrir</button>
          </div>
        </div>
        <div v-else class="cartao pilha">
          <p class="suave sem-margem">Sem meta ainda. Escolha os dias e o tempo (ou a data final); o app distribui as etapas e a trilha aparece em <a :href="link('hoje')">Hoje</a>.</p>
          <div><button class="botao primario" :disabled="!t.progresso.total" @click="dialogoMeta = true">Definir meta</button></div>
        </div>
      </section>

      <!-- Estrutura -->
      <section id="estrutura" class="secao">
        <div class="linha entre">
          <h2>Estrutura</h2>
          <button class="botao" :class="{ primario: organizar }" @click="organizar = !organizar">{{ organizar ? 'Pronto' : 'Organizar' }}</button>
        </div>

        <div v-for="(m, mi) in t.modulos" :key="m.id" class="modulo">
          <div class="linha entre modulo-topo">
            <h3>{{ mi + 1 }}. {{ m.titulo }}</h3>
            <div v-if="organizar" class="linha">
              <button class="botao discreto icone" aria-label="Subir módulo" :disabled="mi === 0" @click="moverModulo(mi, -1)">↑</button>
              <button class="botao discreto icone" aria-label="Descer módulo" :disabled="mi === t.modulos.length - 1" @click="moverModulo(mi, 1)">↓</button>
              <button class="botao discreto" @click="renomearModulo(m)">Renomear</button>
              <button v-if="!m.etapas.length" class="botao discreto" @click="excluirModulo(m)">Excluir</button>
            </div>
          </div>
          <p v-if="m.descricao" class="suave descricao-modulo">{{ m.descricao }}</p>
          <table class="tabela">
            <tbody>
              <tr v-for="(e, ei) in m.etapas" :key="e.id">
                <td class="col-titulo">
                  <a class="titulo" :href="link('trilhas', t.id, 'etapas', e.id)">{{ e.titulo }}</a>
                  <div v-if="e.objetivo" class="descricao">{{ e.objetivo }}</div>
                </td>
                <td>
                  <div class="etiquetas">
                    <span class="etiqueta" :class="STATUS[e.status].cor">{{ STATUS[e.status].nome }}</span>
                    <span v-if="e.tipo === 'base'" class="etiqueta ambar">Etapa de base</span>
                    <span v-if="e.id === proximaId" class="etiqueta azul">Próxima</span>
                  </div>
                </td>
                <td class="direita">
                  <div v-if="!organizar" class="fraco">{{ duracao(e.minutos) }}</div>
                  <div v-if="!organizar && e.bloom" class="fraco">{{ BLOOM[e.bloom] }}</div>
                  <div v-if="organizar" class="linha fim">
                    <button class="botao discreto icone" aria-label="Subir etapa" :disabled="mi === 0 && ei === 0" @click="moverEtapa(mi, ei, -1)">↑</button>
                    <button class="botao discreto icone" aria-label="Descer etapa" :disabled="mi === t.modulos.length - 1 && ei === m.etapas.length - 1" @click="moverEtapa(mi, ei, 1)">↓</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
          <form v-if="organizar" class="linha nova" @submit.prevent="adicionarEtapa(m)">
            <input v-model="novaEtapa[m.id]" class="campo" placeholder="Nova etapa neste módulo" />
            <button class="botao" type="submit">Adicionar</button>
          </form>
        </div>

        <p v-if="!t.modulos.length" class="suave">Comece criando um módulo (um agrupamento de etapas, como "Fundamentos").</p>
        <form v-if="organizar || !t.modulos.length" class="linha nova" @submit.prevent="adicionarModulo">
          <input v-model="novoModulo" class="campo" placeholder="Novo módulo" />
          <button class="botao" type="submit">Criar módulo</button>
        </form>
      </section>

      <!-- Documentos -->
      <section id="documentos" class="secao">
        <h2>Documentos</h2>
        <DocumentosTrilha :trilha="t" @mudou="carregar" />
      </section>

      <!-- Cronograma -->
      <section id="cronograma" class="secao">
        <div class="linha entre">
          <h2>Cronograma</h2>
          <button v-if="cronograma.length > 7" class="botao discreto" @click="verTudo = !verTudo">{{ verTudo ? 'Só os próximos dias' : 'Ver tudo' }}</button>
        </div>
        <p v-if="!cronograma.length" class="suave">O cronograma aparece quando houver uma meta.</p>
        <div v-for="d in diasVisiveis" :key="d.data" class="dia" :class="{ hoje: d.data === hojeIso, passado: d.data < hojeIso }">
          <div class="dia-data">{{ d.data === hojeIso ? 'Hoje' : dataCurta(d.data) }}</div>
          <ul>
            <li v-for="it in d.itens" :key="it.etapa_id">
              <a :href="link('trilhas', t.id, 'etapas', it.etapa_id)" :class="{ feita: it.status === 'concluida' || it.status === 'pulada' }">{{ it.titulo }}</a>
              <span class="fraco"> · {{ duracao(it.minutos) }}<template v-if="it.parte"> (parte {{ it.parte }})</template></span>
            </li>
          </ul>
        </div>
      </section>
    </template>

    <template #lateral>
      <div v-if="t" class="resumo pilha">
        <div class="linha entre"><strong>Progresso</strong><span class="fraco">{{ percentual(t.progresso) }}%</span></div>
        <div class="progresso"><span :style="{ width: percentual(t.progresso) + '%' }" /></div>
        <div class="suave">{{ t.progresso.concluidas }} concluídas · {{ t.progresso.puladas }} puladas · {{ t.progresso.total }} no total</div>
        <div class="suave">Falta {{ duracao(restante) }}</div>
        <div v-if="previsao" class="suave">Previsão: {{ dataCurta(previsao) }}</div>
      </div>
      <ul v-if="t" class="sumario">
        <li v-for="s in secoes" :key="s.id"><a href="" @click.prevent="irPara(s.id)">{{ s.nome }}</a></li>
      </ul>
    </template>
  </Pagina>

  <div v-if="editando" class="dialogo-fundo" @click.self="editando = null">
    <form class="painel pilha largo" role="dialog" aria-label="Editar a trilha" @submit.prevent="salvarDados">
      <h3>Editar a trilha</h3>
      <FormTrilha v-model="editando" />
      <div>
        <label class="rotulo" for="t-desc">Descrição</label>
        <textarea id="t-desc" v-model="editando.descricao" class="campo" rows="3" />
      </div>
      <p v-if="aviso" class="aviso">{{ aviso }}</p>
      <div class="linha entre">
        <div class="linha">
          <button class="botao primario" type="submit">Salvar</button>
          <button class="botao discreto" type="button" @click="editando = null">Cancelar</button>
        </div>
        <button class="botao discreto perigo" type="button" @click="excluirTrilha">Excluir a trilha</button>
      </div>
    </form>
  </div>

  <DialogoMeta v-if="dialogoMeta && t" :meta="t.meta" :restante="restante" :erro="erroMeta" @salvar="salvarMeta" @fechar="dialogoMeta = false" />
</template>

<style scoped>
.etiquetas-topo { margin-top: var(--e3); }
.topo { margin-top: var(--e4); }
.secao { margin-top: var(--e7); scroll-margin-top: calc(var(--barra) + var(--e4)); }
.secao > h2, .secao > .linha > h2 { margin-bottom: var(--e3); }
.entre { justify-content: space-between; }
.fim { justify-content: flex-end; }
.sem-margem { margin: 0; }

.ficha { display: grid; grid-template-columns: max-content 1fr; gap: var(--e2) var(--e5); margin: 0; font-size: 15px; line-height: 1.6; }
.ficha dt { color: var(--texto-fraco); font-size: 13.5px; padding-top: 2px; }
.ficha dd { margin: 0; }
.pre { white-space: pre-wrap; }
@media (max-width: 599px) {
  .ficha { grid-template-columns: 1fr; }
  .ficha dd { margin-bottom: var(--e2); }
}

.modulo { margin-top: var(--e5); }
.modulo-topo h3 { font-size: 16px; }
.descricao-modulo { margin: var(--e1) 0 0; }
.col-titulo { width: 60%; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
.direita { text-align: right; white-space: nowrap; }
.nova { margin-top: var(--e3); flex-wrap: nowrap; }
.nova .campo { flex: 1; }

.dia { display: grid; grid-template-columns: 120px 1fr; gap: var(--e3); padding: var(--e2) 0; border-bottom: 1px solid var(--borda); }
.dia ul { list-style: none; margin: 0; padding: 0; display: grid; gap: 2px; }
.dia a { color: var(--texto); text-decoration: none; }
.dia a:hover { text-decoration: underline; }
.dia a.feita { color: var(--texto-fraco); text-decoration: line-through; }
.dia-data { color: var(--texto-suave); font-size: 13.5px; padding-top: 1px; }
.dia.hoje .dia-data { color: var(--texto); font-weight: 600; }
.dia.passado { opacity: 0.6; }

.resumo { padding: var(--e4); margin-bottom: var(--e5); border: 1px solid var(--borda); border-radius: var(--raio); gap: var(--e2); font-size: 13.5px; }
.painel.largo { max-width: 640px; }
.perigo { color: var(--vermelho); }
</style>
