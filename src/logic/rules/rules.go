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
