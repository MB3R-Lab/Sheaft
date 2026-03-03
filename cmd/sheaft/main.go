package main

import (
	"os"

	"github.com/MB3R-Lab/Sheaft/internal/app"
)

func main() {
	runner := app.NewRunner(os.Stdout, os.Stderr)
	os.Exit(runner.Run(os.Args[1:]))
}
