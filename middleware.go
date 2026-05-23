package peanut

// Middleware wraps a Task to add cross-cutting behavior (logging, tracing,
// retries, ...). Wrappers should preserve identity via WithDef(next.Def(), ...).
type Middleware[S any] func(Task[S]) Task[S]

// Wrap applies middleware to a single task, innermost-first.
func Wrap[S any](task Task[S], mw ...Middleware[S]) Task[S] {
	for i := len(mw) - 1; i >= 0; i-- {
		task = mw[i](task)
	}
	return task
}
