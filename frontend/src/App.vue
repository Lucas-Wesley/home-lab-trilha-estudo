<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import Icone from './Icone.vue'
import Hoje from './telas/Hoje.vue'
import Entrada from './telas/Entrada.vue'
import Trilhas from './telas/Trilhas.vue'
import Projeto from './telas/Projeto.vue'
import Vitrine from './telas/Vitrine.vue'
import Ajustes from './telas/Ajustes.vue'
import { api } from './api.js'
import { linkCompartilhado } from './entrada.js'
import { lerRota, link } from './rotas.js'
import { aplicarTema, lerTema, proximoTema } from './tema.js'

// Aviso que a tela da Entrada mostra quando o app abre pelo compartilhar do celular.
const avisoCompartilhar = ref('')
provide('avisoCompartilhar', avisoCompartilhar)

// Compartilhar do celular (T36): o share_target manda o link para /compartilhar; ele vai para a
// Entrada e o app abre já na seção.
if (location.pathname === '/compartilhar') {
  const url = linkCompartilhado(location.search)
  history.replaceState(null, '', '/#/entrada')
  if (!url) {
    avisoCompartilhar.value = 'Não achei um link do YouTube no que veio.'
  } else {
    api('/api/entrada', { metodo: 'POST', corpo: { url, automatico: true } })
      .then(() => {
        avisoCompartilhar.value = 'Na Entrada'
        buscarProntas()
      })
      .catch((e) => {
        avisoCompartilhar.value = e.message
      })
  }
}

const hash = ref(location.hash)
const rota = computed(() => lerRota(hash.value))
const telas = { hoje: Hoje, trilhas: Trilhas, entrada: Entrada, projeto: Projeto, ajustes: Ajustes, design: Vitrine }
const versao = ref('')

const abas = [
  { secao: 'hoje', nome: 'Hoje', icone: 'hoje' },
  { secao: 'trilhas', nome: 'Trilhas', icone: 'trilhas' },
  { secao: 'entrada', nome: 'Entrada', icone: 'entrada' },
  // T52: o caderno do Projeto de estudo (Banco, Nuvem e Escola) ligado às trilhas.
  { secao: 'projeto', nome: 'Projeto', nomeLongo: 'Projeto de estudo', icone: 'projeto' },
  { secao: 'ajustes', nome: 'Ajustes', icone: 'ajustes' },
]

// Fontes prontas que ainda não viraram fragmentos: número discreto na aba Entrada.
const prontas = ref(0)
async function buscarProntas() {
  try {
    const fontes = await api('/api/entrada')
    prontas.value = fontes.filter((f) => f.estado === 'pronta').length
  } catch {
    // Sem o número, a aba só fica sem o contador.
  }
}
watch(() => rota.value.secao, buscarProntas)

const tema = ref(lerTema())
const NOMES_TEMA = { auto: 'Tema do aparelho', claro: 'Tema claro', escuro: 'Tema escuro' }
const ICONES_TEMA = { auto: 'auto', claro: 'sol', escuro: 'lua' }
function escolherTema(t) {
  tema.value = t
  aplicarTema(t)
}
const trocarTema = () => escolherTema(proximoTema(tema.value))

// A lateral vira gaveta no celular (o item "Menu" da barra do pé abre). A navegação de cada tela
// vai para o #nav-contexto da lateral por <Teleport> (Pagina.vue).
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

onMounted(async () => {
  aplicarTema(tema.value)
  window.addEventListener('hashchange', aoMudarHash)
  buscarProntas()
  try {
    versao.value = (await api('/api/health', { tempo: 4000 })).version
  } catch {
    versao.value = ''
  }
})
onBeforeUnmount(() => window.removeEventListener('hashchange', aoMudarHash))
</script>

