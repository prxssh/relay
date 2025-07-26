package main

import (
	"fmt"
	"os"

	"github.com/prxssh/relay/internal/tui"
)

func main() {
	if err := tui.Start(); err != nil {
		fmt.Println("Error running RELAY: ", err)
		os.Exit(1)
	}
}
