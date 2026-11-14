<script setup>
import { computed, ref } from 'vue'
import { DIAS, dataCurta } from './rotulos.js'

// Meta (T10): dias da semana + minutos por dia, ou a data final (o servidor calcula os minutos).
const props = defineProps({
  meta: { type: Object, required: true },
  restante: { type: Number, default: 0 }, // minutos que faltam na trilha
  erro: { type: String, default: '' },
})
const emit = defineEmits(['salvar', 'fechar'])

const hojeIso = () => {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
const dias = ref(props.meta.dias?.length ? [...props.meta.dias] : [1, 2, 3, 4, 5])
const modo = ref(props.meta.fim && !props.meta.minutos ? 'fim' : 'minutos')
const minutos = ref(props.meta.minutos || 45)
const fim = ref(props.meta.fim || '')
const inicio = ref(props.meta.inicio && props.meta.inicio >= hojeIso() ? props.meta.inicio : hojeIso())

function alternar(d) {
  dias.value = dias.value.includes(d) ? dias.value.filter((x) => x !== d) : [...dias.value, d].sort()
}

function* diasDeEstudo(desde) {
  const [a, m, d] = desde.split('-').map(Number)
  const data = new Date(a, m - 1, d)
  for (let i = 0; i < 3650; i++) {
    if (dias.value.includes(data.getDay())) yield new Date(data)
    data.setDate(data.getDate() + 1)
  }
}
const iso = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

// Prévia: quando termina (pelos minutos) ou quantos minutos por dia (pela data final).
const previa = computed(() => {
  if (!dias.value.length || !props.restante) return ''
  if (modo.value === 'minutos') {
    if (!minutos.value) return ''
    const n = Math.ceil(props.restante / minutos.value)
    let i = 0
    for (const d of diasDeEstudo(inicio.value)) if (++i === n) return `Termina por volta de ${dataCurta(iso(d))} (${n} dias de estudo).`
    return ''
  }
  if (!fim.value) return ''
  let n = 0
  for (const d of diasDeEstudo(inicio.value)) {
    if (iso(d) > fim.value) break
    n++
  }
  if (!n) return 'Não há dia de estudo até essa data.'
  return `Cerca de ${Math.ceil(props.restante / n / 5) * 5} min por dia, em ${n} dias de estudo.`
})

function salvar() {
  emit('salvar', {
    dias: dias.value,
    minutos: modo.value === 'minutos' ? Number(minutos.value) : 0,
    fim: modo.value === 'fim' ? fim.value : '',
    inicio: inicio.value,
  })
}
</script>

<template>
  <div class="dialogo-fundo" @click.self="emit('fechar')">
    <form class="painel pilha" role="dialog" aria-label="Meta de estudo" @submit.prevent="salvar">
      <h3>Meta de estudo</h3>
      <div>
        <span class="rotulo">Dias da semana</span>
        <div class="dias">
          <button v-for="(nome, d) in DIAS" :key="d" type="button" class="botao" :class="{ primario: dias.includes(d) }" :aria-pressed="dias.includes(d)" @click="alternar(d)">{{ nome }}</button>
        </div>
      </div>
      <div class="abas">
        <button type="button" :class="{ ativa: modo === 'minutos' }" @click="modo = 'minutos'">Minutos por dia</button>
        <button type="button" :class="{ ativa: modo === 'fim' }" @click="modo = 'fim'">Terminar até</button>
      </div>
      <div class="campos">
        <div v-if="modo === 'minutos'">
          <label class="rotulo" for="m-min">Minutos por dia</label>
          <input id="m-min" v-model.number="minutos" class="campo" type="number" min="5" max="600" step="5" />
        </div>
        <div v-else>
          <label class="rotulo" for="m-fim">Data final</label>
          <input id="m-fim" v-model="fim" class="campo" type="date" :min="inicio" />
        </div>
        <div>
          <label class="rotulo" for="m-ini">Começar em</label>
          <input id="m-ini" v-model="inicio" class="campo" type="date" :min="hojeIso()" />
        </div>
      </div>
      <p v-if="previa" class="ajuda">{{ previa }}</p>
      <p v-if="erro" class="aviso">{{ erro }}</p>
      <div class="linha">
        <button class="botao primario" type="submit" :disabled="!dias.length">Salvar e montar o cronograma</button>
        <button class="botao discreto" type="button" @click="emit('fechar')">Cancelar</button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.dias { display: flex; gap: var(--e1); flex-wrap: wrap; }
.dias .botao { min-width: 44px; padding: 0 var(--e2); }
</style>
