// Package xzap adapts xerrors log output for [go.uber.org/zap]. It lives in its
// own Go module so the core xerrors module stays free of a zap dependency.
package xzap

import (
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wood-jp/xerrors"
)

// Error returns a [zap.Field] with key "error" containing the same flat
// structure as [xerrors.Log]. A nil err yields [zap.Skip].
//
//	logger.Error("request failed", xzap.Error(err))
func Error(err error) zap.Field {
	if err == nil {
		return zap.Skip()
	}
	return zap.Object("error", attrs(xerrors.Log(err).Value.Group()))
}

// attrs encodes a slice of [slog.Attr] as a zap object.
type attrs []slog.Attr

// MarshalLogObject implements [zapcore.ObjectMarshaler].
func (a attrs) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, attr := range a {
		if err := addAttr(enc, attr); err != nil {
			return err
		}
	}
	return nil
}

func addAttr(enc zapcore.ObjectEncoder, attr slog.Attr) error {
	if attr.Equal(slog.Attr{}) {
		// Like slog handlers, ignore empty attrs.
		return nil
	}
	v := attr.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		enc.AddString(attr.Key, v.String())
	case slog.KindInt64:
		enc.AddInt64(attr.Key, v.Int64())
	case slog.KindUint64:
		enc.AddUint64(attr.Key, v.Uint64())
	case slog.KindFloat64:
		enc.AddFloat64(attr.Key, v.Float64())
	case slog.KindBool:
		enc.AddBool(attr.Key, v.Bool())
	case slog.KindDuration:
		enc.AddDuration(attr.Key, v.Duration())
	case slog.KindTime:
		enc.AddTime(attr.Key, v.Time())
	case slog.KindGroup:
		group := v.Group()
		if len(group) == 0 {
			return nil
		}
		if attr.Key == "" {
			// Per slog semantics, an empty-key group is inlined.
			return attrs(group).MarshalLogObject(enc)
		}
		return enc.AddObject(attr.Key, attrs(group))
	default:
		// KindAny; KindLogValuer cannot occur after Resolve.
		return addAny(enc, attr.Key, v.Any())
	}
	return nil
}

func addAny(enc zapcore.ObjectEncoder, key string, val any) error {
	switch t := val.(type) {
	case zapcore.ObjectMarshaler:
		return enc.AddObject(key, t)
	case error:
		enc.AddString(key, t.Error())
		return nil
	default:
		return enc.AddReflected(key, val)
	}
}
