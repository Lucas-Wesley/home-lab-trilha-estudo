<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../api.js'
import { linhasDaTranscricao, linkNoMinuto } from '../entrada.js'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'
import { ESTADOS_FONTE } from '../rotulos.js'
import { segundosParaTempo } from '../videoaula.js'

const props = defineProps({ fonteId: { type: Number, required: true } })

const fonte = ref(null)
const erro = ref('')
const tentando = ref(false)
let timer = null

function ligarTimer() {
  if (timer) clearInterval(timer)
  timer =
    fonte.value && (fonte.value.estado === 'na_fila' || fonte.value.estado === 'lendo') ? setInterval(buscar, 3000) : null
}

async function buscar() {
  try {
    fonte.value = await api(`/api/entrada/${props.fonteId}`)
    ligarTimer()
  } catch (e) {
    erro.value = e.message
  }
}

async function tentar() {
  tentando.value = true
  try {
    await api(`/api/entrada/${props.fonteId}/tentar`, { metodo: 'POST' })
    await buscar()
  } catch (e) {
    erro.value = e.message
  } finally {
    tentando.value = false
  }
}

const estado = computed(() => ESTADOS_FONTE[fonte.value?.estado] || { nome: fonte.value?.estado, cor: '' })

// Só os blocos [mm:ss] entram na transcrição mostrada (o cabeçalho do arquivo repete o que a tela já
// tem); sem nenhum bloco, mostram-se as linhas como vieram.
const blocos = computed(() => {
  const linhas = linhasDaTranscricao(fonte.value?.transcricao || '')
  const marcadas = linhas.filter((l) => l.marca)
  return marcadas.length ? marcadas : linhas.filter((l) => l.texto.trim())
})

onMounted(buscar)
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <Pagina>
  <div class="pilha">
    <nav class="migalhas">
      <a :href="link('entrada')">Entrada</a>
      <Icone nome="seta-direita" :tamanho="12" />
      <span>{{ fonte?.titulo || 'Fonte' }}</span>
    </nav>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <p v-if="!fonte && !erro" class="fraco">Carregando…</p>

    <template v-if="fonte">
      <header class="cabecalho">
        <div>
          <h1>{{ fonte.titulo || 'Lendo o vídeo…' }}</h1>
          <p class="pequeno">
            <template v-if="fonte.canal">{{ fonte.canal }} · </template>
            <template v-if="fonte.duracao">{{ segundosParaTempo(fonte.duracao) }} · </template>
            <a :href="fonte.url" target="_blank" rel="noopener noreferrer">Abrir no YouTube</a>
          </p>
        </div>
        <span class="etiqueta" :class="estado.cor">{{ estado.nome }}</span>
      </header>

      <template v-if="fonte.estado === 'erro'">
        <p class="aviso">{{ fonte.erro }}</p>
        <div><button class="botao" :disabled="tentando" @click="tentar">Tentar de novo</button></div>
      </template>
      <p v-else-if="fonte.estado !== 'pronta'" class="aviso neutro">
        A fonte está {{ estado.nome }} — a transcrição aparece aqui quando terminar.
      </p>

      <section v-if="fonte.capitulos?.length">
        <h2>Capítulos</h2>
        <ul class="capitulos">
          <li v-for="c in fonte.capitulos" :key="c.inicio">
            <a :href="linkNoMinuto(fonte.video_id, c.inicio)" target="_blank" rel="noopener noreferrer">{{
              segundosParaTempo(c.inicio)
            }}</a>
            {{ c.titulo }}
          </li>
        </ul>
      </section>

      <details v-if="fonte.transcricao" class="transcricao">
        <summary><h2>Transcrição</h2></summary>
        <p v-for="(b, i) in blocos" :key="i" class="bloco">
          <a
            v-if="b.marca"
            class="marca"
            :href="linkNoMinuto(fonte.video_id, b.segundos)"
            target="_blank"
            rel="noopener noreferrer"
            >[{{ b.marca }}]</a
          >
          {{ b.texto }}
        </p>
      </details>

      <section>
        <h2>Fragmentos</h2>
        <p class="fraco">Na próxima versão</p>
      </section>
    </template>
  </div>
  </Pagina>
</template>

<style scoped>
.capitulos {
  list-style: none;
  margin: var(--e3) 0 0;
  padding: 0;
  display: grid;
  gap: var(--e2);
}
.capitulos a {
  font-family: var(--fonte-mono);
  font-size: 13px;
}
.transcricao {
  border: 1px solid var(--borda);
  border-radius: var(--raio);
  padding: var(--e3) var(--e4);
}
.transcricao summary {
  cursor: pointer;
}
.transcricao summary h2 {
  display: inline;
}
.transcricao summary::marker {
  color: var(--texto-fraco);
}
.bloco {
  margin: var(--e3) 0;
  font-size: 14.5px;
  line-height: 1.6;
  color: var(--texto-suave);
}
.bloco .marca {
  font-family: var(--fonte-mono);
  font-size: 12.5px;
}
</style>
