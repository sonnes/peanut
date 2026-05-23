---
title: Middleware
description: Why middleware is a function type, why it propagates through nesting, why RunWithMiddleware is exported.
package: peanut
---

# Middleware

For signatures and usage, see godoc. This doc is for the design calls.

## Decision: middleware is a function type, not an interface

`type Middleware[S any] func(Task[S]) Task[S]`. No struct, no factory, no
lifecycle.

**Considered:**

1. *Interface with `Hooks` bundle* (genkit-style). A middleware is a struct
   with `Name()` and `New(ctx) (*Hooks, error)` returning per-call hook
   functions for distinct operations (model call, tool call, etc.). Cleaner
   per-call state, typed hook-specific parameters.
2. *Interface with a single `RunNext` method* (older peanut design).
   Polymorphic dispatch, easier to mock.

**Why function type:** matches the net/http and grpc-go middleware idiom.
Composing is trivial — `Wrap` is six lines. No factory pattern, no plugin
registration, no per-middleware lifecycle to reason about. Per-call state
lives in closures over middleware-local variables, which is also the Go
idiom.

**Costs:** middlewares that need substantial per-call mutable state shared
across multiple wrapped invocations have to build a stateful closure
themselves. genkit's `Hooks{...}` is more ergonomic for that. Peanut's
target use cases (logging, tracing, retries, timing) don't need it, so
we accept the trade.

## Decision: middleware propagates through nesting via ctx accumulation

Every `Executor.Run` appends its middleware onto a slice stored in `ctx`
under a generic key. Nested executors append on top of whatever's already
there. `RunWithMiddleware` reads the accumulated chain when invoking each
child.

**Considered:**

1. *Isolation* (genkit-style): each executor's middleware applies only to
   its own children. Outer middleware doesn't reach inner. Compositionally
   clean — a retry middleware on the outer doesn't accidentally retry inner
   work.
2. *Single executor in ctx* (v1 peanut): one executor pointer stored in
   ctx; composers look it up and use only that executor's middleware.
   Works for composer functions but breaks for nested executors, which
   would overwrite the ctx pointer with themselves.
3. *No propagation*: middleware applies only to top-level tasks the
   executor was given. Loses observability on nested children entirely.

**Why accumulation:** the common case is "I want my logging middleware to
see everything." That should be one `Use` call, not one per nesting level.
Inner executors can still add their own middleware on top — they're
additive, not replacing.

**Costs:** outer middleware also wraps nested-executor-as-task AND each
nested child individually. For most middleware (logging, tracing, timing)
this is desired — every observable thing gets seen. For middleware that
implements policies (retry, fallback), it can be surprising — a retry on
the outer would retry sub-pipelines AND each leaf inside them. Fix is to
put the policy middleware on the inner executor where it applies. Document.

## Decision: `RunWithMiddleware` is exported

The helper that reads the ambient chain and wraps a task with it is public.

**Considered:** keep unexported. Users writing custom composers (their own
fan-out patterns, batching loops, concurrency-limited pools) would not get
middleware propagation through their composer.

**Why exported:** peanut can't anticipate every composer pattern users want
to build. Without this hook, users either rewrite the ctx machinery
themselves (private package state, can't), or accept that their custom
composers break middleware propagation. Exporting it is the smallest
contract that closes the gap.

**Costs:** one more symbol in the public API. Tiny.

## Decision: middleware preserves identity via `WithDef`

The convention is `return WithDef(next.Def(), func(ctx, s) error { ... })`.
A middleware that returns an anonymous task with empty `Def` makes
observability blank for the wrapped task downstream.

**Considered:** enforcing identity propagation in `Wrap` itself by
re-attaching `next.Def()` to whatever the middleware returns.

**Why convention not enforcement:** middleware sometimes *wants* to alter
the Def (e.g., rename `"fetch-tweets"` to `"fetch-tweets[cached]"` when a
cache hit happened). Enforcing identity preservation forecloses that. The
convention is documented in the `Middleware[S]` GoDoc and in every
example.

**Costs:** middleware authors must remember the discipline. New
contributors get caught by it once and learn. The fast feedback (empty
`Def.Name` in logs) makes the bug self-evident.

## Decision: `Wrap` chains outermost-first

`Wrap(task, A, B, C)` produces `A(B(C(task)))`. `A` runs first on entry,
last on exit.

**Why this order:** matches the natural reading of `Use(A, B, C)` — the
order you register middleware is the order they run on entry. Same
convention as `chi`, `negroni`, net/http stacks, grpc-go interceptors.

**Costs:** the implementation reverses its loop, which can confuse
first-time readers of `Wrap`. Comment in the source calls out why.
