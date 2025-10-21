package convpgn

import (
	"bufio"
	"errors"
	"evilchess/src/chesslib/base"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Portable Game Notation

type PGNStatusGame int

const (
	PGNStatusWW        PGNStatusGame = iota // 1-0
	PGNStatusBW                             // 0-1
	PGNStatusDraw                           // 1/2-1/2
	PGNStatusDW                             // 1/2-0
	PGNStatusDB                             // 0-1/2
	PGNStatusToW                            // +/-
	PGNStatusToB                            // -/+
	PGNStatusToN                            // -/-
	PGNStatusToDraw                         // =/=
	PGNStatusActive                         // *
	PGNStatusUndefined                      // ?
)

type PGNHeader int

const (
	PGNHeaderEvent    PGNHeader = iota // <Seven Tag Roster>
	PGNHeaderSite                      // <Seven Tag Roster>
	PGNHeaderDate                      // <Seven Tag Roster>
	PGNHeaderRound                     // <Seven Tag Roster>
	PGNHeaderResult                    // <Seven Tag Roster>
	PGNHeaderWhite                     // <Seven Tag Roster>
	PGNHeaderWhiteElo                  // white rating
	PGNHeaderBlack                     // <Seven Tag Roster>
	PGNHeaderBlackElo                  // black rating
	PGNHeaderOpening                   // game debut
	PGNHeaderUndefined
)

func ConvStringToPGNStatus(status string) PGNStatusGame {
	switch status {
	case "1-0":
		return PGNStatusWW
	case "0-1":
		return PGNStatusBW
	case "1/2-1/2":
		return PGNStatusDraw
	case "1/2-0":
		return PGNStatusDW
	case "0-1/2":
		return PGNStatusDB
	case "+/-":
		return PGNStatusToW
	case "-/+":
		return PGNStatusToB
	case "-/-":
		return PGNStatusToN
	case "=/=":
		return PGNStatusToDraw
	case "*":
		return PGNStatusActive
	default:
		return PGNStatusUndefined
	}
}

func ConvPGNStatusToString(status PGNStatusGame) string {
	switch status {
	case PGNStatusWW:
		return "1-0"
	case PGNStatusBW:
		return "0-1"
	case PGNStatusDraw:
		return "1/2-1/2"
	case PGNStatusDW:
		return "1/2-0"
	case PGNStatusDB:
		return "0-1/2"
	case PGNStatusToW:
		return "+/-"
	case PGNStatusToB:
		return "-/+"
	case PGNStatusToN:
		return "-/-"
	case PGNStatusToDraw:
		return "=/="
	case PGNStatusActive:
		return "*"
	default:
		return "???"
	}
}

func ConvGameStatusToPGNStatus(gs base.GameStatus, whiteToMove bool) PGNStatusGame {
	switch gs {
	case base.Checkmate:
		if whiteToMove {
			return PGNStatusBW
		}
		return PGNStatusWW
	case base.Stalemate:
		return PGNStatusDraw
	default:
		return PGNStatusActive
	}
}

func ConvStringToPGNHeader(header string) PGNHeader {
	switch header {
	case "Event":
		return PGNHeaderEvent
	case "Site":
		return PGNHeaderSite
	case "Date":
		return PGNHeaderDate
	case "Round":
		return PGNHeaderRound
	case "White":
		return PGNHeaderWhite
	case "WhiteElo":
		return PGNHeaderWhiteElo
	case "Black":
		return PGNHeaderBlack
	case "BlackElo":
		return PGNHeaderBlackElo
	case "Opening":
		return PGNHeaderOpening
	default:
		return PGNHeaderUndefined
	}
}

func ConvPGNHeaderToString(header PGNHeader) string {
	switch header {
	case PGNHeaderEvent:
		return "Event"
	case PGNHeaderSite:
		return "Site"
	case PGNHeaderDate:
		return "Date"
	case PGNHeaderRound:
		return "Round"
	case PGNHeaderWhite:
		return "White"
	case PGNHeaderWhiteElo:
		return "WhiteElo"
	case PGNHeaderBlack:
		return "Black"
	case PGNHeaderBlackElo:
		return "BlackElo"
	case PGNHeaderOpening:
		return "Opening"
	default:
		return "???"
	}
}

var reTag = regexp.MustCompile(`^\s*\[(\w+)\s+"(.*?)"\]\s*$`)
var reResult = regexp.MustCompile(`^(1-0|0-1|1/2-1/2|\*)$`)
var reMoveNum = regexp.MustCompile(`^\d+\.{1,3}$`)
var reNAG = regexp.MustCompile(`^\$\d+$`)

type PGNGame struct {
	Headers map[PGNHeader]string
	Moves   []string
	Result  PGNStatusGame
}

type PGNParser struct {
	r        *bufio.Reader
	pushback []string
}

func NewPGNParser(r io.Reader) *PGNParser {
	return &PGNParser{r: bufio.NewReader(r)}
}

func (p *PGNParser) readLine() (string, error) {
	if len(p.pushback) > 0 {
		// pop last
		n := len(p.pushback)
		ln := p.pushback[n-1]
		p.pushback = p.pushback[:n-1]
		return ln, nil
	}
	ln, err := p.r.ReadString('\n')
	if err != nil {
		// if err == io.EOF && ln != "" {
		// 	// return last partial line with EOF
		// 	return ln, nil
		// }
		return ln, err
	}
	return ln, nil
}

func (p *PGNParser) pushBackLine(ln string) {
	p.pushback = append(p.pushback, ln)
}

func (p *PGNParser) Next() (*PGNGame, error) {
	headers := make(map[PGNHeader]string)
	var line string
	var err error
	found := false

	// read and push headers
	for {
		line, err = p.readLine()
		if err != nil {
			if err == io.EOF {
				if !found {
					return nil, io.EOF
				}
				break
			}
			return nil, err
		}

		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}

		if reTag.MatchString(trim) {
			found = true
			m := reTag.FindStringSubmatch(trim)
			if len(m) >= 3 {
				if header := ConvStringToPGNHeader(m[1]); header != PGNHeaderUndefined {
					headers[header] = m[2]
				}
			}
			continue
		}

		p.pushBackLine(line)
		break
	}

	// read and push body
	var b strings.Builder
	for {
		line, err = p.readLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			p.pushBackLine(line)
			break
		}
		b.WriteString(line)
	}

	cleanBody := normalizeBody(b.String())
	if strings.TrimSpace(cleanBody) == "" && len(headers) == 0 {
		return nil, io.EOF
	}

	streamMoves := strings.Fields(cleanBody)
	moves := make([]string, 0, len(streamMoves))
	result := "*"
	for _, str := range streamMoves {
		if str == "" {
			continue
		}
		if reMoveNum.MatchString(str) {
			continue
		}
		if reNAG.MatchString(str) {
			continue
		}
		if reResult.MatchString(str) {
			result = str
			continue
		}
		moves = append(moves, str)
	}

	return &PGNGame{Headers: headers, Moves: moves, Result: ConvStringToPGNStatus(result)}, nil
}

