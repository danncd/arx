package main

import (
	"fmt"
	"os"

	"arx/internal/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "arx:", err)
		os.Exit(1)
	}
}
