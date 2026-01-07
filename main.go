package main

import (
	"os"

	"github.com/mpjhorner/superralph/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
