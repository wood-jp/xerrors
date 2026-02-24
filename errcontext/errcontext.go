// Package errcontext attaches structured logging context to errors.
// It wraps errors with a [Context] map using [xerrors.Extend], enabling
// downstream callers to extract [slog.Attr] key-value pairs for logging.
package errcontext

import (
	"log/slog"
	"maps"
	"slices"

	"github.com/wood-jp/xerrors"
)

// Context is a map of key-value pairs attached to an error for structured logging.
type Context map[string]slog.Value

// Flatten returns the context as a slice of [slog.Attr] sorted by key.
func (c Context) Flatten() []slog.Attr {
	keys := slices.Sorted(maps.Keys(c))
	attrs := make([]slog.Attr, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.Attr{Key: key, Value: c[key]})
	}
	return attrs
}

// LogValue implements [slog.LogValuer].
// It returns the context as a group value with keys in sorted order.
func (c Context) LogValue() slog.Value {
	if len(c) == 0 {
		return slog.Value{}
	}

	return slog.GroupValue(c.Flatten()...)
}

// Add attaches the given [slog.Attr] key-value pairs to err as logging context.
// If err already has a [Context], the existing map is mutated in place (last-entry-wins).
// Returns nil if err is nil, or err unchanged if no attrs are provided.
func Add(err error, context ...slog.Attr) error {
	if err == nil {
		return nil
	} else if len(context) == 0 {
		return err
	}

	// If the error already has context, mutate the existing map in place
	// to avoid adding a redundant wrapper layer.
	if existing := Get(err); existing != nil {
		for _, attr := range context {
			existing[attr.Key] = attr.Value
		}
		return err
	}

	newContext := make(Context, len(context))
	for _, attr := range context {
		newContext[attr.Key] = attr.Value
	}
	return xerrors.Extend(newContext, err)
}

// Get extracts the [Context] attached to err, or nil if none is present.
func Get(err error) Context {
	if err == nil {
		return nil
	}

	if context, ok := xerrors.Extract[Context](err); ok {
		return context
	}
	return nil
}
