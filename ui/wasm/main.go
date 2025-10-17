package main

import (
	"evilchess/src"
	"evilchess/src/logx"
	"evilchess/ui/gui"
	"fmt"
)

func GetLogger() *logx.Logx {
	l := logx.NewLogx(
		logx.GetLoggerLevelByString("debug"),
		false,
		true,
	)
	l.InitLogger(nil)
	return l
}

func RunGUI() error {
	logger := GetLogger()
	g, err := gui.NewGUI(src.NewBuilderBoard(logger), "/", logger)
	if err != nil {
		logger.Errorf("error init GUI: %v", err)
		return fmt.Errorf("error init GUI: %v", err)
	}
	return g.Run()
}

func main() {
	RunGUI()
}
