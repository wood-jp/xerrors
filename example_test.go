package xerrors_test

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/wood-jp/xerrors"
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

func ExampleLog() {
	type RequestInfo struct {
		UserID string
		Path   string
	}

	err := xerrors.Extend(RequestInfo{UserID: "u123", Path: "/api/items"}, errors.New("unauthorized"))
	newLogger().Error("request failed", xerrors.Log(err))
	// Output:
	// {"level":"ERROR","msg":"request failed","error":{"error":"unauthorized","error_detail":{"data":{"UserID":"u123","Path":"/api/items"}}}}
}

func ExampleExtract() {
	err := errors.New("database error")
	err = xerrors.Extend(503, err)

	code, ok := xerrors.Extract[int](err)
	fmt.Println(ok, code)
	// Output:
	// true 503
}
