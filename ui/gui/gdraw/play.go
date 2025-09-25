package gdraw

import (
	"evilchess/src"
	"evilchess/src/base"
	"evilchess/src/engine"
	"evilchess/src/engine/myengine"
	"evilchess/src/engine/uci"
	"evilchess/ui/gui/gctx"
	"evilchess/ui/gui/ghelper"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// GUIPlayDrawer реализует Scene
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

	// flip board
	flipped bool

	// clocks (in seconds)
	whiteClock   float64
	blackClock   float64
	clockRunning bool
	lastTick     time.Time

	// engine
	engineNotValid bool
	engineThinking bool
	engineMu       sync.Mutex
	engineDoneCh   chan struct{}

	// buttons
	buttons     []*ghelper.Button
	idxResign   int
	idxFlip     int
	idxEngineGo int
	idxBack     int

	// message box reuse
	msg *ghelper.MessageBox

	prevMouseDown bool
}

func NewGUIPlayDrawer(ctx *gctx.GUIGameContext) *GUIPlayDrawer {
	pd := &GUIPlayDrawer{
		selectedSq:   -1,
		dragFrom:     -1,
		engineDoneCh: make(chan struct{}, 1),
		whiteClock:   5 * 60,
		blackClock:   5 * 60,
		clockRunning: true,
		lastTick:     time.Now(),
	}

	if ctx.Builder == nil {
		ctx.Builder = src.NewBuilderBoard(ctx.Logx)
		ctx.Builder.CreateClassic()
	} else if ctx.Builder.Status() == base.InvalidGame {
		ctx.Builder.CreateClassic()
	}

	if ctx.Config.Engine == "internal" {
		e := myengine.NewEvilEngine()
		ctx.Builder.SetEngineWorker(e)
		ctx.Builder.SetEngineLevel(engine.LevelFive)
	} else if ctx.Config.Engine == "external" && ctx.Config.UCIPath != "" {
		e := uci.NewUCIExec(ctx.Logx, ctx.Config.UCIPath)
		ctx.Builder.SetEngineWorker(e)
		ctx.Builder.SetEngineLevel(engine.LevelFive)
	} else {
		pd.engineNotValid = true
	}

	pd.recalcLayout(ctx)
	pd.makeLayoutButtons(ctx)
	pd.msg = &ghelper.MessageBox{}
	return pd
}

// using boardX/Y/size
func (pd *GUIPlayDrawer) recalcLayout(ctx *gctx.GUIGameContext) {
	ww := ctx.Config.WindowW
	wh := ctx.Config.WindowH

	// min(600, 70% высоты)
	maxSize := ww - 400
	if maxSize > wh-120 {
		maxSize = wh - 120
	}
	if maxSize < 320 {
		maxSize = 320
	}
	pd.boardSize = maxSize
	pd.sqSize = pd.boardSize / 8
	pd.boardX = (ww - pd.boardSize) / 2
	pd.boardY = (wh-pd.boardSize)/2 - 20
}

