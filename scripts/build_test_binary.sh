#!/bin/sh

# build_test_binary.sh — lint, test, build, and smoke-test the sudoku-solver binary.
# Run locally or in CI; mirrors exactly what GitHub Actions runs.

# Usage: ./scripts/build_test_binary.sh

# Requirements (install via README.md setup instructions):
# go
# golangci-lint
# Leptonica
# markdownlint-cli2
# Tesseract
# vale

set -eu

# Always run from the repository root, regardless of where the script is called from.
cd "$(dirname "$0")/.."

# CGO flags for Tesseract / Leptonica
# On macOS (local or GitHub Actions), derive paths from Homebrew.
# On Linux, the system package paths are used by the compiler automatically.
if command -v brew >/dev/null 2>&1; then
  # shellcheck disable=SC2155
  export CGO_CPPFLAGS="-I$(brew --prefix leptonica)/include -I$(brew --prefix tesseract)/include"
  # shellcheck disable=SC2155
  export CGO_LDFLAGS="-L$(brew --prefix leptonica)/lib -L$(brew --prefix tesseract)/lib -lleptonica -ltesseract"
fi

# Markdown
printf "\n==> Fixing markdown for any rules which support auto-fixing\n"
markdownlint-cli2 --fix "**/*.md"

printf "\n==> Linting markdown for any rules which don't support auto-fixing\n"
markdownlint-cli2 "**/*.md"

# Spellcheck
printf "\n==> Spellchecking\n"
vale --glob='!{.vale/**,.vale.ini,.claude/**,go.sum,go.mod,**/*.png,**/*.log}' .

# Go
printf "\n==> Linting Go\n"
golangci-lint run ./...

printf "\n==> Running Go tests\n"
go test ./...

# Build
printf "\n==> Building\n"
go build -o sudoku-solver ./cmd/sudoku-solver

# Smoke test
# Pick a random unsolved image from testdata/ and run the binary against it.
unsolved_count=$(find testdata -maxdepth 1 -name '*-unsolved.*' | wc -l | tr -d ' ')
if [ "$unsolved_count" -eq 0 ]; then
  echo "WARNING: no *-unsolved.* images found in testdata/, skipping smoke test"
else
  input=$(find testdata -maxdepth 1 -name '*-unsolved.*' | awk 'BEGIN{srand()} {lines[NR]=$0} END{print lines[int(rand()*NR)+1]}')
  printf "\n==> Smoke testing binary against %s" "$input\n"
  ./sudoku-solver -q "$input"
fi

printf "\n==> All checks passed\n"
