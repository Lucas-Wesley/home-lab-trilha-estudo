<script setup>
import { computed, onMounted, ref } from 'vue'
import Capa from '../Capa.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { STATUS, dataCurta, duracao, percentual } from '../rotulos.js'

// Ao abrir o app: o que há para hoje em cada trilha ativa, o que atrasou e o progresso (T10).
const dados = ref(null)
const revisoes = ref([])
const erro = ref('')

async function carregar() {
  try {
    ;[dados.value, { revisoes: revisoes.value }] = await Promise.all([api('/api/hoje'), api('/api/revisoes')])
  } catch (e) {
    erro.value = e.message
  }
}
onMounted(carregar)

const hojeTexto = computed(() => new Date().toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' }))
const feito = (i) => i.status === 'concluida' || i.status === 'pulada'

// Os números do topo: o que falta estudar hoje, as revisões do dia e o que passou do prazo.
const paraHoje = computed(() => {
  const itens = (dados.value?.trilhas || []).flatMap((th) => th.hoje.filter((i) => !feito(i)))
  return { etapas: itens.length, minutos: itens.reduce((s, i) => s + i.minutos, 0) }
})
const atrasadas = computed(() => (dados.value?.trilhas || []).reduce((s, th) => s + th.atrasadas.length, 0))
// Só aparecem as trilhas com estudo (ou atraso) hoje; as outras ficam fora (T48).
const comAgenda = computed(() => (dados.value?.trilhas || []).filter((th) => th.hoje.length || th.atrasadas.length))
// O próximo dia só aparece quando o de hoje acabou (para adiantar).
const hojeFeito = (th) => !th.atrasadas.length && th.hoje.every(feito)
const paraHojePorTrilha = (th) => th.hoje.filter((i) => !feito(i)).reduce((s, i) => s + i.minutos, 0)

async function recalcular(th) {
  try {
    await api(`/api/trilhas/${th.id}/recalcular`, { metodo: 'POST' })
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <span class="nav-titulo">Trilhas ativas</span>
        <template v-if="dados && dados.trilhas.length">
          <a v-for="th in dados.trilhas" :key="th.id" class="nav-item" :href="link('trilhas', th.id)">{{ th.titulo }}</a>
        </template>
        <p v-else class="nav-vazio">Nenhuma ainda.</p>
      </div>
    </template>

    <div class="topo-pagina">
      <h1>Hoje</h1>
      <div class="acoes"><span class="pequeno">{{ hojeTexto }}</span></div>
    </div>
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <div v-if="dados && (paraHoje.etapas || revisoes.length || atrasadas)" class="numeros-quadro">
      <div class="numero-quadro">
        <p class="rotulo-secao">Para hoje</p>
        <span class="valor-grande">{{ paraHoje.etapas }}</span>
        <span class="pequeno">{{ paraHoje.etapas === 1 ? 'etapa' : 'etapas' }} · {{ duracao(paraHoje.minutos) }}</span>
      </div>
      <div class="numero-quadro">
        <p class="rotulo-secao">Revisões</p>
        <span class="valor-grande">{{ revisoes.length }}</span>
        <span class="pequeno">para fazer hoje</span>
      </div>
      <div v-if="atrasadas" class="numero-quadro">
        <p class="rotulo-secao">Atrasadas</p>
        <span class="valor-grande">{{ atrasadas }}</span>
        <span class="pequeno">{{ atrasadas === 1 ? 'etapa' : 'etapas' }} fora do dia</span>
      </div>
    </div>

    <!-- Revisão espaçada (fase 7): as etapas concluídas que voltaram hoje, de todas as trilhas ativas. -->
    <section v-if="revisoes.length" class="quadro">
      <div class="quadro-topo"><h2 class="rotulo-secao">Revisões de hoje</h2></div>
      <div class="lista">
        <a v-for="r in revisoes" :key="`${r.trilha_id}-${r.chave}`" class="item" :href="link('trilhas', r.trilha_id, 'etapas', r.etapa_id, 'revisao')">
          <div>
            <strong>{{ r.etapa_titulo }}</strong>
            <div class="pequeno">{{ r.trilha_titulo }}<template v-if="r.atraso > 0"> · atrasada {{ r.atraso }} dia(s)</template></div>
          </div>
          <span class="etiqueta ambar">revisar</span>
        </a>
      </div>
    </section>

    <template v-if="dados">
      <section v-for="th in comAgenda" :key="th.id" class="trilha-hoje">
        <div class="cabeca-trilha">
          <Capa :titulo="th.titulo" pequena />
          <h2><a :href="link('trilhas', th.id)">{{ th.titulo }}</a></h2>
          <span class="fraco">{{ percentual(th.progresso) }}% · previsão {{ dataCurta(th.previsao) }}</span>
        </div>
        <div class="progresso"><span :style="{ width: percentual(th.progresso) + '%' }" /></div>

        <template v-if="th.atrasadas.length">
          <p class="aviso">
            {{ th.atrasadas.length }} {{ th.atrasadas.length === 1 ? 'etapa atrasada' : 'etapas atrasadas' }}.
            <button class="botao discreto recalc" @click="recalcular(th)">Recalcular a partir de hoje</button>
          </p>
          <ul class="lista atrasadas">
            <li v-for="i in th.atrasadas" :key="i.etapa_id" class="item">
              <a :href="link('trilhas', th.id, 'etapas', i.etapa_id)">{{ i.titulo }}</a>
              <span class="fraco">{{ duracao(i.total) }}</span>
            </li>
          </ul>
        </template>

        <template v-if="th.hoje.length">
          <p class="rotulo-secao bloco">Para hoje · {{ duracao(paraHojePorTrilha(th)) }}</p>
          <div class="lista">
            <a v-for="i in th.hoje" :key="i.etapa_id" class="item" :class="{ feito: feito(i) }" :href="link('trilhas', th.id, 'etapas', i.etapa_id)">
              <div>
                <strong>{{ i.titulo }}</strong>
                <div class="pequeno">{{ duracao(i.minutos) }}<template v-if="i.parte"> · parte {{ i.parte }}</template></div>
              </div>
              <span class="etiqueta" :class="STATUS[i.status].cor">{{ STATUS[i.status].nome }}</span>
            </a>
          </div>
        </template>

        <div v-if="th.proximo && hojeFeito(th)" class="bloco">
          <p class="rotulo-secao">Próximo dia de estudo · {{ dataCurta(th.proximo.data) }}</p>
          <ul class="lista">
            <li v-for="i in th.proximo.itens" :key="i.etapa_id" class="item">
              <a :href="link('trilhas', th.id, 'etapas', i.etapa_id)">{{ i.titulo }}</a>
              <span class="fraco">{{ duracao(i.minutos) }}<template v-if="i.parte"> · parte {{ i.parte }}</template></span>
            </li>
          </ul>
          <p class="pequeno">Quer adiantar? Pode abrir a etapa e estudar agora.</p>
        </div>
      </section>

    </template>

    <!-- T48: sem nada para hoje, a tela fica vazia (as trilhas estão na lateral e em Trilhas). -->
    <p v-if="dados && !comAgenda.length && !revisoes.length" class="vazio">Nada para estudar hoje.</p>
  </Pagina>
</template>

<style scoped>
.trilha-hoje { margin-top: var(--e6); padding-bottom: var(--e6); border-bottom: 1px solid var(--borda); }
.trilha-hoje:last-child { border-bottom: 0; padding-bottom: 0; }
.trilha-hoje h2 { flex: 1; min-width: 0; margin-bottom: 0; }
.cabeca-trilha { display: flex; align-items: center; gap: var(--e3); flex-wrap: wrap; }
.trilha-hoje h2 a { color: inherit; text-decoration: none; }
.trilha-hoje h2 a:hover { text-decoration: underline; }
.trilha-hoje .progresso { margin: var(--e3) 0 var(--e4); }
.entre { justify-content: space-between; }
.bloco { margin-top: var(--e4); }
.quadro { margin-top: var(--e5); }
.lista .item { display: flex; align-items: center; justify-content: space-between; gap: var(--e3); padding: 9px 0; color: inherit; text-decoration: none; }
.lista .item:hover strong { color: var(--link); }
.lista .item strong { font-weight: 600; font-size: 14.5px; }
.item.feito strong { color: var(--texto-fraco); text-decoration: line-through; }
.lista .pequeno { margin-top: 1px; }
.recalc { height: auto; padding: 0; font-size: inherit; text-decoration: underline; text-underline-offset: 3px; }
.recalc:hover { background: transparent; color: var(--vermelho); }
ul.lista li.item { display: flex; align-items: baseline; justify-content: space-between; gap: var(--e3); padding: 8px 0; font-size: 14.5px; }
ul.lista li.item a { color: inherit; text-decoration: none; }
ul.lista li.item a:hover { color: var(--link); }
ul.lista.atrasadas { margin-bottom: var(--e4); }
.quadro .lista .item:last-child { padding-bottom: 4px; }
.rotulo-secao { margin-bottom: var(--e2); }
.vazio { margin-top: var(--e6); color: var(--texto-fraco); }
</style>
