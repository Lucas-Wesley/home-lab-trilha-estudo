<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Pagina from '../Pagina.vue'
import { api } from '../api.js'
import { link } from '../rotas.js'

// Ajustes da IA (DECISOES.md §5.1, T5, T6, T31), uma página para cada assunto: Geral (instruções
// globais e nota mínima), Modelos (um por tipo de tarefa, com o recomendado), Regras da IA e Fila.
const props = defineProps({ rota: { type: Object, required: true } })

const PAGINAS = [
  { id: 'geral', nome: 'Geral' },
  { id: 'modelos', nome: 'Modelos' },
  { id: 'regras', nome: 'Regras da IA' },
  { id: 'fila', nome: 'Fila' },
]
// #/ajustes/modelos; os endereços antigos (#/ajustes#regras) continuam abrindo a página certa.
const pagina = computed(() => {
  const p = props.rota.partes[0] || { instrucoes: 'geral' }[props.rota.ancora] || props.rota.ancora || 'geral'
  return PAGINAS.some((x) => x.id === p) ? p : 'geral'
})

// Modelos (T31): recomendação medida em 17/11/2026 (teste Haiku × Sonnet nos exercícios, no
// simulado e na correção): qualidade equivalente, o Haiku gasta bem menos do limite da conta.
const MODELOS = [
  { campo: 'modelo_licao', nome: 'Lição', recomendado: 'sonnet', para: 'Escrever a lição e pedir mudança nela; também "Sugerir material". Texto longo e didático: o Haiku perde qualidade, o Opus gasta muito mais para pouco ganho.' },
  { campo: 'modelo_criacao', nome: 'Criação de trilhas', recomendado: 'opus', para: 'Entrevista, mapa, validação da lista, mini-teste de nível, plano e reforço. Roda poucas vezes e decide a trilha inteira: vale o mais cuidadoso. Também revisa o roteiro (os tópicos) quando você pede.' },
  { campo: 'modelo_estudo', nome: 'Exercícios e simulado', recomendado: 'haiku', para: 'Gerar os exercícios e as perguntas do simulado, a partir da lição aprovada. No teste, o Haiku acertou os gabaritos como o Sonnet.' },
  { campo: 'modelo_correcao', nome: 'Correção e conferência', recomendado: 'haiku', para: 'Corrigir o simulado (também o intercalado) e conferir o resumo com a aula. Vazio: o mesmo dos exercícios.' },
  { campo: 'modelo_entrada', nome: 'Entrada: fragmentar e sugerir trilhas', recomendado: 'haiku', para: 'Dividir a fonte em fragmentos por assunto e sugerir as trilhas (fase 4). Tarefa estruturada sobre a transcrição: o Haiku dá conta.' },
]
const NOMES_MODELO = { haiku: 'Haiku', sonnet: 'Sonnet', opus: 'Opus' }
function opcoes(m) {
  const lista = [{ valor: '', nome: ajustes.value?.modelo_padrao ? `Padrão (${ajustes.value.modelo_padrao})` : 'Padrão da conta' }]
  for (const v of ['haiku', 'sonnet', 'opus']) lista.push({ valor: v, nome: NOMES_MODELO[v] + (v === m.recomendado ? ' (recomendado)' : '') })
  const atual = ajustes.value?.[m.campo]
  if (atual && !NOMES_MODELO[atual]) lista.push({ valor: atual, nome: atual })
  return lista
}
function usarRecomendados() {
  for (const m of MODELOS) ajustes.value[m.campo] = m.recomendado
}

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
watch(pagina, () => {
  salvo.value = ''
})
onMounted(async () => {
  await carregar()
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
        <a v-for="p in PAGINAS" :key="p.id" class="nav-item" :class="{ ativo: pagina === p.id }" :href="link('ajustes', p.id)">{{ p.nome }}</a>
      </div>
    </template>

    <nav class="abas paginas" aria-label="Páginas de ajustes">
      <a v-for="p in PAGINAS" :key="p.id" class="aba" :class="{ ativa: pagina === p.id }" :href="link('ajustes', p.id)">{{ p.nome }}</a>
    </nav>
    <p v-if="aviso" class="aviso">{{ aviso }}</p>

    <!-- Geral -->
    <section v-if="pagina === 'geral' && ajustes">
      <div class="topo-pagina"><h1>Geral</h1></div>
      <p class="explicacao">Como a IA trabalha em todas as trilhas.</p>
      <form class="pilha formulario secao-curta" @submit.prevent="salvarAjustes">
        <div>
          <label class="rotulo" for="a-instr">Instruções globais</label>
          <textarea id="a-instr" v-model="ajustes.instrucoes_globais" class="campo" rows="5" placeholder="Ex.: Português do Brasil; fontes em inglês podem; nada de jargão sem explicar." />
          <p class="ajuda">Vão em toda chamada à IA, antes das instruções da trilha e do pedido de cada ação. Em conflito, vale a mais específica.</p>
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

    <!-- Modelos -->
    <section v-else-if="pagina === 'modelos' && ajustes">
      <div class="topo-pagina"><h1>Modelos</h1></div>
      <p class="explicacao">Qual modelo do Claude faz cada tipo de tarefa. O recomendado está marcado; mude para testar.</p>
      <form class="pilha formulario secao-curta" @submit.prevent="salvarAjustes">
        <div v-for="m in MODELOS" :key="m.campo">
          <label class="rotulo" :for="m.campo">{{ m.nome }} <span class="fraco">(recomendado: {{ NOMES_MODELO[m.recomendado] }})</span></label>
          <select :id="m.campo" v-model="ajustes[m.campo]" class="campo">
            <option v-for="o in opcoes(m)" :key="o.valor" :value="o.valor">{{ o.nome }}</option>
          </select>
          <p class="ajuda">{{ m.para }}</p>
        </div>
        <p class="ajuda">Haiku: rápido e econômico. Sonnet: equilibrado. Opus: o mais cuidadoso, gasta o limite da conta bem mais rápido. "Padrão" é o do Claude Code na conta ativa (hoje, o Sonnet).</p>
        <div class="linha">
          <button class="botao primario" type="submit">Salvar</button>
          <button class="botao" type="button" @click="usarRecomendados">Usar os recomendados</button>
          <span v-if="salvo" class="pequeno">{{ salvo }}</span>
        </div>
      </form>
    </section>

    <!-- Regras -->
    <section v-else-if="pagina === 'regras'">
      <div class="topo-pagina"><h1>Regras da IA</h1></div>
      <p class="explicacao">O que a IA segue em cada tipo de tarefa. Editar aqui vale na próxima chamada, sem deploy; a original fica guardada.</p>
      <table class="tabela secao-curta">
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

    <!-- Fila -->
    <section v-else-if="pagina === 'fila'">
      <div class="topo-pagina"><h1>Fila</h1></div>
      <p class="explicacao">Uma tarefa por vez. As mais recentes primeiro, com o modelo de cada uma.</p>
      <table v-if="tarefas.length" class="tabela secao-curta">
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
      <p v-else class="suave secao-curta">Nenhuma tarefa ainda.</p>
    </section>
  </Pagina>
</template>

<style scoped>
.paginas { margin-bottom: var(--e5); }
.paginas .aba { padding: 6px var(--e3); border-radius: var(--raio-p); font-size: 14px; color: var(--texto-suave); text-decoration: none; }
.paginas .aba.ativa { background: var(--ativo); color: var(--texto); }
.secao-curta { margin-top: var(--e5); }
/* No computador as páginas estão na navegação lateral; no celular, em abas no topo. */
@media (min-width: 900px) { .paginas { display: none; } }
.formulario { max-width: var(--leitura); }
.curto { max-width: 120px; }
.editor { margin-top: var(--e5); }
.regra { min-height: 420px; font-family: var(--fonte-mono); font-size: 13px; }
a.titulo { text-decoration: none; }
a.titulo:hover { text-decoration: underline; }
</style>
