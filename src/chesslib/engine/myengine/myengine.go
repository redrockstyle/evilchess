package myengine

import (
	"context"
	"encoding/binary"
	"evilchess/src/chesslib/base"
	"evilchess/src/chesslib/engine"
	"evilchess/src/chesslib/logic/convert/convfen"
	"evilchess/src/chesslib/logic/rules"
	"evilchess/src/chesslib/logic/rules/moves"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MATE_SCORE     = 1000000
	MATE_THRESHOLD = 900000
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

	// simple transposition table
	tt   *transTable
	ttMu sync.RWMutex

	// last best root move per depth (for move ordering between ID iterations)
	lastRootMove *base.Move

	// log (preserve)
	// logx logx.Logger
}

func NewEvilEngine() *EvilEngine {
	return &EvilEngine{
		board: nil,
		subs:  make(map[int]chan<- engine.AnalysisInfo),
		tt:    newTransTable(1 << 20), // ~1M entries (adjust)
	}
}

func (e *EvilEngine) Init() error {
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
	atomic.StoreInt64(&e.nodes, 0)
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
	// clear TT (naive)
	e.tt.clear()

	e.mu.Unlock()

	e.wg.Add(1)
	go e.searchWorker(e.ctx, params)
	return nil
}

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

func (e *EvilEngine) BestNow() engine.AnalysisInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastInfo
}

func (e *EvilEngine) WaitDone() {
	e.wg.Wait()
}

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

func (e *EvilEngine) Close() {
	_ = e.StopAnalysis()
	e.wg.Wait()
	return
}

// --- internal helpers ---

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
		}
	}
	e.subsMu.Unlock()
}

// --------------------------------------------------
// Simple Transposition Table (thread-safe map-based)
// --------------------------------------------------
type ttEntry struct {
	key   uint64
	depth int32
	score int32
	flag  uint8 // 0 exact, 1 lower, 2 upper
	move  base.Move
}

type transTable struct {
	mu    sync.RWMutex
	table map[uint64]ttEntry
	limit int
}

func newTransTable(limit int) *transTable {
	return &transTable{
		table: make(map[uint64]ttEntry, limit),
		limit: limit,
	}
}

func (t *transTable) probe(key uint64) (ttEntry, bool) {
	t.mu.RLock()
	e, ok := t.table[key]
	t.mu.RUnlock()
	return e, ok
}

func (t *transTable) store(key uint64, depth int, flag uint8, score int, mv base.Move) {
	t.mu.Lock()
	// simple replacement: keep deeper or insert
	if old, ok := t.table[key]; ok {
		if int(old.depth) > depth {
			// keep old
			t.mu.Unlock()
			return
		}
	}
	t.table[key] = ttEntry{
		key:   key,
		depth: int32(depth),
		score: int32(score),
		flag:  flag,
		move:  mv,
	}
	// optional: keep map size bounded
	if len(t.table) > t.limit*2 {
		// naive shrink: reallocate (simple)
		newM := make(map[uint64]ttEntry, t.limit)
		i := 0
		for k, v := range t.table {
			if i >= t.limit {
				break
			}
			newM[k] = v
			i++
		}
		t.table = newM
	}
	t.mu.Unlock()
}

func (t *transTable) clear() {
	t.mu.Lock()
	t.table = make(map[uint64]ttEntry, t.limit)
	t.mu.Unlock()
}

// --------------------------------------------------
// Hashing helper (FNV-1a over mailbox + side to move) — cheap & deterministic
// --------------------------------------------------
func hashBoard(b *base.Board) uint64 {
	h := fnv.New64a()
	// mailbox is 64 ints; convert each to 8 bytes for deterministic hashing
	for i := 0; i < 64; i++ {
		pi := int64(b.Mailbox[i])
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(pi))
		_, _ = h.Write(buf[:])
	}
	// side to move
	if b.WhiteToMove {
		_, _ = h.Write([]byte{1})
	} else {
		_, _ = h.Write([]byte{0})
	}
	// Note: castling/enpassant not included (if you have these fields, include them)
	return h.Sum64()
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

// -------------------------------
// Quiescence (captures only)
// -------------------------------
func (e *EvilEngine) quiesce(b *base.Board, alpha, beta int, ctx context.Context, nodes *int64) int {
	// cancellation check
	select {
	case <-ctx.Done():
		return 0
	default:
	}
	atomic.AddInt64(&e.nodes, 1)
	*nodes++
	stand := evaluateMaterial(b)
	if b.WhiteToMove {
		// positive = good for side to move
	} else {
		stand = -stand
	}
	if stand >= beta {
		return beta
	}
	if alpha < stand {
		alpha = stand
	}
	// generate captures only
	caps := moves.GenerateLegalMoves(b)
	// trivial ordering: none (you may implement MVV-LVA if move object stores captured piece)
	for _, mv := range caps {
		select {
		case <-ctx.Done():
			return 0
		default:
		}

		if rules.IsCaptureMove(mv, b) {
			continue
		}

		nb := moves.CloneBoard(b)
		_ = moves.ApplyMove(nb, mv)
		score := -e.quiesce(nb, -beta, -alpha, ctx, nodes)
		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}
	return alpha
}

