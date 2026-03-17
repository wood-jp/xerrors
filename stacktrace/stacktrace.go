// Package stacktrace uses the go runtime to capture stack trace data.
package stacktrace

import (
	"log/slog"
	"regexp"
	"runtime"
	"strings"
)

const (
	maxFrames     = 50
	runtimePrefix = "runtime."
	testingPrefix = "testing."
)

// match the filename of the Go runtime package
// eg `/pkg/mod/golang.org/toolchain@v0.0.1-go1.22.4.linux-amd64/src/runtime/panic.go`
var runtimeRegex = regexp.MustCompile(`go[^/]*/src/runtime/[^.]+\.go`)

// match the filename of the Go testing package
var testingRegex = regexp.MustCompile(`go[^/]*/src/testing/[^.]+\.go`)

// Frame represents human-readable information about a single frame in a stack trace.
type Frame struct {
	// File is the source file path of the frame.
	File string `json:"source"`
	// LineNumber is the line number within File where the call was made.
	LineNumber int `json:"line"`
	// Function is the fully-qualified function name of the frame.
	Function string `json:"func"`
}

// StackTrace represents a program stack trace as a series of frames.
type StackTrace []Frame

// LogValue implements [slog.LogValuer].
// It returns a group containing a single "stacktrace" attr whose value is an
// array of frame objects, each with "func", "line", and "source" keys.
//
// Each frame is represented as map[string]any rather than [slog.GroupValue] because
// slog handlers only resolve [slog.LogValuer] at the top level of an attribute value —
// they do not recursively resolve [slog.Value] elements nested inside a []slog.Value
// wrapped in [slog.AnyValue]. JSON and text handlers would encode those as empty
// objects ({}). map[string]any is handled correctly by encoding/json and produces the
// expected key-value output.
func (st StackTrace) LogValue() slog.Value {
	frames := make([]any, len(st))
	for i, frame := range st {
		frames[i] = map[string]any{
			"func":   frame.Function,
			"line":   frame.LineNumber,
			"source": frame.File,
		}
	}
	return slog.GroupValue(slog.Any("stacktrace", slog.AnyValue(frames)))
}

// GetStack captures the current program stack trace and returns it as a [StackTrace].
// skipFrames controls how many frames to skip: passing 1 makes GetStack itself the first captured frame.
// When skipRuntime is true, frames from the Go runtime (e.g. runtime.main, runtime.panic)
// and the testing package are omitted from the result.
func GetStack(skipFrames int, skipRuntime bool) StackTrace {
	pc := make([]uintptr, maxFrames)
	n := runtime.Callers(skipFrames, pc)
	pc = pc[:n]

	stackTrace := make(StackTrace, 0, n)
	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		if skipRuntime {
			if strings.HasPrefix(frame.Function, runtimePrefix) && runtimeRegex.MatchString(frame.File) {
				continue
			} else if strings.HasPrefix(frame.Function, testingPrefix) && testingRegex.MatchString(frame.File) {
				continue
			}
		}
		stackTrace = append(stackTrace, Frame{
			File:       frame.File,
			LineNumber: frame.Line,
			Function:   frame.Function,
		})
	}

	return stackTrace
}
