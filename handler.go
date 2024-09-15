package peanut

import (
	"context"
)

// Handler interface defines the stage.
// Stage is a basic unit of work in peanut.
type Handler interface {
	Run(ctx context.Context, req Request) error
}

// Named is an interface that defines the name of the stage.
type Named interface {
	Name() string
}

// HandlerFunc is a function that implements Handler interface.
type HandlerFunc func(ctx context.Context, req Request) error

// Run invokes the handler function.
func (f HandlerFunc) Run(ctx context.Context, req Request) error {
	return f(ctx, req)
}

// Middleware is a function that wraps a handler.
type Middleware interface {
	Apply(next Handler) Handler
}

// MiddlewareFunc is a function that implements Middleware interface.
type MiddlewareFunc func(next Handler) Handler

// Apply invokes the middleware function.
func (f MiddlewareFunc) Apply(next Handler) Handler {
	return f(next)
}

// MiddlewareChain is a chain of middlewares.
type MiddlewareChain []Middleware

// Wrap wraps the given handler with the middlewares.
func (mws MiddlewareChain) Wrap(h Handler) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i].Apply(h)
	}

	return h
}
