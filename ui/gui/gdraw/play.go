package gdraw

import (
	"evilchess/src"
	"evilchess/src/base"
	"evilchess/src/engine"
	"evilchess/src/engine/myengine"
	"evilchess/src/engine/uci"
	"evilchess/ui/gui/ghelper"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type GUIPlayDrawer struct {
	// layout
	boardX, boardY int // top-left pixel
	boardSize      int // pixel size (square*8)
	sqSize         int // pixel size per square

	// interaction
	selectedSq  int // -1 (index 0..63)
	dragging    bool
	dragFrom    int
	dragImg     *ebiten.Image
	dragOffsetX int
	dragOffsetY int

	// drag helpers
	pendingDrag   bool // mouse pressed, may become real drag if moved
	dragStartX    int
	dragStartY    int
	dragStartSq   int // square where press started
	dragThreshold int // pixels, e.g. 6

	// flip board
	flipped bool

	// clocks (in seconds)
	started    bool
	allblock   bool
	timeIsUp   bool
	whiteClock float64
	blackClock float64
	lastTick   time.Time

	// engine
	engineNotValid bool
	engineThinking bool
	engineMu       sync.Mutex
	engineDoneCh   chan struct{}

	// buttons
	msg          *ghelper.MessageBox
	buttons      []*ghelper.Button
	btnResignIdx int
	btnFlipIdx   int
	btnEngineIdx int
	btnUndoIdx   int
	btnRedoIdx   int
	btnBackIdx   int

	prevMouseDown bool

	// cache invalidation helpers
	prevWindowW, prevWindowH int
	prevThemeString          string

	// cache
	sqLightImg   *ebiten.Image
	sqDarkImg    *ebiten.Image
	borderImg    *ebiten.Image
	scaledPieces map[base.Piece]*ebiten.Image

	// game status
	status base.GameStatus
}

func NewGUIPlayDrawer(ctx *ghelper.GUIGameContext) *GUIPlayDrawer {
	pd := &GUIPlayDrawer{
		selectedSq:   -1,
		dragFrom:     -1,
		engineDoneCh: make(chan struct{}, 1),
		whiteClock:   float64(ctx.Config.Clock) * time.Hour.Minutes(),
		blackClock:   float64(ctx.Config.Clock) * time.Hour.Minutes(),
		lastTick:     time.Now(),
	}

	pd.dragThreshold = 6
	pd.dragStartSq = -1

	if ctx.Config.PlayAs == "black" {
		pd.flipped = true
	} else if ctx.Config.PlayAs == "random" {
		pd.flipped = pd.lastTick.Second()%2 == 1
	} else {
		pd.flipped = false
	}

	if !ctx.IsReady {
		ctx.Builder = src.NewBuilderBoard(ctx.Logx)
		ctx.Builder.CreateClassic()
	} else if ctx.Builder.Status() == base.InvalidGame {
		ctx.Builder.CreateClassic()
	} else {
		pd.maybeShowStatus(ctx)
	}

	if ctx.Config.UseEngine {
		if ctx.Config.Engine == "internal" {
			e := myengine.NewEvilEngine()
			ctx.Builder.SetEngineWorker(e)
			ctx.Builder.SetEngineLevel(engine.LevelAnalyze(ctx.Config.Strength))
		} else if ctx.Config.Engine == "external" && ctx.Config.UCIPath != "" {
			e := uci.NewUCIExec(ctx.Logx, ctx.Config.UCIPath)
			ctx.Builder.SetEngineWorker(e)
			ctx.Builder.SetEngineLevel(engine.LevelAnalyze(ctx.Config.Strength))
		} else {
			pd.engineNotValid = true
		}
	}

	pd.recalcLayout(ctx)

	pd.buttons = []*ghelper.Button{}
	x := 20
	y := pd.boardY + 160
	w, h := 160, 48
	pd.btnResignIdx, pd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("play.newgame"), x, y, w, h, pd.buttons)
	y += h + 14
	pd.btnFlipIdx, pd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("play.flip"), x, y, w, h, pd.buttons)
	y += h + 14
	pd.btnEngineIdx, pd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("play.engine_go"), x, y, w, h, pd.buttons)
	y += h + 14
	pd.btnBackIdx, pd.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.back"), x, y, w, h, pd.buttons)

	if ctx.Config.Training || ctx.Config.Debug {
		pd.btnUndoIdx, pd.buttons = ghelper.AppendButton(ctx, "<", pd.boardX+pd.boardSize/2-h-7, pd.boardY+pd.boardSize+14, h, h, pd.buttons)
		pd.btnRedoIdx, pd.buttons = ghelper.AppendButton(ctx, ">", pd.boardX+pd.boardSize/2+7, pd.boardY+pd.boardSize+14, h, h, pd.buttons)
	}

	pd.msg = &ghelper.MessageBox{}
	return pd
}

