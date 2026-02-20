# xerrors

A modern Go library for extending errors with typed contextual data using generics.

*Please note this library will be undergoing change and should probably not be used prior to a proper release (and the removal of this note).*

## Features

- **Generic error extension**: Attach any data type to an error
- **Type-safe extraction**: Retrieve attached data by type from wrapped errors
- **Error chain compatible**: Works with `errors.Is`, `errors.Unwrap`, and nested wrapping
- **Structured logging support**: Implements `slog.LogValuer` for clean log output
- **Zero dependencies**: Uses only the standard library

## Subpackages

- [`errclass`](errclass/README.md) - Error classification by severity level (transient, persistent, panic, etc.)

## Requirements

- Go 1.26+

## Installation

```bash
go get github.com/wood-jp/xerrors
```

## Usage

### Extending an error with data

```go
type RequestContext struct {
    UserID    string
    RequestID string
}

rctx := RequestContext{UserID: "123", RequestID: "abc"}
err := xerrors.Extend(rctx, originalErr)
```

### Extracting data from an error

```go
if rctx, ok := xerrors.Extract[RequestContext](err); ok {
    // rctx contains the attached RequestContext
    fmt.Println(rctx.UserID)
}
```

### Works with error wrapping

Extended data can be extracted even through multiple layers of error wrapping:

```go
err := xerrors.Extend(myData, originalErr)
wrapped := fmt.Errorf("operation failed: %w", err)
doubleWrapped := fmt.Errorf("handler error: %w", wrapped)

// still returns myData
data, ok := xerrors.Extract[MyData](doubleWrapped)
```

### Multiple data types

You can extend an error with multiple different data types:

```go
err := xerrors.Extend(userData, originalErr)
err = xerrors.Extend(requestData, err)

// Extract each type independently
user, ok := xerrors.Extract[UserData](err)
request, ok := xerrors.Extract[RequestData](err)
```

### Structured logging

`ExtendedError` implements `slog.LogValuer`, so it logs cleanly:

```go
err := xerrors.Extend(data, originalErr)
slog.Error("operation failed", "error", err)
// Output includes both the error message and the attached data
```

## Behavior Notes

- Extending `nil` returns `nil`
- If the same type is extended multiple times, `Extract` returns the outermost (most recently added) value
- Different type aliases (e.g., `type A int` vs `type B int`) are treated as distinct types

---

*Originally authored by [wood-jp](https://github.com/wood-jp) at [Zircuit](https://www.zircuit.com/). Based on work from [zkr-go-common](https://github.com/zircuit-labs/zkr-go-common-public), licensed under the MIT License.*
