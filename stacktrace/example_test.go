package stacktrace_test

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/stacktrace"
)

func newLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
}

var (
	reSource = regexp.MustCompile(`"source":"[^"]*"`)
	reLine   = regexp.MustCompile(`"line":\d+`)
)

func normalizeStack(s string) string {
	s = reSource.ReplaceAllString(s, `"source":"..."`)
	s = reLine.ReplaceAllString(s, `"line":0`)
	return s
}

func ExampleWrap() {
	var buf bytes.Buffer
	err := stacktrace.Wrap(errors.New("something failed"))
	newLogger(&buf).Error("operation failed", xerrors.Log(err))
	fmt.Print(normalizeStack(buf.String()))
	// Output:
	// {"level":"ERROR","msg":"operation failed","error":{"error":"something failed","error_detail":{"stacktrace":[{"func":"github.com/wood-jp/xerrors/stacktrace_test.ExampleWrap","line":0,"source":"..."},{"func":"main.main","line":0,"source":"..."}]}}}
}

func ExampleExtract_noStack() {
	err := errors.New("plain error")
	fmt.Println(stacktrace.Extract(err) == nil)
	// Output:
	// true
}
