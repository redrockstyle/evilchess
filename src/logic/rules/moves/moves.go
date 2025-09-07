package moves

import (
	"evilchess/src/base"
	"fmt"
)

// find king and return his index
func FindKing(mb *base.Mailbox, white bool) int {
	var target base.Piece
	if white {
		target = base.WKing
	} else {
		target = base.BKing
	}
	for i := 0; i < 64; i++ {
		if mb[i] == target {
			return i
		}
	}
	return -1
}

// checks if the field (index) is attacked or not
func IsSquareAttacked(b *base.Board, idx int, byWhite bool) bool {
	mb := &b.Mailbox
	sq := base.ConvIndexToPoint(idx)
	h := int(sq.H)
	w := int(sq.W)

	// for pawn
	if byWhite {
		for _, dw := range []int{-1, 1} {
			ht := h - 1
			wt := w + dw
			if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
				p := base.GetPieceAt(mb, base.Point{H: uint8(ht), W: uint8(wt)})
				if p == base.WPawn {
					return true
				}
			}
		}
	} else {
		for _, dw := range []int{-1, 1} {
			ht := h + 1
			wt := w + dw
			if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
				p := base.GetPieceAt(mb, base.Point{H: uint8(ht), W: uint8(wt)})
				if p == base.BPawn {
					return true
				}
			}
		}
	}

	// for knights
	nOffsets := [8][2]int{{2, 1}, {1, 2}, {-1, 2}, {-2, 1}, {-2, -1}, {-1, -2}, {1, -2}, {2, -1}}
	for _, o := range nOffsets {
		ht := h + o[0]
		wt := w + o[1]
		if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
			p := base.GetPieceAt(mb, base.Point{H: uint8(ht), W: uint8(wt)})
			if (byWhite && p == base.WKnight) || (!byWhite && p == base.BKnight) {
				return true
			}
		}
	}

	// for bishops/rooks/queens
	dirs := [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for di, d := range dirs {
		for step := 1; ; step++ {
			ht := h + d[0]*step
			wt := w + d[1]*step
			if ht < 0 || ht >= 8 || wt < 0 || wt >= 8 {
				break
			}
			p := base.GetPieceAt(mb, base.Point{H: uint8(ht), W: uint8(wt)})
			if p == base.EmptyPiece {
				continue
			}
			isWhitePiece := base.PieceIsWhite(p)
			if byWhite != isWhitePiece {
				break // if piece is not enemy
			}

			// rook or queen
			if di <= 3 {
				if (byWhite && (p == base.WRook || p == base.WQueen)) || (!byWhite && (p == base.BRook || p == base.BQueen)) {
					return true
				}
				break
			} else {
				if (byWhite && (p == base.WBishop || p == base.WQueen)) || (!byWhite && (p == base.BBishop || p == base.BQueen)) {
					return true
				}
				break
			}
		}
	}

	// for king (adjacent)
	for dh := -1; dh <= 1; dh++ {
		for dw := -1; dw <= 1; dw++ {
			if dh == 0 && dw == 0 {
				continue
			}
			ht := h + dh
			wt := w + dw
			if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
				p := base.GetPieceAt(mb, base.Point{H: uint8(ht), W: uint8(wt)})
				if (byWhite && p == base.WKing) || (!byWhite && p == base.BKing) {
					return true
				}
			}
		}
	}

	return false

}

