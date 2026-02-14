# sudoku-solver

- Solves Sudoku puzzles
- Written in Go
- Runs as a standalone binary
- Released as a Go binary, and a Docker image

## Configure

Configure via environment variables:

| Variable    | Description                            | Default              |
|-------------|----------------------------------------|----------------------|
| `PORT`      | HTTP listening port                    | `8080`               |
| `LOG_LEVEL` | Log level (debug, info, warn, error)   | `info`               |
| `LOG_FILE`  | Log file path (set to `""` to disable) | `sudoku-solver.log`  |

CLI flags:

| Flag            | Description               |
|-----------------|---------------------------|
| `-q`, `--quiet` | Disable logging to stderr |

## Run with Podman / Docker

Pull and run the pre-built image:

```sh
podman compose up -d
```

## Contributing

### Setup

Install the following before contributing:

| Dependency                                                           | Install                          | Purpose                         |
|----------------------------------------------------------------------|----------------------------------|---------------------------------|
| [Go](https://go.dev/) 1.23+                                          | `brew install go`                | Build and test                  |
| [Podman](https://podman.io/)                                         | `brew install podman`            | Container builds and runs       |
| [golangci-lint](https://golangci-lint.run/)                          | `brew install golangci-lint`     | Go linting                      |
| [vale](https://vale.sh/)                                             | `brew install vale`              | Markdown prose linting          |
| [markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2) | `brew install markdownlint-cli2` | Markdown formatting and linting |

After cloning, run `vale sync` to download vale style packages.

### Build

Build the CLI binary:

```sh
go build -o sudoku-solver ./cmd/sudoku-solver
```

Build and run the Docker image:

```sh
podman compose -f docker-compose.build.yaml up -d --build
```

### Lint

```sh
golangci-lint run ./...
vale *.md
markdownlint-cli2 "**/*.md"
markdownlint-cli2 --fix "**/*.md"
```

### Test

```sh
go test ./...
```

### Logging

- Logging uses the OpenTelemetry Logs SDK directly
- By default, logs are written to both stderr and `sudoku-solver.log`
- Use `-q` / `--quiet` to disable stderr output
- Set `LOG_FILE=""` to disable file logging

## Purpose

- To test / validate user experience of cutting edge AI dev tools, hopefully on mobile, without having to sit at my desk, or use a computer
- To understand the code, in a way that I can follow, learn from, and repeat myself manually when playing without running this app
- To learn and practice current best practices, especially around observability

## Inputs

- Screenshot of an unsolved Sudoku puzzle

## Outputs

- Image of the solved Sudoku puzzle

## Project Structure

```text
cmd/sudoku-solver/   CLI entrypoint
internal/solver/     puzzle solving algorithm
internal/extractor/  image → grid (OCR / image processing)
internal/renderer/   solved grid → output image
internal/web/        web UI (Go html/template + static assets)
testdata/            sample puzzle images for testing
```
