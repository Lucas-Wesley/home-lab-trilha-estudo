<script setup>
import { computed, ref } from 'vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'
import { link } from '../rotas.js'

const props = defineProps({ rota: { type: Object, required: true } })
const nova = computed(() => props.rota.partes[0] === 'nova')

// As quatro entradas de criação (DECISOES.md §3). Nesta fase só aparecem; a criação chega nas
// fases 2 (manual) e 4 (com a IA).
const entradas = [
  { id: 'ia', icone: 'trilhas', titulo: 'A IA monta', texto: 'Pré-pesquisa, entrevista com perguntas para este tipo de assunto, mapa completo e você escolhe o recorte.' },
  { id: 'lista', icone: 'menu', titulo: 'Eu tenho a lista', texto: 'Você dá a ordem; a IA só aponta pré-requisitos que faltam, e você aceita ou não.' },
  { id: 'material', icone: 'livro', titulo: 'A partir de material', texto: 'Um livro da Biblioteca, Temas do Acervo ou documentos próprios viram a espinha dorsal.' },
  { id: 'manual', icone: 'mais', titulo: 'Montar à mão', texto: 'Sem IA: módulos e etapas escritos por você.' },
]
const entrada = ref('ia')
const tema = ref('')
const pedido = ref('')
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <span class="nav-titulo">Trilhas</span>
        <a class="nav-item" :class="{ ativo: !nova }" :href="link('trilhas')">Todas</a>
        <a class="nav-item" :class="{ ativo: nova }" :href="link('trilhas', 'nova')">Nova trilha</a>
      </div>
    </template>

    <template v-if="nova">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('trilhas')">Trilhas</a>
        <Icone nome="seta-direita" :tamanho="14" />
        <span>Nova trilha</span>
      </nav>
      <h1>Nova trilha</h1>
      <p class="lead">Diga o que quer aprender com as suas palavras. Tudo o que a IA propuser fica aguardando a sua aprovação, e você pode comentar e pedir ajustes em qualquer ponto.</p>

      <div class="grade entradas" role="radiogroup" aria-label="Como começar">
        <button
          v-for="e in entradas"
          :key="e.id"
          class="cartao"
          :class="{ escolhida: entrada === e.id }"
          role="radio"
          :aria-checked="entrada === e.id"
          @click="entrada = e.id"
        >
          <Icone :nome="e.icone" :tamanho="20" class="entrada-icone" />
          <h3>{{ e.titulo }}</h3>
          <p>{{ e.texto }}</p>
        </button>
      </div>

      <div class="pilha formulario">
        <div>
          <label class="rotulo" for="n-tema">Tema</label>
          <input id="n-tema" v-model="tema" class="campo" placeholder="Ex.: História do Brasil, AWS, a rede do meu homelab" />
        </div>
        <div>
          <label class="rotulo" for="n-pedido">O que você quer com isto? <span class="fraco">(opcional)</span></label>
          <textarea
            id="n-pedido"
            v-model="pedido"
            class="campo"
            placeholder="Foco, o que já sabe, o que evitar, fontes preferidas… O que estiver aqui a IA não pergunta de novo."
          />
        </div>
        <div class="linha">
          <button class="botao primario" disabled>Continuar</button>
          <span class="pequeno">A criação de trilhas chega na próxima fase (2: à mão; 4: com a IA).</span>
        </div>
      </div>
    </template>

    <template v-else>
      <div class="cabecalho">
        <div>
          <h1>Trilhas</h1>
          <p class="lead">Estudo em ordem, com profundidade e progresso.</p>
        </div>
      </div>
      <div class="linha filtros">
        <label class="busca">
          <Icone nome="lupa" />
          <span class="so-leitor">Buscar</span>
          <input class="campo" placeholder="Buscar trilhas…" disabled />
        </label>
        <button class="seletor" disabled>Situação <strong>Todas</strong> <Icone nome="seta-baixo" :tamanho="14" /></button>
      </div>
      <div class="palco fundo-pontos">
        <div class="painel pilha">
          <h3 class="centro">Nenhuma trilha ainda</h3>
          <p class="suave centro">A primeira vai ser a da rede do homelab, montada a partir dos documentos de estudo.</p>
          <a class="botao primario largo" :href="link('trilhas', 'nova')">Nova trilha</a>
        </div>
      </div>
    </template>
  </Pagina>
</template>

<style scoped>
.entradas { margin-top: var(--e6); grid-template-columns: repeat(2, minmax(0, 1fr)); max-width: var(--leitura); }
.entradas .cartao { text-align: left; font: inherit; cursor: pointer; }
.entradas .cartao p { -webkit-line-clamp: 3; }
@media (max-width: 599px) {
  .entradas { grid-template-columns: minmax(0, 1fr); }
}
.entradas .cartao.escolhida { border-color: var(--texto-suave); background: var(--superficie); }
.entrada-icone { color: var(--texto-suave); margin-bottom: var(--e2); }
.formulario { margin-top: var(--e6); max-width: var(--leitura); }
.filtros { margin-top: var(--e5); }
.filtros .busca { flex: 1; min-width: 200px; max-width: 400px; }
.palco { margin-top: var(--e5); padding: var(--e8) var(--e4); border: 1px solid var(--borda); border-radius: var(--raio); }
.centro { text-align: center; margin: 0; }
</style>