func PsuedoLegalPawnMoves(b *base.Board, index int, out *[]base.Move) {
	mb := &b.Mailbox
	from := base.ConvIndexToPoint(index)
	p := base.GetPieceAt(mb, from)
	if p != base.WPawn && p != base.BPawn {
		return
	}
	white := base.PieceIsWhite(p)
	h := int(from.H)
	w := int(from.W)

	dir := 1
	startRank := 1
	promoRank := 7
	if !white {
		dir = -1
		startRank = 6
		promoRank = 0
	}

	promoMoves := func(m base.Move, white bool) {
		if white {
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.WQueen})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.WRook})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.WBishop})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.WKnight})
		} else {
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.BQueen})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.BRook})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.BBishop})
			*out = append(*out, base.Move{From: m.From, To: m.To, Piece: base.BKnight})
		}
	}

	// for pawn
	fh := h + dir
	if fh >= 0 && fh < 8 {
		to := base.Point{H: uint8(fh), W: uint8(w)}
		if base.GetPieceAt(mb, to) == base.EmptyPiece {
			m := base.Move{From: from, To: to, Piece: p}
			// promotion?
			if fh == promoRank {
				// append 4 promotion variants: Q,R,B,N
				promoMoves(m, white)
			} else {
				*out = append(*out, m)
			}
		}
	}
	// first move pawn
	if h == startRank {
		h2 := h + dir*2
		if h2 >= 0 && h2 < 8 {
			mid := base.Point{H: uint8(h + dir), W: uint8(w)}
			to := base.Point{H: uint8(h2), W: uint8(w)}
			if base.GetPieceAt(mb, mid) == base.EmptyPiece && base.GetPieceAt(mb, to) == base.EmptyPiece {
				*out = append(*out, base.Move{From: from, To: to, Piece: p})
			}
		}
	}
	// captures
	for _, dw := range []int{-1, 1} {
		ht := h + dir
		wt := w + dw
		if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
			to := base.Point{H: uint8(ht), W: uint8(wt)}
			q := base.GetPieceAt(mb, to)
			if q != base.EmptyPiece && ((white && base.PieceIsBlack(q)) || (!white && base.PieceIsWhite(q))) {
				m := base.Move{From: from, To: to, Piece: p}
				// promotion?
				if fh == promoRank {
					// append 4 promotion variants: Q,R,B,N
					promoMoves(m, white)
				} else {
					*out = append(*out, m)
				}
			}
		}
	}
	// if pawn can capture en-passant
	if b.EnPassant >= 0 {
		epPoint := base.ConvIndexToPoint(b.EnPassant)
		if epPoint.H == uint8(h+dir) && (int(epPoint.W) == w-1 || int(epPoint.W) == w+1) {
			*out = append(*out, base.Move{From: from, To: epPoint, Piece: p})
		}
	}
}

func PsuedoLegalKnightMoves(b *base.Board, fromIdx int, out *[]base.Move) {
	mb := &b.Mailbox
	from := base.ConvIndexToPoint(fromIdx)
	p := base.GetPieceAt(mb, from)
	if p != base.WKnight && p != base.BKnight {
		return
	}
	white := base.PieceIsWhite(p)
	h := int(from.H)
	w := int(from.W)

	off := [8][2]int{{2, 1}, {1, 2}, {-1, 2}, {-2, 1}, {-2, -1}, {-1, -2}, {1, -2}, {2, -1}}
	for _, o := range off {
		ht := h + o[0]
		wt := w + o[1]
		if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
			pt := base.Point{H: uint8(ht), W: uint8(wt)}
			q := base.GetPieceAt(mb, pt)
			if q == base.EmptyPiece || (white && base.PieceIsBlack(q)) || (!white && base.PieceIsWhite(q)) {
				*out = append(*out, base.Move{From: from, To: pt, Piece: p})
			}
		}
	}
}

