package errcontext_test

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errcontext"
)

var errTest = fmt.Errorf("this is a test error")

func attrsEqual(a, b []slog.Attr) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Value.String() != b[i].Value.String() {
			return false
		}
	}
	return true
}

// TestErrorAs validates that the contextualized error can be cast properly.
func TestErrorAs(t *testing.T) {
	t.Parallel()

	err := errcontext.Add(errTest, slog.String("test", "test"))
	if !errors.Is(err, errTest) {
		t.Error("expected errors.Is to match errTest")
	}
	var extendedError xerrors.ExtendedError[errcontext.Context]
	if !errors.As(err, &extendedError) {
		t.Error("expected errors.As to match ExtendedError[Context]")
	}
	got := errcontext.Get(err).Flatten()
	expected := []slog.Attr{slog.String("test", "test")}
	if !attrsEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// TestAddContext validates that context can be added and retrieved.
func TestAddContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName string
		err      error
		contexts [][]slog.Attr
	}{
		{
			testName: "nil error",
			err:      nil,
			contexts: nil,
		},
		{
			testName: "no context",
			err:      errTest,
			contexts: nil,
		},
		{
			testName: "single context",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one")},
			},
		},
		{
			testName: "double-sized single context",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one"), slog.String("two", "two")},
			},
		},
		{
			testName: "two single contexts",
			err:      errTest,
			contexts: [][]slog.Attr{
				{slog.String("one", "one")},
				{slog.String("two", "two")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			err := tc.err
			var expected []slog.Attr
			for _, context := range tc.contexts {
				expected = append(expected, context...)
				err = errcontext.Add(err, context...)
			}

			actual := errcontext.Get(err).Flatten()
			if len(expected) == 0 && len(actual) == 0 {
				return
			}
			if !attrsEqual(actual, expected) {
				t.Errorf("expected %v, got %v", expected, actual)
			}
		})
	}
}

// TestAddContextOverOthers validates that context can be added across
// different wrapper types (e.g. errclass) without losing data.
func TestAddContextOverOthers(t *testing.T) {
	t.Parallel()

	// add some context
	err := errcontext.Add(errTest, slog.String("one", "one"))

	// wrap the error in a different way (add a class)
	err = errclass.WrapAs(err, errclass.Transient)

	// add some more context
	err = errcontext.Add(err, slog.String("two", "two"))

	// ensure the class remains
	if errclass.GetClass(err) != errclass.Transient {
		t.Errorf("expected class Transient, got %v", errclass.GetClass(err))
	}

	// ensure all added context is present
	got := errcontext.Get(err).Flatten()
	want := []slog.Attr{slog.String("one", "one"), slog.String("two", "two")}
	if !attrsEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}

	// add context with duplicate key - this overwrites (last-entry-wins)
	err = errcontext.Add(err, slog.String("two", "three"))

	got = errcontext.Get(err).Flatten()
	want = []slog.Attr{slog.String("one", "one"), slog.String("two", "three")}
	if !attrsEqual(got, want) {
		t.Errorf("after overwrite: expected %v, got %v", want, got)
	}
}

// TestAddContextInPlace validates that adding context to an error that
// already has context mutates the existing map rather than adding a new
// wrapper layer.
func TestAddContextInPlace(t *testing.T) {
	t.Parallel()

	// First Add creates a wrapper
	err := errcontext.Add(errTest, slog.String("key1", "val1"))

	// Second Add should mutate in place and return the same error
	err2 := errcontext.Add(err, slog.String("key2", "val2"))

	// Verify both keys are present
	got := errcontext.Get(err2).Flatten()
	want := []slog.Attr{slog.String("key1", "val1"), slog.String("key2", "val2")}
	if !attrsEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}

	// Verify there is only one context wrapper layer:
	// unwrapping once should yield the base error with no context.
	unwrapped := errors.Unwrap(err2)
	if _, ok := xerrors.Extract[errcontext.Context](unwrapped); ok {
		t.Error("expected no context on inner error; found a redundant context layer")
	}
	if !errors.Is(unwrapped, errTest) {
		t.Error("expected inner error to be errTest")
	}
}

// TestLogValue validates that Context.LogValue() works correctly.
func TestLogValue(t *testing.T) {
	t.Parallel()

	// Test empty context
	emptyContext := errcontext.Context{}
	v := emptyContext.LogValue()
	if v.Kind() != slog.KindAny || v.String() != "<nil>" {
		// slog.Value{} has KindAny with nil underlying
		if v.String() != "<nil>" && v.Kind() != slog.KindAny {
			t.Errorf("expected zero Value for empty context, got kind=%v", v.Kind())
		}
	}

	// Test context with values
	ctx := errcontext.Context{
		"key1": slog.StringValue("value1"),
		"key2": slog.IntValue(42),
	}

	logValue := ctx.LogValue()
	if logValue.Kind() != slog.KindGroup {
		t.Errorf("expected KindGroup, got %v", logValue.Kind())
	}

	attrs := logValue.Group()
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	attrMap := make(map[string]string)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value.String()
	}

	if attrMap["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", attrMap["key1"])
	}
	if attrMap["key2"] != "42" {
		t.Errorf("expected key2=42, got %v", attrMap["key2"])
	}
}

// TestAddEmptyContext validates that Add with no attrs returns the error unchanged.
func TestAddEmptyContext(t *testing.T) {
	t.Parallel()

	result := errcontext.Add(errTest)
	if result != errTest {
		t.Error("expected same error returned when no context provided")
	}
	if errcontext.Get(result) != nil {
		t.Error("expected no context wrapper when no context provided")
	}
}

// TestAddNilError validates that Add with nil error returns nil.
func TestAddNilError(t *testing.T) {
	t.Parallel()

	result := errcontext.Add(nil, slog.String("key", "value"))
	if result != nil {
		t.Error("expected nil result for nil error")
	}
}

// TestGetNilError validates that Get with nil error returns nil.
func TestGetNilError(t *testing.T) {
	t.Parallel()

	result := errcontext.Get(nil)
	if result != nil {
		t.Error("expected nil context for nil error")
	}
}

// TestGetNoContext validates that Get returns nil for errors without context.
func TestGetNoContext(t *testing.T) {
	t.Parallel()

	result := errcontext.Get(errors.New("no context"))
	if result != nil {
		t.Error("expected nil context for error without context")
	}
}
