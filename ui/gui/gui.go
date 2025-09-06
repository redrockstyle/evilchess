package gui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

const (
	SquareSize = 72
	BoardSize  = 8
	Margin     = 8
)

type GameState struct {
	board        [64]rune   // piece runes: 'P','N','B','R','Q','K' (upper white), lowercase black, '.' empty
	history      [][64]rune // snapshots for undo
	moves        []string   // simple move notation list (uCI-like: e2-e4)
	headers      map[string]string
	selected     int // -1 = none, else index 0..63
	lastFrom     int
	lastTo       int
	windowWidth  int
	windowHeight int
}

func NewGameFromFEN(fen string) *GameState {
	g := &GameState{
		selected: -1,
		headers:  map[string]string{"Event": "Demo", "White": "White", "Black": "Black"},
	}
	// default empty
	for i := range g.board {
		g.board[i] = '.'
	}
	// parse piece placement only (first field)
	fields := strings.Fields(fen)
	if len(fields) == 0 { // fallback to standard start
		fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
		fields = strings.Fields(fen)
	}
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		// invalid, fill empty
		return g
	}
	for r := 0; r < 8; r++ {
		row := ranks[r]
		file := 0
		for _, ch := range row {
			if ch >= '1' && ch <= '8' {
				// empty squares
				empties := int(ch - '0')
				for k := 0; k < empties; k++ {
					idx := r*8 + file
					g.board[idx] = '.'
					file++
				}
			} else {
				idx := r*8 + file
				g.board[idx] = ch
				file++
			}
		}
	}
	// push initial snapshot
	g.pushSnapshot()
	return g
}

func (g *GameState) pushSnapshot() {
	var snap [64]rune
	copy(snap[:], g.board[:])
	g.history = append(g.history, snap)
}

func (g *GameState) undo() {
	if len(g.history) <= 1 {
		return
	}
	// drop last snapshot and restore previous
	g.history = g.history[:len(g.history)-1]
	last := g.history[len(g.history)-1]
	copy(g.board[:], last[:])
	// pop move
	if len(g.moves) > 0 {
		g.moves = g.moves[:len(g.moves)-1]
	}
	// reset lastFrom/To
	g.lastFrom = -1
	g.lastTo = -1
	g.selected = -1
}

func (g *GameState) algebraicFromIndex(idx int) string {
	if idx < 0 || idx >= 64 {
		return "??"
	}
	file := idx % 8
	rank := 8 - (idx / 8) // ranks from 8 to 1 (FEN uses rank 8 at top)
	return fmt.Sprintf("%c%d", 'a'+file, rank)
}

func (g *GameState) handleClick(x, y int) {
	boardLeft := Margin
	boardTop := Margin + 60 // leave space for headers at top
	bx := x - boardLeft
	by := y - boardTop
	if bx < 0 || by < 0 {
		return
	}
	col := bx / SquareSize
	row := by / SquareSize
	if col < 0 || col >= 8 || row < 0 || row >= 8 {
		return
	}
	idx := row*8 + col
	// if nothing selected and clicked a piece, select
	if g.selected == -1 {
		if g.board[idx] != '.' {
			g.selected = idx
		}
		return
	}
	// if selected and clicked same square -> deselect
	if g.selected == idx {
		g.selected = -1
		return
	}
	// perform move (no validation)
	g.lastFrom = g.selected
	g.lastTo = idx
	g.board[idx] = g.board[g.selected]
	g.board[g.selected] = '.'
	// add move string (simple uci-like)
	mv := fmt.Sprintf("%s-%s", g.algebraicFromIndex(g.lastFrom), g.algebraicFromIndex(g.lastTo))
	g.moves = append(g.moves, mv)
	g.pushSnapshot()
	g.selected = -1
}

type EbitenApp struct {
	game *GameState
}

func NewApp(g *GameState) *EbitenApp {
	return &EbitenApp{game: g}
}

func (a *EbitenApp) Update() error {
	// handle mouse clicks
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		_, _ = ebiten.CursorPosition()
		// to avoid rapid multiple clicks, check left-button just-pressed: simple edge detector via frame count would be better,
		// but for demo we'll react to mouse release instead. So do nothing here.
	}
	// key handling: undo with U
	if ebiten.IsKeyPressed(ebiten.KeyU) {
		// naive debounce: only trigger once per key press would be better
		a.game.undo()
	}
	// handle mouse release: detect button just released by checking mouse button state and last frame - simpler: check just previous position?
	// For simplicity, we handle clicks via ebitenutil.IsMouseButton just-pressed style:
	// ebiten does not have IsMouseButtonJustPressed; we emulate it by using InputChars? keep simple: use ebiten.IsMouseButtonPressed and track previous state
	return nil
}

var prevMouseDown = false

func (a *EbitenApp) Layout(outsideWidth, outsideHeight int) (int, int) {
	// total width = board + right panel
	w := Margin*2 + SquareSize*BoardSize + 320
	h := Margin*2 + SquareSize*BoardSize + 80
	a.game.windowWidth = w
	a.game.windowHeight = h
	return w, h
}

