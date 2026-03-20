# xerrors

<!-- badges -->
[![CI](https://github.com/wood-jp/xerrors/actions/workflows/ci.yml/badge.svg)](https://github.com/wood-jp/xerrors/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/wood-jp/xerrors/badge.svg?branch=main)](https://coveralls.io/github/wood-jp/xerrors?branch=main)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/wood-jp/xerrors)](https://github.com/wood-jp/xerrors/releases)
<!-- /badges -->

Wrap any error with any data stucture using generics; automatically log that data, or extract directly it later.

> This library is in active development. Don't use it in production until this note is gone.

## Installation

Go 1.26.1 or later.

```bash
go get github.com/wood-jp/xerrors
```

## Core package

### Attaching data to an error

`Extend` wraps an error with a value of any type. Passing `nil` returns `nil`.

```go
type RequestContext struct {
    UserID    string
    RequestID string
}

err := xerrors.Extend(RequestContext{UserID: "123", RequestID: "abc"}, originalErr)
```

### Getting it back out

`Extract` walks the error chain and returns the first value of the requested type. If the same type has been extended more than once, you get the outermost one.

```go
if rctx, ok := xerrors.Extract[RequestContext](err); ok {
    fmt.Println(rctx.UserID)
}
```

This works through multiple layers of wrapping:

```go
err := xerrors.Extend(myData, originalErr)
wrapped := fmt.Errorf("operation failed: %w", err)
data, ok := xerrors.Extract[MyData](wrapped) // still works
```

### Structured logging

`ExtendedError` implements `slog.LogValuer`, so logging a wrapped error works out of the box by walking the full chain and collecting everything into one flat structure:

```json
{
  "error": "something went wrong",
  "error_detail": {
    "class": "transient",
    "stacktrace": [...],
    "context": { "user_id": "123" }
  }
}
```

`xerrors.Log(err)` returns that as a ready-to-use `slog.Attr` with the key `"error"`. Just drop it into any slog call:

```go
logger.Error("request failed", xerrors.Log(err))
```

Data types contribute to `error_detail` by implementing `slog.LogValuer` and returning a group value. The attrs in that group are merged directly into `error_detail`. Types that don't implement `slog.LogValuer`, or whose `LogValue` doesn't resolve to a group, fall back to a single `"data"` key. See sub-packages for examples.

### Edge cases

- `Extend(nil)` returns nil
- If you extend the same type more than once, `Extract` returns the outermost one
- Type aliases are distinct: `type A int` and `type B int` don't match each other

> **WARNING:** This should not be used in conjuction with `errors.Join` as the resulting joined error may have unexpected behavior.

## Subpackages

### errclass

```text
github.com/wood-jp/xerrors/errclass
```

Attaches a severity class to an error so callers can decide whether to retry.

Classes are ordered by severity:

| Class | Description |
| --- | --- |
| `Nil` | No error (nil) |
| `Unknown` | Unclassified (zero value) |
| `Transient` | May succeed on retry |
| `Persistent` | Will not resolve on retry |
| `Panic` | Came from a recovered panic |

By default, `WrapAs` always applies the class unconditionally:

```go
// Wraps regardless of whether err already has a class
err := errclass.WrapAs(err, errclass.Transient)

class := errclass.GetClass(err)
if class == errclass.Transient {
    // retry
}
```

Two options let you restrict when wrapping happens:

```go
// Only classify if the error has no class yet — leaves already-classified errors alone
err = errclass.WrapAs(err, errclass.Persistent, errclass.WithOnlyUnknown())

// Only classify if the new class is more severe than the current one — useful for escalation
err = errclass.WrapAs(err, errclass.Panic, errclass.WithOnlyMoreSevere())
```

`Class` implements `slog.LogValuer`. It shows up as `"class": "transient"` in flat log output.

`errors.Join` is not supported. Class information on individual errors may be lost when combining into a joined error.

---

### errcontext

```text
github.com/wood-jp/xerrors/errcontext
```

Attaches `slog.Attr` key-value pairs to an error. Useful for carrying request-scoped fields through a call stack without threading them through every function signature.

```go
// Attach context
err := errcontext.Add(err, slog.String("user_id", "123"), slog.Int("attempt", 3))

// Add more later — the existing map is updated in place, no extra wrapper
err = errcontext.Add(err, slog.String("request_id", "abc"))

// Pull it out
ctx := errcontext.Get(err)
if ctx != nil {
    attrs := ctx.Flatten() // sorted by key for deterministic output
    slog.Info("request failed", attrs...)
}
```

`Context` implements `slog.LogValuer`. Attached keys appear under `"context"` in flat log output.

`Add` with nil returns nil. `Add` with no attrs is a no-op. Duplicate keys use last-write-wins. `errors.Join` is not supported.

---

### stacktrace

```text
github.com/wood-jp/xerrors/stacktrace
```

Captures a stack trace where `Wrap` is called and attaches it to the error. If the error already has a trace, `Wrap` is a no-op.

```go
err = stacktrace.Wrap(err)
```

Most likely, stack traces are only used in logging. `StackTrace` implements `slog.LogValuer`, and appears as a `"stacktrace"` array in flat log output.

However, if you wish to directly get at the stack trace data, you can pull the trace back out with `Extract`:

```go
if st := stacktrace.Extract(err); st != nil {
    // st is a []Frame with File, LineNumber, Function
}
```

Alternatively, if you don't want to capture any stack traces but want to keep the code around, just disable them globally:

```go
stacktrace.Disabled.Store(true)
```

This results in all `Wrap` calls becoming no-ops.

## Attribution

*Originally written by [wood-jp](https://github.com/wood-jp) at [Zircuit](https://www.zircuit.com/). Based on [zkr-go-common](https://github.com/zircuit-labs/zkr-go-common-public), MIT license.*
