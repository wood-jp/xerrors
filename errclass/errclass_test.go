package errclass_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/wood-jp/xerrors/errclass"
)

func TestClassString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		class errclass.Class
		want  string
	}{
		{"Nil", errclass.Nil, "nil"},
		{"Unknown", errclass.Unknown, "unknown"},
		{"Transient", errclass.Transient, "transient"},
		{"Persistent", errclass.Persistent, "persistent"},
		{"Panic", errclass.Panic, "panic"},
		{"out of range", errclass.Class(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.class.String(); got != tt.want {
				t.Errorf("Class(%d).String() = %q, want %q", tt.class, got, tt.want)
			}
		})
	}
}

func TestClassLogValue(t *testing.T) {
	t.Parallel()
	val := errclass.Transient.LogValue()
	if val.Kind() != slog.KindGroup {
		t.Fatalf("LogValue().Kind() = %v, want %v", val.Kind(), slog.KindGroup)
	}
	attrs := val.Group()
	if len(attrs) != 1 {
		t.Fatalf("LogValue() group has %d attrs, want 1", len(attrs))
	}
	if attrs[0].Key != "class" || attrs[0].Value.String() != "transient" {
		t.Errorf("LogValue() attr = %v, want class=transient", attrs[0])
	}
}

func TestClassConstants(t *testing.T) {
	t.Parallel()
	if errclass.Nil != -1 {
		t.Errorf("Nil = %d, want -1", errclass.Nil)
	}
	if errclass.Unknown != 0 {
		t.Errorf("Unknown = %d, want 0", errclass.Unknown)
	}
	// Severity ordering: higher values are more severe
	if !(errclass.Nil < errclass.Unknown && errclass.Unknown < errclass.Transient && errclass.Transient < errclass.Persistent && errclass.Persistent < errclass.Panic) {
		t.Error("class severity ordering violated")
	}
}

func TestWrapAs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		class     errclass.Class
		wantNil   bool
		wantMsg   string
		wantIsErr error
	}{
		{
			name:      "wraps error with class",
			err:       errors.New("something failed"),
			class:     errclass.Transient,
			wantNil:   false,
			wantMsg:   "something failed",
			wantIsErr: nil,
		},
		{
			name:    "nil error returns nil",
			err:     nil,
			class:   errclass.Transient,
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := errclass.WrapAs(tt.err, tt.class)
			if tt.wantNil {
				if got != nil {
					t.Errorf("WrapAs(nil, %v) = %v, want nil", tt.class, got)
				}
				return
			}
			if got == nil {
				t.Fatal("WrapAs returned nil for non-nil error")
			}
			if got.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got.Error(), tt.wantMsg)
			}
			if !errors.Is(got, tt.err) {
				t.Error("wrapped error does not unwrap to original")
			}
		})
	}
}

func TestWrapAsWithOnlyUnknown(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		class     errclass.Class
		wantClass errclass.Class
	}{
		{
			name:      "wraps plain error",
			err:       errors.New("plain"),
			class:     errclass.Transient,
			wantClass: errclass.Transient,
		},
		{
			name:      "skips already classified error",
			err:       errclass.WrapAs(errors.New("err"), errclass.Persistent),
			class:     errclass.Transient,
			wantClass: errclass.Persistent,
		},
		{
			name:      "skips even when upgrading severity",
			err:       errclass.WrapAs(errors.New("err"), errclass.Transient),
			class:     errclass.Panic,
			wantClass: errclass.Transient,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := errclass.WrapAs(tt.err, tt.class, errclass.WithOnlyUnknown())
			if gotClass := errclass.GetClass(got); gotClass != tt.wantClass {
				t.Errorf("GetClass() = %v, want %v", gotClass, tt.wantClass)
			}
		})
	}
}

func TestWrapAsWithOnlyMoreSevere(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		err       error
		class     errclass.Class
		wantClass errclass.Class
	}{
		{
			name:      "wraps plain error with higher class",
			err:       errors.New("plain"),
			class:     errclass.Transient,
			wantClass: errclass.Transient,
		},
		{
			name:      "upgrades to more severe class",
			err:       errclass.WrapAs(errors.New("err"), errclass.Transient),
			class:     errclass.Persistent,
			wantClass: errclass.Persistent,
		},
		{
			name:      "skips equal severity",
			err:       errclass.WrapAs(errors.New("err"), errclass.Transient),
			class:     errclass.Transient,
			wantClass: errclass.Transient,
		},
		{
			name:      "skips lower severity",
			err:       errclass.WrapAs(errors.New("err"), errclass.Persistent),
			class:     errclass.Transient,
			wantClass: errclass.Persistent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := errclass.WrapAs(tt.err, tt.class, errclass.WithOnlyMoreSevere())
			if gotClass := errclass.GetClass(got); gotClass != tt.wantClass {
				t.Errorf("GetClass() = %v, want %v", gotClass, tt.wantClass)
			}
		})
	}
}

func TestWrapAsWithUnrestricted(t *testing.T) {
	t.Parallel()
	// WithUnrestricted is the default, but verify the option is accepted and wraps regardless
	err := errclass.WrapAs(errors.New("err"), errclass.Transient)
	got := errclass.WrapAs(err, errclass.Persistent, errclass.WithUnrestricted())
	if gotClass := errclass.GetClass(got); gotClass != errclass.Persistent {
		t.Errorf("GetClass() = %v, want Persistent", gotClass)
	}
}

func TestGetClass(t *testing.T) {
	t.Parallel()
	inner := errclass.WrapAs(errors.New("err"), errclass.Transient)
	tests := []struct {
		name string
		err  error
		want errclass.Class
	}{
		{"nil error returns Nil", nil, errclass.Nil},
		{"plain error returns Unknown", errors.New("plain"), errclass.Unknown},
		{"transient wrapped", errclass.WrapAs(errors.New("err"), errclass.Transient), errclass.Transient},
		{"persistent wrapped", errclass.WrapAs(errors.New("err"), errclass.Persistent), errclass.Persistent},
		{"panic wrapped", errclass.WrapAs(errors.New("err"), errclass.Panic), errclass.Panic},
		{"nested wrap extracts nearest class", errclass.WrapAs(inner, errclass.Persistent), errclass.Persistent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := errclass.GetClass(tt.err); got != tt.want {
				t.Errorf("GetClass() = %v, want %v", got, tt.want)
			}
		})
	}
}
