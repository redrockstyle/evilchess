package myengine

import (
	"context"
	"evilchess/src/base"
	"evilchess/src/engine"
	"evilchess/src/logic/convert/convfen"
	"evilchess/src/logic/rules"
	"evilchess/src/logic/rules/moves"
	"fmt"
	"sync"
	"time"
)

// chess engine implementing deep search
type EvilEngine struct {
	mu       sync.RWMutex
	board    *base.Board
	running  bool
	lastInfo engine.AnalysisInfo

	// concurrency primitives
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	subsMu    sync.Mutex
	subs      map[int]chan<- engine.AnalysisInfo
	nextSubID int

	// internal metrics
	nodes int64
}

// NewMinimaxEngine создает экземпляр
func NewEvilEngine() *EvilEngine {
	return &EvilEngine{
		board: nil,
		subs:  make(map[int]chan<- engine.AnalysisInfo),
	}
}

// Init — опционально принимает opts (не используем)
func (e *EvilEngine) Init(opts map[string]interface{}) error {
	// можно настроить threadpool / eval weights и т.д.
	return nil
}

// setup position from FEN
func (e *EvilEngine) SetPositionFEN(fen string) error {
	b, err := convfen.ConvertFENToBoard(fen)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.board = b
	e.mu.Unlock()
	return nil
}

// setup position
func (e *EvilEngine) SetPosition(b *base.Board) error {
	e.mu.Lock()
	e.board = b
	e.mu.Unlock()
	return nil
}

func (e *EvilEngine) LevelToParams(lvl engine.LevelAnalyze) *engine.SearchParams {
	switch lvl {
	case engine.LevelOne:
	case engine.LevelTwo:
		return &engine.SearchParams{
			MaxDepth:  0,
			MaxTimeMs: 2000,
			MaxNodes:  0,
			Infinite:  false,
		}
	case engine.LevelThree:
	case engine.LevelFour:
		return &engine.SearchParams{
			MaxDepth:  0,
			MaxTimeMs: 3000,
			MaxNodes:  0,
			Infinite:  false,
		}
	case engine.LevelFive:
	case engine.LevelSix:
		return &engine.SearchParams{
			MaxDepth:  0,
			MaxTimeMs: 6000,
			MaxNodes:  0,
			Infinite:  false,
		}
	case engine.LevelSeven:
		return &engine.SearchParams{
			MaxDepth:  0,
			MaxTimeMs: 10000,
			MaxNodes:  0,
			Infinite:  false,
		}
	case engine.LevelLast:
	default:
		break
	}
	return &engine.SearchParams{
		MaxDepth:  0,
		MaxTimeMs: 0,
		MaxNodes:  0,
		Infinite:  false,
	}
}

// StartAnalysis запускает поиск асинхронно. Если уже запущен — ошибка.
func (e *EvilEngine) StartAnalysis(params engine.SearchParams) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("analysis already running")
	}
	// require a board
	if e.board == nil {
		e.mu.Unlock()
		return fmt.Errorf("position not set")
	}

	// prepare context and state
	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.running = true
	e.nodes = 0
	// reset lastInfo
	e.lastInfo = engine.AnalysisInfo{
		Depth:   0,
		TimeMs:  0,
		Nodes:   0,
		NPS:     0,
		ScoreCP: 0,
		MateIn:  0,
		PV:      nil,
	}
	e.mu.Unlock()

	e.wg.Add(1)
	go e.searchWorker(e.ctx, params)
	return nil
}

// StopAnalysis — просит остановиться (мягко)
func (e *EvilEngine) StopAnalysis() error {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return fmt.Errorf("not running")
	}
	e.cancel() // signal
	e.mu.Unlock()
	return nil
}

// BestNow — возвращает последний опубликованный снэпшот
func (e *EvilEngine) BestNow() (engine.AnalysisInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.lastInfo.Depth == 0 && len(e.lastInfo.PV) == 0 && !e.running {
		// maybe no data
		return engine.AnalysisInfo{}, fmt.Errorf("no analysis data")
	}
	// return a copy
	info := e.lastInfo
	return info, nil
}

// WaitDone — ждёт завершения и возвращает последний snapshot
func (e *EvilEngine) WaitDone() (engine.AnalysisInfo, error) {
	e.wg.Wait()
	return e.BestNow()
}

// Subscribe — подписка на обновления. Возвращает функцию отписки.
func (e *EvilEngine) Subscribe(ch chan<- engine.AnalysisInfo) (unsubscribe func()) {
	e.subsMu.Lock()
	id := e.nextSubID
	e.nextSubID++
	e.subs[id] = ch
	e.subsMu.Unlock()
	return func() {
		e.subsMu.Lock()
		delete(e.subs, id)
		e.subsMu.Unlock()
	}
}

// Close — освобождает ресурсы, останавливает анализ если нужен
func (e *EvilEngine) Close() error {
	_ = e.StopAnalysis()
	e.wg.Wait()
	return nil
}

// --- internal helpers ---

// publish сообщает всем подписчикам новый snapshot (не блокирует)
func (e *EvilEngine) publish(info engine.AnalysisInfo) {
	// store lastInfo
	e.mu.Lock()
	e.lastInfo = info
	e.mu.Unlock()

	// push to subscribers (non-blocking)
	e.subsMu.Lock()
	for _, ch := range e.subs {
		select {
		case ch <- info:
		default:
			// если канал заполнен — пропускаем (не хотим блокировать движок)
		}
	}
	e.subsMu.Unlock()
}

