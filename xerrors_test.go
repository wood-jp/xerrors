package xerrors_test

import (
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errcontext"
	"github.com/wood-jp/xerrors/stacktrace"
)

var errTest = fmt.Errorf("this is a test error")

func wrap(err error) error {
	return fmt.Errorf("wrapping: %w", err)
}

func TestExtendedError(t *testing.T) {
	t.Parallel()

	type dataOne struct {
		s1 string
		s2 string
	}

	type dataTwo struct {
		t time.Time
		i int
	}

	type dataThree struct{}

	d1 := dataOne{
		s1: "hello",
		s2: "world",
	}

	d2 := dataTwo{
		t: time.Now(),
		i: 17,
	}

	// extending nil is still nil
	e0 := xerrors.Extend(d1, nil)
	if e0 != nil {
		t.Errorf("unexpected error: want: %v, got %v", nil, e0)
	}

	// extending errTest with d1 is still an errTest
	e1 := xerrors.Extend(d1, errTest)
	if !errors.Is(e1, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e1)
	}

	// extending e1 with d2 is still an errTest and an e1
	e2 := xerrors.Extend(d2, e1)
	if !errors.Is(e2, e1) {
		t.Errorf("unmatched error: want: %v, got %v", e1, e2)
	}
	if !errors.Is(e2, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e2)
	}

	// wrapping e2 (twice) is still an errTest, e1, and e2
	e3 := wrap(wrap(e2))
	if !errors.Is(e3, e2) {
		t.Errorf("unmatched error: want: %v, got %v", e2, e3)
	}
	if !errors.Is(e3, e1) {
		t.Errorf("unmatched error: want: %v, got %v", e1, e3)
	}
	if !errors.Is(e3, errTest) {
		t.Errorf("unmatched error: want: %v, got %v", errTest, e3)
	}

	// able to extract the data from e3 that was added to e1
	f1, ok := xerrors.Extract[dataOne](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d1 != f1 {
		t.Errorf("expected equal values: want %v, got %v", d1, f1)
	}

	// able to extract the data from e3 that was added to e2
	f2, ok := xerrors.Extract[dataTwo](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d2 != f2 {
		t.Errorf("expected equal values: want %v, got %v", d2, f2)
	}

	// properly fails to extract data that was never added
	_, ok = xerrors.Extract[dataThree](e3)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}
}

func TestExtendedWithSameType(t *testing.T) {
	t.Parallel()

	type dataOne struct {
		s1 string
		s2 string
	}

	d1 := dataOne{
		s1: "hello",
		s2: "world",
	}

	d2 := dataOne{
		s1: "goodbye",
		s2: "friend",
	}

	// extending an error with the same data type is fine
	// but extracting it will only give the outer-most (i.e., the last extended) value
	e1 := xerrors.Extend(d1, errTest)
	e2 := xerrors.Extend(d2, e1)

	f1, ok := xerrors.Extract[dataOne](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d2 != f1 {
		t.Errorf("expected equal values: want %v, got %v", d2, f1)
	}

	// however if unwrap manually, the data is still there and accessible
	e3 := errors.Unwrap(e2)

	f2, ok := xerrors.Extract[dataOne](e3)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if d1 != f2 {
		t.Errorf("expected equal values: want %v, got %v", d1, f2)
	}
}

// findAttr returns the first attr in attrs whose key equals key, plus a found flag.
func findAttr(attrs []slog.Attr, key string) (slog.Attr, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a, true
		}
	}
	return slog.Attr{}, false
}

func TestLogValue(t *testing.T) {
	t.Parallel()

	type unknownData struct{ name string }

	tests := []struct {
		name            string
		err             error
		wantDetailKeys  []string // keys expected inside error_detail group
		wantNoDetail    bool
	}{
		{
			name:         "plain non-extended error",
			err:          errors.New("plain error"),
			wantNoDetail: true,
		},
		{
			name:           "unknown data type falls back to data attr",
			err:            xerrors.Extend(unknownData{"x"}, errors.New("extended")),
			wantDetailKeys: []string{"data"},
		},
		{
			name:           "errclass and stacktrace composition",
			err:            stacktrace.Wrap(errclass.WrapAs(errors.New("something went wrong"), errclass.Transient)),
			wantDetailKeys: []string{"class", "stacktrace"},
		},
		{
			name:           "errcontext adds context group",
			err:            errcontext.Add(errors.New("ctx error"), slog.String("user_id", "123")),
			wantDetailKeys: []string{"context"},
		},
		{
			name:           "fmt.Errorf wrapper is transparent",
			err:            fmt.Errorf("wrapped: %w", errclass.WrapAs(errors.New("inner"), errclass.Persistent)),
			wantDetailKeys: []string{"class"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			val := xerrors.Log(tt.err).Value
			if val.Kind() != slog.KindGroup {
				t.Fatalf("Log().Value kind = %v, want KindGroup", val.Kind())
			}

			topAttrs := val.Group()

			// The top-level "error" string attr must always be present.
			errAttr, ok := findAttr(topAttrs, "error")
			if !ok {
				t.Fatal("Log().Value missing top-level 'error' attr")
			}
			if errAttr.Value.Kind() != slog.KindString {
				t.Errorf("'error' attr kind = %v, want KindString", errAttr.Value.Kind())
			}
			if errAttr.Value.String() != tt.err.Error() {
				t.Errorf("'error' attr = %q, want %q", errAttr.Value.String(), tt.err.Error())
			}

			if tt.wantNoDetail {
				if _, found := findAttr(topAttrs, "error_detail"); found {
					t.Error("unexpected 'error_detail' attr for plain error")
				}
				return
			}

			detailAttr, found := findAttr(topAttrs, "error_detail")
			if !found {
				t.Fatal("missing 'error_detail' attr")
			}
			if detailAttr.Value.Kind() != slog.KindGroup {
				t.Fatalf("'error_detail' kind = %v, want KindGroup", detailAttr.Value.Kind())
			}

			detailAttrs := detailAttr.Value.Group()
			for _, key := range tt.wantDetailKeys {
				if _, found := findAttr(detailAttrs, key); !found {
					t.Errorf("'error_detail' missing expected key %q", key)
				}
			}
		})
	}
}

type (
	ClassA int
	ClassB int
)

const (
	AZero ClassA = iota
	AOne
	ATwo

	BZero ClassB = iota
	BOne
)

func TestExtendedWithMultipleTypedefs(t *testing.T) {
	t.Parallel()

	// ClassA and ClassB are different types but both are int under the hood
	// This test proves that Extract can tell the difference as expected
	e1 := xerrors.Extend(ATwo, errTest)
	e2 := xerrors.Extend(BOne, e1)

	// e2 has a ClassA of ATwo
	f1, ok := xerrors.Extract[ClassA](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if f1 != ATwo {
		t.Errorf("expected equal values: want %v, got %v", ATwo, f1)
	}

	// e2 also has a ClassB of BOne
	f2, ok := xerrors.Extract[ClassB](e2)
	if !ok {
		t.Errorf("expected true: got %v", ok)
	}
	if f2 != BOne {
		t.Errorf("expected equal values: want %v, got %v", BOne, f2)
	}

	// e1 was never wrapped with a ClassB
	_, ok = xerrors.Extract[ClassB](e1)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}

	// ClassB didn't have a value defined for 2. Make sure that wasn't why the above passes.
	e3 := xerrors.Extend(AZero, errTest)
	_, ok = xerrors.Extract[ClassB](e3)
	if ok {
		t.Errorf("expected false: got %v", ok)
	}
}
