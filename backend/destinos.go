package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Entrada, fase 4 (T38): para os fragmentos confirmados, a IA sugere as trilhas (a tarefa
// "destino", barata: só títulos e resumos); o Lucas confirma, tira ou acrescenta e manda. Para
// cada trilha escolhida vai um encaixe (T32, reaproveitado) com um material "youtube" cujo
// localizador são os intervalos dos fragmentos dela — a IA diz a etapa de cada um.

const tipoDestino = "destino" // a tarefa da fila que sugere as trilhas

// ---------- os intervalos "mm:ss–mm:ss; mm:ss–mm:ss" do localizador ----------

var tracoIntervalo = regexp.MustCompile(`[–—]`)

// lerIntervalos: "06:00–10:00; 25:10-31:00" → pares início/fim em segundos (traço ou hífen).
func lerIntervalos(localizador string) ([][2]int, error) {
	var intervalos [][2]int
	for _, parte := range strings.Split(localizador, ";") {
		parte = strings.TrimSpace(parte)
		if parte == "" {
			continue
		}
		lados := strings.Split(tracoIntervalo.ReplaceAllString(parte, "-"), "-")
		if len(lados) != 2 {
			return nil, fmt.Errorf("intervalo ilegível: %q", parte)
		}
		inicio, err1 := segundosDoTempo(lados[0])
		fim, err2 := segundosDoTempo(lados[1])
		switch {
		case err1 != nil || err2 != nil:
			return nil, fmt.Errorf("intervalo ilegível: %q", parte)
		case fim <= inicio:
			return nil, fmt.Errorf("o fim não passa do início: %q", parte)
		}
		intervalos = append(intervalos, [2]int{inicio, fim})
	}
	if len(intervalos) == 0 {
		return nil, fmt.Errorf("sem intervalo em %q", localizador)
	}
	return intervalos, nil
}

// escreverIntervalos: o localizador de um material youtube — os intervalos com "; ".
func escreverIntervalos(intervalos [][2]int) string {
	var partes []string
	for _, x := range intervalos {
		partes = append(partes, tempoDos(x[0])+"–"+tempoDos(x[1]))
	}
	return strings.Join(partes, "; ")
}

var marcaDoBloco = regexp.MustCompile(`^\[((?:\d+:)?\d{1,2}:\d{2})\]`)

// trechoDaTranscricao: só as linhas [mm:ss] cujo tempo cai dentro de inicio–fim (fim fora).
func trechoDaTranscricao(transcricao string, inicio, fim int) string {
	var b strings.Builder
	for _, linha := range strings.Split(transcricao, "\n") {
		m := marcaDoBloco.FindStringSubmatch(linha)
		if m == nil {
			continue
		}
		s, err := segundosDoTempo(m[1])
		if err != nil || s < inicio || s >= fim {
			continue
		}
		b.WriteString(strings.TrimSpace(linha) + "\n\n")
	}
	return b.String()
}

// localizadorDentro: o localizador de um encaixe de material youtube é um intervalo só e cai
// dentro de um dos intervalos mandados.
func localizadorDentro(localizador string, mandados [][2]int) bool {
	xs, err := lerIntervalos(localizador)
	if err != nil || len(xs) != 1 {
		return false
	}
	for _, m := range mandados {
		if xs[0][0] >= m[0] && xs[0][1] <= m[1] {
			return true
		}
	}
	return false
}

// trechosDoVideo: o texto de um material youtube para a pasta da tarefa — um bloco
// "## início–fim · título do fragmento" por intervalo, com o resumo e as falas dele.
func (a *app) trechosDoVideo(ctx context.Context, referencia, localizador string) (string, error) {
	var fonteID int64
	err := a.db.QueryRowContext(ctx, `SELECT id FROM fontes WHERE url = ? AND tipo IN ('youtube', 'acervo')`, referencia).Scan(&fonteID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("o vídeo não está na Entrada")
	}
	if err != nil {
		return "", err
	}
	intervalos, err := lerIntervalos(localizador)
	if err != nil {
		return "", err
	}
	transcricao, _ := a.conteudo.ler(pastaDaFonte(fonteID) + "/transcricao.md")
	if strings.TrimSpace(transcricao) == "" {
		return "", errors.New("o vídeo não tem transcrição")
	}
	fragmentos, err := a.fragmentosDaFonte(ctx, fonteID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, iv := range intervalos {
		var fr *fragmento
		for i := range fragmentos {
			if fragmentos[i].Inicio == iv[0] && fragmentos[i].Fim == iv[1] {
				fr = &fragmentos[i]
			}
		}
		cabecalho := tempoDos(iv[0]) + "–" + tempoDos(iv[1])
		if fr != nil {
			cabecalho += " · " + fr.Titulo
		}
		fmt.Fprintf(&b, "## %s\n\n", cabecalho)
		if fr != nil && fr.Resumo != "" {
			fmt.Fprintf(&b, "%s\n\n", fr.Resumo)
		}
		b.WriteString(trechoDaTranscricao(transcricao, iv[0], iv[1]))
	}
	texto, cortado := cortar(b.String(), limiteTextoMaterial)
	if cortado {
		texto += "\n\n> (Cortado aqui: os trechos passam do limite.)\n"
	}
	return texto, nil
}

// ---------- a tarefa: a IA sugere as trilhas ----------

const esquemaDestino = `{"type":"object","properties":{
"destinos":{"type":"array","items":{"type":"object","properties":{
  "fragmento":{"type":"integer","description":"o número do fragmento no pedido"},
  "trilhas":{"type":"array","items":{"type":"object","properties":{
     "trilha_id":{"type":"integer"},"motivo":{"type":"string"}},"required":["trilha_id","motivo"]}},
  "trilha_nova":{"type":"string","description":"nome de uma trilha nova, se nenhuma serve e o assunto merece; senão vazio"},
  "motivo_nenhuma":{"type":"string","description":"se não vai a lugar nenhum (opinião, fora do tema), por quê; senão vazio"}},
  "required":["fragmento","trilhas","trilha_nova","motivo_nenhuma"]}},
"notas":{"type":"string"}},
"required":["destinos","notas"]}`

// trilhaDestino: o resumo de uma trilha que vai no pedido (sem as etapas inteiras).
type trilhaDestino struct {
	ID       int64
	Titulo   string
	Objetivo string
	Modulos  []struct {
		Titulo string
		Etapas []string
	}
}

// limiteEtapasDestino: só uma proteção para trilha gigante; as de hoje têm até ~100 etapas.
const limiteEtapasDestino = 300

// trilhasParaDestino: todas as trilhas não concluídas, com os títulos dos módulos e das etapas
// (todas: só os títulos, uma linha por módulo, cabem bem; com corte, a IA deixava de ver tópicos
// que existiam, como a VPN no fim da trilha de redes, e sugeria trilha nova).
func (a *app) trilhasParaDestino(ctx context.Context) ([]trilhaDestino, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, titulo, objetivo FROM trilhas WHERE estado != 'concluida' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trilhas []trilhaDestino
	for rows.Next() {
		var t trilhaDestino
		if err := rows.Scan(&t.ID, &t.Titulo, &t.Objetivo); err != nil {
			return nil, err
		}
		trilhas = append(trilhas, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range trilhas {
		md, err := a.db.QueryContext(ctx, `SELECT m.titulo, COALESCE(e.titulo, '') FROM modulos m
			LEFT JOIN etapas e ON e.modulo_id = m.id WHERE m.trilha_id = ? ORDER BY m.ordem, e.ordem`, trilhas[i].ID)
		if err != nil {
			return nil, err
		}
		etapas := 0
		for md.Next() {
			var mt, et string
			if err := md.Scan(&mt, &et); err != nil {
				md.Close()
				return nil, err
			}
			mods := &trilhas[i].Modulos
			if len(*mods) == 0 || (*mods)[len(*mods)-1].Titulo != mt {
				*mods = append(*mods, struct {
					Titulo string
					Etapas []string
				}{Titulo: mt})
			}
			if et != "" {
				etapas++
				if etapas <= limiteEtapasDestino {
					(*mods)[len(*mods)-1].Etapas = append((*mods)[len(*mods)-1].Etapas, et)
				} else if etapas == limiteEtapasDestino+1 {
					(*mods)[len(*mods)-1].Etapas = append((*mods)[len(*mods)-1].Etapas, "…")
				}
			}
		}
		if err := md.Err(); err != nil {
			md.Close()
			return nil, err
		}
		md.Close()
	}
	return trilhas, nil
}

// pedidoDaDestino: o pedido.md com os fragmentos e as trilhas (a IA não lê a transcrição).
func pedidoDaDestino(f fonte, fragmentos []fragmento, trilhas []trilhaDestino) map[string]string {
	var s strings.Builder
	s.WriteString("# Pedido: destino\n\n")
	fmt.Fprintf(&s, "O vídeo %q (%s) está dividido nos fragmentos abaixo. Para cada um diga a que trilha(s) ele vai — pode ser mais de uma, uma trilha nova, ou nenhuma.\n", f.Titulo, f.URL)
	s.WriteString("\n## Fragmentos\n\n")
	for i, fr := range fragmentos {
		fmt.Fprintf(&s, "%d. %s–%s — %q (%s)\n", i+1, tempoDos(fr.Inicio), tempoDos(fr.Fim), fr.Titulo, fr.Tipo)
		if fr.Resumo != "" {
			fmt.Fprintf(&s, "   Resumo: %s\n", fr.Resumo)
		}
		if len(fr.Assuntos) > 0 {
			fmt.Fprintf(&s, "   Assuntos: %s\n", strings.Join(fr.Assuntos, ", "))
		}
	}
	s.WriteString("\n## Trilhas\n\n")
	for _, tr := range trilhas {
		fmt.Fprintf(&s, "- Trilha %d — %q\n", tr.ID, tr.Titulo)
		if tr.Objetivo != "" {
			fmt.Fprintf(&s, "  Objetivo: %s\n", tr.Objetivo)
		}
		for _, md := range tr.Modulos {
			fmt.Fprintf(&s, "  Módulo %q: %s\n", md.Titulo, strings.Join(md.Etapas, " · "))
		}
	}
	if p := strings.TrimSpace(f.PedidoDestino); p != "" {
		fmt.Fprintf(&s, "\n## Pedido do Lucas\n\n%s\n", p)
	}
	return map[string]string{"pedido.md": s.String()}
}

func (a *app) fonteDaTarefaDestino(ctx context.Context, tarefaID int64) (fonte, error) {
	f, err := lerFonte(a.db.QueryRowContext(ctx, `SELECT `+colunasFonte+` WHERE tarefa_destino = ?`, tarefaID))
	if errors.Is(err, sql.ErrNoRows) {
		return f, errNaoEncontrou
	}
	return f, err
}

// sugerirDestinos é a tarefa: a IA sugere as trilhas dos fragmentos confirmados. A sugestão vai
// em fragmentos.sugestao e já fica marcada em fragmento_destinos (o Lucas tira ou acrescenta).
func (a *app) sugerirDestinos(ctx context.Context, t *tarefa) error {
	f, err := a.fonteDaTarefaDestino(ctx, t.ID)
	if errors.Is(err, errNaoEncontrou) {
		return errors.New("a fonte desta tarefa não existe mais")
	}
	if err != nil {
		return err
	}
	todos, err := a.fragmentosDaFonte(ctx, f.ID)
	if err != nil {
		return err
	}
	var fragmentos []fragmento
	for _, fr := range todos {
		if fr.Estado == "confirmado" {
			fragmentos = append(fragmentos, fr)
		}
	}
	if len(fragmentos) == 0 {
		return errors.New("a fonte não tem fragmentos confirmados")
	}
	trilhas, err := a.trilhasParaDestino(ctx)
	if err != nil {
		return err
	}
	pasta := filepath.Join(a.ia.tarefas, strconv.FormatInt(t.ID, 10))
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(pasta)
	if err := gravarNaPasta(pasta, pedidoDaDestino(f, fragmentos, trilhas)); err != nil {
		return err
	}
	rg, err := a.ia.lerRegra("destino.md")
	if err != nil {
		return err
	}
	resp, err := a.ia.chamar(ctx, pedidoClaude{
		pasta: pasta, regra: rg.Texto, ferramentas: "Read", esquema: esquemaDestino, modelo: t.Modelo,
		instrucao: "Diga a que trilha(s) vai cada fragmento do vídeo. Os fragmentos e as trilhas estão em `pedido.md`, nesta pasta: leia-o inteiro antes.",
	})
	if err != nil {
		return err
	}
	var r struct {
		Destinos []struct {
			Fragmento int `json:"fragmento"`
			Trilhas   []struct {
				TrilhaID int64  `json:"trilha_id"`
				Motivo   string `json:"motivo"`
			} `json:"trilhas"`
			TrilhaNova    string `json:"trilha_nova"`
			MotivoNenhuma string `json:"motivo_nenhuma"`
		} `json:"destinos"`
		Notas string `json:"notas"`
	}
	if err := dadosDaResposta(resp, &r); err != nil {
		return err
	}
	porID := map[int64]trilhaDestino{}
	for _, tr := range trilhas {
		porID[tr.ID] = tr
	}
	var tirados []string
	sugestoes := map[int64]sugestaoDestino{}
	for _, d := range r.Destinos {
		if d.Fragmento < 1 || d.Fragmento > len(fragmentos) {
			tirados = append(tirados, fmt.Sprintf("o fragmento %d (não existe)", d.Fragmento))
			continue
		}
		fr := fragmentos[d.Fragmento-1]
		sg := sugestaoDestino{TrilhaNova: textoDaIA(d.TrilhaNova), MotivoNenhuma: textoDaIA(d.MotivoNenhuma)}
		for _, tr := range d.Trilhas {
			if t, ok := porID[tr.TrilhaID]; ok {
				sg.Trilhas = append(sg.Trilhas, sugestaoTrilha{TrilhaID: tr.TrilhaID, Titulo: t.Titulo, Motivo: textoDaIA(tr.Motivo)})
			} else {
				tirados = append(tirados, fmt.Sprintf("a trilha %d (não existe)", tr.TrilhaID))
			}
		}
		sugestoes[fr.ID] = sg
	}
	nota := textoDaIA(r.Notas)
	if len(tirados) > 0 {
		nota = strings.TrimSpace(nota + "\n\n(O app tirou: " + strings.Join(tirados, "; ") + ".)")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, fr := range fragmentos {
		sg, ok := sugestoes[fr.ID]
		if !ok {
			continue
		}
		bruto, _ := json.Marshal(sg)
		if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET sugestao = ?, atualizado_em = ? WHERE id = ?`, string(bruto), agora(), fr.ID); err != nil {
			return err
		}
		for _, tr := range sg.Trilhas {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO fragmento_destinos (fragmento_id, trilha_id, criado_em) VALUES (?, ?, ?)`,
				fr.ID, tr.TrilhaID, agora()); err != nil {
				return err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET notas = ? WHERE id = ?`, nota, f.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// destinosDaFonte: as trilhas marcadas de cada fragmento (e o estado do encaixe, quando mandado).
func (a *app) destinosDaFonte(ctx context.Context, fonteID int64) (map[int64][]destino, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT d.fragmento_id, d.trilha_id, t.titulo,
		COALESCE(d.proposta_id, 0), COALESCE(p.estado, '')
		FROM fragmento_destinos d
		JOIN fragmentos f ON f.id = d.fragmento_id
		JOIN trilhas t ON t.id = d.trilha_id
		LEFT JOIN propostas p ON p.id = d.proposta_id
		WHERE f.fonte_id = ? ORDER BY d.trilha_id`, fonteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	destinos := map[int64][]destino{}
	for rows.Next() {
		var d destino
		var fragmentoID int64
		if err := rows.Scan(&fragmentoID, &d.TrilhaID, &d.Titulo, &d.PropostaID, &d.Estado); err != nil {
			return nil, err
		}
		destinos[fragmentoID] = append(destinos[fragmentoID], d)
	}
	return destinos, rows.Err()
}

// ---------- handlers ----------

// handleEntradaDestino põe a sugestão de trilhas na fila (pedido opcional), para os fragmentos
// confirmados — as opiniões vão também: a IA costuma dizer "nenhuma".
func (a *app) handleEntradaDestino(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pedido string `json:"pedido"`
	}
	// O pedido é opcional: um corpo vazio vale.
	if err := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)).Decode(&corpo); err != nil && !errors.Is(err, io.EOF) {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("corpo inválido: %w", err)})
		return
	}
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if f.Estado != "fragmentada" {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("só dá para sugerir trilhas de fonte fragmentada (esta está %s)", f.Estado)})
		return
	}
	var confirmados int
	if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fragmentos WHERE fonte_id = ? AND estado = 'confirmado'`, id).Scan(&confirmados); err != nil {
		a.responderErro(w, err)
		return
	}
	if confirmados == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("confirme os fragmentos antes de pedir as trilhas")})
		return
	}
	if f.TarefaDestinoID > 0 {
		var estado string
		if err := a.db.QueryRowContext(ctx, `SELECT estado FROM tarefas WHERE id = ?`, f.TarefaDestinoID).Scan(&estado); err == nil && (estado == naFila || estado == rodando) {
			a.responderErro(w, erroDeEntrada{errors.New("a sugestão de trilhas já está na fila ou rodando")})
			return
		}
	}
	modelo := a.modeloPara(ctx, tipoDestino)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	tarefaID, err := a.fila.enfileirar(ctx, tx, tipoDestino, 0, "Destino: "+primeiraLinha(f.Titulo, f.URL), modelo)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if _, err := tx.ExecContext(ctx, `UPDATE fontes SET tarefa_destino = ?, pedido_destino = ? WHERE id = ?`,
		tarefaID, strings.TrimSpace(corpo.Pedido), id); err != nil {
		a.responderErro(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	a.fila.acordar()
	f, _ = a.entradaCompleta(ctx, id)
	responderJSON(w, http.StatusAccepted, f)
}

// handleFragmentoDestinos: {"trilhas":[8,12]} ou {"sem_destino":true} — o que o Lucas escolheu,
// substituindo o que ainda não foi enviado.
func (a *app) handleFragmentoDestinos(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	var corpo struct {
		Trilhas    []int64 `json:"trilhas"`
		SemDestino bool    `json:"sem_destino"`
	}
	if err := lerCorpo(r, &corpo); err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	fr, err := a.fragmentoPorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	if fr.Estado != "confirmado" && fr.Estado != "sem_destino" {
		a.responderErro(w, erroDeEntrada{fmt.Errorf("só dá para escolher o destino de fragmento confirmado ou sem destino (este está %s)", fr.Estado)})
		return
	}
	if !corpo.SemDestino {
		for _, trilhaID := range corpo.Trilhas {
			var existe int
			if err := a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trilhas WHERE id = ?`, trilhaID).Scan(&existe); err != nil {
				a.responderErro(w, err)
				return
			}
			if existe == 0 {
				a.responderErro(w, erroDeEntrada{fmt.Errorf("a trilha %d não existe", trilhaID)})
				return
			}
		}
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM fragmento_destinos WHERE fragmento_id = ? AND proposta_id IS NULL`, id); err != nil {
		a.responderErro(w, err)
		return
	}
	if corpo.SemDestino {
		if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET estado = 'sem_destino', sem_destino_em = ?, atualizado_em = ? WHERE id = ?`,
			agora(), agora(), id); err != nil {
			a.responderErro(w, err)
			return
		}
	} else {
		for _, trilhaID := range corpo.Trilhas {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO fragmento_destinos (fragmento_id, trilha_id, criado_em) VALUES (?, ?, ?)`,
				id, trilhaID, agora()); err != nil {
				a.responderErro(w, err)
				return
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET estado = 'confirmado', sem_destino_em = '', atualizado_em = ? WHERE id = ?`,
			agora(), id); err != nil {
			a.responderErro(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		a.responderErro(w, err)
		return
	}
	fr, _ = a.fragmentoPorID(ctx, id)
	destinos, err := a.destinosDaFonte(ctx, fr.FonteID)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	fr.Destinos = destinos[fr.ID]
	responderJSON(w, http.StatusOK, fr)
}

// encaixeEnviado: um item da resposta do "Mandar para as trilhas".
type encaixeEnviado struct {
	TrilhaID   int64  `json:"trilha_id"`
	Titulo     string `json:"titulo"`
	PropostaID int64  `json:"proposta_id"`
}

// handleEntradaEnviar: para cada trilha marcada nos fragmentos da fonte, um encaixe (a função
// de handleEncaixar reaproveitada) com os intervalos dos fragmentos dela; o fragmento vira
// "enviado" e o Lucas aprova o encaixe na trilha, como sempre.
func (a *app) handleEntradaEnviar(w http.ResponseWriter, r *http.Request) {
	id, err := idDaRota(r)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	ctx := r.Context()
	f, err := a.fontePorID(ctx, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	rows, err := a.db.QueryContext(ctx, `SELECT fr.id, fr.inicio, fr.fim, d.trilha_id, t.titulo
		FROM fragmento_destinos d
		JOIN fragmentos fr ON fr.id = d.fragmento_id
		JOIN trilhas t ON t.id = d.trilha_id
		WHERE fr.fonte_id = ? AND fr.estado = 'confirmado' AND d.proposta_id IS NULL
		ORDER BY d.trilha_id, fr.inicio`, id)
	if err != nil {
		a.responderErro(w, err)
		return
	}
	defer rows.Close()
	type grupo struct {
		titulo     string
		fragmentos []int64
		intervalos [][2]int
	}
	porTrilha := map[int64]*grupo{}
	var ordem []int64
	for rows.Next() {
		var fragmentoID, trilhaID, inicio, fim int64
		var titulo string
		if err := rows.Scan(&fragmentoID, &inicio, &fim, &trilhaID, &titulo); err != nil {
			a.responderErro(w, err)
			return
		}
		g, ok := porTrilha[trilhaID]
		if !ok {
			g = &grupo{titulo: titulo}
			porTrilha[trilhaID] = g
			ordem = append(ordem, trilhaID)
		}
		g.fragmentos = append(g.fragmentos, fragmentoID)
		g.intervalos = append(g.intervalos, [2]int{int(inicio), int(fim)})
	}
	if err := rows.Err(); err != nil {
		a.responderErro(w, err)
		return
	}
	rows.Close()
	if len(ordem) == 0 {
		a.responderErro(w, erroDeEntrada{errors.New("marque as trilhas dos fragmentos antes de mandar")})
		return
	}
	titulo := f.Titulo
	if f.Canal != "" {
		titulo += " (" + f.Canal + ")"
	}
	var enviados []encaixeEnviado
	for _, trilhaID := range ordem {
		g := porTrilha[trilhaID]
		tr, err := a.trilhaCompleta(ctx, trilhaID)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		m := material{
			Tipo: "youtube", Titulo: titulo, Referencia: f.URL,
			Localizador: escreverIntervalos(g.intervalos),
		}
		pedido := fmt.Sprintf("São %d fragmentos deste vídeo; encaixe cada um pelo seu intervalo.", len(g.fragmentos))
		if len(g.fragmentos) == 1 {
			pedido = "É 1 fragmento deste vídeo; encaixe pelo seu intervalo."
		}
		p, err := a.criarEncaixe(ctx, tr, m, pedido, "")
		if err != nil {
			a.responderErro(w, err)
			return
		}
		tx, err := a.db.BeginTx(ctx, nil)
		if err != nil {
			a.responderErro(w, err)
			return
		}
		for _, fragmentoID := range g.fragmentos {
			if _, err := tx.ExecContext(ctx, `UPDATE fragmento_destinos SET proposta_id = ? WHERE fragmento_id = ? AND trilha_id = ?`,
				p.ID, fragmentoID, trilhaID); err != nil {
				tx.Rollback()
				a.responderErro(w, err)
				return
			}
		}
		// O fragmento vai a "enviado" quando tem destino e nenhum espera mais encaixe (o que
		// ficou confirmado sem trilha marcada continua confirmado).
		if _, err := tx.ExecContext(ctx, `UPDATE fragmentos SET estado = 'enviado', atualizado_em = ? WHERE estado = 'confirmado'
			AND EXISTS (SELECT 1 FROM fragmento_destinos d WHERE d.fragmento_id = fragmentos.id)
			AND NOT EXISTS (SELECT 1 FROM fragmento_destinos d WHERE d.fragmento_id = fragmentos.id AND d.proposta_id IS NULL)`,
			agora()); err != nil {
			tx.Rollback()
			a.responderErro(w, err)
			return
		}
		if err := tx.Commit(); err != nil {
			a.responderErro(w, err)
			return
		}
		enviados = append(enviados, encaixeEnviado{TrilhaID: trilhaID, Titulo: g.titulo, PropostaID: p.ID})
	}
	responderJSON(w, http.StatusOK, vazioSeNulo(enviados))
}
