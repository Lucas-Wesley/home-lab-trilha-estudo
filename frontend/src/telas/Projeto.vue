<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Leitor from '../Leitor.vue'
import Pagina from '../Pagina.vue'
import RefProjeto from '../RefProjeto.vue'
import { api } from '../api.js'
import { renderizar, textoComCodigo } from '../markdown.js'
import { candidatosDoItem, copiarTexto, itensPorId, resumoFase, rotuloSituacao, situacaoEstudo } from '../projeto.js'
import { link } from '../rotas.js'

// Projeto de estudo (T52): o caderno do GitHub (roadmap, estado e cadernos dos itens) ligado às
// trilhas. #/projeto = visão geral; #/projeto/<id> = o item. A tela só lê; o "Conversar com a IA"
// copia o prompt para colar no Claude Code.
const props = defineProps({ rota: { type: Object, required: true } })

const dados = ref(null)
const erro = ref('')
const atualizando = ref(false)
const recado = ref({ texto: '', erro: false })

async function carregar(forcar = false) {
  erro.value = ''
  atualizando.value = true
  try {
    dados.value = forcar
      ? await api('/api/projeto/atualizar', { metodo: 'POST', tempo: 45000 })
      : await api('/api/projeto', { tempo: 45000 })
  } catch (e) {
    erro.value = e.message
  } finally {
    atualizando.value = false
  }
}
onMounted(() => carregar())

const itens = computed(() => itensPorId(dados.value))
const itemId = computed(() => props.rota.partes[0] || '')
const item = computed(() => itens.value[itemId.value] || null)
const estado = computed(() => dados.value?.estado || {})
const itemAtual = computed(() => itens.value[estado.value.item_id] || itens.value[estado.value.proximo_id] || null)

const proximosHtml = computed(() => renderizar(estado.value.proximos || '').html)
const markdownItem = computed(() => (item.value ? renderizar(item.value.markdown || '').html : ''))

const livros = computed(() => dados.value?.livros || [])
const totalLivros = computed(() =>
  livros.value.reduce((s, l) => ({ feitos: s.feitos + l.concluidas, total: s.total + l.total }), { feitos: 0, total: 0 }),
)
// Texto curto do roadmap (negrito, `código`, links) como Markdown.
const md = (t) => renderizar(t || '').html
const pct = (a, b) => (b ? Math.round((a / b) * 100) : 0)
const tituloLivro = (l) => (l.titulo || '').replace(/^Livro:\s*/, '')

const baixadoEm = computed(() => {
  if (!dados.value?.baixado_em) return ''
  return new Date(dados.value.baixado_em).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
})

// "Fase 0: o primeiro centavo (Go puro…)" → "Fase 0" para a navegação; o resto vai ao lado.
function partesFase(titulo) {
  const m = String(titulo || '').match(/^([^:]+):\s*(.*)$/)
  return m ? { nome: m[1], resto: m[2] } : { nome: titulo, resto: '' }
}
const faseDoItem = computed(() => (dados.value?.fases || []).find((f) => f.itens.some((i) => i.id === itemId.value)))

let relogio = 0
async function conversar(it) {
  if (!it) return
  const ok = await copiarTexto(it.prompt)
  recado.value = ok
    ? { texto: `Copiado: o prompt do ${it.id}. Cole no Claude Code.`, erro: false }
    : { texto: 'Não consegui copiar. Abra "Ver o prompt" e copie à mão.', erro: true }
  clearTimeout(relogio)
  relogio = setTimeout(() => (recado.value = { texto: '', erro: false }), 4000)
}
onBeforeUnmount(() => clearTimeout(relogio))

const candidatos = computed(() => candidatosDoItem(item.value))
const ICONE_SITUACAO = { feito: '✓', em_andamento: '●', a_fazer: '' }
</script>

