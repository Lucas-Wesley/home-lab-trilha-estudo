# Regras: criar uma trilha

Como montar uma trilha de estudo com qualidade. Vale para quem monta: hoje, o Claude Code à mão
(trilha piloto); na fase 4, a IA do app, que recebe este arquivo como regra. As lições aprendidas
em cada trilha criada entram aqui (seção 6) e o histórico fica em `docs/CRIACAO-DE-TRILHAS.md`.

**Princípio:** uma trilha não é criada, é **iterada**. Escreve-se a versão 1, revisa-se contra os
critérios (seção 4), registra-se o que falhou, corrige-se e revisa-se de novo, até passar. Cada
rodada fica registrada (o que se achou, o que se mudou e por quê).

---

## 1. Entrada

Antes de escrever qualquer etapa, deixe por escrito:

1. **Objetivo** com verbo observável ("ao final, você será capaz de explicar / configurar /
   diagnosticar…"). Nunca "conhecer" ou "entender X" sozinho.
2. **Recorte**: o que entra **e o que fica de fora**, com o motivo.
3. **Nível de partida** (o que a pessoa já domina: pare de descer nos fundamentos ao chegar nisso)
   e **nível alvo**.
4. **Profundidade** da trilha (1 a 5, `docs/PEDAGOGIA.md` §3).
5. **Material disponível**: documentos próprios, livros, Temas do Acervo, vídeos. Leia o material
   antes de planejar. Não planeje pelo título dos arquivos.

## 2. Pesquisa

1. **Currículo de referência.** Ache 2 ou 3 fontes reconhecidas que ensinam o assunto em ordem
   (sumário de livro-texto, objetivos de certificação, documentação oficial com trilha de
   aprendizado). Elas servem para conferir **ordem** e **cobertura**, não para copiar.
2. **Inventário do material próprio**: liste as seções de cada documento e os conceitos que cada
   uma ensina. Material feito na ordem da prática (diário de bordo) quase nunca está na ordem
   conceitual: espere conceitos usados antes de explicados.
3. **Mapa de cobertura**: conceito × onde é ensinado (material próprio, fonte externa, nenhum).
   Os "nenhum" são as lacunas.
4. **Veracidade**: confira na fonte primária (padrão, documentação oficial, banco de dados
   regulatório) as afirmações 🔴 com números, datas, faixas e nomes. Erro achado vira **errata**:
   nota na etapa + correção proposta para o documento. Nunca corrija o material em silêncio.

## 3. Montagem

1. **Ordem conceitual, não cronológica**: cada conceito 🔴 é ensinado antes de ser usado. O
   material da prática entra como aplicação.
2. **Etapa = uma ideia central** que cabe numa sessão: **20 a 90 minutos**. Maior que isso,
   divida.
3. **Módulo = 2 a 6 etapas** com um tema comum.
4. **Cada etapa tem:**
   - objetivo com verbo de Bloom coerente com o nível marcado;
   - pré-requisitos (só os diretos; nada transitivo);
   - **material principal** (pelo menos um, apontando a **seção exata**, não o documento inteiro
     quando só uma parte serve);
   - critério de conclusão **verificável** ("responder às perguntas X sem olhar", "explicar Y com
     um diagrama", "achar Z na LuCI");
   - tempo estimado = leitura (cerca de 120 palavras por minuto em texto técnico) + teste + prática
     de observação;
   - nota no material dizendo **o que ler e para quê** quando a seção não é óbvia.
5. **Lacuna**: preencha com fonte primária confiável ou com uma lição curta escrita para a etapa,
   citando as fontes. Não deixe etapa sem material principal.
6. **Etapa de base** (tipo `base`): pré-requisito que o material próprio supõe sem ensinar.
7. **Fora do recorte**: o que aparece no material mas não é do recorte fica de fora, e a exclusão é
   registrada (para não parecer esquecimento).

## 4. Critérios de revisão (a cada rodada)

O script `scripts/revisar_trilha.py` confere os automáticos (A); os manuais (M) são conferidos
lendo, e o resultado vai para o registro da rodada.

| # | Critério | Tipo |
|---|---|---|
| C1 | Objetivo da trilha com verbo observável; recorte com o que fica de fora | M |
| C2 | Toda etapa tem objetivo com verbo de Bloom, coerente com o nível marcado | A + M |
| C3 | Etapa entre 20 e 90 min | A |
| C4 | Módulo com 2 a 6 etapas | A |
| C5 | Nenhum conceito 🔴 é usado antes da etapa que o ensina (mapa de conceitos) | M |
| C6 | Cada conceito central é ensinado em uma etapa só; revisitas são explícitas | M |
| C7 | Cobertura conferida contra o currículo de referência; lacunas preenchidas ou excluídas com motivo | M |
| C8 | Toda seção em escopo do material próprio é usada por alguma etapa (ou excluída com motivo) | A |
| C9 | Toda etapa tem material principal | A |
| C10 | Toda âncora de documento existe | A |
| C11 | Todo link web responde (proteção anti-robô = "não verificável", registrar) | A |
| C12 | Fontes externas preferem primárias e trazem o rótulo `[oficial]` `[padrão]` `[livro]` `[artigo]` `[vídeo]` na nota | A + M |
| C13 | Afirmações 🔴 conferidas; errata registrada | M |
| C14 | Tempo declarado coerente com a leitura medida (diferença de até 50%) | A |
| C15 | Critério de conclusão presente e verificável | A + M |

Uma rodada **passa** quando não há falha em C1–C15. Avisos (ex.: link que bloqueia robô) podem
ficar, registrados.

## 5. Registro

Cada rodada registra: versão, o que o script achou, o que a leitura achou, o que mudou e por quê.
O registro fica no conteúdo, junto da trilha, e as lições gerais vão para a seção 6.

## 6. Lições aprendidas (cresce a cada trilha)

Ver `docs/CRIACAO-DE-TRILHAS.md` para o contexto de cada uma.

1. **Documentos de prática estão na ordem do dia, não do conceito.** Na trilha piloto, NAT era
   "revisitado" num documento sem nunca ter sido ensinado antes. Montar o mapa de conceitos (C5)
   antes da ordem das etapas.
2. **"Citado" não é "ensinado".** Contar menções a um termo (ex.: DNS aparecia 13 vezes) não prova
   cobertura: as menções eram regras de firewall, nenhuma explicava como a resolução funciona.
   Ler a seção antes de marcar o conceito como coberto.
3. **Regras e números dependem do país.** Canais de Wi-Fi, DFS e potência mudam por país: conferir
   no banco regulatório (no Linux, o `wireless-regdb`) e não em matérias de imprensa, que divergem.
4. **Bloqueio de robô depende do cliente, não só do site.** A documentação do OpenWrt bloqueou a
   ferramenta de leitura da IA, mas respondeu ao revisor; as páginas da Cloudflare (Learning)
   respondem 403 a qualquer script. O revisor registra 403 com página de proteção como "não
   verificável", e a referência só fica se houver outra evidência de que existe (resultado de
   busca) **e** se não for material principal.
5. **Etapa magra infla o tempo.** Na v1 da piloto, 21 etapas tinham de 89 a 177 palavras de leitura
   principal (1 a 2 min) e estavam declaradas com 20 a 30 min. Fatiar demais parece organizado e
   esconde que não há o que estudar. Medir a leitura (o revisor faz) e juntar etapas abaixo de
   ~250 palavras; o tempo declarado fica entre 0,6× e 1,6× do medido.
6. **O critério de conclusão mede o nível do objetivo.** Objetivo "aplicar" com critério "listar as
   regras" mede "lembrar". Critério bom pede o mesmo verbo do objetivo num caso novo ("escrever as
   regras para uma rede nova de câmeras").
7. **Reler a ordem dentro do módulo depois de mexer.** Ao juntar etapas, a lição de roteamento
   passou a mostrar `br-lan` antes da etapa que explica bridge. Toda reordenação pede nova checagem
   de C5 (termo usado antes de explicado), inclusive nos exemplos e nas tabelas.
8. **As seções de prática e de teste do material próprio valem ouro.** Na v1 elas estavam
   excluídas ("não é conteúdo"); na v2 viraram o critério de conclusão das etapas e o material do
   módulo de diagnóstico, que o currículo de referência mostrou faltar.
9. **Errata visível, nunca silenciosa.** O erro achado no material (NAT entre redes internas) virou
   nota no material da etapa + explicação correta numa lição + correção proposta ao dono do
   documento. Correção mecânica e inequívoca (links quebrados) pode ser aplicada direto, avisando.
