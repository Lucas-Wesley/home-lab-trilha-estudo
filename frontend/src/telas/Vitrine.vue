<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Icone from '../Icone.vue'
import Pagina from '../Pagina.vue'

// A vitrine viva do design system (docs/DESIGN-SYSTEM.md): mudou o design, muda aqui também.
const grupos = [
  { nome: 'Fundamentos', secoes: [
    { id: 'cores', nome: 'Cores' },
    { id: 'tipografia', nome: 'Tipografia' },
  ] },
  { nome: 'Componentes', secoes: [
    { id: 'botoes', nome: 'Botões' },
    { id: 'campos', nome: 'Campos' },
    { id: 'abas', nome: 'Abas e seletores' },
    { id: 'etiquetas', nome: 'Etiquetas' },
    { id: 'cartoes', nome: 'Cartões' },
    { id: 'lista', nome: 'Lista em tabela' },
    { id: 'codigo', nome: 'Código' },
    { id: 'painel', nome: 'Painel central' },
  ] },
  { nome: 'Padrões', secoes: [
    { id: 'pedido', nome: 'Pedido e proposta' },
    { id: 'leitura', nome: 'Leitura' },
  ] },
]
const secoes = grupos.flatMap((g) => g.secoes)

const cores = [
  ['--fundo', 'página'], ['--superficie', 'cartões, código'], ['--superficie-2', 'campos, hover'],
  ['--ativo', 'item ativo'], ['--borda', 'bordas'], ['--borda-forte', 'borda em destaque'],
  ['--texto', 'texto'], ['--texto-suave', 'descrições'], ['--texto-fraco', 'rótulos'],
  ['--primario', 'botão principal'], ['--link', 'links e foco'],
]
const etiquetas = ['azul', 'verde', 'vermelho', 'ambar', 'cinza']

const linguagem = ref('Go')
const linguagens = ['Go', 'JavaScript', 'SQL']
const exemplos = {
  Go: 'func handleHealth(db *sql.DB) http.HandlerFunc {\n\treturn func(w http.ResponseWriter, r *http.Request) {\n\t\tresponderJSON(w, 200, map[string]string{"status": "ok"})\n\t}\n}',
  JavaScript: "const rota = lerRota(location.hash)\n// { secao: 'trilhas', partes: ['home-lab-rede'] }",
  SQL: "SELECT titulo, status FROM etapas\nWHERE trilha_id = ? ORDER BY ordem;",
}

const pedido = ref('Foque em segurança: firewall e isolamento entre as redes. Exemplos sempre com o meu OpenWrt.')

// "Nesta página": a seção visível fica marcada.
const atual = ref(secoes[0].id)
let observador
onMounted(() => {
  observador = new IntersectionObserver(
    (entradas) => {
      const visivel = entradas.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0]
      if (visivel) atual.value = visivel.target.id
    },
    { rootMargin: '-64px 0px -70% 0px' },
  )
  secoes.forEach((s) => {
    const el = document.getElementById(s.id)
    if (el) observador.observe(el)
  })
})
onBeforeUnmount(() => observador?.disconnect())

function irPara(id) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  atual.value = id
}
</script>

