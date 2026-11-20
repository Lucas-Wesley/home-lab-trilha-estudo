<script setup>
import { computed, onMounted, ref } from 'vue'
import Capa from '../Capa.vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { ESTADOS, PROFUNDIDADES, duracao, percentual } from '../rotulos.js'

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

const visiveis = computed(() => {
  const termo = busca.value.trim().toLowerCase()
  return (trilhas.value || []).filter(
    (t) => (filtro.value === 'todas' || t.estado === filtro.value) && (!termo || (t.titulo + ' ' + t.objetivo).toLowerCase().includes(termo)),
  )
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
      <h1>Trilhas</h1>
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
        <li v-for="t in visiveis" :key="t.id">
          <a class="cartao-trilha" :href="link('trilhas', t.id)" :title="t.objetivo || t.descricao">
            <div class="capa-caixa">
              <Capa :titulo="t.titulo" />
              <span v-if="t.estado !== 'ativa'" class="etiqueta estado" :class="ESTADOS[t.estado]?.cor">{{ ESTADOS[t.estado]?.nome }}</span>
            </div>
            <strong class="titulo">{{ t.titulo }}</strong>
            <span class="meta">
              {{ t.progresso.concluidas + t.progresso.puladas }}/{{ t.progresso.total }} etapas · {{ duracao(t.progresso.minutos_total) }}
              · {{ PROFUNDIDADES[t.profundidade] }}
            </span>
            <span class="progresso"><span :style="{ width: percentual(t.progresso) + '%' }" /></span>
          </a>
        </li>
      </ul>
      <p v-if="!visiveis.length" class="fraco">Nenhuma trilha com esse filtro.</p>
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