func (pd *GUIPlayDrawer) recalcLayout(ctx *ghelper.GUIGameContext) {
	// calculate board metrics
	pd.boardSize = ctx.Config.WindowW - 400
	if pd.boardSize < 320 {
		pd.boardSize = 320
	}
	pd.sqSize = pd.boardSize / 8
	pd.boardX = (ctx.Config.WindowW - pd.boardSize) / 2
	pd.boardY = (ctx.Config.WindowH-pd.boardSize)/2 - 20

	// store current window & theme snapshot
	pd.prevWindowW = ctx.Config.WindowW
	pd.prevWindowH = ctx.Config.WindowH
	pd.prevThemeString = ctx.Theme.String()

	// prepare cache for new sizes / theme
	pd.prepareCache(ctx)
}

func (pd *GUIPlayDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {
	// handle engine done
	select {
	case <-pd.engineDoneCh:
		pd.engineThinking = false
	default:
	}

	now := time.Now()
	dt := now.Sub(pd.lastTick).Seconds()
	pd.lastTick = now

	// clocks update
	if ctx.Config.UseClock && !pd.allblock && pd.started {
		if ctx.Builder.IsWhiteToMove() {
			pd.whiteClock -= dt
		} else {
			pd.blackClock -= dt
		}
		pd.maybeTimeIsUp(ctx)
	}

	// detect window/ theme changes
	if ctx.Config.WindowW != pd.prevWindowW || ctx.Config.WindowH != pd.prevWindowH || ctx.Theme.String() != pd.prevThemeString {
		pd.recalcLayout(ctx)
	}

	// Input
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justPressed := mouseDown && !pd.prevMouseDown
	justReleased := !mouseDown && pd.prevMouseDown
	pd.prevMouseDown = mouseDown

	// if message box open -> handle clicks on it
	if pd.msg.Open {
		if justPressed {
			// check OK button area in modal coords (we place it centered)
			// Modal geometry: centered rectangle
			bounds := text.BoundString(ctx.AssetsWorker.Fonts().Normal, ctx.AssetsWorker.Lang().T("play.flip_warning"))
			pd.msg.CollapseMessageInRect(ctx.Config.WindowW, ctx.Config.WindowH, bounds.Dx(), bounds.Dy())
		}
		// animate open/close
		pd.msg.AnimateMessage()
		return SceneNotChanged, nil
	}

	// Buttons handling
	for i, b := range pd.buttons {
		clicked := b.HandleInput(mx, my, justPressed, !mouseDown && b.Pressed == true)
		b.UpdateAnim(dt)
		if clicked {
			if ctx.Config.Debug || ctx.Config.Training {
				switch i {
				case pd.btnRedoIdx:
					pd.status = ctx.Builder.Redo()
					pd.maybeShowStatus(ctx)
				case pd.btnUndoIdx:
					pd.status = ctx.Builder.Undo()
					pd.allblock = false
				}
			}
			switch i {
			case pd.btnResignIdx:
				// start new game
				ctx.Builder.CreateClassic()
				pd.selectedSq = -1
				pd.flipped = pd.lastTick.Second()%2 == 1
				pd.whiteClock = float64(ctx.Config.Clock) * time.Hour.Minutes()
				pd.blackClock = float64(ctx.Config.Clock) * time.Hour.Minutes()
			case pd.btnFlipIdx:
				if ctx.Builder.CountHalfMoves() != 0 {
					pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.flip_warning"), nil)
				} else {
					pd.flipped = !pd.flipped
				}
			case pd.btnEngineIdx:
				if ctx.Config.UseEngine {
					if pd.engineNotValid == false {
						if !pd.started {
							pd.started = true
						}
						pd.maybeStartEngine(ctx)
					} else {
						pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.no_engine"), nil)
					}
				}
			case pd.btnBackIdx:
				ctx.IsReady = false
				return SceneMenu, nil
			}
		}
	}

	// Board interaction: drag & click-click with movement threshold
	if inBoard(mx, my, pd.boardX, pd.boardY, pd.sqSize) && !pd.engineThinking && !pd.msg.Open {
		sq := pixelToSquare(mx, my, pd.boardX, pd.boardY, pd.sqSize, pd.flipped)

		// mouse pressed -> prepare possible drag
		if justPressed && !pd.pendingDrag && !pd.engineThinking {
			mb := ctx.Builder.CurrentBoard()
			piece := base.GetPieceAt(&mb, base.ConvIndexToPoint(sq))
			// always remember where press started
			pd.dragStartSq = sq
			pd.dragStartX = mx
			pd.dragStartY = my

			if piece != base.EmptyPiece && pd.isPieceOwnedByPlayer(ctx, piece) {
				// mark possible drag start
				pd.pendingDrag = true
				// prepare drag image but only set pd.dragImg when real dragging begins

				// pd.dragImg = ctx.AssetsWorker.Piece(piece)
				pd.dragImg = pd.scaledPieces[piece]
			} else {
				// click started on empty or opponent piece — not a pending drag
				pd.pendingDrag = false
			}
		}

		// while mouse held, check movement to start real dragging
		if mouseDown && pd.pendingDrag && !pd.dragging {
			dx := mx - pd.dragStartX
			dy := my - pd.dragStartY
			if dx*dx+dy*dy >= pd.dragThreshold*pd.dragThreshold {
				// start real drag
				pd.dragging = true
				pd.dragFrom = pd.dragStartSq
				// compute offset inside square
				sqPx, sqPy := pd.indexToScreenXY(pd.dragFrom)
				pd.dragOffsetX = pd.dragStartX - sqPx
				pd.dragOffsetY = pd.dragStartY - sqPy
				// now visually show selection while dragging
				pd.selectedSq = pd.dragFrom
			}
		}

		if justReleased {
			// 1) if we were dragging -> drop to `sq`
			if pd.dragging {
				if pd.dragFrom != sq {
					mb := ctx.Builder.CurrentBoard()
					mv := base.Move{
						From:  base.ConvIndexToPoint(pd.dragFrom),
						To:    base.ConvIndexToPoint(sq),
						Piece: base.GetPieceAt(&mb, base.ConvIndexToPoint(pd.dragFrom)),
					}
					ctx.Logx.Debugf("dragging attempt move from=%d to=%d", pd.dragFrom, sq)
					if !pd.started {
						pd.started = true
					}
					pd.status = ctx.Builder.Move(mv)
					pd.maybeShowStatus(ctx)
					if pd.status != base.InvalidGame {
						pd.maybeStartEngine(ctx)
					}
				}
				// cleanup drag state
				pd.dragging = false
				pd.pendingDrag = false
				pd.dragFrom = -1
				pd.dragImg = nil
				pd.selectedSq = -1
				pd.dragStartSq = -1

			}

			// 2) Not dragging: handle click-click logic
			// If there is a selection already -> second click -> attempt move
			if pd.selectedSq != -1 {
				// second click: try move selectedSq -> sq
				if pd.selectedSq != sq {
					mb := ctx.Builder.CurrentBoard()
					pntSq := base.ConvIndexToPoint(sq)
					pntSelectedSq := base.ConvIndexToPoint(pd.selectedSq)
					pieceSq := base.GetPieceAt(&mb, pntSq)
					piece := base.GetPieceAt(&mb, pntSelectedSq)
					if pieceSq != base.EmptyPiece && pd.isPieceOwnedByPlayer(ctx, pieceSq) {
						pd.selectedSq = sq
					} else if piece == base.EmptyPiece || !pd.isPieceOwnedByPlayer(ctx, piece) {
						// sanity check failed — clear selection
						pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
						pd.selectedSq = -1
					} else {
						ctx.Logx.Debugf("click-click attempt move from=%d to=%d", pd.selectedSq, sq)
						mv := base.Move{
							From:  base.ConvIndexToPoint(pd.selectedSq),
							To:    base.ConvIndexToPoint(sq),
							Piece: piece,
						}
						if !pd.started {
							pd.started = true
						}
						pd.status = ctx.Builder.Move(mv)
						pd.maybeShowStatus(ctx)
						if pd.status != base.InvalidGame {
							pd.maybeStartEngine(ctx)
						}
						// after attempt clear selection (regardless of result)
						pd.selectedSq = -1
					}
				} else {
					// clicked same square -> toggle off selection
					pd.selectedSq = -1
				}
				// reset pending info
				pd.pendingDrag = false
				pd.dragStartSq = -1

			}

			// 3) No current selection: this is first click (press+release without drag)
			// If press started on a player's piece (pendingDrag was true) -> select that source square.
			if pd.pendingDrag && pd.dragStartSq >= 0 {
				// select source square (use dragStartSq to be precise)
				pd.selectedSq = pd.dragStartSq
				// keep pendingDrag false now (we consumed it as a click selection)
				pd.pendingDrag = false
				pd.dragStartSq = -1
			} else {
				// pressed & released on empty square (nothing to do)
				pd.pendingDrag = false
				pd.dragStartSq = -1
			}
		}

	} else {
		// clicked outside board -> cancel pending/select/drag on release
		if justReleased {
			pd.selectedSq = -1
			if pd.dragging {
				pd.dragging = false
				pd.dragFrom = -1
				pd.dragImg = nil
			}
			pd.pendingDrag = false
			pd.dragStartSq = -1
		}
	}

	// if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
	// 	if ctx.Config.Training || ctx.Config.Debug {
	// 		_ = ctx.Builder.Undo()
	// 	}
	// }
	// if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
	// 	if ctx.Config.Training || ctx.Config.Debug {
	// 		_ = ctx.Builder.Redo()
	// 	}
	// }

	// escape -> redo
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		ctx.IsReady = false
		return SceneMenu, nil
	}

	return SceneNotChanged, nil
}

