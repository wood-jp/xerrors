# xerrors

<!-- badges -->
[![Go Version](https://img.shields.io/github/go-mod/go-version/wood-jp/xerrors)](https://pkg.go.dev/github.com/wood-jp/xerrors)
[![CI](https://github.com/wood-jp/xerrors/actions/workflows/ci.yml/badge.svg)](https://github.com/wood-jp/xerrors/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/wood-jp/xerrors/badge.svg?branch=main)](https://coveralls.io/github/wood-jp/xerrors?branch=main)
[![Release](https://img.shields.io/github/v/release/wood-jp/xerrors)](https://github.com/wood-jp/xerrors/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/wood-jp/xerrors)](https://goreportcard.com/report/github.com/wood-jp/xerrors)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/wood-jp/xerrors.svg)](https://pkg.go.dev/github.com/wood-jp/xerrors)
<!-- /badges -->

Wrap any error with any data stucture using generics; automatically log that data, or extract directly it later. Loggable stacktraces out of the box.

- [Stability](#stability)
- [Installation](#installation)
- [Core package](#core-package)
- [Subpackages](#subpackages)
  - [errclass](#errclass)
  - [errcontext](#errcontext)
  - [stacktrace](#stacktrace)
- [Performance](#performance)
- [Contributing](#contributing)
- [Security](#security)
- [Attribution](#attribution)

## Stability

v1.x releases make no breaking changes to exported APIs. New functionality may be added in minor releases; patches are bug fixes, or administrative work only.

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

`StackTrace` implements `slog.LogValuer`, and appears as a `"stacktrace"` array in flat log output. For example:

```go
var errTest = errors.New("something went wrong")

func c() error {
    return stacktrace.Wrap(errclass.WrapAs(errTest, errclass.Transient))
}

func b() error { return c() }
func a() error { return b() }

err := a()
logger.Error("request failed", xerrors.Log(err))
```

Outputs a log similar to:

```json
{
  "level": "ERROR",
  "msg": "request failed",
  "error": {
    "error": "something went wrong",
    "error_detail": {
      "class": "transient",
      "stacktrace": [
        {"func": "main.c", "line": 16, "source": "main.go"},
        {"func": "main.b", "line": 20, "source": "main.go"},
        {"func": "main.a", "line": 24, "source": "main.go"},
        {"func": "main.main", "line": 31, "source": "main.go"}
      ]
    }
  }
}
```

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

## Performance

Benchmarks cover the three operations users care about: stack capture, generic wrapping/extraction, and context attachment. Run them yourself with:

```bash
just bench
```

Results on an Intel Core Ultra 7 155H (Go 1.26.1, linux/amd64, `-count=3`):

```text
goos: linux
goarch: amd64
cpu: Intel(R) Core(TM) Ultra 7 155H

pkg: github.com/wood-jp/xerrors/stacktrace
BenchmarkWrap_New-22                	  804999	      1314 ns/op	     880 B/op	   5 allocs/op
BenchmarkWrap_Existing-22           	64795417	        19 ns/op	       0 B/op	   0 allocs/op
BenchmarkWrap_New_Deep-22           	  497211	      2772 ns/op	    1104 B/op	   5 allocs/op
BenchmarkWrap_Existing_Deep-22      	33198662	        31 ns/op	       0 B/op	   0 allocs/op

pkg: github.com/wood-jp/xerrors
BenchmarkExtend-22                  	34270780	        32 ns/op	      48 B/op	   1 allocs/op
BenchmarkExtract_Shallow-22         	89609863	        11 ns/op	       0 B/op	   0 allocs/op
BenchmarkExtract_Deep-22            	28955666	        42 ns/op	       0 B/op	   0 allocs/op
BenchmarkLog-22                     	 1398642	       857 ns/op	     960 B/op	  20 allocs/op

pkg: github.com/wood-jp/xerrors/errcontext
BenchmarkAdd_New-22                 	 5845732	       207 ns/op	     440 B/op	   4 allocs/op
BenchmarkAdd_Existing-22            	48651021	        22 ns/op	       0 B/op	   0 allocs/op
BenchmarkAdd_Existing_Deep-22       	35166369	        36 ns/op	       0 B/op	   0 allocs/op
BenchmarkFlatten-22                 	 2332621	       511 ns/op	     512 B/op	   8 allocs/op
```

As one might expect, call-depth (for stacktraces) and error-chain depth impact the actual costs. The "deep" benchmarks here only have depth/length of 5 for illustrative purposes.

Actually obtaining a stack trace is expensive, but only happens once in the call-chain. Re-wrapping an already-traced error is a no-op (aside walking the error chain).

Adding error context is also very cheap after the first. It also has an error-chain depth traversal cost if adding context at different call sites.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## Attribution

*This library is a simplified fork of one written by [wood-jp](https://github.com/wood-jp) at [Zircuit](https://www.zircuit.com/). The original code is available here: [zkr-go-common-public/xerrors](https://github.com/zircuit-labs/zkr-go-common-public/tree/main/xerrors)*
