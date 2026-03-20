package xerrors_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errcontext"
	"github.com/wood-jp/xerrors/stacktrace"
)

type benchData struct {
	UserID    string
	RequestID string
}

// extendSink prevents the compiler from eliminating the Extend call.
// Extend has no observable side effects beyond the allocation, so without
// a package-level sink the result is dead-code-eliminated and reports 0 ns/op.
var extendSink error

func BenchmarkExtend(b *testing.B) {
	base := errors.New("base error")
	data := benchData{UserID: "123", RequestID: "abc"}
	b.ReportAllocs()
	for b.Loop() {
		extendSink = xerrors.Extend(data, base)
	}
}

func BenchmarkExtract_Shallow(b *testing.B) {
	base := errors.New("base error")
	err := xerrors.Extend(benchData{UserID: "123", RequestID: "abc"}, base)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = xerrors.Extract[benchData](err)
	}
}

func BenchmarkExtract_Deep(b *testing.B) {
	base := errors.New("base error")
	err := xerrors.Extend(benchData{UserID: "123", RequestID: "abc"}, base)
	for range 4 {
		err = xerrors.Extend(slog.String("extra", "layer"), err)
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = xerrors.Extract[benchData](err)
	}
}

func BenchmarkLog(b *testing.B) {
	base := errors.New("base error")
	err := stacktrace.Wrap(errcontext.Add(base, slog.String("user_id", "123")))
	b.ReportAllocs()
	for b.Loop() {
		_ = xerrors.Log(err)
	}
}
