<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { api } from './api.js'
import { textoComCodigo } from './markdown.js'

// Mini-teste de nível (T9, fase 7): opcional, a partir do mapa ou da lista. A IA escreve de 8 a 12
// perguntas de escolha espalhadas pelos itens; o servidor corrige e sugere onde começar e o nível.
// O Lucas decide: "Usar a sugestão" só preenche o nível e tira do recorte o que ele já domina.
const props = defineProps({
  trilhaId: { type: Number, required: true },
  nivelamento: { type: Object, default: null },
  titulos: { type: Object, default: () => ({}) }, // chave → título do item
  noMapa: { type: Boolean, default: false },
})
const emit = defineEmits(['atualizado', 'usar'])

const n = computed(() => props.nivelamento)
const aberto = ref(false)
const erro = ref('')
const ocupado = ref(false)
const respostas = ref([])
watch(
  () => (n.value?.estado === 'pronto' ? n.value.perguntas.length : 0),
  (total) => (respostas.value = Array(total).fill(null)),
  { immediate: true },
)

let relogio = 0
watch(
  () => n.value?.estado,
  (estado) => {
    clearInterval(relogio)
    if (estado === 'gerando') relogio = setInterval(atualizar, 4000)
  },
  { immediate: true },
)
onBeforeUnmount(() => clearInterval(relogio))
async function atualizar() {
  try {
    emit('atualizado', await api(`/api/trilhas/${props.trilhaId}/criacao`))
  } catch (e) {
    erro.value = e.message
  }
}

async function acao(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    emit('atualizado', await fn())
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}
const pedir = () => acao(() => api(`/api/trilhas/${props.trilhaId}/criacao/nivelamento`, { metodo: 'POST' }))
const faltam = computed(() => respostas.value.filter((r) => r === null).length)
function corrigir() {
  if (faltam.value && !confirm(`Faltam ${faltam.value}; as em branco contam como "não sei". Corrigir assim?`)) return
  acao(() =>
    api(`/api/trilhas/${props.trilhaId}/criacao/nivelamento/respostas`, {
      metodo: 'POST',
      corpo: { respostas: respostas.value.map((r) => (r === null ? -1 : r)) },
    }),
  )
}
const letra = (i) => String.fromCharCode(65 + i)
const r = computed(() => n.value?.resultado)
const titulo = (chave) => props.titulos[chave] || chave
const NIVEL = { basico: 'básico', intermediario: 'intermediário', avancado: 'avançado' }
const porNivel = computed(() =>
  Object.keys(NIVEL)
    .filter((k) => r.value?.por_nivel?.[k])
    .map((k) => `${NIVEL[k]}: ${r.value.por_nivel[k][0]} de ${r.value.por_nivel[k][1]}`)
    .join(' · '),
)
</script>

