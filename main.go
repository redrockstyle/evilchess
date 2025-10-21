package main

import (
	"evilchess/src/ui"
	"fmt"
)

func main() {
	if err := ui.RunEvilChess(); err != nil {
		fmt.Println(err)
	}
}
