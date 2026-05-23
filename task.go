package peanut

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"
)

// Def carries identity (Name, Description) and behavioral settings (Timeout)
// for tasks and executors. Middleware reads it via Task.Def to log, trace, or
// attribute errors; Executor exposes its own Def for the same reasons.
type Def struct {
	Name        string
	Description string
	Timeout     time.Duration
}

// Task is a unit of work that reads from and writes to a shared state S.
// Every Task carries a Def accessible via Def().
type Task[S any] interface {
	Run(ctx context.Context, s S) error
	Def() Def
}

// TaskFunc is the bare function shape of a task body. Use DefineTask to turn
// one into a Task with identity, or WithDef inside middleware to preserve
// identity when wrapping.
type TaskFunc[S any] = func(ctx context.Context, s S) error

// definedTask carries a Def and a body.
type definedTask[S any] struct {
	def Def
	fn  TaskFunc[S]
}

func (d *definedTask[S]) Run(ctx context.Context, s S) error {
	ctx, cancel := withTimeout(ctx, d.def.Timeout)
	defer cancel()
	return d.fn(ctx, s)
}

func (d *definedTask[S]) Def() Def { return d.def }

// DefineTask wraps fn into a Task identified by name.
func DefineTask[S any](name string, fn TaskFunc[S], opts ...Option) (Task[S], error) {
	def := Def{Name: name}
	for _, opt := range opts {
		if err := opt.apply(&def); err != nil {
			return nil, fmt.Errorf("peanut.DefineTask(%q): %w", name, err)
		}
	}
	return &definedTask[S]{def: def, fn: fn}, nil
}

// MustDefine is like DefineTask but panics on option error. Use it for
// package-level task definitions where a bad option is a programmer bug.
// Modeled on regexp.MustCompile and template.Must.
func MustDefine[S any](name string, fn TaskFunc[S], opts ...Option) Task[S] {
	t, err := DefineTask(name, fn, opts...)
	if err != nil {
		panic(err)
	}
	return t
}

// WithDef wraps fn into a Task carrying def. Use this inside middleware to
// preserve the identity of the task being wrapped.
func WithDef[S any](def Def, fn TaskFunc[S]) Task[S] {
	return &definedTask[S]{def: def, fn: fn}
}

// DefineFunc wraps fn into a Task whose Def.Name is derived from fn via
// reflection. For named top-level functions and methods this gives a useful
// qualified name like "pkg.FetchTweets" or "pkg.(*Service).FetchTweets-fm".
// For closures and inline lambdas the name is a compiler-internal label like
// "pkg.parent.func1" — pass peanut.Name("...") to override when you need a
// stable, intentional label. Panics on option validation error.
func DefineFunc[S any](fn TaskFunc[S], opts ...Option) Task[S] {
	def := Def{Name: funcName(fn)}
	for _, opt := range opts {
		if err := opt.apply(&def); err != nil {
			panic(fmt.Errorf("peanut.DefineFunc(%q): %w", def.Name, err))
		}
	}
	return &definedTask[S]{def: def, fn: fn}
}

// funcName recovers fn's qualified name via runtime reflection. Returns ""
// if reflection fails (nil func, stripped binary, etc.).
func funcName(fn any) string {
	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return ""
	}
	rf := runtime.FuncForPC(v.Pointer())
	if rf == nil {
		return ""
	}
	return rf.Name()
}
