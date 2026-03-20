package errcontext_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errcontext"
)

func BenchmarkAdd_New(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = errcontext.Add(errors.New("base error"), slog.String("user_id", "123"), slog.Int("attempt", 3))
	}
}

func BenchmarkAdd_Existing(b *testing.B) {
	base := errcontext.Add(errors.New("base error"), slog.String("user_id", "123"))
	b.ReportAllocs()
	for b.Loop() {
		_ = errcontext.Add(base, slog.String("request_id", "abc"))
	}
}

// BenchmarkAdd_Existing_Deep measures the in-place mutation path when the
// existing Context is buried 5 ExtendedError layers deep, exercising the
// errors.AsType chain walk in Get.
func BenchmarkAdd_Existing_Deep(b *testing.B) {
	base := errcontext.Add(errors.New("base error"), slog.String("user_id", "123"))
	for range 5 {
		base = xerrors.Extend(struct{}{}, base)
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = errcontext.Add(base, slog.String("request_id", "abc"))
	}
}

func BenchmarkFlatten(b *testing.B) {
	base := errcontext.Add(
		errors.New("base error"),
		slog.String("user_id", "123"),
		slog.String("request_id", "abc"),
		slog.Int("attempt", 3),
		slog.String("service", "api"),
		slog.String("region", "us-east-1"),
	)
	ctx := errcontext.Get(base)
	b.ReportAllocs()
	for b.Loop() {
		_ = ctx.Flatten()
	}
}