<template>
  <Pagina>
    <template #nav>
      <div class="nav-grupo">
        <a class="nav-item" :class="{ ativo: !itemId }" :href="link('projeto')"><strong>Visão geral</strong></a>
      </div>
      <div v-for="f in dados?.fases || []" :key="f.titulo" class="nav-grupo">
        <span class="nav-titulo">{{ partesFase(f.titulo).nome }}</span>
        <a
          v-for="it in f.itens"
          :key="it.id"
          class="nav-item"
          :class="{ ativo: it.id === itemId }"
          :href="link('projeto', it.id)"
        >
          <span class="nav-id">{{ it.id }}</span>
          <span class="nav-nome">{{ it.titulo }}</span>
          <span v-if="ICONE_SITUACAO[it.situacao]" class="nav-situacao" :class="it.situacao">{{ ICONE_SITUACAO[it.situacao] }}</span>
        </a>
      </div>
    </template>

    <template v-if="item" #lateral>
      <section class="quadro">
        <div class="quadro-topo"><h2 class="rotulo-secao">Capítulos dos livros</h2></div>
        <p v-if="!item.capitulos.length" class="pequeno sem-margem">Nenhum capítulo de livro neste item.</p>
        <ul v-else class="lista capitulos">
          <li v-for="(c, i) in item.capitulos" :key="i">
            <div class="linha entre">
              <strong>{{ c.livro }} · cap. {{ c.caps }}</strong>
              <span v-if="c.revisao" class="etiqueta">revisão</span>
            </div>
            <p class="pequeno sem-margem">{{ c.assunto }}</p>
            <div class="refs">
              <RefProjeto v-for="(r, j) in c.refs" :key="j" :r="r" />
            </div>
          </li>
        </ul>
      </section>
    </template>

    <p v-if="erro" class="aviso">{{ erro }}</p>
    <p v-if="dados?.aviso" class="aviso">{{ dados.aviso }}</p>
    <p v-if="!dados && !erro" class="suave">Lendo o caderno no GitHub…</p>

    <!-- ---------- o item ---------- -->
    <template v-if="dados && itemId">
      <nav class="migalhas" aria-label="Você está em">
        <a :href="link('projeto')">Projeto de estudo</a>
        <template v-if="faseDoItem">
          <Icone nome="seta-direita" :tamanho="14" />
          <span>{{ partesFase(faseDoItem.titulo).nome }}</span>
        </template>
      </nav>
      <p v-if="!item" class="aviso">O item {{ itemId }} não está no roadmap.</p>
      <template v-else>
        <div class="topo-pagina">
          <h1 class="titulo-item">{{ item.id }} · <span v-html="textoComCodigo(item.titulo)" /></h1>
          <span class="etiqueta" :class="rotuloSituacao(item).cor">{{ rotuloSituacao(item).nome }}</span>
        </div>
        <div class="ferramentas">
          <button class="botao primario" @click="conversar(item)"><Icone nome="copiar" /> Conversar com a IA</button>
          <a class="botao discreto" :href="item.link" target="_blank" rel="noopener noreferrer"><Icone nome="link" /> No roadmap</a>
          <a v-if="item.link_caderno" class="botao discreto" :href="item.link_caderno" target="_blank" rel="noopener noreferrer"><Icone nome="link" /> Caderno no GitHub</a>
        </div>

        <div class="pilha">
          <section v-if="item.implementar" class="quadro">
            <div class="quadro-topo"><h2 class="rotulo-secao">Implementar</h2></div>
            <div class="texto" v-html="md(item.implementar)" />
          </section>
          <section v-if="item.pronto" class="quadro">
            <div class="quadro-topo"><h2 class="rotulo-secao">Pronto quando</h2></div>
            <div class="texto" v-html="md(item.pronto)" />
          </section>
          <div v-if="item.repositorio || item.outros.length" class="quadros">
            <section v-if="item.repositorio" class="quadro">
              <div class="quadro-topo"><h2 class="rotulo-secao">Repositório</h2></div>
              <div class="texto" v-html="md(item.repositorio)" />
            </section>
            <section v-for="o in item.outros" :key="o.rotulo" class="quadro">
              <div class="quadro-topo"><h2 class="rotulo-secao">{{ o.rotulo }}</h2></div>
              <div class="texto" v-html="md(o.texto)" />
            </section>
          </div>

          <section v-if="item.estudo_escolhido?.length" class="quadro">
            <div class="quadro-topo">
              <h2 class="rotulo-secao">O que estudar, nesta ordem</h2>
              <span class="pequeno">escolhido com a IA · caderno do item</span>
            </div>
            <ol class="lista escolhido">
              <li v-for="(e, i) in item.estudo_escolhido" :key="i">
                <span class="num">{{ i + 1 }}</span>
                <div class="escolhido-corpo">
                  <div class="linha entre">
                    <div class="refs">
                      <RefProjeto v-for="(r, j) in e.refs" :key="j" :r="r" />
                      <span v-if="!e.refs.length">{{ e.fonte }}</span>
                    </div>
                    <span v-if="e.situacao" class="etiqueta" :class="situacaoEstudo(e.situacao)">{{ e.situacao }}</span>
                  </div>
                  <p v-if="e.por_que" class="pequeno sem-margem">{{ e.por_que }}</p>
                </div>
              </li>
            </ol>
          </section>

          <component :is="item.estudo_escolhido?.length ? 'details' : 'section'" v-if="candidatos.length" class="quadro">
            <component :is="item.estudo_escolhido?.length ? 'summary' : 'div'" class="quadro-topo">
              <h2 class="rotulo-secao">{{ item.estudo_escolhido?.length ? 'Outras fontes candidatas' : 'Candidatos ao estudo' }}</h2>
            </component>
            <p v-if="!item.estudo_escolhido?.length" class="pequeno aviso-candidatos">
              Não é para estudar tudo isto. Na conversa, a IA pergunta o que você já sabe, escolhe o mínimo
              que destrava as decisões do item e explica o porquê. A lista escolhida aparece aqui, em ordem.
            </p>
            <div v-for="g in candidatos" :key="g.id" class="grupo-trilha">
              <a v-if="g.id && g.refs[0]?.trilha_id" class="grupo-nome" :href="link('trilhas', g.id)">{{ g.titulo }}</a>
              <span v-else class="grupo-nome">{{ g.titulo }}</span>
              <ul class="lista candidatos">
                <li v-for="(r, j) in g.refs" :key="j"><RefProjeto :r="r" curto /></li>
              </ul>
            </div>
          </component>

          <section class="quadro">
            <div class="quadro-topo">
              <h2 class="rotulo-secao">Caderno do item</h2>
              <span v-if="item.caderno_arquivo" class="pequeno">itens/{{ item.caderno_arquivo }}</span>
            </div>
            <Leitor v-if="item.caderno" :texto="item.caderno" />
            <p v-else class="pequeno sem-margem">Ainda não existe: ele nasce na primeira conversa com a IA sobre este item.</p>
          </section>

          <details class="quadro">
            <summary class="rotulo-secao">Ver o prompt</summary>
            <pre class="prompt">{{ item.prompt }}</pre>
          </details>
          <details class="quadro">
            <summary class="rotulo-secao">O item no roadmap</summary>
            <div class="leitura" v-html="markdownItem" />
          </details>
        </div>
      </template>
    </template>

    <!-- ---------- visão geral ---------- -->
    <template v-else-if="dados">
      <div class="topo-pagina">
        <h1>Projeto de estudo</h1>
        <div class="acoes">
          <span v-if="baixadoEm" class="pequeno">GitHub · {{ baixadoEm }}</span>
          <button class="botao discreto" :disabled="atualizando" @click="carregar(true)">{{ atualizando ? 'Atualizando…' : 'Atualizar' }}</button>
          <a class="botao discreto icone" :href="dados.repositorio" target="_blank" rel="noopener noreferrer" title="Abrir no GitHub" aria-label="Abrir no GitHub"><Icone nome="link" /></a>
        </div>
      </div>

      <section class="quadro estado">
        <div class="quadro-topo">
          <h2 class="rotulo-secao">Onde paramos</h2>
          <a :href="dados.link_estado" target="_blank" rel="noopener noreferrer">ESTADO.md</a>
        </div>
        <div class="estado-corpo">
          <div class="estado-atual">
            <p class="sem-margem">
              <template v-if="itens[estado.item_id]">
                Você está no item <a :href="link('projeto', estado.item_id)"><strong>{{ estado.item_id }} · {{ itens[estado.item_id].titulo }}</strong></a>,
                na fase <strong>{{ estado.fase || itens[estado.item_id].estado || 'problema' }}</strong>.
              </template>
              <template v-else>
                <strong>{{ estado.item_atual || 'Nenhum item em andamento' }}</strong>
                <span v-if="estado.fase"> · fase {{ estado.fase }}</span>
              </template>
            </p>
            <p v-if="estado.atualizado_em" class="pequeno sem-margem">Atualizado em {{ estado.atualizado_em }}</p>
            <button v-if="itemAtual" class="botao primario" @click="conversar(itemAtual)">
              <Icone nome="copiar" /> Conversar com a IA · {{ itemAtual.id }}
            </button>
          </div>
          <div v-if="estado.proximos" class="estado-proximos">
            <p class="rotulo-secao sub">Próximos passos</p>
            <div class="leitura curta" v-html="proximosHtml" />
          </div>
        </div>
      </section>

      <div class="numeros-quadro livros">
        <a v-for="l in livros" :key="l.id" class="numero-quadro" :href="l.trilha_id ? link('trilhas', l.trilha_id) : undefined" :title="l.titulo">
          <p class="rotulo-secao">{{ l.id }} · livro</p>
          <span class="valor-grande">{{ l.concluidas }}<span class="de">/{{ l.total }}</span></span>
          <span class="pequeno titulo-livro">{{ tituloLivro(l) }}</span>
          <div class="progresso"><span :style="{ width: pct(l.concluidas, l.total) + '%' }" /></div>
        </a>
        <div class="numero-quadro">
          <p class="rotulo-secao">Os três livros</p>
          <span class="valor-grande">{{ pct(totalLivros.feitos, totalLivros.total) }}%</span>
          <span class="pequeno">{{ totalLivros.feitos }} de {{ totalLivros.total }} capítulos. O projeto só termina com todos lidos.</span>
        </div>
      </div>

      <section v-for="f in dados.fases" :key="f.titulo" class="quadro fase">
        <div class="quadro-topo">
          <h2 class="rotulo-secao">{{ f.titulo }}</h2>
          <span class="pequeno">{{ resumoFase(f).feitos }} de {{ resumoFase(f).total }}</span>
        </div>
        <p v-if="f.objetivo" class="pequeno objetivo">{{ f.objetivo }}</p>
        <ul class="lista">
          <li v-for="it in f.itens" :key="it.id">
            <a class="item" :href="link('projeto', it.id)" :class="{ atual: it.id === estado.item_id }">
              <span class="id">{{ it.id }}</span>
              <span class="nome" v-html="textoComCodigo(it.titulo)" />
              <span class="etiqueta" :class="[rotuloSituacao(it).cor, { 'a-fazer': it.situacao === 'a_fazer' }]">{{ rotuloSituacao(it).nome }}</span>
            </a>
          </li>
        </ul>
      </section>

      <p class="pequeno">
        {{ dados.referencias.casadas }} referências do roadmap ligadas às trilhas
        <template v-if="dados.referencias.parciais"> · {{ dados.referencias.parciais }} só até o módulo ou a trilha</template>
        <template v-if="dados.referencias.nao_casadas"> · {{ dados.referencias.nao_casadas }} sem par</template>.
      </p>
    </template>

    <p v-if="recado.texto" class="recado" :class="{ erro: recado.erro }" role="status">{{ recado.texto }}</p>
  </Pagina>
