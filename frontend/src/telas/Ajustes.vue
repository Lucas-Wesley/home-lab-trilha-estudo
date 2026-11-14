<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'

// Ajustes da IA (DECISOES.md §5.1, T5, T6): instruções globais, modelo por tipo de tarefa, as
// regras que a IA segue (editáveis, com "restaurar") e a fila de tarefas.
const props = defineProps({ rota: { type: Object, required: true } })

const ajustes = ref(null)
const regras = ref([])
const tarefas = ref([])
const aviso = ref('')
const salvo = ref('')

async function carregar() {
  try {
    ;[ajustes.value, regras.value, tarefas.value] = await Promise.all([api('/api/ajustes'), api('/api/regras'), api('/api/tarefas')])
  } catch (e) {
    aviso.value = e.message
  }
}

// A fila muda sozinha enquanto há tarefa rodando.
let relogio = 0
// #/ajustes#regras: vai até a seção.
async function irPara(id) {
  await nextTick()
  if (id) document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
watch(() => props.rota.ancora, irPara)
onMounted(async () => {
  await carregar()
  irPara(props.rota.ancora)
  relogio = setInterval(async () => {
    if (tarefas.value.some((t) => t.estado === 'na_fila' || t.estado === 'rodando')) tarefas.value = await api('/api/tarefas').catch(() => tarefas.value)
  }, 4000)
})
onBeforeUnmount(() => clearInterval(relogio))

async function salvarAjustes() {
  aviso.value = salvo.value = ''
  try {
    ajustes.value = await api('/api/ajustes', { metodo: 'PUT', corpo: ajustes.value })
    salvo.value = 'Salvo.'
  } catch (e) {
    aviso.value = e.message
  }
}

// ---------- regras ----------
const regra = ref(null)
const texto = ref('')
const erroRegra = ref('')
async function abrirRegra(arquivo) {
  erroRegra.value = ''
  regra.value = await api(`/api/regras/${arquivo}`)
  texto.value = regra.value.texto
}
const mudou = computed(() => regra.value && texto.value !== regra.value.texto)
async function salvarRegra() {
  erroRegra.value = ''
  try {
    regra.value = await api(`/api/regras/${regra.value.arquivo}`, { metodo: 'PUT', corpo: { texto: texto.value } })
    regras.value = await api('/api/regras')
  } catch (e) {
    erroRegra.value = e.message
  }
}
async function restaurar() {
  if (!confirm('Voltar à regra original? A sua edição se perde.')) return
  regra.value = await api(`/api/regras/${regra.value.arquivo}`, { metodo: 'DELETE' })
  texto.value = regra.value.texto
  regras.value = await api('/api/regras')
}

// ---------- fila ----------
const ESTADOS = {
  na_fila: { nome: 'na fila', cor: '' },
  rodando: { nome: 'rodando', cor: 'azul' },
  concluida: { nome: 'pronta', cor: 'verde' },
  falhou: { nome: 'falhou', cor: 'vermelho' },
  cancelada: { nome: 'cancelada', cor: '' },
}
async function cancelar(t) {
  try {
    await api(`/api/tarefas/${t.id}/cancelar`, { metodo: 'POST' })
  } catch (e) {
    aviso.value = e.message
  }
  tarefas.value = await api('/api/tarefas')
}
const quando = (iso) => (iso ? new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }) : '')
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <span class="nav-titulo">Ajustes</span>
        <a class="nav-item" href="#/ajustes#instrucoes">Instruções e modelo</a>
        <a class="nav-item" href="#/ajustes#regras">Regras da IA</a>
        <a class="nav-item" href="#/ajustes#fila">Fila</a>
      </div>
    </template>

    <h1>Ajustes</h1>
    <p class="lead">Como a IA trabalha em todas as trilhas.</p>
    <p v-if="aviso" class="aviso">{{ aviso }}</p>

    <section v-if="ajustes" id="instrucoes" class="secao">
      <h2>Instruções e modelo</h2>
      <form class="pilha formulario" @submit.prevent="salvarAjustes">
        <div>
          <label class="rotulo" for="a-instr">Instruções globais</label>
          <textarea id="a-instr" v-model="ajustes.instrucoes_globais" class="campo" rows="4" placeholder="Ex.: Português do Brasil; fontes em inglês podem; nada de jargão sem explicar." />
          <p class="ajuda">Vão em toda chamada à IA, antes das instruções da trilha e do pedido de cada ação. Em conflito, vale a mais específica.</p>
        </div>
        <div>
          <label class="rotulo" for="a-modelo">Modelo da lição</label>
          <input id="a-modelo" v-model="ajustes.modelo_licao" class="campo" list="modelos" :placeholder="ajustes.modelo_padrao ? `padrão: ${ajustes.modelo_padrao}` : 'padrão do Claude Code'" />
          <datalist id="modelos">
            <option value="opus">mais cuidadoso, gasta o limite mais rápido</option>
            <option value="sonnet">equilibrado</option>
            <option value="haiku">rápido e econômico</option>
          </datalist>
          <p class="ajuda">opus, sonnet, haiku ou o nome completo (ex.: claude-opus-5-5). Vazio: o padrão da conta.</p>
        </div>
        <div>
          <label class="rotulo" for="a-modelo-criacao">Modelo da criação de trilhas</label>
          <input id="a-modelo-criacao" v-model="ajustes.modelo_criacao" class="campo" list="modelos" :placeholder="ajustes.modelo_padrao ? `padrão: ${ajustes.modelo_padrao}` : 'padrão do Claude Code'" />
          <p class="ajuda">Entrevista, mapa, validação da lista e plano. O plano decide a trilha inteira: vale o opus, se o limite permitir.</p>
        </div>
        <div>
          <label class="rotulo" for="a-modelo-estudo">Modelo dos exercícios e do simulado</label>
          <input id="a-modelo-estudo" v-model="ajustes.modelo_estudo" class="campo" list="modelos" :placeholder="ajustes.modelo_padrao ? `padrão: ${ajustes.modelo_padrao}` : 'padrão do Claude Code'" />
          <p class="ajuda">Gerar os exercícios, as perguntas do simulado e corrigir as respostas do simulado. O sonnet costuma bastar.</p>
        </div>
        <div>
          <label class="rotulo" for="a-nota">Nota mínima nos exercícios</label>
          <input id="a-nota" v-model="ajustes.nota_minima" class="campo curto" inputmode="numeric" placeholder="80" />
          <p class="ajuda">Em %. Atingida, a etapa sugere concluir (o critério automático); concluir à mão e pular continuam valendo.</p>
        </div>
        <div class="linha">
          <button class="botao primario" type="submit">Salvar</button>
          <span v-if="salvo" class="pequeno">{{ salvo }}</span>
        </div>
      </form>
    </section>

    <section id="regras" class="secao">
      <h2>Regras da IA</h2>
      <p class="suave">O que a IA segue em cada tipo de tarefa. Editar aqui vale na próxima chamada, sem deploy; a original fica guardada.</p>
      <table class="tabela">
        <tbody>
          <tr v-for="r in regras" :key="r.arquivo">
            <td>
              <a class="titulo" href="" @click.prevent="abrirRegra(r.arquivo)">{{ r.titulo }}</a>
              <div class="descricao">{{ r.para }}</div>
            </td>
            <td class="direita"><span v-if="r.editada" class="etiqueta ambar">editada</span></td>
          </tr>
        </tbody>
      </table>
      <div v-if="regra" class="pilha editor">
        <div class="linha entre">
          <strong>{{ regra.titulo }} <span class="fraco">· {{ regra.arquivo }}</span></strong>
          <button class="botao discreto" @click="regra = null">Fechar</button>
        </div>
        <textarea v-model="texto" class="campo regra" spellcheck="true" />
        <p v-if="erroRegra" class="aviso">{{ erroRegra }}</p>
        <div class="linha">
          <button class="botao primario" :disabled="!mudou" @click="salvarRegra">Salvar</button>
          <button v-if="regra.editada" class="botao" @click="restaurar">Restaurar a original</button>
        </div>
      </div>
    </section>

    <section id="fila" class="secao">
      <h2>Fila</h2>
      <p class="suave">Uma tarefa por vez. As mais recentes primeiro.</p>
      <table v-if="tarefas.length" class="tabela">
        <tbody>
          <tr v-for="t in tarefas" :key="t.id">
            <td>
              <a v-if="t.trilha_id" class="titulo" :href="link('trilhas', t.trilha_id)">{{ t.titulo }}</a>
              <span v-else class="titulo">{{ t.titulo }}</span>
              <div class="descricao">
                {{ quando(t.criado_em) }}<template v-if="t.modelo"> · {{ t.modelo }}</template>
                <template v-if="t.erro && t.estado === 'falhou'"> · {{ t.erro }}</template>
              </div>
            </td>
            <td class="direita">
              <div class="linha fim">
                <span class="etiqueta" :class="ESTADOS[t.estado].cor">{{ ESTADOS[t.estado].nome }}</span>
                <button v-if="t.estado === 'na_fila' || t.estado === 'rodando'" class="botao discreto" @click="cancelar(t)">Cancelar</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="suave">Nenhuma tarefa ainda.</p>
    </section>
  </Pagina>
</template>

<style scoped>
.secao { margin-top: var(--e7); }
.secao > h2 { margin-bottom: var(--e3); }
.formulario { max-width: var(--leitura); }
.curto { max-width: 120px; }
.editor { margin-top: var(--e5); }
.regra { min-height: 420px; font-family: var(--fonte-mono); font-size: 13px; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
</style>
