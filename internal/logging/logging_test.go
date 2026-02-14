package logging

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

func TestSetup_DefaultConfig(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	provider, cleanup, err := Setup(context.Background(), Config{
		Level:   "info",
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}

	if _, err := os.Stat(logFile); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestSetup_QuietMode(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	provider, cleanup, err := Setup(context.Background(), Config{
		Level:   "info",
		Quiet:   true,
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestSetup_NoFileLogging(t *testing.T) {
	provider, cleanup, err := Setup(context.Background(), Config{
		Level:   "info",
		Quiet:   true,
		LogFile: "",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()

	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestSetup_InvalidLogFilePath(t *testing.T) {
	_, _, err := Setup(context.Background(), Config{
		Level:   "info",
		Quiet:   true,
		LogFile: "/no/such/directory/test.log",
	})
	if err == nil {
		t.Fatal("expected error for invalid log file path")
	}
}

func TestSetup_FileReceivesLogs(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	provider, cleanup, err := Setup(context.Background(), Config{
		Level:   "info",
		Quiet:   true,
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	logger := provider.Logger("test")
	var rec otellog.Record
	rec.SetSeverity(otellog.SeverityInfo)
	rec.SetBody(otellog.StringValue("hello from test"))
	logger.Emit(context.Background(), rec)

	cleanup()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected log file to contain data")
	}
}

func TestSetup_SeverityFiltering(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	provider, cleanup, err := Setup(context.Background(), Config{
		Level:   "error",
		Quiet:   true,
		LogFile: logFile,
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	logger := provider.Logger("test")

	// Emit an info record — should be filtered out.
	var infoRec otellog.Record
	infoRec.SetSeverity(otellog.SeverityInfo)
	infoRec.SetBody(otellog.StringValue("should be filtered"))
	logger.Emit(context.Background(), infoRec)

	cleanup()

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty log file, got: %s", data)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  otellog.Severity
	}{
		{"debug", otellog.SeverityDebug},
		{"DEBUG", otellog.SeverityDebug},
		{"info", otellog.SeverityInfo},
		{"warn", otellog.SeverityWarn},
		{"error", otellog.SeverityError},
		{"", otellog.SeverityInfo},
		{"unknown", otellog.SeverityInfo},
		{"  Info  ", otellog.SeverityInfo},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterProcessor_DropsBelow(t *testing.T) {
	inner := &recordingProcessor{}
	fp := &filterProcessor{inner: inner, minSev: otellog.SeverityWarn}

	var rec sdklog.Record
	rec.SetSeverity(otellog.SeverityInfo)
	_ = fp.OnEmit(context.Background(), &rec)

	if inner.count != 0 {
		t.Fatalf("expected 0 records, got %d", inner.count)
	}
}

func TestFilterProcessor_PassesAtOrAbove(t *testing.T) {
	inner := &recordingProcessor{}
	fp := &filterProcessor{inner: inner, minSev: otellog.SeverityWarn}

	var warnRec sdklog.Record
	warnRec.SetSeverity(otellog.SeverityWarn)
	_ = fp.OnEmit(context.Background(), &warnRec)

	var errorRec sdklog.Record
	errorRec.SetSeverity(otellog.SeverityError)
	_ = fp.OnEmit(context.Background(), &errorRec)

	if inner.count != 2 {
		t.Fatalf("expected 2 records, got %d", inner.count)
	}
}

// recordingProcessor counts OnEmit calls for testing.
type recordingProcessor struct {
	count int
}

func (r *recordingProcessor) OnEmit(_ context.Context, _ *sdklog.Record) error {
	r.count++
	return nil
}

func (r *recordingProcessor) Enabled(_ context.Context, _ sdklog.EnabledParameters) bool {
	return true
}

func (r *recordingProcessor) Shutdown(_ context.Context) error { return nil }
func (r *recordingProcessor) ForceFlush(_ context.Context) error { return nil }
