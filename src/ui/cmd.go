package ui

import (
	"context"
	"evilchess/src/chesslib"
	"evilchess/src/chesslib/engine"
	"evilchess/src/chesslib/engine/uci"
	"evilchess/src/logx"
	clic "evilchess/src/ui/cli"
	"evilchess/src/ui/gui"
	"evilchess/src/ui/gui/gbase"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

const logfile string = "evilchess.log"

func GetLogger(file *os.File, c *cli.Command) *logx.Logx {
	l := logx.NewLogx(
		logx.GetLoggerLevelByString(c.String("level")),
		c.Bool("dev"),
		c.Bool("console"),
	)
	l.InitLogger(file)
	return l
}

func RunGUI(c *cli.Command) error {
	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error open logfile: %v", err)
	}
	defer file.Close()
	logger := GetLogger(file, c)
	g, err := gui.NewGUI(chesslib.NewBuilderBoard(logger), "", logger)
	if err != nil {
		logger.Errorf("error init GUI: %v", err)
		return fmt.Errorf("error init GUI: %v", err)
	}
	return g.Run()
}

func RunEvilChess() error {
	ff := &cli.StringFlag{
		Name:  "fen",
		Usage: "string FEN format",
	}
	pf := &cli.StringFlag{
		Name:  "pgn",
		Usage: "path to PGN file",
	}
	df := &cli.BoolFlag{
		Name:    "dev",
		Aliases: []string{"d"},
		Usage:   "dev encode log",
	}
	lf := &cli.StringFlag{
		Name:        "level",
		Aliases:     []string{"l"},
		Usage:       "level log",
		DefaultText: "info",
	}
	cf := &cli.BoolFlag{
		Name:    "console",
		Aliases: []string{"c"},
		Usage:   "console log",
	}
	cliff := []cli.Flag{ff, pf, df, lf, cf}
	guiff := []cli.Flag{df, lf, cf}

	return (&cli.Command{
		Name:  "evilchess",
		Usage: "mini chess game",
		Commands: []*cli.Command{
			{
				Name:  "cli",
				Flags: cliff,
				Action: func(ctx context.Context, c *cli.Command) error {
					fen := c.String("fen")
					pgn := c.String("pgn")
					file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
					if err != nil {
						fmt.Printf("error open logfile: %v", err)
						return nil
					}
					defer file.Close()
					logger := GetLogger(file, c)
					gb := chesslib.NewBuilderBoard(logger)
					if pgn != "" {
						file, err := os.Open(pgn)
						if err != nil {
							fmt.Printf("error open file: %v", err)
							return nil
						}
						defer file.Close()
						if _, err = gb.CreateFromPGN(file); err != nil {
							fmt.Printf("error read PGN file: %v", err)
							return nil
						}
					} else if fen != "" {
						if _, err := gb.CreateFromFEN(fen); err != nil {
							return nil
						}
					} else {
						// for my engine
						// gb.SetEngineWorker(myengine.NewEvilEngine())
						// for UCI engine
						gb.SetEngineWorker(uci.NewUCIExec(logger, "materials/engine/stockfish/stockfish"))
						// set level
						gb.SetEngineLevel(engine.LevelFive)
						gb.CreateClassic()
					}

					// clic.EnableANSI()
					cl := clic.NewCLI(gb, clic.PrintMailbox)
					// if err := cl.RunLineMode(); err != nil {
					if err := cl.Run(); err != nil {
						fmt.Printf("error evilchess: %v", err)
					}
					return nil
				},
			},
			{
				Name:  "gui",
				Flags: guiff,
				Action: func(ctx context.Context, c *cli.Command) error {
					if err := RunGUI(c); err != nil && err != gbase.ErrExit {
						fmt.Printf("error GUI: %v", err)
					}
					return nil
				},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if err := RunGUI(c); err != nil && err != gbase.ErrExit {
				fmt.Printf("error GUI: %v", err)
			}
			return nil
		},
	}).Run(context.Background(), os.Args)
}
