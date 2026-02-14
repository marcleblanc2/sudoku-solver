// Package logging configures OpenTelemetry log export to stderr and/or a file.
package logging

import (
	"context"
	"fmt"
	"os"
	"strings"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
)

// Config controls where and at what level logs are emitted.
type Config struct {
	Level   string // debug, info, warn, error
	Quiet   bool   // suppress stderr output
	LogFile string // path to log file; empty disables file logging
}

// Setup creates a LoggerProvider with stderr and/or file exporters, filtered
// to the configured minimum severity. It returns the provider and a cleanup
// function that shuts everything down.
func Setup(ctx context.Context, cfg Config) (*sdklog.LoggerProvider, func(), error) {
	minSev := parseLevel(cfg.Level)

	var processors []sdklog.LoggerProviderOption
	var closers []func()

	if !cfg.Quiet {
		exp, err := stdoutlog.New(stdoutlog.WithWriter(os.Stderr))
		if err != nil {
			return nil, nil, fmt.Errorf("stderr exporter: %w", err)
		}
		proc := sdklog.NewSimpleProcessor(exp)
		processors = append(processors, sdklog.WithProcessor(&filterProcessor{
			inner:  proc,
			minSev: minSev,
		}))
	}

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		closers = append(closers, func() { f.Close() })

		exp, err := stdoutlog.New(stdoutlog.WithWriter(f))
		if err != nil {
			return nil, nil, fmt.Errorf("file exporter: %w", err)
		}
		proc := sdklog.NewSimpleProcessor(exp)
		processors = append(processors, sdklog.WithProcessor(&filterProcessor{
			inner:  proc,
			minSev: minSev,
		}))
	}

	provider := sdklog.NewLoggerProvider(processors...)

	cleanup := func() {
		_ = provider.Shutdown(ctx)
		for _, c := range closers {
			c()
		}
	}

	return provider, cleanup, nil
}

// parseLevel maps a string to an OTel log severity, defaulting to Info.
func parseLevel(s string) otellog.Severity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return otellog.SeverityDebug
	case "info":
		return otellog.SeverityInfo
	case "warn":
		return otellog.SeverityWarn
	case "error":
		return otellog.SeverityError
	default:
		return otellog.SeverityInfo
	}
}

// filterProcessor wraps an sdklog.Processor and drops records below minSev.
type filterProcessor struct {
	inner  sdklog.Processor
	minSev otellog.Severity
}

func (f *filterProcessor) OnEmit(ctx context.Context, rec *sdklog.Record) error {
	if rec.Severity() < f.minSev {
		return nil
	}
	return f.inner.OnEmit(ctx, rec)
}

func (f *filterProcessor) Enabled(ctx context.Context, param sdklog.EnabledParameters) bool {
	return f.inner.Enabled(ctx, param)
}

func (f *filterProcessor) Shutdown(ctx context.Context) error {
	return f.inner.Shutdown(ctx)
}

func (f *filterProcessor) ForceFlush(ctx context.Context) error {
	return f.inner.ForceFlush(ctx)
}
