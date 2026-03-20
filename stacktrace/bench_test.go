package stacktrace_test

import (
	"errors"
	"testing"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/stacktrace"
)

//go:noinline
func wrapDepth1(err error) error { return stacktrace.Wrap(err) }

//go:noinline
func wrapDepth2(err error) error { return wrapDepth1(err) }

//go:noinline
func wrapDepth3(err error) error { return wrapDepth2(err) }

//go:noinline
func wrapDepth4(err error) error { return wrapDepth3(err) }

//go:noinline
func wrapDepth5(err error) error { return wrapDepth4(err) }

func BenchmarkWrap_New(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = stacktrace.Wrap(errors.New("test error"))
	}
}

func BenchmarkWrap_Existing(b *testing.B) {
	base := stacktrace.Wrap(errors.New("test error"))
	b.ReportAllocs()
	for b.Loop() {
		_ = stacktrace.Wrap(base)
	}
}

// BenchmarkWrap_New_Deep measures Wrap when called 5 frames deep, so GetStack
// captures a longer stack than BenchmarkWrap_New.
func BenchmarkWrap_New_Deep(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = wrapDepth5(errors.New("test error"))
	}
}

// BenchmarkWrap_Existing_Deep measures the no-op path when the existing
// StackTrace is buried 5 ExtendedError layers deep, exercising the
// errors.AsType chain walk in Extract.
func BenchmarkWrap_Existing_Deep(b *testing.B) {
	base := stacktrace.Wrap(errors.New("test error"))
	for range 5 {
		base = xerrors.Extend(struct{}{}, base)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = stacktrace.Wrap(base)
	}
}