// -------------------------------
// Minimax with alpha-beta + TT + PV extraction support
// -------------------------------
// returns score (centipawns) from side-to-move POV
func (e *EvilEngine) minimax(b *base.Board, depth int, alpha, beta int, ctx context.Context, nodes *int64, ply int) int {
	// cancellation check
	select {
	case <-ctx.Done():
		return 0
	default:
	}
	atomic.AddInt64(&e.nodes, 1)
	*nodes++

	// probe TT
	key := hashBoard(b)
	if entry, ok := e.tt.probe(key); ok && int(entry.depth) >= depth {
		// use stored score according to flag
		if entry.flag == 0 { // exact
			return int(entry.score)
		}
		if entry.flag == 1 { // lower bound
			if int(entry.score) > alpha {
				alpha = int(entry.score)
			}
		} else if entry.flag == 2 { // upper bound
			if int(entry.score) < beta {
				beta = int(entry.score)
			}
		}
		if alpha >= beta {
			return int(entry.score)
		}
	}

	if depth == 0 {
		// quiescence search instead of raw eval
		return e.quiesce(b, alpha, beta, ctx, nodes)
	}

	mvs := moves.GenerateLegalMoves(b)
	// reorder moves: captures first (MVV-LVA-like)
	sort.SliceStable(mvs, func(i, j int) bool {
		si := moveOrderScore(b, mvs[i])
		sj := moveOrderScore(b, mvs[j])
		return si > sj
	})
	if len(mvs) == 0 {
		// terminal: checkmate or stalemate
		status := rules.GameStatusOf(b)
		if status == base.Checkmate {
			return -MATE_SCORE + ply
		}
		return 0
	}

	// ordering: if TT has a move for this key, try it first
	if entry, ok := e.tt.probe(key); ok {
		if entry.move != (base.Move{}) {
			// move entry.move to front if present in mvs
			for i, mv := range mvs {
				if mv == entry.move {
					if i != 0 {
						mvs[0], mvs[i] = mvs[i], mvs[0]
					}
					break
				}
			}
		}
	}
	// also, if we have a last root move (from previous depth) and this is root call depth==initial? we cannot detect root here,
	// but we reorder root separately in searchWorker.

	alphaOrig := alpha
	best := -1_000_000_000
	var bestMove base.Move

	for _, mv := range mvs {
		select {
		case <-ctx.Done():
			return 0
		default:
		}
		nb := moves.CloneBoard(b)
		_ = moves.ApplyMove(nb, mv)
		score := -e.minimax(nb, depth-1, -beta, -alpha, ctx, nodes, ply+1)
		if score > best {
			best = score
			bestMove = mv
		}
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			// beta cutoff
			break
		}
	}

	// store in TT: determine flag
	var flag uint8 = 0 // exact
	if best <= alphaOrig {
		flag = 2 // upper
	} else if best >= beta {
		flag = 1 // lower
	} else {
		flag = 0 // exact
	}
	e.tt.store(key, depth, flag, best, bestMove)

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

// -------------------------------
// helper: extract PV from TT by following stored moves (root-only)
// -------------------------------
func (e *EvilEngine) extractPV(root *base.Board, maxPly int) []base.Move {
	var pv []base.Move
	b := moves.CloneBoard(root)
	for ply := 0; ply < maxPly; ply++ {
		key := hashBoard(b)
		entry, ok := e.tt.probe(key)
		if !ok || entry.move == (base.Move{}) {
			break
		}
		// sanity: check move is legal in current pos
		legal := false
		mvs := moves.GenerateLegalMoves(b)
		for _, mv := range mvs {
			if mv == entry.move {
				legal = true
				break
			}
		}
		if !legal {
			break
		}
		pv = append(pv, entry.move)
		_ = moves.ApplyMove(b, entry.move)
	}
	return pv
}

