# Plano: Entrada (vídeos do YouTube e cursos do Telegram nas trilhas)

> Plano escrito em 19/11/2026 para uma IA executar **fase por fase**. Cada fase é um arquivo desta
> pasta. Faça **uma fase por vez**, na ordem, e pare no fim de cada uma: outra IA (o Claude) confere
> e publica antes da próxima.

## 1. O que o Lucas quer (resumo)

Hoje, para encaixar um vídeo do YouTube numa trilha, o vídeo vai primeiro ao **Acervo** (que
transcreve e reescreve o texto com a IA) e depois a Trilha chama a IA de novo para encaixar. Isso
gasta crédito duas vezes. O Lucas quer:

1. **Entrada** (nova área da Trilha): ele cola um link do YouTube (vídeo ou playlist), ou compartilha
   pelo celular, e o vídeo cai na Entrada, **sem passar pelo Acervo**.
2. A Trilha baixa **só a legenda** (yt-dlp, sem IA, com os minutos) e a IA **divide o vídeo em
   fragmentos por assunto** ("06:00–10:00: TLS"). Uma live pode ter 10 assuntos; um vídeo curto, 1.
3. Para cada fragmento, a IA **sugere a(s) trilha(s)** (ou uma trilha nova, ou nenhuma); o Lucas
   confirma ou escolhe à mão (pode ser mais de uma).
4. Para cada trilha escolhida, o **encaixe** (que já existe, T32) diz **em que etapa** o fragmento
   entra, ou propõe uma etapa nova no lugar certo da ordem (o conceito B depois do A).
5. Na etapa, o fragmento **é a aula**: o vídeo do YouTube embutido começa no minuto do início, mostra
   até onde vai o trecho e avisa quando o trecho acabou (sem parar o vídeo). Ver até o fim conta
   como progresso. Não há lição escrita da IA para isso (fica opcional, como já é).
6. **Meus arquivos** (dentro da Entrada): os cursos do Telegram que estão no Streaming (curso ›
   módulo › aula, S55 de lá). Para cada curso, módulo ou aula: só guardar, baixar, apagar (lixeira),
   **criar uma trilha igual ao curso** (sem IA), **encaixar numa trilha pelos títulos** (IA barata),
   **juntar cursos do mesmo assunto** e **processar** (baixar, transcrever e fragmentar como um
   vídeo do YouTube). Nada baixa sozinho: na etapa, a aula que não está no servidor tem "Baixar esta
   aula".

## 2. Decisões já tomadas (não mude)

| # | Decisão |
|---|---|
| D1 | O nome da área é **Entrada** (não "Referências", para não confundir com o Acervo). |
| D2 | A legenda do YouTube é lida **dentro da Trilha** (código copiado do Acervo, `acervo/backend/youtube.go`). Nada de chamar o Acervo para isso. |
| D3 | A IA lê a transcrição **uma vez só** (para fragmentar). Sugerir trilha e encaixar usam os resumos dos fragmentos e só o trecho de transcrição de cada fragmento. |
| D4 | Modelos: fragmentar e sugerir trilha usam um modelo novo de Ajustes, **`modelo_entrada`** (recomendado: Haiku); o encaixe continua com o da lição (Sonnet). |
| D5 | O fragmento é a aula. A etapa com vídeo do YouTube como `principal` não precisa de lição. |
| D6 | Fragmento sem destino fica na Entrada marcado "sem destino" e **some depois de 30 dias**, se o Lucas não o mandar a uma trilha. |
| D7 | Os vídeos do Akita que já estão no Acervo entram na Entrada **pela transcrição do Acervo** (sem baixar de novo e sem IA para ler a legenda). Os encaixes antigos deles na trilha 8 são descartados. |
| D8 | O Streaming continua **dono do Telegram** (sessão, downloads, SSD). A Trilha só conversa com ele pela API. |
| D9 | Nada baixa sozinho do Telegram. Download só por clique; a lixeira apaga do servidor e a aula continua no catálogo. |
| D10 | Tudo que a IA decide é **proposta** (ou sugestão marcada) que o Lucas confirma (T12). |

## 3. Como trabalhar (regras para você, IA que executa)

1. **Antes de qualquer fase**, leia: `CLAUDE.md`, `docs/DECISOES.md` (seções 1, 2, 6 e 7, e as
   decisões T27–T34), e o arquivo da fase. Leia também os arquivos de código que a fase cita.
2. **Uma fase por vez.** Não adiante trabalho de outra fase. Se algo da fase parecer errado ou
   impossível, **pare e escreva a dúvida** no fim do arquivo da fase (seção "Dúvidas"), em vez de
   inventar.
3. **Siga o estilo do código que já existe**: Go só com a biblioteca padrão (um pacote, `backend/`),
   SQLite, Vue 3 em JavaScript (sem TypeScript), nomes e comentários **em português**, como no resto.
   Procure uma função parecida que já existe e copie o jeito dela (erros com `erroDeEntrada`,
   respostas com `responderJSON`/`responderErro`, rotas em `servidor.go`, tabelas em `banco.go`,
   colunas novas na lista de `ALTER TABLE` de `banco.go`).
4. **Tela nova = dentro de `<Pagina>`** (`frontend/src/Pagina.vue`): é ele que dá a margem, a
   largura e a navegação. Copie o começo de uma tela que já existe (ex.: `telas/ListaTrilhas.vue`).
   Olhe toda tela nova em 360 px de largura antes do commit.
