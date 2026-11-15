<script setup>
import { computed, onMounted, ref } from 'vue'
import PropostaReforco from './PropostaReforco.vue'
import { api } from './api.js'
import { link } from './rotas.js'
import { dataCurta } from './rotulos.js'

// Fase 7 (T28), na página da etapa: a revisão espaçada (quando ela volta), os pontos fracos (o que
// ele errou da última vez) e o reforço que a IA propõe.
const props = defineProps({
  etapa: { type: Object, required: true },
  trilhaId: { type: Number, required: true },
})
const emit = defineEmits(['mudou'])

const dados = ref(null)
const erro = ref('')
async function carregar() {
  try {
    dados.value = await api(`/api/etapas/${props.etapa.id}/estudo`)
  } catch (e) {
    erro.value = e.message
  }
}
onMounted(carregar)

const revisao = computed(() => dados.value?.revisao)
const pontos = computed(() => dados.value?.pontos_fracos || [])
const motivos = computed(() => dados.value?.sinais?.motivos || [])
const reforcoAberto = ref(false)
const mostrar = computed(() => props.etapa.status === 'concluida' || pontos.value.length || motivos.value.length || reforcoAberto.value)

function quando(r) {
  if (r.pendente) return 'hoje'
  return dataCurta(r.proxima)
}
const ONDE = { exercícios: 'nos exercícios', simulado: 'no simulado', revisão: 'na revisão', intercalado: 'no intercalado' }
</script>

<template>
  <section v-if="dados" id="dominio" class="secao">
    <div class="linha cabeca-secao">
      <h2>Domínio</h2>
    </div>
    <p v-if="erro" class="aviso">{{ erro }}</p>

    <div v-if="etapa.status === 'concluida' && revisao" class="cartao pilha" :class="{ pendente: revisao.pendente }">
      <div class="linha entre">
        <div>
          <strong>Revisão {{ revisao.pendente ? 'para hoje' : 'em ' + quando(revisao) }}</strong>
          <div class="pequeno">
            Degrau {{ revisao.nivel + 1 }} · intervalo de {{ revisao.intervalo }} dia(s)
            <template v-if="revisao.ultima_nota >= 0"> · última: {{ revisao.ultima_nota }}%</template>
          </div>
        </div>
        <a class="botao" :class="{ primario: revisao.pendente }" :href="link('trilhas', trilhaId, 'etapas', etapa.id, 'revisao')">{{ revisao.pendente ? 'Revisar' : 'Revisar agora' }}</a>
      </div>
    </div>
    <p v-else-if="etapa.status !== 'concluida'" class="suave sem-margem">Concluída, a etapa volta para revisões rápidas: 1, 3, 7, 14, 30, 60 e 120 dias depois (acertou, o intervalo cresce; errou, recomeça).</p>

    <div v-if="motivos.length || pontos.length" class="cartao pilha fraca">
      <span class="rotulo">Pontos fracos</span>
      <p v-if="motivos.length" class="sem-margem pequeno">{{ motivos.join(' · ') }}</p>
      <ul v-if="pontos.length" class="pontos">
        <li v-for="(p, i) in pontos" :key="i">
          <strong>{{ p.conceito }}</strong>
          <span class="fraco"> · errou {{ p.erros }}×{{ p.acertos ? `, acertou ${p.acertos}×` : '' }} · {{ ONDE[p.onde] || p.onde }}</span>
          <div v-if="p.exemplo && p.exemplo !== p.conceito" class="pequeno">{{ p.exemplo }}</div>
          <div class="pequeno"><span class="fraco">Sua resposta:</span> {{ p.resposta || '(em branco)' }}<template v-if="p.esperado"> · <span class="fraco">esperado:</span> {{ p.esperado }}</template></div>
        </li>
      </ul>
    </div>

    <PropostaReforco v-if="mostrar" :key="`reforco-${etapa.id}`" :etapa-id="etapa.id" :tem-erros="pontos.length > 0" @aberta="reforcoAberto = $event" @aprovada="emit('mudou')" />
  </section>
</template>

<style scoped>
.cabeca-secao { margin-bottom: var(--e3); }
.entre { justify-content: space-between; }
.sem-margem { margin: 0; }
.pilha + .pilha, .cartao + .pilha { margin-top: var(--e3); }
.pendente { border-color: var(--ambar-fundo, var(--borda-forte)); }
.fraca { border-color: var(--vermelho-fundo); }
.pontos { margin: 0; padding-left: var(--e5); display: grid; gap: var(--e2); line-height: 1.5; overflow-wrap: anywhere; }
</style>