func (pd *GUIPlayDrawer) maybeTimeIsUp(ctx *ghelper.GUIGameContext) {
	if pd.blackClock <= 0 || pd.whiteClock <= 0 {
		if !ctx.Config.Debug && !ctx.Config.Training {
			pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.timeisup"), nil)
		}
		pd.allblock = true
	}
}

func (pd *GUIPlayDrawer) maybeShowStatus(ctx *ghelper.GUIGameContext) {
	switch pd.status {
	case base.Check:
		if ctx.Config.Debug || ctx.Config.Training {
			pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.check"), nil)
		}
	case base.Stalemate:
		pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.stalemate"), nil)
		pd.allblock = true
	case base.Checkmate:
		pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.checkmate"), nil)
		pd.allblock = true
	case base.InvalidGame:
		if ctx.Config.Debug || ctx.Config.Training {
			pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
		}
	case base.Pass:
	default:
	}
}

// should be a method on GUIPlayDrawer
func (pd *GUIPlayDrawer) maybeStartEngine(ctx *ghelper.GUIGameContext) {
	// check end game
	if pd.allblock {
		pd.maybeShowStatus(ctx)
		return
	}
	// engine disabled or invalid -> nothing to do
	if !ctx.Config.UseEngine || pd.engineNotValid {
		return
	}

	// decide which color is the human player (same logic как в isPieceOwnedByPlayer)
	playerIsWhite := !pd.flipped
	if ctx.Builder.IsWhiteToMove() != playerIsWhite {
		// небольшая пауза, чтобы пользователь успел увидеть свой ход
		go func() {
			time.Sleep(120 * time.Millisecond) // 0.12s
			pd.startEngineMoveAsync(ctx)
		}()
	}
}

