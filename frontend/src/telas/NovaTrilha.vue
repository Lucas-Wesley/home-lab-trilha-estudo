<script setup>
import { ref } from 'vue'
import FormTrilha from '../FormTrilha.vue'
import Icone from '../Icone.vue'
import NavTrilhas from '../NavTrilhas.vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { enviarDocumentos } from '../documentos.js'
import { link } from '../rotas.js'

// As quatro entradas (DECISOES.md §3). Na fase 2 só a manual funciona.
const entradas = [
  { id: 'manual', icone: 'mais', titulo: 'Montar à mão', texto: 'Sem IA: você escreve módulos e etapas, na ordem que quiser.', fase: '' },
  { id: 'ia', icone: 'trilhas', titulo: 'A IA monta', texto: 'Pré-pesquisa, entrevista com perguntas para este tipo de assunto, mapa completo e você escolhe o recorte.', fase: 'fase 4' },
  { id: 'lista', icone: 'menu', titulo: 'Eu tenho a lista', texto: 'Você dá a ordem; a IA só aponta pré-requisitos que faltam, e você aceita ou não.', fase: 'fase 4' },
  { id: 'material', icone: 'livro', titulo: 'A partir de material', texto: 'Um livro da Biblioteca, Temas do Acervo ou documentos próprios viram a espinha dorsal.', fase: 'fase 6' },
]
const entrada = ref('manual')
const dados = ref({ titulo: '', objetivo: '', recorte: '', nivel_partida: '', nivel_alvo: '', profundidade: 3, instrucoes: '' })
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
      <FormTrilha v-model="dados" />
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
        <button class="botao primario" type="submit" :disabled="enviando || !dados.titulo.trim()">Criar trilha</button>
        <span class="pequeno">Depois de criar, você monta os módulos e as etapas.</span>
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
@media (max-width: 599px) {
  .entradas { grid-template-columns: minmax(0, 1fr); }
}
</style>