5. **Testes são obrigatórios.** Cada fase lista os testes. Rode tudo antes de dizer que terminou:
   - `cd backend && go vet ./... && go test ./...`
   - `cd frontend && npm test && npm run build`
   Se Go ou Node não estiverem no Mac, use os containers: `docker compose -f compose.dev.yaml run --rm api go test ./...`
   e `docker compose -f compose.dev.yaml run --rm web npm test`.
6. **Nunca gaste a IA de verdade nem a internet nos testes.** O Claude nos testes é o
   `scripts/claude-falso.mjs` (estenda-o para as tarefas novas). O yt-dlp nos testes é um script
   falso em `backend/testdata/` (como o do Acervo, `acervo/backend/testdata/yt-dlp-falso`). O
   Streaming e o Acervo nos testes são servidores falsos (`httptest`), como os da Biblioteca e do
   Acervo em `material_test.go`.
7. **Não publique.** Não rode `./deploy.sh`, não faça `git push`. No fim da fase, faça **um commit**
   na `main` local com a mensagem que a fase indica (autoria do Lucas, **sem** `Co-Authored-By`).
8. **Documente**: cada fase acrescenta uma decisão nova em `docs/DECISOES.md` (tabela da seção 7,
   próximo número livre: T35, T36…), no mesmo estilo das outras, e atualiza `CLAUDE.md` com uma seção
   curta quando cria arquivo ou conceito novo. Mudou o que a IA recebe ou devolve: atualize a regra
   em `regras/` e os testes.
9. **Não mexa** em: outros repositórios (Streaming, Acervo), `deploy.sh`, `compose.prod.yaml` (a
   não ser que a fase diga), dados em `data/`, regras de IA que a fase não cita.
10. **Repo público**: nenhum link de grupo do Telegram, token, senha ou dado pessoal no código, nos
   testes ou nos commits. Nos testes, só dados inventados.
10. Ao terminar, escreva no fim do arquivo da fase uma seção **"Feito"** com: o que foi feito, os
    arquivos mudados, os comandos de teste rodados e o resultado, e o que ficou de fora.

## 4. As fases

| Fase | Arquivo | O que entrega | IA? | Situação |
|---|---|---|---|---|
| 1 | [F01-legenda-do-youtube.md](F01-legenda-do-youtube.md) | A Trilha lê vídeo e playlist do YouTube (legenda com minutos, capítulos) e guarda como **fonte** | não | ✅ no ar (92aa4d2) |
| 2 | [F02-tela-entrada.md](F02-tela-entrada.md) | Tela **Entrada**: colar link, compartilhar do celular, lista e página da fonte | não | ✅ no ar (f9477be) |
| 3 | [F03-fragmentar.md](F03-fragmentar.md) | A IA divide a fonte em **fragmentos**; o Lucas ajusta e confirma | sim | ✅ no ar |
| 4 | [F04-destino-e-encaixe.md](F04-destino-e-encaixe.md) | A IA sugere as trilhas de cada fragmento; o encaixe por trilha | sim | ✅ no ar |
| 5 | [F05-trecho-do-youtube-na-etapa.md](F05-trecho-do-youtube-na-etapa.md) | O trecho do YouTube toca na etapa, com aviso do fim e progresso | não | ✅ no ar |
| 6 | [F06-sem-destino-e-acervo.md](F06-sem-destino-e-acervo.md) | "Sem destino" some em 30 dias; trazer do Acervo (os vídeos do Akita) | não | ✅ no ar |
| 7 | [F07-meus-arquivos.md](F07-meus-arquivos.md) | **Meus arquivos**: a árvore dos cursos do Streaming, baixar e lixeira | não | ✅ no ar |
| 8 | [F08-trilha-igual-ao-curso.md](F08-trilha-igual-ao-curso.md) | Criar uma trilha igual a um curso; a aula baixa na etapa, por clique | não | ✅ no ar |
| 9 | [F09-encaixar-cursos-pelos-titulos.md](F09-encaixar-cursos-pelos-titulos.md) | Encaixar cursos numa trilha pelos títulos; juntar cursos | sim | ✅ no ar |
| 10 | [F10-processar-aula.md](F10-processar-aula.md) | Processar aulas do Telegram: transcrever e fragmentar como o YouTube | sim | ✅ |

Fora deste plano (ficam com o Claude): os roteiros das trilhas novas (DDD, Conhecimento de base,
Algoritmos e Estruturas de Dados, Linux, Golang, AWS para engenharia de software, System Design) e o
"stream" do Telegram sem baixar antes (no Streaming).

Depois de cada fase, o Claude confere, às vezes acerta detalhes pequenos num commit próprio
(leia o `git log` antes de começar a fase seguinte) e marca a fase como feita na tabela acima.

## 5. Vocabulário novo

- **Fonte:** um vídeo que entrou na Entrada (do YouTube, do Acervo ou uma aula do Telegram), com a
  transcrição e os minutos.
- **Fragmento:** um trecho da fonte sobre um assunto: início, fim, título, resumo, assuntos e tipo
  (`conteudo`, `opiniao` ou `fora` do tema). É ele que vai para as trilhas.
- **Destino:** a trilha (ou trilhas) para onde um fragmento vai.
- **Trecho na etapa:** um material do tipo `youtube` numa etapa, com `localizador` = `mm:ss–mm:ss`.
