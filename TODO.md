# TODO

## To Do

- CLI helper text
- Accept `context.Context` and `log.Logger` in `solver`, `extractor`, and `renderer` packages
- Add wide event logging to `web/server.go` (method, path, status, duration)

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
