package xzap_test

import (
	"errors"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wood-jp/xerrors/errcontext"
	"github.com/wood-jp/xerrors/xzap"
)

func ExampleError() {
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = ""
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(os.Stdout), zap.ErrorLevel))

	err := errcontext.Add(errors.New("request failed"),
		slog.String("user_id", "u123"),
		slog.Int("status", 503),
	)
	logger.Error("handler error", xzap.Error(err))
	// Output:
	// {"level":"error","msg":"handler error","error":{"error":"request failed","error_detail":{"context":{"status":503,"user_id":"u123"}}}}
}
