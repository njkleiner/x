package log

import (
	"context"
	"log/slog"
	"os"
)

func ExampleContext() {
	// create a [*log/slog.Logger] which prints
	// JSON-formatted messages to [os.Stderr].
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// create new a [context.Context] with a
	// given [*log/slog.Logger] associated.
	ctx := Context(context.Background(), log)

	// print a [log/slog.LevelInfo] message using the [*log/slog.Logger]
	// associated with ctx, if any, else default to [log/slog.Default].
	Info(ctx, "hello world", slog.String("foo", "bar"))
}
