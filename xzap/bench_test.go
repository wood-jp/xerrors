package xzap_test

import (
	"errors"
	"log/slog"
	"testing"

	"go.uber.org/zap/zapcore"

	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errcontext"
	"github.com/wood-jp/xerrors/xzap"
)

func BenchmarkError_Plain(b *testing.B) {
	err := errors.New("test error")
	b.ReportAllocs()
	for b.Loop() {
		enc := zapcore.NewMapObjectEncoder()
		xzap.Error(err).AddTo(enc)
	}
}

func BenchmarkError_Extended(b *testing.B) {
	err := errcontext.Add(errors.New("test error"), slog.String("user_id", "u123"), slog.Int("attempt", 3))
	err = errclass.WrapAs(err, errclass.Transient)
	b.ReportAllocs()
	for b.Loop() {
		enc := zapcore.NewMapObjectEncoder()
		xzap.Error(err).AddTo(enc)
	}
}

