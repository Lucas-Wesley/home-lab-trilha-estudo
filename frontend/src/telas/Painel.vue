<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { textoComCodigo } from '../markdown.js'
import { link } from '../rotas.js'
import { STATUS, dataCurta, percentual } from '../rotulos.js'

// Painel da trilha (fase 7, T28): progresso, notas por etapa, revisões, pontos fracos e o simulado
// intercalado (perguntas de várias etapas misturadas, corrigidas pela IA). Sinais, não um número
// mágico (DECISOES §11, risco 5).
const props = defineProps({ trilhaId: { type: Number, required: true } })

const t = ref(null)
const p = ref(null)
const erro = ref('')
async function carregar() {
  try {
    ;[t.value, p.value] = await Promise.all([api(`/api/trilhas/${props.trilhaId}`), api(`/api/trilhas/${props.trilhaId}/painel`)])
  } catch (e) {
    erro.value = e.status === 404 ? 'Trilha não encontrada.' : e.message
  }
}
onMounted(carregar)

const minima = computed(() => p.value?.nota_minima || 80)
const corNota = (n) => (n >= minima.value ? 'verde' : n >= 50 ? 'ambar' : 'vermelho')
const fracas = computed(() => (p.value?.etapas || []).filter((e) => e.sinais?.motivos?.length))
const quando = (iso) => new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
const ONDE = { exercícios: 'exercícios', simulado: 'simulado', revisão: 'revisão', intercalado: 'intercalado' }

