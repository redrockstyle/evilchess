package cli

import (
	"bufio"
	"evilchess/src"
	"evilchess/src/base"
	"evilchess/src/engine"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

type DrawFunc func(mb base.Mailbox)

type CLIProcessing struct {
	builder *src.GameBuilder
	draw    DrawFunc
	in      *os.File
	out     io.Writer
}

func NewCLI(b *src.GameBuilder, draw DrawFunc) *CLIProcessing {
	return &CLIProcessing{builder: b, draw: draw, in: os.Stdin, out: os.Stdout}
}

// raw processing
// - enter SAN move
// - left/right arrow keys to undo/redo
// - q or Ctrl+C to exit
// - redraw board every move
func (c *CLIProcessing) Run() error {
	fd := int(c.in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return c.RunLineMode()
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	// use a buffered reader to read bytes
	r := bufio.NewReader(c.in)
	var inputBuf strings.Builder

	// initial draw
	c.draw(c.builder.CurrentBoard())
	c.printStatus()
	fmt.Fprint(c.out, "\nType SAN and press Enter, or use left/right arrows to undo/redo, 'i' to PGN, 'm' to moves, 'l' 'q' to quit.\n")

	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}

		// handle control
		if b == 3 { // Ctrl+C
			fmt.Fprintln(c.out, "\nInterrupted")
			return nil
		}
		if b == 0x1b { // escape sequence — possible arrow
			// read next two bytes (CSI)
			b1, err := r.ReadByte()
			if err != nil {
				continue
			}
			b2, err := r.ReadByte()
			if err != nil {
				continue
			}
			if b1 == '[' {
				switch b2 {
				case 'D': // left arrow
					c.builder.Undo()
					c.draw(c.builder.CurrentBoard())
					c.printStatus()
					if terminalFinished(c.builder.Status()) {
						return nil
					}
				case 'C': // right arrow
					c.builder.Redo()
					c.draw(c.builder.CurrentBoard())
					c.printStatus()
					if terminalFinished(c.builder.Status()) {
						return nil
					}
				}
			}
			continue
		}

		// newline/enter
		if b == '\r' || b == '\n' {
			s := strings.TrimSpace(inputBuf.String())
			inputBuf.Reset()
			if s == "" {
				continue
			}
			// quit command
			if s == "q" || s == "Q" || s == "quit" {
				fmt.Fprintln(c.out, "\nQuitting")
				return nil
			}
			if s >= "1" && s <= "9" {
				lvl, _ := strconv.Atoi(s)
				fmt.Fprintf(c.out, "\nSet level engine %d\n", lvl)
				c.builder.SetEngineLevel(engine.LevelAnalyze(lvl))
				continue
			}
			if s == "m" {
				fmt.Fprintf(c.out, "\nMoves: %s\n", c.builder.PGNBody())
				continue
			}
			if s == "i" {
				fmt.Fprintf(c.out, "\nMoves: %s\n", c.builder.PGNBody())
				continue
			}

			if s == "?" {
				status := c.builder.EngineMove()
				if status == base.InvalidGame {
					fmt.Fprintln(c.out, "\nError runtime engine")
				}
				c.draw(c.builder.CurrentBoard())
				c.printStatus()
				continue
			} else {
				// try SAN move
				status := c.builder.MoveSAN(s)
				if status == base.InvalidGame {
					fmt.Fprintf(c.out, "Invalid move: %s\n", s)
					// redraw board anyway
					c.draw(c.builder.CurrentBoard())
					c.printStatus()
					continue
				}
				// success
				c.draw(c.builder.CurrentBoard())
				c.printStatus()
				if terminalFinished(status) {
					return nil
				}
			}
			continue
		}

		// printable chars: append to buffer
		if b >= 32 && b <= 126 {
			// support utf8 — but SAN uses ascii
			inputBuf.WriteByte(b)
			// echo
			fmt.Fprintf(c.out, "%c", b)
			continue
		}
		// other keys ignored
	}
}

func (c *CLIProcessing) RunLineMode() error {
	// fallback: basic line mode using bufio.Scanner
	scanner := bufio.NewScanner(c.in)
	c.draw(c.builder.CurrentBoard())
	c.printStatus()
	fmt.Fprintln(c.out, "Enter SAN and press Enter. Use 'undo'/'redo' to navigate, 'q' to quit.")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "q" || line == "Q" {
			return nil
		}
		if line == "undo" {
			c.builder.Undo()
			c.draw(c.builder.CurrentBoard())
			c.printStatus()
			continue
		}
		if line == "redo" {
			c.builder.Redo()
			c.draw(c.builder.CurrentBoard())
			c.printStatus()
			continue
		}
		if line == "moves" {
			fmt.Fprintln(c.out, c.builder.PGNBody())
			continue
		}
		if line == "pgn" {
			fmt.Fprintln(c.out, "--------- PGN FORMAT---------")
			if err := c.builder.PGN(c.out); err != nil {
				fmt.Fprintf(c.out, "error write pgn %v\n", err)
			}
			fmt.Fprintln(c.out, "--------- PGN FORMAT---------")
			continue
		}
		status := c.builder.MoveSAN(line)
		if status == base.InvalidGame {
			fmt.Fprintf(c.out, "Invalid move: %s\n", line)
		} else {
			c.draw(c.builder.CurrentBoard())
			c.printStatus()
			if terminalFinished(status) {
				return nil
			}
		}
	}
	return scanner.Err()
}

func (c *CLIProcessing) printStatus() {
	status := c.builder.Status()
	fmt.Fprintln(c.out)
	fmt.Fprintf(c.out, "FEN: %s\n", c.builder.FEN())
	fmt.Fprintf(c.out, "Moves: %s\n", c.builder.PGNBody())
	fmt.Fprintf(c.out, "Status: %s\n", statusString(status))
}

func statusString(s base.GameStatus) string {
	switch s {
	case base.Check:
		return "Check"
	case base.Checkmate:
		return "Checkmate"
	case base.Stalemate:
		return "Stalemate"
	case base.Pass:
		return "Normal"
	case base.InvalidGame:
		return "Invalid"
	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

func terminalFinished(s base.GameStatus) bool {
	return s == base.Checkmate || s == base.Stalemate
}
