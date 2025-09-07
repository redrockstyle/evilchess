package main

import (
	"evilchess/ui"
	"fmt"
)

func main() {
	if err := ui.RunEvilChess(); err != nil {
		fmt.Println(err)
	}
}
