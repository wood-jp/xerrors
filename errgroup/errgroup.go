// Package errgroup provides synchronization, error propagation, and Context
// cancellation for groups of goroutines working on subtasks of a common task.
// It wraps [golang.org/x/sync/errgroup], with the addition that goroutines
// launched via [Group.Go] and [Group.TryGo] are wrapped with [calm.Unpanic],
// so any panic is recovered and returned as an error rather than crashing the
// program.
package errgroup

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/wood-jp/xerrors/calm"
)

// Group is a collection of goroutines working on subtasks that are part of the
// same overall task.
//
// A zero Group is valid and does not cancel on error by default. A Group should
// not be reused for different tasks.
type Group struct {
	group *errgroup.Group
}

// New returns a new Group with no associated context.
func New() *Group {
	return &Group{group: new(errgroup.Group)}
}

// WithContext returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func WithContext(ctx context.Context) (*Group, context.Context) {
	group, ctx := errgroup.WithContext(ctx)
	return &Group{group: group}, ctx
}

// Go calls the given function in a new goroutine. Panics inside f are
// recovered by [calm.Unpanic] and returned as errors.
//
// The first call to return a non-nil error (or panic) cancels the group's context, if the
// group was created by calling WithContext. The error is then returned by Wait.
//
// Go blocks until the new goroutine can be added without exceeding the
// configured limit.
func (g *Group) Go(f func() error) {
	g.group.Go(func() error {
		return calm.Unpanic(f)
	})
}

// SetLimit limits the number of active goroutines in this group to at most n.
// A negative value indicates no limit. A zero value prevents new goroutines
// from starting.
func (g *Group) SetLimit(n int) {
	g.group.SetLimit(n)
}

// TryGo calls the given function in a new goroutine only if the number of
// active goroutines in the group is currently below the configured limit.
// Panics inside f are recovered by [calm.Unpanic] and returned as errors.
//
// The return value reports whether the goroutine was started. If TryGo would
// exceed the group's limit, it returns false without calling f.
func (g *Group) TryGo(f func() error) bool {
	return g.group.TryGo(func() error {
		return calm.Unpanic(f)
	})
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	return g.group.Wait()
}
