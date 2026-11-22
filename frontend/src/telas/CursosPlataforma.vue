<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import AbasPagina from '../AbasPagina.vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { ESTADOS_AULA, agruparAulas, temDownload } from '../aulasCurso.js'
import { tamanho } from '../pastas.js'
import { segundosParaTempo } from '../videoaula.js'
import { link } from '../rotas.js'

// Cursos (T50, T55): as aulas que a extensão do Chrome mandou baixar, em vídeo, na qualidade máxima
// (ou só o texto, na aula sem vídeo). Plataforma › curso › módulo; um download por vez, no servidor.
// Cada curso pode virar uma trilha (uma etapa por aula, no grupo da plataforma).
const ABAS = [
  { id: 'videos', nome: 'Vídeos' },
  { id: 'arquivos', nome: 'Meus arquivos' },
  { id: 'cursos', nome: 'Cursos' },
]
const mudarAba = (id) => {
  location.hash = id === 'videos' ? '#/entrada' : `#/entrada/${id}`
}

const dados = ref(null)
const erro = ref('')
let timer = null

async function buscar() {
  try {
    dados.value = await api('/api/cursos/aulas')
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
  clearTimeout(timer)
  if (temDownload(dados.value?.aulas || [])) timer = setTimeout(buscar, 3000)
}

const grupos = computed(() => agruparAulas(dados.value?.aulas || [], dados.value?.plataformas || {}, dados.value?.trilhas || []))

const criando = ref('')
async function criarTrilha(c) {
  criando.value = c.chave
  try {
    const r = await api('/api/cursos/trilha', { metodo: 'POST', corpo: { plataforma: c.plataforma, curso: c.nome }, tempo: 60000 })
    location.hash = link('trilhas', r.trilha_id)
  } catch (e) {
    erro.value = e.message
  } finally {
    criando.value = ''
  }
}

async function tentar(a) {
  try {
    await api(`/api/cursos/aulas/${a.id}/tentar`, { metodo: 'POST' })
    await buscar()
  } catch (e) {
    erro.value = e.message
  }
}

async function apagar(a) {
  const oQue = a.estado === 'pronta' ? 'a aula e o vídeo' : 'a aula'
  if (!confirm(`Apagar ${oQue} "${a.titulo}"?`)) return
  try {
    await api(`/api/cursos/aulas/${a.id}`, { metodo: 'DELETE' })
    await buscar()
  } catch (e) {
    erro.value = e.message
  }
}

function meta(a) {
  const peso = a.estado === 'pronta' ? tamanho(a.tamanho) : a.tamanho_estimado ? `~${tamanho(a.tamanho_estimado)}` : ''
  return [a.qualidade, a.duracao ? segundosParaTempo(a.duracao) : '', peso].filter(Boolean).join(' · ')
}

onMounted(buscar)
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <Pagina>
    <template #topo>
      <div class="topo-pagina">
        <h1>Entrada</h1>
        <div class="acoes">
          <span v-if="dados?.livre" class="pequeno fraco">{{ tamanho(dados.livre) }} livres na pasta dos cursos</span>
        </div>
      </div>
      <AbasPagina class="abas-entrada" :abas="ABAS" model-value="cursos" rotulo="Partes da Entrada" @update:model-value="mudarAba" />
    </template>

    <p class="explicacao">
      As aulas da Branas.io, LINUXtips, Full Cycle, Tech Leads Club e Chef Enzo Neto, em vídeo, na qualidade máxima. Na página da
      aula, clique no ícone da extensão do Homelab e em <strong>Baixar esta aula</strong> (ou no curso inteiro, ou, no Chef Enzo
      Neto, na escola inteira): ela vem para cá, no curso e no módulo dela, e o servidor baixa uma por vez. Cada curso pode virar
      uma trilha, com uma etapa por aula.
    </p>
    <p v-if="erro" class="aviso">{{ erro }}</p>
    <p v-if="dados && !grupos.length" class="fraco">Nenhuma aula ainda.</p>

    <section v-for="p in grupos" :key="p.chave" class="plataforma">
      <h2 class="rotulo-secao">{{ p.nome }}</h2>
      <div v-for="c in p.cursos" :key="c.chave" class="quadro curso">
        <div class="quadro-topo">
          <h3 class="nome-curso">{{ c.nome }}</h3>
          <span class="pequeno fraco">{{ [c.total > c.soTexto ? `${c.baixadas} de ${c.total - c.soTexto} baixadas` : '', c.soTexto ? `${c.soTexto} só texto` : '', c.tamanho ? tamanho(c.tamanho) : ''].filter(Boolean).join(' · ') }}</span>
          <span class="espaco" />
          <a v-if="c.trilha" class="botao discreto" :href="link('trilhas', c.trilha)"><Icone nome="trilhas" :tamanho="15" /> Abrir a trilha</a>
          <button v-else class="botao discreto" :disabled="criando === c.chave" title="Uma etapa por aula, com o vídeo e o texto da página" @click="criarTrilha(c)">
            {{ criando === c.chave ? 'Criando…' : 'Criar trilha deste curso' }}
          </button>
        </div>
        <div v-for="m in c.modulos" :key="m.chave" class="modulo">
          <p v-if="m.nome" class="nome-modulo"><Icone nome="pasta" :tamanho="14" /> {{ m.nome }}</p>
          <ul class="lista linhas">
            <li v-for="a in m.aulas" :key="a.id" class="linha-aula">
              <span class="icone-aula" :class="{ vazia: a.estado !== 'pronta' }"><Icone :nome="a.estado === 'texto' ? 'livro' : 'video'" :tamanho="15" /></span>
              <span class="sobre">
                <a v-if="a.estado === 'pronta'" class="nome" :href="`/api/cursos/aulas/${a.id}/video`" target="_blank" rel="noopener">{{ a.titulo }}</a>
                <span v-else class="nome">{{ a.titulo }}</span>
                <span class="meta">{{ meta(a) }}</span>
                <span v-if="a.estado === 'baixando'" class="progresso mini" :title="`${a.progresso}%`"><span :style="{ width: a.progresso + '%' }" /></span>
                <span v-if="a.estado === 'erro'" class="erro-aula">{{ a.erro }}</span>
              </span>
              <span class="etiqueta" :class="ESTADOS_AULA[a.estado]?.cor">
                {{ a.estado === 'baixando' ? `${a.progresso}%` : ESTADOS_AULA[a.estado]?.nome || a.estado }}
              </span>
              <span class="botoes">
                <a v-if="a.estado === 'pronta'" class="botao discreto" :href="`/api/cursos/aulas/${a.id}/video`" target="_blank" rel="noopener">Assistir</a>
                <button v-if="a.estado === 'erro'" class="botao discreto" @click="tentar(a)">Tentar de novo</button>
                <a class="botao discreto icone" :href="a.pagina" target="_blank" rel="noopener" title="Abrir na plataforma" aria-label="Abrir na plataforma"><Icone nome="link" :tamanho="15" /></a>
                <button class="botao discreto icone" title="Apagar" aria-label="Apagar a aula" @click="apagar(a)"><Icone nome="fechar" /></button>
              </span>
            </li>
          </ul>
        </div>
      </div>
    </section>
  </Pagina>
</template>

<style scoped>
.abas-entrada { margin-bottom: var(--e5); }
.plataforma { margin-top: var(--e5); }
.curso { margin-top: var(--e3); }
.nome-curso { font-size: 16px; font-weight: 600; margin: 0; }
.curso .quadro-topo { display: flex; align-items: center; gap: var(--e3); flex-wrap: wrap; }
.espaco { flex: 1; }
.modulo + .modulo { margin-top: var(--e4); }
.nome-modulo { display: flex; align-items: center; gap: var(--e2); margin: var(--e3) 0 var(--e1); font-size: 13.5px; font-weight: 600; color: var(--texto-suave); }
.linha-aula { display: flex; align-items: center; gap: var(--e3); padding: var(--e2) 0; }
.icone-aula { flex: none; display: grid; place-items: center; width: 30px; height: 30px; border-radius: var(--raio-p); background: var(--verde-fundo); color: var(--verde); }
.icone-aula.vazia { background: var(--superficie-2); color: var(--texto-fraco); }
.sobre { flex: 1; min-width: 0; display: grid; gap: 2px; }
.sobre .nome { font-size: 14.5px; font-weight: 500; color: inherit; text-decoration: none; overflow-wrap: anywhere; }
.sobre a.nome:hover { color: var(--link); text-decoration: underline; }
.meta { font-size: 12.5px; color: var(--texto-fraco); }
.progresso.mini { max-width: 240px; margin-top: 2px; }
.erro-aula { font-size: 12.5px; color: var(--vermelho); overflow-wrap: anywhere; }
.botoes { display: flex; align-items: center; gap: 2px; flex: none; }
@media (max-width: 599px) {
  .linha-aula { flex-wrap: wrap; }
  .botoes { width: 100%; justify-content: flex-end; }
}
</style>
