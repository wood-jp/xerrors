package errgroup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/errgroup"
)

var errTest = fmt.Errorf("this is a test error")

func a() error {
	return nil
}

func b() error {
	return errTest
}

func c() error {
	panic("this is a test panic")
}

type errFunc func() error

func TestGo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName      string
		funcs         []errFunc
		expectedClass errclass.Class
	}{
		{
			testName:      "funcs return nil",
			funcs:         []errFunc{a, a, a},
			expectedClass: errclass.Nil,
		},
		{
			testName:      "one func has error",
			funcs:         []errFunc{a, a, b},
			expectedClass: errclass.Unknown,
		},
		{
			testName:      "one func has panic",
			funcs:         []errFunc{a, a, c},
			expectedClass: errclass.Panic,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			g := errgroup.New()
			for _, f := range tc.funcs {
				g.Go(f)
			}

			err := g.Wait()
			class := errclass.GetClass(err)
			if class != tc.expectedClass {
				t.Errorf("unexpected error class: want: %s got %s", tc.expectedClass, class)
			}
		})
	}
}

func TestTryGo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName      string
		funcs         []errFunc
		expectedClass errclass.Class
	}{
		{
			testName:      "funcs return nil",
			funcs:         []errFunc{a, a, a},
			expectedClass: errclass.Nil,
		},
		{
			testName:      "one func has error",
			funcs:         []errFunc{a, a, b},
			expectedClass: errclass.Unknown,
		},
		{
			testName:      "one func has panic",
			funcs:         []errFunc{a, a, c},
			expectedClass: errclass.Panic,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()

			g := errgroup.New()
			for _, f := range tc.funcs {
				if !g.TryGo(f) {
					t.Errorf("expected TryGo to return true")
				}
			}

			err := g.Wait()
			class := errclass.GetClass(err)
			if class != tc.expectedClass {
				t.Errorf("unexpected error class: want: %s got %s", tc.expectedClass, class)
			}
		})
	}
}

func TestTryGoLimit(t *testing.T) {
	t.Parallel()

	block := make(chan struct{})
	d := func() error {
		<-block
		return nil
	}

	g := errgroup.New()
	g.SetLimit(1)

	// First TryGo starts d, which blocks on the channel — goroutine occupies the slot.
	if !g.TryGo(d) {
		t.Error("expected first TryGo to return true")
	}

	// Limit is 1 and d is still running: TryGo must return false.
	if g.TryGo(a) {
		t.Error("expected TryGo to return false when at limit")
	}

	// Unblock d, then wait for it to complete.
	close(block)
	if err := g.Wait(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWithContext(t *testing.T) {
	t.Parallel()

	// error cancels context: a goroutine returning an error should cancel the
	// context so that other goroutines can observe ctx.Done().
	t.Run("error cancels context", func(t *testing.T) {
		t.Parallel()

		block := make(chan struct{})
		t.Cleanup(func() { close(block) })
		g, ctx := errgroup.WithContext(context.Background())

		g.Go(func() error {
			select {
			case <-block:
			case <-ctx.Done():
			}
			return nil
		})
		g.Go(b) // returns errTest, cancelling ctx

		err := g.Wait()
		if !errors.Is(err, errTest) {
			t.Errorf("unexpected error: want %v got %v", errTest, err)
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("expected context.Canceled: got %v", ctx.Err())
		}
	})

	// panic cancels context: a recovered panic is an error, so the same
	// cancellation applies.
	t.Run("panic cancels context", func(t *testing.T) {
		t.Parallel()

		block := make(chan struct{})
		t.Cleanup(func() { close(block) })
		g, ctx := errgroup.WithContext(context.Background())

		g.Go(func() error {
			select {
			case <-block:
			case <-ctx.Done():
			}
			return nil
		})
		g.Go(c) // panics, recovered as errclass.Panic, cancelling ctx

		err := g.Wait()
		if errclass.GetClass(err) != errclass.Panic {
			t.Errorf("unexpected error class: want %s got %s", errclass.Panic, errclass.GetClass(err))
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("expected context.Canceled: got %v", ctx.Err())
		}
	})

	// nil return: context is cancelled when Wait returns, even on success.
	t.Run("nil return cancels context on Wait", func(t *testing.T) {
		t.Parallel()

		g, ctx := errgroup.WithContext(context.Background())
		g.Go(a)
		g.Go(a)

		if err := g.Wait(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("expected context.Canceled: got %v", ctx.Err())
		}
	})
}
