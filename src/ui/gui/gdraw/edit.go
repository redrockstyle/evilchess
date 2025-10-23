package gdraw

import (
	"evilchess/src/chesslib/base"
	"evilchess/src/chesslib/logic/convert/convfen"
	"evilchess/src/ui/gui/ghelper"
	"evilchess/src/ui/gui/ghelper/gclipboard"
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

type GUIEditDrawer struct {
	board base.Board
	msg   *ghelper.MessageBox

	// layout
	boardX, boardY int
	boardSize      int
	sqSize         int

	// interaction
	prevMouseDown bool
	prevRightDown bool

	// actual FEN
	fen string

	// selection & preview
	selectedSq     int
	previewPiece   base.Piece
	previewIsWhite bool

	// modes
	modePlace  bool
	modeMove   bool
	modeDelete bool

	// buttons
	buttons []*ghelper.Button
	// board setup
	btnWhiteMove int
	btnBlackMove int
	btnCastingWK int
	btnCastingWQ int
	btnCastingBK int
	btnCastingBQ int
	btnStartPos  int
	btnFlip      int
	btnTraining  int
	btnClear     int
	// clipboard
	btnPaste int
	btnCopy  int
	// selectors
	btnPlace  int
	btnMove   int
	btnDelete int
	// navigation
	btnPlay    int
	btnAnalyze int
	btnBack    int

	// cache visuals
	scaledPieces map[base.Piece]*ebiten.Image
	sqLightImg   *ebiten.Image
	sqDarkImg    *ebiten.Image
	borderImg    *ebiten.Image

	lastTick time.Time
}

func NewGUIEditDrawer(ctx *ghelper.GUIGameContext) *GUIEditDrawer {
	ed := &GUIEditDrawer{
		selectedSq:     -1,
		previewPiece:   base.EmptyPiece,
		previewIsWhite: true,
		lastTick:       time.Now(),
		msg:            &ghelper.MessageBox{},
	}
	if b, _ := convfen.ConvertFENToBoard(base.FEN_EMPTY_GAME); b != nil {
		ed.fen = base.FEN_EMPTY_GAME
		ed.board = *b
	}

	spacingX, spacingY := 10, 16
	// layout
	ed.boardSize = ctx.Config.WindowW - 400
	if ed.boardSize < 320 {
		ed.boardSize = 320
	}
	ed.sqSize = ed.boardSize / 8
	ed.boardX = (ctx.Config.WindowW - ed.boardSize) / 2
	ed.boardY = (ctx.Config.WindowH-ed.boardSize)/2 - 20

	// prepare visuals cache
	ed.prepareCache(ctx)

	// buttons
	ed.buttons = []*ghelper.Button{}
	x := ed.boardX + ed.boardSize + 20
	y := ed.boardY
	w, h := 160, 44
	// navigation
	ed.btnPlay, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.play"), ctx.Config.WindowW-w-25, ctx.Config.WindowH-h-60, w, h, ed.buttons)
	ed.btnAnalyze, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.analyze"), ctx.Config.WindowW-w*2-spacingX-25, ctx.Config.WindowH-h-60, w, h, ed.buttons)
	ed.btnBack, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("button.back"), ctx.Config.WindowW-w*3-spacingX*2-25, ctx.Config.WindowH-h-60, w, h, ed.buttons)
	// context board
	ed.btnWhiteMove, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.move.white"), x, y, w/2-spacingX, h, ed.buttons)
	ed.btnBlackMove, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.move.black"), x+w/2+spacingX, y, w/2-spacingX, h, ed.buttons)
	ed.btnCastingWK, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.casting.wk"), x, y+h+spacingY, w/2-spacingX, h, ed.buttons)
	ed.btnCastingBK, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.casting.bk"), x+w/2+spacingX, y+h+spacingY, w/2-spacingX, h, ed.buttons)
	ed.btnCastingWQ, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.casting.wq"), x, y+h*2+spacingY*2, w/2-spacingX, h, ed.buttons)
	ed.btnCastingBQ, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.casting.bq"), x+w/2+spacingX, y+h*2+spacingY*2, w/2-spacingX, h, ed.buttons)
	ed.btnStartPos, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.start_position"), x, y+h*3+spacingY*3, w, h, ed.buttons)
	ed.btnFlip, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.flip"), x, y+h*4+spacingY*4, w, h, ed.buttons)
	ed.btnTraining, ed.buttons = ghelper.AppendButton(ctx, "", x, y+h*5+spacingY*5, w, h, ed.buttons)
	ed.btnClear, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.clear"), x, y+h*6+spacingY*6, w, h, ed.buttons)
	// clipboard
	ed.btnPaste, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.paste_fen"), x, y+h*8+spacingY*7, w, h, ed.buttons)
	ed.btnCopy, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.copy_fen"), x, y+h*9+spacingY*8, w, h, ed.buttons)

	// mode buttons near pieces
	mBtnW, mBtnY := 140, 36
	ed.btnPlace, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.place"), ed.boardX+8-160, ed.boardY+ed.boardSize+8-120, mBtnW, mBtnY, ed.buttons)
	ed.btnMove, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.move"), ed.boardX+8-160, ed.boardY+ed.boardSize+8+36+5-120, mBtnW, mBtnY, ed.buttons)
	ed.btnDelete, ed.buttons = ghelper.AppendButton(ctx, ctx.AssetsWorker.Lang().T("editor.delete"), ed.boardX+8-160, ed.boardY+ed.boardSize+8+36*2+5*2-120, mBtnW, mBtnY, ed.buttons)

	// default mode = Place
	ed.modePlace = true
	ed.modeMove = false
	ed.modeDelete = false

	return ed
}

