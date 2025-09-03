package history

import (
	"errors"
	"evilchess/src/base"
	"evilchess/src/logic/convert/convpgn"
	"evilchess/src/logic/rules"
	"evilchess/src/logic/rules/moves"
	"fmt"
	"strings"
)

// turncate history module
type History struct {
	info    *InfoGame
	moves   []MoveEntry
	current uint // actual move
}

type MoveEntry struct {
	Move  base.Move
	SAN   string
	Board base.Board // copy board
}

func NewHistory() *History {
	return &History{moves: make([]MoveEntry, 0), current: 0, info: NewInfoGame()}
}
func (h *History) Len() int          { return len(h.moves) }
func (h *History) CurrentMove() uint { return h.current }

func (h *History) Moves() []MoveEntry {
	out := make([]MoveEntry, len(h.moves))
	copy(out, h.moves)
	return out
}

// Check Move and push to history
func (h *History) PushMove(b *base.Board, mv base.Move) error {
	if b == nil {
		return errors.New("nil board")
	}

	if !rules.IsLegalMove(b, mv) {
		return fmt.Errorf("illegal move: %+v", mv)
	}

	if h.Len() == 0 {
		h.moves = append(h.moves, MoveEntry{Board: *b, Move: base.Move{}, SAN: ""})
	}

	// Truncate future moves if we are in the middle (truncate behavior).
	if h.current+1 < uint(h.Len()) {
		h.moves = h.moves[:h.current]
	}

	// Apply move to the board
	if err := moves.ApplyMove(b, mv); err != nil {
		return fmt.Errorf("ApplyMove failed: %w", err)
	}

	h.moves = append(h.moves, MoveEntry{Board: *b, Move: mv, SAN: moves.MoveToShortSAN(mv)})
	h.current++
	return nil
}

func (h *History) GotoMove(b *base.Board, index uint) error {
	if b == nil {
		return errors.New("nil board")
	}
	if h.Len() == 0 {
		return errors.New("empty history")
	}
	if index > uint(h.Len()) {
		return errors.New("invalid index")
	}
	if index == uint(h.Len()) {
		*b = h.moves[h.Len()-1].Board
		h.current = uint(h.Len() - 1)
	} else {
		// copy board as value
		*b = h.moves[index].Board
		h.current = index
	}
	return nil
}

// undo and rewrite board
func (h *History) Undo(b *base.Board) error {
	return h.GotoMove(b, h.current-1)
}

// redo and rewrite board
func (h *History) Redo(b *base.Board) error {
	return h.GotoMove(b, h.current+1)
}

// returned string with all moves
// example: "1. e4 e5 2. Nf3 Nc6 3. Bb5"
func (h *History) MovesAsPGN() string {
	if h == nil || h.Len() == 0 {
		return ""
	}

	var b strings.Builder
	moveNum := 1

	for i := 1; i < h.Len(); i += 2 {
		white := strings.TrimSpace(h.moves[i].SAN)
		if white == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("%d. %s", moveNum, white))
		if i+1 < h.Len() {
			black := strings.TrimSpace(h.moves[i+1].SAN)
			if black != "" {
				b.WriteString(" ")
				b.WriteString(black)
			}
		}

		if i+2 < h.Len() {
			b.WriteString(" ")
		}

		moveNum++
	}

	return b.String()
}

func (h *History) SAN() []string {
	len := len(h.moves)
	out := make([]string, len)
	for i := 1; i < len; i++ {
		out[i-1] = h.moves[i].SAN
	}
	return out
}

// copy game history and go to last position
func (h *History) ImportPGNGame(pgn *convpgn.PGNGame, b *base.Board) error {
	h.info.headers = pgn.Headers
	len := len(pgn.Moves)
	h.moves = make([]MoveEntry, len)
	h.moves[0].Board = *b
	for i := 1; i < len; i++ {
		mv, err := moves.SANToMove(&h.moves[i-1].Board, pgn.Moves[i-1])
		if err != nil {
			h.moves = nil
			h.info = nil
			h.current = 0
			return err
		}
		if err = moves.ApplyMove(b, mv); err != nil {
			h.moves = nil
			h.info = nil
			h.current = 0
			return err
		}
		h.moves[i].Board = *b
		h.moves[i].Move = mv
	}
	h.current = uint(len)

	return nil
}

func (h *History) ExportPGNGame() *convpgn.PGNGame {
	var status base.GameStatus
	var move bool
	if h.Len() > 0 {
		status = rules.GameStatusOf(&h.moves[h.Len()-1].Board)
		move = h.moves[h.Len()-1].Board.WhiteToMove
	} else {
		status = base.Pass
		move = true
	}

	return &convpgn.PGNGame{
		Headers: h.info.headers,
		Moves:   h.SAN(),
		Result: convpgn.ConvGameStatusToPGNStatus(
			status,
			move,
		),
	}
}

func (h *History) InfoGame() *InfoGame {
	return h.info
}

// default
func (h *History) SetDefaultInfoGame() {
	var status base.GameStatus
	var move bool
	if h.Len() > 0 {
		status = rules.GameStatusOf(&h.moves[h.Len()-1].Board)
		move = h.moves[h.Len()-1].Board.WhiteToMove
	} else {
		status = base.Pass
		move = true
	}

	h.info.SetWhitePlayer("PlayerOne")
	h.info.SetBlackPlayer("PlayerTwo")
	h.info.SetEvent("evilchess test game")
	h.info.SetDate("2025.09.07")
	h.info.SetWhiteElo("1500")
	h.info.SetBlackElo("1888")
	h.info.SetRound("0")
	h.info.SetSite("localhost")
	h.info.SetOpening("testyle")
	h.info.SetResult(
		convpgn.ConvPGNStatusToString(
			convpgn.ConvGameStatusToPGNStatus(
				status,
				move,
			),
		),
	)
}