<template>
  <Pagina>
    <template #nav>
      <div v-for="g in grupos" :key="g.nome" class="nav-grupo">
        <span class="nav-titulo">{{ g.nome }}</span>
        <a
          v-for="s in g.secoes"
          :key="s.id"
          class="nav-item"
          :class="{ ativo: atual === s.id }"
          href="#/design"
          @click.prevent="irPara(s.id)"
        >{{ s.nome }}</a>
      </div>
    </template>

    <nav class="migalhas" aria-label="Você está em">
      <a href="#/design">Design</a>
      <Icone nome="seta-direita" :tamanho="14" />
      <span>Design system</span>
    </nav>
    <div class="cabecalho">
      <h1>Design system</h1>
      <a class="botao discreto" href="https://github.com/Lucas-Wesley/home-lab-trilha-estudo/blob/main/docs/DESIGN-SYSTEM.md" target="_blank" rel="noopener">
        <Icone nome="link" /> Ver o documento
      </a>
    </div>
    <p class="lead">Escuro, neutro e quieto; pequeno e denso, mas respirando. Inspirado nas Docs, no Console e no Cookbook da Claude Platform. Esta página é a vitrine viva de <code>docs/DESIGN-SYSTEM.md</code>.</p>

    <!-- Cores -->
    <section id="cores" class="secao">
      <h2>Cores</h2>
      <p class="suave">Tokens em <code>style.css</code>. Troque o tema no botão da barra para ver o claro.</p>
      <div class="amostras">
        <div v-for="[token, uso] in cores" :key="token" class="amostra">
          <span class="cor" :style="{ background: `var(${token})` }" />
          <div>
            <code>{{ token }}</code>
            <div class="pequeno">{{ uso }}</div>
          </div>
        </div>
      </div>
    </section>

    <!-- Tipografia -->
    <section id="tipografia" class="secao">
      <h2>Tipografia</h2>
      <div class="pilha tipos">
        <div><span class="pequeno">Título · Source Serif 4 · 30</span><h1>Redes separadas e firewall</h1></div>
        <div><span class="pequeno">Seção · Source Serif 4 · 23</span><h2>Zonas e regras</h2></div>
        <div><span class="pequeno">Subseção · Inter · 18/600</span><h3>Input, Output e Forward</h3></div>
        <div><span class="pequeno">Destaque · Inter · 17</span><p class="lead">Como o roteador decide o que passa entre as redes.</p></div>
        <div><span class="pequeno">Leitura · Inter · 16/1,7</span><p class="leitura">Uma <strong>zona</strong> agrupa interfaces que recebem o mesmo tratamento. A regra de <code>forward</code> diz se o tráfego pode atravessar de uma zona para outra.</p></div>
        <div><span class="pequeno">Interface · Inter · 14</span><p>Trilhas ativas · 3 etapas para hoje</p></div>
        <div><span class="pequeno">Código · JetBrains Mono · 13,5</span><p><code>uci show firewall.@zone[1]</code></p></div>
      </div>
    </section>

    <!-- Botões -->
    <section id="botoes" class="secao">
      <h2>Botões</h2>
      <p class="suave">32 px (36 no toque). Um principal por tela, invertido.</p>
      <div class="linha">
        <button class="botao primario">Criar trilha</button>
        <button class="botao">Secundário</button>
        <button class="botao discreto"><Icone nome="copiar" /> Copiar página</button>
        <div class="botao-dividido">
          <button class="botao"><Icone nome="copiar" /> Copiar</button>
          <button class="botao" aria-label="Mais opções"><Icone nome="seta-baixo" /></button>
        </div>
        <button class="botao icone" aria-label="Adicionar"><Icone nome="mais" /></button>
        <button class="botao" disabled>Desativado</button>
      </div>
    </section>

    <!-- Campos -->
    <section id="campos" class="secao">
      <h2>Campos</h2>
      <div class="pilha estreito">
        <label class="busca">
          <Icone nome="lupa" />
          <span class="so-leitor">Buscar</span>
          <input class="campo" placeholder="Buscar nas trilhas…" />
          <kbd>Ctrl K</kbd>
        </label>
        <div>
          <label class="rotulo" for="v-tema">Tema</label>
          <input id="v-tema" class="campo" placeholder="Ex.: História do Brasil" />
        </div>
        <div>
          <label class="rotulo" for="v-texto">Texto livre</label>
          <textarea id="v-texto" class="campo" placeholder="Escreva com as suas palavras…" />
          <p class="ajuda">Ajuda curta embaixo do campo, em cinza fraco.</p>
        </div>
      </div>
    </section>

    <!-- Abas -->
    <section id="abas" class="secao">
      <h2>Abas e seletores</h2>
      <div class="linha">
        <div class="abas">
          <button v-for="l in linguagens" :key="l" :class="{ ativa: linguagem === l }" @click="linguagem = l">{{ l }}</button>
        </div>
        <button class="seletor">Profundidade <strong>Sólido</strong> <Icone nome="seta-baixo" :tamanho="14" /></button>
      </div>
    </section>

    <!-- Etiquetas -->
    <section id="etiquetas" class="secao">
      <h2>Etiquetas</h2>
      <div class="linha">
        <span v-for="c in etiquetas" :key="c" class="etiqueta" :class="c">{{ c }}</span>
      </div>
      <div class="linha" style="margin-top: var(--e3)">
        <span class="etiqueta verde">Concluída</span>
        <span class="etiqueta azul">Em andamento</span>
        <span class="etiqueta ambar">Aguardando você</span>
        <span class="etiqueta vermelho">Atrasada</span>
        <span class="etiqueta">Bloqueada</span>
      </div>
    </section>

    <!-- Cartões -->
    <section id="cartoes" class="secao">
      <h2>Cartões</h2>
      <div class="grade">
        <a class="cartao" href="#/design" @click.prevent>
          <Icone nome="trilhas" :tamanho="28" class="cartao-icone" />
          <h3>Home lab: rede</h3>
          <p>Firmware do roteador, Wi-Fi, redes separadas, firewall, WAN e SQM, a partir dos documentos do estudo.</p>
        </a>
        <a class="cartao" href="#/design" @click.prevent>
          <Icone nome="livro" :tamanho="28" class="cartao-icone" />
          <h3>Sistemas operacionais</h3>
          <p>Trilha com um livro como espinha dorsal: capítulos em ordem, com complementos onde o livro é raso.</p>
        </a>
        <a class="cartao" href="#/design" @click.prevent>
          <Icone nome="video" :tamanho="28" class="cartao-icone" />
          <h3>Gramática normativa</h3>
          <p>As aulas de Meus vídeos do Streaming, quando tiverem subpastas e legendas.</p>
        </a>
      </div>
    </section>

    <!-- Lista -->
    <section id="lista" class="secao">
      <h2>Lista em tabela</h2>
      <table class="tabela">
        <thead>
          <tr><th>Etapa</th><th>Situação</th><th class="direita">Prazo</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><div class="titulo">Bridges e interfaces</div><div class="descricao">Como uma bridge junta portas e Wi-Fi numa mesma rede.</div></td>
            <td><div class="etiquetas"><span class="etiqueta verde">Concluída</span></div></td>
            <td class="direita fraco">12 nov</td>
          </tr>
          <tr>
            <td><div class="titulo">Zonas de firewall</div><div class="descricao">Input, Output e Forward; por que a IoT não fala com a rede principal.</div></td>
            <td><div class="etiquetas"><span class="etiqueta azul">Em andamento</span><span class="etiqueta ambar">Proposta v2</span></div></td>
            <td class="direita fraco">14 nov</td>
          </tr>
          <tr>
            <td><div class="titulo">NAT e CGNAT</div><div class="descricao">Etapa de base inserida antes de WAN (deep dive).</div></td>
            <td><div class="etiquetas"><span class="etiqueta">Bloqueada</span></div></td>
            <td class="direita fraco">16 nov</td>
          </tr>
        </tbody>
      </table>
    </section>

    <!-- Código -->
    <section id="codigo" class="secao">
      <h2>Código</h2>
      <div class="codigo">
        <div class="codigo-topo">
          <div class="abas">
            <button v-for="l in linguagens" :key="l" :class="{ ativa: linguagem === l }" @click="linguagem = l">{{ l }}</button>
          </div>
          <button class="botao discreto icone" aria-label="Copiar o código"><Icone nome="copiar" /></button>
        </div>
        <pre><code>{{ exemplos[linguagem] }}</code></pre>
      </div>
    </section>

    <!-- Painel -->
    <section id="painel" class="secao">
      <h2>Painel central</h2>
      <div class="palco fundo-pontos">
        <div class="painel pilha">
          <h3 class="centro">Comece uma trilha</h3>
          <button class="botao largo">Montar à mão</button>
          <span class="pequeno centro">ou</span>
          <input class="campo" placeholder="O que você quer aprender?" />
          <button class="botao primario largo">Continuar com a IA</button>
          <p class="pequeno centro">A IA sempre propõe; nada vale antes de você aprovar.</p>
        </div>
      </div>
    </section>

    <!-- Pedido e proposta -->
    <section id="pedido" class="secao">
      <h2>Pedido e proposta</h2>
      <p class="suave">Toda ação da IA aceita um pedido em texto livre (antes), e toda proposta aceita comentários gerais ou ancorados (depois). Ver T13 em <code>docs/DECISOES.md</code>.</p>

      <div class="cartao cheio pilha">
        <div>
          <label class="rotulo" for="v-pedido">Pedido para a IA <span class="fraco">(opcional)</span></label>
          <textarea id="v-pedido" v-model="pedido" class="campo" />
          <p class="ajuda">Somado às instruções globais e às da trilha.</p>
        </div>
        <div class="linha">
          <button class="botao primario">Gerar o plano</button>
          <button class="botao discreto">Sem pedido</button>
        </div>
      </div>

      <div class="proposta">
        <div class="linha">
          <h3>Plano da trilha</h3>
          <span class="etiqueta ambar">v2 · aguardando você</span>
        </div>
        <p class="leitura">Etapa 4 — <span class="trecho-comentado">Zonas de firewall: Input, Output e Forward</span>, com o caminho completo de um pacote da rede IoT até a internet.</p>
        <div class="pilha">
          <div class="comentario">
            <div class="comentario-topo"><strong>Lucas</strong><span class="fraco">· ancorado na etapa 4</span></div>
            <span class="ancora">“Zonas de firewall: Input, Output e Forward”</span>
            <p>Faltou o porquê de a IoT não poder iniciar conexão com a rede principal. Aprofunde aqui.</p>
          </div>
          <div class="comentario">
            <div class="comentario-topo"><strong>IA</strong><span class="etiqueta verde">feito na v2</span></div>
            <p>Acrescentei a seção “Por que a ida é bloqueada e a volta não” com o estado das conexões (conntrack).</p>
          </div>
        </div>
        <div class="linha">
          <button class="botao primario">Aprovar</button>
          <button class="botao"><Icone nome="comentario" /> Pedir ajuste</button>
          <button class="botao discreto">Descartar</button>
        </div>
      </div>
    </section>

    <!-- Leitura -->
    <section id="leitura" class="secao">
      <h2>Leitura</h2>
      <article class="leitura">
        <p>O <strong>DHCP</strong> distribui endereços automaticamente: quem entra na rede pede um IP emprestado e recebe também o gateway e o DNS. Uma <em>reserva</em> amarra sempre o mesmo IP ao mesmo MAC.</p>
        <h3>Teste seu entendimento</h3>
        <ol>
          <li>Por que o servidor tem IP fora do <em>pool</em> do DHCP?</li>
          <li>O que acontece com a reserva se você trocar a placa de rede?</li>
        </ol>
        <blockquote>Citação: fio à esquerda, texto suave.</blockquote>
        <ul class="verificacao">
          <li><Icone nome="check" /> Lição lida</li>
          <li><Icone nome="check" /> Exercícios ≥ 80%</li>
        </ul>
      </article>
    </section>

    <template #lateral>
      <div class="resumo">
        <div class="linha resumo-topo">
          <strong>Design system</strong>
          <span class="etiqueta ambar">v1</span>
        </div>
        <span class="resumo-rotulo">Fontes</span>
        <ul class="verificacao">
          <li><Icone nome="check" /> Source Serif 4</li>
          <li><Icone nome="check" /> Inter</li>
          <li><Icone nome="check" /> JetBrains Mono</li>
        </ul>
        <span class="resumo-rotulo">Temas</span>
        <div class="linha"><span class="etiqueta">escuro</span><span class="etiqueta">claro</span><span class="etiqueta">auto</span></div>
      </div>
      <ul class="sumario">
        <li v-for="s in secoes" :key="s.id">
          <a href="#/design" :class="{ atual: atual === s.id }" @click.prevent="irPara(s.id)">{{ s.nome }}</a>
        </li>
      </ul>
    </template>
  </Pagina>
