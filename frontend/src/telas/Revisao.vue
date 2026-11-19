<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { textoComCodigo } from '../markdown.js'
import { link } from '../rotas.js'
import { dataCurta } from '../rotulos.js'

// Revisão espaçada de uma etapa concluída (fase 7, T28): até 5 exercícios dela (os que ele errou
// antes primeiro), corrigidos na hora. Sem exercícios, ele tenta lembrar das partes da lição e diz
// como foi. Acertou, o intervalo cresce; foi mais ou menos, repete; errou, recomeça.
const props = defineProps({
  trilhaId: { type: Number, required: true },
  etapaId: { type: Number, required: true },
})

const etapa = ref(null)
const dados = ref(null)
const erro = ref('')
const respostas = ref([])
const enviando = ref(false)
const feito = ref(null)

async function carregar() {
  try {
    ;[etapa.value, dados.value] = await Promise.all([api(`/api/etapas/${props.etapaId}`), api(`/api/etapas/${props.etapaId}/revisao`)])
    respostas.value = dados.value.perguntas.map((x) => (x.tipo === 'escolha' ? { escolha: null } : { texto: '' }))
  } catch (e) {
    erro.value = e.status === 404 ? 'Etapa não encontrada.' : e.message
  }
}
onMounted(carregar)

const letra = (i) => String.fromCharCode(65 + i)
const faltam = computed(() => respostas.value.filter((r) => (r.escolha === undefined ? !r.texto?.trim() : r.escolha === null)).length)

async function mandar(corpo) {
  enviando.value = true
  erro.value = ''
  try {
    feito.value = await api(`/api/etapas/${props.etapaId}/revisao`, { metodo: 'POST', corpo })
    window.scrollTo({ top: 0 })
  } catch (e) {
    erro.value = e.message
  } finally {
    enviando.value = false
  }
}
function corrigir() {
  if (faltam.value && !confirm(`Faltam ${faltam.value}; as em branco contam como erro. Corrigir assim?`)) return
  mandar({
    respostas: dados.value.perguntas.map((x, i) => {
      const r = respostas.value[i]
      if (r.escolha === undefined) return { indice: x.indice, texto: r.texto }
      return r.escolha === null ? { indice: x.indice } : { indice: x.indice, escolha: r.escolha }
    }),
  })
}
const EFEITO = {
  subiu: 'Bom: o intervalo cresceu.',
  manteve: 'Quase: o intervalo continua o mesmo.',
  voltou: 'Esqueceu boa parte: a revisão recomeça amanhã. Vale reler a lição (e, se for o caso, pedir um reforço na página da etapa).',
}
const correcoes = computed(() => (Array.isArray(feito.value?.tentativa?.resultado) ? feito.value.tentativa.resultado : []))
</script>

