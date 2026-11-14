<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import Icone from './Icone.vue'
import { api, ErroApi } from './api.js'
import { localizadorDeCapitulos, localizadorDePaginas } from './material.js'
import { PAPEIS } from './rotulos.js'

// Escolher um material (fase 6, T27): um livro da Biblioteca (inteiro, alguns capítulos ou
// páginas), um Tema ou uma Referência do Acervo, um documento da trilha ou uma página da web. Um
// PDF próprio sobe para a Biblioteca (T15) e já fica escolhido. O Streaming aparece desativado
// (T16). Na criação a partir de material (simples), só livro e Acervo, sem papel nem nota.
const props = defineProps({
  documentos: { type: Array, default: () => [] },
  simples: { type: Boolean, default: false },
})
const emit = defineEmits(['escolher', 'fechar'])

const ligados = ref({ biblioteca: false, acervo: false })
const aba = ref('livro')
const abas = computed(() =>
  [
    { id: 'livro', nome: 'Livro', ligado: ligados.value.biblioteca },
    { id: 'acervo', nome: 'Acervo', ligado: ligados.value.acervo },
    { id: 'documento', nome: 'Documento', ligado: !props.simples },
    { id: 'web', nome: 'Web', ligado: !props.simples },
  ].filter((a) => a.ligado),
)
const erro = ref('')
const papel = ref('apoio')
const nota = ref('')
const filtro = ref('')
const normal = (s) => (s || '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()
const casa = (...textos) => !filtro.value.trim() || textos.some((t) => normal(t).includes(normal(filtro.value.trim())))

onMounted(async () => {
  try {
    ligados.value = await api('/api/integracoes')
  } catch {
    // Sem a lista, só documento e web.
  }
  aba.value = abas.value[0]?.id || 'web'
  carregarAba()
})
function carregarAba() {
  if (aba.value === 'livro' && !livros.value) carregarLivros()
  if (aba.value === 'acervo' && !temas.value) carregarAcervo()
}
watch(aba, () => {
  erro.value = ''
  filtro.value = ''
  carregarAba()
})

function escolher(m) {
  emit('escolher', { localizador: '', ...m, papel: props.simples ? 'principal' : papel.value, nota: props.simples ? '' : nota.value.trim() })
  nota.value = ''
}

// ---------- livro ----------
const livros = ref(null)
const livro = ref(null) // o escolhido, com o sumário
const carregandoLivro = ref(false)
const capitulos = ref(new Set())
const paginas = ref({ de: '', ate: '' })
async function carregarLivros() {
  try {
    livros.value = await api('/api/biblioteca/livros')
  } catch (e) {
    erro.value = e.message
    livros.value = []
  }
}
const livrosFiltrados = computed(() => (livros.value || []).filter((l) => casa(l.title, l.author)))
async function abrirLivro(l) {
  livro.value = { ...l, chapters: null }
  capitulos.value = new Set()
  paginas.value = { de: '', ate: '' }
  carregandoLivro.value = true
  try {
    livro.value = { ...l, ...(await api(`/api/biblioteca/livros/${encodeURIComponent(l.id)}`, { tempo: 70000 })) }
  } catch (e) {
    erro.value = e.message
  } finally {
    carregandoLivro.value = false
  }
}
function alternarCapitulo(n) {
  const s = new Set(capitulos.value)
  s.has(n) ? s.delete(n) : s.add(n)
  capitulos.value = s
}
const localizadorLivro = computed(() =>
  capitulos.value.size ? localizadorDeCapitulos([...capitulos.value]) : localizadorDePaginas(Number(paginas.value.de) || 0, Number(paginas.value.ate) || 0),
)
const tituloDoTrecho = computed(() => {
  const l = livro.value
  if (!l) return ''
  if (capitulos.value.size === 1) {
    const c = l.chapters?.find((x) => x.n === [...capitulos.value][0])
    if (c && normal(c.title) !== normal(l.title)) return `${c.title} (${l.title})`
  }
  return l.title
})
function escolherLivro() {
  escolher({ tipo: 'livro', titulo: tituloDoTrecho.value, referencia: livro.value.id, localizador: localizadorLivro.value })
  livro.value = null
}
const mil = (n) => (n >= 1000 ? `${Math.round(n / 1000)} mil` : `${n}`)

// PDF próprio: sobe para a Biblioteca e abre o sumário dele.
const enviando = ref(false)
const tituloPdf = ref('')
async function enviarPdf(evento) {
  const arquivo = evento.target.files[0]
  evento.target.value = ''
  if (!arquivo) return
  erro.value = ''
  enviando.value = true
  try {
    const dados = new FormData()
    if (tituloPdf.value.trim()) dados.append('title', tituloPdf.value.trim())
    dados.append('file', arquivo)
    const resposta = await fetch('/api/biblioteca/livros', { method: 'POST', body: dados })
    const corpo = await resposta.json().catch(() => null)
    if (!resposta.ok) throw new ErroApi(corpo?.erro || `Erro ${resposta.status}`, resposta.status)
    tituloPdf.value = ''
    await carregarLivros()
    await abrirLivro(corpo)
  } catch (e) {
    erro.value = e.status === 409 ? 'Este arquivo já está na Biblioteca: procure-o na lista.' : e.message
  } finally {
    enviando.value = false
  }
}

// ---------- Acervo ----------
const temas = ref(null)
const referencias = ref([])
async function carregarAcervo() {
  try {
    ;[temas.value, referencias.value] = await Promise.all([api('/api/acervo/temas'), api('/api/acervo/referencias')])
  } catch (e) {
    erro.value = e.message
    temas.value = []
  }
}
const temasFiltrados = computed(() => (temas.value || []).filter((t) => casa(t.titulo, t.resumo, t.caminho)))
const referenciasFiltradas = computed(() => (filtro.value.trim() ? referencias.value.filter((r) => casa(r.titulo)) : referencias.value.slice(0, 8)))
const profundidadeDoTema = (t) => (t.caminho.match(/\//g) || []).length - 1
const TIPOS_REF = { video: 'vídeo', artigo: 'artigo', podcast: 'podcast', reel: 'Reel', documentario: 'documentário', pesquisa: 'pesquisa', livro: 'livro' }

// ---------- documento e web ----------
const doc = ref({ referencia: '', localizador: '' })
const web = ref({ titulo: '', referencia: '' })
function escolherDocumento() {
  const d = props.documentos.find((x) => x.slug === doc.value.referencia)
  if (!d) return
  escolher({ tipo: 'documento', titulo: d.titulo, referencia: d.slug, localizador: doc.value.localizador.trim().replace(/^#/, '') })
  doc.value = { referencia: '', localizador: '' }
}
const webValida = computed(() => /^https?:\/\/\S+\.\S+/.test(web.value.referencia.trim()) && web.value.titulo.trim())
function escolherWeb() {
  escolher({ tipo: 'web', titulo: web.value.titulo.trim(), referencia: web.value.referencia.trim() })
  web.value = { titulo: '', referencia: '' }
}
</script>

<template>
  <div class="cartao pilha escolher">
    <div class="linha entre">
      <div class="abas" role="tablist" aria-label="De onde vem o material">
        <button v-for="a in abas" :key="a.id" type="button" role="tab" :aria-selected="aba === a.id" :class="{ ativa: aba === a.id }" @click="aba = a.id">{{ a.nome }}</button>
        <button v-if="!simples" type="button" disabled class="desativada" title="Quando o Streaming tiver subpastas e legendas">Streaming <span class="etiqueta">em breve</span></button>
      </div>
      <button type="button" class="botao discreto icone" aria-label="Fechar" @click="emit('fechar')"><Icone nome="fechar" :tamanho="16" /></button>
    </div>

    <p v-if="erro" class="aviso sem-margem">{{ erro }}</p>

    <div v-if="!simples && aba !== 'livro'" class="campos">
      <div>
        <label class="rotulo" for="em-papel">Papel na etapa</label>
        <select id="em-papel" v-model="papel" class="campo">
          <option v-for="(nome, v) in PAPEIS" :key="v" :value="v">{{ nome }}</option>
        </select>
      </div>
      <div>
        <label class="rotulo" for="em-nota">Nota <span class="fraco">(opcional)</span></label>
        <input id="em-nota" v-model="nota" class="campo" placeholder="O que ler e por quê" />
      </div>
    </div>

    <!-- Livro da Biblioteca -->
    <template v-if="aba === 'livro'">
      <template v-if="!livro">
        <div class="busca">
          <Icone nome="lupa" :tamanho="16" />
          <input v-model="filtro" class="campo" placeholder="Procurar livro" aria-label="Procurar livro" />
        </div>
        <p v-if="!livros" class="suave sem-margem">Carregando os livros…</p>
        <ul v-else class="lista-livros">
          <li v-for="l in livrosFiltrados" :key="l.id">
            <button type="button" class="livro" @click="abrirLivro(l)">
              <img v-if="l.has_cover" :src="`/api/biblioteca/livros/${encodeURIComponent(l.id)}/capa`" alt="" loading="lazy" />
              <span v-else class="sem-capa"><Icone nome="livro" :tamanho="18" /></span>
              <span><strong>{{ l.title }}</strong><br /><span class="suave">{{ l.author || '—' }} · {{ l.format.toUpperCase() }}</span></span>
            </button>
          </li>
          <li v-if="!livrosFiltrados.length" class="suave">Nenhum livro{{ filtro ? ' com esse nome' : '' }}.</li>
        </ul>
        <div class="pdf pilha">
          <span class="rotulo sem-margem">Um PDF seu</span>
          <p class="pequeno sem-margem">Ele sobe para a Biblioteca (ganha busca, capítulos e o leitor) e a trilha liga a ele.</p>
          <div class="linha enviar">
            <input v-model="tituloPdf" class="campo" placeholder="Título (opcional; o PDF não traz)" aria-label="Título do PDF" />
            <label class="botao" :class="{ ocupado: enviando }">
              <Icone nome="mais" /> {{ enviando ? 'Enviando…' : 'Enviar PDF' }}
              <input type="file" accept=".pdf,.epub,application/pdf,application/epub+zip" class="so-leitor" :disabled="enviando" @change="enviarPdf" />
            </label>
          </div>
        </div>
      </template>

      <template v-else>
        <div class="linha entre">
          <strong>{{ livro.title }}</strong>
          <button type="button" class="botao discreto" @click="livro = null">Outro livro</button>
        </div>
        <p v-if="carregandoLivro" class="suave sem-margem">Lendo o sumário…</p>
        <template v-else-if="livro.chapters">
          <p v-if="!livro.indexed" class="aviso neutro sem-margem">A Biblioteca ainda não tem o texto deste livro{{ livro.error ? ` (${livro.error})` : '' }}. Dá para escolher; a IA recebe o texto quando ela terminar de ler, e até lá usa só o sumário.</p>
          <template v-if="livro.chapters.length">
            <p class="pequeno sem-margem">Marque os capítulos da etapa, ou nenhum para o livro inteiro.</p>
            <ul class="capitulos">
              <li v-for="c in livro.chapters" :key="c.n" :style="{ paddingLeft: `${(c.level - 1) * 16}px` }">
                <label class="linha">
                  <input type="checkbox" :checked="capitulos.has(c.n)" @change="alternarCapitulo(c.n)" />
                  <span>
                    <span class="fraco">{{ c.n }}.</span> {{ c.title }}
                    <span class="fraco"> · p. {{ c.first_page }}–{{ c.last_page }}<template v-if="c.chars"> · {{ mil(c.chars) }} caracteres</template></span>
                    <span v-if="c.highlights" class="etiqueta ambar">{{ c.highlights }} {{ c.highlights > 1 ? 'marcações' : 'marcação' }}</span>
                  </span>
                </label>
              </li>
            </ul>
          </template>
          <div v-else class="campos">
            <p class="pequeno sem-margem">O livro não tem sumário: escolha as páginas (ou nada, para o livro inteiro).</p>
            <div><label class="rotulo" for="em-de">Da página</label><input id="em-de" v-model="paginas.de" class="campo" type="number" min="1" /></div>
            <div><label class="rotulo" for="em-ate">Até</label><input id="em-ate" v-model="paginas.ate" class="campo" type="number" min="1" /></div>
          </div>
          <div v-if="!simples" class="campos">
            <div>
              <label class="rotulo" for="em-papel-l">Papel na etapa</label>
              <select id="em-papel-l" v-model="papel" class="campo">
                <option v-for="(nome, v) in PAPEIS" :key="v" :value="v">{{ nome }}</option>
              </select>
            </div>
            <div>
              <label class="rotulo" for="em-nota-l">Nota <span class="fraco">(opcional)</span></label>
              <input id="em-nota-l" v-model="nota" class="campo" placeholder="O que ler e por quê" />
            </div>
          </div>
          <div class="linha">
            <button type="button" class="botao primario" @click="escolherLivro">Usar {{ localizadorLivro || 'o livro inteiro' }}</button>
          </div>
        </template>
      </template>
    </template>

    <!-- Acervo -->
    <template v-if="aba === 'acervo'">
      <div class="busca">
        <Icone nome="lupa" :tamanho="16" />
        <input v-model="filtro" class="campo" placeholder="Procurar Tema ou Referência" aria-label="Procurar no Acervo" />
      </div>
      <p v-if="!temas" class="suave sem-margem">Carregando o Acervo…</p>
      <template v-else>
        <span class="rotulo sem-margem">Temas <span class="fraco">(as suas notas, revisadas)</span></span>
        <ul class="lista-acervo">
          <li v-for="t in temasFiltrados" :key="t.caminho" :style="{ paddingLeft: `${Math.max(profundidadeDoTema(t), 0) * 16}px` }">
            <button type="button" class="item" @click="escolher({ tipo: 'acervo', titulo: t.titulo, referencia: t.caminho })">
              <strong>{{ t.titulo }}</strong><span v-if="t.resumo" class="suave"> · {{ t.resumo }}</span>
            </button>
          </li>
          <li v-if="!temasFiltrados.length" class="suave">Nenhum Tema{{ filtro ? ' com esse nome' : '' }}.</li>
        </ul>
        <template v-if="referenciasFiltradas.length">
          <span class="rotulo sem-margem">Referências <span class="fraco">{{ filtro ? '' : '(as mais recentes; procure para ver outras)' }}</span></span>
          <ul class="lista-acervo">
            <li v-for="r in referenciasFiltradas" :key="r.id">
              <button type="button" class="item" @click="escolher({ tipo: 'acervo', titulo: r.titulo, referencia: r.id })">
                <span class="etiqueta">{{ TIPOS_REF[r.tipo] || r.tipo }}</span> {{ r.titulo }}
              </button>
            </li>
          </ul>
        </template>
      </template>
    </template>

    <!-- Documento da trilha -->
    <form v-if="aba === 'documento'" class="pilha" @submit.prevent="escolherDocumento">
      <p v-if="!documentos.length" class="suave sem-margem">A trilha não tem documentos. Eles entram na seção Documentos da página da trilha.</p>
      <template v-else>
        <select v-model="doc.referencia" class="campo" aria-label="Documento">
          <option value="" disabled>Escolha o documento</option>
          <option v-for="d in documentos" :key="d.slug" :value="d.slug">{{ d.titulo }}</option>
        </select>
        <input v-model="doc.localizador" class="campo" placeholder="Âncora da seção (opcional; ex.: 23-nat)" aria-label="Âncora da seção" />
        <div><button class="botao primario" type="submit" :disabled="!doc.referencia">Usar o documento</button></div>
      </template>
    </form>

    <!-- Web -->
    <form v-if="aba === 'web'" class="pilha" @submit.prevent="escolherWeb">
      <input v-model="web.titulo" class="campo" placeholder="Título" aria-label="Título" />
      <input v-model="web.referencia" class="campo" type="url" placeholder="https://…" aria-label="Endereço" />
      <div><button class="botao primario" type="submit" :disabled="!webValida">Usar a página</button></div>
      <p class="pequeno sem-margem">Para a IA procurar material na web por você, use "Sugerir material" na página da etapa.</p>
    </form>
  </div>
</template>

<style scoped>
.escolher { gap: var(--e3); }
.sem-margem { margin: 0; }
.entre { justify-content: space-between; flex-wrap: nowrap; }
.abas .desativada { cursor: not-allowed; opacity: 0.55; gap: 6px; }
.lista-livros, .lista-acervo, .capitulos { list-style: none; margin: 0; padding: 0; display: grid; gap: 2px; max-height: 340px; overflow-y: auto; }
.livro, .item {
  display: flex; gap: var(--e3); align-items: center; width: 100%; padding: var(--e2); text-align: left;
  font: inherit; font-size: 14px; line-height: 1.4; color: var(--texto); background: transparent; border: 0; border-radius: var(--raio-p); cursor: pointer;
}
.item { display: block; }
.livro:hover, .item:hover { background: var(--ativo); }
.livro img, .sem-capa { flex: none; width: 32px; height: 44px; object-fit: cover; border-radius: 3px; border: 1px solid var(--borda); }
.sem-capa { display: grid; place-items: center; color: var(--texto-fraco); background: var(--superficie-2); }
.capitulos li { font-size: 14px; line-height: 1.4; padding: 3px 0; }
.capitulos label { align-items: flex-start; flex-wrap: nowrap; }
.capitulos input { margin-top: 3px; }
.pdf { padding-top: var(--e3); border-top: 1px solid var(--borda); gap: var(--e2); }
.enviar { flex-wrap: wrap; }
.enviar .campo { flex: 1 1 200px; width: auto; }
.enviar .botao { cursor: pointer; }
.ocupado { opacity: 0.6; pointer-events: none; }
</style>
