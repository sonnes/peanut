---
title: Option
description: Why options are shared between tasks and executors, why apply returns error, why Concurrent was dropped.
package: peanut
---

# Option

For signatures and usage, see godoc. This doc is for the design calls.

## Decision: single `Option` interface, not `TaskOption` + `ExecOption`

Both `DefineTask` and the executor constructors accept `...Option`. Same
options. Same `Def`.

**Considered:** split into `TaskOption` (with `applyTask(*TaskDef)`) and
`ExecOption` (with `applyExec(*ExecDef)`). The compiler would prevent
passing a Task-only option to an executor and vice versa.

**Why unified:** tasks and executors share the same fields — `Name`,
`Description`, `Timeout`. Every option that exists today applies meaningfully
to both. The split was solving a problem we didn't have, with two-of-everything
boilerplate as the cost.

**Costs:** if a future option only makes sense for tasks (e.g.,
`Retries(n)` on a leaf, but meaningless on a composer), it'd silently
no-op or behave unexpectedly when passed to the wrong target. We'll
re-introduce the split if/when that lands. Not preemptively.

## Decision: `apply` returns an error

The unexported method is `apply(*Def) error`, not `apply(*Def)`.

**Considered:** signature without error, with each option panicking from
inside `apply` on bad input.

**Why error:** the error path lets the constructor add context ("peanut.New
(\"pipeline\"): timeout must be non-negative ..."). Panicking from inside
`apply` would either lose that context or require every option to know its
caller. Today all constructors panic on apply error, but the error path
exists in case we ever want to surface option errors (a hypothetical
`NewSafe[S]` for runtime-driven options).

**Costs:** every option implementation returns `error`, even ones that
can't fail. Trivial boilerplate.

## Decision: `Option` is a sealed interface

The `apply` method is unexported. External packages can't implement
`Option`.

**Why sealed:** option implementations mutate `*Def` directly. Letting
third-party code do that means they can stash arbitrary state in fields
peanut doesn't know about, breaking observability invariants. If a user
needs a custom configuration mechanism, they build it on top — e.g., a
helper that calls `MustDefine` with their pre-validated options.

**Costs:** users can't extend the option vocabulary. We'd open a hook
point when justified, not preemptively.

## Decision: `Concurrent()` was dropped

There was briefly a `Concurrent()` option that flipped an executor's mode
to parallel.

**Why removed:** the `Option` contract is "configures `Def`."
`Concurrent` didn't — it had to reach the executor's private `mode` field,
which lives outside `Def`. The implementation worked through a covert
unexported `modeSetter` interface that `New` type-asserted on, but the
public contract was a lie: an option that does nothing in `apply` and
does its real work through a hidden side channel.

**Replaced by:** the `Parallel(name, opts...)` constructor. Mode is set
by which function you call. No covert interfaces.

**The general lesson:** when an "Option" needs side channels, it's not
really an option. Promote it to a first-class API surface.
