# errcontext

Structured logging context for errors, built on top of `xerrors`.

## Overview

`errcontext` attaches `slog.Attr` key-value pairs to errors using `xerrors.Extend`, allowing downstream callers to extract and log structured context alongside error messages.

## Usage

### Adding context to an error

```go
err := errcontext.Add(err, slog.String("user_id", "123"), slog.Int("attempt", 3))
```

### Adding context incrementally

`Add` can be called multiple times. If the error already has context, the existing map is mutated in place (last-entry-wins) without adding extra wrapper layers:

```go
err := errcontext.Add(err, slog.String("user_id", "123"))
err = errcontext.Add(err, slog.String("request_id", "abc"))
// Both keys are present in a single context layer
```

### Extracting context

```go
ctx := errcontext.Get(err)
if ctx != nil {
    attrs := ctx.Flatten() // sorted by key
    slog.Info("request failed", attrs...)
}
```

### Structured logging

`Context` implements `slog.LogValuer`, so context is included automatically when logging an error that carries it:

```go
slog.Error("request failed", slog.Any("error", err))
// Output includes context key-value pairs within the error structure
```

## Behavior Notes

- `Add` with a nil error returns nil
- `Add` with no attrs returns the error unchanged
- Duplicate keys are resolved with last-entry-wins semantics
- `Flatten` returns attrs sorted by key for deterministic output
- Works alongside other `xerrors.Extend` wrappers (e.g., `errclass`) without interference

## Limitations

Joined errors created with `errors.Join` are not supported for wrapping using this package.