func PsuedoLegalKingMoves(b *base.Board, fromIdx int, out *[]base.Move) {
	mb := &b.Mailbox
	from := base.ConvIndexToPoint(fromIdx)
	p := base.GetPieceAt(mb, from)
	if p != base.WKing && p != base.BKing {
		return
	}
	white := base.PieceIsWhite(p)
	h := int(from.H)
	w := int(from.W)

	for dh := -1; dh <= 1; dh++ {
		for dw := -1; dw <= 1; dw++ {
			if dh == 0 && dw == 0 {
				continue
			}
			ht := h + dh
			wt := w + dw
			if ht >= 0 && ht < 8 && wt >= 0 && wt < 8 {
				pt := base.Point{H: uint8(ht), W: uint8(wt)}
				q := base.GetPieceAt(mb, pt)
				if q == base.EmptyPiece || (white && base.PieceIsBlack(q)) || (!white && base.PieceIsWhite(q)) {
					*out = append(*out, base.Move{From: from, To: pt, Piece: p})
				}
			}
		}
	}

	// Castling (simplified): проверяем права и пустые клетки и отсутствие атак по маршруту
	if white {
		// white king assumed at e1: H=0,W=4
		if p == base.WKing && from.H == 0 && from.W == 4 {
			// king side
			if b.Casting.WK {
				// squares f1 (0,5) and g1 (0,6) empty and not attacked
				f := base.Point{H: 0, W: 5}
				g := base.Point{H: 0, W: 6}
				if base.GetPieceAt(mb, f) == base.EmptyPiece && base.GetPieceAt(mb, g) == base.EmptyPiece {
					if !IsSquareAttacked(b, base.ConvPointToIndex(from), false) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(f), false) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(g), false) {
						*out = append(*out, base.Move{From: from, To: g, Piece: p})
					}
				}
			}
			// queen side
			if b.Casting.WQ {
				b1 := base.Point{H: 0, W: 3}
				c1 := base.Point{H: 0, W: 2}
				d1 := base.Point{H: 0, W: 1}
				if base.GetPieceAt(mb, b1) == base.EmptyPiece &&
					base.GetPieceAt(mb, c1) == base.EmptyPiece && base.GetPieceAt(mb, d1) == base.EmptyPiece {
					if !IsSquareAttacked(b, base.ConvPointToIndex(from), false) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(b1), false) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(c1), false) {
						*out = append(*out, base.Move{From: from, To: c1, Piece: p})
					}
				}
			}
		}
	} else {
		// black
		if p == base.BKing && from.H == 7 && from.W == 4 {
			if b.Casting.BK {
				f := base.Point{H: 7, W: 5}
				g := base.Point{H: 7, W: 6}
				if base.GetPieceAt(mb, f) == base.EmptyPiece &&
					base.GetPieceAt(mb, g) == base.EmptyPiece {
					if !IsSquareAttacked(b, base.ConvPointToIndex(from), true) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(f), true) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(g), true) {
						*out = append(*out, base.Move{From: from, To: g, Piece: p})
					}
				}
			}
			if b.Casting.BQ {
				b1 := base.Point{H: 7, W: 3}
				c1 := base.Point{H: 7, W: 2}
				d1 := base.Point{H: 7, W: 1}
				if base.GetPieceAt(mb, b1) == base.EmptyPiece &&
					base.GetPieceAt(mb, c1) == base.EmptyPiece &&
					base.GetPieceAt(mb, d1) == base.EmptyPiece {
					if !IsSquareAttacked(b, base.ConvPointToIndex(from), true) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(b1), true) &&
						!IsSquareAttacked(b, base.ConvPointToIndex(c1), true) {
						*out = append(*out, base.Move{From: from, To: c1, Piece: p})
					}
				}
			}
		}
	}
}

// genSliding for bishops/rooks/queens
func genSliding(b *base.Board, fromIdx int, directions [][2]int, out *[]base.Move) {
	mb := &b.Mailbox
	from := base.ConvIndexToPoint(fromIdx)
	p := base.GetPieceAt(mb, from)
	white := base.PieceIsWhite(p)
	h := int(from.H)
	w := int(from.W)

	for _, d := range directions {
		for step := 1; ; step++ {
			ht := h + d[0]*step
			wt := w + d[1]*step
			if ht < 0 || ht >= 8 || wt < 0 || wt >= 8 {
				break
			}
			pt := base.Point{H: uint8(ht), W: uint8(wt)}
			q := base.GetPieceAt(mb, pt)
			if q == base.EmptyPiece {
				*out = append(*out, base.Move{From: from, To: pt, Piece: p})
				continue
			}
			// occupied
			if white && base.PieceIsWhite(q) || (!white && base.PieceIsBlack(q)) {
				break
			}
			// capture
			*out = append(*out, base.Move{From: from, To: pt, Piece: p})
			break
		}
	}
}

