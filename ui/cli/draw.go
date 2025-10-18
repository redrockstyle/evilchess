package cli

import (
	"evilchess/src/base"
	"fmt"
)

// func EnableANSI() {
// 	if runtime.GOOS != "windows" {
// 		return
// 	}

// 	stdout := windows.Handle(os.Stdout.Fd())
// 	var mode uint32
// 	if err := windows.GetConsoleMode(stdout, &mode); err != nil {
// 		return
// 	}
// 	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
// 	_ = windows.SetConsoleMode(stdout, mode)
// }

func PrintMailbox(m base.Mailbox) {
	// ANSI-code
	const (
		reset   = "\033[0m"
		lightBg = "\033[47m"
		darkBg  = "\033[100m"
		whiteF  = "\033[97m"
		blackF  = "\033[30m"
		dimF    = "\033[90m"
	)

	// Piece -> unicode glyph
	pieceGlyph := func(p base.Piece) string {
		switch p {
		case base.WKing:
			return "♔"
		case base.WQueen:
			return "♕"
		case base.WRook:
			return "♖"
		case base.WBishop:
			return "♗"
		case base.WKnight:
			return "♘"
		case base.WPawn:
			return "♙"
		case base.BKing:
			return "♚"
		case base.BQueen:
			return "♛"
		case base.BRook:
			return "♜"
		case base.BBishop:
			return "♝"
		case base.BKnight:
			return "♞"
		case base.BPawn:
			return "♟"
		case base.EmptyPiece:
			return " "
		default:
			return "?"
		}
	}

	isWhite := func(p base.Piece) bool {
		return p == base.WKing || p == base.WQueen || p == base.WRook || p == base.WBishop || p == base.WKnight || p == base.WPawn
	}
	isBlack := func(p base.Piece) bool {
		return p == base.BKing || p == base.BQueen || p == base.BRook || p == base.BBishop || p == base.BKnight || p == base.BPawn
	}

	fmt.Println()
	fmt.Println("   a  b  c  d  e  f  g  h")
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := 0; file < 8; file++ {
			idx := rank*8 + file
			p := m[idx]
			g := pieceGlyph(p)

			lightSquare := (rank+file)%2 == 0

			var bg, fg string
			if lightSquare {
				bg = lightBg
				if g == " " {
					fg = dimF
				} else {
					fg = blackF
				}
			} else {
				bg = darkBg
				if isWhite(p) {
					fg = whiteF
				} else if isBlack(p) {
					fg = blackF
				} else {
					fg = dimF
				}
			}

			fmt.Printf("%s%s %s %s", bg, fg, g, reset)
		}
		fmt.Printf(" %d\n", rank+1)
	}
	fmt.Println("   a  b  c  d  e  f  g  h")
	fmt.Println()
}
