<script setup>
import { inject, ref } from 'vue'

// Layout das Docs: navegação à esquerda, conteúdo no centro, "nesta página" à direita.
// No celular a navegação vira gaveta (aberta pelo ☰ da barra) e a coluna da direita some.
const gaveta = inject('gaveta', ref(false))
</script>

<template>
  <div class="pagina" :class="{ 'sem-nav': !$slots.nav, 'sem-lateral': !$slots.lateral }">
    <aside v-if="$slots.nav" class="nav" :class="{ aberta: gaveta }">
      <slot name="nav" />
    </aside>
    <div v-if="$slots.nav && gaveta" class="veu" @click="gaveta = false" />
    <main class="conteudo">
      <slot />
    </main>
    <aside v-if="$slots.lateral" class="lateral">
      <slot name="lateral" />
    </aside>
  </div>
</template>

<style scoped>
.pagina {
  display: grid;
  grid-template-columns: var(--nav) minmax(0, 1fr) var(--sumario);
  gap: var(--e6);
  max-width: 1440px;
  margin: 0 auto;
}
.pagina.sem-nav { grid-template-columns: minmax(0, 1fr) var(--sumario); }
.pagina.sem-lateral { grid-template-columns: var(--nav) minmax(0, 1fr); }
.pagina.sem-nav.sem-lateral { grid-template-columns: minmax(0, 1fr); }

.nav {
  position: sticky;
  top: var(--barra);
  height: calc(100vh - var(--barra));
  overflow-y: auto;
  padding: var(--e4) var(--e3) var(--e6) var(--e4);
  border-right: 1px solid var(--borda);
}
.conteudo {
  min-width: 0;
  padding: var(--e6) var(--e5) var(--e8);
  max-width: calc(var(--leitura) + 2 * var(--e5) + 120px);
  width: 100%;
  justify-self: center;
}
.lateral {
  position: sticky;
  top: var(--barra);
  align-self: start;
  max-height: calc(100vh - var(--barra));
  overflow-y: auto;
  padding: var(--e6) var(--e4) var(--e6) 0;
}
.veu { display: none; }

@media (max-width: 1199px) {
  .pagina, .pagina.sem-lateral { grid-template-columns: var(--nav) minmax(0, 1fr); }
  .pagina.sem-nav, .pagina.sem-nav.sem-lateral { grid-template-columns: minmax(0, 1fr); }
  .lateral { display: none; }
}

@media (max-width: 899px) {
  .pagina, .pagina.sem-lateral, .pagina.sem-nav { grid-template-columns: minmax(0, 1fr); gap: 0; }
  .conteudo { padding: var(--e5) var(--e4) calc(var(--e8) + 32px); }
  .nav {
    position: fixed;
    top: var(--barra);
    left: 0;
    bottom: 0;
    z-index: 25;
    width: min(300px, 85vw);
    height: auto;
    background: var(--fundo);
    transform: translateX(-100%);
    transition: transform 160ms ease;
  }
  .nav.aberta { transform: none; }
  .veu {
    display: block;
    position: fixed;
    inset: var(--barra) 0 0 0;
    z-index: 24;
    background: rgb(0 0 0 / 0.45);
  }
}
</style>
