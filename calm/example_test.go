package calm_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/calm"
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

func ExampleUnpanic() {
	var buf bytes.Buffer
	err := calm.Unpanic(func() error {
		panic("something went wrong")
	})
	newLogger(&buf).Error("recovered panic", xerrors.Log(err))
	fmt.Print(normalizeStack(buf.String()))
	// Output:
	// {"level":"ERROR","msg":"recovered panic","error":{"error":"panic: something went wrong","error_detail":{"stacktrace":[{"func":"github.com/wood-jp/xerrors/calm_test.ExampleUnpanic.func1","line":0,"source":"..."},{"func":"github.com/wood-jp/xerrors/calm.Unpanic","line":0,"source":"..."},{"func":"github.com/wood-jp/xerrors/calm_test.ExampleUnpanic","line":0,"source":"..."},{"func":"main.main","line":0,"source":"..."}],"class":"panic"}}}
}
