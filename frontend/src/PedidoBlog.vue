<script setup>
import { computed, ref } from 'vue'
import { SITUACOES_BLOG } from './blog.js'
import { textoComCodigo } from './markdown.js'

// Blog (T62): o resultado de um pedido — as sugestões de artigo ou os tópicos de um título — com as
// marcas (guardar, descartar, escrevi) e o "Pedir os tópicos deste".
const props = defineProps({ pedido: { type: Object, required: true } })
const emit = defineEmits(['marcar', 'topicos'])

const r = computed(() => props.pedido.resultado || {})
const marca = (n) => props.pedido.marcas?.[n]?.estado || ''
const html = (t) => textoComCodigo(t || '')

const escrevendo = ref(-1)
const link = ref('')
function abrirEscrevi(n) {
  escrevendo.value = n
  link.value = props.pedido.marcas?.[n]?.link || ''
}
function confirmarEscrevi(n) {
  emit('marcar', props.pedido, n, 'escrita', link.value)
  escrevendo.value = -1
}
const alternar = (n, estado) => emit('marcar', props.pedido, n, marca(n) === estado ? '' : estado, '')
const abertas = ref(new Set())
const descartadaAberta = (n) => abertas.value.has(n)
function abrirDescartada(n) {
  abertas.value = new Set([...abertas.value, n])
}
</script>

<template>
  <div class="pedido-blog pilha">
    <p v-if="r.notas" class="suave sem-margem" v-html="html(r.notas)" />

    <!-- sugestões de artigo -->
    <template v-if="pedido.modo === 'sugerir'">
      <article v-for="(s, n) in r.sugestoes || []" :key="n" class="cartao cheio sugestao" :class="{ descartada: marca(n) === 'descartada' && !descartadaAberta(n) }">
        <div class="linha entre">
          <h3 class="sem-margem" v-html="html(s.titulo)" />
          <span v-if="marca(n) === 'guardada'" class="etiqueta azul">guardada</span>
          <span v-else-if="marca(n) === 'escrita'" class="etiqueta verde">escrita</span>
          <button v-else-if="marca(n) === 'descartada'" class="botao discreto pequeno" @click="descartadaAberta(n) ? alternar(n, 'descartada') : abrirDescartada(n)">
            {{ descartadaAberta(n) ? 'Desfazer descarte' : 'descartada · ver' }}
          </button>
        </div>
        <template v-if="marca(n) !== 'descartada' || descartadaAberta(n)">
          <p v-if="s.alternativas?.length" class="pequeno sem-margem">ou: {{ s.alternativas.join(' · ') }}</p>
          <p class="tese sem-margem" v-html="html(s.tese)" />
          <dl class="campos-blog">
            <dt>Para quem</dt><dd v-html="html(s.leitor)" />
            <dt>Formato</dt><dd v-html="html(s.formato)" />
            <dt>Horizonte</dt><dd v-html="html(s.horizonte)" />
            <dt>Começo</dt><dd v-html="html(s.arco?.comeco)" />
            <dt>Meio</dt><dd v-html="html(s.arco?.meio)" />
            <dt>Fim</dt><dd v-html="html(s.arco?.fim)" />
            <dt>Exemplo central</dt><dd v-html="html(s.exemplo_central)" />
            <template v-if="s.fora"><dt>Deixar de fora</dt><dd v-html="html(s.fora)" /></template>
          </dl>
          <div v-if="s.topicos?.length">
            <span class="rotulo">Tópicos</span>
            <ul class="topicos-blog">
              <li v-for="(t, i) in s.topicos" :key="i">
                <span class="etiqueta" :class="t.obrigatorio ? 'ambar' : ''">{{ t.obrigatorio ? 'obrigatório' : 'opcional' }}</span>
                <div><span v-html="html(t.topico)" /><div class="suave pequeno" v-html="html(t.porque)" /></div>
              </li>
            </ul>
          </div>
          <div v-if="s.base?.length" class="linha">
            <span class="pequeno">Base no item:</span>
            <span v-for="(b, i) in s.base" :key="i" class="etiqueta" :class="SITUACOES_BLOG[b.situacao]?.[1]" :title="SITUACOES_BLOG[b.situacao]?.[0]">
              {{ b.fonte }} · {{ SITUACOES_BLOG[b.situacao]?.[0] }}
            </span>
          </div>
          <p v-if="s.fontes?.length" class="pequeno sem-margem">Fontes para conferir: <span v-html="html(s.fontes.join('; '))" /></p>
          <div class="linha acoes-sugestao">
            <button class="botao" @click="emit('topicos', s.titulo, s.tese)">Pedir os tópicos deste</button>
            <button class="botao discreto" @click="alternar(n, 'guardada')">{{ marca(n) === 'guardada' ? 'Tirar das guardadas' : 'Guardar' }}</button>
            <button class="botao discreto" @click="abrirEscrevi(n)">{{ marca(n) === 'escrita' ? 'Trocar o link' : 'Escrevi' }}</button>
            <button v-if="marca(n) !== 'descartada'" class="botao discreto" @click="alternar(n, 'descartada')">Descartar</button>
          </div>
          <div v-if="escrevendo === n" class="linha">
            <input v-model="link" class="campo" type="url" placeholder="Link do post (opcional)" @keyup.enter="confirmarEscrevi(n)" />
            <button class="botao primario" @click="confirmarEscrevi(n)">Marcar como escrita</button>
          </div>
        </template>
      </article>
    </template>

    <!-- tópicos de um título -->
    <article v-else class="cartao cheio sugestao">
      <div class="linha entre">
        <h3 class="sem-margem">{{ pedido.titulo }}</h3>
        <span v-if="marca(0) === 'guardada'" class="etiqueta azul">guardada</span>
        <span v-else-if="marca(0) === 'escrita'" class="etiqueta verde">escrita</span>
      </div>
      <p v-if="pedido.nota" class="pequeno sem-margem">O que você quer dizer: {{ pedido.nota }}</p>
      <p class="tese sem-margem" v-html="html(r.tese)" />
      <p v-if="r.titulo_melhor" class="pequeno sem-margem">Título mais preciso: <strong v-html="html(r.titulo_melhor)" /></p>
      <div v-if="r.topicos?.length">
        <span class="rotulo">Tópicos, na ordem do texto</span>
        <ul class="topicos-blog">
          <li v-for="(t, i) in r.topicos" :key="i">
            <span class="etiqueta" :class="t.obrigatorio ? 'ambar' : ''">{{ t.obrigatorio ? 'obrigatório' : 'opcional' }}</span>
            <div>
              <span v-html="html(t.topico)" />
              <span class="etiqueta pequena" :class="SITUACOES_BLOG[t.situacao]?.[1]">{{ SITUACOES_BLOG[t.situacao]?.[0] }}</span>
              <div class="suave pequeno" v-html="html(t.porque)" />
              <div v-if="t.onde" class="fraco pequeno" v-html="html(t.onde)" />
            </div>
          </li>
        </ul>
      </div>
      <dl class="campos-blog">
        <template v-if="r.arco"><dt>Começo</dt><dd v-html="html(r.arco.comeco)" /><dt>Meio</dt><dd v-html="html(r.arco.meio)" /><dt>Fim</dt><dd v-html="html(r.arco.fim)" /></template>
        <template v-if="r.fora"><dt>Deixar de fora</dt><dd v-html="html(r.fora)" /></template>
      </dl>
      <div v-if="r.armadilhas?.length">
        <span class="rotulo">Armadilhas comuns no tema</span>
        <ul class="itens"><li v-for="(x, i) in r.armadilhas" :key="i" v-html="html(x)" /></ul>
      </div>
      <div v-if="r.perguntas?.length">
        <span class="rotulo">Perguntas que o leitor vai ter</span>
        <ul class="itens"><li v-for="(x, i) in r.perguntas" :key="i" v-html="html(x)" /></ul>
      </div>
      <p v-if="r.fontes?.length" class="pequeno sem-margem">Fontes para conferir: <span v-html="html(r.fontes.join('; '))" /></p>
      <div class="linha acoes-sugestao">
        <button class="botao discreto" @click="alternar(0, 'guardada')">{{ marca(0) === 'guardada' ? 'Tirar das guardadas' : 'Guardar' }}</button>
        <button class="botao discreto" @click="abrirEscrevi(0)">{{ marca(0) === 'escrita' ? 'Trocar o link' : 'Escrevi' }}</button>
      </div>
      <div v-if="escrevendo === 0" class="linha">
        <input v-model="link" class="campo" type="url" placeholder="Link do post (opcional)" @keyup.enter="confirmarEscrevi(0)" />
        <button class="botao primario" @click="confirmarEscrevi(0)">Marcar como escrita</button>
      </div>
    </article>
  </div>