</template>

<style scoped>
.sem-margem { margin: 0; }
.topo-pagina .titulo-item { flex: 0 1 auto; min-width: 0; overflow-wrap: anywhere; }
.texto { font-size: 15px; line-height: 1.6; }
.texto :deep(p) { margin: 0; }
.texto :deep(p + p), .texto :deep(ul) { margin: var(--e2) 0 0; }
.texto :deep(code) { font-size: 0.88em; }
.rotulo-secao.sub { margin: var(--e4) 0 var(--e2); }

.nav-item .nav-id { flex: none; min-width: 2.4em; font-family: var(--fonte-mono); font-size: 12px; color: var(--texto-fraco); }
.nav-item .nav-nome { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.nav-situacao { flex: none; font-size: 12px; color: var(--texto-fraco); }
.nav-situacao.feito { color: var(--verde); }
.nav-situacao.em_andamento { color: var(--azul); }

.estado { margin-bottom: var(--e4); }
.estado-corpo { display: grid; gap: var(--e4); grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); align-items: start; }
.estado-atual { display: grid; gap: var(--e3); justify-items: start; font-size: 15px; line-height: 1.5; }
.estado-atual a { color: var(--link); text-decoration: none; }
.estado-proximos .rotulo-secao.sub { margin-top: 0; }
.leitura.curta { font-size: 14.5px; line-height: 1.6; }
.leitura.curta :deep(ol), .leitura.curta :deep(ul) { margin: 0; padding-left: var(--e5); }
.leitura.curta :deep(li) { margin: 2px 0; }