func (pd *GUIPlayDrawer) makeLayoutButtons(ctx *gctx.GUIGameContext) {
	pd.buttons = []*ghelper.Button{}

	// небольшой helper
	addBtn := func(label string, x, y, w, h int) int {
		img := ghelper.RenderRoundedRect(w, h, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
		b := &ghelper.Button{
			Label: label,
			X:     x, Y: y, W: w, H: h,
			Image: img,
			Scale: 1.0, TargetScale: 1.0, OffsetY: 0, TargetOffsetY: 0, AnimSpeed: 10.0,
		}
		idx := len(pd.buttons)
		pd.buttons = append(pd.buttons, b)
		return idx
	}

	x := pd.boardX - 200
	if x < 20 {
		x = 20
	}
	y := pd.boardY + 160
	w, h := 160, 48
	pd.idxResign = addBtn(ctx.AssetsWorker.Lang().T("play.newgame"), x, y, w, h)
	y += h + 14
	pd.idxFlip = addBtn(ctx.AssetsWorker.Lang().T("play.flip"), x, y, w, h)
	y += h + 14
	pd.idxEngineGo = addBtn(ctx.AssetsWorker.Lang().T("play.engine_go"), x, y, w, h)
	y += h + 14
	pd.idxBack = addBtn(ctx.AssetsWorker.Lang().T("button.back"), x, y, w, h)
}

// Update
func (pd *GUIPlayDrawer) Update(ctx *gctx.GUIGameContext) (SceneType, error) {
	// ресайз/реагирование
	pd.recalcLayout(ctx)

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
	if pd.clockRunning {
		if ctx.Builder.IsWhiteToMove() {
			pd.whiteClock -= dt
		} else {
			pd.blackClock -= dt
		}
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

			// if ghelper.PointInRect(mx, my, pd.aboutBoxX, pd.aboutBoxY, pd.aboutBoxS, pd.aboutBoxS) {
			// 	ghelper.ShowMessage(&pd.msg, ctx.AssetsWorker.Lang().T("about.body"), nil)
			// 	return SceneNotChanged, nil
			// }

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
			switch i {
			case pd.idxResign:
				// start new game
				ctx.Builder.CreateClassic()
				pd.selectedSq = -1
				pd.flipped = false
				pd.whiteClock = 5 * 60
				pd.blackClock = 5 * 60
			case pd.idxFlip:
				if ctx.Builder.CountHalfMoves() != 0 {
					pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.flip_warning"), nil)
				} else {
					pd.flipped = !pd.flipped
				}
			case pd.idxEngineGo:
				if pd.engineNotValid == false {
					go pd.startEngineMoveAsync(ctx)
				} else {
					pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.no_engine"), nil)
				}
			case pd.idxBack:
				return SceneMenu, nil
			}
		}
	}

	// Board interaction: drag & click-click
	if inBoard(mx, my, pd.boardX, pd.boardY, pd.sqSize) && !pd.engineThinking && !pd.msg.Open {
		sq := pixelToSquare(mx, my, pd.boardX, pd.boardY, pd.sqSize, pd.flipped)

		// start drag on mouse press (justPressed) if player piece present AND piece belongs to player
		if justPressed && !pd.dragging {
			mb := ctx.Builder.CurrentBoard()
			piece := base.GetPieceAt(&mb, base.ConvIndexToPoint(sq))
			if piece != base.InvalidPiece && pd.isPieceOwnedByPlayer(ctx, piece) {
				pd.dragging = true
				pd.dragFrom = sq
				pd.selectedSq = sq
				pd.dragImg = ctx.AssetsWorker.Piece(piece)
				// offset inside square
				pd.dragOffsetX = mx - (pd.boardX + (func() int {
					f, r := indexToFileRank(sq)
					fs := f
					rs := 7 - r
					if pd.flipped {
						fs = 7 - f
						rs = 7 - rs
					}
					return fs * pd.sqSize
				}()))
				pd.dragOffsetY = my - (pd.boardY + (func() int {
					_, r := indexToFileRank(sq)
					// fs := f
					rs := 7 - r
					if pd.flipped {
						// fs = 7 - f
						rs = 7 - rs
					}
					return rs * pd.sqSize
				}()))
			}
		} else if pd.dragging && justReleased {
			// drop: try move from dragFrom -> sq
			if pd.dragFrom >= 0 {
				mb := ctx.Builder.CurrentBoard()
				mv := base.Move{
					From:  base.ConvIndexToPoint(pd.dragFrom),
					To:    base.ConvIndexToPoint(sq),
					Piece: base.GetPieceAt(&mb, base.ConvIndexToPoint(pd.dragFrom)),
				}
				if status := ctx.Builder.Move(mv); status == base.InvalidGame {
					pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
					ctx.Logx.Errorf("error move: %s", mv.String())
				}
			}
			pd.dragging = false
			pd.dragFrom = -1
			pd.dragImg = nil
			pd.selectedSq = -1
		} else if !pd.dragging && justReleased {
			// click-click behavior: on mouse up (release) treat as click
			if pd.selectedSq == -1 {
				// select if piece and belongs to player
				mb := ctx.Builder.CurrentBoard()
				piece := base.GetPieceAt(&mb, base.ConvIndexToPoint(sq))
				if piece != base.InvalidPiece && pd.isPieceOwnedByPlayer(ctx, piece) {
					pd.selectedSq = sq
				}
			} else {
				// attempt move
				mb := ctx.Builder.CurrentBoard()
				mv := base.Move{
					From:  base.ConvIndexToPoint(pd.dragFrom),
					To:    base.ConvIndexToPoint(sq),
					Piece: base.GetPieceAt(&mb, base.ConvIndexToPoint(pd.dragFrom)),
				}
				if status := ctx.Builder.Move(mv); status == base.InvalidGame {
					pd.msg.ShowMessage(ctx.AssetsWorker.Lang().T("play.bad_move"), nil)
					ctx.Logx.Errorf("error move: %s", mv.String())
				}
				pd.selectedSq = -1
			}
		}
	} else {
		// click outside board cancels selection on release
		if justReleased {
			pd.selectedSq = -1
			if pd.dragging {
				pd.dragging = false
				pd.dragFrom = -1
				pd.dragImg = nil
			}
		}
	}

	return SceneNotChanged, nil
}