// Update основной цикл
func (ed *GUIEditDrawer) Update(ctx *ghelper.GUIGameContext) (SceneType, error) {

	now := time.Now()
	dt := now.Sub(ed.lastTick).Seconds()
	_ = dt
	ed.lastTick = now

	// mouse
	mx, my := ebiten.CursorPosition()
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	rightDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	justPressed := mouseDown && !ed.prevMouseDown
	justReleased := !mouseDown && ed.prevMouseDown
	justRightPressed := rightDown && !ed.prevRightDown
	ed.prevMouseDown = mouseDown
	ed.prevRightDown = rightDown

	// if message box open -> handle clicks on it
	if ed.msg.Open {
		ed.msg.Update(ctx, mx, my, justReleased)
		ed.msg.AnimateMessage()
		return SceneNotChanged, nil
	}

	// buttons handling
	for i, b := range ed.buttons {
		clicked := b.HandleInput(mx, my, justPressed, !mouseDown && b.Pressed == true)
		b.UpdateAnim(dt)
		if clicked {
			switch i {
			case ed.btnPlay:
				if status, err := ctx.Builder.CreateFromBoard(&ed.board); err != nil || status == base.InvalidGame {
					ctx.Logx.Errorf("Bad Position: status->%s err->%v", status.String(), err)
					ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("editor.apply_failed"), nil)
				} else {
					// apply successful -> switch to Play scene
					ctx.IsReady = true
					return ScenePlay, nil
				}
			case ed.btnAnalyze:
				if status, err := ctx.Builder.CreateFromBoard(&ed.board); err != nil || status == base.InvalidGame {
					ctx.Logx.Errorf("Bad Position: status->%s err->%v", status.String(), err)
					ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("editor.apply_failed"), nil)
				} else {
					// apply successful -> switch to Play scene
					ctx.IsReady = true
					return SceneAnalyzer, nil
				}
			case ed.btnWhiteMove:
				ed.board.WhiteToMove = true
			case ed.btnBlackMove:
				ed.board.WhiteToMove = false
			case ed.btnCastingWK:
				if base.IsPossibleCasting(ed.board.Mailbox, true, false, true) {
					ed.board.Casting.WK = !ed.board.Casting.WK
				}
			case ed.btnCastingWQ:
				if base.IsPossibleCasting(ed.board.Mailbox, true, true, true) {
					ed.board.Casting.WQ = !ed.board.Casting.WQ
				}
			case ed.btnCastingBK:
				if base.IsPossibleCasting(ed.board.Mailbox, false, false, true) {
					ed.board.Casting.BK = !ed.board.Casting.BK
				}
			case ed.btnCastingBQ:
				if base.IsPossibleCasting(ed.board.Mailbox, false, true, true) {
					ed.board.Casting.BQ = !ed.board.Casting.BQ
				}
			case ed.btnBack:
				return SceneMenu, nil
			case ed.btnPaste:
				// read clipboard and try to CreateFromFEN (fallback to builder's CreateFromFEN if available)
				if s, err := gclipboard.ReadAll(); err == nil {
					str := s
					if len(s) > 60 {
						str = s[:60] + "..."
					}
					ed.msg.ShowMessage(fmt.Sprintf("%v: \"%s\"", ctx.AssetsWorker.Lang().T("message.paste"), str), func() {
						b, err2 := convfen.ConvertFENToBoard(s)
						if err2 != nil {
							ctx.Logx.Errorf("error parse FEN: %v", err2)
							ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("editor.fen_invalid"), nil)
						} else {
							ed.board = *b
							ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("editor.fen_loaded"), nil)
							ed.fen = convfen.ConvertBoardToFEN(ed.board)
						}
					})
				} else {
					ed.msg.ShowMessage("clipboard read error", nil)
				}
			case ed.btnCopy:
				fen := convfen.ConvertBoardToFEN(ed.board)
				if err := gclipboard.WriteAll(fen); err != nil {
					ctx.Logx.Errorf("error copy FEN to clipboard: %v", err)
					ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("message.copy.failed"), nil)
				} else {
					ed.msg.ShowMessage(fmt.Sprintf("%v: \"%s\"", ctx.AssetsWorker.Lang().T("message.copy"), fen), nil)
				}
			case ed.btnStartPos:
				if b, _ := convfen.ConvertFENToBoard(base.FEN_START_GAME); b != nil {
					ed.board = *b
				} else {
					ctx.Logx.Error("error create start position")
				}
			case ed.btnFlip:
				// TODO: flipped
				ed.msg.ShowMessage(ctx.AssetsWorker.Lang().T("message.todo"), nil)
				//
			case ed.btnTraining:
				ctx.Config.Training = !ctx.Config.Training
			case ed.btnClear:
				if b, _ := convfen.ConvertFENToBoard(base.FEN_EMPTY_GAME); b != nil {
					ed.fen = base.FEN_EMPTY_GAME
					ed.board = *b
				}
			case ed.btnPlace:
				ed.modePlace = true
				ed.modeMove = false
				ed.modeDelete = false
			case ed.btnMove:
				ed.modePlace = false
				ed.modeMove = true
				ed.modeDelete = false

				ed.selectedSq = -1
				ed.previewPiece = base.EmptyPiece
			case ed.btnDelete:
				ed.modePlace = false
				ed.modeMove = false
				ed.modeDelete = true

				ed.selectedSq = -1
				ed.previewPiece = base.EmptyPiece
			}
			ed.fen = convfen.ConvertBoardToFEN(ed.board)
		}
	}

	// palette clicks and right-click color toggle
	if justPressed {
		wx, wy := ed.paletteWhiteRect()
		bx, by := ed.paletteBlackRect()

		// check white palette column
		if ghelper.InsideRect(mx, my, wx, wy, ed.sqSize, len(whitePaletteOrder())*(ed.sqSize+8)-8) {
			cell := (my - wy) / (ed.sqSize + 8)
			if cell >= 0 && cell < len(whitePaletteOrder()) {
				piece := whitePaletteOrder()[cell]
				ed.previewPiece = piece
				ed.previewIsWhite = true

			}
		} else if ghelper.InsideRect(mx, my, bx, by, ed.sqSize, len(blackPaletteOrder())*(ed.sqSize+8)-8) {
			// check black palette column
			cell := (my - by) / (ed.sqSize + 8)
			if cell >= 0 && cell < len(blackPaletteOrder()) {
				piece := blackPaletteOrder()[cell]
				ed.previewPiece = piece
				ed.previewIsWhite = false
			}
		}

	}

	if justRightPressed && ed.previewPiece != base.EmptyPiece {
		ed.previewPiece = base.SwapColorPiece(ed.previewPiece)
		ed.previewIsWhite = !ed.previewIsWhite
	}

	// board interactions
	if inBoard(mx, my, ed.boardX, ed.boardY, ed.sqSize) && !ed.msg.Open {
		sq := pixelToSquare(mx, my, ed.boardX, ed.boardY, ed.sqSize, false) // not using flipped here, adjust if needed
		if justPressed {
			// Place mode
			if ed.modePlace {
				if ed.previewPiece != base.EmptyPiece {
					// if same piece present -> toggle to empty
					if ed.board.Mailbox[sq] == ed.previewPiece {
						ed.board.Mailbox[sq] = base.EmptyPiece
					} else {
						ed.board.Mailbox[sq] = ed.previewPiece
					}
				}
			} else if ed.modeDelete {
				ed.board.Mailbox[sq] = base.EmptyPiece
			} else if ed.modeMove {
				// move mode: select source -> then destination
				if ed.selectedSq == -1 {
					if ed.board.Mailbox[sq] != base.EmptyPiece {
						ed.selectedSq = sq
					}
				} else {
					// attempt move
					if ed.selectedSq != sq {
						ed.board.Mailbox[sq] = ed.board.Mailbox[ed.selectedSq]
						ed.board.Mailbox[ed.selectedSq] = base.EmptyPiece
					}
					ed.selectedSq = -1
				}
			}
			ed.fen = convfen.ConvertBoardToFEN(ed.board)
		}
	}

	// animate messagebox if open
	if ed.msg.Open || ed.msg.Animating {
		ed.msg.AnimateMessage()
	}

	return SceneNotChanged, nil
}

