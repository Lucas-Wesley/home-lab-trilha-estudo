package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Cursos do Telegram na trilha (T43, fase 9 do plano-entrada): um material "curso" é uma ou mais
// pastas de Meus arquivos; a IA encaixa só pelos TÍTULOS das aulas, sem baixar nem transcrever.

// ---------- a árvore (a mesma montagem de frontend/src/pastas.js, em Go) ----------

type aulaArvore struct {
	Chave    string // "link#mensagem" no catálogo, "video:<id>" só no disco
	Nome     string
	Mensagem int
	Link     string
	Tamanho  int64
	Duracao  int // segundos
	Video    *videoArvore
}

type videoArvore struct {
	ID      int64
	Nome    string
	Tamanho int64
	Duracao int
}

type pastaArvore struct {
	Nome    string
	Caminho []string
	Pastas  []*pastaArvore
	Aulas   []aulaArvore
	porNome map[string]*pastaArvore
	Total   int
}

func novaPastaArvore(nome string, caminho []string) *pastaArvore {
	return &pastaArvore{Nome: nome, Caminho: append([]string{}, caminho...), porNome: map[string]*pastaArvore{}}
}

func (p *pastaArvore) pastaDe(caminho []string) *pastaArvore {
	atual := p
	for i, nome := range caminho {
		filha := atual.porNome[nome]
		if filha == nil {
			filha = novaPastaArvore(nome, caminho[:i+1])
			atual.porNome[nome] = filha
			atual.Pastas = append(atual.Pastas, filha)
		}
		atual = filha
	}
	return atual
}

// achar: a pasta do caminho ("Grupo/Curso/Módulo"), como o acharPasta da tela.
func (p *pastaArvore) achar(caminho string) *pastaArvore {
	atual := p
	for _, nome := range strings.FieldsFunc(caminho, func(r rune) bool { return r == '/' }) {
		atual = atual.porNome[nome]
		if atual == nil {
			return nil
		}
	}
	return atual
}

func (p *pastaArvore) somar() int {
	p.Total = len(p.Aulas)
	for _, f := range p.Pastas {
		p.Total += f.somar()
	}
	return p.Total
}

// arvoreDosArquivos: disco + catálogo dos salvos, na ordem das mensagens; as pastas só do disco
// vêm por último, em ordem alfabética. Recebe o /api/meus-videos já decodificado.
func arvoreDosArquivos(dados *meusVideosStreaming) *pastaArvore {
	raiz := novaPastaArvore("", nil)
	porId := map[int64]videoArvore{}
	for _, v := range dados.Videos {
		porId[v.ID] = videoArvore{ID: v.ID, Nome: v.Nome, Tamanho: v.Tamanho, Duracao: int(v.Duracao)}
	}
	doCatalogo := map[int64]bool{}
	for _, s := range dados.Salvos {
		if !s.Tudo {
			continue
		}
		grupo := s.Pasta
		if grupo == "" {
			grupo = s.Titulo
		}
		if grupo == "" {
			continue
		}
		for _, v := range s.Videos {
			var video *videoArvore
			if vv, ok := porId[v.Video]; ok && v.Video != 0 {
				video = &vv
				doCatalogo[v.Video] = true
			}
			duracao := int(v.Duracao)
			if duracao == 0 && video != nil {
				duracao = video.Duracao
			}
			dest := raiz.pastaDe(append([]string{grupo}, v.Caminho...))
			dest.Aulas = append(dest.Aulas, aulaArvore{
				Chave:    fmt.Sprintf("%s#%d", s.Link, v.Mensagem),
				Nome:     v.Nome,
				Mensagem: v.Mensagem,
				Link:     s.Link,
				Tamanho:  v.Tamanho,
				Duracao:  duracao,
				Video:    video,
			})
		}
	}
	type disco struct {
		v videoArvore
		p string
	}
	var soDisco []disco
	for _, v := range dados.Videos {
		if !doCatalogo[v.ID] {
			soDisco = append(soDisco, disco{v: videoArvore{ID: v.ID, Nome: v.Nome, Tamanho: v.Tamanho, Duracao: int(v.Duracao)}, p: v.Pasta})
		}
	}
	sort.Slice(soDisco, func(i, j int) bool { return soDisco[i].p < soDisco[j].p })
	for _, d := range soDisco {
		dest := raiz
		if d.p != "" {
			dest = raiz.pastaDe(strings.Split(d.p, "/"))
		}
		vv := d.v
		dest.Aulas = append(dest.Aulas, aulaArvore{
			Chave: fmt.Sprintf("video:%d", vv.ID), Nome: vv.Nome, Tamanho: vv.Tamanho, Duracao: vv.Duracao, Video: &vv,
		})
	}
	raiz.somar()
	return raiz
}

