package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Videoaula (T30): numa trilha feita de um curso em vídeo, o vídeo é o principal. A etapa toca o
// vídeo (material `streaming` principal, `referencia` = o arquivo no Streaming) e guarda as notas
// que o Lucas toma assistindo e o resumo dele no fim. "Conferir com a aula" manda o resumo para a
// IA, que compara com a transcrição (os documentos da etapa) e a lição: o que bate, o que está
// errado, o que faltou e em que minuto rever. A conferência é uma tentativa do tipo `resumo`
// (como a correção do simulado: tarefa sem proposta, falha fica registrada).

const (
	tipoResumo         = "resumo"         // a tentativa
	tipoConferirResumo = "conferir_resumo" // a tarefa da fila
	limiteAnotacao     = 60000             // caracteres, por campo
)

const esquemaConferencia = `{"type":"object","properties":{
"veredito":{"type":"string","enum":["bate","parcial","nao_bate"],"description":"o resumo, no todo, bate com a aula?"},
"nota":{"type":"integer","description":"de 0 a 100: quanto do essencial da aula o resumo traz, e certo"},
"certos":{"type":"array","items":{"type":"string"},"description":"o que o resumo diz certo (pontos curtos)"},
"erros":{"type":"array","items":{"type":"object","properties":{
  "trecho":{"type":"string","description":"o que o Lucas escreveu, citado"},
  "correcao":{"type":"string","description":"o que a aula diz de fato"},
  "onde":{"type":"string","description":"o minuto na aula (mm:ss ou h:mm:ss), se a transcrição tiver; senão vazio"}},
  "required":["trecho","correcao","onde"]}},
"faltou":{"type":"array","items":{"type":"object","properties":{
  "ponto":{"type":"string","description":"o que a aula ensina e o resumo não traz"},
  "onde":{"type":"string","description":"o minuto na aula, se a transcrição tiver; senão vazio"}},
  "required":["ponto","onde"]}},
"comentario":{"type":"string","description":"duas ou três frases: como está o entendimento e o que rever primeiro"}},
"required":["veredito","nota","certos","erros","faltou","comentario"]}`

type anotacoes struct {
	Notas        string      `json:"notas"`
	Resumo       string      `json:"resumo"`
	AtualizadoEm string      `json:"atualizado_em"`
	Conferencias []tentativa `json:"conferencias"`
}

func (a *app) lerAnotacoes(ctx context.Context, e etapa) (anotacoes, error) {
	an := anotacoes{Conferencias: []tentativa{}}
	err := a.db.QueryRowContext(ctx, `SELECT notas, resumo, atualizado_em FROM anotacoes WHERE trilha_id = ? AND etapa_chave = ?`,
		e.TrilhaID, e.Chave).Scan(&an.Notas, &an.Resumo, &an.AtualizadoEm)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return an, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasTentativa+` FROM tentativas
		WHERE trilha_id = ? AND etapa_chave = ? AND tipo = ? ORDER BY id DESC LIMIT 10`, e.TrilhaID, e.Chave, tipoResumo)
	if err != nil {
		return an, err
	}
	defer rows.Close()
	for rows.Next() {
		t, err := lerTentativa(rows)
		if err != nil {
			return an, err
		}
		an.Conferencias = append(an.Conferencias, t)
	}
	return an, rows.Err()
}

func (a *app) handleAnotacoes(w http.ResponseWriter, r *http.Request) {
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	an, err := a.lerAnotacoes(r.Context(), e)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, an)
}

