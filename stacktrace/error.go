package stacktrace

import (
	"sync/atomic"

	"github.com/wood-jp/xerrors"
)

const (
	// depth of stack to ignore so that callers of Wrap don't see the call to Wrap itself.
	wrapStackDepth = 3
)

// Disabled disables stacktrace collection in Wrap when set to true.
var Disabled atomic.Bool

// Wrap extends err by attaching a [StackTrace] captured at the call site.
// If err is nil or [Disabled] is true, err is returned unchanged.
// If err already carries a [StackTrace], it is not wrapped again.
func Wrap(err error) error {
	if Disabled.Load() || err == nil {
		return err
	}
	if _, ok := xerrors.Extract[StackTrace](err); !ok {
		return xerrors.Extend(GetStack(wrapStackDepth, true), err)
	}
	return err
}

// Extract returns the [StackTrace] attached to err, or nil if none is present or err is nil.
func Extract(err error) StackTrace {
	st, ok := xerrors.Extract[StackTrace](err)
	if !ok {
		return nil
	}
	return st
}