<template>
  <Pagina>
    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-if="etapa && dados">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('hoje')">Hoje</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <a :href="link('trilhas', trilhaId, 'etapas', etapaId)">{{ etapa.titulo }}</a>
      </nav>
      <div class="topo-pagina">
        <div>
          <h1>Revisão</h1>
          <p class="contexto">{{ etapa.titulo }}</p>
        </div>
      </div>

      <p v-if="!dados.concluida" class="aviso neutro">A revisão é das etapas concluídas. Conclua esta na página dela.</p>

      <!-- Resultado -->
      <div v-else-if="feito" class="pilha">
        <div class="cartao cheio pilha">
          <div class="linha">
            <strong v-if="correcoes.length">{{ feito.tentativa.acertos }} de {{ feito.tentativa.total }}</strong>
            <span class="etiqueta" :class="{ subiu: 'verde', manteve: 'ambar', voltou: 'vermelho' }[feito.efeito]">{{ feito.tentativa.nota }}%</span>
          </div>
          <p class="sem-margem">{{ EFEITO[feito.efeito] }} Próxima revisão: <strong>{{ dataCurta(feito.revisao.proxima) }}</strong> ({{ feito.revisao.intervalo }} dia(s)).</p>
        </div>
        <ol v-if="correcoes.length" class="questoes">
          <li v-for="(c, i) in correcoes" :key="i" class="cartao pilha" :class="c.certa ? 'certa' : 'errada'">
            <div class="linha entre">
              <span class="etiqueta" :class="c.certa ? 'verde' : 'vermelho'">{{ c.certa ? 'certo' : 'errado' }}</span>
              <span class="pequeno">{{ c.conceito }}</span>
            </div>
            <div class="enunciado" v-html="textoComCodigo(c.enunciado)" />
            <div v-if="!c.certa" class="pequeno">Você: {{ c.resposta || '(em branco)' }} · Certa: <strong>{{ c.correta }}</strong></div>
            <div class="explicacao">{{ c.explicacao }}</div>
          </li>
        </ol>
        <div class="linha">
          <a class="botao primario" :href="link('hoje')">Voltar ao Hoje</a>
          <a class="botao" :href="link('trilhas', trilhaId, 'etapas', etapaId)">Abrir a etapa</a>
        </div>
      </div>

      <!-- Perguntas rápidas -->
      <form v-else-if="dados.modo === 'exercicios'" class="pilha" @submit.prevent="corrigir">
        <p class="suave sem-margem">{{ dados.perguntas.length }} pergunta(s) rápida(s) dos exercícios da etapa, sem olhar a lição. Os que você errou antes vêm primeiro.</p>
        <ol class="questoes">
          <li v-for="(x, i) in dados.perguntas" :key="x.indice" class="cartao pilha">
            <span class="pequeno">{{ i + 1 }} de {{ dados.perguntas.length }}</span>
            <div class="enunciado" v-html="textoComCodigo(x.enunciado)" />
            <div v-if="x.tipo === 'escolha'" class="alternativas" role="radiogroup">
              <label v-for="(alt, j) in x.alternativas" :key="j" class="alternativa" :class="{ marcada: respostas[i].escolha === j }">
                <input v-model="respostas[i].escolha" type="radio" :name="`rev-${i}`" :value="j" />
                <span><span class="fraco">{{ letra(j) }})</span> <span v-html="textoComCodigo(alt)" /></span>
              </label>
            </div>
            <input v-else v-model="respostas[i].texto" class="campo" autocomplete="off" autocapitalize="off" placeholder="Sua resposta" />
          </li>
        </ol>
        <div class="linha">
          <button class="botao primario" type="submit" :disabled="enviando">Corrigir</button>
          <span v-if="faltam" class="pequeno">Faltam {{ faltam }}</span>
        </div>
      </form>

      <!-- Sem exercícios: lembrar das partes -->
      <div v-else class="pilha">
        <p class="suave sem-margem">A etapa não tem exercícios. Sem abrir a lição, tente explicar para você mesmo cada parte dela:</p>
        <ol v-if="dados.secoes.length" class="cartao partes">
          <li v-for="s in dados.secoes" :key="s">{{ s }}</li>
        </ol>
        <p v-else class="pequeno sem-margem">(A lição não tem seções; tente lembrar do objetivo da etapa.)</p>
        <p class="pequeno sem-margem">Depois confira na <a :href="link('trilhas', trilhaId, 'etapas', etapaId)">lição</a> e diga como foi:</p>
        <div class="linha">
          <button class="botao primario" :disabled="enviando" @click="mandar({ autoavaliacao: 'bem' })">Lembrei bem</button>
          <button class="botao" :disabled="enviando" @click="mandar({ autoavaliacao: 'parcial' })">Mais ou menos</button>
          <button class="botao" :disabled="enviando" @click="mandar({ autoavaliacao: 'esqueci' })">Esqueci</button>
        </div>
      </div>
    </template>
  </Pagina>
</template>

<style scoped>
.sem-margem { margin: 0; }
.contexto { margin: var(--e2) 0 0; font-size: 14px; color: var(--texto-fraco); }
.entre { justify-content: space-between; }
.cartao p { display: block; overflow: visible; }
.questoes { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.enunciado { font-size: 15px; line-height: 1.55; display: block; }
.enunciado :deep(code), .alternativa :deep(code) { font-size: 0.86em; padding: 1px 5px; background: var(--superficie-2); border: 1px solid var(--borda); border-radius: var(--raio-e); }
.alternativas { display: grid; gap: var(--e1); }
.alternativa { display: flex; gap: var(--e2); align-items: flex-start; padding: var(--e2) var(--e3); border: 1px solid var(--borda); border-radius: var(--raio); cursor: pointer; font-size: 14.5px; line-height: 1.5; }
.alternativa input { margin-top: 4px; flex: none; }
.alternativa.marcada { border-color: var(--texto-suave); background: var(--superficie); }
.certa { border-left: 3px solid var(--verde); }
.errada { border-left: 3px solid var(--vermelho); }
.explicacao { font-size: 14px; color: var(--texto-suave); white-space: pre-wrap; }
.partes { margin: 0; padding: var(--e3) var(--e3) var(--e3) var(--e7); line-height: 1.8; }
</style>