// ---------- a referência "telegram:<link>::<caminho>" (várias com " | ") ----------

type cursoRef struct {
	Link    string
	Caminho string
}

func lerCursos(referencia string) []cursoRef {
	var lista []cursoRef
	for _, parte := range strings.Split(referencia, "|") {
		parte = strings.TrimSpace(parte)
		resto, ok := strings.CutPrefix(parte, "telegram:")
		if !ok {
			continue
		}
		link, caminho, ok := strings.Cut(resto, "::")
		if !ok || !strings.HasPrefix(link, "https://t.me/") || strings.TrimSpace(caminho) == "" {
			continue
		}
		lista = append(lista, cursoRef{Link: strings.TrimSpace(link), Caminho: strings.TrimSpace(caminho)})
	}
	return lista
}

func escreverCursos(lista []cursoRef) string {
	var partes []string
	for _, c := range lista {
		partes = append(partes, "telegram:"+c.Link+"::"+c.Caminho)
	}
	return strings.Join(partes, " | ")
}

// ---------- os números A<n> do localizador ("A12", "A12–A15", "A3, A7") ----------

// lerAulas: os números de aula do localizador, na ordem, sem repetir. "A12–A15" abre a faixa.
func lerAulas(localizador string) []int {
	var nums []int
	visto := map[int]bool{}
	add := func(n int) {
		if !visto[n] {
			visto[n] = true
			nums = append(nums, n)
		}
	}
	for _, tok := range strings.FieldsFunc(localizador, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '.' || r == ':'
	}) {
		up := strings.ToUpper(tok)
		if !strings.HasPrefix(up, "A") {
			continue // todo número de aula vem com o prefixo A; "cap. 4" ou "p. 45–60" não conta
		}
		rest := up[1:]
		// Faixa "A12–A15" (hífen ou travessão); o segundo número pode repetir o A.
		if partes := strings.FieldsFunc(rest, func(r rune) bool { return r == '-' || r == '–' }); len(partes) == 2 {
			de, err1 := strconv.Atoi(strings.TrimPrefix(partes[0], "A"))
			ate, err2 := strconv.Atoi(strings.TrimPrefix(partes[1], "A"))
			if err1 == nil && err2 == nil && de > 0 && ate >= de {
				for n := de; n <= ate; n++ {
					add(n)
				}
				continue
			}
		}
		if n, err := strconv.Atoi(rest); err == nil && n > 0 {
			add(n)
		}
	}
	return nums
}

// ---------- o sumário do curso para o pedido ----------

// sumarioDoCurso: a árvore no Streaming → um texto com os níveis como títulos e cada aula como
// "- A<n> · <nome> (12 min)". Os números A<n> seguem entre os cursos; os mapas devolvem
// "A<n>" → referência e "A<n>" → nome, para a aprovação montar os materiais.
func (a *app) sumarioDoCurso(ctx context.Context, referencia string) (string, map[string]string, map[string]string, error) {
	pastas := lerCursos(referencia)
	if len(pastas) == 0 {
		return "", nil, nil, fmt.Errorf("a referência do curso precisa ser telegram:<link>::<caminho>")
	}
	dados, _, err := a.meusVideos(ctx)
	if err != nil {
		return "", nil, nil, fmt.Errorf("o Streaming não respondeu (%s)", err)
	}
	raiz := arvoreDosArquivos(dados)
	var b strings.Builder
	referencias := map[string]string{}
	nomes := map[string]string{}
	n := 0
	aula := func(au aulaArvore) {
		n++
		chave := fmt.Sprintf("A%d", n)
		fmt.Fprintf(&b, "- %s · %s (%d min)\n", chave, au.Nome, (au.Duracao+30)/60)
		nomes[chave] = au.Nome
		if au.Link != "" {
			referencias[chave] = fmt.Sprintf("telegram:%s#%d", au.Link, au.Mensagem)
		} else if au.Video != nil {
			referencias[chave] = fmt.Sprintf("%s/api/video/proprio/%d.mp4", a.integra.streamingLink, au.Video.ID)
		}
	}
	var escrever func(p *pastaArvore, nivel int)
	escrever = func(p *pastaArvore, nivel int) {
		for _, au := range p.Aulas {
			aula(au)
		}
		for _, f := range p.Pastas {
			fmt.Fprintf(&b, "\n%s %s\n\n", strings.Repeat("#", nivel), f.Nome)
			escrever(f, nivel+1)
		}
	}
	for _, ref := range pastas {
		pasta := raiz.achar(ref.Caminho)
		if pasta == nil {
			return "", nil, nil, fmt.Errorf("a pasta %q não existe mais nos arquivos", ref.Caminho)
		}
		fmt.Fprintf(&b, "## %s\n\n", pasta.Nome)
		escrever(pasta, 3)
	}
	return b.String(), referencias, nomes, nil
}
