<script setup>
import { computed, ref } from 'vue'
import FormTrilha from '../FormTrilha.vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { enviarDocumentos } from '../documentos.js'
import { link } from '../rotas.js'
import { PROFUNDIDADES } from '../rotulos.js'

// As quatro entradas (DECISOES.md §3). A partir de material chega na fase 6.
const entradas = [
  { id: 'ia', icone: 'trilhas', titulo: 'A IA monta', texto: 'Pré-pesquisa, entrevista com perguntas para este tipo de assunto, mapa completo e você escolhe o recorte.', fase: '' },
  { id: 'lista', icone: 'menu', titulo: 'Eu tenho a lista', texto: 'Você dá a ordem; a IA só aponta pré-requisitos que faltam, e você aceita ou não.', fase: '' },
  { id: 'manual', icone: 'mais', titulo: 'Montar à mão', texto: 'Sem IA: você escreve módulos e etapas, na ordem que quiser.', fase: '' },
  { id: 'material', icone: 'livro', titulo: 'A partir de material', texto: 'Um livro da Biblioteca, Temas do Acervo ou documentos próprios viram a espinha dorsal.', fase: 'fase 6' },
]
const entrada = ref('ia')
const dados = ref({ titulo: '', objetivo: '', recorte: '', nivel_partida: '', nivel_alvo: '', profundidade: 3, instrucoes: '' })
// Com IA (fase 4): o tema e o que você quer, em texto livre, e o tamanho da entrevista ou a lista.
const comIA = computed(() => entrada.value !== 'manual')
const ia = ref({ tema: '', pedido: '', entrevista: 'padrao', lista: '', profundidade: 3, instrucoes: '' })
const TAMANHOS = [
  { id: 'rapida', nome: 'Rápida', texto: '2 a 3 perguntas' },
  { id: 'padrao', nome: 'Padrão', texto: '5 a 8 perguntas, em 1 ou 2 rodadas' },
  { id: 'profunda', nome: 'Profunda', texto: 'até ~15 perguntas, em rodadas' },
]
const itensLista = computed(() => ia.value.lista.split('\n').filter((l) => l.trim()).length)
const podeCriar = computed(() => {
  if (!comIA.value) return !!dados.value.titulo.trim()
  return !!ia.value.tema.trim() && (entrada.value !== 'lista' || itensLista.value >= 2)
})
const erro = ref('')
const enviando = ref(false)

// Documentos próprios valem para qualquer forma de começar: vão junto quando a trilha é criada.
const arquivos = ref([])
function escolher(evento) {
  const novos = [...evento.target.files].filter((f) => !arquivos.value.some((a) => a.name === f.name))
  arquivos.value = [...arquivos.value, ...novos]
  evento.target.value = ''
}

async function criar() {
  erro.value = ''
  enviando.value = true
  try {
    if (comIA.value) return await criarComIA()
    const t = await api('/api/trilhas', { metodo: 'POST', corpo: dados.value })
    if (arquivos.value.length) {
      const { erros } = await enviarDocumentos(t.id, arquivos.value)
      // A trilha já existe: os que falharam podem ser mandados de novo na página dela.
      if (erros.length) alert(`A trilha foi criada, mas alguns documentos não entraram:\n${erros.join('\n')}\n\nDá para mandar de novo na seção Documentos da trilha.`)
    }
    location.hash = link('trilhas', t.id)
  } catch (e) {
    erro.value = e.message
  } finally {
    enviando.value = false
  }
}

// A trilha nasce como rascunho, os documentos entram, e só então a IA começa (ela lê os documentos).
async function criarComIA() {
  const r = await api('/api/criacoes', { metodo: 'POST', corpo: { entrada: entrada.value, ...ia.value } })
  const id = r.criacao.trilha_id
  if (arquivos.value.length) {
    const { erros } = await enviarDocumentos(id, arquivos.value)
    if (erros.length) alert(`Alguns documentos não entraram:\n${erros.join('\n')}\n\nA IA começa sem eles; dá para mandar de novo na página da trilha.`)
  }
  await api(`/api/trilhas/${id}/criacao/gerar`, { metodo: 'POST', corpo: {} })
  location.hash = link('trilhas', id, 'criar')
}
</script>

