<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref } from 'vue'
import Icone from './Icone.vue'
import Hoje from './telas/Hoje.vue'
import Trilhas from './telas/Trilhas.vue'
import Documentos from './telas/Documentos.vue'
import Vitrine from './telas/Vitrine.vue'
import { lerRota, link } from './rotas.js'
import { aplicarTema, lerTema, proximoTema } from './tema.js'

const hash = ref(location.hash)
const rota = computed(() => lerRota(hash.value))
const telas = { hoje: Hoje, trilhas: Trilhas, documentos: Documentos, design: Vitrine }

const abas = [
  { secao: 'hoje', nome: 'Hoje', icone: 'hoje' },
  { secao: 'trilhas', nome: 'Trilhas', icone: 'trilhas' },
  { secao: 'documentos', nome: 'Documentos', icone: 'livro' },
  { secao: 'design', nome: 'Design', icone: 'design' },
]

const tema = ref(lerTema())
const NOMES_TEMA = { auto: 'Tema do aparelho', claro: 'Tema claro', escuro: 'Tema escuro' }
const ICONES_TEMA = { auto: 'auto', claro: 'sol', escuro: 'lua' }
function trocarTema() {
  tema.value = proximoTema(tema.value)
  aplicarTema(tema.value)
}

// Gaveta da navegação lateral no celular; cada tela (Pagina.vue) mostra a sua navegação nela.
const gaveta = ref(false)
provide('gaveta', gaveta)

// Outra página (não só outra âncora da mesma página) começa do topo.
const pagina = (r) => [r.secao, ...r.partes].join('/')
function aoMudarHash() {
  const anterior = pagina(rota.value)
  hash.value = location.hash
  gaveta.value = false
  if (pagina(rota.value) !== anterior && !rota.value.ancora) window.scrollTo(0, 0)
}

onMounted(() => {
  aplicarTema(tema.value)
  window.addEventListener('hashchange', aoMudarHash)
})
onBeforeUnmount(() => window.removeEventListener('hashchange', aoMudarHash))
</script>

<template>
  <header class="barra">
    <button class="botao discreto icone so-celular" :aria-expanded="gaveta" aria-label="Abrir a navegação" @click="gaveta = !gaveta">
      <Icone :nome="gaveta ? 'fechar' : 'menu'" />
    </button>
    <a class="marca" :href="link('hoje')">
      <span class="marca-icone"><Icone nome="marca" :tamanho="18" /></span>
      <span>Trilha de Estudo</span>
    </a>
    <nav class="abas abas-topo" aria-label="Seções">
      <a
        v-for="aba in abas"
        :key="aba.secao"
        :href="link(aba.secao)"
        :class="{ ativa: rota.secao === aba.secao }"
        :aria-current="rota.secao === aba.secao ? 'page' : undefined"
      >{{ aba.nome }}</a>
    </nav>
    <div class="barra-direita">
      <button class="botao discreto icone" :title="NOMES_TEMA[tema]" :aria-label="NOMES_TEMA[tema]" @click="trocarTema">
        <Icone :nome="ICONES_TEMA[tema]" />
      </button>
      <a class="botao primario" :href="link('trilhas', 'nova')">
        <Icone nome="mais" />
        <span class="so-largo">Nova trilha</span>
      </a>
    </div>
  </header>

  <component :is="telas[rota.secao]" :rota="rota" />

  <nav class="abas-celular" aria-label="Seções">
    <a
      v-for="aba in abas"
      :key="aba.secao"
      :href="link(aba.secao)"
      :class="{ ativa: rota.secao === aba.secao }"
      :aria-current="rota.secao === aba.secao ? 'page' : undefined"
    >
      <Icone :nome="aba.icone" :tamanho="20" />
      <span>{{ aba.nome }}</span>
    </a>
  </nav>
</template>

<style scoped>
.barra {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  gap: var(--e4);
  height: var(--barra);
  padding: 0 var(--e5);
  background: var(--fundo);
  border-bottom: 1px solid var(--borda);
}
.marca {
  display: inline-flex;
  align-items: center;
  gap: var(--e2);
  font-family: var(--fonte-titulo);
  font-size: 19px;
  font-weight: 500;
  color: var(--texto);
  text-decoration: none;
  white-space: nowrap;
}
.marca-icone { display: inline-flex; color: var(--ambar); }
.abas-topo { margin: 0 auto; }
.barra-direita { display: flex; align-items: center; gap: var(--e2); margin-left: auto; }
.so-celular { display: none; }

.abas-celular { display: none; }

@media (max-width: 899px) {
  .barra { padding: 0 var(--e3); gap: var(--e2); }
  .so-celular { display: inline-flex; }
  .abas-topo { display: none; }
  .so-largo { display: none; }
  .barra-direita .botao.primario { width: var(--controle); padding: 0; }

  .abas-celular {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 30;
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    padding: var(--e1) var(--e2) calc(var(--e1) + env(safe-area-inset-bottom));
    background: var(--fundo);
    border-top: 1px solid var(--borda);
  }
  .abas-celular a {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 6px 0;
    font-size: 11.5px;
    font-weight: 500;
    color: var(--texto-fraco);
    text-decoration: none;
    border-radius: var(--raio-p);
  }
  .abas-celular a.ativa { color: var(--texto); background: var(--superficie-2); }
}
</style>