// ---------- simulado intercalado ----------
const inter = ref({ perguntas: null, respostas: {}, n: 5, enviando: false, aviso: '' })
const tentativasInter = computed(() => p.value?.intercalado?.tentativas || [])
const corrigindo = computed(() => tentativasInter.value.find((x) => x.estado === 'corrigindo'))
const ultimaInter = computed(() => tentativasInter.value.find((x) => x.estado !== 'corrigindo'))
let relogio = 0
watch(corrigindo, (c) => {
  clearInterval(relogio)
  if (c) relogio = setInterval(carregar, 4000)
})
onBeforeUnmount(() => clearInterval(relogio))
async function sortear() {
  inter.value.aviso = ''
  try {
    const r = await api(`/api/trilhas/${props.trilhaId}/intercalado?n=${inter.value.n}`)
    inter.value.perguntas = r.perguntas
    inter.value.respostas = {}
    if (!r.perguntas.length) inter.value.aviso = 'Ainda não há perguntas: o simulado intercalado usa os bancos do simulado das etapas que você já estudou.'
  } catch (e) {
    inter.value.aviso = e.message
  }
}
const respondidas = computed(() => Object.values(inter.value.respostas).filter((r) => r.trim()).length)
async function mandar() {
  inter.value.enviando = true
  inter.value.aviso = ''
  try {
    const respostas = (inter.value.perguntas || [])
      .map((q, i) => ({ etapa_chave: q.etapa_chave, pergunta: q.indice, resposta: inter.value.respostas[i] || '' }))
      .filter((r) => r.resposta.trim())
    await api(`/api/trilhas/${props.trilhaId}/intercalado`, { metodo: 'POST', corpo: { respostas } })
    inter.value.perguntas = null
    await carregar()
  } catch (e) {
    inter.value.aviso = e.message
  } finally {
    inter.value.enviando = false
  }
}
const VEREDITO = { certo: 'verde', parcial: 'ambar', errado: 'vermelho' }
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas v-if="t" :trilha="t" /></template>
    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-if="t && p">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('trilhas')">Trilhas</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <a :href="link('trilhas', t.id)">{{ t.titulo }}</a>
      </nav>
      <div class="topo-pagina">
        <div>
          <h1>Painel</h1>
          <p class="contexto">Como vai o domínio da trilha: notas, revisões e o que ainda não firmou.</p>
        </div>
      </div>

      <div class="numeros-quadro">
        <div class="numero-quadro"><p class="rotulo-secao">Progresso</p><span class="valor-grande">{{ percentual(p.trilha.progresso) }}%</span><span class="pequeno">{{ p.trilha.progresso.concluidas }} de {{ p.trilha.progresso.total }} etapas</span></div>
        <div class="numero-quadro"><p class="rotulo-secao">Exercícios</p><span class="valor-grande">{{ p.resumo.media_exercicios >= 0 ? p.resumo.media_exercicios + '%' : '—' }}</span><span class="pequeno">média da última vez</span></div>
        <div class="numero-quadro"><p class="rotulo-secao">Revisões</p><span class="valor-grande">{{ p.revisoes.pendentes.length }}</span><span class="pequeno">para hoje · {{ p.revisoes.em_dia }} em dia</span></div>
        <div class="numero-quadro"><p class="rotulo-secao">Etapas fracas</p><span class="valor-grande">{{ p.resumo.etapas_fracas }}</span><span class="pequeno">{{ p.pontos_fracos.length }} ponto(s) fraco(s)</span></div>
      </div>

      <!-- Revisões -->
      <section v-if="p.revisoes.pendentes.length" class="quadro secao">
        <div class="quadro-topo"><h2 class="rotulo-secao">Revisões para hoje</h2></div>
        <ul class="lista">
          <li v-for="r in p.revisoes.pendentes" :key="r.chave" class="linha entre item">
            <div>
              <a :href="link('trilhas', t.id, 'etapas', r.etapa_id)"><strong>{{ r.etapa_titulo }}</strong></a>
              <div class="pequeno">{{ r.atraso > 0 ? `atrasada ${r.atraso} dia(s)` : 'hoje' }} · {{ r.exercicios ? 'perguntas rápidas' : 'lembrar das partes' }}</div>
            </div>
            <a class="botao primario" :href="link('trilhas', t.id, 'etapas', r.etapa_id, 'revisao')">Revisar</a>
          </li>
        </ul>
      </section>

      <!-- Pontos fracos -->
      <section class="quadro secao">
        <div class="quadro-topo"><h2 class="rotulo-secao">Pontos fracos</h2></div>
        <p v-if="!p.pontos_fracos.length && !fracas.length" class="suave">Nada por enquanto: um ponto fraco é o que você errou da última vez que apareceu (nos exercícios, no simulado, na revisão ou no intercalado).</p>
        <ul v-else class="lista">
          <li v-for="e in fracas" :key="e.chave" class="item-fraca">
            <div class="linha entre">
              <a :href="link('trilhas', t.id, 'etapas', e.id)"><strong>{{ e.titulo }}</strong></a>
              <a class="botao discreto" :href="link('trilhas', t.id, 'etapas', e.id) + '#dominio'">Pedir reforço</a>
            </div>
            <span class="pequeno">{{ e.sinais.motivos.join(' · ') }}</span>
            <ul class="pontos">
              <li v-for="(x, i) in p.pontos_fracos.filter((x) => x.etapa_chave === e.chave)" :key="i">
                {{ x.conceito }} <span class="fraco">· errou {{ x.erros }}× · {{ ONDE[x.onde] || x.onde }}</span>
              </li>
            </ul>
          </li>
        </ul>
      </section>

      <!-- Simulado intercalado -->
      <section class="quadro secao">
        <div class="quadro-topo"><h2 class="rotulo-secao">Simulado intercalado</h2></div>
        <p class="suave">Perguntas abertas de várias etapas já estudadas, misturadas (as etapas fracas primeiro): lembrar fora da ordem da trilha é o que fixa. A IA corrige.
          <template v-if="p.intercalado.perguntas"> {{ p.intercalado.perguntas }} pergunta(s) de {{ p.intercalado.etapas }} etapa(s) no banco.</template></p>
        <p v-if="inter.aviso" class="aviso">{{ inter.aviso }}</p>

        <div v-if="corrigindo" class="cartao linha">
          <strong>Corrigindo {{ corrigindo.total }} resposta(s)…</strong>
          <span class="suave">A IA corrige na fila; pode sair desta tela.</span>
        </div>

        <form v-if="inter.perguntas && inter.perguntas.length" class="pilha" @submit.prevent="mandar">
          <p class="pequeno sem-margem">Responda às que quiser; as em branco ficam de fora. A etapa de cada uma aparece na correção.</p>
          <ol class="questoes">
            <li v-for="(q, i) in inter.perguntas" :key="`${q.etapa_chave}-${q.indice}`" class="cartao pilha">
              <label class="enunciado" :for="`int-${i}`" v-html="textoComCodigo(q.pergunta)" />
              <textarea :id="`int-${i}`" v-model="inter.respostas[i]" class="campo" rows="4" placeholder="Com as suas palavras." />
            </li>
          </ol>
          <div class="linha">
            <button class="botao primario" type="submit" :disabled="inter.enviando || !respondidas">Mandar para correção{{ respondidas ? ` (${respondidas})` : '' }}</button>
            <button class="botao discreto" type="button" @click="inter.perguntas = null">Cancelar</button>
          </div>
        </form>
        <div v-else-if="!corrigindo" class="linha">
          <button class="botao" :class="{ primario: !ultimaInter }" :disabled="!p.intercalado.perguntas" @click="sortear">Sortear</button>
          <select v-model.number="inter.n" class="campo curto" aria-label="Quantas perguntas">
            <option v-for="n in [3, 5, 8, 10]" :key="n" :value="n">{{ n }} perguntas</option>
          </select>
        </div>

        <div v-if="ultimaInter" class="pilha bloco">
          <p v-if="ultimaInter.estado === 'falhou'" class="aviso sem-margem">A correção de {{ quando(ultimaInter.criado_em) }} não saiu: {{ ultimaInter.erro }}</p>
          <template v-else>
            <div class="cartao cheio pilha">
              <div class="linha">
                <strong>Última correção</strong>
                <span class="etiqueta" :class="corNota(ultimaInter.nota)">{{ ultimaInter.nota }}%</span>
                <span class="pequeno">{{ quando(ultimaInter.criado_em) }}</span>
              </div>
              <p v-if="ultimaInter.resultado?.resumo" class="sem-margem">{{ ultimaInter.resultado.resumo }}</p>
            </div>
            <details v-for="(it, i) in ultimaInter.resultado?.itens || []" :key="i" class="cartao corrigida">
              <summary class="linha">
                <span class="etiqueta" :class="VEREDITO[it.veredito]">{{ it.nota }}%</span>
                <span><span class="fraco">{{ it.etapa_titulo }} ·</span> <span v-html="textoComCodigo(it.pergunta)" /></span>
              </summary>
              <div class="pilha detalhe">
                <div><span class="rotulo">Sua resposta</span><div class="pre">{{ it.resposta }}</div></div>
                <div><span class="rotulo">Correção</span><div class="pre">{{ it.comentario }}</div></div>
                <div v-if="it.faltou"><span class="rotulo">Faltou</span><div class="pre">{{ it.faltou }}</div></div>
              </div>
            </details>
          </template>
        </div>
      </section>

      <!-- Etapas -->
      <section class="quadro secao">
        <div class="quadro-topo"><h2 class="rotulo-secao">Etapas</h2></div>
        <ul class="lista">
          <li v-for="e in p.etapas" :key="e.id" class="etapa">
            <div class="linha entre">
              <a :href="link('trilhas', t.id, 'etapas', e.id)">{{ e.titulo }}</a>
              <span class="etiqueta" :class="STATUS[e.status].cor">{{ STATUS[e.status].nome }}</span>
            </div>
            <div class="linha sinais">
              <span v-if="e.tipo === 'reforco'" class="etiqueta ambar">reforço</span>
              <span v-if="e.sinais?.exercicios" class="pequeno">Exercícios <span class="etiqueta" :class="corNota(e.sinais.exercicios.ultima)">{{ e.sinais.exercicios.ultima }}%</span></span>
              <span v-if="e.sinais?.simulado" class="pequeno">Simulado <span class="etiqueta" :class="corNota(e.sinais.simulado.ultima)">{{ e.sinais.simulado.ultima }}</span></span>
              <span v-if="e.revisao" class="pequeno">Revisão {{ e.revisao.pendente ? 'hoje' : dataCurta(e.revisao.proxima) }} (degrau {{ e.revisao.nivel + 1 }})</span>
              <span v-if="!e.sinais && !e.revisao && e.tipo !== 'reforco'" class="pequeno fraco">sem notas ainda</span>
            </div>
          </li>
        </ul>
      </section>
    </template>
  </Pagina>
