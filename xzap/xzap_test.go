package xzap_test

import (
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errcontext"
	"github.com/wood-jp/xerrors/stacktrace"
	"github.com/wood-jp/xerrors/xzap"
)

var errTest = errors.New("boom")

func encode(f zap.Field) map[string]any {
	enc := zapcore.NewMapObjectEncoder()
	f.AddTo(enc)
	return enc.Fields
}

// detail returns the "error_detail" object from an encoded xzap.Error field.
func detail(t *testing.T, err error) map[string]any {
	t.Helper()
	top, ok := encode(xzap.Error(err))["error"].(map[string]any)
	if !ok {
		t.Fatal("missing top-level error object")
	}
	d, ok := top["error_detail"].(map[string]any)
	if !ok {
		t.Fatalf("missing error_detail: %v", top)
	}
	return d
}

// attrData is extended-error data whose log value is an arbitrary group.
type attrData []slog.Attr

func (a attrData) LogValue() slog.Value { return slog.GroupValue(a...) }

// resolvable is a LogValuer that resolves to a string.
type resolvable struct{}

func (resolvable) LogValue() slog.Value { return slog.StringValue("resolved") }

// marshaler is a zapcore.ObjectMarshaler.
type marshaler struct{ err error }

func (m marshaler) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("inner", "value")
	return m.err
}

func TestErrorNil(t *testing.T) {
	t.Parallel()
	if got := encode(xzap.Error(nil)); len(got) != 0 {
		t.Fatalf("expected no fields, got %v", got)
	}
}

func TestErrorPlain(t *testing.T) {
	t.Parallel()
	got := encode(xzap.Error(errTest))
	want := map[string]any{"error": map[string]any{"error": "boom"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestErrorWrapped(t *testing.T) {
	t.Parallel()
	err := errcontext.Add(errTest, slog.String("user_id", "123"), slog.Int("attempt", 3))
	err = xerrors.Extend(errclass.Transient, err)
	err = stacktrace.Wrap(err)

	d := detail(t, err)
	if d["class"] != "transient" {
		t.Errorf("class = %v", d["class"])
	}
	wantCtx := map[string]any{"user_id": "123", "attempt": int64(3)}
	if !reflect.DeepEqual(d["context"], wantCtx) {
		t.Errorf("context = %v, want %v", d["context"], wantCtx)
	}
	if d["stacktrace"] == nil {
		t.Error("missing stacktrace")
	}
}

func TestErrorDataFallback(t *testing.T) {
	t.Parallel()
	// Data that is not a group falls back to a single "data" key.
	d := detail(t, xerrors.Extend(42, errTest))
	if !reflect.DeepEqual(d, map[string]any{"data": int64(42)}) {
		t.Fatalf("got %v", d)
	}
}

func TestErrorAttrKinds(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name string
		attr slog.Attr
		want map[string]any
	}{
		{"string", slog.String("k", "v"), map[string]any{"k": "v"}},
		{"int64", slog.Int64("k", -7), map[string]any{"k": int64(-7)}},
		{"uint64", slog.Uint64("k", 7), map[string]any{"k": uint64(7)}},
		{"float64", slog.Float64("k", 1.5), map[string]any{"k": 1.5}},
		{"bool", slog.Bool("k", true), map[string]any{"k": true}},
		{"duration", slog.Duration("k", time.Second), map[string]any{"k": time.Second}},
		{"time", slog.Time("k", now), map[string]any{"k": now}},
		{"logvaluer", slog.Any("k", resolvable{}), map[string]any{"k": "resolved"}},
		{"error", slog.Any("k", errTest), map[string]any{"k": "boom"}},
		{"reflected", slog.Any("k", []int{1, 2}), map[string]any{"k": []int{1, 2}}},
		{
			"object marshaler",
			slog.Any("k", marshaler{}),
			map[string]any{"k": map[string]any{"inner": "value"}},
		},
		{
			"group",
			slog.Group("k", slog.String("a", "b"), slog.Int("c", 1)),
			map[string]any{"k": map[string]any{"a": "b", "c": int64(1)}},
		},
		{
			"nested group",
			slog.Group("k", slog.Group("n", slog.Bool("x", true))),
			map[string]any{"k": map[string]any{"n": map[string]any{"x": true}}},
		},
		{
			"inlined group",
			slog.Group("", slog.String("a", "b")),
			map[string]any{"a": "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := detail(t, xerrors.Extend(attrData{tt.attr}, errTest))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestErrorEmptyAttrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr slog.Attr
		// want is the expected error_detail; nil means it is omitted entirely.
		want map[string]any
	}{
		// slog drops empty groups when building a group value.
		{"empty group", slog.Group("k"), nil},
		// An empty attr is ignored by the encoder.
		{"empty attr", slog.Attr{}, map[string]any{}},
		// A LogValuer resolving to an empty group is only seen at encode time.
		{"empty resolved group", slog.Any("k", attrData{}), map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			top, ok := encode(xzap.Error(xerrors.Extend(attrData{tt.attr}, errTest)))["error"].(map[string]any)
			if !ok {
				t.Fatal("missing top-level error object")
			}
			got, present := top["error_detail"]
			if tt.want == nil {
				if present {
					t.Fatalf("expected no error_detail, got %v", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestErrorMarshalFailure(t *testing.T) {
	t.Parallel()

	// A failing nested marshaler must propagate to zap, which reports it as
	// an "<key>Error" field instead of dropping the log line.
	failing := marshaler{err: errors.New("marshal failed")}
	tests := []struct {
		name string
		data attrData
	}{
		{"object marshaler", attrData{slog.Any("m", failing)}},
		{"group", attrData{slog.Group("g", slog.Any("m", failing))}},
		{"inlined group", attrData{slog.Group("", slog.Any("m", failing))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := encode(xzap.Error(xerrors.Extend(tt.data, errTest)))
			if got["errorError"] != "marshal failed" {
				t.Fatalf("expected errorError field, got %v", got)
			}
		})
	}
}

func TestErrorWithLogger(t *testing.T) {
	t.Parallel()
	core, logs := observer.New(zap.ErrorLevel)
	zap.New(core).Error("failed", xzap.Error(errTest))
	if logs.Len() != 1 {
		t.Fatalf("logs = %d, want 1", logs.Len())
	}
	if _, ok := logs.All()[0].ContextMap()["error"]; !ok {
		t.Error("missing error field")
	}
}