// genRook/Bishop/Queen wrapper
func PsuedoLegalRookMoves(b *base.Board, fromIdx int, out *[]base.Move) {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	genSliding(b, fromIdx, dirs, out)
}
func PsuedoLegalBishopMoves(b *base.Board, fromIdx int, out *[]base.Move) {
	dirs := [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	genSliding(b, fromIdx, dirs, out)
}
func PsuedoLegalQueenMoves(b *base.Board, fromIdx int, out *[]base.Move) {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	genSliding(b, fromIdx, dirs, out)
}

func PsuedoLegalMoves(b *base.Board) []base.Move {
	mb := &b.Mailbox
	moves := make([]base.Move, 0, 256)
	for i := 0; i < 64; i++ {
		p := base.GetPieceAt(mb, base.ConvIndexToPoint(i))
		if p == base.EmptyPiece || p == base.InvalidPiece {
			continue
		}
		if b.WhiteToMove && !base.PieceIsWhite(p) || (!b.WhiteToMove && !base.PieceIsBlack(p)) {
			continue
		}

		switch p {
		case base.WPawn, base.BPawn:
			PsuedoLegalPawnMoves(b, i, &moves)
		case base.WKnight, base.BKnight:
			PsuedoLegalKnightMoves(b, i, &moves)
		case base.WBishop, base.BBishop:
			PsuedoLegalBishopMoves(b, i, &moves)
		case base.WRook, base.BRook:
			PsuedoLegalRookMoves(b, i, &moves)
		case base.WQueen, base.BQueen:
			PsuedoLegalQueenMoves(b, i, &moves)
		case base.WKing, base.BKing:
			PsuedoLegalKingMoves(b, i, &moves)
		}
	}

	return moves
}

// apply move to current board
func ApplyMove(b *base.Board, mv base.Move) error {
	if b == nil {
		return fmt.Errorf("nil board")
	}
	if !base.IsValidPoint(mv.From) || !base.IsValidPoint(mv.To) {
		return fmt.Errorf("out of bounds move")
	}
	// fromIdx := base.ConvPointToIndex(mv.From)
	toIdx := base.ConvPointToIndex(mv.To)
	mb := &b.Mailbox
	pc := base.GetPieceAt(mb, mv.From)
	if pc == base.EmptyPiece || pc == base.InvalidPiece {
		return fmt.Errorf("no piece at from")
	}
	// Basic ownership check
	if b.WhiteToMove && !base.PieceIsWhite(pc) || (!b.WhiteToMove && !base.PieceIsBlack(pc)) {
		return fmt.Errorf("not side to move")
	}

	// handle en-passant capture
	isEnPassant := false
	if pc == base.WPawn || pc == base.BPawn {
		if b.EnPassant >= 0 && toIdx == b.EnPassant {
			// capture the pawn behind the en-passant target
			isEnPassant = true
		}
	}

	// move piece
	base.SetPieceAt(mb, mv.To, pc)
	base.SetPieceAt(mb, mv.From, base.EmptyPiece)

	// remove captured pawn on en-passant
	if isEnPassant {
		// captured pawn is one rank behind/forward depending on mover
		var capIdx int
		if base.PieceIsWhite(pc) {
			capIdx = toIdx - 8
		} else {
			capIdx = toIdx + 8
		}
		base.SetPieceAt(mb, base.ConvIndexToPoint(capIdx), base.EmptyPiece)
	}

	// handle promotion convention: if From had pawn but mv.Piece is queen/rook/... then set destination
	if (pc == base.WPawn || pc == base.BPawn) && mv.Piece != pc {
		// treat mv.Piece as promoted piece
		base.SetPieceAt(mb, mv.To, mv.Piece)
	}

	// update castling rights: if king moved, clear both castling rights for side
	if pc == base.WKing {
		b.Casting.WK = false
		b.Casting.WQ = false
		// if kingside castle (to g1) — move rook
		if mv.From.H == 0 && mv.From.W == 4 && mv.To.H == 0 && mv.To.W == 6 {
			// move rook from h1 to f1
			base.SetPieceAt(mb, base.Point{H: 0, W: 5}, base.WRook)
			base.SetPieceAt(mb, base.Point{H: 0, W: 7}, base.EmptyPiece)
		}
		// queen side
		if mv.From.H == 0 && mv.From.W == 4 && mv.To.H == 0 && mv.To.W == 2 {
			base.SetPieceAt(mb, base.Point{H: 0, W: 3}, base.WRook)
			base.SetPieceAt(mb, base.Point{H: 0, W: 0}, base.EmptyPiece)
		}
	}
	if pc == base.BKing {
		b.Casting.BK = false
		b.Casting.BQ = false
		if mv.From.H == 7 && mv.From.W == 4 && mv.To.H == 7 && mv.To.W == 6 {
			base.SetPieceAt(mb, base.Point{H: 7, W: 5}, base.BRook)
			base.SetPieceAt(mb, base.Point{H: 7, W: 7}, base.EmptyPiece)
		}
		if mv.From.H == 7 && mv.From.W == 4 && mv.To.H == 7 && mv.To.W == 2 {
			base.SetPieceAt(mb, base.Point{H: 7, W: 3}, base.BRook)
			base.SetPieceAt(mb, base.Point{H: 7, W: 0}, base.EmptyPiece)
		}
	}

	// if rook moved — clear corresponding castling right
	if pc == base.WRook {
		if mv.From.H == 0 && mv.From.W == 0 {
			b.Casting.WQ = false
		}
		if mv.From.H == 0 && mv.From.W == 7 {
			b.Casting.WK = false
		}
	}
	if pc == base.BRook {
		if mv.From.H == 7 && mv.From.W == 0 {
			b.Casting.BQ = false
		}
		if mv.From.H == 7 && mv.From.W == 7 {
			b.Casting.BK = false
		}
	}

	// update en-passant target: if pawn moved two squares, set target to square passed over
	b.EnPassant = -1
	if pc == base.WPawn || pc == base.BPawn {
		delta := int(mv.To.H) - int(mv.From.H)
		if delta == 2 {
			b.EnPassant = base.ConvPointToIndex(base.Point{H: mv.From.H + 1, W: mv.From.W})
		}
		if delta == -2 {
			b.EnPassant = base.ConvPointToIndex(base.Point{H: mv.From.H - 1, W: mv.From.W})
		}
	}

	// halfmove clock: reset on pawn move or capture
	if pc == base.WPawn || pc == base.BPawn || base.GetPieceAt(mb, base.ConvIndexToPoint(toIdx)) != base.EmptyPiece {
		b.Halfmove = 0
	} else {
		b.Halfmove++
	}

	// fullmove increment: after black move
	if !b.WhiteToMove {
		b.Fullmove++
	}

	// flip side
	b.WhiteToMove = !b.WhiteToMove

	return nil
}

func CloneBoard(b *base.Board) *base.Board {
	if b == nil {
		return nil
	}
	c := *b // shallow copy MailBox as value
	return &c
}

func GenerateLegalMoves(b *base.Board) []base.Move {
	pl := PsuedoLegalMoves(b)
	legal := make([]base.Move, 0, len(pl))
	for _, mv := range pl {
		cl := CloneBoard(b)
		if err := ApplyMove(cl, mv); err != nil {
			continue
		}

		// is check king
		var kingIdx int
		if !cl.WhiteToMove {
			kingIdx = FindKing(&cl.Mailbox, true)
			if kingIdx >= 0 && IsSquareAttacked(cl, kingIdx, false) {
				continue
			}
		} else {
			kingIdx = FindKing(&cl.Mailbox, false)
			if kingIdx >= 0 && IsSquareAttacked(cl, kingIdx, true) {
				continue
			}
		}
		legal = append(legal, mv)
	}
	return legal
}
