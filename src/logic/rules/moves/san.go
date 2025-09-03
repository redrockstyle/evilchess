package moves

import (
	"errors"
	"evilchess/src/base"
	"fmt"
	"strings"
)

// Standard Algebraic Notation

// var reSAN = regexp.MustCompile(`^([KQRBN])?([a-h1-8]{0,2})(x?)([a-h][1-8])(=([QRBN]))?([+#])?$`)

// SAN->Move converter (strip all except piece/file/rank)
func SANToMove(b *base.Board, san string) (base.Move, error) {
	var pc base.Piece
	var from base.Point
	var to base.Point
	var err error
	var halfcmp bool
	var allcheck bool
	var halffrom bool

	var tsan string
	if tsan = strings.Map(func(r rune) rune {
		if r == '+' || r == '#' || r == '!' || r == '?' || r == ' ' || r == 'x' || r == 'X' || r == '=' {
			return -1
		}
		return r
	}, san); tsan == "" {
		return base.Move{}, fmt.Errorf("empty SAN")
	}

	// short castling
	if upper := strings.ToUpper(tsan); strings.HasPrefix(upper, "O-O") || strings.HasPrefix(upper, "0-0") {
		from.W = 4
		// long castling
		if strings.HasPrefix(upper, "O-O-O") || strings.HasPrefix(upper, "0-0-0") {
			to.W = 2
		} else {
			to.W = 6
		}
		if b.WhiteToMove {
			to.H = 0
			from.H = 0
			pc = base.WKing
		} else {
			to.H = 7
			from.H = 7
			pc = base.BKing
		}
	} else {
		// if first == base.InvalidPiece then mb Pawn
		first := base.ConvertWPieceFromRune(rune(tsan[0]))
		if first != base.InvalidPiece {
			if !b.WhiteToMove {
				first = base.SwapColorPiece(first)
			}
			tsan = tsan[1:]
		}

		// if last != base.InvalidPiece then mb Pawn to up to a new piece (Q,R,B,N)
		last := base.ConvertWPieceFromRune(rune(tsan[len(tsan)-1]))
		if last != base.InvalidPiece {
			if !b.WhiteToMove {
				last = base.SwapColorPiece(last)
			}
			tsan = tsan[:len(tsan)-1]
		}

		var subtsan string
		if parts := func(s string) []string {
			var parts []string
			var current string
			for i, r := range s {
				if r >= 'a' && r <= 'h' {
					if len(current) > 0 {
						parts = append(parts, current)
					}
					current = string(r)
				} else {
					current += string(r)
				}
				if i == len(s)-1 && len(current) > 0 {
					parts = append(parts, current)
				}
			}
			return parts
		}(tsan); len(parts) == 2 {
			subtsan = parts[0]
			tsan = parts[1]
		}
		// else {
		// 	return base.Move{}, errors.New("parts overflow")
		// }

		getpoint := func(s string) (base.Point, base.Piece, error) {
			var index int
			if index, err = base.SquareFromAlgebraic(s); err != nil {
				return base.Point{}, base.InvalidPiece, fmt.Errorf("invalid index: %v", err)
			}
			if last != base.InvalidPiece && !(last == base.WPawn || last == base.BPawn) &&
				(first == base.WPawn || first == base.BPawn || first == base.InvalidPiece) {
				pc = last // up to a new piece
			} else if first == base.InvalidPiece {
				if b.WhiteToMove {
					pc = base.WPawn
				} else {
					pc = base.BPawn
				}
			} else {
				pc = first
			}
			return base.ConvIndexToPoint(index), pc, nil

		}

		if len(tsan) == 1 {
			if len(subtsan) == 2 {
				if from, pc, err = getpoint(tsan); err != nil {
					return base.Move{}, err
				}
				allcheck = true
			} else if len(subtsan) == 1 {
				// super short format for only pawn
				halfcmp = true

				from.W = uint8(subtsan[0] - 'a')
				to.W = uint8(tsan[0] - 'a')
				if last != base.InvalidPiece {
					pc = last
				} else if b.WhiteToMove {
					pc = base.WPawn
				} else {
					pc = base.BPawn
				}
			} else {
				return base.Move{}, errors.New("invalid short move")
			}
		} else {
			if to, pc, err = getpoint(tsan); err != nil {
				return base.Move{}, err
			}
			{ // substan for clarifying move
				if len(subtsan) == 2 {
					if from, pc, err = getpoint(tsan); err != nil {
						return base.Move{}, err
					}
					allcheck = true
				} else if len(subtsan) == 1 {
					halffrom = true
					from.W = uint8(subtsan[0] - 'a')
				}
			}
		}
	}
	moves := GenerateLegalMoves(b)

	var matched []base.Move
	for _, move := range moves {
		if halfcmp {
			if move.To.W == to.W && move.From.W == from.W && move.Piece == pc {
				matched = append(matched, move)
			}
		} else if halffrom {
			if move.From.W == from.W && move.To == to && move.Piece == pc {
				matched = append(matched, move)
			}
		} else if allcheck {
			if move.From == from && move.To == to && move.Piece == pc {
				matched = append(matched, move)
			}
		} else if move.To == to && move.Piece == pc {
			matched = append(matched, move)
		}
	}

	if len(matched) > 1 {
		return base.Move{}, errors.New("multiple matched move")
	} else if len(matched) == 0 {
		return base.Move{}, errors.New("move is not found")
	}

	return matched[0], nil
}

func MoveToShortSAN(mv base.Move) string {
	// castling
	if mv.From.W == 4 && mv.To.W == 6 {
		return "O-O"
	}
	if mv.From.W == 4 && mv.To.W == 2 {
		return "O-O-O"
	}

	to, _ := base.AlgebraicFromSquare(base.ConvPointToIndex(mv.To))
	if mv.Piece == base.WPawn || mv.Piece == base.BPawn {
		return fmt.Sprintf("%s", to)
	}
	return fmt.Sprintf("%c%s", base.ConvertUpperRuneFromPiece(mv.Piece), to)
}
