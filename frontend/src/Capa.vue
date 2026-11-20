<script setup>
import { computed } from 'vue'
import { capaDaTrilha } from './capa.js'

// A capa gerada da trilha (T47): cor do assunto, um desenho de fundo e a sigla. "pequena" é o
// quadrado das listas compactas.
const props = defineProps({ titulo: { type: String, default: '' }, pequena: Boolean })
const capa = computed(() => capaDaTrilha(props.titulo))
</script>

<template>
  <div class="capa" :class="{ pequena }" :style="{ '--matiz': capa.matiz }" aria-hidden="true">
    <svg class="desenho" viewBox="0 0 200 120" preserveAspectRatio="xMidYMid slice">
      <template v-if="capa.desenho === 'pontos'">
        <template v-for="l in 7" :key="l">
          <circle v-for="c in 12" :key="c" :cx="c * 17 - 4" :cy="l * 17 - 6" :r="1.6 + ((l + c) % 3) * 0.6" />
        </template>
      </template>
      <template v-else-if="capa.desenho === 'arcos'">
        <circle v-for="i in 7" :key="i" cx="185" cy="10" :r="i * 22" class="traco" />
      </template>
      <template v-else-if="capa.desenho === 'diagonais'">
        <line v-for="i in 16" :key="i" :x1="i * 18 - 120" y1="130" :x2="i * 18" y2="-10" class="traco" />
      </template>
      <template v-else>
        <rect v-for="i in 9" :key="i" :x="92 + i * 12" :y="120 - ((i * 37) % 70) - 20" width="7" :height="((i * 37) % 70) + 20" rx="2" />
      </template>
    </svg>
    <span class="sigla">{{ capa.sigla }}</span>
  </div>
</template>

<style scoped>
.capa {
  position: relative;
  overflow: hidden;
  aspect-ratio: 16 / 7;
  container-type: inline-size;
  border-radius: var(--raio);
  background: linear-gradient(135deg, hsl(var(--matiz) var(--capa-fundo)), hsl(calc(var(--matiz) + 25) var(--capa-fundo)));
  color: hsl(var(--matiz) var(--capa-tinta));
}
/* O desenho some perto da sigla (canto de baixo à esquerda), para ela ler bem. */
.desenho { position: absolute; inset: 0; width: 100%; height: 100%; fill: currentColor; opacity: 0.32;
  -webkit-mask-image: linear-gradient(to top right, transparent 22%, #000 58%); mask-image: linear-gradient(to top right, transparent 22%, #000 58%); }
.desenho .traco { fill: none; stroke: currentColor; stroke-width: 1.4; }
.sigla {
  position: absolute;
  left: 14px;
  bottom: 8px;
  font-family: var(--fonte-titulo);
  font-size: clamp(20px, 13cqw, 38px);
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1;
}
.capa.pequena { aspect-ratio: 1; width: 48px; flex: none; border-radius: var(--raio-p); }
.capa.pequena .sigla { left: 0; right: 0; bottom: auto; top: 50%; transform: translateY(-50%); text-align: center; font-size: 16px; }
.capa.pequena .desenho { opacity: 0.22; -webkit-mask-image: none; mask-image: none; }
</style>
