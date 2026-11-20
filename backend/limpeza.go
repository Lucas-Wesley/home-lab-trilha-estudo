package main

import (
	"context"
	"log"
	"time"
)

// Entrada, fase 6 (T40, D6): o fragmento "sem destino" que o Lucas não manda a nenhuma trilha
// some sozinho depois de 30 dias — uma limpeza ao ligar o app e depois a cada 6 horas.

const diasSemDestino = 30
const pausaLimpeza = 6 * time.Hour

// limparSemDestino: a goroutine — roda já na subida e a cada pausaLimpeza, até o app parar.
func (a *app) limparSemDestino(ctx context.Context) {
	a.rodarLimpeza(ctx)
	t := time.NewTicker(pausaLimpeza)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.rodarLimpeza(ctx)
		}
	}
}

func (a *app) rodarLimpeza(ctx context.Context) {
	if n, err := a.apagarSemDestino(time.Now()); err != nil && ctx.Err() == nil {
		log.Printf("limpeza de fragmentos sem destino: %v", err)
	} else if n > 0 {
		log.Printf("limpeza: %d fragmento(s) sem destino apagados", n)
	}
}

// apagarSemDestino: fragmentos no estado `sem_destino` marcados há mais de diasSemDestino dias
// (o carimbo `sem_destino_em` é RFC3339 UTC, como o agora() — a comparação de texto vale).
// Recebe o "agora" para o teste não precisar esperar.
func (a *app) apagarSemDestino(agora time.Time) (int64, error) {
	limite := agora.AddDate(0, 0, -diasSemDestino).UTC().Format(time.RFC3339)
	res, err := a.db.Exec(`DELETE FROM fragmentos
		WHERE estado = 'sem_destino' AND sem_destino_em != '' AND sem_destino_em < ?`, limite)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
