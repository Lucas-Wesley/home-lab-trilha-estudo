<script setup>
// Encaixar material (T32): o Lucas manda um material à trilha inteira e a IA diz em que etapas ele
// entra (com o capítulo ou o trecho) e que tópicos ele traz que a trilha não tem. Cada material é
// uma proposta: ele marca o que aceita, pede ajuste ou descarta; só o marcado entra na trilha.
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import EscolherMaterial from './EscolherMaterial.vue'
import { api } from './api.js'
import { link } from './rotas.js'
import { BLOOM, PAPEIS, TIPOS_MATERIAL } from './rotulos.js'

const props = defineProps({ trilha: { type: Object, required: true } })
const emit = defineEmits(['mudou'])

const lista = ref([])
const erro = ref('')
const ocupado = ref(false)
const escolhendo = ref(false)
const escolhido = ref(null)
const pedido = ref('')

const etapas = computed(() => props.trilha.modulos.flatMap((m) => m.etapas))
const comMaterial = computed(() => etapas.value.filter((e) => e.materiais?.length).length)
const porChave = computed(() => Object.fromEntries(props.trilha.modulos.flatMap((m, mi) => m.etapas.map((e, ei) => [e.chave, { ...e, numero: `${mi + 1}.${ei + 1}` }]))))

async function carregar() {
  try {
    lista.value = await api(`/api/trilhas/${props.trilha.id}/encaixes`)
    erro.value = ''
  } catch (e) {
    erro.value = e.message
  }
}
let relogio = 0
const gerando = computed(() => lista.value.some((x) => x.proposta.estado === 'gerando'))
watch(gerando, (g) => {
  clearInterval(relogio)
  if (g) relogio = setInterval(carregar, 5000)
})
onMounted(carregar)
onBeforeUnmount(() => clearInterval(relogio))

const abertas = computed(() => lista.value.filter((x) => x.proposta.estado !== 'aprovada'))
const feitas = computed(() => lista.value.filter((x) => x.proposta.estado === 'aprovada'))

async function acao(fn) {
  ocupado.value = true
  erro.value = ''
  try {
    await fn()
    await carregar()
    return true
  } catch (e) {
    erro.value = e.message
    return false
  } finally {
    ocupado.value = false
  }
}

function escolher(m) {
  escolhido.value = m
  escolhendo.value = false
}
async function encaixar(material = escolhido.value, texto = pedido.value) {
  const ok = await acao(() => api(`/api/trilhas/${props.trilha.id}/encaixar`, { metodo: 'POST', corpo: { material, pedido: texto } }))
  if (ok && material === escolhido.value) {
    escolhido.value = null
    pedido.value = ''
  }
}

// Marcados: tudo, no começo; o Lucas desmarca o que não quer.
const marcados = ref({})
watch(
  lista,
  (l) => {
    for (const x of l) {
      const p = x.proposta
      if (p.estado !== 'aguardando' || marcados.value[p.id]) continue
      marcados.value[p.id] = {
        encaixes: new Set((p.conteudo?.encaixes || []).map((_, i) => i)),
        novos: new Set((p.conteudo?.novos || []).map((_, i) => i)),
      }
    }
  },
  { immediate: true },
)
function alternar(pid, grupo, i) {
  const s = new Set(marcados.value[pid][grupo])
  s.has(i) ? s.delete(i) : s.add(i)
  marcados.value[pid] = { ...marcados.value[pid], [grupo]: s }
}
const total = (pid) => (marcados.value[pid]?.encaixes.size || 0) + (marcados.value[pid]?.novos.size || 0)

async function aplicar(p) {
  const m = marcados.value[p.id]
  if (await acao(() => api(`/api/propostas/${p.id}/aprovar`, { metodo: 'POST', corpo: { encaixes: [...m.encaixes], novos: [...m.novos] } }))) emit('mudou')
}
const ajuste = ref({})
async function ajustar(p) {
  if (await acao(() => api(`/api/propostas/${p.id}/ajustar`, { metodo: 'POST', corpo: { pedido: ajuste.value[p.id] || '' } }))) ajuste.value[p.id] = ''
}
const descartar = (p) => acao(() => api(`/api/propostas/${p.id}/descartar`, { metodo: 'POST' }))
const cancelar = (p) => p.tarefa && acao(() => api(`/api/tarefas/${p.tarefa.id}/cancelar`, { metodo: 'POST' }))
const onde = (n) => porChave.value[n.depois_de] ? `depois de ${porChave.value[n.depois_de].numero} ${porChave.value[n.depois_de].titulo}` : 'no fim do módulo'
</script>

