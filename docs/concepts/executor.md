---
title: Executor
description: Why one type covers sequential, parallel, and nesting; why all constructors panic; why parallel panic propagates.
package: peanut
---

# Executor

For signatures and usage, see godoc. This doc is for the design calls.

## Decision: one Executor type, two run modes

`Sequential` and `Parallel` are not separate types. They flip an internal
`mode` field on the same `Executor[S]` struct.

**Considered:**

1. *Two structs, `SeqExecutor` and `ParExecutor`.* Doubles surface area for
   what is structurally one thing — both hold tasks, both have middleware,
   both have a Def, both satisfy Task.
2. *Top-level composer functions* `Sequential(tasks...)` and
   `Parallel(tasks...)` returning anonymous tasks, with `Executor` as a
   separate concept. The original design. Two abstractions covering the same
   ground.
3. *Mode-as-Option* — a `Concurrent()` Option that flipped the executor's
   internal mode. The cleanest-looking API at the call site, but the
   implementation lied: an `Option` is documented as "configures `Def`," and
   `Concurrent` didn't — it mutated a private field via a covert unexported
   `modeSetter` interface that the constructor type-asserted on. Smell.

**Why two constructors won:** mode is set at construction and never changes
during life. The choice belongs at the call site — `Parallel(name)` reads
better than `New(name, Concurrent())`. No covert interfaces.

**Costs:** can't switch a built executor's mode. If you want both, build
two. Has not come up in practice.

## Decision: `*Executor[S]` satisfies `Task[S]`

A compile-time assertion (`var _ Task[struct{}] = (*Executor[struct{}])(nil)`)
guarantees the satisfaction. Add executors to other executors.

**Considered:** a separate "pipeline" type that holds executors but isn't
one. Adds an abstraction layer for what is again structurally the same
thing — something with a name that runs and returns an error.

**Why nesting won:** one composition primitive. The assertion is load-bearing
documentation — any drift in `Run`'s signature or `Def`'s return type breaks
compilation at the assertion line, not silently at call sites.

**Costs:** `Executor`'s public surface mixes the Task contract (`Run`,
`Def`) with lifecycle methods (`Add`, `Use`). godoc readers see a wide API.
Documented; the smell is real but small.

## Decision: all three constructors panic on bad options

`New`, `Sequential`, and `Parallel` all return `*Executor[S]`. None of them
return an error. Bad option → panic.

**Considered:**

1. *`New` returns `(*Executor, error)`; `Sequential`/`Parallel` panic.*
   Mixed ergonomics — which constructor is the "real" one and which is sugar?
   Not obvious.
2. *All return `error`.* Forces `, err :=` boilerplate at every call site,
   even though options are almost always constants.

**Why all panic:** bad options at construction are programmer bugs, not
runtime conditions. `peanut.Timeout(-1)` is a typo, not a recoverable error.
Go's convention for this is panic (see `regexp.MustCompile`,
`template.Must`, `sql.Register`). Uniform panic across all three means the
choice between them is purely about naming clarity.

**Costs:** runtime-driven options (rare) need pre-validation. We accept
that — it's the right place to validate anyway.

## Decision: Run is repeatable; mutations between Runs are honored

Calling `Run` twice runs twice. `Add` or `Use` between runs takes effect on
the next run.

**Considered:** freeze after first Run (panic on Add/Use after); treat
executor as one-shot.

**Why mutable:** an executor is structurally just two slices and a Def.
Freezing creates a state machine where there was a simple struct. The naive
behavior matches the obvious mental model.

**Costs:** users who assume immutability could be surprised. The
`Executor.Run` GoDoc states the contract explicitly.

## Decision: parallel panic propagates, no recover

A panic in a `Parallel` goroutine crashes the process. Sequential mode
unwinds to the caller as usual.

**Considered:** recover in each goroutine and convert to an error; recover
and re-panic on the main goroutine.

**Why no recover:** peanut is a task runner, not a supervisor framework.
Panics are bugs. Silently converting to error masks the bug; it doesn't fix
anything. Users who need panic survival can add a 10-line middleware that
does `defer recover()` and apply it via `Use` — that's the right scope for
the concern, not "in the framework forever."

**Costs:** a single buggy task crashes the program. Same reason Go panics
crash the program — "the assumptions are broken, fail loud."

## Decision: timeouts via `context.WithTimeout` at both task and executor level

If `Def.Timeout > 0`, the relevant `Run` wraps `ctx` in
`context.WithTimeout`. Nested timeouts compose naturally — inner deadline
wins when shorter.

**Considered:** custom timeout mechanism with explicit cancellation; goroutine-watchdog
patterns.

**Why ctx:** standard library, zero allocation overhead per cancel,
composes with anything else in the Go ecosystem that respects ctx. Tasks
that don't `select` on `ctx.Done()` can still be timed out (the executor
returns the deadline error), they just won't preempt mid-work — that's Go,
not us.

**Costs:** task authors must respect `ctx.Done()` for prompt
cancellation. There's no preemption.
