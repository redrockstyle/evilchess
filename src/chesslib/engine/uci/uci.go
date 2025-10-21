package uci

import (
	"bufio"
	"context"
	"errors"
	"evilchess/src/chesslib/base"
	"evilchess/src/chesslib/engine"
	"evilchess/src/chesslib/logic/convert/convfen"
	"evilchess/src/logx"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UCIExecutor struct {
	// init
	path string
	args []string

	// process
	cmd *exec.Cmd
	in  io.WriteCloser
	out io.ReadCloser

	// read stdout
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// subscribers
	submu sync.Mutex
	subs  map[int]chan<- engine.AnalysisInfo
	subid int

	// runtime
	mu          sync.RWMutex
	running     bool
	info        engine.AnalysisInfo
	timeout     time.Duration
	whiteToMove bool
	lines       chan string
	bestMoveCh  chan struct{}
	logx        logx.Logger

	lastBoard base.Board
}

// to open a process, need to call Init()
func NewUCIExec(logx logx.Logger, enginePath string, engineArgs ...string) *UCIExecutor {
	return &UCIExecutor{
		path: enginePath, args: engineArgs, logx: logx,
		subs: make(map[int]chan<- engine.AnalysisInfo), subid: 0,
		bestMoveCh: make(chan struct{}, 1), // buffered: send won't block if nobody WaitDone yet
	}
}

// open process and check
func (e *UCIExecutor) Init() error {
	if e.path == "" {
		return errors.New("path engine is most be empty")
	}

	cmd := exec.Command(e.path, e.args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error connect to stdin of a process (%d) engine: %v", cmd.Process.Pid, err)
	}

	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error connect to stdout of a process (%d) engine: %v", cmd.Process.Pid, err)
	}
	// if err := cmd.Run(); err != nil {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error open %s engine: %v", e.path, err)
	}

	// process
	e.cmd = cmd
	e.in = in
	e.out = out
	e.lines = make(chan string, 256) // mb need < 256

	// concurrency
	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.wg.Add(1)
	go e.stdoutLoop(e.ctx)

	title, err := e.waitLine(engine.UCIHandshakeTimeout)
	if err != nil {
		return err
	}
	e.logx.Infof("open engine: %s", title)

	if !e.checkUCI() {
		go e.Close()
		return errors.New("error read uciok")
	}
	if !e.checkReady() {
		go e.Close()
		return errors.New("error read readyok")
	}
	return nil
}

// command executable
func (e *UCIExecutor) Exec(cmd string) error {
	if e.in == nil {
		return errors.New("stdin not available")
	}
	_, err := io.WriteString(e.in, cmd+"\n")
	return err
}

func (e *UCIExecutor) SetPosition(b *base.Board) error {
	// store last board for parsing PV -> base.Move
	e.mu.Lock()
	e.lastBoard = *b
	e.mu.Unlock()

	return e.SetPositionFEN(convfen.ConvertBoardToFEN(*b))
}

func (e *UCIExecutor) SetPositionFEN(fen string) error {
	if tu, err := convfen.ConvertFENToBoard(fen); err == nil { // pizdec reshenie XD
		e.mu.Lock()
		e.lastBoard = *tu
		e.whiteToMove = tu.WhiteToMove
		e.mu.Unlock()
	}

	e.logx.Debugf("init postition FEN: %s\n", fen)
	if err := e.Exec("ucinewgame"); err != nil {
		return err
	}
	if err := e.Exec(fmt.Sprintf("position fen %s", fen)); err != nil {
		return err
	}
	if !e.checkReady() {
		return fmt.Errorf("error read readyok")
	}
	return nil
}

// actual info
func (e *UCIExecutor) BestNow() engine.AnalysisInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.info
}

func (e *UCIExecutor) StartAnalysis(prm engine.SearchParams) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd == nil {
		return errors.New("no running uci-process")
	}
	if e.running {
		return errors.New("already running")
	}

	var b strings.Builder
	b.WriteString("go")
	if prm.Infinite {
		b.WriteString(" infinite")
	} else {
		if prm.MaxDepth > 0 {
			b.WriteString(fmt.Sprintf(" depth %v", strconv.Itoa(prm.MaxDepth)))
		}
		if prm.MaxTimeMs > 0 {
			e.timeout = time.Duration(prm.MaxTimeMs) * time.Millisecond
			b.WriteString(fmt.Sprintf(" movetime %v", strconv.FormatInt(prm.MaxTimeMs, 10)))
		} else {
			e.timeout = 0
		}
	}
	e.info = engine.AnalysisInfo{}
	e.running = true
	cmd := b.String()

	e.logx.Infof("start analyze: %s", cmd)
	if err := e.Exec(cmd); err != nil {
		e.running = false
		return err
	}

	return nil
}

// calling if analysis is infinite
func (e *UCIExecutor) StopAnalysis() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd == nil {
		return errors.New("no running uci-process")
	}
	if !e.running {
		return nil
	}

	e.logx.Info("stop analyze")
	return e.Exec("stop")
}

