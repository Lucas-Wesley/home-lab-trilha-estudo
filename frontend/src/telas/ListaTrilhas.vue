<script setup>
import { computed, onMounted, ref } from 'vue'
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

    <h1>Trilhas</h1>
    <p class="lead">Estudo em ordem, com profundidade e progresso.</p>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-else-if="trilhas && trilhas.length">
      <div class="linha filtros">
        <label class="busca">
          <Icone nome="lupa" />
          <span class="so-leitor">Buscar</span>
          <input v-model="busca" class="campo" placeholder="Buscar trilhas…" />
        </label>
        <button class="seletor" @click="trocarFiltro">Situação <strong>{{ FILTROS[filtro] }}</strong> <Icone nome="seta-baixo" :tamanho="14" /></button>
      </div>
      <table class="tabela">
        <thead>
          <tr><th>Trilha</th><th>Situação</th><th class="direita">Progresso</th></tr>
        </thead>
        <tbody>
          <tr v-for="t in visiveis" :key="t.id">
            <td>
              <a class="titulo" :href="link('trilhas', t.id)">{{ t.titulo }}</a>
              <div class="descricao">{{ t.objetivo || t.descricao }}</div>
            </td>
            <td>
              <div class="etiquetas">
                <span class="etiqueta" :class="ESTADOS[t.estado]?.cor">{{ ESTADOS[t.estado]?.nome }}</span>
                <span class="etiqueta">{{ PROFUNDIDADES[t.profundidade] }}</span>
              </div>
            </td>
            <td class="direita">
              <div class="fraco">{{ t.progresso.concluidas + t.progresso.puladas }}/{{ t.progresso.total }} etapas</div>
              <div class="fraco">{{ duracao(t.progresso.minutos_total) }}</div>
              <div class="progresso mini"><span :style="{ width: percentual(t.progresso) + '%' }" /></div>
            </td>
          </tr>
          <tr v-if="!visiveis.length"><td colspan="3" class="fraco">Nenhuma trilha com esse filtro.</td></tr>
        </tbody>
      </table>
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
.filtros { margin: var(--e5) 0 var(--e4); }
.filtros .busca { flex: 1; min-width: 200px; max-width: 400px; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
.direita { text-align: right; white-space: nowrap; }
.mini { width: 96px; margin: var(--e2) 0 0 auto; }
.palco { margin-top: var(--e5); padding: var(--e8) var(--e4); border: 1px solid var(--borda); border-radius: var(--raio); }
.centro { text-align: center; margin: 0; }
</style>
