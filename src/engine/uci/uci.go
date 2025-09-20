package uci

import (
	"bufio"
	"context"
	"errors"
	"evilchess/src/base"
	"evilchess/src/engine"
	"evilchess/src/logic/convert/convfen"
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
	return e.SetPositionFEN(convfen.ConvertBoardToFEN(*b))
}

func (e *UCIExecutor) SetPositionFEN(fen string) error {
	if tu, err := convfen.ConvertFENToBoard(fen); err == nil { // pizdec reshenie XD
		e.whiteToMove = tu.WhiteToMove
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
	if e.cmd != nil {
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

func (e *UCIExecutor) saveInfo(info string) {
	preinfo := engine.AnalysisInfo{}
	fld := strings.Fields(info)
	len := len(fld)
	for i := 0; i < len; i++ {
		switch fld[i] {
		case "depth":
			if i+1 < len {
				preinfo.Depth, _ = strconv.Atoi(fld[i+1])
				i++
			}
		case "nodes":
			if i+1 < len {
				preinfo.Nodes, _ = strconv.ParseInt(fld[i+1], 10, 64)
				i++
			}
		case "nps":
			if i+1 < len {
				preinfo.Nodes, _ = strconv.ParseInt(fld[i+1], 10, 64)
				i++
			}
		case "time":
			if i+1 < len {
				if v, err := strconv.ParseInt(fld[i+1], 10, 64); err == nil {
					preinfo.TimeMs = v
				}
				i++
			}
		case "score":
			if i+2 < len {
				typ := fld[i+1]
				val := fld[i+2]
				if typ == "cp" {
					if v, err := strconv.Atoi(val); err == nil {
						preinfo.ScoreCP = v
						preinfo.MateIn = 0
					}
				} else if typ == "mate" {
					if v, err := strconv.Atoi(val); err == nil {
						// UCI mate is in plies to mate (positive or negative)
						preinfo.MateIn = v
					}
				}
				i += 2
			}
		case "pv":
			if i+1 < len {
				pv := make([]string, 0, len-i-1)
				for j := i + 1; j < len; j++ {
					pv = append(pv, fld[j])
				}
				preinfo.UCIPV = pv
				preinfo.UCIBestMove = pv[len-i-2]
				i = len // done, bc pv tag is always last
			}
		default:
			// skip like "seldepth", "currmove" etc
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.publish(preinfo)
}

func (e *UCIExecutor) saveBest(best string) {
	e.logx.Debugf("save best move: %s", best)
	f := strings.Fields(best)
	if len(f) >= 2 {
		bm := f[1]
		e.mu.Lock()
		e.running = false

		e.info.UCIBestMove = bm
		if len(e.info.PV) == 0 && e.info.BestMove != nil {
			e.info.UCIPV = []string{bm}
		}

		select {
		// signal to WaitDone()
		case e.bestMoveCh <- struct{}{}:
		default:
		}

		e.mu.Unlock()
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
