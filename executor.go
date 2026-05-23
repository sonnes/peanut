package peanut

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// An Executor is itself a Task[S], so it can be nested as a sub-task of
// another Executor for the same state type.
var _ Task[struct{}] = (*Executor[struct{}])(nil)

// runMode controls how an Executor iterates its tasks.
type runMode int

const (
	modeSequential runMode = iota
	modeParallel
)

// Executor runs tasks against a shared state of type S. New (and Sequential)
// produce a sequential executor; Parallel produces a parallel one.
type Executor[S any] struct {
	def        Def
	mode       runMode
	tasks      []Task[S]
	middleware []Middleware[S]
}

// New constructs a sequential Executor for state type S, identified by name.
// Panics on option validation error — bad options are programmer bugs, not
// runtime conditions. [Sequential] is a synonym; [Parallel] constructs a
// parallel executor.
func New[S any](name string, opts ...Option) *Executor[S] {
	return buildExecutor[S]("New", name, modeSequential, opts...)
}

// buildExecutor is the internal shared constructor. Panics on option
// validation failure, prefixed with the caller's label.
func buildExecutor[S any](caller, name string, mode runMode, opts ...Option) *Executor[S] {
	def := Def{Name: name}
	for _, opt := range opts {
		if err := opt.apply(&def); err != nil {
			panic(fmt.Errorf("peanut.%s(%q): %w", caller, name, err))
		}
	}
	return &Executor[S]{def: def, mode: mode}
}

// Def returns the executor's definition.
func (e *Executor[S]) Def() Def { return e.def }

// Add appends tasks to the executor in the order they should run.
func (e *Executor[S]) Add(tasks ...Task[S]) *Executor[S] {
	e.tasks = append(e.tasks, tasks...)
	return e
}

// Use appends middleware. Middleware applies to every task in Use-registration
// order: outermost wraps first, innermost wraps last (so a, b, c => a(b(c(task)))).
// When this executor is nested inside another, the outer executor's middleware
// propagates and wraps this executor's children too; the inner executor's
// own middleware is layered on top of the outer chain.
func (e *Executor[S]) Use(mw ...Middleware[S]) *Executor[S] {
	e.middleware = append(e.middleware, mw...)
	return e
}

// Run executes the executor's tasks against state. If the mode is sequential
// (default), tasks run in order and stop on the first error. If the mode is
// parallel ([Parallel]), tasks run concurrently and the first task to error
// wins; siblings are canceled via ctx. In either mode, a child's error is
// wrapped in *Error carrying the child's Def.Name. If Def.Timeout > 0, the
// whole Run is bounded by it.
//
// Run is repeatable. Tasks or middleware added between calls to Run are
// included in subsequent runs.
//
// A panic inside a task body propagates. In sequential mode the panic
// unwinds to the caller. In parallel mode the panic occurs inside a
// goroutine and will crash the process unless your task body recovers
// itself — peanut does not install a recover handler.
func (e *Executor[S]) Run(ctx context.Context, state S) error {
	ctx, cancel := withTimeout(ctx, e.def.Timeout)
	defer cancel()
	ctx = appendMiddleware(ctx, e.middleware)
	if e.mode == modeParallel {
		return e.runParallel(ctx, state)
	}
	return e.runSequential(ctx, state)
}

func (e *Executor[S]) runSequential(ctx context.Context, state S) error {
	for _, t := range e.tasks {
		if err := RunWithMiddleware(ctx, t, state); err != nil {
			return attribute(t.Def().Name, err)
		}
	}
	return nil
}

func (e *Executor[S]) runParallel(ctx context.Context, state S) error {
	if len(e.tasks) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		firstErr error
		once     sync.Once
		wg       sync.WaitGroup
	)
	for _, t := range e.tasks {
		wg.Add(1)
		go func(t Task[S]) {
			defer wg.Done()
			if err := RunWithMiddleware(ctx, t, state); err != nil {
				once.Do(func() {
					firstErr = attribute(t.Def().Name, err)
					cancel()
				})
			}
		}(t)
	}
	wg.Wait()
	return firstErr
}

// withTimeout returns ctx with a deadline applied if d > 0, plus its cancel.
// If d <= 0, ctx is returned unchanged with a no-op cancel.
func withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// middlewareChainKey is a generic context key holding the accumulated
// middleware chain for state type S. Each Executor.Run appends its own
// middleware so nested executors layer on top of their parents' chains.
type middlewareChainKey[S any] struct{}

func appendMiddleware[S any](ctx context.Context, mws []Middleware[S]) context.Context {
	if len(mws) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(middlewareChainKey[S]{}).([]Middleware[S])
	combined := make([]Middleware[S], 0, len(existing)+len(mws))
	combined = append(combined, existing...)
	combined = append(combined, mws...)
	return context.WithValue(ctx, middlewareChainKey[S]{}, combined)
}

func middlewareFromContext[S any](ctx context.Context) []Middleware[S] {
	mws, _ := ctx.Value(middlewareChainKey[S]{}).([]Middleware[S])
	return mws
}

// RunWithMiddleware invokes task wrapped by the ambient middleware chain
// (set by enclosing Executor.Run calls). Custom composers should use this
// instead of task.Run directly so executor-registered middleware propagates
// to their children.
func RunWithMiddleware[S any](ctx context.Context, task Task[S], state S) error {
	mws := middlewareFromContext[S](ctx)
	if len(mws) == 0 {
		return task.Run(ctx, state)
	}
	return Wrap(task, mws...).Run(ctx, state)
}