// simple material evaluation (very naive)
func evaluateMaterial(b *base.Board) int {
	sum := 0
	for i := 0; i < 64; i++ {
		p := b.Mailbox[i]
		switch p {
		case base.WPawn:
			sum += 100
		case base.WKnight:
			sum += 320
		case base.WBishop:
			sum += 330
		case base.WRook:
			sum += 500
		case base.WQueen:
			sum += 900
		case base.WKing:
			sum += 10000
		case base.BPawn:
			sum -= 100
		case base.BKnight:
			sum -= 320
		case base.BBishop:
			sum -= 330
		case base.BRook:
			sum -= 500
		case base.BQueen:
			sum -= 900
		case base.BKing:
			sum -= 10000
		}
	}
	return sum
}

// searchWorker — демонстрационный iterative deepening + minimax (без оптимизаций)
func (e *EvilEngine) searchWorker(ctx context.Context, params engine.SearchParams) {
	defer e.wg.Done()
	start := time.Now()
	maxDepth := params.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 6 // безопасный default для demo
	}
	// iterative deepening
	var bestPV []base.Move
	var bestScore int
	var totalNodes int64 = 0
	for depth := 1; depth <= maxDepth; depth++ {
		// check cancel
		select {
		case <-ctx.Done():
			// publish final and return
			e.publish(engine.AnalysisInfo{
				Depth:    depth - 1,
				TimeMs:   time.Since(start).Milliseconds(),
				Nodes:    totalNodes,
				NPS:      0,
				ScoreCP:  bestScore,
				MateIn:   0,
				PV:       bestPV,
				BestMove: nilIfEmpty(bestPV),
			})
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			return
		default:
		}

		// run a depth-limited search, basic minimax with clone boards
		nodesThisDepth := int64(0)
		// obtain current position snapshot
		e.mu.RLock()
		pos := moves.CloneBoard(e.board)
		e.mu.RUnlock()

		// for each legal move at root, search recursively to depth-1
		rootMoves := moves.GenerateLegalMoves(pos)
		if len(rootMoves) == 0 {
			// no legal moves: publish and stop
			e.publish(engine.AnalysisInfo{
				Depth:    depth,
				TimeMs:   time.Since(start).Milliseconds(),
				Nodes:    totalNodes,
				NPS:      0,
				ScoreCP:  0,
				MateIn:   0,
				PV:       nil,
				BestMove: nil,
			})
			break
		}

		localBestScore := -1_000_000_000
		var localBestPV []base.Move

		for _, mv := range rootMoves {
			// respect cancellation periodically
			select {
			case <-ctx.Done():
				e.publish(engine.AnalysisInfo{
					Depth:    depth,
					TimeMs:   time.Since(start).Milliseconds(),
					Nodes:    totalNodes + nodesThisDepth,
					NPS:      0,
					ScoreCP:  bestScore,
					MateIn:   0,
					PV:       bestPV,
					BestMove: nilIfEmpty(bestPV),
				})
				e.mu.Lock()
				e.running = false
				e.mu.Unlock()
				return
			default:
			}

			// clone and apply move
			nb := moves.CloneBoard(pos)
			_ = moves.ApplyMove(nb, mv)
			nodes := int64(0)
			score := -e.minimax(nb, depth-1, -1_000_000_000, 1_000_000_000, ctx, &nodes)
			nodesThisDepth += nodes

			// build PV: mv + best line returned (minimax doesn't build PV here, we approximate)
			pv := []base.Move{mv}
			// naive: we don't track continuation moves; for demo OK

			if score > localBestScore {
				localBestScore = score
				localBestPV = pv
			}
		}

		totalNodes += nodesThisDepth

		// update current best
		bestScore = localBestScore
		bestPV = localBestPV

		// publish snapshot for this depth
		info := engine.AnalysisInfo{
			Depth:    depth,
			TimeMs:   time.Since(start).Milliseconds(),
			Nodes:    totalNodes,
			NPS:      computeNPS(totalNodes, time.Since(start)),
			ScoreCP:  bestScore,
			MateIn:   0,
			PV:       bestPV,
			BestMove: nilIfEmpty(bestPV),
		}
		e.publish(info)

		// if time limit reached
		if params.MaxTimeMs > 0 && time.Since(start).Milliseconds() >= params.MaxTimeMs {
			break
		}
	}

	// mark stopped
	e.mu.Lock()
	e.running = false
	e.mu.Unlock()
}

// helper: minimax with alpha-beta (returns evaluation in centipawns)
// returns evaluation from side-to-move POV (positive good for side to move)
func (e *EvilEngine) minimax(b *base.Board, depth int, alpha, beta int, ctx context.Context, nodes *int64) int {
	// cancellation check
	select {
	case <-ctx.Done():
		return 0
	default:
	}
	(*nodes)++
	if depth == 0 {
		// evaluate material (positive means advantage for side-to-move)
		val := evaluateMaterial(b)
		if b.WhiteToMove {
			return val
		}
		return -val
	}

	mvs := moves.GenerateLegalMoves(b)
	if len(mvs) == 0 {
		// terminal: checkmate or stalemate
		status := rules.GameStatusOf(b)
		if status == base.Checkmate {
			// very large negative value for side to move
			return -1000000
		}
		return 0
	}

	best := -1_000_000_000
	for _, mv := range mvs {
		nb := moves.CloneBoard(b)
		_ = moves.ApplyMove(nb, mv)
		v := -e.minimax(nb, depth-1, -beta, -alpha, ctx, nodes)
		if v > best {
			best = v
		}
		if v > alpha {
			alpha = v
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

// utility
func computeNPS(nodes int64, dur time.Duration) int64 {
	s := dur.Seconds()
	if s < 1e-6 {
		return nodes
	}
	return int64(float64(nodes) / s)
}

func nilIfEmpty(pv []base.Move) *base.Move {
	if len(pv) == 0 {
		return nil
	}
	return &pv[0]
}
