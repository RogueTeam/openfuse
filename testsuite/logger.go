//go:build openbsd

package testsuite

import (
	"log/slog"
	"testing"
)

func Logger(tb testing.TB, level slog.Leveler) (logger *slog.Logger) {
	return slog.New(slog.NewTextHandler(tb.Output(), &slog.HandlerOptions{Level: level})).WithGroup(tb.Name())
}
