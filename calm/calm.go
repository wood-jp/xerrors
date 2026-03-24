// Package calm provides panic recovery that converts panics into errors with
// stack traces and an [errclass.Panic] classification.
package calm

import (
	"fmt"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errclass"
	"github.com/wood-jp/xerrors/stacktrace"
)

const (
	// depth of stack to ignore so that the stack trace from the panic recovery
	// does not include the deferred recovery function itself.
	panicStackDepth = 3
)

// Unpanic executes the given function catching any panic and returning it as an error with stack trace
// and an [errclass.Panic] classification.
// WARNING: It is not possible to recover from a panic in a goroutine spawned by `f()`. Users should ensure
// that any goroutines created by `f()` are likewise guarded against panics.
func Unpanic(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// panic can be called with anything. If called with an error, recover the actual error.
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = fmt.Errorf("panic: %w", e)
			} else {
				panicErr = fmt.Errorf("panic: %v", r)
			}
			err = xerrors.Extend(stacktrace.GetStack(panicStackDepth, true), panicErr)
			err = errclass.WrapAs(err, errclass.Panic)
		}
	}()

	return f()
}
