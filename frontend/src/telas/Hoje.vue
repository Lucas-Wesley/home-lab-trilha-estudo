<script setup>
import { computed, onMounted, ref } from 'vue'
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
const minutosHoje = (th) => th.hoje.filter((i) => !['concluida', 'pulada'].includes(i.status)).reduce((s, i) => s + i.minutos, 0)
const feito = (i) => i.status === 'concluida' || i.status === 'pulada'

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

    <h1>Hoje</h1>
    <p class="lead">{{ hojeTexto }}</p>
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <!-- Revisão espaçada (fase 7): as etapas concluídas que voltaram hoje, de todas as trilhas ativas. -->
    <section v-if="revisoes.length" class="trilha-hoje">
      <h2>Revisões de hoje</h2>
      <p class="suave">Perguntas rápidas das etapas que você já concluiu, no dia de cada uma voltar.</p>
      <div class="pilha bloco">
        <a v-for="r in revisoes" :key="`${r.trilha_id}-${r.chave}`" class="cartao item" :href="link('trilhas', r.trilha_id, 'etapas', r.etapa_id, 'revisao')">
          <div>
            <strong>{{ r.etapa_titulo }}</strong>
            <div class="fraco">{{ r.trilha_titulo }}<template v-if="r.atraso > 0"> · atrasada {{ r.atraso }} dia(s)</template></div>
          </div>
          <span class="etiqueta ambar">revisar</span>
        </a>
      </div>
    </section>

    <template v-if="dados && dados.trilhas.length">
      <section v-for="th in dados.trilhas" :key="th.id" class="trilha-hoje">
        <div class="linha entre">
          <h2><a :href="link('trilhas', th.id)">{{ th.titulo }}</a></h2>
          <span class="fraco">{{ percentual(th.progresso) }}% · previsão {{ dataCurta(th.previsao) }}</span>
        </div>
        <div class="progresso"><span :style="{ width: percentual(th.progresso) + '%' }" /></div>

        <div v-if="th.atrasadas.length" class="cartao atraso pilha">
          <div class="linha entre">
            <strong>{{ th.atrasadas.length }} etapa(s) atrasada(s)</strong>
            <button class="botao" @click="recalcular(th)">Recalcular a partir de hoje</button>
          </div>
          <ul class="itens">
            <li v-for="i in th.atrasadas" :key="i.etapa_id">
              <a :href="link('trilhas', th.id, 'etapas', i.etapa_id)">{{ i.titulo }}</a>
              <span class="fraco"> · {{ duracao(i.total) }}</span>
            </li>
          </ul>
        </div>

        <div v-if="th.hoje.length" class="pilha bloco">
          <span class="rotulo">Para hoje · {{ duracao(minutosHoje(th)) }}</span>
          <a v-for="i in th.hoje" :key="i.etapa_id" class="cartao item" :class="{ feito: feito(i) }" :href="link('trilhas', th.id, 'etapas', i.etapa_id)">
            <div>
              <strong>{{ i.titulo }}</strong>
              <div class="fraco">{{ duracao(i.minutos) }}<template v-if="i.parte"> · parte {{ i.parte }}</template></div>
            </div>
            <span class="etiqueta" :class="STATUS[i.status].cor">{{ STATUS[i.status].nome }}</span>
          </a>
        </div>
        <p v-else-if="!th.atrasadas.length" class="suave bloco">Hoje não é dia de estudo nesta trilha.</p>

        <div v-if="th.proximo" class="bloco">
          <span class="rotulo">Próximo dia de estudo · {{ dataCurta(th.proximo.data) }}</span>
          <ul class="itens">
            <li v-for="i in th.proximo.itens" :key="i.etapa_id">
              <a :href="link('trilhas', th.id, 'etapas', i.etapa_id)">{{ i.titulo }}</a>
              <span class="fraco"> · {{ duracao(i.minutos) }}<template v-if="i.parte"> (parte {{ i.parte }})</template></span>
            </li>
          </ul>
          <p class="pequeno">Quer adiantar? Pode abrir a etapa e estudar agora.</p>
        </div>
      </section>
    </template>

    <div v-else-if="dados" class="palco fundo-pontos">
      <div class="painel pilha">
        <h3 class="centro">Nada para hoje ainda</h3>
        <p class="suave centro">Uma trilha aparece aqui quando ganha uma meta (dias e tempo de estudo).</p>
        <a class="botao primario largo" :href="link('trilhas')">Ver as trilhas</a>
      </div>
    </div>
  </Pagina>
</template>

<style scoped>
.trilha-hoje { margin-top: var(--e6); padding-bottom: var(--e6); border-bottom: 1px solid var(--borda); }
.trilha-hoje h2 a { color: inherit; text-decoration: none; }
.trilha-hoje h2 a:hover { text-decoration: underline; }
.trilha-hoje .progresso { margin: var(--e3) 0 var(--e4); }
.entre { justify-content: space-between; }
.bloco { margin-top: var(--e4); }
.atraso { border-color: var(--vermelho-fundo); }
.item { display: flex; justify-content: space-between; align-items: center; gap: var(--e3); }
.item.feito strong { color: var(--texto-fraco); text-decoration: line-through; }
.itens { margin: var(--e2) 0 0; padding-left: var(--e5); line-height: 1.7; }
.palco { margin-top: var(--e6); padding: var(--e8) var(--e4); border: 1px solid var(--borda); border-radius: var(--raio); }
.centro { text-align: center; margin: 0; }
</style>
