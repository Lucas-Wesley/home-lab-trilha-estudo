<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import PropostaEstudo from './PropostaEstudo.vue'
import { api } from './api.js'
import { textoComCodigo } from './markdown.js'

// Estudar a etapa (fase 5, T26): exercícios corrigidos na hora, simulado corrigido pela IA,
// o critério automático (nota mínima nos exercícios) e o envio da lição ao Flashcards.
const props = defineProps({
  etapa: { type: Object, required: true },
  trilha: { type: Object, required: true },
  // Na página da etapa, cada parte fica numa aba: mostra só a da vez (vazio = todas).
  aba: { type: String, default: '' },
})
const emit = defineEmits(['concluir', 'mudou', 'contagem'])

const est = ref(null)
watch(est, (x) => x && emit('contagem', { exercicios: x.exercicios.length, simulado: x.simulado.length }))
const ver = (parte) => !props.aba || props.aba === parte
const erro = ref('')
async function carregar() {
  try {
    est.value = await api(`/api/etapas/${props.etapa.id}/estudo`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
onMounted(carregar)

const temLicao = computed(() => !!props.etapa.licao)
const tentativas = computed(() => est.value?.tentativas || [])
const exTentativas = computed(() => tentativas.value.filter((t) => t.tipo === 'exercicios'))
const simTentativas = computed(() => tentativas.value.filter((t) => t.tipo === 'simulado'))
const concluida = computed(() => props.etapa.status === 'concluida')
const letra = (i) => String.fromCharCode(65 + i)
const quando = (iso) => new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
const corNota = (n) => (n >= (est.value?.nota_minima || 80) ? 'verde' : n >= 50 ? 'ambar' : 'vermelho')

// ---------- exercícios ----------
const propEx = ref(null)
const fazendo = ref(false)
const respostasEx = ref([])
const resultadoEx = ref(null)
const enviandoEx = ref(false)
function comecarExercicios() {
  respostasEx.value = est.value.exercicios.map((x) => (x.tipo === 'escolha' ? { escolha: null } : { texto: '' }))
  resultadoEx.value = null
  fazendo.value = true
}
const faltam = computed(() => respostasEx.value.filter((r) => (r.escolha === undefined ? !r.texto?.trim() : r.escolha === null)).length)
async function corrigirExercicios() {
  if (faltam.value && !confirm(`Faltam ${faltam.value} exercício(s); os em branco contam como erro. Corrigir assim?`)) return
  enviandoEx.value = true
  try {
    const corpo = respostasEx.value.map((r) => (r.escolha === undefined ? { texto: r.texto } : r.escolha === null ? {} : { escolha: r.escolha }))
    const t = await api(`/api/etapas/${props.etapa.id}/exercicios/tentativas`, { metodo: 'POST', corpo: { respostas: corpo } })
    resultadoEx.value = t
    fazendo.value = false
    await carregar()
    emit('mudou')
  } catch (e) {
    erro.value = e.message
  } finally {
    enviandoEx.value = false
  }
}
const correcoesEx = computed(() => resultadoEx.value?.resultado || [])
const atingiuAgora = computed(() => resultadoEx.value && resultadoEx.value.nota >= est.value.nota_minima)

// ---------- simulado ----------
const propSim = ref(null)
const respostasSim = ref({})
const enviandoSim = ref(false)
const respondidas = computed(() => Object.values(respostasSim.value).filter((r) => r.trim()).length)
const corrigindo = computed(() => simTentativas.value.find((t) => t.estado === 'corrigindo'))
let relogio = 0
watch(corrigindo, (c) => {
  clearInterval(relogio)
  if (c) relogio = setInterval(carregar, 4000)
})
onBeforeUnmount(() => clearInterval(relogio))
async function mandarSimulado() {
  enviandoSim.value = true
  try {
    const respostas = Object.entries(respostasSim.value)
      .filter(([, r]) => r.trim())
      .map(([i, r]) => ({ pergunta: Number(i), resposta: r }))
    await api(`/api/etapas/${props.etapa.id}/simulado/tentativas`, { metodo: 'POST', corpo: { respostas } })
    respostasSim.value = {}
    await carregar()
    emit('mudou')
  } catch (e) {
    erro.value = e.message
  } finally {
    enviandoSim.value = false
  }
}
const ultimaSim = computed(() => simTentativas.value.find((t) => t.estado !== 'corrigindo'))
const VEREDITO = { certo: { nome: 'certo', cor: 'verde' }, parcial: { nome: 'parcial', cor: 'ambar' }, errado: { nome: 'errado', cor: 'vermelho' } }

// ---------- Flashcards ----------
const fc = ref({ aberto: false, baralhos: null, escolha: '', novo: '', tipo: '', instrucao: '', enviando: false, aviso: '' })
async function abrirFlashcards() {
  fc.value.aberto = true
  fc.value.aviso = ''
  fc.value.novo ||= props.trilha.titulo
  if (fc.value.baralhos) return
  try {
    fc.value.baralhos = await api('/api/flashcards/baralhos')
    const mesmo = fc.value.baralhos.find((b) => b.nome === props.trilha.titulo)
    fc.value.escolha = mesmo ? mesmo.id : ''
  } catch (e) {
    fc.value.aviso = e.message
    fc.value.baralhos = []
  }
}
async function mandarFlashcards() {
  fc.value.enviando = true
  fc.value.aviso = ''
  try {
    const corpo = { tipo: fc.value.tipo, instrucao: fc.value.instrucao }
    if (fc.value.escolha) corpo.baralho_id = fc.value.escolha
    else corpo.baralho = fc.value.novo
    await api(`/api/etapas/${props.etapa.id}/flashcards`, { metodo: 'POST', corpo })
    fc.value.aberto = false
    fc.value.instrucao = ''
    await carregar()
  } catch (e) {
    fc.value.aviso = e.message
  } finally {
    fc.value.enviando = false
  }
}
</script>

<template>
  <div class="estudo">
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <!-- Exercícios -->
    <section v-show="ver('exercicios')" id="exercicios" class="secao">
      <div class="linha entre">
        <h2>Exercícios</h2>
        <button v-if="est?.exercicios.length && !fazendo" class="botao discreto" @click="propEx?.abrir()">Pedir exercícios novos</button>
      </div>
      <p v-if="!temLicao" class="suave">Os exercícios saem da lição: primeiro a lição da etapa.</p>
      <template v-else-if="est">
        <p v-if="!est.exercicios.length" class="suave">
          Perguntas de escolha e de completar, tiradas da lição e corrigidas na hora, sem gastar a assinatura. Com {{ est.nota_minima }}% ou mais, a etapa atinge o critério para concluir.
        </p>
        <PropostaEstudo ref="propEx" :key="`ex-${etapa.id}`" :etapa-id="etapa.id" tipo="exercicios" :tem-atual="!!est.exercicios.length" @aprovada="carregar" />

        <template v-if="est.exercicios.length">
          <!-- Resultado da última vez -->
          <div v-if="resultadoEx" class="pilha bloco">
            <div class="cartao cheio linha entre">
              <div>
                <strong>{{ resultadoEx.acertos }} de {{ resultadoEx.total }}</strong>
                <span class="etiqueta" :class="corNota(resultadoEx.nota)">{{ resultadoEx.nota }}%</span>
              </div>
              <div v-if="atingiuAgora && !concluida" class="linha">
                <span class="pequeno">Passou da nota mínima ({{ est.nota_minima }}%).</span>
                <button class="botao primario" @click="emit('concluir')">Concluir a etapa</button>
              </div>
            </div>
            <ol class="questoes">
              <li v-for="(c, i) in correcoesEx" :key="i" class="cartao pilha" :class="c.certa ? 'certa' : 'errada'">
                <div class="linha entre">
                  <span class="etiqueta" :class="c.certa ? 'verde' : 'vermelho'">{{ c.certa ? 'certo' : 'errado' }}</span>
                  <span class="pequeno">{{ c.conceito }}</span>
                </div>
                <div class="enunciado" v-html="textoComCodigo(c.enunciado)" />
                <div v-if="!c.certa" class="pequeno">Você: {{ c.resposta || '(em branco)' }} · Certa: <strong>{{ c.correta }}</strong></div>
                <div class="explicacao">{{ c.explicacao }}</div>
              </li>
            </ol>
          </div>

          <!-- Fazendo -->
          <form v-if="fazendo" class="pilha bloco" @submit.prevent="corrigirExercicios">
            <ol class="questoes">
              <li v-for="(x, i) in est.exercicios" :key="i" class="cartao pilha">
                <span class="pequeno">{{ i + 1 }} de {{ est.exercicios.length }}</span>
                <div class="enunciado" v-html="textoComCodigo(x.enunciado)" />
                <div v-if="x.tipo === 'escolha'" class="alternativas" role="radiogroup">
                  <label v-for="(alt, j) in x.alternativas" :key="j" class="alternativa" :class="{ marcada: respostasEx[i].escolha === j }">
                    <input v-model="respostasEx[i].escolha" type="radio" :name="`ex-${i}`" :value="j" />
                    <span><span class="fraco">{{ letra(j) }})</span> <span v-html="textoComCodigo(alt)" /></span>
                  </label>
                </div>
                <input v-else v-model="respostasEx[i].texto" class="campo" autocomplete="off" autocapitalize="off" placeholder="Sua resposta" />
              </li>
            </ol>
            <div class="linha">
              <button class="botao primario" type="submit" :disabled="enviandoEx">Corrigir</button>
              <button class="botao discreto" type="button" @click="fazendo = false">Cancelar</button>
              <span v-if="faltam" class="pequeno">Faltam {{ faltam }}</span>
            </div>
          </form>
          <div v-else class="linha bloco">
            <button class="botao" :class="{ primario: !resultadoEx }" @click="comecarExercicios">{{ exTentativas.length ? 'Fazer de novo' : `Fazer os ${est.exercicios.length} exercícios` }}</button>
            <span v-if="exTentativas.length" class="pequeno">
              Últimas:
              <template v-for="(t, i) in exTentativas.slice(0, 5)" :key="t.id"><span v-if="i">, </span>{{ t.nota }}%</template>
              <template v-if="est.criterio_atingido"> · critério atingido</template>
            </span>
          </div>
          <p v-if="est.criterio_atingido && !concluida && !resultadoEx" class="pequeno sem-margem">
            A última vez passou da nota mínima ({{ est.nota_minima }}%). <a href="#" @click.prevent="emit('concluir')">Concluir a etapa</a>
          </p>
        </template>
      </template>
    </section>

    <!-- Simulado -->
    <section v-show="ver('simulado')" id="simulado" class="secao">
      <div class="linha entre">
        <h2>Simulado</h2>
        <button v-if="est?.simulado.length" class="botao discreto" @click="propSim?.abrir()">Gerar mais</button>
      </div>
      <p v-if="!temLicao" class="suave">As perguntas saem da lição: primeiro a lição da etapa.</p>
      <template v-else-if="est">
        <p v-if="!est.simulado.length" class="suave">Perguntas abertas sobre o porquê. Você responde com as suas palavras e a IA corrige, dizendo o que faltou. O banco cresce com "Gerar mais", com o foco que você pedir.</p>
        <PropostaEstudo ref="propSim" :key="`sim-${etapa.id}`" :etapa-id="etapa.id" tipo="simulado" :tem-atual="!!est.simulado.length" @aprovada="carregar" />

        <div v-if="corrigindo" class="cartao linha bloco">
          <strong>Corrigindo {{ corrigindo.total }} resposta(s)…</strong>
          <span class="suave">A IA corrige na fila; pode sair desta tela.</span>
        </div>

        <!-- A última correção -->
        <div v-if="ultimaSim" class="pilha bloco">
          <p v-if="ultimaSim.estado === 'falhou'" class="aviso sem-margem">A correção de {{ quando(ultimaSim.criado_em) }} não saiu: {{ ultimaSim.erro }}. As respostas estão abaixo; dá para mandar de novo.</p>
          <template v-else>
            <div class="cartao cheio pilha">
              <div class="linha">
                <strong>Última correção</strong>
                <span class="etiqueta" :class="corNota(ultimaSim.nota)">{{ ultimaSim.nota }}%</span>
                <span class="pequeno">{{ quando(ultimaSim.criado_em) }}</span>
              </div>
              <p v-if="ultimaSim.resultado?.resumo" class="sem-margem">{{ ultimaSim.resultado.resumo }}</p>
            </div>
            <details v-for="(it, i) in ultimaSim.resultado?.itens || []" :key="i" class="cartao corrigida">
              <summary class="linha">
                <span class="etiqueta" :class="VEREDITO[it.veredito]?.cor">{{ it.nota }}%</span>
                <span v-html="textoComCodigo(it.pergunta)" />
              </summary>
              <div class="pilha detalhe">
                <div><span class="rotulo">Sua resposta</span><div class="pre">{{ it.resposta }}</div></div>
                <div><span class="rotulo">Correção</span><div class="pre">{{ it.comentario }}</div></div>
                <div v-if="it.faltou"><span class="rotulo">Faltou</span><div class="pre">{{ it.faltou }}</div></div>
              </div>
            </details>
          </template>
        </div>

        <form v-if="est.simulado.length && !corrigindo" class="pilha bloco" @submit.prevent="mandarSimulado">
          <p class="pequeno sem-margem">Responda às que quiser (uma já vale); as em branco ficam de fora.</p>
          <ol class="questoes">
            <li v-for="(q, i) in est.simulado" :key="i" class="cartao pilha">
              <span class="pequeno">{{ q.conceito }}</span>
              <label class="enunciado" :for="`sim-${i}`" v-html="textoComCodigo(q.pergunta)" />
              <textarea :id="`sim-${i}`" v-model="respostasSim[i]" class="campo" rows="4" placeholder="Com as suas palavras." />
            </li>
          </ol>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="enviandoSim || !respondidas">Mandar para correção{{ respondidas ? ` (${respondidas})` : '' }}</button>
          </div>
        </form>
      </template>
    </section>

    <!-- Flashcards -->
    <section v-show="ver('flashcards')" id="flashcards" class="secao">
      <h2>Flashcards</h2>
      <p v-if="!temLicao" class="suave">A lição vira cartões no app Flashcards: primeiro a lição da etapa.</p>
      <p v-else-if="est && !est.flashcards.disponivel" class="suave">O Flashcards não está configurado neste servidor (FLASHCARDS_URL).</p>
      <template v-else-if="est">
        <p class="suave">A lição vai ao app Flashcards, que tira os cartões dela e deixa a proposta para você revisar lá. Cada cartão leva o link de volta para a seção da lição.</p>
        <ul v-if="est.flashcards.envios.length" class="envios">
          <li v-for="e in est.flashcards.envios" :key="e.id">
            Mandada em {{ quando(e.criado_em) }}
            <a v-if="e.geracao_id" :href="`${est.flashcards.link}/#/geracao/${e.geracao_id}`" target="_blank" rel="noopener">ver a proposta no Flashcards</a>
            <span v-if="e.erro" class="aviso-texto"> · {{ e.erro }}</span>
          </li>
        </ul>
        <button v-if="!fc.aberto" class="botao" @click="abrirFlashcards">{{ est.flashcards.envios.length ? 'Mandar de novo' : 'Mandar a lição ao Flashcards' }}</button>
        <form v-else class="cartao pilha" @submit.prevent="mandarFlashcards">
          <div>
            <label class="rotulo" for="fc-baralho">Baralho</label>
            <select id="fc-baralho" v-model="fc.escolha" class="campo">
              <option value="">Um baralho novo…</option>
              <option v-for="b in fc.baralhos || []" :key="b.id" :value="b.id">{{ b.nome }}</option>
            </select>
          </div>
          <div v-if="!fc.escolha">
            <label class="rotulo" for="fc-novo">Nome do baralho novo</label>
            <input id="fc-novo" v-model="fc.novo" class="campo" />
          </div>
          <fieldset class="sem-borda">
            <legend class="rotulo">Cartões</legend>
            <label class="opcao"><input v-model="fc.tipo" type="radio" value="" /> Pergunta e resposta curta</label>
            <label class="opcao"><input v-model="fc.tipo" type="radio" value="leitura" /> De leitura (um resumo para reler)</label>
          </fieldset>
          <div>
            <label class="rotulo" for="fc-instr">Foco (opcional)</label>
            <input id="fc-instr" v-model="fc.instrucao" class="campo" placeholder="Ex.: só os números e as faixas de endereço." />
          </div>
          <p v-if="fc.aviso" class="aviso sem-margem">{{ fc.aviso }}</p>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="fc.enviando || (!fc.escolha && !fc.novo.trim())">Mandar</button>
            <button class="botao discreto" type="button" @click="fc.aberto = false">Cancelar</button>
          </div>
        </form>
      </template>
    </section>
  </div>
</template>

<style scoped>
.secao { margin-top: var(--e7); }
.secao h2 { margin-bottom: var(--e3); }
.bloco { margin-top: var(--e4); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; }
.pre { white-space: pre-wrap; }
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
.corrigida summary { cursor: pointer; align-items: flex-start; flex-wrap: nowrap; font-size: 15px; }
.corrigida summary .etiqueta { flex: none; }
.detalhe { margin-top: var(--e3); font-size: 14.5px; }
.envios { list-style: none; margin: 0 0 var(--e3); padding: 0; display: grid; gap: var(--e1); font-size: 14px; }
.aviso-texto { color: var(--vermelho); }
.sem-borda { border: 0; padding: 0; margin: 0; display: grid; gap: var(--e1); }
.opcao { display: flex; gap: var(--e2); align-items: center; font-size: 14.5px; }
</style>
