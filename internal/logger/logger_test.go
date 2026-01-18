package logger

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		opts   Options
		logMsg string
		level  Level
		want   bool // should message appear in output
	}{
		{
			name:   "default options log warn",
			opts:   Options{},
			logMsg: "test warning",
			level:  LevelWarn,
			want:   true,
		},
		{
			name:   "default options skip debug",
			opts:   Options{},
			logMsg: "test debug",
			level:  LevelDebug,
			want:   false,
		},
		{
			name:   "debug level logs debug",
			opts:   Options{Level: LevelDebug},
			logMsg: "test debug",
			level:  LevelDebug,
			want:   true,
		},
		{
			name:   "error level skips warn",
			opts:   Options{Level: LevelError},
			logMsg: "test warning",
			level:  LevelWarn,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.opts.Output = &buf

			Init(tt.opts)

			switch tt.level {
			case LevelDebug:
				Debug(tt.logMsg)
			case LevelInfo:
				Info(tt.logMsg)
			case LevelWarn:
				Warn(tt.logMsg)
			case LevelError:
				Error(tt.logMsg)
			}

			got := strings.Contains(buf.String(), tt.logMsg)
			if got != tt.want {
				t.Errorf("Init() with level %v: message appeared = %v, want %v\nOutput: %s",
					tt.opts.Level, got, tt.want, buf.String())
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		check  func(string) bool
	}{
		{
			name:   "text format",
			format: FormatText,
			check: func(s string) bool {
				// Text format has key=value pairs
				return strings.Contains(s, "level=")
			},
		},
		{
			name:   "json format",
			format: FormatJSON,
			check: func(s string) bool {
				// JSON format has quoted keys
				return strings.Contains(s, `"level"`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Init(Options{
				Level:  LevelInfo,
				Format: tt.format,
				Output: &buf,
			})

			Info("test message")

			if !tt.check(buf.String()) {
				t.Errorf("Format %s: unexpected output: %s", tt.format, buf.String())
			}
		})
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	logger := With("component", "test")
	logger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "component") || !strings.Contains(output, "test") {
		t.Errorf("With() should add attributes: got %s", output)
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: &buf,
	})

	logger := WithGroup("request")
	logger.Info("test", "id", "123")

	output := buf.String()
	// In JSON format, group creates nested structure
	if !strings.Contains(output, "request") {
		t.Errorf("WithGroup() should create group: got %s", output)
	}
}

func TestContextLogging(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	// Create a logger with context-specific attributes
	ctxLogger := With("request_id", "abc123")
	ctx := NewContext(context.Background(), ctxLogger)

	InfoContext(ctx, "handling request")

	output := buf.String()
	if !strings.Contains(output, "request_id") || !strings.Contains(output, "abc123") {
		t.Errorf("Context logging should include context attributes: got %s", output)
	}
}

func TestFromContextDefault(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	// Use empty context - should fall back to default logger
	ctx := context.Background()
	InfoContext(ctx, "test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Error("FromContext with empty context should use default logger")
	}
}

func TestEnabled(t *testing.T) {
	tests := []struct {
		name        string
		configLevel Level
		checkLevel  Level
		want        bool
	}{
		{"debug enabled at debug", LevelDebug, LevelDebug, true},
		{"info enabled at debug", LevelDebug, LevelInfo, true},
		{"warn enabled at debug", LevelDebug, LevelWarn, true},
		{"debug disabled at info", LevelInfo, LevelDebug, false},
		{"info enabled at info", LevelInfo, LevelInfo, true},
		{"debug disabled at warn", LevelWarn, LevelDebug, false},
		{"info disabled at warn", LevelWarn, LevelInfo, false},
		{"warn enabled at warn", LevelWarn, LevelWarn, true},
		{"error enabled at warn", LevelWarn, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Init(Options{
				Level:  tt.configLevel,
				Output: &buf,
			})

			if got := Enabled(tt.checkLevel); got != tt.want {
				t.Errorf("Enabled(%v) = %v, want %v (config level: %v)",
					tt.checkLevel, got, tt.want, tt.configLevel)
			}
		})
	}
}

func TestHelpers(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	testErr := errors.New("test error")
	Logger().Info("test",
		Err(testErr),
		String("key", "value"),
		Int("count", 42),
		Bool("enabled", true),
		Any("data", map[string]string{"a": "b"}),
	)

	output := buf.String()
	checks := []string{"error", "test error", "key", "value", "count", "42", "enabled", "true"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Helper output should contain %q: got %s", check, output)
		}
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelError,
		Output: &buf,
	})

	// Warn should not appear at error level
	Warn("should not appear")
	if strings.Contains(buf.String(), "should not appear") {
		t.Error("Warn should not log at error level")
	}

	// Change level to warn
	buf.Reset()
	Init(Options{
		Level:  LevelWarn,
		Output: &buf,
	})

	Warn("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("Warn should log after SetLevel to warn")
	}
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil")
	}

	l.Info("direct logger call")
	if !strings.Contains(buf.String(), "direct logger call") {
		t.Error("Logger() should return working logger")
	}
}

func TestAllLogLevels(t *testing.T) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelDebug,
		Output: &buf,
	})

	Debug("debug msg", "key", "debug")
	Info("info msg", "key", "info")
	Warn("warn msg", "key", "warn")
	Error("error msg", "key", "error")

	output := buf.String()
	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR"} {
		if !strings.Contains(output, level) {
			t.Errorf("Output should contain %s level: got %s", level, output)
		}
	}
}

func BenchmarkLogging(b *testing.B) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelInfo,
		Output: &buf,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLoggingDisabled(b *testing.B) {
	var buf bytes.Buffer
	Init(Options{
		Level:  LevelError, // Debug is disabled
		Output: &buf,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("benchmark message", "iteration", i) // Should be very fast (no-op)
	}
}
