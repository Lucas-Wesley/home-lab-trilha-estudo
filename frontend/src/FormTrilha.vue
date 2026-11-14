<script setup>
import { PROFUNDIDADES } from './rotulos.js'

// Os dados da trilha, para criar e para editar. "instrucoes" são as instruções da trilha (T13):
// valem para toda chamada à IA desta trilha, a partir da fase 3.
const dados = defineModel({ type: Object, required: true })
</script>

<template>
  <div class="pilha">
    <div>
      <label class="rotulo" for="t-titulo">Título</label>
      <input id="t-titulo" v-model="dados.titulo" class="campo" placeholder="Ex.: Home lab: rede" required />
    </div>
    <div>
      <label class="rotulo" for="t-objetivo">Ao final, você será capaz de…</label>
      <textarea id="t-objetivo" v-model="dados.objetivo" class="campo" rows="2" placeholder="O objetivo, com um verbo: explicar, configurar, diagnosticar, comparar…" />
    </div>
    <div>
      <label class="rotulo" for="t-recorte">Recorte <span class="fraco">(o que entra e o que fica de fora)</span></label>
      <textarea id="t-recorte" v-model="dados.recorte" class="campo" rows="2" />
    </div>
    <div class="campos">
      <div>
        <label class="rotulo" for="t-partida">Nível de partida</label>
        <input id="t-partida" v-model="dados.nivel_partida" class="campo" placeholder="Ex.: dev, sem base de redes" />
      </div>
      <div>
        <label class="rotulo" for="t-alvo">Nível alvo</label>
        <input id="t-alvo" v-model="dados.nivel_alvo" class="campo" placeholder="Ex.: diagnosticar a própria rede" />
      </div>
      <div>
        <label class="rotulo" for="t-prof">Profundidade</label>
        <select id="t-prof" v-model.number="dados.profundidade" class="campo">
          <option v-for="(nome, n) in PROFUNDIDADES" :key="n" :value="Number(n)">{{ n }} · {{ nome }}</option>
        </select>
      </div>
    </div>
    <div>
      <label class="rotulo" for="t-instr">Instruções da trilha <span class="fraco">(opcional)</span></label>
      <textarea
        id="t-instr"
        v-model="dados.instrucoes"
        class="campo"
        placeholder="O que você quer com isto: foco, o que já sabe, o que evitar, fontes preferidas, exemplos que quer ver. Vale para tudo o que a IA fizer nesta trilha."
      />
    </div>
  </div>
</template>
