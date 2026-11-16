package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	_ "time/tzdata" // o fuso de São Paulo sem depender do tzdata da imagem
)

// minutosPadrao: etapa sem estimativa ocupa este tempo no cronograma.
const minutosPadrao = 45

const formatoData = "2006-01-02"

var fuso = time.UTC

func carregarFuso(nome string) {
	if loc, err := time.LoadLocation(nome); err == nil {
		fuso = loc
	}
}

// hoje: o dia no fuso do Lucas, à meia-noite (relógio injetável nos testes).
var relogio = time.Now

func hoje() time.Time {
	agora := relogio().In(fuso)
	return time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, fuso)
}

func lerData(s string) (time.Time, error) {
	return time.ParseInLocation(formatoData, s, fuso)
}

func diaDeEstudo(d time.Time, dias []int) bool {
	for _, x := range dias {
		if int(d.Weekday()) == x {
			return true
		}
	}
	return false
}

// capacidadeDiaria: os minutos por dia da meta, ou (sem eles) o total restante dividido pelos dias de
// estudo até a data final, arredondado para cima de 5 em 5.
func capacidadeDiaria(m meta, desde time.Time, totalRestante int) (int, error) {
	if m.Minutos > 0 {
		return m.Minutos, nil
	}
	fim, err := lerData(m.Fim)
	if err != nil {
		return 0, erroDeEntrada{errors.New("defina os minutos por dia ou a data final")}
	}
	dias := 0
	for d := desde; !d.After(fim); d = d.AddDate(0, 0, 1) {
		if diaDeEstudo(d, m.Dias) {
			dias++
		}
	}
	if dias == 0 {
		return 0, erroDeEntrada{errors.New("não há dia de estudo entre hoje e a data final")}
	}
	porDia := (totalRestante + dias - 1) / dias
	if porDia%5 != 0 {
		porDia += 5 - porDia%5
	}
	if porDia < 5 {
		porDia = 5
	}
	return porDia, nil
}

type fatia struct {
	EtapaID int64
	Data    string
	Minutos int
}

// distribuir põe as etapas pendentes, em ordem, nos dias de estudo a partir de "desde", enchendo cada
// dia até a capacidade; uma etapa maior que o que sobra no dia continua no dia seguinte.
func distribuir(etapas []etapa, dias []int, porDia int, desde time.Time) []fatia {
	fatias := []fatia{}
	d := desde
	for !diaDeEstudo(d, dias) {
		d = d.AddDate(0, 0, 1)
	}
	usado := 0
	for _, e := range etapas {
		if e.Status == "concluida" || e.Status == "pulada" {
			continue
		}
		restante := e.Minutos
		if restante <= 0 {
			restante = minutosPadrao
		}
		for restante > 0 {
			if usado >= porDia {
				d, usado = d.AddDate(0, 0, 1), 0
				for !diaDeEstudo(d, dias) {
					d = d.AddDate(0, 0, 1)
				}
			}
			pega := min(restante, porDia-usado)
			fatias = append(fatias, fatia{e.ID, d.Format(formatoData), pega})
			restante -= pega
			usado += pega
		}
	}
	return fatias
}

// errSemDias: a trilha ainda não tem meta (dias de estudo).
var errSemDias = erroDeEntrada{errors.New("escolha os dias da semana de estudo")}

func (a *app) gerarCronograma(ctx context.Context, trilhaID int64, desde time.Time) error {
	t, err := a.trilhaCompleta(ctx, trilhaID)
	if err != nil {
		return err
	}
	if len(t.Meta.Dias) == 0 {
		return errSemDias
	}
	etapas, err := a.etapasDaTrilha(ctx, trilhaID)
	if err != nil {
		return err
	}
	restante := 0
	for _, e := range etapas {
		if e.Status != "concluida" && e.Status != "pulada" {
			restante += max(e.Minutos, 0)
			if e.Minutos <= 0 {
				restante += minutosPadrao
			}
		}
	}
	porDia, err := capacidadeDiaria(t.Meta, desde, restante)
	if err != nil {
		return err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM cronograma WHERE trilha_id = ?`, trilhaID); err != nil {
		return err
	}
	for _, f := range distribuir(etapas, t.Meta.Dias, porDia, desde) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO cronograma (trilha_id, etapa_id, data, minutos) VALUES (?, ?, ?, ?)`,
			trilhaID, f.EtapaID, f.Data, f.Minutos); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------- handlers ----------

func (a *app) handleMeta(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var m meta
	if err := lerCorpo(r, &m); err != nil {
		a.responderErro(w, err)
		return
	}
	if len(lerDias(escreverDias(m.Dias))) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("escolha pelo menos um dia da semana")})
		return
	}
	if m.Minutos < 0 || m.Minutos > 600 {
		a.responderErro(w, erroDeEntrada{errors.New("minutos por dia vão de 5 a 600")})
		return
	}
	if m.Inicio == "" {
		m.Inicio = hoje().Format(formatoData)
	}
	inicio, err := lerData(m.Inicio)
	if err != nil {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("data de início inválida: %q", m.Inicio)})
		return
	}
	if m.Fim != "" {
		fim, err := lerData(m.Fim)
		if err != nil || fim.Before(inicio) {
			a.responderErro(w, erroDeEntrada{errors.New("a data final precisa ser depois do início")})
			return
		}
	}
	if m.Minutos == 0 && m.Fim == "" {
		a.responderErro(w, erroDeEntrada{errors.New("defina os minutos por dia ou a data final")})
		return
	}
	res, err := a.db.ExecContext(r.Context(), `UPDATE trilhas SET meta_dias = ?, meta_minutos = ?, meta_inicio = ?, meta_fim = ?,
		estado = 'ativa', atualizado_em = ? WHERE id = ?`, escreverDias(m.Dias), m.Minutos, m.Inicio, m.Fim, agora(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		a.responderErro(w, errNaoEncontrou)
		return
	}
	desde := inicio
	if h := hoje(); desde.Before(h) {
		desde = h
	}
	if err := a.gerarCronograma(r.Context(), id, desde); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleTrilha(w, r)
}