<template>
  <Pagina>
    <template #nav><NavTrilhas nova /></template>

    <nav class="migalhas" aria-label="Você está em">
      <a :href="link('trilhas')">Trilhas</a>
      <Icone nome="seta-direita" :tamanho="14" />
      <span>Nova trilha</span>
    </nav>
    <h1>Nova trilha</h1>
    <p class="lead">Diga o que quer aprender com as suas palavras. Tudo o que a IA propuser vai ficar aguardando a sua aprovação, e você pode comentar e pedir ajustes em qualquer ponto.</p>

    <div class="grade entradas" role="radiogroup" aria-label="Como começar">
      <button
        v-for="e in entradas"
        :key="e.id"
        class="cartao"
        :class="{ escolhida: entrada === e.id }"
        role="radio"
        :aria-checked="entrada === e.id"
        :disabled="!!e.fase"
        @click="entrada = e.id"
      >
        <div class="linha topo">
          <Icone :nome="e.icone" :tamanho="20" class="entrada-icone" />
          <span v-if="e.fase" class="etiqueta">{{ e.fase }}</span>
        </div>
        <h3>{{ e.titulo }}</h3>
        <p>{{ e.texto }}</p>
      </button>
    </div>

    <form class="pilha formulario" @submit.prevent="criar">
      <FormTrilha v-if="!comIA" v-model="dados" />
      <template v-else>
        <div>
          <label class="rotulo" for="n-tema">O que você quer aprender?</label>
          <input id="n-tema" v-model="ia.tema" class="campo" placeholder="Ex.: AWS; História do Brasil; crase e regência" required />
        </div>
        <div>
          <label class="rotulo" for="n-pedido">O que você quer com isto? <span class="fraco">(foco, o que já sabe, o que evitar, fontes preferidas)</span></label>
          <textarea id="n-pedido" v-model="ia.pedido" class="campo" rows="3" placeholder="Ex.: para o trabalho; já uso Linux e Docker; quero entender o porquê, não decorar comandos. O que estiver aqui a IA não pergunta de novo." />
        </div>
        <div v-if="entrada === 'lista'">
          <label class="rotulo" for="n-lista">A sua lista, na ordem <span class="fraco">(um assunto por linha)</span></label>
          <textarea id="n-lista" v-model="ia.lista" class="campo" rows="7" placeholder="EC2&#10;S3&#10;IAM&#10;VPC" />
          <p class="ajuda">A ordem é sua. A IA só aponta o que falta (um pré-requisito, um item grande demais) e você aceita ou não cada sugestão.</p>
        </div>
        <fieldset v-else class="sem-borda">
          <legend class="rotulo">Tamanho da entrevista</legend>
          <div class="linha tamanhos">
            <label v-for="tm in TAMANHOS" :key="tm.id" class="cartao tamanho" :class="{ escolhida: ia.entrevista === tm.id }">
              <input v-model="ia.entrevista" type="radio" name="entrevista" :value="tm.id" class="so-leitor" />
              <strong>{{ tm.nome }}</strong>
              <span class="pequeno">{{ tm.texto }}</span>
            </label>
          </div>
        </fieldset>
        <div class="campos">
          <div>
            <label class="rotulo" for="n-prof">Profundidade</label>
            <select id="n-prof" v-model.number="ia.profundidade" class="campo">
              <option v-for="(nome, n) in PROFUNDIDADES" :key="n" :value="Number(n)">{{ n }} · {{ nome }}</option>
            </select>
          </div>
        </div>
        <div>
          <label class="rotulo" for="n-instr">Instruções da trilha <span class="fraco">(opcional)</span></label>
          <textarea id="n-instr" v-model="ia.instrucoes" class="campo" rows="2" placeholder="Vale para tudo o que a IA fizer nesta trilha, agora e depois. Ex.: exemplos sempre com o homelab." />
        </div>
      </template>
      <fieldset class="pilha documentos">
        <legend class="rotulo">Documentos próprios (opcional)</legend>
        <p class="pequeno sem-margem">Material seu em Markdown sobre o assunto. Fica guardado na trilha e as etapas podem usar como leitura. Vale para qualquer forma de começar; depois dá para adicionar mais na página da trilha.</p>
        <ul v-if="arquivos.length" class="arquivos">
          <li v-for="(f, i) in arquivos" :key="f.name" class="linha entre">
            <span>{{ f.name }}</span>
            <button type="button" class="botao discreto" @click="arquivos.splice(i, 1)">Tirar</button>
          </li>
        </ul>
        <div>
          <label class="botao escolher">
            <Icone nome="mais" /> Escolher arquivos .md
            <input type="file" accept=".md,text/markdown" multiple class="so-leitor" @change="escolher" />
          </label>
        </div>
      </fieldset>
      <p v-if="erro" class="aviso">{{ erro }}</p>
      <div class="linha">
        <button class="botao primario" type="submit" :disabled="enviando || !podeCriar">{{ comIA ? 'Começar' : 'Criar trilha' }}</button>
        <span class="pequeno">{{
          entrada === 'ia' ? 'A IA pesquisa o assunto e faz as primeiras perguntas. Leva alguns minutos.'
          : entrada === 'lista' ? 'A IA confere a lista e volta com as sugestões. Leva alguns minutos.'
          : 'Depois de criar, você monta os módulos e as etapas.'
        }}</span>
      </div>
    </form>
  </Pagina>
</template>

<style scoped>
.entradas { margin-top: var(--e6); grid-template-columns: repeat(2, minmax(0, 1fr)); max-width: var(--leitura); }
.entradas .cartao { text-align: left; font: inherit; cursor: pointer; }
.entradas .cartao:disabled { cursor: not-allowed; opacity: 0.55; }
.entradas .cartao p { -webkit-line-clamp: 3; }
.entradas .cartao.escolhida { border-color: var(--texto-suave); background: var(--superficie); }
.topo { justify-content: space-between; margin-bottom: var(--e2); }
.entrada-icone { color: var(--texto-suave); }
.formulario { margin-top: var(--e6); max-width: var(--leitura); }
.documentos { border: 0; padding: 0; margin: 0; }
.arquivos { list-style: none; margin: 0; padding: 0; border: 1px solid var(--borda); border-radius: var(--raio); }
.arquivos li { padding: var(--e1) var(--e3); border-bottom: 1px solid var(--borda); font-size: 14px; }
.arquivos li:last-child { border-bottom: 0; }
.escolher { cursor: pointer; }
.sem-borda { border: 0; padding: 0; margin: 0; }
.tamanhos { align-items: stretch; flex-wrap: wrap; }
.tamanho { display: grid; gap: 2px; cursor: pointer; flex: 1 1 160px; padding: var(--e2) var(--e3); }
.tamanho.escolhida { border-color: var(--texto-suave); background: var(--superficie); }
@media (max-width: 599px) {
  .entradas { grid-template-columns: minmax(0, 1fr); }
}
</style>
