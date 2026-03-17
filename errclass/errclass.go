// Package errclass provides error classification by severity level.
// It wraps errors with a [Class] using [xerrors.Extend], enabling downstream
// callers to inspect and act on error severity.
package errclass

import (
	"log/slog"

	"github.com/wood-jp/xerrors"
)

// Class represents the severity classification of an error.
// Higher values indicate more severe errors.
type Class int

const (
	// Nil indicates a nil error (no error). It has value -1.
	Nil Class = iota - 1
	// Unknown is the zero value, used for errors that have not been classified.
	Unknown
	// Transient indicates a temporary error that may succeed on retry.
	Transient
	// Persistent indicates a permanent error that will not resolve on retry.
	Persistent
	// Panic indicates an error resulting from a recovered panic.
	Panic
)

// String returns the lowercase name of the Class.
// Unrecognized values return "unknown".
func (c Class) String() string {
	switch c {
	case Nil:
		return "nil"
	case Panic:
		return "panic"
	case Transient:
		return "transient"
	case Persistent:
		return "persistent"
	default:
		return "unknown"
	}
}

// LogValue implements [slog.LogValuer], returning the class name as a grouped slog value.
func (c Class) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("class", c.String()),
	)
}

type wrappingRestriction int

const (
	wrappingRestrictionNone wrappingRestriction = iota
	wrappingRestrictionOnlyUnknown
	wrappingRestrictionOnlyMoreSevere
)

type wrapOptions struct {
	restriction wrappingRestriction
}

// WrapOption configures the behavior of [WrapAs].
type WrapOption func(opt *wrapOptions)

// WithUnrestricted allows [WrapAs] to wrap the error unconditionally.
// This is the default behavior when no option is provided.
func WithUnrestricted() WrapOption {
	return func(opt *wrapOptions) {
		opt.restriction = wrappingRestrictionNone
	}
}

// WithOnlyUnknown restricts [WrapAs] to only wrap errors whose current class
// is [Unknown]. Errors that already have a class are returned unchanged.
func WithOnlyUnknown() WrapOption {
	return func(opt *wrapOptions) {
		opt.restriction = wrappingRestrictionOnlyUnknown
	}
}

// WithOnlyMoreSevere restricts [WrapAs] to only wrap errors when the provided
// class is strictly more severe than the error's current class. Otherwise the
// error is returned unchanged.
func WithOnlyMoreSevere() WrapOption {
	return func(opt *wrapOptions) {
		opt.restriction = wrappingRestrictionOnlyMoreSevere
	}
}

// WrapAs wraps err with the given [Class]. If err is nil, it returns nil.
// By default the class is applied unconditionally. Use [WrapOption] values
// such as [WithOnlyUnknown] or [WithOnlyMoreSevere] to restrict when
// wrapping occurs.
func WrapAs(err error, class Class, opts ...WrapOption) error {
	if err == nil {
		return nil
	}

	// Apply options
	options := wrapOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	currentClass := GetClass(err)
	switch options.restriction {
	case wrappingRestrictionOnlyUnknown:
		if currentClass == Unknown {
			return xerrors.Extend(class, err)
		}
		return err

	case wrappingRestrictionOnlyMoreSevere:
		if class > currentClass {
			return xerrors.Extend(class, err)
		}
		return err

	default:
		return xerrors.Extend(class, err)
	}
}

// GetClass extracts the [Class] from err. It returns [Nil] if err is nil,
// and [Unknown] if err does not carry a class.
func GetClass(err error) Class {
	if err == nil {
		return Nil
	}
	if class, ok := xerrors.Extract[Class](err); ok {
		return class
	}
	return Unknown
}