func (ed *GUIEditDrawer) Draw(ctx *ghelper.GUIGameContext, screen *ebiten.Image) {
	// background
	screen.Fill(ctx.Theme.Bg)

	// title
	text.Draw(screen, ctx.AssetsWorker.Lang().T("editor.title"), ctx.AssetsWorker.Fonts().Bold, 40, 40, ctx.Theme.MenuText)
	text.Draw(screen, fmt.Sprintf("FEN: %v", ed.fen), ctx.AssetsWorker.Fonts().Pixel, ed.boardX+20, ed.boardY-10, ctx.Theme.MenuText)
	text.Draw(screen, ctx.AssetsWorker.Lang().T("editor.clipboard"), ctx.AssetsWorker.Fonts().PixelLow, ed.buttons[ed.btnPaste].X+35, ed.buttons[ed.btnPaste].Y-10, ctx.Theme.MenuText)

	if ed.borderImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(ed.boardX-4), float64(ed.boardY-4))
		screen.DrawImage(ed.borderImg, op)
	}

	// squares
	for rank := 0; rank < 8; rank++ {
		for file := 0; file < 8; file++ {
			sx := ed.boardX + file*ed.sqSize
			sy := ed.boardY + rank*ed.sqSize
			var img *ebiten.Image
			if ((file + rank) & 1) == 0 {
				img = ed.sqLightImg
			} else {
				img = ed.sqDarkImg
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(sx), float64(sy))
			screen.DrawImage(img, op)
		}
	}

	// pieces from mailbox
	for idx := 0; idx < 64; idx++ {
		p := base.GetPieceAt(&ed.board.Mailbox, base.ConvIndexToPoint(idx))
		if p == base.EmptyPiece {
			continue
		}
		px, py := ed.indexToScreenXY(idx)
		// skip if selected for move? (we still draw)
		img := ed.scaledPieces[p]
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(img, op)
		}
	}

	// draw selection highlight (move mode)
	if ed.selectedSq >= 0 {
		sx, sy := ed.indexToScreenXY(ed.selectedSq)
		ghelper.EbitenutilDrawRectStroke(screen, float64(sx)+2, float64(sy)+2, float64(ed.sqSize)-4, float64(ed.sqSize)-4, 2, ctx.Theme.Accent)
	}

	// preview piece under cursor if any
	mx, my := ebiten.CursorPosition()
	if ed.previewPiece != base.EmptyPiece && inBoard(mx, my, ed.boardX, ed.boardY, ed.sqSize) {
		iwImg := ed.scaledPieces[ed.previewPiece]
		if iwImg != nil {
			op := &ebiten.DrawImageOptions{}
			// draw centered in square under cursor
			sq := pixelToSquare(mx, my, ed.boardX, ed.boardY, ed.sqSize, false)
			px, py := ed.indexToScreenXY(sq)
			// draw translucent preview
			var cm ebiten.ColorM
			cm.Scale(1, 1, 1, 0.85)
			op.ColorM = cm
			op.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(iwImg, op)
		}
	}

	// draw palettes under board
	// white palette
	wx, wy := ed.paletteWhiteRect()
	for i, piece := range whitePaletteOrder() {
		sy := wy + i*(ed.sqSize+8)
		img := ed.scaledPieces[piece]
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(wx), float64(sy))
			screen.DrawImage(img, op)
		}
		// highlight selected
		if ed.previewPiece == piece && ed.previewIsWhite {
			ghelper.EbitenutilDrawRectStroke(screen, float64(wx)+1, float64(sy)+1, float64(ed.sqSize)-2, float64(ed.sqSize)-2, 3, ctx.Theme.Accent)
		}
	}
	// black palette
	bx, by := ed.paletteBlackRect()
	for i, piece := range blackPaletteOrder() {
		sy := by + i*(ed.sqSize+8)
		img := ed.scaledPieces[piece]
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(bx), float64(sy))
			screen.DrawImage(img, op)
		}
		if ed.previewPiece == piece && !ed.previewIsWhite {
			ghelper.EbitenutilDrawRectStroke(screen, float64(bx)+1, float64(sy)+1, float64(ed.sqSize)-2, float64(ed.sqSize)-2, 3, ctx.Theme.Accent)
		}
	}

	// draw mode buttons (accent them if active)
	for i, b := range ed.buttons {
		// accent mode buttons
		if i == ed.btnTraining {
			if ctx.Config.Training {
				b.Label = ctx.AssetsWorker.Lang().T("editor.training.on")
				b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
			} else {
				b.Label = ctx.AssetsWorker.Lang().T("editor.training.off")
				b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
			}
		} else if (i == ed.btnPlace && ed.modePlace) ||
			(i == ed.btnMove && ed.modeMove) ||
			(i == ed.btnDelete && ed.modeDelete) ||
			(i == ed.btnWhiteMove && ed.board.WhiteToMove) ||
			(i == ed.btnBlackMove && !ed.board.WhiteToMove) ||
			(i == ed.btnCastingWK && ed.board.Casting.WK) ||
			(i == ed.btnCastingWQ && ed.board.Casting.WQ) ||
			(i == ed.btnCastingBK && ed.board.Casting.BK) ||
			(i == ed.btnCastingBQ && ed.board.Casting.BQ) {
			b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.Accent, ctx.Theme.ButtonStroke, 3)
		} else {
			// reset normal images for non-mode buttons (or they remain cached elsewhere)
			// We re-render using theme; it's fine for a small number of buttons
			b.Image = ghelper.RenderRoundedRect(b.W, b.H, 12, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)
		}
		b.DrawAnimated(screen, ctx.AssetsWorker.Fonts().PixelLow, ctx.Theme)
	}

	// message box
	// if ed.msg.Open || ed.msg.Animating {
	// 	DrawModal(ctx, ed.msg.Scale, ed.msg.Text, screen)
	// }
	ed.msg.Draw(ctx, screen)

	// debug
	if ctx.Config.Debug {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("Editor TPS: %0.2f", ebiten.ActualTPS()))
	}
}

