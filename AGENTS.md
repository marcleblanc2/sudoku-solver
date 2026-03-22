# AGENTS.md

## Project Overview

- A standalone Sudoku solver
- Must feel magical to the human user, meaning simple, minimalist, intuitive, clear, and polished

## Usage

- Flow
  - Accept an image (screenshot) of an unsolved Sudoku puzzle
  - Extract the puzzle grid from the image (OCR / image processing)
  - Solve the puzzle algorithmically
  - Render the solved puzzle as an output image

## Agent behaviour

- Default to conversation, instead of writing code
  - Propose solutions, and ask the user to pick a solution, before writing code
- Keep AGENTS.md and README.md up to date, and deduplicated
  - AGENTS.md for agents
  - README.md for humans
- Enforce security best practices
- Before informing the user that a change is completed:
  - Build, test, and execute the binary to verify the change works
  - Update code documentation
- Prompt the user to accept the change
  - If the user accepts the change, then update the TODO.md file, moving the task from the ## To Do section to the bottom of the ## Done section

## Tech Stack

- Languages:
  - Go for CLI / backend
  - Go `html/template` + vanilla HTML/CSS/JS for web UI (no frontend framework or build step)
- Distribution:
  - Go binary, which functions as a CLI
  - Docker image, which includes a web UI
    - Web UI is served by a Go `net/http` server using `html/template`
    - Web UI calls the CLI
  - `docker-compose.yaml` — pulls pre-built image from ghcr.io (for end users)
  - `docker-compose.build.yaml` — builds from source (for contributors)
- Container environment
  - Use Podman instead of Docker

## Directory Structure

```text
sudoku-solver/
├── AGENTS.md / README.md / TODO.md
├── Dockerfile
├── go.mod
├── cmd/sudoku-solver/main.go    # CLI entrypoint
├── internal/
│   ├── solver/                  # puzzle solving algorithm
│   ├── extractor/               # image → 9x9 grid (OCR)
│   ├── renderer/                # solved grid → output image
│   └── web/                     # HTTP server, templates, static assets
└── testdata/                    # test fixtures (sample puzzle images)
```

## Conventions

- Write idiomatic Go (gofmt, effective Go style)
- Keep the binary self-contained with minimal dependencies
- Ensure the app works both as a standalone binary and inside a Docker container
- Use OpenTelemetry for instrumentation and observability
- Use the OTel Logs SDK directly for all logging (not `log/slog`)
- Always prefer OTel SDK primitives over custom implementations; do not build custom wrappers when the SDK already provides the capability
- Wide events are implemented as OTel spans, not custom accumulators:
  - Create a span at the start of execution with `tracer.Start(ctx, "name")`
  - Add attributes throughout execution with `span.SetAttributes(...)`
  - Defer `span.End()` so the span is emitted on exit
  - Pass `ctx` through the call stack; any package can enrich the span via `trace.SpanFromContext(ctx).SetAttributes(...)`
- `internal/logging.Setup` returns both a `LoggerProvider` and a `TracerProvider`
- Default logging to both stderr and `sudoku-solver.log` in the current directory
- `-q` / `--quiet` flag disables stderr logging; `LOG_FILE=""` disables file logging

## Key Goals

- The solver should be understandable — code should be clear enough that a user can follow the solving logic and learn from it
- Prioritise simplicity, readability, and maintainability over cleverness
