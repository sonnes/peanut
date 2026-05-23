package peanut

// Sequential constructs a sequential Executor — synonym for [New] with a more
// explicit name. Panics on option validation error.
func Sequential[S any](name string, opts ...Option) *Executor[S] {
	return buildExecutor[S]("Sequential", name, modeSequential, opts...)
}

// Parallel constructs a parallel Executor (tasks run concurrently). Panics on
// option validation error. Queue children with .Add and middleware with .Use
// after construction.
func Parallel[S any](name string, opts ...Option) *Executor[S] {
	return buildExecutor[S]("Parallel", name, modeParallel, opts...)
}
