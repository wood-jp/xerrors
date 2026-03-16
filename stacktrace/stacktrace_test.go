package stacktrace_test

import (
	"bytes"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/wood-jp/xerrors/stacktrace"
)

func TestStackTraceLogValue(t *testing.T) {
	t.Parallel()

	err := stacktrace.Wrap(errors.New("test error"))
	withFrames := stacktrace.Extract(err)
	if withFrames == nil {
		t.Fatal("expected stacktrace")
	}

	tests := []struct {
		name          string
		st            stacktrace.StackTrace
		wantKind      slog.Kind
		wantLogOutput bool
	}{
		{"empty", nil, slog.KindAny, false},
		{"with frames", withFrames, slog.KindAny, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.st.LogValue().Kind(); got != tt.wantKind {
				t.Errorf("LogValue().Kind() = %v, want %v", got, tt.wantKind)
			}
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			logger.Info("test", slog.Any("st", tt.st))
			if tt.wantLogOutput && buf.Len() == 0 {
				t.Error("expected log output")
			}
		})
	}
}

// stackDuringPanic captures a stack trace from inside a deferred function during
// panic unwinding. This places runtime.gopanic (defined in src/runtime/panic.go)
// on the call stack, giving us a runtime .go frame to exercise the runtime-package
// skip branch of GetStack.
func stackDuringPanic(skipRuntime bool) (st stacktrace.StackTrace) {
	defer func() {
		st = stacktrace.GetStack(1, skipRuntime)
		_ = recover()
	}()
	panic("stack capture")
}

func TestGetStack_SkipRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		skipRuntime bool
		wantRuntime bool // runtime.gopanic (panic.go) should appear
		wantTesting bool // testing.tRunner (testing.go) should appear
	}{
		{"skipRuntime=false includes runtime and testing frames", false, true, true},
		{"skipRuntime=true excludes runtime and testing frames", true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stack := stackDuringPanic(tt.skipRuntime)
			if len(stack) == 0 {
				t.Fatal("expected non-empty stack trace")
			}

			hasRuntime := false
			hasTesting := false
			for _, frame := range stack {
				if strings.HasPrefix(frame.Function, "runtime.") {
					hasRuntime = true
				}
				if strings.HasPrefix(frame.Function, "testing.") {
					hasTesting = true
				}
			}
			if hasRuntime != tt.wantRuntime {
				t.Errorf("runtime frames present = %v, want %v", hasRuntime, tt.wantRuntime)
			}
			if hasTesting != tt.wantTesting {
				t.Errorf("testing frames present = %v, want %v", hasTesting, tt.wantTesting)
			}
		})
	}
}

func TestGetStack_SkipValues(t *testing.T) {
	t.Parallel()
	stack0 := stacktrace.GetStack(0, true)
	stack1 := stacktrace.GetStack(1, true)
	stack2 := stacktrace.GetStack(2, true)

	if len(stack0) == 0 {
		t.Fatal("expected non-empty stack0")
	}
	if len(stack0) < len(stack1) {
		t.Errorf("stack0 (%d frames) should be >= stack1 (%d frames)", len(stack0), len(stack1))
	}
	if len(stack1) < len(stack2) {
		t.Errorf("stack1 (%d frames) should be >= stack2 (%d frames)", len(stack1), len(stack2))
	}
}

func TestGetStack_HighSkipValue(t *testing.T) {
	t.Parallel()
	stack := stacktrace.GetStack(1000, true)
	if len(stack) > 5 {
		t.Errorf("expected short or empty stack with high skip value, got %d frames", len(stack))
	}
}

func TestStackTraceFlatLogAttrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		st       stacktrace.StackTrace
		wantNil  bool
		wantKey  string
	}{
		{"nil stacktrace returns one attr", nil, false, "stacktrace"},
		{"with frames returns one attr", stacktrace.GetStack(1, true), false, "stacktrace"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			attrs := tt.st.FlatLogAttrs()
			if len(attrs) != 1 {
				t.Fatalf("FlatLogAttrs() len = %d, want 1", len(attrs))
			}
			if attrs[0].Key != tt.wantKey {
				t.Errorf("FlatLogAttrs()[0].Key = %q, want %q", attrs[0].Key, tt.wantKey)
			}
		})
	}
}

func TestStackTraceTypes(t *testing.T) {
	t.Parallel()
	err := stacktrace.Wrap(errors.New("test"))
	st := stacktrace.Extract(err)

	if st == nil {
		t.Fatal("expected stacktrace")
	}
	if got := reflect.TypeOf(st).String(); got != "stacktrace.StackTrace" {
		t.Errorf("unexpected type: %s", got)
	}
	if len(st) > 0 {
		frame := st[0]
		if frame.File == "" {
			t.Error("expected non-empty File")
		}
		if frame.LineNumber == 0 {
			t.Error("expected non-zero LineNumber")
		}
		if frame.Function == "" {
			t.Error("expected non-empty Function")
		}
	}
}
