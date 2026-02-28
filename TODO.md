# TODO

## To Do

- internal/solver/ — Pure algorithm, no external dependencies. Takes a 9x9 grid, returns the solved grid
- internal/renderer/ — Solved grid → output image. Needs an image generation library.
- Accept `context.Context` in `solver`, `extractor`, and `renderer` packages; enrich span via `trace.SpanFromContext(ctx)`
- Add wide event logging to `web/server.go` (method, path, status, duration, all client headers, client ip, etc.)
- Add OTel MeterProvider to `logging.Setup` (stdout exporter, same stderr/file pattern as logs and traces)
- Add OTel metrics to web server: request rate, latency histogram, error counter

## Done

- Chose Go `html/template` for web UI
- Set up project directory structure (`cmd/`, `internal/`, `testdata/`)
- Created `docker-compose.yaml` (pull from ghcr.io) and `docker-compose.build.yaml` (build from source)
- Added `PORT` and `LOG_LEVEL` environment variable configuration
- Chose OTel Logs SDK directly for all logging
- Go build verified
- Docker build verified
- Switched to Chainguard base images for zero CVEs
- Implemented OTel logging in `internal/logging` with stderr + file exporters, severity filtering, `-q`/`--quiet` flag, and `LOG_FILE` env var
- Added CLI helper text with `-i`/`--input` flag, positional arg support, and usage examples
- Implemented wide events using OTel spans (replaced custom `internal/event` with `trace.SpanFromContext`)
- Implemented grid detection in `internal/extractor` (decode, grayscale, threshold, bounding box, cell splitting) using pure Go stdlib
- Implemented digit recognition using Tesseract OCR (`gosseract`), verified all 81 cells against known expected values