func (a *EbitenApp) Draw(screen *ebiten.Image) {
	// clear bg
	screen.Fill(color.RGBA{0xf0, 0xf0, 0xf0, 0xff})

	boardLeft := Margin
	boardTop := Margin + 60

	// draw headers (PGN headers) at top
	hx := boardLeft
	hy := Margin
	text.Draw(screen, fmt.Sprintf("Event: %s", a.game.headers["Event"]), basicfont.Face7x13, hx, hy+12, color.Black)
	text.Draw(screen, fmt.Sprintf("White: %s", a.game.headers["White"]), basicfont.Face7x13, hx+220, hy+12, color.Black)
	text.Draw(screen, fmt.Sprintf("Black: %s", a.game.headers["Black"]), basicfont.Face7x13, hx+420, hy+12, color.Black)

	// draw board squares
	for r := 0; r < 8; r++ {
		for c := 0; c < 8; c++ {
			x := boardLeft + c*SquareSize
			y := boardTop + r*SquareSize
			light := ((r + c) % 2) == 0
			var sqColor color.RGBA
			if light {
				sqColor = color.RGBA{0xee, 0xdd, 0xc8, 0xff}
			} else {
				sqColor = color.RGBA{0x99, 0x77, 0x55, 0xff}
			}
			// highlight last move background
			idx := r*8 + c
			if idx == a.game.lastFrom || idx == a.game.lastTo {
				// overlay highlight color
				// blend simple: use a different color
				sqColor = color.RGBA{0xa8, 0xe6, 0xa8, 0xff}
			}
			ebitenutil.DrawRect(screen, float64(x), float64(y), SquareSize, SquareSize, sqColor)

			// selection border
			if a.game.selected == idx {
				ebitenutil.DrawRect(screen, float64(x), float64(y), SquareSize, 4, color.RGBA{0xff, 0xd7, 0, 0xff})              // top
				ebitenutil.DrawRect(screen, float64(x), float64(y+SquareSize-4), SquareSize, 4, color.RGBA{0xff, 0xd7, 0, 0xff}) // bottom
				ebitenutil.DrawRect(screen, float64(x), float64(y), 4, SquareSize, color.RGBA{0xff, 0xd7, 0, 0xff})              // left
				ebitenutil.DrawRect(screen, float64(x+SquareSize-4), float64(y), 4, SquareSize, color.RGBA{0xff, 0xd7, 0, 0xff}) // right
			}
			// draw coordinate label (bottom-left of square)
			label := fmt.Sprintf("%c%d", 'a'+c, 8-r)
			text.Draw(screen, label, basicfont.Face7x13, x+4, y+SquareSize-4, color.RGBA{0x22, 0x22, 0x22, 0x66})

			// draw piece
			p := a.game.board[idx]
			if p != '.' {
				// center text
				var col color.Color = color.Black
				if p >= 'A' && p <= 'Z' {
					col = color.White
					// draw dark outline circle for white pieces
					ebitenutil.DrawRect(screen, float64(x+8), float64(y+8), SquareSize-16, SquareSize-16, color.RGBA{0x33, 0x33, 0x33, 0x60})
				} else {
					col = color.Black
				}
				// display letter (map piece to letter)
				letter := rune(' ')
				switch p {
				case 'P', 'p':
					letter = 'P'
				case 'N', 'n':
					letter = 'N'
				case 'B', 'b':
					letter = 'B'
				case 'R', 'r':
					letter = 'R'
				case 'Q', 'q':
					letter = 'Q'
				case 'K', 'k':
					letter = 'K'
				default:
					letter = '?'
				}
				// text center approximate
				tx := x + SquareSize/2 - 6
				ty := y + SquareSize/2 + 6
				text.Draw(screen, string(letter), basicfont.Face7x13, tx, ty, col)
			}
		}
	}

	// draw right panel: moves & controls
	panelX := boardLeft + SquareSize*BoardSize + 16
	px := panelX
	py := boardTop
	text.Draw(screen, "Moves:", basicfont.Face7x13, px, py+12, color.Black)
	// list moves
	for i, mv := range a.game.moves {
		text.Draw(screen, fmt.Sprintf("%d. %s", i+1, mv), basicfont.Face7x13, px, py+30+i*14, color.Black)
		if py+30+i*14 > a.game.windowHeight-40 {
			break
		}
	}
	// controls hint
	text.Draw(screen, "Left click: select/move", basicfont.Face7x13, px, a.game.windowHeight-60, color.Black)
	text.Draw(screen, "U: undo", basicfont.Face7x13, px, a.game.windowHeight-44, color.Black)

	// handle mouse click (simple edge detector)
	mouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if mouseDown && !prevMouseDown {
		// just pressed -> compute which square and handle
		x, y := ebiten.CursorPosition()
		a.game.handleClick(x, y)
	}
	prevMouseDown = mouseDown
}