</template>

<style scoped>
.secao { margin-top: var(--e4); }
.bloco { margin-top: var(--e4); }
.sem-margem { margin: 0; }
.contexto { margin: var(--e2) 0 0; font-size: 14px; color: var(--texto-fraco); }
.entre { justify-content: space-between; }
.pre { white-space: pre-wrap; }
.cartao p { display: block; overflow: visible; }
.lista .item, .lista .etapa { padding: var(--e3) 0; }
.item a, .etapa a { color: inherit; text-decoration: none; overflow-wrap: anywhere; }
.item a:hover, .etapa a:hover { text-decoration: underline; }
.item-fraca { padding: var(--e3) 0; display: grid; gap: var(--e2); }
.item-fraca a { color: inherit; text-decoration: none; }
.item-fraca a:hover { text-decoration: underline; }
.pontos { margin: 0; padding-left: var(--e5); font-size: 14.5px; line-height: 1.6; overflow-wrap: anywhere; }
.etapa { display: grid; gap: var(--e1); }
.sinais { gap: var(--e3); }
.questoes { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e3); }
.enunciado { font-size: 15px; line-height: 1.55; display: block; }
.corrigida summary { cursor: pointer; align-items: flex-start; flex-wrap: nowrap; font-size: 15px; }
.corrigida summary .etiqueta { flex: none; }
.detalhe { margin-top: var(--e3); font-size: 14.5px; }
.curto { width: auto; }
</style>
