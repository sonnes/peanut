---
title: Task
description: Why Task is the only unit of work, why identity is mandatory, why there's no Step.
package: peanut
---

# Task

For signatures and usage, see godoc. This doc is for the design calls that
aren't visible from the code.

## Decision: identity is mandatory

`Task[S]` requires both `Run` and `Def`. There is no escape hatch for an
anonymous task.

**Considered:** an optional `Named` interface that middleware type-asserted
on; a reflection fallback via `runtime.FuncForPC` to recover function names.

**Why mandatory wins:** every middleware that wants to log, trace, or
attribute an error needs a name. Making identity optional means every
middleware writes the same `if n, ok := next.(Named); ok` dance, and unnamed
tasks become silent black holes in observability. Reflection works for plain
functions but breaks for closures, methods, and anonymous funcs — and bakes
runtime introspection into a library that otherwise doesn't need it.

**Costs:** can't pass a bare `func(ctx, s) error` as a Task. You go through
`DefineTask`, `MustDefine`, or `WithDef`. We accept the extra step in
exchange for never having to deal with nameless tasks.

## Decision: `Def` merges identity and behavioral config

One struct for `Name`, `Description`, `Timeout` — not "TaskDef" (identity)
plus "taskConfig" (runtime) as separate types.

**Considered:** splitting them so observability and runtime had distinct
shapes. Earlier in development we had exactly that: an outer `taskConfig`
holding a `TaskDef` plus a `timeout` field.

**Why one struct:** every behavioral field is also something observability
wants to know. A tracer reading "this task has a 5s budget" is useful, not
leaky. Two structs for one field's distinction was the smell of premature
separation.

**Costs:** if we ever add purely-internal runtime fields (something
observability shouldn't see, like an internal cache key), they'd have to
live elsewhere — we'd be back to two structs. We don't have such a field
today.

## Decision: `TaskFunc` is a type alias, not a named type

A type alias (`= func(...)`) carries no methods. There's no way to make a
`TaskFunc` value satisfy `Task` on its own.

**Considered:** a named type with a `Run` method and an empty `Def`. That
would let `var t Task[S] = TaskFunc[S](myFn)` compile.

**Why alias:** an "anonymous Task with empty Def" undermines the
mandatory-identity decision above. Forcing construction through
`DefineTask`/`MustDefine`/`WithDef` guarantees every Task that exists has a
name.

**Costs:** marginally more verbose at call sites. Outweighed by consistency
with the identity stance.

## Decision: a third constructor `DefineFunc` exists for reflection-based naming

`DefineFunc[S](fn, opts...)` derives `Def.Name` from `fn` via
`runtime.FuncForPC`. `peanut.Name("override")` overrides it.

**Considered:**

1. *Reflection as the default for everything.* The original push: drop the
   mandatory-name requirement entirely, make all constructors fall back to
   reflection unless a `Name` option is passed.
2. *No reflection at all.* Keep `DefineTask` and `MustDefine` as the only
   ways to construct a task. Forces explicit naming everywhere.

**Why a third constructor and not default:** reflection works well for named
top-level functions (`pkg.FetchTweets`) and methods
(`pkg.(*Service).FetchTweets-fm`). For closures the runtime gives names
like `pkg.parent.func1` — more informative than I feared (it includes the
enclosing function) but still compiler-internal, unstable across refactors,
and likely to surprise someone reading logs months later. Making reflection
the default would silently inject those names into production traces.

Pushing reflection to an opt-in constructor (`DefineFunc`) preserves the
mandatory-name guarantee for `DefineTask`/`MustDefine` users while giving
the "I don't care, infer it" path to users who prefer ergonomics over
stability. The same `peanut.Name(...)` option works as the override
mechanism inside `DefineFunc` and as a last-write-wins override in the
explicit constructors.

**Costs:** three constructors instead of two. The library now imports
`reflect` and `runtime` (stdlib only, no external deps). Each `DefineFunc`
call walks a stack frame at construction time — negligible, but it's not
free.

## Decision: both `DefineTask` (error) and `MustDefine` (panic)

Two constructors with identical inputs.

**Considered:** only error-returning (forces boilerplate even when options
are constants); only panic (no path for runtime-driven options).

**Why both:** matches `regexp.Compile` / `regexp.MustCompile` and
`template.New` / `template.Must`. Go programmers recognize the idiom and
reach for the right one — `MustDefine` for package-level wiring with
constant options, `DefineTask` for the rare runtime-driven case.

**Costs:** two ways to do one thing. The stdlib precedent makes this
non-controversial.

## Decision: no `Step` abstraction

Earlier versions had `peanut.Step(ctx, name, in, fn)` for tracing
sub-operations inside a task body. It was removed.

**Why removed:** Step was a parallel concept to Task — both had names, both
ran code, both could be wrapped by middleware. But Step worked on typed
`In`/`Out` while Middleware works on state-typed `Task[S]`, so bridging
them required a context-stashed type-erased "step runner" closure. The
bridge worked but added 80 lines and a second mental model for what was
structurally one thing: "a named piece of work."

**The lesson:** if something deserves a name and middleware, make it a
task. If it doesn't, just call the function inline. There's no middle tier.

**Costs:** users lose the one-liner for ad-hoc sub-op tracing. Workarounds:
extract as a sub-task with its own Def, or use OpenTelemetry spans directly
inside the task body. We considered this loss worth the simpler model.
