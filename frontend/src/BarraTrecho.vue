<script setup>
// BarraTrecho: o que a Videoaula e o TrechoYouTube têm em igual — o título com o localizador,
// a barra que mostra o trecho, a linha "faltam…" e o aviso "Trecho visto ✓" (T39/T44).
import { computed } from 'vue'
import { estadoDoTrecho, pctDoTrecho } from './trechoYoutube.js'
import { segundosParaTempo } from './videoaula.js'

const props = defineProps({
  titulo: { type: String, default: '' },
  inicio: { type: Number, required: true },
  fim: { type: Number, required: true },
  posicao: { type: Number, default: 0 },
  aviso: { type: Boolean, default: false }, // mostra "Trecho visto ✓" sem parar o vídeo
  proxima: { type: String, default: '' }, // link da próxima etapa, se houver
})
const emit = defineEmits(['continuar'])

// A barra é a janela do trecho: 0% = início, 100% = fim; o marcador mostra a posição do vídeo.
const pct = computed(() => pctDoTrecho(props.posicao, props.inicio, props.fim))
const estado = computed(() => estadoDoTrecho(props.posicao, props.inicio, props.fim))
</script>

<template>
  <div class="linha barra">
    <span class="titulo-video">{{ titulo }}</span>
    <span class="pequeno localizador">{{ segundosParaTempo(inicio) }} → {{ segundosParaTempo(fim) }}</span>
  </div>
  <div class="trilho" role="progressbar" :aria-valuenow="Math.round(pct)" aria-valuemin="0" aria-valuemax="100">
    <div class="preenchido" :style="{ width: pct + '%' }" />
  </div>
  <div class="linha barra">
    <span class="pequeno">{{ estado.passou ? 'Trecho visto' : `faltam ${segundosParaTempo(estado.faltam)}` }}<template v-if="estado.antes"> · o vídeo está antes do trecho</template></span>
    <span class="espaco" />
    <slot />
  </div>

  <p v-if="aviso" class="aviso visto">
    <strong>Trecho visto ✓</strong>
    <button class="botao discreto" type="button" @click="emit('continuar')">Continuar assistindo</button>
    <a v-if="proxima" class="botao" :href="proxima">Ir para a próxima etapa</a>
  </p>
</template>

<style scoped>
.barra { margin-top: var(--e2); }
.titulo-video { font-size: 14px; color: var(--texto-suave); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1 1 200px; }
.localizador { font-variant-numeric: tabular-nums; white-space: nowrap; }
.espaco { flex: 1; }
.trilho { height: 6px; margin-top: var(--e2); background: var(--superficie-2); border-radius: 3px; overflow: hidden; }
.preenchido { height: 100%; background: var(--azul); border-radius: 3px; transition: width .4s linear; }
.aviso.visto { display: flex; align-items: center; gap: var(--e3); flex-wrap: wrap; }
</style>