</template>

<style scoped>
.secao { margin-top: var(--e7); scroll-margin-top: calc(var(--barra) + var(--e5)); }
.secao > h2 { margin-bottom: var(--e3); }
.secao > p.suave { margin: 0 0 var(--e4); }

.amostras { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: var(--e3); }
.amostra { display: flex; gap: var(--e3); align-items: center; }
.cor { width: 36px; height: 36px; flex: none; border-radius: var(--raio-p); border: 1px solid var(--borda-forte); }

.tipos > div { display: grid; gap: var(--e1); padding-bottom: var(--e4); border-bottom: 1px solid var(--borda); }
.tipos p { margin: 0; }

.estreito { max-width: 440px; }

.cartao-icone { color: var(--texto-suave); margin-bottom: var(--e3); }

.direita { text-align: right; white-space: nowrap; }

.palco { padding: var(--e7) var(--e4); border: 1px solid var(--borda); border-radius: var(--raio); }
.centro { text-align: center; }

.proposta { display: grid; gap: var(--e4); margin-top: var(--e5); padding-top: var(--e5); border-top: 1px solid var(--borda); }
.proposta .leitura { margin: 0; }

.resumo { display: grid; gap: var(--e2); padding: var(--e4); margin-bottom: var(--e5); border: 1px solid var(--borda); border-radius: var(--raio); }
.resumo-topo { justify-content: space-between; margin-bottom: var(--e1); }
.resumo-rotulo { margin-top: var(--e2); font-size: 12.5px; color: var(--texto-fraco); text-decoration: underline; text-underline-offset: 3px; }
</style>