func (pd *GUIPlayDrawer) isPieceOwnedByPlayer(ctx *gctx.GUIGameContext, p base.Piece) bool {
	switch p {
	case base.WKing, base.WQueen, base.WBishop, base.WKnight, base.WRook, base.WPawn:
		return ctx.Builder.IsWhiteToMove()
	case base.BKing, base.BQueen, base.BBishop, base.BKnight, base.BRook, base.BPawn:
		return !ctx.Builder.IsWhiteToMove()
	default:
		return false
	}
}

// async call ctx.Builder.EngineMove
func (pd *GUIPlayDrawer) startEngineMoveAsync(ctx *gctx.GUIGameContext) {
	pd.engineMu.Lock()
	if pd.engineThinking {
		pd.engineMu.Unlock()
		return
	}
	pd.engineThinking = true
	pd.engineMu.Unlock()

	// run engine
	go func() {
		if status := ctx.Builder.EngineMove(); status == base.InvalidGame {
			// message box ???
			ctx.Logx.Error("error move engine")
		}

		// call to main loop
		pd.engineDoneCh <- struct{}{}
	}()
}

// Draw
func (pd *GUIPlayDrawer) Draw(ctx *gctx.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// draw board background (border)
	borderImg := ghelper.RenderRoundedRect(pd.boardSize+8, pd.boardSize+8, 6, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pd.boardX-4), float64(pd.boardY-4))
	screen.DrawImage(borderImg, op)

	// draw squares
	for rank := 0; rank < 8; rank++ {
		for file := 0; file < 8; file++ {
			sx := pd.boardX + file*pd.sqSize
			sy := pd.boardY + rank*pd.sqSize
			// color choice
			col := ctx.Theme.Bg
			if ((file + rank) & 1) == 0 {
				col = ctx.Theme.ButtonFill
			}
			sqImg := ebiten.NewImage(pd.sqSize, pd.sqSize)
			sqImg.Fill(col)
			op2 := &ebiten.DrawImageOptions{}
			op2.GeoM.Translate(float64(sx), float64(sy))
			screen.DrawImage(sqImg, op2)
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

		file, rank := indexToFileRank(idx)
		// screen coordinates:
		fileScreen := file
		rankScreen := 7 - rank // because rank=0 -> bottom row, screen top=0
		if pd.flipped {
			// flip horizontally and vertically
			fileScreen = 7 - fileScreen
			rankScreen = 7 - rankScreen
		}
		px := pd.boardX + fileScreen*pd.sqSize
		py := pd.boardY + rankScreen*pd.sqSize

		// skip piece if it's being dragged from this square
		if pd.dragging && pd.dragFrom == idx {
			continue
		}

		img := ctx.AssetsWorker.Piece(piece)
		if img != nil {
			iw, _ := img.Size()
			scale := float64(pd.sqSize) / float64(iw)
			op3 := &ebiten.DrawImageOptions{}
			op3.GeoM.Scale(scale, scale)
			op3.GeoM.Translate(float64(px), float64(py))
			op3.Filter = ebiten.FilterLinear
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
		file, rank := indexToFileRank(pd.selectedSq)
		if pd.flipped {
			file = 7 - file
			rank = 7 - rank
		}
		sx := pd.boardX + file*pd.sqSize
		sy := pd.boardY + rank*pd.sqSize
		// thin stroke
		ghelper.EbitenutilDrawRectStroke(screen, float64(sx)+2, float64(sy)+2, float64(pd.sqSize)-4, float64(pd.sqSize)-4, 2, ctx.Theme.Accent)
	}

	// draw engine name near top-left corner of board
	engineName := "Unknown"
	if ctx.Builder != nil && pd.engineNotValid == false {
		if ctx.Config.Engine == "internal" {
			engineName = "Internal Engine"
		} else if ctx.Config.Engine == "external" {
			engineName = fmt.Sprintf("External Engine (%s)", filepath.Base(ctx.Config.UCIPath))
		} else {

		}
	}
	text.Draw(screen, engineName, ctx.AssetsWorker.Fonts().Pixel, pd.boardX+8, pd.boardY-8, ctx.Theme.MenuText)

	// -------------------- clocks --------------------
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

	// engine clock on left
	leftX := pd.boardX - 160
	if leftX < 10 {
		leftX = 10
	}
	drawClock(leftX, pd.boardY+10, "Engine", bc, !ctx.Builder.IsWhiteToMove()) // active when black to move

	// player clock on right
	rightX := pd.boardX + pd.boardSize + 20
	drawClock(rightX, pd.boardY+10, "You", wc, ctx.Builder.IsWhiteToMove()) // active when white to move

	// draw UI buttons (animated via b.DrawAnimated)
	for _, b := range pd.buttons {
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().Normal, ctx.Theme)
	}

	// draw message box if open
	if pd.msg.Open || pd.msg.Animating {
		DrawModal(ctx, pd.msg.Scale, pd.msg.Text, screen)
	}

	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
	}
}

// inBoard проверяет, находится ли пиксель в прямоугольнике доски
func inBoard(px, py, bx, by, sqSize int) bool {
	return px >= bx && py >= by && px < bx+sqSize*8 && py < by+sqSize*8
}

// indexToFileRank: index 0..63 -> file(0..7), rank(0..7) where rank 0 == bottom (a1..h1).
func indexToFileRank(idx int) (int, int) {
	f := idx % 8
	r := idx / 8
	return f, r
}

// return: 0..63
// px,py — screen cords; flipped — chessboard flipped
func pixelToSquare(px, py, bx, by, sqSize int, flipped bool) int {
	fx := (px - bx) / sqSize
	fy := (py - by) / sqSize
	if fx < 0 {
		fx = 0
	}
	if fx > 7 {
		fx = 7
	}
	if fy < 0 {
		fy = 0
	}
	if fy > 7 {
		fy = 7
	}

	var file, rank int
	if !flipped {
		file = fx
		// fy: 0 = top row on screen -> that's rank 7, so rank = 7 - fy
		rank = 7 - fy
	} else {
		// flipped: top-left on screen corresponds to a1 (rank 0)
		file = 7 - fx
		rank = fy
	}
	return rank*8 + file
}