func (pd *GUIPlayDrawer) isPieceOwnedByPlayer(ctx *ghelper.GUIGameContext, p base.Piece) bool {
	isWhitePiece := false
	switch p {
	case base.WKing, base.WQueen, base.WBishop, base.WKnight, base.WRook, base.WPawn:
		isWhitePiece = true
	case base.BKing, base.BQueen, base.BBishop, base.BKnight, base.BRook, base.BPawn:
		isWhitePiece = false
	default:
		return false
	}

	playerIsWhite := !pd.flipped
	// если фигура принадлежит игроку
	if isWhitePiece != playerIsWhite {
		return false
	}

	if !(ctx.Config.Debug || ctx.Config.UseEngine == false || ctx.Config.Training) {
		// если ход игрока
		if ctx.Builder.IsWhiteToMove() != playerIsWhite {
			return false
		}
	}
	return true

}

// async call ctx.Builder.EngineMove
func (pd *GUIPlayDrawer) startEngineMoveAsync(ctx *ghelper.GUIGameContext) {
	pd.engineMu.Lock()
	if pd.engineThinking {
		pd.engineMu.Unlock()
		return
	}
	pd.engineThinking = true
	pd.engineMu.Unlock()

	// run engine
	go func() {
		pd.status = ctx.Builder.EngineMove()
		pd.maybeShowStatus(ctx)

		// call to main loop
		pd.engineDoneCh <- struct{}{}
	}()
}

