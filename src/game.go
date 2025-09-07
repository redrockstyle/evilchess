package src

import (
	"evilchess/src/base"
	"evilchess/src/engine"
	"evilchess/src/logic/convert/convfen"
	"evilchess/src/logic/convert/convpgn"
	"evilchess/src/logic/history"
	"evilchess/src/logic/rules"
	"evilchess/src/logic/rules/moves"
	"evilchess/src/logx"
	"fmt"
	"io"
)

// at firts use Create* methods
type GameBuilder struct {
	board   *base.Board
	history *history.History
	status  base.GameStatus
	level   engine.LevelAnalyze
	engine  engine.Engine
	logger  logx.Logger
}

func NewBuilderBoard(logger logx.Logger) *GameBuilder {
	return &GameBuilder{board: nil, history: history.NewHistory(), status: base.Pass, engine: nil, level: engine.LevelInvalid, logger: logger}
}

func (gb *GameBuilder) CreateFromPGN(r io.Reader) (base.GameStatus, error) {
	gb.logger.Debug("create game by PGN")
	pgn, err := convpgn.ParseOne(r)
	if err != nil {
		return base.InvalidGame, err
	}
	gb.status, _ = gb.CreateFromFEN(base.FEN_START_GAME)
	if err = gb.history.ImportPGNGame(pgn, gb.board); err != nil {
		return base.InvalidGame, err
	}
	return gb.status, nil
}

func (gb *GameBuilder) CreateFromFEN(fen string) (base.GameStatus, error) {
	gb.logger.Debugf("create game by FEN: %v", fen)
	board, err := convfen.ConvertFENToBoard(fen)
	if err != nil {
		return base.InvalidGame, fmt.Errorf("error parse FEN: %v", err)
	}
	gb.board = board
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status, nil
}

func (gb *GameBuilder) CreateClassic() {
	gb.logger.Debug("create classic game")
	gb.status, _ = gb.CreateFromFEN(base.FEN_START_GAME)
	gb.history.SetDefaultInfoGame()
}

func (gb *GameBuilder) Status() base.GameStatus {
	return gb.status
}

func (gb *GameBuilder) Move(move base.Move) base.GameStatus {
	gb.logger.Infof("move from %d to %d", base.ConvPointToIndex(move.From), base.ConvPointToIndex(move.To))
	if err := gb.history.PushMove(gb.board, move); err != nil {
		return base.InvalidGame
	}
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) MoveSAN(san string) base.GameStatus {
	gb.logger.Infof("move SAN: %v", san)
	mv, err := moves.SANToMove(gb.board, san)
	if err != nil {
		return base.InvalidGame
	}
	return gb.Move(mv)
}

func (gb *GameBuilder) Undo() base.GameStatus {
	gb.logger.Debug("call undo")
	gb.history.Undo(gb.board)
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) Redo() base.GameStatus {
	gb.logger.Debug("call redo")
	gb.history.Redo(gb.board)
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) CurrentMove(number uint) base.GameStatus {
	gb.logger.Debugf("call undo")
	// pass <some_number>: offset game to current move
	gb.history.GotoMove(gb.board, number)
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) CurrentBoard() base.Mailbox {
	return gb.board.Mailbox
}

// return FEN of this game
func (gb *GameBuilder) FEN() string {
	return convfen.ConvertBoardToFEN(*gb.board)
}

// return PGN of this game
func (gb *GameBuilder) PGN(w io.Writer) error {
	gb.logger.Debug("get actual PGN")
	return convpgn.WritePGN(w, *gb.history.ExportPGNGame())
}

// all SAN moves
func (gb *GameBuilder) PGNBody() string {
	gb.logger.Debug("get actual moves")
	return gb.history.MovesAsPGN()
}

func (gb *GameBuilder) InfoGame() *history.InfoGame {
	return gb.history.InfoGame()
}

// ---- Engine ----

func (gb *GameBuilder) InitEngine(lvl engine.LevelAnalyze, e engine.Engine) {
	gb.engine = e
	gb.level = lvl
}

func (gb *GameBuilder) EngineMove() base.GameStatus {
	if gb.engine == nil || gb.level == engine.LevelInvalid {
		return base.InvalidGame
	}

	var err error
	var res engine.AnalysisInfo
	err = gb.engine.SetPosition(gb.board)
	if err != nil {
		return base.InvalidGame
	}
	err = gb.engine.StartAnalysis(*gb.engine.LevelToParams(gb.level))
	if err != nil {
		return base.InvalidGame
	}
	res, err = gb.engine.WaitDone()
	if err != nil {
		return base.InvalidGame
	}
	return gb.Move(*res.BestMove)
}
