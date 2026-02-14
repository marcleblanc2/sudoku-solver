// Command sudoku-solver is the entry point for the sudoku-solver CLI.
// It wires up subcommands (solve, serve) and runs the app.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/marcleblanc2/sudoku-solver/internal/logging"
)

func main() {
	quiet := flag.Bool("q", false, "suppress stderr logging")
	flag.BoolVar(quiet, "quiet", false, "suppress stderr logging")
	flag.Parse()

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFile := "sudoku-solver.log"
	if v, ok := os.LookupEnv("LOG_FILE"); ok {
		logFile = v
	}

	ctx := context.Background()
	_, cleanup, err := logging.Setup(ctx, logging.Config{
		Level:   logLevel,
		Quiet:   *quiet,
		LogFile: logFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	fmt.Println("sudoku-solver: ready")
}