<template>
  <div class="pilha">
    <p class="suave sem-margem">
      Mande um material e a IA diz em que etapas ele entra (com o capítulo ou o trecho) e que tópicos ele traz que a trilha ainda não tem. Você marca o que aceita.
      <strong>{{ comMaterial }} de {{ etapas.length }}</strong> etapas já têm material.
    </p>
    <details class="dica">
      <summary>Que material dá para mandar?</summary>
      <ul>
        <li><strong>Livro ou PDF</strong>: aba Livro (o PDF sobe para a Biblioteca com "Enviar PDF"). Livro inteiro: a IA encaixa por capítulo, pelo sumário.</li>
        <li><strong>Vídeo do YouTube ou Reel do Instagram</strong>: mande antes ao Acervo (Compartilhar no celular), que transcreve; quando a Referência tiver texto, escolha-a na aba Acervo. Um link direto na aba Web só dá à IA o título e a descrição.</li>
        <li><strong>Artigo ou página</strong>: na aba Web (a IA lê a página) ou pelo Acervo.</li>
        <li><strong>Texto seu</strong>: como documento da trilha (seção Documentos) e escolha na aba Documento.</li>
      </ul>
    </details>

    <EscolherMaterial v-if="escolhendo" :documentos="trilha.documentos || []" sem-papel @escolher="escolher" @fechar="escolhendo = false" />
    <div v-else-if="escolhido" class="cartao cheio pilha">
      <div class="linha">
        <span class="etiqueta">{{ TIPOS_MATERIAL[escolhido.tipo] }}</span>
        <strong>{{ escolhido.titulo }}</strong>
        <span v-if="escolhido.localizador" class="fraco">· {{ escolhido.localizador }}</span>
      </div>
      <div>
        <label class="rotulo" for="encaixe-pedido">Pedido para a IA <span class="fraco">(opcional)</span></label>
        <textarea id="encaixe-pedido" v-model="pedido" class="campo" rows="2" placeholder="Ex.: é a minha referência principal de TCP; os capítulos 9 e 10 são de nível avançado." />
      </div>
      <div class="linha">
        <button class="botao primario" :disabled="ocupado" @click="encaixar()">Ver onde encaixa</button>
        <button class="botao discreto" @click="escolhido = null">Cancelar</button>
      </div>
    </div>
    <div v-else><button class="botao" @click="escolhendo = true">Encaixar um material</button></div>
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <div v-for="x in abertas" :key="x.proposta.id" class="cartao pilha encaixe">
      <div class="linha entre">
        <div class="linha">
          <span class="etiqueta">{{ TIPOS_MATERIAL[x.material.tipo] }}</span>
          <strong>{{ x.material.titulo }}</strong>
          <span v-if="x.material.localizador" class="fraco">· {{ x.material.localizador }}</span>
        </div>
        <span v-if="x.proposta.versao > 1" class="pequeno">versão {{ x.proposta.versao }}</span>
      </div>

      <template v-if="x.proposta.estado === 'gerando'">
        <p class="suave sem-margem">A IA está lendo o material e a trilha… (leva alguns minutos; a fila faz uma tarefa por vez)</p>
        <div><button class="botao discreto" :disabled="ocupado" @click="cancelar(x.proposta)">Cancelar</button></div>
      </template>

      <template v-else-if="x.proposta.estado === 'falhou'">
        <p class="aviso sem-margem">Não deu: {{ x.proposta.erro }}</p>
        <div class="linha">
          <button class="botao" :disabled="ocupado" @click="encaixar(x.material, x.proposta.pedido)">Tentar de novo</button>
          <button class="botao discreto" :disabled="ocupado" @click="descartar(x.proposta)">Descartar</button>
        </div>
      </template>

      <template v-else-if="x.proposta.conteudo">
        <p v-if="x.proposta.conteudo.notas" class="suave sem-margem pre">{{ x.proposta.conteudo.notas }}</p>

        <div v-if="x.proposta.conteudo.encaixes.length">
          <span class="rotulo">Entra nestas etapas</span>
          <label v-for="(e, i) in x.proposta.conteudo.encaixes" :key="'e' + i" class="opcao">
            <input type="checkbox" :checked="marcados[x.proposta.id]?.encaixes.has(i)" @change="alternar(x.proposta.id, 'encaixes', i)" />
            <span>
              <strong>{{ e.numero }}</strong>
              <a :href="link('trilhas', trilha.id, 'etapas', porChave[e.etapa_chave]?.id)" target="_blank">{{ e.titulo }}</a>
              <span class="fraco"> · {{ e.localizador || 'o material todo' }} · {{ PAPEIS[e.papel] }}</span>
              <span class="bloco-nota">{{ e.nota }}<span v-if="e.motivo" class="fraco"> — {{ e.motivo }}</span></span>
            </span>
          </label>
        </div>

        <div v-if="x.proposta.conteudo.novos.length">
          <span class="rotulo">Tópicos que a trilha não tem</span>
          <label v-for="(n, i) in x.proposta.conteudo.novos" :key="'n' + i" class="opcao">
            <input type="checkbox" :checked="marcados[x.proposta.id]?.novos.has(i)" @change="alternar(x.proposta.id, 'novos', i)" />
            <span>
              <strong>Novo:</strong> {{ n.titulo }}
              <span class="fraco"> · em "{{ n.modulo }}"<template v-if="n.modulo_novo"> (módulo novo)</template>, {{ onde(n) }} · {{ BLOOM[n.bloom] }} · {{ n.minutos }} min</span>
              <span class="bloco-nota">{{ n.objetivo }}<span v-if="n.motivo" class="fraco"> — {{ n.motivo }}</span></span>
            </span>
          </label>
        </div>

        <div class="linha">
          <button class="botao primario" :disabled="ocupado || !total(x.proposta.id)" @click="aplicar(x.proposta)">Aplicar os marcados ({{ total(x.proposta.id) }})</button>
          <button class="botao discreto" :disabled="ocupado" @click="descartar(x.proposta)">Descartar</button>
        </div>
        <div class="linha ajuste">
          <input v-model="ajuste[x.proposta.id]" class="campo" placeholder="Pedir ajuste: ex.: o cap. 7 também serve à etapa 5.3" @keydown.enter="ajuste[x.proposta.id] && ajustar(x.proposta)" />
          <button class="botao" :disabled="ocupado || !ajuste[x.proposta.id]" @click="ajustar(x.proposta)">Ajustar</button>
        </div>
      </template>
    </div>

    <details v-if="feitas.length" class="dica">
      <summary>{{ feitas.length }} material(is) já encaixado(s)</summary>
      <ul>
        <li v-for="x in feitas" :key="x.proposta.id">
          {{ x.material.titulo }}
          <span class="fraco">· {{ (x.proposta.conteudo?.encaixes || []).map((e) => e.numero).join(', ') }}</span>
        </li>
      </ul>
    </details>
  </div>
</template>

<style scoped>
.sem-margem { margin: 0; }
.pre { white-space: pre-line; }
.dica > summary { cursor: pointer; color: var(--texto-suave); font-size: 14px; }
.dica ul { margin: var(--e2) 0 0; padding-left: var(--e5); font-size: 14px; color: var(--texto-suave); display: grid; gap: var(--e1); }
.opcao { display: flex; gap: var(--e2); align-items: flex-start; padding: 6px 0; font-size: 14.5px; line-height: 1.5; }
.opcao input { margin-top: 4px; }
.bloco-nota { display: block; color: var(--texto-suave); font-size: 13.5px; }
.ajuste .campo { flex: 1 1 260px; }
</style>
