<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import Icone from './Icone.vue'
import Hoje from './telas/Hoje.vue'
import Entrada from './telas/Entrada.vue'
import Trilhas from './telas/Trilhas.vue'
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
    api('/api/entrada', { metodo: 'POST', corpo: { url } })
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
const telas = { hoje: Hoje, trilhas: Trilhas, entrada: Entrada, ajustes: Ajustes, design: Vitrine }

const abas = [
  { secao: 'hoje', nome: 'Hoje', icone: 'hoje' },
  { secao: 'trilhas', nome: 'Trilhas', icone: 'trilhas' },
  { secao: 'entrada', nome: 'Entrada', icone: 'entrada' },
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
  buscarProntas()
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
      >{{ aba.nome }} <span v-if="aba.secao === 'entrada' && prontas" class="contador">{{ prontas }}</span></a>
    </nav>
    <div class="barra-direita">
      <a class="botao discreto icone" :class="{ ativa: rota.secao === 'ajustes' }" :href="link('ajustes')" title="Ajustes" aria-label="Ajustes">
        <Icone nome="ajustes" />
      </a>
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
      <span>{{ aba.nome }} <span v-if="aba.secao === 'entrada' && prontas" class="contador">{{ prontas }}</span></span>
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
.barra-direita .botao.ativa { background: var(--superficie-2); color: var(--texto); }
.so-celular { display: none; }

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
    grid-template-columns: repeat(3, 1fr);
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
