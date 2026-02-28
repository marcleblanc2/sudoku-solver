// Command sudoku-solver solves a Sudoku puzzle from a screenshot.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/marcleblanc2/sudoku-solver/internal/logging"

	"go.opentelemetry.io/otel/attribute"
)

const usage = `Usage: sudoku-solver [options] [input-image]

Solve a Sudoku puzzle from a screenshot.

Arguments:
  input-image          Path to the unsolved puzzle image

Options:
  -i, --input string   Path to the unsolved puzzle image
  -q, --quiet          Suppress logging to stderr
  -h, --help           Show this help message

Environment variables:
  LOG_LEVEL            Log level: debug, info, warn, error (default: info)
  LOG_FILE             Log file path; set to "" to disable (default: sudoku-solver.log)

Examples:
  sudoku-solver puzzle.png
  sudoku-solver -i puzzle.png
  sudoku-solver -q -i puzzle.png
`

func main() {
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	var inputFlag string
	flag.StringVar(&inputFlag, "i", "", "path to the unsolved puzzle image")
	flag.StringVar(&inputFlag, "input", "", "path to the unsolved puzzle image")

	var quiet bool
	flag.BoolVar(&quiet, "q", false, "suppress stderr logging")
	flag.BoolVar(&quiet, "quiet", false, "suppress stderr logging")

	flag.Parse()

	input, err := resolveInput(inputFlag, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFile := "sudoku-solver.log"
	if v, ok := os.LookupEnv("LOG_FILE"); ok {
		logFile = v
	}

	ctx := context.Background()
	_, tracerProvider, cleanup, err := logging.Setup(ctx, logging.Config{
		Level:   logLevel,
		Quiet:   quiet,
		LogFile: logFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "logging setup: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	tracer := tracerProvider.Tracer("sudoku-solver")
	ctx, span := tracer.Start(ctx, "solve")
	defer span.End()

	span.SetAttributes(
		attribute.String("input", input),
		attribute.String("log_level", logLevel),
		attribute.Bool("quiet", quiet),
		attribute.String("log_file", logFile),
	)

	// TODO: solve the puzzle using ctx
	// packages can add attributes via trace.SpanFromContext(ctx).SetAttributes(...)
	_ = ctx
}

// resolveInput determines the input image path from the -i flag and positional
// arguments. It returns an error if both are provided or neither is provided.
func resolveInput(flagValue string, args []string) (string, error) {
	hasFlag := flagValue != ""
	hasPositional := len(args) == 1

	if len(args) > 1 {
		return "", fmt.Errorf("unexpected arguments: %v", args[1:])
	}
	if hasFlag && hasPositional {
		return "", fmt.Errorf("input image specified as both flag (-i %s) and argument (%s); use one or the other", flagValue, args[0])
	}
	if !hasFlag && !hasPositional {
		return "", fmt.Errorf("no input image specified")
	}

	if hasFlag {
		return flagValue, nil
	}
	return args[0], nil
}