func (a *app) handleRecalcular(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if err := a.gerarCronograma(r.Context(), id, hoje()); err != nil {
		a.responderErro(w, err)
		return
	}
	a.handleCronograma(w, r)
}

type itemDia struct {
	EtapaID int64  `json:"etapa_id"`
	Titulo  string `json:"titulo"`
	Status  string `json:"status"`
	Minutos int    `json:"minutos"` // deste dia
	Total   int    `json:"total"`   // da etapa inteira
	Parte   string `json:"parte"`   // "1/2" quando a etapa ocupa mais de um dia
}

type dia struct {
	Data  string    `json:"data"`
	Itens []itemDia `json:"itens"`
}

func (a *app) cronogramaDe(ctx context.Context, trilhaID int64) ([]dia, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT c.data, e.id, e.titulo, e.status, c.minutos, e.minutos,
		(SELECT COUNT(*) FROM cronograma x WHERE x.etapa_id = c.etapa_id AND x.data <= c.data),
		(SELECT COUNT(*) FROM cronograma x WHERE x.etapa_id = c.etapa_id)
		FROM cronograma c JOIN etapas e ON e.id = c.etapa_id JOIN modulos m ON m.id = e.modulo_id
		WHERE c.trilha_id = ? ORDER BY c.data, m.ordem, e.ordem`, trilhaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dias := []dia{}
	for rows.Next() {
		var data string
		var it itemDia
		var n, de int
		if err := rows.Scan(&data, &it.EtapaID, &it.Titulo, &it.Status, &it.Minutos, &it.Total, &n, &de); err != nil {
			return nil, err
		}
		if de > 1 {
			it.Parte = fmt.Sprintf("%d/%d", n, de)
		}
		if len(dias) == 0 || dias[len(dias)-1].Data != data {
			dias = append(dias, dia{Data: data, Itens: []itemDia{}})
		}
		dias[len(dias)-1].Itens = append(dias[len(dias)-1].Itens, it)
	}
	return dias, rows.Err()
}

func (a *app) handleCronograma(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	dias, err := a.cronogramaDe(r.Context(), id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	responderJSON(w, http.StatusOK, dias)
}

type trilhaHoje struct {
	ID        int64     `json:"id"`
	Titulo    string    `json:"titulo"`
	Progresso progresso `json:"progresso"`
	Meta      meta      `json:"meta"`
	Hoje      []itemDia `json:"hoje"`
	Atrasadas []itemDia `json:"atrasadas"`
	Proximo   *dia      `json:"proximo"`  // o próximo dia de estudo, se hoje não há nada pendente
	Previsao  string    `json:"previsao"` // último dia do cronograma
}

func (a *app) handleHoje(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h := hoje().Format(formatoData)
	trilhas, err := a.listarTrilhas(ctx)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	resposta := struct {
		Data    string       `json:"data"`
		Trilhas []trilhaHoje `json:"trilhas"`
	}{Data: h, Trilhas: []trilhaHoje{}}

	for _, t := range trilhas {
		if t.Estado != "ativa" {
			continue
		}
		dias, err := a.cronogramaDe(ctx, t.ID)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		th := trilhaHoje{ID: t.ID, Titulo: t.Titulo, Progresso: t.Progresso, Meta: t.Meta, Hoje: []itemDia{}, Atrasadas: []itemDia{}}
		ultimoDia := map[int64]string{}
		for _, d := range dias {
			for _, it := range d.Itens {
				ultimoDia[it.EtapaID] = d.Data
			}
		}
		atrasada := map[int64]bool{}
		pendenteHoje := false
		for _, d := range dias {
			th.Previsao = d.Data
			for _, it := range d.Itens {
				pendente := it.Status != "concluida" && it.Status != "pulada"
				switch {
				case d.Data < h && pendente && ultimoDia[it.EtapaID] < h && !atrasada[it.EtapaID]:
					atrasada[it.EtapaID] = true
					it.Minutos, it.Parte = it.Total, ""
					th.Atrasadas = append(th.Atrasadas, it)
				case d.Data == h:
					th.Hoje = append(th.Hoje, it)
					pendenteHoje = pendenteHoje || pendente
				}
			}
		}
		if !pendenteHoje {
			for _, d := range dias {
				if d.Data <= h {
					continue
				}
				prox := dia{Data: d.Data, Itens: []itemDia{}}
				for _, it := range d.Itens {
					if it.Status != "concluida" && it.Status != "pulada" {
						prox.Itens = append(prox.Itens, it)
					}
				}
				if len(prox.Itens) > 0 {
					th.Proximo = &prox
					break
				}
			}
		}
		resposta.Trilhas = append(resposta.Trilhas, th)
	}
	responderJSON(w, http.StatusOK, resposta)
}
