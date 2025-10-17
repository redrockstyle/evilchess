package rules

import (
	"evilchess/src/base"
	"evilchess/src/logic/rules/moves"
)

// checks legal move for current board
func IsLegalMove(b *base.Board, mv base.Move) bool {
	legal := moves.GenerateLegalMoves(b)
	for _, m := range legal {
		if m.From == mv.From && m.To == mv.To && m.Piece == mv.Piece {
			return true
		}
	}
	return false
}

func IsInCheck(b *base.Board, white bool) bool {
	kingIdx := moves.FindKing(&b.Mailbox, white)
	if kingIdx < 0 {
		// ??? king not found
		return false
	}
	return moves.IsSquareAttacked(b, kingIdx, !white)
}

// return status: Check, Checkmate, Stalemate or Pass
func GameStatusOf(b *base.Board) base.GameStatus {
	if b == nil {
		return base.InvalidGame
	}
	if IsDrawPosition(b) {
		return base.Draw
	}
	inCheck := IsInCheck(b, b.WhiteToMove)
	legal := moves.GenerateLegalMoves(b)
	if len(legal) == 0 {
		if inCheck {
			return base.Checkmate
		}
		return base.Stalemate
	}
	if inCheck {
		return base.Check
	}
	return base.Pass
}

// srtict check draw
func IsDrawPosition(b *base.Board) bool {
	// fifty-move rule: 100 halfmove == 50 move
	// if board.Halfmove >= 100 {
	// 	return true
	// }

	// insufficient material checks
	var (
		wpawns, bpawns                 int
		wrooks, brooks                 int
		wqueens, bqueens               int
		wknights, bknights             int
		wbishops, bbishops             int
		wbishopSqColor, bbishopSqColor []int // 0/1 parity lists
	)

	for idx := 0; idx < 64; idx++ {
		pc := b.Mailbox[idx]
		switch pc {
		case base.WPawn:
			wpawns++
		case base.BPawn:
			bpawns++
		case base.WRook:
			wrooks++
		case base.BRook:
			brooks++
		case base.WQueen:
			wqueens++
		case base.BQueen:
			bqueens++
		case base.WKnight:
			wknights++
		case base.BKnight:
			bknights++
		case base.WBishop:
			wbishops++
			file := idx % 8
			rank := idx / 8
			bcolor := (file + rank) & 1
			wbishopSqColor = append(wbishopSqColor, bcolor)
		case base.BBishop:
			bbishops++
			file := idx % 8
			rank := idx / 8
			bcolor := (file + rank) & 1
			bbishopSqColor = append(bbishopSqColor, bcolor)
		}
	}

	if wpawns+bpawns > 0 || wrooks+brooks > 0 || wqueens+bqueens > 0 {
		// ???
	} else {
		totalKnights := wknights + bknights
		totalBishops := wbishops + bbishops
		totalMinor := totalKnights + totalBishops

		if totalMinor == 0 {
			return true
		}

		// 1 minor piece total -> K+N vs K or K+B vs K => draw
		if totalMinor == 1 {
			return true
		}

		// K+N+N vs K
		// if totalKnights == 2 && totalBishops == 0 {
		// 	return true
		// }

		if wbishops == 1 && bbishops == 1 && totalKnights == 0 {
			if len(wbishopSqColor) > 0 && len(bbishopSqColor) > 0 {
				if wbishopSqColor[0] == bbishopSqColor[0] {
					return true
				}
			}
		}
	}
	return false
}