// called if analysis is time-limited
func (e *UCIExecutor) WaitDone() {
	e.mu.RLock()
	running := e.running
	timeout := e.timeout
	ctx := e.ctx
	e.mu.RUnlock()

	e.logx.Infof("wait running engine: %v ms", timeout.Seconds())
	if !running {
		return
	}
	timer := time.NewTimer(e.timeout)
	defer timer.Stop()
	select {
	case <-e.bestMoveCh:
		return
	case <-timer.C:
		return
	case <-ctx.Done():
		return
	}
}

func (e *UCIExecutor) Subscribe(ch chan<- engine.AnalysisInfo) (unsubscribe func()) {
	e.submu.Lock()
	defer e.submu.Unlock()

	id := e.subid
	e.subs[id] = ch
	e.subid++

	return func() {
		e.submu.Lock()
		defer e.submu.Unlock()
		delete(e.subs, id)
	}
}

// Terminate process
func (e *UCIExecutor) Close() {
	if e.cmd == nil {
		return
	}
	e.mu.Lock()
	_ = e.Exec("quit")
	e.cancel()
	e.mu.Unlock()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		if e.cmd != nil && e.cmd.Process != nil {
			_ = e.cmd.Process.Kill()
		}
		e.wg.Wait()
	}

	if e.cmd != nil {
		_ = e.cmd.Wait()
	}
	e.logx.Info("uci-process terminated")

}

func (e *UCIExecutor) checkUCI() bool {
	if err := e.Exec("uci"); err != nil {
		return false
	}
	if err := e.waitCompare("uciok", engine.UCIHandshakeTimeout); err != nil {
		e.logx.Error(err.Error())
		return false
	}
	return true
}

func (e *UCIExecutor) checkReady() bool {
	if err := e.Exec("isready"); err != nil {
		return false
	}
	if err := e.waitCompare("readyok", engine.UCIHandshakeTimeout); err != nil {
		e.logx.Error(err.Error())
		return false
	}
	return true
}

func (e *UCIExecutor) waitLine(timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case line := <-e.lines:
			return line, nil
		case <-timer.C:
			return "", errors.New("timeout waiting")
		case <-e.ctx.Done():
			return "", errors.New("stopped")
		}
	}
}

func (e *UCIExecutor) waitCompare(str string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case line := <-e.lines:
			if strings.HasPrefix(line, str) {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timeout waiting for %s", str)
		case <-e.ctx.Done():
			return errors.New("stopped")
		}
	}
}

func (e *UCIExecutor) stdoutLoop(ctx context.Context) {
	defer e.wg.Done()
	scr := bufio.NewScanner(e.out)
	for scr.Scan() {
		line := scr.Text()
		line = strings.TrimSpace(line) // drop \n\t

		e.logx.Debugf("ENGINE: %s", line)
		select {
		case e.lines <- line:
		default:
			e.logx.Debugf("drop engine line (buffer full)")
		}
		if line != "" {
			if strings.HasPrefix(line, "info ") {
				// info line
				e.saveInfo(line)
			} else if strings.HasPrefix(line, "bestmove ") {
				// bestmove line
				e.saveBest(line)
			}
		}
		// check goroutine
		select {
		case <-ctx.Done():
			// exit goroutine
			return
		default:
		}
	}
}

// type AnalysisInfo struct {
// 	Depth       int         // текущая глубина
// 	TimeMs      int64       // прошедшее время в ms
// 	Nodes       int64       // число просмотренных узлов
// 	NPS         int64       // nodes per second
// 	ScoreCP     int         // оценка в сантиматах (+ = advantage for side to move)
// 	MateIn      int         // mate in N plies (0 if none)
// 	PV          []base.Move // principal variation (список ходов, начиная с best)
// 	BestMove    *base.Move  // лучший ход на данный момент (ссылка на первый ход PV)
// 	UCIPV       []string
// 	UCIBestMove string
// }

