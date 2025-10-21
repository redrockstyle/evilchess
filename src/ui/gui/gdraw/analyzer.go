package gdraw

import (
	"errors"
	"evilchess/src/chesslib/base"
	"evilchess/src/chesslib/engine"
	"evilchess/src/chesslib/engine/myengine"
	"evilchess/src/chesslib/engine/uci"
	"evilchess/src/ui/gui/ghelper"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type TypeCandidate struct {
	Move    base.Move
	MoveStr string
	Info    engine.AnalysisInfo
	ScoreF  float64
}

type GUIAnalyzeDrawer struct {
	// layout + cache
	boardX, boardY int
	boardSize      int
	sqSize         int
	scaledPieces   map[base.Piece]*ebiten.Image
	sqLightImg     *ebiten.Image
	sqDarkImg      *ebiten.Image
	borderImg      *ebiten.Image

	// engine subscription
	infoCh chan engine.AnalysisInfo
	unsub  func()

	// latest analysis snapshot
	mu       sync.Mutex
	lastInfo engine.AnalysisInfo
	history  []engine.AnalysisInfo

	// analyze info
	listX      int
	listOffset int
	listY      int

	// candidates list (first moves of PV/BestMove)
	typeCandidate TypeCandidate
	candidates    []TypeCandidate

	// controls
	running      bool
	paused       bool
	depthLimit   int
	owningEngine bool
	labelEngine  string

	// UI buttons
	btnStartIdx int
	btnStopIdx  int
	btnBackIdx  int
	btnUndoIdx  int
	btnRedoIdx  int
	buttons     []*ghelper.Button

	// message box (promotion etc)
	msg *ghelper.MessageBox

	// drag/select state (copied from Play scene)
	selectedSq    int
	dragging      bool
	dragFrom      int
	dragImg       *ebiten.Image
	dragOffsetX   int
	dragOffsetY   int
	pendingDrag   bool
	dragStartX    int
	dragStartY    int
	dragStartSq   int
	dragThreshold int

	prevMouseDown bool
	lastTick      time.Time
}

func NewGUIAnalyzeDrawer(ctx *ghelper.GUIGameContext) *GUIAnalyzeDrawer {
	ad := &GUIAnalyzeDrawer{
		scaledPieces:  make(map[base.Piece]*ebiten.Image),
		infoCh:        nil,
		lastTick:      time.Now(),
		depthLimit:    0,
		msg:           &ghelper.MessageBox{},
		selectedSq:    -1,
		dragFrom:      -1,
		dragStartSq:   -1,
		dragThreshold: 6,
	}
	if ctx.Config.Engine == "internal" {
		ad.labelEngine = fmt.Sprintf("%s: %s", ctx.AssetsWorker.Lang().T("analyzer.engine.title"), ctx.AssetsWorker.Lang().T("analyzer.engine.internal"))
	} else if ctx.Config.Engine == "external" {
		textBrowse := filepath.Base(ctx.Config.UCIPath)
		ad.labelEngine = fmt.Sprintf("%s: %s", ctx.AssetsWorker.Lang().T("analyzer.engine.title"), textBrowse)
	} else {
		ad.labelEngine = fmt.Sprintf("%s: %s", ctx.AssetsWorker.Lang().T("analyzer.engine.title"), ctx.AssetsWorker.Lang().T("analyzer.engine.empty"))
	}

	ad.recalcLayout(ctx)
	ad.prepareCache(ctx)

	// buttons
	x := ctx.Config.WindowW - 300
	y := ctx.Config.WindowH - 320
	w, h := 200, 48
	ad.buttons = []*ghelper.Button{}
	ad.btnStartIdx, ad.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.start"), x, y, w, h, ad.buttons)
	y += h + 12
	ad.btnStopIdx, ad.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.stop"), x, y, w, h, ad.buttons)
	y += h + 12
	ad.btnBackIdx, ad.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.back"), x, y, w, h, ad.buttons)

	// Undo/Redo under the board in center
	ux := ad.boardX + ad.boardSize/2 - h - 8
	uy := ad.boardY + ad.boardSize + 18
	ad.btnUndoIdx, ad.buttons = ghelper.AppendButton(ctx, "<", ux, uy, h, h, ad.buttons)
	ad.btnRedoIdx, ad.buttons = ghelper.AppendButton(ctx, ">", ux+h+16, uy, h, h, ad.buttons)

	// analyze info
	panelX := ctx.Config.WindowW - 380
	ad.listX = panelX + 12
	ad.listOffset = 100
	ad.listY = ad.listOffset + 28*5 + 12 // same spacing as in Draw

	return ad
}

func (ad *GUIAnalyzeDrawer) recalcLayout(ctx *ghelper.GUIGameContext) {
	ad.boardSize = ctx.Config.WindowW - 420
	if ad.boardSize < 320 {
		ad.boardSize = 320
	}
	ad.sqSize = ad.boardSize / 8
	ad.boardX = 40
	ad.boardY = 80
}

func (ad *GUIAnalyzeDrawer) prepareCache(ctx *ghelper.GUIGameContext) {
	ad.sqLightImg = ebiten.NewImage(ad.sqSize, ad.sqSize)
	ad.sqLightImg.Fill(ctx.Theme.ButtonFill)
	ad.sqDarkImg = ebiten.NewImage(ad.sqSize, ad.sqSize)
	ad.sqDarkImg.Fill(ctx.Theme.Bg)
	ad.borderImg = ghelper.RenderRoundedRect(ad.boardSize+8, ad.boardSize+8, 6, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)

	keys := []base.Piece{
		base.WKing, base.BKing,
		base.WQueen, base.BQueen,
		base.WBishop, base.BBishop,
		base.WKnight, base.BKnight,
		base.WRook, base.BRook,
		base.WPawn, base.BPawn,
	}
	for _, k := range keys {
		src := ctx.AssetsWorker.Piece(k)
		if src == nil {
			continue
		}
		dst := ebiten.NewImage(ad.sqSize, ad.sqSize)
		iw, ih := src.Size()
		if iw > 0 && ih > 0 {
			sx := float64(ad.sqSize) / float64(iw)
			sy := float64(ad.sqSize) / float64(ih)
			s := math.Min(sx, sy)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(s, s)
			tw := float64(iw) * s
			th := float64(ih) * s
			tx := (float64(ad.sqSize) - tw) / 2.0
			ty := (float64(ad.sqSize) - th) / 2.0
			op.GeoM.Translate(tx, ty)
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(src, op)
			ad.scaledPieces[k] = dst
		} else {
			ad.scaledPieces[k] = src
		}
	}
}

func (ad *GUIAnalyzeDrawer) StartAnalysis(ctx *ghelper.GUIGameContext, level engine.LevelAnalyze) error {
	if ad.running {
		return nil
	}

	var e engine.Engine
	if ctx.Config.Engine == "internal" {
		e = myengine.NewEvilEngine()
	} else if ctx.Config.Engine == "external" {
		e = uci.NewUCIExec(ctx.Logx, ctx.Config.UCIPath)
	} else {
		return errors.New("unsupported engine")
	}

	ctx.Builder.SetEngineWorker(e)

	if err := ctx.Builder.EngineWorker().Init(); err != nil {
		ctx.Builder.EngineWorker().Close()
		return fmt.Errorf("engine init failed: %w", err)
	}

	ch := make(chan engine.AnalysisInfo, 64)
	unsub := ctx.Builder.EngineWorker().Subscribe(ch)

	// set position for engine
	b := ctx.Builder.CurrentPosition()
	if err := ctx.Builder.EngineWorker().SetPosition(&b); err != nil {
		unsub()
		ctx.Builder.EngineWorker().Close()
		return fmt.Errorf("engine setposition failed: %w", err)
	}

	if err := ctx.Builder.EngineWorker().StartAnalysis(engine.LevelToParams(level)); err != nil {
		unsub()
		ctx.Builder.EngineWorker().Close()
		return fmt.Errorf("engine start analysis failed: %w", err)
	}

	ad.infoCh = ch
	ad.unsub = unsub
	ad.running = true
	ad.paused = false

	// reader goroutine
	go func() {
		for info := range ch {
			ctx.Logx.Debugf("analyze info: depth=%d time=%d ms nodes=%d pv_len=%d", info.Depth, info.TimeMs, info.Nodes, len(info.PV))
			// compute candidate first move
			var firstMove *base.Move
			if info.BestMove != nil {
				firstMove = info.BestMove
			} else if len(info.PV) > 0 {
				firstMove = &info.PV[0]
			}

			scoreF := float64(info.ScoreCP) / 100.0
			if info.MateIn != 0 {
				if info.MateIn > 0 {
					scoreF = 10000.0 - float64(info.MateIn)
				} else {
					scoreF = -10000.0 - float64(info.MateIn)
				}
			}

			ad.mu.Lock()
			ad.lastInfo = info
			if firstMove != nil {
				ms := firstMove.String()
				found := -1
				for i := range ad.candidates {
					if ad.candidates[i].MoveStr == ms {
						found = i
						break
					}
				}
				if found >= 0 {
					ad.candidates[found].Info = info
					ad.candidates[found].ScoreF = scoreF
				} else {
					ad.candidates = append(ad.candidates, TypeCandidate{
						Move:    *firstMove,
						MoveStr: ms,
						Info:    info,
						ScoreF:  scoreF,
					})
				}
				// sort and trim
				sort.Slice(ad.candidates, func(i, j int) bool {
					return ad.candidates[i].ScoreF > ad.candidates[j].ScoreF
				})
				if len(ad.candidates) > 12 {
					ad.candidates = ad.candidates[:12]
				}
			}
			if len(ad.history) >= 40 {
				copy(ad.history[1:], ad.history[0:len(ad.history)-1])
				ad.history[len(ad.history)-1] = info
			} else {
				ad.history = append(ad.history, info)
			}
			ad.mu.Unlock()
		}
	}()

	return nil
}

func (ad *GUIAnalyzeDrawer) StopAnalysis(ctx *ghelper.GUIGameContext) {
	if !ad.running {
		return
	}
	if ctx.Builder.EngineWorker() != nil {
		_ = ctx.Builder.EngineWorker().StopAnalysis()
	}
	if ad.unsub != nil {
		ad.unsub()
		ad.unsub = nil
	}
	if ad.infoCh != nil {
		// reader goroutine wait -> safe close
		close(ad.infoCh)
		ad.infoCh = nil
	}
	if ctx.Builder.EngineWorker() != nil {
		ctx.Builder.EngineWorker().Close()
	}
	ad.running = false
	ad.paused = false
}

// helper: perform move (handles promotion UI)
func (ad *GUIAnalyzeDrawer) performMoveAndRestartIfNeeded(ctx *ghelper.GUIGameContext, mv base.Move) {
	wasRunning := ad.running

	// actual move
	status := ctx.Builder.Move(mv)
	if status == base.InvalidGame {
		ad.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
		return
	}

	// clear candidates/history because position changed
	ad.mu.Lock()
	ad.candidates = nil
	ad.history = nil
	ad.lastInfo = engine.AnalysisInfo{}
	ad.mu.Unlock()

	// if analysis was running before, restart it on new position
	if wasRunning {
		// stop & restart
		ad.StopAnalysis(ctx)
		// small sleep is NOT required but safe; we'll just immediately start new analysis
		_ = ad.StartAnalysis(ctx, engine.LevelLast)
	}
}

func (ad *GUIAnalyzeDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	now := time.Now()
	dt := now.Sub(ad.lastTick).Seconds()
	ad.lastTick = now

	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justPressed := mouseDown && !ad.prevMouseDown
	justReleased := !mouseDown && ad.prevMouseDown
	ad.prevMouseDown = mouseDown

	// message box
	ad.msg.Update(ctx, mx, my, justReleased)
	ad.msg.AnimateMessage()
	if ad.msg.IsOverlayed() {
		return SceneNotChanged, nil
	}

	// buttons
	for i, b := range ad.buttons {
		clicked := b.HandleInput(mx, my, justPressed, !mouseDown && b.Pressed == true)
		b.UpdateAnim(dt)
		if clicked {
			switch i {
			case ad.btnStartIdx:
				if err := ad.StartAnalysis(ctx, engine.LevelLast); err != nil {
					ctx.Logx.Errorf("error start analyze: %v", err)
				}
			case ad.btnStopIdx:
				ad.StopAnalysis(ctx)
			case ad.btnBackIdx:
				ad.StopAnalysis(ctx)
				return SceneMenu, nil
			case ad.btnUndoIdx:
				// Undo/Redo like in Play
				status := ctx.Builder.Undo()
				ctx.Logx.Debugf("undo status=%v", status)
				// clear lastInfo so UI updates
				ad.mu.Lock()
				ad.lastInfo = engine.AnalysisInfo{}
				ad.mu.Unlock()
				// restart analysis if it was running
				ad.maybeRestartAnalysisAfterChange(ctx)
			case ad.btnRedoIdx:
				status := ctx.Builder.Redo()
				ctx.Logx.Debugf("redo status=%v", status)
				ad.mu.Lock()
				ad.lastInfo = engine.AnalysisInfo{}
				ad.mu.Unlock()
				ad.maybeRestartAnalysisAfterChange(ctx)
			}
		}
	}

	// Board interaction: reuse Play logic but WITHOUT ownership checks.
	// We allow moving both colors following the rules by delegating validation to ctx.Builder.Move.
	if inBoard(mx, my, ad.boardX, ad.boardY, ad.sqSize) && !ad.msg.Open {
		sq := pixelToSquare(mx, my, ad.boardX, ad.boardY, ad.sqSize, false)

		// mouse pressed -> prepare possible drag
		if justPressed && !ad.pendingDrag {
			mb := ctx.Builder.CurrentBoard()
			piece := base.GetPieceAt(&mb, base.ConvIndexToPoint(sq))
			ad.dragStartSq = sq
			ad.dragStartX = mx
			ad.dragStartY = my

			if piece != base.EmptyPiece {
				ad.pendingDrag = true
				ad.dragImg = ad.scaledPieces[piece]
			} else {
				ad.pendingDrag = false
			}
		}

		// check movement to start real dragging
		if mouseDown && ad.pendingDrag && !ad.dragging {
			dx := mx - ad.dragStartX
			dy := my - ad.dragStartY
			if dx*dx+dy*dy >= ad.dragThreshold*ad.dragThreshold {
				ad.dragging = true
				ad.dragFrom = ad.dragStartSq
				sqPx, sqPy := ad.indexToScreenXY(ad.dragFrom)
				ad.dragOffsetX = ad.dragStartX - sqPx
				ad.dragOffsetY = ad.dragStartY - sqPy
				ad.selectedSq = ad.dragFrom
			}
		}

		// release
		if justReleased {
			// 1) if dragging -> drop
			if ad.dragging {
				if ad.dragFrom != sq {
					mb := ctx.Builder.CurrentBoard()
					mv := base.Move{
						From:  base.ConvIndexToPoint(ad.dragFrom),
						To:    base.ConvIndexToPoint(sq),
						Piece: base.GetPieceAt(&mb, base.ConvIndexToPoint(ad.dragFrom)),
					}
					// promotion?
					if base.IsPawnPromotionFromIndices(&mb, ad.dragFrom, sq) {
						// show choices
						// ad.msg.ShowMessageWithChoices(ctx.AssetsWorker.Lang().T("play.promote"), *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
						ad.msg.ShowMessageWithChoices("", *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
							mv.Piece = v.(base.Piece)
							ad.performMoveAndRestartIfNeeded(ctx, mv)
						})
					} else {
						ad.performMoveAndRestartIfNeeded(ctx, mv)
					}
				}
				// cleanup
				ad.dragging = false
				ad.pendingDrag = false
				ad.dragFrom = -1
				ad.dragImg = nil
				ad.selectedSq = -1
				ad.dragStartSq = -1
			}

			// 2) click-click logic (selection then move)
			if ad.selectedSq != -1 {
				if ad.selectedSq != sq {
					mb := ctx.Builder.CurrentBoard()
					pntSq := base.ConvIndexToPoint(sq)
					pntSelected := base.ConvIndexToPoint(ad.selectedSq)
					piece := base.GetPieceAt(&mb, pntSelected)
					// sanity: if second click is another piece, change selection instead of attempting move
					pieceAtDest := base.GetPieceAt(&mb, pntSq)
					if pieceAtDest != base.EmptyPiece {
						// change selection to new source
						ad.selectedSq = sq
					} else if piece == base.EmptyPiece {
						ad.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
						ad.selectedSq = -1
					} else {
						mv := base.Move{
							From:  pntSelected,
							To:    pntSq,
							Piece: piece,
						}
						if base.IsPawnPromotionFromIndices(&mb, ad.selectedSq, sq) {
							// ad.msg.ShowMessageWithChoices(ctx.AssetsWorker.Lang().T("play.promote"), *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
							ad.msg.ShowMessageWithChoices("", *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
								mv.Piece = v.(base.Piece)
								ad.performMoveAndRestartIfNeeded(ctx, mv)
							})
						} else {
							ad.performMoveAndRestartIfNeeded(ctx, mv)
						}
						ad.selectedSq = -1
					}
				} else {
					// clicked same -> toggle off
					ad.selectedSq = -1
				}
				ad.pendingDrag = false
				ad.dragStartSq = -1
			} else {
				// no selection: if pendingDrag true and dragStartSq >=0 -> select
				if ad.pendingDrag && ad.dragStartSq >= 0 {
					ad.selectedSq = ad.dragStartSq
					ad.pendingDrag = false
					ad.dragStartSq = -1
				} else {
					ad.pendingDrag = false
					ad.dragStartSq = -1
				}
			}
		}
	} else {
		// clicked outside board -> cancel
		if justReleased {
			ad.selectedSq = -1
			if ad.dragging {
				ad.dragging = false
				ad.dragFrom = -1
				ad.dragImg = nil
			}
			ad.pendingDrag = false
			ad.dragStartSq = -1
		}
	}

	// clicking on candidate list (right panel)
	if justReleased {
		// candidate list params must match Draw layout
		// panelX := ctx.Config.WindowW - 380
		// listX := panelX + 12
		// // start Y after numeric header (Depth/Time/Nodes/NPS/Score + a small gap)
		// listY := 40 + 28*5 + 12 // same spacing as in Draw
		listX := ad.listX
		listY := ad.listY
		lineH := 24
		ad.mu.Lock()
		cands := make([]TypeCandidate, len(ad.candidates))
		copy(cands, ad.candidates)
		ad.mu.Unlock()
		for i := 0; i < len(cands); i++ {
			rx := listX
			ry := listY + i*(lineH+6) - 4
			// bounding box width
			if ghelper.PointInRect(mx, my, rx, ry, 340, lineH+8) {
				// clicked a candidate -> apply its move (with promotion check)
				ad.performMoveWithPromotionCheck(ctx, cands[i].Move)
				break
			}
		}
	}

	// escape -> back
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		ad.StopAnalysis(ctx)
		return SceneMenu, nil
	}

	return SceneNotChanged, nil
}

