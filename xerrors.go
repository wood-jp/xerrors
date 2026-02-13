// Package xerrors provides a generic implementation of error wrapping, allowing any data type to be captured alongside an error.
package xerrors

import (
	"errors"
	"log/slog"
)

// ExtendedError is a generic custom error wrapper
type ExtendedError[T any] struct {
	err  error
	Data T
}

// Error implements the error interface
func (e ExtendedError[T]) Error() string {
	return e.err.Error()
}

// Unwrap allows access to the underlying error (used by errors.Is and other Go 1.13 error handling funcs)
func (e ExtendedError[T]) Unwrap() error {
	return e.err
}

// LogValue implements slog.LogValuer interface (Go 1.21+)
func (e ExtendedError[T]) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("error", e.err),
		slog.Any("data", e.Data),
	)
}

// Extend wraps an error with additional data
func Extend[T any](data T, err error) error {
	if err == nil {
		return nil
	}
	return ExtendedError[T]{Data: data, err: err}
}

// Extract returns wrapped data if possible, even in cases of deeply nested wrapping.
// NOTE: If an error is extended multiple times with the same data type,
// then only the nearest matching type is returned. See tests for examples.
func Extract[T any](err error) (T, bool) {
	e, ok := errors.AsType[ExtendedError[T]](err)
	return e.Data, ok
}
