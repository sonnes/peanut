package peanut

import (
	"context"
)

// Executor manages the execution lifecycle of
// stages.
type Executor struct {
	stages     []Handler
	middleware MiddlewareChain
}

// New creates a new Executor.
func New() *Executor {
	return &Executor{
		stages:     make([]Handler, 0),
		middleware: make([]Middleware, 0),
	}
}

// Use adds middlewares to the executor.
// Middlewares are applied to all stages run
// by this executor.
func (e *Executor) Use(middlewares ...Middleware) {
	e.middleware = append(e.middleware, middlewares...)
}

// Add adds one or more stages to the executor.
// Stages are executed in the order they are added.
func (e *Executor) Add(stages ...Handler) {
	e.stages = append(e.stages, stages...)
}

// Run executes all stages in the executor.
func (e *Executor) Run(ctx context.Context, req Request) error {
	ctx = withExecutor(ctx, e)

	for _, stage := range e.stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			s := e.middleware.Wrap(stage)

			err := s.Run(ctx, req)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type ctxKey int

const (
	executorKey ctxKey = iota
)

func withExecutor(ctx context.Context, e *Executor) context.Context {
	return context.WithValue(ctx, executorKey, e)
}

// RunWithContext executes the given stage using the executor
// in the context.
//
// Meta stages must use this function to execute stages.
func RunWithContext(ctx context.Context, stage Handler, req Request) error {
	s := stage

	e, ok := ctx.Value(executorKey).(*Executor)
	if ok {
		s = e.middleware.Wrap(stage)
	}

	return s.Run(ctx, req)
}
