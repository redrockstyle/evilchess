package engine

import "evilchess/src/base"

type AnalysisInfo struct {
	Depth    int         // текущая глубина
	TimeMs   int64       // прошедшее время в ms
	Nodes    int64       // число просмотренных узлов
	NPS      int64       // nodes per second
	ScoreCP  int         // оценка в сантиматах (+ = advantage for side to move)
	MateIn   int         // mate in N plies (0 if none)
	PV       []base.Move // principal variation (список ходов, начиная с best)
	BestMove *base.Move  // лучший ход на данный момент (ссылка на первый ход PV)
}

type SearchParams struct {
	MaxDepth  int   // 0 = unlimited (but bounded by MaxTimeMs/MaxNodes)
	MaxTimeMs int64 // 0 = no time limit
	MaxNodes  int64 // 0 = no node limit
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
	// mb more
	LevelLast
	LevelInvalid
)

// Engine — интерфейс движка
type Engine interface {
	Init(opts map[string]interface{}) error
	SetPositionFEN(fen string) error
	SetPosition(b *base.Board) error
	LevelToParams(lvl LevelAnalyze) *SearchParams
	StartAnalysis(params SearchParams) error
	StopAnalysis() error
	BestNow() (AnalysisInfo, error)
	WaitDone() (AnalysisInfo, error)
	Subscribe(ch chan<- AnalysisInfo) (unsubscribe func())
	Close() error
}
