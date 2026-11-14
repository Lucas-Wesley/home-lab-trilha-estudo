<script setup>
import { onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'

// Leitura de um documento da trilha (#/trilhas/3/documentos/02-wifi#canais).
const props = defineProps({
  trilhaId: { type: Number, required: true },
  doc: { type: String, required: true },
  ancora: { type: String, default: '' },
})

const t = ref(null)
const d = ref(null)
const erro = ref('')
const titulos = ref([])
const base = `#/trilhas/${props.trilhaId}/documentos/`

onMounted(async () => {
  try {
    ;[t.value, d.value] = await Promise.all([
      api(`/api/trilhas/${props.trilhaId}`),
      api(`/api/trilhas/${props.trilhaId}/documentos/${encodeURIComponent(props.doc)}`),
    ])
  } catch (e) {
    erro.value = e.status === 404 ? 'Documento não encontrado nesta trilha.' : e.message
  }
})
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas v-if="t" :trilha="t" :documento-atual="doc" /></template>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <template v-else-if="t && d">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('trilhas')">Trilhas</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <a :href="link('trilhas', t.id)">{{ t.titulo }}</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <a :href="link('trilhas', t.id) + '#documentos'">Documentos</a>
      </nav>
      <Leitor :texto="d.conteudo" :ancora="ancora" :base-docs="base" @titulos="titulos = $event" />
    </template>

    <template #lateral>
      <ul v-if="d && titulos.length" class="sumario">
        <li v-for="h in titulos" :key="h.id" :class="{ sub: h.nivel === 3 }">
          <a :href="`${base}${d.slug}#${h.id}`">{{ h.texto }}</a>
        </li>
      </ul>
    </template>
  </Pagina>
</template>
