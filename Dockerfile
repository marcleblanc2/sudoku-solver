# Stage 1: Build the Go binary using Chainguard's Go image.
FROM cgr.dev/chainguard/go:latest AS builder

WORKDIR /app
COPY . .

RUN go build -o sudoku-solver ./cmd/sudoku-solver

# Stage 2: Create a minimal, zero-CVE runtime image using Chainguard's static image.
FROM cgr.dev/chainguard/static:latest

COPY --from=builder /app/sudoku-solver /sudoku-solver

ENTRYPOINT ["/sudoku-solver"]