// ---------- helpers ----------

func (ed *GUIEditDrawer) prepareCache(ctx *ghelper.GUIGameContext) {
	ed.sqLightImg = ebiten.NewImage(ed.sqSize, ed.sqSize)
	ed.sqLightImg.Fill(ctx.Theme.SquareLight)
	ed.sqDarkImg = ebiten.NewImage(ed.sqSize, ed.sqSize)
	ed.sqDarkImg.Fill(ctx.Theme.SquareDark)
	ed.borderImg = ghelper.RenderRoundedRect(ed.boardSize+8, ed.boardSize+8, 6, ctx.Theme.ButtonFill, ctx.Theme.ButtonStroke, 3)

	ed.scaledPieces = make(map[base.Piece]*ebiten.Image)
	keys := []base.Piece{
		base.WKing, base.WQueen, base.WRook, base.WBishop, base.WKnight, base.WPawn,
		base.BKing, base.BQueen, base.BRook, base.BBishop, base.BKnight, base.BPawn,
	}
	for _, k := range keys {
		src := ctx.AssetsWorker.Piece(k)
		if src == nil {
			continue
		}
		dst := ebiten.NewImage(ed.sqSize, ed.sqSize)
		iw, ih := src.Size()
		if iw > 0 && ih > 0 {
			s := math.Min(float64(ed.sqSize)/float64(iw), float64(ed.sqSize)/float64(ih))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(s, s)
			tw := float64(iw) * s
			th := float64(ih) * s
			op.GeoM.Translate((float64(ed.sqSize)-tw)/2.0, (float64(ed.sqSize)-th)/2.0)
			op.Filter = ebiten.FilterLinear
			dst.DrawImage(src, op)
			ed.scaledPieces[k] = dst
		} else {
			ed.scaledPieces[k] = src
		}
	}
}

func (ed *GUIEditDrawer) indexToScreenXY(idx int) (x, y int) {
	f, r := indexToFileRank(idx)
	file := f
	rank := 7 - r
	return ed.boardX + file*ed.sqSize, ed.boardY + rank*ed.sqSize
}

// palette placement helpers
func (ed *GUIEditDrawer) paletteWhiteRect() (x, y int) {
	x = ed.boardX - 8 - ed.sqSize*2
	y = ed.boardY - 8 // slightly lower to make room for mode buttons
	return x, y
}
func (ed *GUIEditDrawer) paletteBlackRect() (x, y int) {
	x = ed.boardX - 8 - ed.sqSize
	y = ed.boardY - 8
	return x, y
}

func whitePaletteOrder() []base.Piece {
	return []base.Piece{base.WKing, base.WQueen, base.WRook, base.WBishop, base.WKnight, base.WPawn}
}
func blackPaletteOrder() []base.Piece {
	return []base.Piece{base.BKing, base.BQueen, base.BRook, base.BBishop, base.BKnight, base.BPawn}
}
