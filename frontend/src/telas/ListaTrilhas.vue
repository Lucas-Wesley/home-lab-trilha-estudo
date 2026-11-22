<script setup>
import { computed, onMounted, ref } from 'vue'
import Capa from '../Capa.vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { ESTADOS, PROFUNDIDADES, duracao, percentual } from '../rotulos.js'

// T55: as trilhas com o mesmo grupo (a escola dos cursos, ex.: "Chef Enzo Neto") viram um cartão só
// na lista, que abre #/trilhas/grupo/<nome> com as trilhas dele na ordem em que foram criadas.
const props = defineProps({ grupo: { type: String, default: '' } })

const trilhas = ref(null)
const erro = ref('')
const busca = ref('')
const filtro = ref('todas')

onMounted(async () => {
  try {
    trilhas.value = await api('/api/trilhas')
  } catch (e) {
    erro.value = e.message
  }
})

const filtradas = computed(() => {
  const termo = busca.value.trim().toLowerCase()
  return (trilhas.value || []).filter(
    (t) =>
      (!props.grupo || t.grupo === props.grupo) &&
      (filtro.value === 'todas' || t.estado === filtro.value) &&
      (!termo || (t.titulo + ' ' + t.objetivo + ' ' + (t.grupo || '')).toLowerCase().includes(termo)),
  )
})

// Os cartões: a trilha solta, ou o grupo no lugar da primeira trilha dele. Buscando, tudo solto.
const cartoes = computed(() => {
  if (props.grupo) return [...filtradas.value].sort((a, b) => a.id - b.id).map((t) => ({ trilha: t }))
  if (busca.value.trim()) return filtradas.value.map((t) => ({ trilha: t }))
  const lista = []
  const grupos = new Map()
  for (const t of filtradas.value) {
    if (!t.grupo) {
      lista.push({ trilha: t })
      continue
    }
    let g = grupos.get(t.grupo)
    if (!g) {
      g = { grupo: t.grupo, trilhas: [], progresso: { total: 0, concluidas: 0, puladas: 0, minutos_total: 0, minutos_feitos: 0 } }
      grupos.set(t.grupo, g)
      lista.push(g)
    }
    g.trilhas.push(t)
    for (const k of Object.keys(g.progresso)) g.progresso[k] += t.progresso[k] || 0
  }
  return lista
})
const FILTROS = { todas: 'Todas', ativa: 'Ativas', rascunho: 'Rascunhos', pausada: 'Pausadas', concluida: 'Concluídas' }
function trocarFiltro() {
  const chaves = Object.keys(FILTROS)
  filtro.value = chaves[(chaves.indexOf(filtro.value) + 1) % chaves.length]
}
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas /></template>

    <div class="topo-pagina">
      <h1 v-if="!grupo">Trilhas</h1>
      <h1 v-else><a class="voltar-grupo" :href="link('trilhas')">Trilhas</a> <span class="fraco">›</span> {{ grupo }}</h1>
      <div v-if="trilhas && trilhas.length" class="meio">
        <label class="busca">
          <Icone nome="lupa" />
          <span class="so-leitor">Buscar</span>
          <input v-model="busca" class="campo" placeholder="Buscar trilhas…" />
        </label>
      </div>
      <div class="acoes">
        <button v-if="trilhas && trilhas.length" class="seletor" @click="trocarFiltro">Situação <strong>{{ FILTROS[filtro] }}</strong> <Icone nome="seta-baixo" :tamanho="14" /></button>
        <a class="botao primario" :href="link('trilhas', 'nova')"><Icone nome="mais" /> Nova trilha</a>
      </div>
    </div>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-else-if="trilhas && trilhas.length">
      <!-- T47: um cartão por trilha, com a capa; o objetivo fica no título do cartão (passar o mouse). -->
      <ul class="cartoes-trilha">
        <li v-for="c in cartoes" :key="c.grupo ? 'g:' + c.grupo : c.trilha.id">
          <a v-if="c.grupo" class="cartao-trilha" :href="link('trilhas', 'grupo', c.grupo)" :title="c.trilhas.map((t) => t.titulo).join(' · ')">
            <div class="capa-caixa grupo">
              <Capa :titulo="c.grupo" />
              <span class="etiqueta estado">{{ c.trilhas.length }} {{ c.trilhas.length === 1 ? 'trilha' : 'trilhas' }}</span>
            </div>
            <strong class="titulo">{{ c.grupo }}</strong>
            <span class="meta">
              {{ c.progresso.concluidas + c.progresso.puladas }}/{{ c.progresso.total }} etapas · {{ duracao(c.progresso.minutos_total) }}
            </span>
            <span class="progresso"><span :style="{ width: percentual(c.progresso) + '%' }" /></span>
          </a>
          <a v-else class="cartao-trilha" :href="link('trilhas', c.trilha.id)" :title="c.trilha.objetivo || c.trilha.descricao">
            <div class="capa-caixa">
              <Capa :titulo="c.trilha.titulo" />
              <span v-if="c.trilha.estado !== 'ativa'" class="etiqueta estado" :class="ESTADOS[c.trilha.estado]?.cor">{{ ESTADOS[c.trilha.estado]?.nome }}</span>
            </div>
            <strong class="titulo">{{ c.trilha.titulo }}</strong>
            <span class="meta">
              {{ c.trilha.progresso.concluidas + c.trilha.progresso.puladas }}/{{ c.trilha.progresso.total }} etapas · {{ duracao(c.trilha.progresso.minutos_total) }}
              · {{ PROFUNDIDADES[c.trilha.profundidade] }}
            </span>
            <span class="progresso"><span :style="{ width: percentual(c.trilha.progresso) + '%' }" /></span>
          </a>
        </li>
      </ul>
      <p v-if="!cartoes.length" class="fraco">Nenhuma trilha com esse filtro.</p>
    </template>
    <div v-else-if="trilhas" class="palco fundo-pontos">
      <div class="painel pilha">
        <h3 class="centro">Nenhuma trilha ainda</h3>
        <p class="suave centro">Monte a primeira à mão, ou importe uma trilha pronta.</p>
        <a class="botao primario largo" :href="link('trilhas', 'nova')">Nova trilha</a>
      </div>
    </div>
  </Pagina>
</template>

<style scoped>
.cartoes-trilha {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--e5) var(--e4);
  margin: 0;
  padding: 0;
  list-style: none;
}
.cartao-trilha { display: grid; gap: 6px; color: var(--texto); text-decoration: none; }
.capa-caixa { position: relative; margin-bottom: 4px; transition: transform 140ms ease; }
.cartao-trilha:hover .capa-caixa { transform: translateY(-2px); }
.cartao-trilha:hover .titulo { color: var(--link); }
.estado { position: absolute; top: 8px; right: 8px; }
/* O grupo parece uma pilha de capas: uma borda deslocada atrás. */
.capa-caixa.grupo { box-shadow: 6px -6px 0 -1px var(--superficie-2), 6px -6px 0 0 var(--borda); border-radius: var(--raio); }
.voltar-grupo { color: var(--texto-suave); text-decoration: none; }
.voltar-grupo:hover { color: var(--link); }
.titulo { font-size: 15px; font-weight: 600; line-height: 1.35; }
.meta { font-size: 12.5px; color: var(--texto-suave); }
.progresso { display: block; height: 4px; margin-top: 2px; }
@media (max-width: 599px) {
  .cartoes-trilha { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--e4) var(--e3); }
  .titulo { font-size: 14px; }
  .meta { font-size: 12px; }
}
.palco { margin-top: var(--e5); }
.centro { margin: 0; }
</style>
