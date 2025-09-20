package engine

import (
	"evilchess/src/base"
	"time"
)

type AnalysisInfo struct {
	Depth       int         // текущая глубина
	TimeMs      int64       // прошедшее время в ms
	Nodes       int64       // число просмотренных узлов
	NPS         int64       // nodes per second
	ScoreCP     int         // оценка в сантиматах (+ = advantage for side to move)
	MateIn      int         // mate in N plies (0 if none)
	PV          []base.Move // principal variation (список ходов, начиная с best)
	BestMove    *base.Move  // лучший ход на данный момент (ссылка на первый ход PV)
	UCIPV       []string
	UCIBestMove string
}

type SearchParams struct {
	MaxDepth  int   // 0 = unlimited (but bounded by MaxTimeMs/MaxNodes)
	MaxTimeMs int64 // 0 = no time limits
	Infinite  bool  // if true, search indefinitely until StopAnalysis()
}

type LevelAnalyze int

const (
	LevelOne LevelAnalyze = iota
	LevelTwo
	LevelThree
	LevelFour
	LevelFive
	LevelSix
	LevelSeven
	LevelEight
	LevelNine
	LevelTen
	// ...
	LevelLast
	LevelInvalid
)

const (
	UCIHandshakeTimeout = 2 * time.Second  // uci / isready
	UCIBestMoveTimeout  = 30 * time.Second // go ...
	StopAnalyzeTimeout  = 5 * time.Second  // for infinite analysis
)

// Engine — интерфейс движка
type Engine interface {
	Init() error
	SetPositionFEN(fen string) error
	SetPosition(b *base.Board) error
	StartAnalysis(params SearchParams) error
	StopAnalysis() error
	BestNow() AnalysisInfo
	WaitDone()
	Subscribe(ch chan<- AnalysisInfo) (unsubscribe func())
	Close()
}

// helper: parse uci move (e2e4, e7e8q, etc.) into base.Move.
// whiteToMove indicates whether this move is made by White (for promotion piece color).
func (i *AnalysisInfo) GetBestMove(b *base.Board) *base.Move {
	if i.BestMove != nil {
		return i.BestMove
	}

	if len(i.UCIBestMove) < 4 {
		return nil
	}
	from := i.UCIBestMove[0:2]
	to := i.UCIBestMove[2:4]
	fromIdx, err := base.SquareFromAlgebraic(from)
	if err != nil {
		return nil
	}
	toIdx, err := base.SquareFromAlgebraic(to)
	if err != nil {
		return nil
	}
	return &base.Move{
		From:  base.ConvIndexToPoint(fromIdx),
		To:    base.ConvIndexToPoint(toIdx),
		Piece: b.Mailbox[fromIdx],
	}
}

func LevelToParams(lvl LevelAnalyze) SearchParams {
	switch lvl {
	case LevelOne:
		return SearchParams{
			MaxDepth:  1,
			MaxTimeMs: 500,
			Infinite:  false,
		}
	case LevelTwo:
		return SearchParams{
			MaxDepth:  2,
			MaxTimeMs: 800,
			Infinite:  false,
		}
	case LevelThree:
		return SearchParams{
			MaxDepth:  3,
			MaxTimeMs: 1000,
			Infinite:  false,
		}
	case LevelFour:
		return SearchParams{
			MaxDepth:  5,
			MaxTimeMs: 1500,
			Infinite:  false,
		}
	case LevelFive:
		return SearchParams{
			MaxDepth:  7,
			MaxTimeMs: 2500,
			Infinite:  false,
		}
	case LevelSix:
		return SearchParams{
			MaxDepth:  9,
			MaxTimeMs: 4000,
			Infinite:  false,
		}
	case LevelSeven:
		return SearchParams{
			MaxDepth:  11,
			MaxTimeMs: 6000,
			Infinite:  false,
		}
	case LevelEight:
		return SearchParams{
			MaxDepth:  13,
			MaxTimeMs: 8000,
			Infinite:  false,
		}
	case LevelNine:
		return SearchParams{
			MaxDepth:  16,
			MaxTimeMs: 10000,
			Infinite:  false,
		}
	case LevelTen:
		return SearchParams{
			MaxDepth:  18,
			MaxTimeMs: 15000,
			Infinite:  false,
		}
	default:
		// Full strength
		return SearchParams{
			MaxDepth:  0,
			MaxTimeMs: 0,
			Infinite:  true,
		}
	}
}
