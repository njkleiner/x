package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestContext(t *testing.T) {
	timeNow = func() time.Time {
		// NOTE: override the "current" timestamp with a static value during testing
		// to be able to compare the actual against the expected log message output.
		return time.Date(2025, time.December, 25, 12, 28, 19, 0, time.UTC)
	}

	t.Cleanup(func() { timeNow = time.Now })

	t.Run("default logger", func(t *testing.T) {
		var buf bytes.Buffer

		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

		// NOTE: we never associate any specific [*log/slog.Logger] with ctx,
		// therefore we expect the default logger to be "retrieved" from it.
		ctx := context.Background()

		Info(ctx, "hello world", slog.Int("answer", 42))

		want := `{"time":"2025-12-25T12:28:19Z","level":"INFO","msg":"hello world","answer":42}`

		if got := strings.TrimSpace(buf.String()); got != want {
			t.Errorf("invalid message: got=%q want=%q", got, want)
		}
	})

	t.Run("context logger", func(t *testing.T) {
		slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))

		ctx := context.Background()

		var buf bytes.Buffer

		ctx = Context(ctx, slog.New(slog.NewJSONHandler(&buf, nil)))

		Info(ctx, "hello world", slog.Int("answer", 42))

		want := `{"time":"2025-12-25T12:28:19Z","level":"INFO","msg":"hello world","answer":42}`

		if got := strings.TrimSpace(buf.String()); got != want {
			t.Errorf("invalid message: got=%q want=%q", got, want)
		}
	})
}

func TestWith(t *testing.T) {
	timeNow = func() time.Time {
		// NOTE: override the "current" timestamp with a static value during testing
		// to be able to compare the actual against the expected log message output.
		return time.Date(2025, time.December, 25, 12, 28, 19, 0, time.UTC)
	}

	t.Cleanup(func() { timeNow = time.Now })

	var buf bytes.Buffer

	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	// NOTE: we never explicitly associate a non-default logger with ctx,
	// but we implicitly do so when adding attributes via calling [With].
	ctx := context.Background()

	ctx = With(ctx, slog.Int("answer", 42))

	// NOTE: we do not actually provide any attributes in this call to [Info].
	// Therefore, all attributes actually present in the record come from ctx.
	Info(ctx, "hello world")

	want := `{"time":"2025-12-25T12:28:19Z","level":"INFO","msg":"hello world","answer":42}`

	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("invalid message: got=%q want=%q", got, want)
	}
}

func TestHandle(t *testing.T) {
	timeNow = func() time.Time {
		// NOTE: override the "current" timestamp with a static value during testing
		// to be able to compare the actual against the expected log message output.
		return time.Date(2025, time.December, 25, 12, 28, 19, 0, time.UTC)
	}

	t.Cleanup(func() { timeNow = time.Now })

	tests := []struct {
		name  string
		print func(ctx context.Context, msg string, args ...any)

		options *slog.HandlerOptions

		want string
	}{
		{
			name:  "discard",
			print: Debug,

			options: &slog.HandlerOptions{Level: slog.LevelInfo},

			want: ``,
		},
		{
			name:  "debug",
			print: Debug,

			options: &slog.HandlerOptions{Level: slog.LevelDebug},

			want: `{"time":"2025-12-25T12:28:19Z","level":"DEBUG","msg":"hello world","answer":42}`,
		},
		{
			name:  "info",
			print: Info,

			options: &slog.HandlerOptions{Level: slog.LevelDebug},

			want: `{"time":"2025-12-25T12:28:19Z","level":"INFO","msg":"hello world","answer":42}`,
		},
		{
			name:  "warn",
			print: Warn,

			options: &slog.HandlerOptions{Level: slog.LevelDebug},

			want: `{"time":"2025-12-25T12:28:19Z","level":"WARN","msg":"hello world","answer":42}`,
		},
		{
			name:  "error",
			print: Error,

			options: &slog.HandlerOptions{Level: slog.LevelDebug},

			want: `{"time":"2025-12-25T12:28:19Z","level":"ERROR","msg":"hello world","answer":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, tt.options)))

			ctx := context.Background()

			tt.print(ctx, "hello world", slog.Int("answer", 42))

			if got := strings.TrimSpace(buf.String()); got != tt.want {
				t.Errorf("invalid message: got=%q want=%q", got, tt.want)
			}
		})
	}
}