<template>
  <div class="moldura">
    <aside class="menu" :class="{ aberta: gaveta }" aria-label="Menu">
      <a class="marca" :href="link('hoje')">
        <span class="marca-icone"><Icone nome="marca" :tamanho="18" /></span>
        <span class="marca-texto">
          <span class="marca-sobre">Homelab</span>
          <span class="marca-nome">Trilhas</span>
        </span>
      </a>
      <nav>
        <a
          v-for="aba in abas"
          :key="aba.secao"
          class="nav-item"
          :class="{ ativo: rota.secao === aba.secao }"
          :href="link(aba.secao)"
          :aria-current="rota.secao === aba.secao ? 'page' : undefined"
        >
          <Icone :nome="aba.icone" :tamanho="18" />
          <span>{{ aba.nomeLongo || aba.nome }}</span>
          <span v-if="aba.secao === 'entrada' && prontas" class="contador">{{ prontas }}</span>
        </a>
      </nav>
      <!-- A navegação de cada tela (as etapas da trilha aberta, as trilhas ativas…) entra aqui. -->
      <div id="nav-contexto" />
      <div class="rodape">
        <span>{{ versao ? `versão ${versao.slice(0, 7)}` : 'sem conexão' }}</span>
        <button class="botao discreto icone" :title="NOMES_TEMA[tema]" :aria-label="NOMES_TEMA[tema]" @click="trocarTema">
          <Icone :nome="ICONES_TEMA[tema]" />
        </button>
      </div>
    </aside>
    <div v-if="gaveta" class="veu" @click="gaveta = false" />

    <main class="conteudo">
      <component :is="telas[rota.secao]" :rota="rota" :tema="rota.secao === 'ajustes' ? tema : undefined" @tema="escolherTema" />
    </main>
  </div>

  <nav class="abas-celular" aria-label="Seções" :style="{ gridTemplateColumns: `repeat(${abas.length + 1}, 1fr)` }">
    <button @click="gaveta = !gaveta">
      <Icone :nome="gaveta ? 'fechar' : 'menu'" :tamanho="20" />
      <span>Menu</span>
    </button>
    <a
      v-for="aba in abas"
      :key="aba.secao"
      :href="link(aba.secao)"
      :class="{ ativa: rota.secao === aba.secao }"
      :aria-current="rota.secao === aba.secao ? 'page' : undefined"
    >
      <Icone :nome="aba.icone" :tamanho="20" />
      <span>{{ aba.nome }} <span v-if="aba.secao === 'entrada' && prontas" class="contador">{{ prontas }}</span></span>
    </a>
  </nav>
</template>

<style scoped>
/* Sem barra no topo (referência de espaço): a marca abre a lateral e o tema fica no pé dela. */
.moldura { display: flex; }

.menu {
  position: sticky;
  top: 0;
  flex: none;
  display: flex;
  flex-direction: column;
  width: var(--nav);
  height: 100vh;
  padding: var(--e4) var(--e3);
  overflow-y: auto;
  border-right: 1px solid var(--borda);
  background: var(--superficie);
}
.menu nav { display: grid; gap: 2px; }

#nav-contexto:not(:empty) {
  margin-top: var(--e4);
  padding-top: var(--e3);
  border-top: 1px solid var(--borda);
}

.marca {
  display: flex;
  align-items: center;
  gap: var(--e3);
  margin: 0 0 var(--e4);
  padding: var(--e2);
  color: var(--texto);
  text-decoration: none;
}
.marca-icone {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--borda);
  border-radius: var(--raio-p);
  background: var(--fundo);
}
.marca-texto { display: grid; line-height: 1.2; }
.marca-sobre { font-size: 10.5px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; color: var(--texto-fraco); }
.marca-nome { font-family: var(--fonte-titulo); font-size: 18px; font-weight: 500; }

.rodape {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding: var(--e3) 0 0 var(--e2);
  color: var(--texto-fraco);
  font-size: 12px;
}

.conteudo {
  flex: 1;
  min-width: 0;
  min-height: 100vh;
}

.contador {
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  padding: 3px 6px;
  border-radius: 999px;
  background: var(--verde-fundo);
  color: var(--verde);
}

.abas-celular { display: none; }
.veu { display: none; }

@media (max-width: 899px) {
  /* A lateral sai da tela e vira gaveta, aberta pelo item Menu da barra do pé. */
  .menu {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 40;
    width: min(300px, 85vw);
    height: auto;
    transform: translateX(-100%);
    visibility: hidden;
    transition: transform 160ms ease, visibility 160ms;
  }
  .menu.aberta { transform: none; visibility: visible; }
  .veu {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 39;
    background: rgb(0 0 0 / 0.45);
  }

  .abas-celular {
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 30;
    display: grid;
    grid-template-columns: repeat(5, 1fr);
    padding: var(--e1) var(--e2) calc(var(--e1) + env(safe-area-inset-bottom));
    background: var(--fundo);
    border-top: 1px solid var(--borda);
  }
  .abas-celular a, .abas-celular button {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 6px 0;
    font-size: 11.5px;
    font-weight: 500;
    color: var(--texto-fraco);
    text-decoration: none;
    border: 0;
    background: transparent;
    border-radius: var(--raio-p);
    cursor: pointer;
  }
  .abas-celular a.ativa { color: var(--texto); background: var(--superficie-2); }
}
</style>
