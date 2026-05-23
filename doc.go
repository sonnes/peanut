// Package peanut is a small, generic task runner for Go.
//
// A peanut program is built from three things:
//
//   - [Task] — a unit of work that reads from and writes to a shared state S.
//   - [Executor] — runs a sequence of tasks, applies middleware, attributes errors.
//   - [Middleware] — wraps task invocations to add logging, tracing, retries, etc.
//
// The shared state S is fully generic; peanut imposes no shape on it. Tasks mutate
// state directly (typically S is a pointer type), and the executor threads that
// state through the chain.
//
// # Quick start
//
//	type State struct{ Result string }
//
//	greet := peanut.MustDefine[*State]("greet", func(_ context.Context, s *State) error {
//	    s.Result = "hello"
//	    return nil
//	})
//
//	exec := peanut.New[*State]("pipeline")
//	exec.Add(greet)
//
//	state := &State{}
//	_ = exec.Run(context.Background(), state)
//	fmt.Println(state.Result) // hello
//
// # Three constructors
//
// [New] and [Sequential] are synonyms that construct a sequential Executor;
// [Parallel] constructs a parallel one (first-error-wins, siblings canceled).
// All three return *Executor[S] and panic on option validation errors — bad
// options are programmer bugs, not runtime conditions.
//
// An *Executor is itself a [Task], so it can be nested as a sub-task of another
// executor for the same state type. This is the only composition primitive
// peanut needs.
//
// # Middleware propagation
//
// Middleware registered via [Executor.Use] wraps every task at every nesting
// level. Internally, each Executor.Run appends its middleware to a chain stored
// in ctx; nested executors layer their middleware on top of the outer chain.
// Custom composers can opt into this propagation by invoking children via
// [RunWithMiddleware] instead of task.Run directly.
//
// # Error attribution
//
// When a task fails inside an [Executor.Run], the child's error is wrapped in
// *[Error] carrying the child's Def.Name. errors.Is and errors.As traverse the
// wrap, so callers can still match against the underlying error.
//
// # Timeouts
//
// [Timeout] is an option that applies to both tasks and executors. On a task,
// it bounds that task's Run; on an executor, it bounds the whole Run.
package peanut