// Draw
func (pd *GUIPlayDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// draw board background (border)
	if pd.borderImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(pd.boardX-4), float64(pd.boardY-4))
		screen.DrawImage(pd.borderImg, op)
	}

	// draw squares
	for rank := 0; rank < 8; rank++ {
		for file := 0; file < 8; file++ {
			sx := pd.boardX + file*pd.sqSize
			sy := pd.boardY + rank*pd.sqSize
			var img *ebiten.Image
			if ((file + rank) & 1) == 0 {
				img = pd.sqLightImg
			} else {
				img = pd.sqDarkImg
			}
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(sx), float64(sy))
			screen.DrawImage(img, op2)
		}
	}

	// draw pieces from builder board
	// get mailbox/array from builder
	mailbox := ctx.Builder.CurrentBoard()
	for idx := 0; idx < 64; idx++ {
		piece := base.GetPieceAt(&mailbox, base.ConvIndexToPoint(idx))
		if piece == base.EmptyPiece {
			continue
		}

		px, py := pd.indexToScreenXY(idx)

		// skip piece if it's being dragged from this square
		if pd.dragging && pd.dragFrom == idx {
			continue
		}

		img := pd.scaledPieces[piece]
		if img != nil {
			op3 := &ebiten.DrawImageOptions{}
			op3.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(img, op3)
		}

	}

	// draw dragged piece on top of everything
	if pd.dragging && pd.dragImg != nil {
		mx, my := ebiten.CursorPosition()
		op4 := &ebiten.DrawImageOptions{}
		iw, _ := pd.dragImg.Size()
		sc := float64(pd.sqSize) / float64(iw)
		op4.GeoM.Scale(sc, sc)
		op4.GeoM.Translate(float64(mx-pd.dragOffsetX), float64(my-pd.dragOffsetY))
		op4.Filter = ebiten.FilterLinear
		screen.DrawImage(pd.dragImg, op4)
	}

	// draw selection highlight
	if pd.selectedSq >= 0 {
		sx, sy := pd.indexToScreenXY(pd.selectedSq)
		ghelper.EbitenutilDrawRectStroke(screen, float64(sx)+2, float64(sy)+2, float64(pd.sqSize)-4, float64(pd.sqSize)-4, 2, ctx.Theme.Accent)
	}

	// draw engine name near top-left corner of board
	engineName := "Human"
	if ctx.Config.UseEngine {
		if ctx.Builder != nil && pd.engineNotValid == false {
			if ctx.Config.Engine == "internal" {
				engineName = "Internal Engine"
			} else if ctx.Config.Engine == "external" {
				engineName = fmt.Sprintf("External Engine (%s)", filepath.Base(ctx.Config.UCIPath))
			}
		}
	}
	text.Draw(screen, engineName, ctx.AssetsWorker.Fonts().Pixel, pd.boardX+8, pd.boardY-8, ctx.Theme.MenuText)

	// -------------------- clocks --------------------
	if ctx.Config.UseClock {
		// helper to render clock box
		drawClock := func(x, y int, label, timeStr string, active bool) {
			w, h := 140, 56
			img := ghelper.RenderRoundedRect(w, h, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(img, op)

			// border accent if active
			if active {
				ghelper.EbitenutilDrawRectStroke(screen, float64(x)+1, float64(y)+1, float64(w)-2, float64(h)-2, 3, ctx.Theme.Accent)
			}
			// label and time
			text.Draw(screen, label, ctx.AssetsWorker.Fonts().Pixel, x+10, y+18, ctx.Theme.MenuText)
			text.Draw(screen, timeStr, ctx.AssetsWorker.Fonts().Pixel, x+10, y+42, ctx.Theme.ButtonText)
		}

		// format times
		wc := fmt.Sprintf("%02d:%02d", int(pd.whiteClock)/60, int(pd.whiteClock)%60)
		bc := fmt.Sprintf("%02d:%02d", int(pd.blackClock)/60, int(pd.blackClock)%60)
		if pd.flipped {
			wc, bc = bc, wc
		}

		drawClock(pd.boardX+pd.boardSize+20, pd.boardY+10, "Engine", bc, pd.flipped)
		drawClock(pd.boardX+pd.boardSize+20, pd.boardY+pd.boardSize-70, "You", wc, !pd.flipped)
	}
	// draw UI buttons (animated via b.DrawAnimated)
	for _, b := range pd.buttons {
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
	}

	// draw message box if open
	if pd.msg.Open || pd.msg.Animating {
		DrawModal(ctx, pd.msg.Scale, pd.msg.Text, screen)
	}

	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

func (pd *GUIPlayDrawer) prepareCache(ctx *ghelper.GUIGameContext) {
	if pd.sqSize <= 0 || pd.boardSize <= 0 {
		return
	}

	// square images
	pd.sqLightImg = ebiten.NewImage(pd.sqSize, pd.sqSize)
	pd.sqLightImg.Fill(ctx.Theme.ButtonFill)

	pd.sqDarkImg = ebiten.NewImage(pd.sqSize, pd.sqSize)
	pd.sqDarkImg.Fill(ctx.Theme.Bg)

	// border around board
	pd.borderImg = ghelper.RenderRoundedRect(pd.boardSize+8, pd.boardSize+8, 6, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)

	// scaled pieces (cache)
	pd.scaledPieces = make(map[base.Piece]*ebiten.Image, 12)

	// explicit list of pieces
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
		// create destination image exactly sqSize x sqSize
		dst := ebiten.NewImage(pd.sqSize, pd.sqSize)

		iw, ih := src.Size()
		if iw <= 0 || ih <= 0 {
			pd.scaledPieces[k] = src
			continue
		}
		sx := float64(pd.sqSize) / float64(iw)
		sy := float64(pd.sqSize) / float64(ih)
		// keep aspect ratio: use min(sx, sy) and center if necessary
		s := math.Min(sx, sy)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(s, s)

		// center the scaled sprite inside dst (optional)
		// compute translation so image centered in cell
		tw := float64(iw) * s
		th := float64(ih) * s
		tx := (float64(pd.sqSize) - tw) / 2.0
		ty := (float64(pd.sqSize) - th) / 2.0
		op.GeoM.Translate(tx, ty)

		op.Filter = ebiten.FilterLinear
		dst.DrawImage(src, op)
		pd.scaledPieces[k] = dst
	}
}

func (pd *GUIPlayDrawer) indexToScreenXY(idx int) (x, y int) {
	f, r := indexToFileRank(idx) // 0..7
	file := f
	rank := 7 - r
	if pd.flipped {
		file = 7 - file
		rank = 7 - rank
	}
	return pd.boardX + file*pd.sqSize, pd.boardY + rank*pd.sqSize
}

// func (pd *GUIPlayDrawer) screenToIndex(px, py int) int {
// 	fx := (px - pd.boardX) / pd.sqSize
// 	fy := (py - pd.boardY) / pd.sqSize
// 	if fx < 0 {
// 		fx = 0
// 	} else if fx > 7 {
// 		fx = 7
// 	}
// 	if fy < 0 {
// 		fy = 0
// 	} else if fy > 7 {
// 		fy = 7
// 	}
// 	if !pd.flipped {
// 		file := fx
// 		rank := 7 - fy
// 		return rank*8 + file
// 	}
// 	file := 7 - fx
// 	rank := fy
// 	return rank*8 + file
// }
