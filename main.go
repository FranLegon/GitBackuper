package main

import (
	"os"

	"github.com/FranLegon/GitBackuper/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