<template>
  <div class="cartao pilha mini-teste">
    <div class="linha entre">
      <strong>Mini-teste de nível</strong>
      <span class="pequeno">opcional</span>
    </div>
    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <template v-if="!n || n.estado === 'falhou'">
      <p class="suave sem-margem">Não sabe bem onde está? Uns 10 minutos de perguntas de escolha, espalhadas pelo {{ noMapa ? 'mapa' : 'conteúdo' }}, dizem o que você já domina e onde começar. Você decide se usa.</p>
      <p v-if="n?.estado === 'falhou'" class="aviso sem-margem">O mini-teste não saiu: {{ n.erro }}</p>
      <div><button class="botao" :disabled="ocupado" @click="pedir">{{ n ? 'Tentar de novo' : 'Fazer o mini-teste' }}</button></div>
    </template>

    <p v-else-if="n.estado === 'gerando'" class="suave sem-margem">A IA está escrevendo as perguntas… (pode sair desta tela)</p>

    <template v-else-if="n.estado === 'pronto'">
      <button v-if="!aberto" class="botao primario" @click="aberto = true">Responder às {{ n.perguntas.length }} perguntas</button>
      <form v-else class="pilha" @submit.prevent="corrigir">
        <p class="pequeno sem-margem">Sem consultar. Não sabe? Marque "não sei": é melhor que chutar.</p>
        <ol class="questoes">
          <li v-for="(q, i) in n.perguntas" :key="i" class="pilha">
            <span class="pequeno">{{ i + 1 }} de {{ n.perguntas.length }}</span>
            <div class="enunciado" v-html="textoComCodigo(q.enunciado)" />
            <div class="alternativas" role="radiogroup">
              <label v-for="(alt, j) in q.alternativas" :key="j" class="alternativa" :class="{ marcada: respostas[i] === j }">
                <input v-model="respostas[i]" type="radio" :name="`niv-${i}`" :value="j" />
                <span><span class="fraco">{{ letra(j) }})</span> <span v-html="textoComCodigo(alt)" /></span>
              </label>
              <label class="alternativa" :class="{ marcada: respostas[i] === -1 }">
                <input v-model="respostas[i]" type="radio" :name="`niv-${i}`" :value="-1" />
                <span class="fraco">Não sei</span>
              </label>
            </div>
          </li>
        </ol>
        <div class="linha">
          <button class="botao primario" type="submit" :disabled="ocupado">Corrigir</button>
          <span v-if="faltam" class="pequeno">Faltam {{ faltam }}</span>
        </div>
      </form>
    </template>

    <template v-else-if="n.estado === 'corrigido' && r">
      <p class="sem-margem">
        Acertou <strong>{{ r.acertos }} de {{ r.total }}</strong>. Nível sugerido: <strong>{{ r.nivel_sugerido }}</strong>.
        <template v-if="r.comecar_em"> Comece por <strong>{{ titulo(r.comecar_em) }}</strong>.</template>
        <template v-else> Acertou tudo o que foi perguntado.</template>
      </p>
      <p class="pequeno sem-margem">{{ porNivel }}</p>
      <p v-if="r.dominados.length" class="pequeno sem-margem">Já domina: {{ r.dominados.map(titulo).join(', ') }}.</p>
      <div class="linha">
        <button class="botao" @click="emit('usar', r)">{{ noMapa && r.dominados.length ? 'Usar a sugestão (nível e recorte)' : 'Usar o nível sugerido' }}</button>
        <button class="botao discreto" :disabled="ocupado" @click="pedir">Fazer outro</button>
      </div>
      <details class="pequeno">
        <summary>Ver as respostas</summary>
        <ol class="correcoes">
          <li v-for="(c, i) in r.correcoes" :key="i" :class="c.certa ? 'certa' : 'errada'">
            <span v-html="textoComCodigo(c.enunciado)" /> <span class="fraco">({{ titulo(c.conceito) }})</span>
            <div v-if="!c.certa">Você: {{ c.resposta || 'não sei' }} · Certa: <strong>{{ c.correta }}</strong></div>
            <div class="fraco">{{ c.explicacao }}</div>
          </li>
        </ol>
      </details>
      <p class="pequeno sem-margem">O resultado vai junto para o plano: o que você já domina pode ir mais rápido.</p>
    </template>
  </div>
</template>

<style scoped>
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.cartao p { display: block; overflow: visible; }
.questoes { margin: 0; padding-left: 0; list-style: none; display: grid; gap: var(--e4); }
.enunciado { font-size: 15px; line-height: 1.55; }
.alternativas { display: grid; gap: var(--e1); }
.alternativa { display: flex; gap: var(--e2); align-items: flex-start; padding: var(--e2) var(--e3); border: 1px solid var(--borda); border-radius: var(--raio); cursor: pointer; font-size: 14.5px; line-height: 1.5; }
.alternativa input { margin-top: 4px; flex: none; }
.alternativa.marcada { border-color: var(--texto-suave); background: var(--superficie); }
.correcoes { margin: var(--e2) 0 0; padding-left: var(--e5); display: grid; gap: var(--e2); line-height: 1.5; }
.certa::marker { color: var(--verde); }
.errada::marker { color: var(--vermelho); }
</style>
