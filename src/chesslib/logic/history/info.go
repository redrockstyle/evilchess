package history

import "evilchess/src/chesslib/logic/convert/convpgn"

type InfoGame struct {
	headers map[convpgn.PGNHeader]string
}

func NewInfoGame() *InfoGame {
	return &InfoGame{headers: make(map[convpgn.PGNHeader]string)}
}

// event
func (i *InfoGame) SetEvent(name string) { i.headers[convpgn.PGNHeaderEvent] = name }
func (i *InfoGame) GetEvent() string     { return i.headers[convpgn.PGNHeaderEvent] }

// date
func (i *InfoGame) SetDate(name string) { i.headers[convpgn.PGNHeaderDate] = name }
func (i *InfoGame) GetDate() string     { return i.headers[convpgn.PGNHeaderDate] }

// players
func (i *InfoGame) SetWhitePlayer(name string) { i.headers[convpgn.PGNHeaderWhite] = name }
func (i *InfoGame) GetWhitePlayer() string     { return i.headers[convpgn.PGNHeaderWhite] }
func (i *InfoGame) SetBlackPlayer(name string) { i.headers[convpgn.PGNHeaderBlack] = name }
func (i *InfoGame) GetBlackPlayer() string     { return i.headers[convpgn.PGNHeaderBlack] }

// elo
func (i *InfoGame) SetWhiteElo(name string) { i.headers[convpgn.PGNHeaderWhiteElo] = name }
func (i *InfoGame) GetWhiteElo() string     { return i.headers[convpgn.PGNHeaderWhiteElo] }
func (i *InfoGame) SetBlackElo(name string) { i.headers[convpgn.PGNHeaderBlackElo] = name }
func (i *InfoGame) GetBlackElo() string     { return i.headers[convpgn.PGNHeaderBlackElo] }

// result
func (i *InfoGame) SetResult(name string) { i.headers[convpgn.PGNHeaderResult] = name }
func (i *InfoGame) GetResult() string     { return i.headers[convpgn.PGNHeaderResult] }

// round
func (i *InfoGame) SetRound(name string) { i.headers[convpgn.PGNHeaderRound] = name }
func (i *InfoGame) GetRound() string     { return i.headers[convpgn.PGNHeaderRound] }

// round
func (i *InfoGame) SetSite(name string) { i.headers[convpgn.PGNHeaderSite] = name }
func (i *InfoGame) GetSite() string     { return i.headers[convpgn.PGNHeaderSite] }

// opening
func (i *InfoGame) SetOpening(name string) { i.headers[convpgn.PGNHeaderOpening] = name }
func (i *InfoGame) GetOpening() string     { return i.headers[convpgn.PGNHeaderOpening] }