func ParseOne(r io.Reader) (*PGNGame, error) {
	p := NewPGNParser(r)
	return p.Next()
}

func ParseAll(r io.Reader) ([]*PGNGame, error) {
	p := NewPGNParser(r)
	var games []*PGNGame
	for {
		g, err := p.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return games, err
		}
		games = append(games, g)
	}
	return games, nil
}

// strip comments and side moves: "{ ... }" ";" "( ... )" spaces->space
func normalizeBody(s string) string {
	var sb strings.Builder
	lines := strings.Split(s, "\n")
	for i := range lines {
		ln := lines[i]
		if idx := strings.IndexByte(ln, ';'); idx >= 0 {
			ln = ln[:idx]
		}
		sb.WriteString(ln)
		sb.WriteByte('\n')
	}
	s = sb.String()

	s = removeDelimited(s, '{', '}')
	s = removeDelimited(s, '(', ')')
	reSpace := regexp.MustCompile(`\s+`)
	clean := strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
	return clean
}

func removeDelimited(s string, open, close rune) string {
	runes := []rune(s)
	var out []rune
	depth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == open {
			depth++
			continue
		}
		if r == close {
			if depth > 0 {
				depth--
				continue
			}
			// unmatched close
		}
		if depth == 0 {
			out = append(out, r)
		}
	}
	return string(out)
}

// write PGN info
func WritePGN(w io.Writer, game PGNGame) error {
	if w == nil {
		return fmt.Errorf("nil writer")
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()

	headerOrder := []PGNHeader{
		PGNHeaderEvent,
		PGNHeaderSite,
		PGNHeaderDate,
		PGNHeaderRound,
		PGNHeaderWhite,
		PGNHeaderWhiteElo,
		PGNHeaderBlack,
		PGNHeaderBlackElo,
		PGNHeaderOpening,
	}

	printed := make(map[PGNHeader]bool)
	escape := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		return s
	}

	for _, hh := range headerOrder {
		if v, ok := game.Headers[hh]; ok && strings.TrimSpace(v) != "" {
			if _, err := fmt.Fprintf(bw, "[%s \"%s\"]\n", ConvPGNHeaderToString(hh), escape(v)); err != nil {
				return err
			}
			printed[hh] = true
		}
	}

	for hh, v := range game.Headers {
		if printed[hh] {
			continue
		}
		if strings.TrimSpace(v) == "" {
			continue
		}
		if _, err := fmt.Fprintf(bw, "[%s \"%s\"]\n", ConvPGNHeaderToString(hh), escape(v)); err != nil {
			return err
		}
		printed[hh] = true
	}

	resStr := ConvPGNStatusToString(game.Result)
	if _, err := fmt.Fprintf(bw, "[Result \"%s\"]\n\n", resStr); err != nil {
		return err
	}

	// body moves
	moves := []string(game.Moves)
	if len(moves) > 0 {
		moveNum := 1
		for i := 0; i < len(moves); i += 2 {
			white := strings.TrimSpace(moves[i])
			if white == "" {
				continue
			}
			if _, err := fmt.Fprintf(bw, "%d. %s", moveNum, white); err != nil {
				return err
			}
			if i+1 < len(moves) {
				black := strings.TrimSpace(moves[i+1])
				if black != "" {
					if _, err := fmt.Fprintf(bw, " %s", black); err != nil {
						return err
					}
				}
			}
			if i+2 < len(moves) {
				if _, err := bw.WriteString(" "); err != nil {
					return err
				}
			}
			moveNum++
		}
		if _, err := bw.WriteString(" "); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(bw, "%s\n", resStr); err != nil {
		return err
	}

	return bw.Flush()
}
