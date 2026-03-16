// Package xerrors provides generic error wrapping, allowing any data type
// to be captured alongside an error. Wrapped errors remain compatible with
// [errors.Is] and [errors.As] via the standard [errors.Unwrap] interface.
package xerrors

import (
	"errors"
	"log/slog"
)

// LogDetailer is implemented by error data types that contribute attributes
// to the error_detail group in [FlatLogValue].
type LogDetailer interface {
	FlatLogAttrs() []slog.Attr
}

// extendedErrFlat is the unexported interface used by [collectDetails] to walk
// the error chain and gather flat log attributes from each extended-error layer.
type extendedErrFlat interface {
	flatLogAttrs() []slog.Attr
	innerError() error
}

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

// innerError implements [extendedErrFlat], returning the wrapped error.
func (e ExtendedError[T]) innerError() error {
	return e.err
}

// flatLogAttrs implements [extendedErrFlat]. If T implements [LogDetailer],
// it delegates to that; otherwise it falls back to a single "data" attribute.
func (e ExtendedError[T]) flatLogAttrs() []slog.Attr {
	if fa, ok := any(e.Data).(LogDetailer); ok {
		return fa.FlatLogAttrs()
	}
	return []slog.Attr{slog.Any("data", e.Data)}
}

// FlatLogValue returns a flat [slog.Value] for err, combining the top-level
// error message with an optional error_detail group that aggregates attributes
// from every extended-error layer in the chain.
//
// This is an opt-in alternative to the default nested [ExtendedError.LogValue]
// format. The existing LogValue behaviour is unchanged.
func FlatLogValue(err error) slog.Value {
	detailAttrs := collectDetails(err)
	result := []slog.Attr{slog.String("error", err.Error())}
	if len(detailAttrs) > 0 {
		result = append(result, slog.Attr{
			Key:   "error_detail",
			Value: slog.GroupValue(detailAttrs...),
		})
	}
	return slog.GroupValue(result...)
}

// collectDetails walks the error chain and gathers flat log attributes from
// every [extendedErrFlat] layer, in innermost-to-outermost order.
func collectDetails(err error) []slog.Attr {
	if err == nil {
		return nil
	}
	if ee, ok := err.(extendedErrFlat); ok {
		inner := collectDetails(ee.innerError())
		return append(inner, ee.flatLogAttrs()...)
	}
	// Transparent for fmt.Errorf %w wrappers and similar.
	if u := errors.Unwrap(err); u != nil {
		return collectDetails(u)
	}
	return nil
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

// Log returns an [slog.Attr] with key "error" and the flat log value of err,
// suitable for passing directly to slog methods:
//
//	logger.Error("request failed", xerrors.Log(err))
func Log(err error) slog.Attr {
	return slog.Any("error", FlatLogValue(err))
}
