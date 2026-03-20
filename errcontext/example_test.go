package errcontext_test

import (
	"errors"
	"log/slog"
	"os"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errcontext"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func ExampleAdd() {
	err := errors.New("request failed")
	err = errcontext.Add(err,
		slog.String("user_id", "u123"),
		slog.Int("status", 503),
	)
	newLogger().Error("handler error", xerrors.Log(err))
	// Output:
	// {"level":"ERROR","msg":"handler error","error":{"error":"request failed","error_detail":{"context":{"status":503,"user_id":"u123"}}}}
}