// performMoveWithPromotionCheck — helper to show promotion choices if needed
func (ad *GUIAnalyzeDrawer) performMoveWithPromotionCheck(ctx *ghelper.GUIGameContext, mv base.Move) {
	mb := ctx.Builder.CurrentBoard()
	fromIdx := base.ConvPointToIndex(mv.From)
	toIdx := base.ConvPointToIndex(mv.To)
	if base.IsPawnPromotionFromIndices(&mb, fromIdx, toIdx) {
		// ad.msg.ShowMessageWithChoices(ctx.AssetsWorker.Lang().T("play.promote"), *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
		ad.msg.ShowMessageWithChoices("", *GetChoises(ctx, ctx.Builder.IsWhiteToMove()), func(idx int, v interface{}) {
			mv.Piece = v.(base.Piece)
			ad.performMoveAndRestartIfNeeded(ctx, mv)
		})
		return
	}
	ad.performMoveAndRestartIfNeeded(ctx, mv)
}

func (ad *GUIAnalyzeDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// board border
	if ad.borderImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ad.boardX-4), float64(ad.boardY-4))
		screen.DrawImage(ad.borderImg, op)
	}

	// squares
	for r := 0; r < 8; r++ {
		for f := 0; f < 8; f++ {
			sx := ad.boardX + f*ad.sqSize
			sy := ad.boardY + r*ad.sqSize
			img := ad.sqLightImg
			if ((f + r) & 1) == 1 {
				img = ad.sqDarkImg
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(sx), float64(sy))
			screen.DrawImage(img, op)
		}
	}

	// pieces
	mailbox := ctx.Builder.CurrentBoard()
	for idx := 0; idx < 64; idx++ {
		pc := base.GetPieceAt(&mailbox, base.ConvIndexToPoint(idx))
		if pc == base.EmptyPiece {
			continue
		}
		// skip if dragging from this square
		if ad.dragging && ad.dragFrom == idx {
			continue
		}
		px, py := ad.indexToScreenXY(idx)
		img := ad.scaledPieces[pc]
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(img, op)
		}
	}

	// draw dragged piece on top
	if ad.dragging && ad.dragImg != nil {
		mx, my := ebiten.CursorPosition()
		op4 := &ebiten.DrawImageOptions{}
		iw, _ := ad.dragImg.Size()
		sc := float64(ad.sqSize) / float64(iw)
		op4.GeoM.Scale(sc, sc)
		op4.GeoM.Translate(float64(mx-ad.dragOffsetX), float64(my-ad.dragOffsetY))
		op4.Filter = ebiten.FilterLinear
		screen.DrawImage(ad.dragImg, op4)
	}

	// selection highlight
	if ad.selectedSq >= 0 {
		sx, sy := ad.indexToScreenXY(ad.selectedSq)
		ghelper.EbitenutilDrawRectStroke(screen, float64(sx)+2, float64(sy)+2, float64(ad.sqSize)-4, float64(ad.sqSize)-4, 2, ctx.Theme.Accent)
	}

	// right panel: analysis numbers and candidates
	ad.mu.Lock()
	info := ad.lastInfo
	cands := make([]TypeCandidate, len(ad.candidates))
	copy(cands, ad.candidates)
	ad.mu.Unlock()

	// titles
	text.Draw(screen, ad.labelEngine, ctx.AssetsWorker.Fonts().Pixel, ad.boardX+20, ad.boardY-20, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("analyzer.analyze.title"), ctx.AssetsWorker.Fonts().Pixel, ad.boardX+ad.boardSize+16, ad.boardY-15, ctx.Theme.MenuText)

	// panelX := ctx.Config.WindowW - 380
	// x := panelX + 12
	x := ad.listX
	y := ad.listOffset
	text.Draw(screen, fmt.Sprintf("%s: %d", ctx.AssetsWorker.Lang().T("analyzer.depth"), info.Depth), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 22
	text.Draw(screen, fmt.Sprintf("%s: %d ms", ctx.AssetsWorker.Lang().T("analyzer.time"), info.TimeMs), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 22
	text.Draw(screen, fmt.Sprintf("%s: %d", ctx.AssetsWorker.Lang().T("analyzer.nodes"), info.Nodes), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 22
	text.Draw(screen, fmt.Sprintf("%s: %d", ctx.AssetsWorker.Lang().T("analyzer.nps"), info.NPS), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 28

	scoreText := "+0.00"
	if info.MateIn != 0 {
		scoreText = fmt.Sprintf("%s %d", ctx.AssetsWorker.Lang().T("analyzer.mate_in"), info.MateIn)
	} else {
		scoreText = fmt.Sprintf("%+.2f", float64(info.ScoreCP)/100.0)
	}
	text.Draw(screen, fmt.Sprintf("%s: %s", ctx.AssetsWorker.Lang().T("analyzer.score"), scoreText), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 28

	text.Draw(screen, ctx.AssetsWorker.Lang().T("analyzer.top_moves"), ctx.AssetsWorker.Fonts().Pixel, x, y, ctx.Theme.MenuText)
	y += 18

	// draw list
	rowX := x
	rowY := y
	rowH := 22
	maxRows := 12
	cx, cy := ebiten.CursorPosition()
	for i, c := range cands {
		if i >= maxRows {
			break
		}
		rx := rowX
		ry := rowY + i*(rowH+6)
		rowW := 340
		hover := ghelper.PointInRect(cx, cy, rx, ry-4, rowW+16, rowH+4)
		if hover {
			ghelper.EbitenutilDrawRectStroke(screen, float64(rx)-2, float64(ry-4), float64(rowW+16), float64(rowH+4), 2, ctx.Theme.Accent)
		}
		text.Draw(screen, fmt.Sprintf("%2d. %s", i+1, c.MoveStr), ctx.AssetsWorker.Fonts().Pixel, rx+2, ry+14, ctx.Theme.MenuText)

		scoreS := "+0.00"
		if c.Info.MateIn != 0 {
			scoreS = fmt.Sprintf("%s %d", ctx.AssetsWorker.Lang().T("analyzer.mate"), c.Info.MateIn)
		} else {
			scoreS = fmt.Sprintf("%+.2f", float64(c.Info.ScoreCP)/100.0)
		}
		text.Draw(screen, scoreS, ctx.AssetsWorker.Fonts().Pixel, rx+140, ry+14, ctx.Theme.MenuText)

		meta := fmt.Sprintf("d%d n%d", c.Info.Depth, c.Info.Nodes)
		text.Draw(screen, meta, ctx.AssetsWorker.Fonts().Pixel, rx+220, ry+14, ctx.Theme.MenuText)
	}

	// draw buttons
	for _, b := range ad.buttons {
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
	}

	// draw messagebox (promotion)
	ad.msg.Draw(ctx, screen)
	// debug overlay
	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

// indexToScreenXY
func (ad *GUIAnalyzeDrawer) indexToScreenXY(idx int) (int, int) {
	f, r := indexToFileRank(idx)
	file := f
	rank := 7 - r
	return ad.boardX + file*ad.sqSize, ad.boardY + rank*ad.sqSize
}

// helper: restart analysis if it was running before the change
func (ad *GUIAnalyzeDrawer) maybeRestartAnalysisAfterChange(ctx *ghelper.GUIGameContext) {
	if ad.running {
		// stop (will cause goroutine to stop & unsub)
		ad.StopAnalysis(ctx)
		// small delay to let engine stop if needed (not strictly required)
		time.Sleep(50 * time.Millisecond)
		_ = ad.StartAnalysis(ctx, engine.LevelLast)
	}
}
