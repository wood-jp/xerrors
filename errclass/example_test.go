package errclass_test

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/wood-jp/xerrors"
	"github.com/wood-jp/xerrors/errclass"
)

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func ExampleWrapAs() {
	err := errclass.WrapAs(errors.New("connection timed out"), errclass.Transient)
	newLogger().Error("operation failed", xerrors.Log(err))
	// Output:
	// {"level":"ERROR","msg":"operation failed","error":{"error":"connection timed out","error_detail":{"class":"transient"}}}
}

func ExampleGetClass() {
	err := errors.New("disk full")
	err = errclass.WrapAs(err, errclass.Persistent)
	fmt.Println(errclass.GetClass(err))
	// Output:
	// persistent
}

func ExampleGetClass_nil() {
	fmt.Println(errclass.GetClass(nil))
	// Output:
	// nil
}

func ExampleWithOnlyMoreSevere() {
	err := errors.New("failure")
	err = errclass.WrapAs(err, errclass.Persistent)
	// Transient is less severe than Persistent, so the class is unchanged.
	err = errclass.WrapAs(err, errclass.Transient, errclass.WithOnlyMoreSevere())
	fmt.Println(errclass.GetClass(err))
	// Output:
	// persistent
}

func ExampleWithOnlyUnknown() {
	err := errors.New("failure")
	err = errclass.WrapAs(err, errclass.Transient)
	// Error already has a class, so WithOnlyUnknown leaves it unchanged.
	err = errclass.WrapAs(err, errclass.Persistent, errclass.WithOnlyUnknown())
	fmt.Println(errclass.GetClass(err))
	// Output:
	// transient
}
