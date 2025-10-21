package convfen

import (
	"errors"
	"evilchess/src/chesslib/base"
	"fmt"
	"strconv"
	"strings"
)

func ConvertBoardToFEN(board base.Board) string {
	// pieces
	var b strings.Builder
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			pc := board.Mailbox[rank*8+file]
			if pc == base.EmptyPiece {
				empty++
			} else {
				if empty > 0 {
					b.WriteString(strconv.Itoa(empty))
					empty = 0
				}
				if r := base.ConvertRuneFromPiece(pc); r != 0 {
					b.WriteRune(r)
				}
			}
		}
		if empty > 0 {
			b.WriteString(strconv.Itoa(empty))
		}
		if rank > 0 {
			b.WriteByte('/')
		}
	}

	// side to move
	if board.WhiteToMove {
		b.WriteString(" w ")
	} else {
		b.WriteString(" b ")
	}

	// casting
	cast := ""
	if board.Casting.WK {
		cast += "K"
	}
	if board.Casting.WQ {
		cast += "Q"
	}
	if board.Casting.BK {
		cast += "k"
	}
	if board.Casting.BQ {
		cast += "q"
	}
	if cast == "" {
		cast = "-"
	}
	b.WriteString(cast + " ")

	// en-passant
	if board.EnPassant == -1 {
		b.WriteString("- ")
	} else {
		if str, err := base.AlgebraicFromSquare(board.EnPassant); err == nil {
			b.WriteString(str + " ")
		}
	}

	// moves
	b.WriteString(strconv.Itoa(board.Halfmove) + " ")
	b.WriteString(strconv.Itoa(board.Fullmove))

	return b.String()
}

func ConvertFENToBoard(fen string) (*base.Board, error) {
	board := &base.Board{}

	parts := strings.Fields(fen)
	if len(parts) < 4 {
		return nil, fmt.Errorf("must be < 4 parts, but there are %d", len(parts))
	}

	ranks := strings.Split(parts[0], "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("must be != 8 rows, but there are %d", len(ranks))
	}

	// pieces
	var err error
	for r := 0; r < 8; r++ {
		row := ranks[r]
		count := 0
		for _, ch := range row {
			if count == 8 {
				return nil, fmt.Errorf("row overflow: most be > 8, but count is %d", count)
			}
			if ch >= '1' && ch <= '8' {
				empty, _ := strconv.Atoi(string(ch))
				if ((empty + count) > 8) || (count > 0 && empty == 8) {
					return nil, fmt.Errorf("row overflow: most be > 8, but count is %d", count)
				}
				for i := 0; i < empty; i++ {
					board.Mailbox[(7-r)*8+count] = base.EmptyPiece
					count++
				}
			} else {
				board.Mailbox[(7-r)*8+count] = base.ConvertPieceFromRune(ch)
				if board.Mailbox[(7-r)*8+count] == base.InvalidPiece {
					return nil, errors.New("error convert piece")
				}
				count++
			}
		}
		if count != 8 {
			return nil, fmt.Errorf("most be != 8 fields in row[%d], but there are %d", r+1, len(row))
		}
	}

	// side to move
	board.WhiteToMove = parts[1] == "w"

	// casting
	cast := parts[2]
	if cast == "-" {
		board.Casting.WK, board.Casting.WQ, board.Casting.BK, board.Casting.BQ = false, false, false, false
	} else {
		board.Casting.WK = strings.Contains(cast, "K")
		board.Casting.WQ = strings.Contains(cast, "Q")
		board.Casting.BK = strings.Contains(cast, "k")
		board.Casting.BQ = strings.Contains(cast, "q")
	}

	// en passant
	ep := parts[3]
	if ep == "-" {
		board.EnPassant = -1
	} else {
		if board.EnPassant, err = base.SquareFromAlgebraic(ep); err != nil {
			return nil, fmt.Errorf("error parsing en-parrant: %s", ep)
		}
	}

	// halfmove
	if len(parts) >= 5 {
		if board.Halfmove, err = strconv.Atoi(parts[4]); err != nil {
			return nil, fmt.Errorf("incorrect halfmove %s: %v", parts[5], err)
		}
	}

	// fullmove
	if len(parts) >= 6 {
		if board.Fullmove, err = strconv.Atoi(parts[5]); err != nil {
			return nil, fmt.Errorf("incorrect fullmove %s: %v", parts[6], err)
		}
	}

	return board, nil
}