func (e *UCIExecutor) saveInfo(info string) {
	preinfo := engine.AnalysisInfo{}
	fld := strings.Fields(info)
	n := len(fld)
	for i := 0; i < n; i++ {
		switch fld[i] {
		case "depth":
			if i+1 < n {
				preinfo.Depth, _ = strconv.Atoi(fld[i+1])
				i++
			}
		case "nodes":
			if i+1 < n {
				preinfo.Nodes, _ = strconv.ParseInt(fld[i+1], 10, 64)
				i++
			}
		case "nps":
			if i+1 < n {
				preinfo.NPS, _ = strconv.ParseInt(fld[i+1], 10, 64)
				i++
			}
		case "time":
			if i+1 < n {
				if v, err := strconv.ParseInt(fld[i+1], 10, 64); err == nil {
					preinfo.TimeMs = v
				}
				i++
			}
		case "score":
			if i+2 < n {
				typ := fld[i+1]
				val := fld[i+2]
				if typ == "cp" {
					if v, err := strconv.Atoi(val); err == nil {
						preinfo.ScoreCP = v
						preinfo.MateIn = 0
					}
				} else if typ == "mate" {
					if v, err := strconv.Atoi(val); err == nil {
						preinfo.MateIn = v
					}
				}
				i += 2
			}
		case "pv":
			if i+1 < n {
				pvStrs := make([]string, 0, n-i-1)
				for j := i + 1; j < n; j++ {
					pvStrs = append(pvStrs, fld[j])
				}
				preinfo.UCIPV = pvStrs
				if len(pvStrs) > 0 {
					preinfo.UCIBestMove = pvStrs[0]
				}
				// try to parse PV -> []base.Move using lastBoard (if available)
				e.mu.RLock()
				last := e.lastBoard // copy
				e.mu.RUnlock()

				if len(last.Mailbox) > 0 {
					parsedPV := make([]base.Move, 0, len(pvStrs))
					for _, s := range pvStrs {
						if pm := parseUCIStringToMove(s, last.Mailbox); pm != nil {
							parsedPV = append(parsedPV, *pm)
						} else {
							break // can't parse further — stop
						}
					}
					if len(parsedPV) > 0 {
						preinfo.PV = parsedPV
						// BestMove -> pointer to copy of first element
						mv0 := parsedPV[0]
						preinfo.BestMove = &mv0
					}
				}
				// pv tag is last in common UCI "info" lines; break parsing
				i = n
			}
		default:
			// skip
		}
	}

	// write into shared e.info and publish
	e.mu.Lock()
	e.info = preinfo
	e.mu.Unlock()

	// publish a copy to subscribers
	e.publish(preinfo)
}

func (e *UCIExecutor) saveBest(best string) {
	e.logx.Debugf("save best move: %s", best)
	f := strings.Fields(best)
	if len(f) >= 2 {
		bm := f[1]

		e.mu.Lock()
		e.running = false
		// set UCIBestMove
		e.info.UCIBestMove = bm

		// if PV empty, try to set PV/basic BestMove from bm using lastBoard
		if len(e.info.PV) == 0 {
			e.mu.RUnlock() // temporarily unlock to use parse util that takes lastBoard under RLock (we'll lock again below)
			e.mu.RLock()
			last := e.lastBoard
			e.mu.RUnlock()

			if len(last.Mailbox) > 0 {
				if pm := parseUCIStringToMove(bm, last.Mailbox); pm != nil {
					// put into e.info (we already have e.mu locked above, so relock)
					e.mu.Lock()
					e.info.PV = []base.Move{*pm}
					mv0 := *pm
					e.info.BestMove = &mv0
					e.mu.Unlock()
				} else {
					e.mu.Unlock()
				}
			} else {
				e.mu.Unlock()
			}
		} else {
			e.mu.Unlock()
		}

		// notify WaitDone() (non-blocking)
		select {
		case e.bestMoveCh <- struct{}{}:
		default:
		}
	}
}

func (e *UCIExecutor) publish(info engine.AnalysisInfo) {
	e.submu.Lock()
	defer e.submu.Unlock()

	for _, ch := range e.subs {
		select {
		case ch <- info:
		default:
		}
	}
}

// return *base.Move or nil
func parseUCIStringToMove(u string, mailbox base.Mailbox) *base.Move {
	if len(u) < 4 {
		return nil
	}
	from := u[0:2]
	to := u[2:4]

	fromIdx, err := base.SquareFromAlgebraic(from)
	if err != nil {
		return nil
	}
	toIdx, err := base.SquareFromAlgebraic(to)
	if err != nil {
		return nil
	}

	mv := base.Move{
		From: base.ConvIndexToPoint(fromIdx),
		To:   base.ConvIndexToPoint(toIdx),
	}

	// determine piece on from-square (if any)
	p := mailbox[fromIdx]
	mv.Piece = p

	// promotion (e2e8q or e7e8q) - optional 5th char
	if len(u) >= 5 {
		c := u[4]
		isWhite := false
		switch p {
		case base.WPawn, base.WKing, base.WQueen, base.WBishop, base.WKnight, base.WRook:
			isWhite = true
		case base.BPawn, base.BKing, base.BQueen, base.BBishop, base.BKnight, base.BRook:
			isWhite = false
		default:
			// fallback: infer color from char? best-effort skip
			isWhite = true
		}
		switch c {
		case 'q', 'Q':
			if isWhite {
				mv.Piece = base.WQueen
			} else {
				mv.Piece = base.BQueen
			}
		case 'r', 'R':
			if isWhite {
				mv.Piece = base.WRook
			} else {
				mv.Piece = base.BRook
			}
		case 'b', 'B':
			if isWhite {
				mv.Piece = base.WBishop
			} else {
				mv.Piece = base.BBishop
			}
		case 'n', 'N':
			if isWhite {
				mv.Piece = base.WKnight
			} else {
				mv.Piece = base.BKnight
			}
		default:
			// unknown char -> leave mv.Piece as original (best-effort)
		}
	}

	return &mv
}