</template>

<style scoped>
.sem-margem { margin: 0; }
.sugestao { display: grid; gap: var(--e3); }
.sugestao h3 { font-size: 17px; line-height: 1.35; }
.sugestao.descartada { opacity: 0.6; }
.tese { font-size: 15.5px; line-height: 1.55; font-weight: 500; }
.campos-blog { display: grid; grid-template-columns: max-content 1fr; gap: var(--e2) var(--e4); margin: 0; font-size: 15px; line-height: 1.5; }
.campos-blog dt { color: var(--texto-fraco); font-size: 13px; font-weight: 500; padding-top: 2px; }
.campos-blog dd { margin: 0; }
.topicos-blog { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--e2); font-size: 15px; line-height: 1.5; }
.topicos-blog li { display: grid; grid-template-columns: 96px 1fr; gap: var(--e3); align-items: start; }
.topicos-blog .etiqueta { justify-self: start; }
.etiqueta.pequena { margin-left: var(--e2); }
.itens { margin: 0; padding-left: var(--e5); display: grid; gap: var(--e1); font-size: 15px; line-height: 1.5; }
.acoes-sugestao { padding-top: var(--e2); border-top: 1px solid var(--borda); }
@media (max-width: 560px) {
  .campos-blog { grid-template-columns: 1fr; gap: 0; }
  .campos-blog dd { margin-bottom: var(--e2); }
  .topicos-blog li { grid-template-columns: 1fr; gap: var(--e1); }
}
</style>
