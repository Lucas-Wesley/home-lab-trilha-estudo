<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'
import { slugDoArquivo } from '../slug.js'

// Material próprio em Markdown (ex.: os documentos de estudo do homelab). As etapas apontam para
// um documento e uma seção dele (material do tipo "documento").
const props = defineProps({ rota: { type: Object, required: true } })
const slug = computed(() => props.rota.partes[0] || '')

const lista = ref([])
const doc = ref(null)
const erro = ref('')
const aviso = ref('')
const titulos = ref([])

async function carregar() {
  erro.value = ''
  try {
    lista.value = await api('/api/documentos')
    doc.value = slug.value ? await api(`/api/documentos/${encodeURIComponent(slug.value)}`) : null
  } catch (e) {
    erro.value = e.status === 404 ? 'Documento não encontrado.' : e.message
  }
}
onMounted(carregar)

async function importar(evento) {
  aviso.value = ''
  const arquivos = [...evento.target.files]
  let ok = 0
  for (const f of arquivos) {
    try {
      await api(`/api/documentos/${slugDoArquivo(f.name)}`, { metodo: 'PUT', corpo: { conteudo: await f.text() } })
      ok++
    } catch (e) {
      aviso.value = `${f.name}: ${e.message}`
    }
  }
  evento.target.value = ''
  if (ok) aviso.value = (aviso.value ? aviso.value + ' · ' : '') + `${ok} documento(s) importado(s).`
  await carregar()
}
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <span class="nav-titulo">Documentos</span>
        <a class="nav-item" :class="{ ativo: !slug }" :href="link('documentos')">Todos</a>
        <a v-for="d in lista" :key="d.slug" class="nav-item" :class="{ ativo: slug === d.slug }" :href="link('documentos', d.slug)">{{ d.titulo }}</a>
      </div>
    </template>

    <p v-if="erro" class="aviso">{{ erro }}</p>

    <template v-else-if="doc">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('documentos')">Documentos</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <span>{{ doc.slug }}</span>
      </nav>
      <Leitor :texto="doc.conteudo" :ancora="rota.ancora" @titulos="titulos = $event" />
    </template>

    <template v-else>
      <div class="cabecalho">
        <div>
          <h1>Documentos</h1>
          <p class="lead">Material próprio em Markdown, usado como leitura nas etapas.</p>
        </div>
      </div>
      <div class="linha importar">
        <label class="botao">
          <Icone nome="mais" /> Importar arquivos .md
          <input type="file" accept=".md,text/markdown" multiple class="so-leitor" @change="importar" />
        </label>
        <span class="pequeno">O nome do arquivo vira o nome do documento; importar de novo substitui (o histórico fica no git do conteúdo).</span>
      </div>
      <p v-if="aviso" class="aviso neutro">{{ aviso }}</p>
      <table v-if="lista.length" class="tabela">
        <thead><tr><th>Documento</th><th>Nome</th></tr></thead>
        <tbody>
          <tr v-for="d in lista" :key="d.slug">
            <td><a class="titulo" :href="link('documentos', d.slug)">{{ d.titulo }}</a></td>
            <td class="fraco"><code>{{ d.slug }}</code></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="suave">Nenhum documento ainda.</p>
    </template>

    <template #lateral>
      <ul v-if="doc && titulos.length" class="sumario">
        <li v-for="h in titulos" :key="h.id" :class="{ sub: h.nivel === 3 }">
          <a :href="`#/documentos/${doc.slug}#${h.id}`">{{ h.texto }}</a>
        </li>
      </ul>
    </template>
  </Pagina>
</template>

<style scoped>
.importar { margin: var(--e5) 0 var(--e4); }
.importar label { cursor: pointer; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
</style>