// -------------------------------
// searchWorker — iterative deepening + improvements
// -------------------------------
func (e *EvilEngine) searchWorker(ctx context.Context, params engine.SearchParams) {
	defer e.wg.Done()
	start := time.Now()
	maxDepth := params.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 6 // default
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
				NPS:      computeNPS(totalNodes, time.Since(start)),
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

		// snapshot root position
		e.mu.RLock()
		pos := moves.CloneBoard(e.board)
		e.mu.RUnlock()

		// generate root moves
		rootMoves := moves.GenerateLegalMoves(pos)
		if len(rootMoves) == 0 {
			// no legal moves: publish and stop
			e.publish(engine.AnalysisInfo{
				Depth:    depth,
				TimeMs:   time.Since(start).Milliseconds(),
				Nodes:    totalNodes,
				NPS:      computeNPS(totalNodes, time.Since(start)),
				ScoreCP:  0,
				MateIn:   0,
				PV:       nil,
				BestMove: nil,
			})
			break
		}

		// reorder root moves: prefer previous best root move (from last iteration) and TT move
		if e.lastRootMove != nil {
			for i, mv := range rootMoves {
				if mv == *e.lastRootMove {
					if i != 0 {
						rootMoves[0], rootMoves[i] = rootMoves[i], rootMoves[0]
					}
					break
				}
			}
		}
		// if TT has move for root, try to put it first
		if entry, ok := e.tt.probe(hashBoard(pos)); ok && entry.move != (base.Move{}) {
			for i, mv := range rootMoves {
				if mv == entry.move {
					if i != 0 {
						rootMoves[0], rootMoves[i] = rootMoves[i], rootMoves[0]
					}
					break
				}
			}
		}

		localBestScore := -1_000_000_000
		var localBestPV []base.Move
		nodesThisDepth := int64(0)

		// loop root moves
		for i, mv := range rootMoves {
			// respect cancellation periodically
			select {
			case <-ctx.Done():
				e.publish(engine.AnalysisInfo{
					Depth:    depth,
					TimeMs:   time.Since(start).Milliseconds(),
					Nodes:    totalNodes + nodesThisDepth,
					NPS:      computeNPS(totalNodes+nodesThisDepth, time.Since(start)),
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
			score := -e.minimax(nb, depth-1, -1_000_000_000, 1_000_000_000, ctx, &nodes, 1)
			nodesThisDepth += nodes
			// if score >= MATE_THRESHOLD {
			// 	// найден мат для side-to-move — можно завершить перебор корневых ходов ранне
			// 	localBestScore = score
			// 	localBestPV = []base.Move{mv}
			// 	// preserve TT entries; break root loop to publish mate now
			// 	// nodesThisDepth += nodes
			// 	break
			// }

			if score > localBestScore {
				localBestScore = score
				// we will extract PV from TT after search iteration
				localBestPV = []base.Move{mv}
			}

			// aspiration: small window for subsequent root moves may be applied (left simple)
			// optionally, you can reorder moves after measuring scores
			_ = i
		}

		totalNodes += nodesThisDepth

		// update current best
		bestScore = localBestScore

		// use TT to extract a fuller PV (follow TT entries from root)
		bestPV = e.extractPV(pos, depth+4) // depth+4 to capture deeper PV if available
		if len(bestPV) == 0 && len(localBestPV) > 0 {
			bestPV = localBestPV
		}

		// store last root move (if exists)
		if len(bestPV) > 0 {
			e.lastRootMove = &bestPV[0]
		} else {
			e.lastRootMove = nil
		}

		// publish snapshot for this depth
		info := engine.AnalysisInfo{
			Depth:    depth,
			TimeMs:   time.Since(start).Milliseconds(),
			Nodes:    totalNodes,
			NPS:      computeNPS(totalNodes, time.Since(start)),
			ScoreCP:  bestScore,
			PV:       bestPV,
			BestMove: nilIfEmpty(bestPV),
			MateIn:   0,
		}
		// compute MateIn if score indicates mate
		absScore := bestScore
		if absScore < 0 {
			absScore = -absScore
		}
		if absScore >= MATE_THRESHOLD {
			matePly := MATE_SCORE - absScore
			if bestScore > 0 {
				info.MateIn = matePly
			} else {
				// optional: represent mate against us as negative
				info.MateIn = -matePly
			}
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

func pieceValueSimple(p base.Piece) int {
	switch p {
	case base.WPawn, base.BPawn:
		return 100
	case base.WKnight, base.BKnight:
		return 320
	case base.WBishop, base.BBishop:
		return 330
	case base.WRook, base.BRook:
		return 500
	case base.WQueen, base.BQueen:
		return 900
	case base.WKing, base.BKing:
		return 10000
	default:
		return 0
	}
}

// score for ordering move: higher -> try earlier
func moveOrderScore(b *base.Board, mv base.Move) int {
	// if mv captures, score = captured-value*1000 - attacker-value (so MVV-LVA-ish)
	toIdx := base.ConvPointToIndex(mv.To)
	captured := b.Mailbox[toIdx]
	if captured != base.EmptyPiece {
		capVal := pieceValueSimple(captured)
		// attacker value
		fromIdx := base.ConvPointToIndex(mv.From)
		attacker := b.Mailbox[fromIdx]
		attVal := pieceValueSimple(attacker)
		return capVal*1000 - attVal
	}
	// optionally small bonus for promotions if Move contains that info (skip if not)
	return 0
}