.livros a.numero-quadro { color: inherit; text-decoration: none; }
.livros a.numero-quadro:hover { border-color: var(--borda-forte); }
.livros .de { font-size: 15px; font-weight: 500; color: var(--texto-fraco); }
.livros .titulo-livro { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.livros .progresso { margin-top: var(--e2); }

.fase { margin-bottom: var(--e4); }
.fase .objetivo { margin: calc(-1 * var(--e2)) 0 var(--e3); }
.item {
  display: flex;
  align-items: center;
  gap: var(--e3);
  padding: var(--e2) 0;
  color: var(--texto);
  text-decoration: none;
}
.item:hover .nome { text-decoration: underline; }
.item .id { flex: none; min-width: 2.6em; font-family: var(--fonte-mono); font-size: 13px; color: var(--texto-fraco); }
.item .nome { flex: 1; min-width: 0; font-size: 14.5px; }
.item .etiqueta { flex: none; }
.item.atual .nome { font-weight: 600; }

.escolhido { margin: 0; padding: 0; list-style: none; }
.escolhido li { display: flex; gap: var(--e3); align-items: baseline; padding: var(--e3) 0; }
.escolhido .num { flex: none; min-width: 1.4em; font-family: var(--fonte-mono); font-size: 13px; color: var(--texto-fraco); }
.escolhido-corpo { flex: 1; min-width: 0; display: grid; gap: var(--e1); }
.aviso-candidatos { margin: 0 0 var(--e3); }
.grupo-trilha { padding: var(--e2) 0; }
.grupo-nome { display: block; font-size: 13px; font-weight: 600; color: var(--texto); text-decoration: none; }
a.grupo-nome:hover { text-decoration: underline; }
.candidatos { margin: var(--e1) 0 0; padding: 0; list-style: none; font-size: 14.5px; line-height: 1.5; }
.candidatos li { padding: 2px 0; }
details > summary.quadro-topo { list-style: revert; }
.refs { display: flex; flex-wrap: wrap; gap: 2px var(--e3); min-width: 0; font-size: 14.5px; line-height: 1.5; }
.capitulos li { display: grid; gap: var(--e1); padding: var(--e3) 0; }
.capitulos .refs { flex-direction: column; gap: 2px; font-size: 13.5px; }
.linha.entre { justify-content: space-between; }

details > summary { cursor: pointer; }
details[open] > summary { margin-bottom: var(--e3); }
.prompt {
  margin: 0;
  padding: var(--e3);
  overflow-x: auto;
  white-space: pre-wrap;
  font-size: 12.5px;
  line-height: 1.5;
  background: var(--superficie-2);
  border-radius: var(--raio-p);
}

@media (max-width: 699px) {
  .item { flex-wrap: wrap; gap: var(--e1) var(--e3); }
  .item .nome { flex-basis: calc(100% - 3.6em); }
  .item .etiqueta { margin-left: calc(2.6em + var(--e3)); }
  .item .etiqueta.a-fazer { display: none; }
  .escolhido .linha.entre { flex-wrap: wrap; gap: var(--e1); }
}
</style>
