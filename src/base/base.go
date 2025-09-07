package base

import "fmt"

// Forsyth–Edwards Notation
const FEN_START_GAME string = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

type Piece uint8

const (
	WKing        Piece = 19
	WQueen       Piece = 18
	WRook        Piece = 15
	WBishop      Piece = 14
	WKnight      Piece = 13
	WPawn        Piece = 11
	BKing        Piece = 9
	BQueen       Piece = 8
	BRook        Piece = 5
	BBishop      Piece = 4
	BKnight      Piece = 3
	BPawn        Piece = 1
	EmptyPiece   Piece = 99
	InvalidPiece Piece = 0
)

type GameStatus uint8

const (
	Check       GameStatus = 10
	Checkmate   GameStatus = 11
	Stalemate   GameStatus = 12
	InvalidGame GameStatus = 88
	Pass        GameStatus = 99
)

func (gs GameStatus) String() string {
	switch gs {
	case Check:
		return "check"
	case Checkmate:
		return "checkmate"
	case Stalemate:
		return "stalemate"
	case Pass:
		return "pass"
	default:
		return "invalid"
	}
}

type Mailbox [64]Piece

type Point struct {
	H uint8
	W uint8
}

type Move struct {
	From  Point
	To    Point
	Piece Piece
}

type StatusCasting struct {
	WK bool
	WQ bool
	BK bool
	BQ bool
}

type Board struct {
	Mailbox     Mailbox
	Halfmove    int
	Fullmove    int
	WhiteToMove bool
	EnPassant   int
	Casting     StatusCasting
}

func ConvPointToIndex(p Point) int {
	return int(p.H)*8 + int(p.W)
}

func ConvIndexToPoint(i int) Point {
	return Point{H: uint8(i / 8), W: uint8(i % 8)}
}

func IsValidPoint(p Point) bool {
	return !(p.H > 7 || p.W > 7)
}

func PieceIsWhite(p Piece) bool {
	return p >= WPawn && p <= WKing
}

func PieceIsBlack(p Piece) bool {
	return p >= BPawn && p <= BKing
}

func SwapColorPiece(p Piece) Piece {
	switch p {
	case WKing:
		return BKing
	case WQueen:
		return BQueen
	case WRook:
		return BRook
	case WBishop:
		return BBishop
	case WKnight:
		return BKnight
	case WPawn:
		return BPawn
	case BKing:
		return WKing
	case BQueen:
		return WQueen
	case BRook:
		return WRook
	case BBishop:
		return WBishop
	case BKnight:
		return WKnight
	case BPawn:
		return WPawn
	default:
		return InvalidPiece
	}
}

func GetPieceAt(mb *Mailbox, p Point) Piece {
	if !IsValidPoint(p) || mb == nil {
		return InvalidPiece
	}
	return mb[ConvPointToIndex(p)]
}

func SetPieceAt(mb *Mailbox, p Point, pc Piece) {
	if !IsValidPoint(p) || mb == nil {
		return
	}
	mb[ConvPointToIndex(p)] = pc
}

func SquareFromAlgebraic(pos string) (int, error) {
	// 'a' ~ 'h' to number
	// '1' ~ '8' to 0-7
	if len(pos) != 2 || pos[0] < 'a' || pos[0] > 'h' || pos[1] < '1' || pos[1] > '8' {
		return -1, fmt.Errorf("invalid position")
	}
	return int(pos[1]-'1')*8 + int(pos[0]-'a'), nil
}

func AlgebraicFromSquare(index int) (string, error) {
	if index < 0 || index >= 64 {
		return "", fmt.Errorf("invalid square index")
	}
	return string([]rune{rune(index%8 + 'a'), rune(index/8 + '1')}), nil
}

func ConvertPieceFromRune(p rune) Piece {
	switch p {
	case 'P':
		return WPawn
	case 'R':
		return WRook
	case 'N':
		return WKnight
	case 'B':
		return WBishop
	case 'Q':
		return WQueen
	case 'K':
		return WKing
	case 'p':
		return BPawn
	case 'r':
		return BRook
	case 'n':
		return BKnight
	case 'b':
		return BBishop
	case 'q':
		return BQueen
	case 'k':
		return BKing
	default:
		return InvalidPiece
	}
}

func ConvertWPieceFromRune(p rune) Piece {
	switch p {
	case 'P':
		return WPawn
	case 'R':
		return WRook
	case 'N':
		return WKnight
	case 'B':
		return WBishop
	case 'Q':
		return WQueen
	case 'K':
		return WKing
	default:
		return InvalidPiece
	}
}

func ConvertRuneFromPiece(p Piece) rune {
	switch p {
	case WPawn:
		return 'P'
	case WKnight:
		return 'N'
	case WBishop:
		return 'B'
	case WRook:
		return 'R'
	case WQueen:
		return 'Q'
	case WKing:
		return 'K'
	case BPawn:
		return 'p'
	case BKnight:
		return 'n'
	case BBishop:
		return 'b'
	case BRook:
		return 'r'
	case BQueen:
		return 'q'
	case BKing:
		return 'k'
	default:
		return '.'
	}
}

func ConvertUpperRuneFromPiece(p Piece) rune {
	switch p {
	case WPawn:
		return 'P'
	case WKnight:
		return 'N'
	case WBishop:
		return 'B'
	case WRook:
		return 'R'
	case WQueen:
		return 'Q'
	case WKing:
		return 'K'
	case BPawn:
		return 'P'
	case BKnight:
		return 'N'
	case BBishop:
		return 'B'
	case BRook:
		return 'R'
	case BQueen:
		return 'Q'
	case BKing:
		return 'K'
	default:
		return '.'
	}
}
