<script setup>
import { computed } from 'vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'

const hoje = computed(() =>
  new Date().toLocaleDateString('pt-BR', { weekday: 'long', day: 'numeric', month: 'long' }),
)
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <span class="nav-titulo">Trilhas ativas</span>
        <p class="nav-vazio">Nenhuma ainda.</p>
      </div>
    </template>

    <h1>Hoje</h1>
    <p class="lead">{{ hoje }}</p>

    <div class="palco fundo-pontos">
      <div class="painel pilha">
        <h3 class="centro">Nada para hoje ainda</h3>
        <p class="suave centro">Quando uma trilha estiver ativa, aqui aparecem as etapas do dia, o que está atrasado e o progresso de cada trilha.</p>
        <a class="botao primario largo" :href="link('trilhas', 'nova')">Criar a primeira trilha</a>
      </div>
    </div>
  </Pagina>
</template>

<style scoped>
.palco { margin-top: var(--e6); padding: var(--e8) var(--e4); border: 1px solid var(--borda); border-radius: var(--raio); }
.centro { text-align: center; margin: 0; }
</style>
