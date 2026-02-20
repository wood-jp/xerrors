// Package xerrors provides generic error wrapping, allowing any data type
// to be captured alongside an error. Wrapped errors remain compatible with
// [errors.Is] and [errors.As] via the standard [errors.Unwrap] interface.
package xerrors

import (
	"errors"
	"log/slog"
)

// ExtendedError wraps an error with an additional value of type T.
// It implements [error], [interface{ Unwrap() error }], and [slog.LogValuer].
type ExtendedError[T any] struct {
	err  error
	Data T
}

// Error returns the error string of the underlying error.
func (e ExtendedError[T]) Error() string {
	return e.err.Error()
}

// Unwrap returns the underlying error, allowing [errors.Is] and [errors.As]
// to traverse the error chain.
func (e ExtendedError[T]) Unwrap() error {
	return e.err
}

// LogValue implements [slog.LogValuer], returning a group containing the
// underlying error and the attached data.
func (e ExtendedError[T]) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("error", e.err),
		slog.Any("data", e.Data),
	)
}

// Extend wraps err with the given data, returning an [ExtendedError].
// If err is nil, it returns nil.
func Extend[T any](data T, err error) error {
	if err == nil {
		return nil
	}
	return ExtendedError[T]{Data: data, err: err}
}

// Extract walks the error chain and returns the Data field from the first
// [ExtendedError] whose type parameter matches T. If no match is found,
// it returns the zero value of T and false.
//
// When an error has been extended multiple times with the same type T,
// only the outermost (nearest) match is returned.
func Extract[T any](err error) (T, bool) {
	e, ok := errors.AsType[ExtendedError[T]](err)
	return e.Data, ok
}
