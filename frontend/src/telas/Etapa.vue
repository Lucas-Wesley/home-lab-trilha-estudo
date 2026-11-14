<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { textoComCodigo } from '../markdown.js'
import { BLOOM, PAPEIS, PROFUNDIDADES, STATUS, TIPOS_MATERIAL, duracao } from '../rotulos.js'

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

const materiaisPorPapel = computed(() => {
  const grupos = { principal: [], apoio: [], aprofundar: [] }
  for (const m of e.value?.materiais || []) (grupos[m.papel] || grupos.apoio).push(m)
  return Object.entries(grupos).filter(([, lista]) => lista.length)
})

function hrefMaterial(m) {
  if (m.tipo === 'web') return m.referencia
  if (m.tipo === 'documento') return `#/documentos/${encodeURIComponent(m.referencia)}${m.localizador ? '#' + m.localizador : ''}`
  return ''
}

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
  const x = e.value
  edicao.value = {
    titulo: x.titulo,
    objetivo: x.objetivo,
    bloom: x.bloom,
    profundidade: x.profundidade,
    tipo: x.tipo,
    minutos: x.minutos,
    criterio: x.criterio,
    requer: [...x.requer],
    materiais: x.materiais.map((m) => ({ ...m })),
    licao: x.licao || '',
  }
}
function novoMaterial() {
  edicao.value.materiais.push({ tipo: 'web', papel: 'apoio', titulo: '', referencia: '', localizador: '', nota: '' })
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
        <div class="cabecalho">
          <h1>{{ e.titulo }}</h1>
          <button class="botao discreto" @click="editar">Editar</button>
        </div>
        <div class="linha etiquetas-topo">
          <span class="etiqueta" :class="STATUS[e.status].cor">{{ STATUS[e.status].nome }}</span>
          <span v-if="e.tipo === 'base'" class="etiqueta ambar">Etapa de base</span>
          <span v-if="e.bloom" class="etiqueta">{{ BLOOM[e.bloom] }}</span>
          <span class="etiqueta">{{ PROFUNDIDADES[profundidade] }}</span>
          <span class="etiqueta">{{ duracao(e.minutos) }}</span>
        </div>
        <p v-if="e.objetivo" class="lead" v-html="textoComCodigo(e.objetivo)" />
        <p v-if="aviso" class="aviso topo">{{ aviso }}</p>

        <div class="linha acoes">
          <button v-if="e.status === 'disponivel'" class="botao" @click="mudarStatus('em_andamento')">Começar</button>
          <button v-if="e.status !== 'concluida'" class="botao primario" @click="mudarStatus('concluida')">Concluir</button>
          <button v-if="e.status === 'disponivel' || e.status === 'em_andamento'" class="botao discreto" @click="mudarStatus('pulada')">Pular</button>
          <button v-if="e.status === 'concluida' || e.status === 'pulada'" class="botao discreto" @click="mudarStatus('disponivel')">Reabrir</button>
        </div>

        <div v-if="e.requer.length || e.criterio" class="cartao cheio pilha bloco">
          <div v-if="e.requer.length">
            <span class="rotulo">Antes desta etapa</span>
            <ul class="lista-simples">
              <li v-for="r in e.requer" :key="r">
                <a :href="link('trilhas', t.id, 'etapas', r)">{{ porId[r]?.titulo }}</a>
                <span class="etiqueta" :class="STATUS[porId[r]?.status]?.cor">{{ STATUS[porId[r]?.status]?.nome }}</span>
              </li>
            </ul>
          </div>
          <div v-if="e.criterio">
            <span class="rotulo">Para concluir</span>
            <p class="sem-margem criterio" v-html="textoComCodigo(e.criterio)" />
          </div>
        </div>

        <section v-if="materiaisPorPapel.length" class="secao">
          <h2>Material</h2>
          <div v-for="[papel, lista] in materiaisPorPapel" :key="papel" class="grupo-material">
            <span class="rotulo">{{ PAPEIS[papel] }}</span>
            <ul class="materiais">
              <li v-for="(m, i) in lista" :key="i">
                <span class="etiqueta">{{ TIPOS_MATERIAL[m.tipo] }}</span>
                <div>
                  <a v-if="hrefMaterial(m)" :href="hrefMaterial(m)" :target="m.tipo === 'web' ? '_blank' : undefined" :rel="m.tipo === 'web' ? 'noopener noreferrer' : undefined">{{ m.titulo }}</a>
                  <span v-else>{{ m.titulo }}</span>
                  <span v-if="m.localizador && m.tipo !== 'documento'" class="fraco"> · {{ m.localizador }}</span>
                  <div v-if="m.nota" class="suave nota">{{ m.nota }}</div>
                </div>
              </li>
            </ul>
          </div>
        </section>

        <section class="secao">
          <h2>Lição</h2>
          <Leitor v-if="e.licao" class="licao-etapa" :texto="e.licao" :ancora="ancora" @titulos="titulos = $event" />
          <div v-else class="cartao">
            <p class="suave sem-margem">Esta etapa ainda não tem lição própria: estude pela leitura principal acima. A lição gerada pela IA, na profundidade da etapa, chega na fase 3; até lá, dá para escrever uma em Editar.</p>
          </div>
        </section>

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
            <input v-model="m.referencia" class="campo" :placeholder="m.tipo === 'web' ? 'https://…' : m.tipo === 'documento' ? 'nome do documento (ex.: 03-redes-separadas-firewall)' : 'Referência'" />
            <input v-model="m.localizador" class="campo" :placeholder="m.tipo === 'documento' ? 'âncora da seção (opcional)' : 'cap., página, tempo (opcional)'" />
            <input v-model="m.nota" class="campo" placeholder="Nota: o que ler e por quê (opcional)" />
            <div class="linha">
              <button type="button" class="botao discreto icone" aria-label="Subir" @click="moverMaterial(i, -1)">↑</button>
              <button type="button" class="botao discreto icone" aria-label="Descer" @click="moverMaterial(i, 1)">↓</button>
              <button type="button" class="botao discreto" @click="edicao.materiais.splice(i, 1)">Remover</button>
            </div>
          </div>
          <div><button type="button" class="botao" @click="novoMaterial">Adicionar material</button></div>
          <p class="ajuda">Streaming: vincular uma pasta de Meus vídeos chega quando o Streaming tiver subpastas e legendas.</p>
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
      <ul v-if="titulos.length && !edicao" class="sumario">
        <li v-for="h in titulos" :key="h.id" :class="{ sub: h.nivel === 3 }">
          <a :href="`#/trilhas/${trilhaId}/etapas/${etapaId}#${h.id}`">{{ h.texto }}</a>
        </li>
      </ul>
    </template>
  </Pagina>
</template>

<style scoped>
.etiquetas-topo { margin-top: var(--e3); }
.topo { margin-top: var(--e4); }
.acoes { margin-top: var(--e5); }
.bloco { margin-top: var(--e5); }
.secao { margin-top: var(--e7); }
.secao > h2 { margin-bottom: var(--e3); }
.sem-margem { margin: 0; }
.lead :deep(code), .criterio :deep(code) { font-size: 0.86em; padding: 1px 5px; background: var(--superficie-2); border: 1px solid var(--borda); border-radius: var(--raio-e); }
.lista-simples { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e1); }
.lista-simples li { display: flex; gap: var(--e2); align-items: center; flex-wrap: wrap; }
.grupo-material + .grupo-material { margin-top: var(--e4); }
.materiais { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.materiais li { display: flex; gap: var(--e3); align-items: flex-start; font-size: 15px; line-height: 1.5; }
.materiais .etiqueta { flex: none; margin-top: 2px; min-width: 76px; justify-content: center; }
.nota { font-size: 14px; margin-top: 2px; }
.vizinhas { display: grid; grid-template-columns: 1fr 1fr; gap: var(--e3); margin-top: var(--e7); }
.vizinhas .cartao { display: grid; gap: 2px; }
.vizinhas .direita { text-align: right; }
.entre { justify-content: space-between; }
.caixa { border: 1px solid var(--borda); border-radius: var(--raio); padding: var(--e3) var(--e4) var(--e4); margin: 0; }
.caixa legend { padding: 0 var(--e1); margin: 0; }
.opcao { display: flex; gap: var(--e2); align-items: flex-start; padding: 3px 0; font-size: 14px; }
.material-edicao { display: grid; gap: var(--e2); padding-bottom: var(--e3); border-bottom: 1px solid var(--borda); }
.licao { min-height: 360px; font-family: var(--fonte-mono); font-size: 13px; }
/* O título da lição repete o da etapa, que já está no topo da página. */
.licao-etapa :deep(h1:first-child) { display: none; }
.perigo { color: var(--vermelho); }
@media (max-width: 599px) {
  .vizinhas { grid-template-columns: 1fr; }
}
</style>
