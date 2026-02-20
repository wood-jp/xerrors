# errclass

Error classification by severity level, built on top of `xerrors`.

## Overview

`errclass` attaches a severity `Class` to errors using `xerrors.Extend`, allowing downstream callers to inspect and act on how severe an error is.

## Severity Levels

Classes are ordered by severity (lowest to highest):

| Class | Description |
|---|---|
| `Nil` | No error (nil) |
| `Unknown` | Error has not been classified (zero value) |
| `Transient` | Temporary error that may succeed on retry |
| `Persistent` | Permanent error that will not resolve on retry |
| `Panic` | Error resulting from a recovered panic |

## Usage

### Classifying an error

```go
err := errclass.WrapAs(err, errclass.Transient)
```

### Inspecting the class

```go
class := errclass.GetClass(err)
if class == errclass.Transient {
    // retry
}
```

### Conditional wrapping

`WrapAs` accepts options that control when wrapping occurs:

```go
// Only classify if the error has no class yet
err = errclass.WrapAs(err, errclass.Persistent, errclass.WithOnlyUnknown())

// Only classify if the new class is more severe than the current one
err = errclass.WrapAs(err, errclass.Panic, errclass.WithOnlyMoreSevere())
```

### Structured logging

`Class` implements `slog.LogValuer`, so the class is included automatically when you log a classified error:

```go
slog.Error("request failed", slog.Any("error", err))
// Output includes "class":"transient" within the error structure
```

## Limitations

Joined errors created with `errors.Join` are not supported. Because `errors.Join` returns an unexported `joinError` type that does not implement `slog.LogValuer`, class information attached to the individual errors is lost when logging the joined error. Additionally, `GetClass` will only return the class of the first error in the join, and the result changes depending on the order the errors were joined.