// handleSalvarAnotacoes: a tela salva sozinha enquanto ele escreve; campo ausente fica como está.
func (a *app) handleSalvarAnotacoes(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Notas  *string `json:"notas"`
		Resumo *string `json:"resumo"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	if err := a.salvarAnotacoes(r.Context(), e, corpo.Notas, corpo.Resumo); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleAnotacoes(w, r)
}

func (a *app) salvarAnotacoes(ctx context.Context, e etapa, notas, resumo *string) error {
	for _, s := range []*string{notas, resumo} {
		if s != nil && len([]rune(*s)) > limiteAnotacao {
			return erroDeEntrada{fmt.Errorf("o texto passou de %d caracteres", limiteAnotacao)}
		}
	}
	_, err := a.db.ExecContext(ctx, `INSERT INTO anotacoes (trilha_id, etapa_chave, notas, resumo, atualizado_em) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (trilha_id, etapa_chave) DO UPDATE SET
			notas = CASE WHEN ? THEN excluded.notas ELSE notas END,
			resumo = CASE WHEN ? THEN excluded.resumo ELSE resumo END,
			atualizado_em = excluded.atualizado_em`,
		e.TrilhaID, e.Chave, valorOu(notas), valorOu(resumo), agora(), notas != nil, resumo != nil)
	return err
}

func valorOu(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// handleConferirResumo salva o resumo e põe a conferência na fila.
func (a *app) handleConferirResumo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Resumo *string `json:"resumo"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	e, ok := a.etapaDaRota(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if corpo.Resumo != nil {
		if err := a.salvarAnotacoes(ctx, e, nil, corpo.Resumo); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	an, err := a.lerAnotacoes(ctx, e)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	resumo := strings.TrimSpace(an.Resumo)
	if len([]rune(resumo)) < 40 {
		a.responderErro(w, erroDeEntrada{errors.New("escreva o resumo (algumas frases) antes de conferir")})
		return
	}
	if len(fontesDaConferencia(e)) == 0 && !e.TemLicao {
		a.responderErro(w, erroDeEntrada{errors.New("a etapa não tem transcrição nem lição para conferir o resumo")})
		return
	}
	for _, c := range an.Conferencias {
		if c.Estado == tentCorrigindo {
			a.responderErro(w, erroDeEntrada{errors.New("a IA ainda está conferindo o resumo anterior desta etapa")})
			return
		}
	}
	modelo := a.modeloPara(ctx, tipoConferirResumo)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoConferirResumo, 0, "Conferir o resumo: "+e.Titulo, modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	respostas, _ := json.Marshal(map[string]string{"resumo": resumo, "notas": an.Notas})
	id, err := a.novaTentativa(ctx, e, tipoResumo, tentCorrigindo, 0, 0, 1, string(respostas), "", tarefaID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	a.responderTentativa(w, r, id, http.StatusCreated)
}

// fontesDaConferencia: os documentos da etapa (as transcrições, numa trilha de curso em vídeo).
func fontesDaConferencia(e etapa) []material {
	var lista []material
	for _, m := range e.Materiais {
		if m.Tipo == "documento" && m.Referencia != "" {
			lista = append(lista, m)
		}
	}
	return lista
}

// conferirResumo é a tarefa: a IA lê a transcrição, a lição e o resumo e diz o que bate.
func (a *app) conferirResumo(ctx context.Context, t *tarefa) error {
	var tid, trilhaID int64
	var chave, respostas string
	err := a.db.QueryRowContext(ctx, `SELECT id, trilha_id, etapa_chave, respostas FROM tentativas WHERE tarefa_id = ?`, t.ID).
		Scan(&tid, &trilhaID, &chave, &respostas)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("a conferência desta tarefa não existe mais")
	}
	if err != nil {
		return err
	}
	var dados struct {
		Resumo string `json:"resumo"`
		Notas  string `json:"notas"`
	}
	if err := json.Unmarshal([]byte(respostas), &dados); err != nil {
		return err
	}
	slug := a.slugDaTrilha(ctx, trilhaID)
	e, err := a.etapaPorChave(ctx, trilhaID, chave)
	if err != nil {
		return errors.New("a etapa desta conferência não existe mais")
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)

	arquivos := map[string]string{}
	var s strings.Builder
	fmt.Fprintf(&s, "# Pedido: conferir o resumo da etapa \"%s\"\n\n", e.Titulo)
	if e.Objetivo != "" {
		fmt.Fprintf(&s, "- Objetivo da etapa: %s\n", e.Objetivo)
	}
	for _, m := range e.Materiais {
		if m.Tipo == "streaming" && m.Papel == "principal" {
			fmt.Fprintf(&s, "- Videoaula: %s\n", m.Titulo)
		}
	}
	s.WriteString("\nO Lucas assistiu à aula e escreveu o resumo abaixo com as palavras dele. Confira se o que ele entendeu bate com o que a aula ensina.\n\n")
	if fontes := fontesDaConferencia(e); len(fontes) > 0 {
		s.WriteString("**A aula** (transcrição automática, com o minuto de cada trecho entre colchetes; pode ter erros de reconhecimento de fala):\n\n")
		for _, m := range fontes {
			texto, _ := a.conteudo.ler(caminhoDocumento(slug, m.Referencia))
			if strings.TrimSpace(texto) == "" {
				continue
			}
			nome := "aula/" + m.Referencia + ".md"
			arquivos[nome] = texto
			fmt.Fprintf(&s, "- `%s`: %s\n", nome, m.Titulo)
		}
	}
	if e.TemLicao {
		licao, _ := a.conteudo.ler(caminhoLicao(slug, chave))
		if strings.TrimSpace(licao) != "" {
			arquivos["licao.md"] = licao
			s.WriteString("\n**A lição** da etapa (texto escrito a partir da aula) está em `licao.md`: use para conferir conceitos, mas o que vale é a aula.\n")
		}
	}
	fmt.Fprintf(&s, "\n## Resumo do Lucas\n\n%s\n", dados.Resumo)
	if strings.TrimSpace(dados.Notas) != "" {
		fmt.Fprintf(&s, "\n## Notas que ele tomou assistindo (contexto; não é o que está sendo conferido)\n\n%s\n", dados.Notas)
	}
	arquivos["pedido.md"] = s.String()
	if err := gravarNaPasta(pasta, arquivos); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("conferir-resumo.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read,Glob", esquema: esquemaConferencia, modelo: t.Modelo,
		instrucao: "Confira o resumo do Lucas contra a aula. Tudo está em `pedido.md`, nesta pasta: leia o pedido e a transcrição inteira primeiro.",
	})
	if err != nil {
		return err
	}
	var c struct {
		Veredito string   `json:"veredito"`
		Nota     int      `json:"nota"`
		Certos   []string `json:"certos"`
		Erros    []struct {
			Trecho   string `json:"trecho"`
			Correcao string `json:"correcao"`
			Onde     string `json:"onde"`
		} `json:"erros"`
		Faltou []struct {
			Ponto string `json:"ponto"`
			Onde  string `json:"onde"`
		} `json:"faltou"`
		Comentario string `json:"comentario"`
	}
	if err := dadosDaResposta(resp, &c); err != nil {
		return err
	}
	if !contem([]string{"bate", "parcial", "nao_bate"}, c.Veredito) {
		c.Veredito = "parcial"
	}
	for i := range c.Certos {
		c.Certos[i] = textoDaIA(c.Certos[i])
	}
	for i := range c.Erros {
		c.Erros[i].Trecho, c.Erros[i].Correcao = textoDaIA(c.Erros[i].Trecho), textoDaIA(c.Erros[i].Correcao)
	}
	for i := range c.Faltou {
		c.Faltou[i].Ponto = textoDaIA(c.Faltou[i].Ponto)
	}
	c.Comentario = textoDaIA(c.Comentario)
	if c.Certos == nil {
		c.Certos = []string{}
	}
	resultado, _ := json.Marshal(c)
	nota := min(100, max(0, c.Nota))
	acertos := 0
	if c.Veredito == "bate" {
		acertos = 1
	}
	_, err = a.db.ExecContext(ctx, `UPDATE tentativas SET estado = ?, nota = ?, acertos = ?, resultado = ?, atualizado_em = ? WHERE id = ?`,
		tentCorrigida, nota, acertos, string(resultado), agora(), tid)
	return err
}
