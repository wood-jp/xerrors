package calm_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/wood-jp/xerrors/calm"
	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/stacktrace"
)

var errTest = fmt.Errorf("test error")

func a() error {
	return b()
}

func b() error {
	return c()
}

func c() error {
	panic("this is a test panic")
}

func d() error {
	return e()
}

func e() error {
	return f()
}

func f() error {
	panic(errTest)
}

// TestStackTrace checks that a panic is caught correctly.
func TestUnpanic(t *testing.T) {
	t.Parallel()

	err := calm.Unpanic(a)
	if err == nil {
		t.Errorf("expected error: got %v", err)
	}

	class := errclass.GetClass(err)
	if class != errclass.Panic {
		t.Errorf("unexpected error class: want: %s got %s", errclass.Panic, class)
	}

	trace := stacktrace.Extract(err)
	if trace == nil {
		t.Errorf("expected stack trace: got %v", trace)
	}

	if len(trace) != 5 {
		t.Errorf("unexpected stack trace len: want: %d got %d.\n-----\n%v\n-----\n", 5, len(trace), trace)
	}

	// Intentionally skip checking the line numbers: would make this test to brittle.
	expected := []stacktrace.Frame{
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.c",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.b",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.a",
		},
		{
			File:     "calm/calm.go",
			Function: "calm.Unpanic",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.TestUnpanic",
		},
	}

	for i, frame := range trace {
		if !strings.HasSuffix(frame.File, expected[i].File) {
			t.Errorf("unexpected file name suffix: want: %s got %s", expected[i].File, frame.File)
		}
		if !strings.HasSuffix(frame.Function, expected[i].Function) {
			t.Errorf("unexpected function name suffix: want: %s got %s", expected[i].Function, frame.Function)
		}
	}
}

func TestUnpanicError(t *testing.T) {
	t.Parallel()

	err := calm.Unpanic(d)
	if err == nil {
		t.Errorf("expected error: got %v", err)
	}

	if !errors.Is(err, errTest) {
		t.Errorf("unexpected error: want: %s (%T) got %s (%T)", errTest, errTest, err, err)
	}

	class := errclass.GetClass(err)
	if class != errclass.Panic {
		t.Errorf("unexpected error class: want: %s got %s", errclass.Panic, class)
	}

	trace := stacktrace.Extract(err)
	if trace == nil {
		t.Errorf("expected stack trace: got %v", trace)
	}

	if len(trace) != 5 {
		t.Errorf("unexpected stack trace len: want: %d got %d.\n-----\n%v\n-----\n", 5, len(trace), trace)
	}

	// Intentionally skip checking the line numbers: would make this test to brittle.
	expected := []stacktrace.Frame{
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.f",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.e",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.d",
		},
		{
			File:     "calm/calm.go",
			Function: "calm.Unpanic",
		},
		{
			File:     "calm/calm_test.go",
			Function: "calm_test.TestUnpanicError",
		},
	}

	for i, frame := range trace {
		if !strings.HasSuffix(frame.File, expected[i].File) {
			t.Errorf("unexpected file name suffix: want: %s got %s", expected[i].File, frame.File)
		}
		if !strings.HasSuffix(frame.Function, expected[i].Function) {
			t.Errorf("unexpected function name suffix: want: %s got %s", expected[i].Function, frame.Function)
		}
	}
}
