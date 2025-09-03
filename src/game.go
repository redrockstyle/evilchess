package src

import (
	"evilchess/src/base"
	"evilchess/src/logic/convert/convfen"
	"evilchess/src/logic/convert/convpgn"
	"evilchess/src/logic/history"
	"evilchess/src/logic/rules"
	"evilchess/src/logic/rules/moves"
	"fmt"
	"io"
)

// at firts use Create* methods
type GameBuilder struct {
	board   *base.Board
	history *history.History
	status  base.GameStatus
}

func NewBuilderBoard() *GameBuilder {
	return &GameBuilder{board: nil, history: history.NewHistory(), status: base.Pass}
}

func (gb *GameBuilder) CreateFromPGN(r io.Reader) (base.GameStatus, error) {
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
	board, err := convfen.ConvertFENToBoard(fen)
	if err != nil {
		return base.InvalidGame, fmt.Errorf("error parse FEN: %v", err)
	}
	gb.board = board
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status, nil
}

func (gb *GameBuilder) CreateClassic() {
	gb.status, _ = gb.CreateFromFEN(base.FEN_START_GAME)
	gb.history.SetDefaultInfoGame()
}

func (gb *GameBuilder) Status() base.GameStatus {
	return gb.status
}

func (gb *GameBuilder) Move(move base.Move) base.GameStatus {
	if err := gb.history.PushMove(gb.board, move); err != nil {
		return base.InvalidGame
	}
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) MoveSAN(san string) base.GameStatus {
	mv, err := moves.SANToMove(gb.board, san)
	if err != nil {
		return base.InvalidGame
	}
	return gb.Move(mv)
}

func (gb *GameBuilder) Undo() base.GameStatus {
	gb.history.Undo(gb.board)
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) Redo() base.GameStatus {
	gb.history.Redo(gb.board)
	gb.status = rules.GameStatusOf(gb.board)
	return gb.status
}

func (gb *GameBuilder) CurrentMove(number uint) base.GameStatus {
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
	return convpgn.WritePGN(w, *gb.history.ExportPGNGame())
}

// all SAN moves
func (gb *GameBuilder) PGNBody() string {
	return gb.history.MovesAsPGN()
}

func (gb *GameBuilder) InfoGame() *history.InfoGame {
	return gb.history.InfoGame()
}
