package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"sync"
)

// Fila (T4): uma chamada ao Claude por vez, como no Acervo. As tarefas ficam no banco: se o app
// para no meio, a que estava rodando volta para a fila no próximo início.

const (
	naFila    = "na_fila"
	rodando   = "rodando"
	concluida = "concluida"
	falhou    = "falhou"
	cancelada = "cancelada"
)

type tarefa struct {
	ID          int64  `json:"id"`
	Tipo        string `json:"tipo"`
	PropostaID  int64  `json:"proposta_id"`
	Titulo      string `json:"titulo"`
	Estado      string `json:"estado"`
	Erro        string `json:"erro"`
	Modelo      string `json:"modelo"`
	CriadoEm    string `json:"criado_em"`
	IniciadoEm  string `json:"iniciado_em"`
	TerminadoEm string `json:"terminado_em"`
	// Para a tela: onde está o que a tarefa produz.
	TrilhaID   int64  `json:"trilha_id,omitempty"`
	EtapaChave string `json:"etapa_chave,omitempty"`
}

type executor func(ctx context.Context, t *tarefa) error

type fila struct {
	db         *sql.DB
	executores map[string]executor
	// aoTerminar: depois de cada tarefa (com o estado final), para a proposta acompanhar.
	aoTerminar func(t *tarefa)
	acorda     chan struct{}

	mu       sync.Mutex
	atual    int64
	cancelar context.CancelFunc
}

func novaFila(db *sql.DB) *fila {
	return &fila{db: db, executores: map[string]executor{}, acorda: make(chan struct{}, 1)}
}

func (f *fila) recuperar(ctx context.Context) error {
	_, err := f.db.ExecContext(ctx, `UPDATE tarefas SET estado = ?, iniciado_em = '' WHERE estado = ?`, naFila, rodando)
	return err
}

func (f *fila) enfileirar(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, tipo string, propostaID int64, titulo, modelo string) (int64, error) {
	var proposta any // 0 = tarefa sem proposta (a correção do simulado aponta para a tentativa)
	if propostaID > 0 {
		proposta = propostaID
	}
	res, err := q.ExecContext(ctx, `INSERT INTO tarefas (tipo, proposta_id, titulo, estado, modelo, criado_em)
		VALUES (?, ?, ?, ?, ?, ?)`, tipo, proposta, titulo, naFila, modelo, agora())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// acordar avisa o laço da fila que chegou tarefa (chamar depois do commit).
func (f *fila) acordar() {
	select {
	case f.acorda <- struct{}{}:
	default:
	}
}

func (f *fila) rodar(ctx context.Context) {
	for {
		if !f.rodarUma(ctx) {
			select {
			case <-ctx.Done():
				return
			case <-f.acorda:
			}
		}
	}
}

// rodarUma pega a próxima da fila e executa; false quando não havia nenhuma.
func (f *fila) rodarUma(ctx context.Context) bool {
	t, err := lerTarefa(f.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.estado = ? ORDER BY t.id LIMIT 1`, naFila))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			log.Printf("fila: %v", err)
		}
		return false
	}
	t.Estado, t.IniciadoEm = rodando, agora()
	if _, err := f.db.ExecContext(ctx, `UPDATE tarefas SET estado = ?, iniciado_em = ? WHERE id = ?`, t.Estado, t.IniciadoEm, t.ID); err != nil {
		log.Printf("fila: %v", err)
		return false
	}
	f.executar(ctx, t)
	return true
}

// esvaziar roda tudo o que está na fila, uma por vez (os testes usam no lugar do laço).
func (f *fila) esvaziar(ctx context.Context) {
	for f.rodarUma(ctx) {
	}
}

func (f *fila) executar(ctx context.Context, t *tarefa) {
	exec, ok := f.executores[t.Tipo]
	if !ok {
		f.terminar(t, falhou, "tipo de tarefa desconhecido: "+t.Tipo)
		return
	}
	tctx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	f.atual, f.cancelar = t.ID, cancel
	f.mu.Unlock()

	err := exec(tctx, t)
	cancelou := tctx.Err() != nil && ctx.Err() == nil
	cancel()

	f.mu.Lock()
	f.atual, f.cancelar = 0, nil
	f.mu.Unlock()

	switch {
	case ctx.Err() != nil:
		// O app está parando: a tarefa volta para a fila no próximo início.
	case cancelou:
		f.terminar(t, cancelada, "cancelada")
	case err != nil:
		log.Printf("tarefa %d (%s): %v", t.ID, t.Tipo, err)
		f.terminar(t, falhou, err.Error())
	default:
		f.terminar(t, concluida, "")
	}
}

func (f *fila) terminar(t *tarefa, estado, erro string) {
	t.Estado, t.Erro = estado, erro
	if _, err := f.db.Exec(`UPDATE tarefas SET estado = ?, erro = ?, terminado_em = ? WHERE id = ?`, estado, erro, agora(), t.ID); err != nil {
		log.Printf("fila: %v", err)
	}
	if f.aoTerminar != nil {
		f.aoTerminar(t)
	}
}

var errNaoCancelavel = erroDeEntrada{errors.New("só dá para cancelar tarefa na fila ou rodando")}

func (f *fila) cancelarTarefa(ctx context.Context, id int64) error {
	f.mu.Lock()
	if f.atual == id && f.cancelar != nil {
		f.cancelar()
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()
	t, err := lerTarefa(f.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return errNaoEncontrou
	}
	if err != nil {
		return err
	}
	if t.Estado != naFila {
		return errNaoCancelavel
	}
	f.terminar(t, cancelada, "cancelada")
	return nil
}

const colunasTarefa = `t.id, t.tipo, COALESCE(t.proposta_id, 0), t.titulo, t.estado, t.erro, t.modelo, t.criado_em,
	t.iniciado_em, t.terminado_em, COALESCE(p.trilha_id, 0), COALESCE(p.etapa_chave, '')
	FROM tarefas t LEFT JOIN propostas p ON p.id = t.proposta_id`

func lerTarefa(row interface{ Scan(...any) error }) (*tarefa, error) {
	var t tarefa
	err := row.Scan(&t.ID, &t.Tipo, &t.PropostaID, &t.Titulo, &t.Estado, &t.Erro, &t.Modelo, &t.CriadoEm,
		&t.IniciadoEm, &t.TerminadoEm, &t.TrilhaID, &t.EtapaChave)
	return &t, err
}

func (a *app) handleTarefas(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+colunasTarefa+` ORDER BY t.id DESC LIMIT 50`)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer rows.Close()
	lista := []tarefa{}
	for rows.Next() {
		t, err := lerTarefa(rows)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		lista = append(lista, *t)
	}
	responderJSON(w, http.StatusOK, lista)
}

func (a *app) handleCancelarTarefa(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.fila.cancelarTarefa(r.Context(), id); err != nil {
		a.responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
