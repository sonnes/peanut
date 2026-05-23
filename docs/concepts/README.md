---
title: Concepts
description: Index of concept documents covering peanut's abstractions, the decisions behind them, and what those decisions traded off.
package: peanut
---

# Concepts

Each file here covers one abstraction in the peanut library. They are
**decision-focused** rather than reference-focused — for signatures and full
GoDoc, see [pkg.go.dev/github.com/sonnes/peanut](https://pkg.go.dev/github.com/sonnes/peanut)
or run `go doc -all github.com/sonnes/peanut`. The point of these documents is
to capture **why** the API looks the way it does and what alternatives were
rejected, so future changes can revisit the same crossroads without re-deriving
the trade-offs.

| Concept | What it is | Key decision |
|---|---|---|
| [Task](task.md) | The unit of work | Identity is mandatory; no Step abstraction |
| [Executor](executor.md) | Runs tasks; is itself a Task | One type covers sequential, parallel, and nesting |
| [Option](option.md) | Construction-time configuration | Single Option type, shared between tasks and executors |
| [Middleware](middleware.md) | Wraps task invocations | Propagates through nesting via ctx-accumulated chain |
| [Error](error.md) | Failure attribution | `*Error` carries `Def.Name`, plays nicely with `errors.Is/As` |

## The single guiding principle

> One concept (`Task[S]`), one composition primitive (`Executor[S]` which is itself a `Task[S]`),
> and one observability layer (`Middleware[S]` that wraps every task at every level).

Every decision below was made in service of collapsing toward that line. When
a previous design had multiple concepts where one would do — `Named` + `Task`,
`Sequential` + `Parallel` + `Executor`, `TaskOption` + `ExecOption`, `Step` +
`Task` — we merged.

## Decisions snapshot

Quick tour of the biggest design calls, with detail in each concept doc:

1. **Identity is mandatory on `Task`.** No optional `Named` interface, no
   reflection-based name extraction. Every task has a `Def`. (see [task.md](task.md))
2. **`Executor[S]` satisfies `Task[S]`.** Nested executors are the only
   composition primitive. (see [executor.md](executor.md))
3. **`Sequential` and `Parallel` are constructors of `Executor`**, not separate
   types. Same code path, different mode. (see [executor.md](executor.md))
4. **All constructors panic on bad options.** Bad options are programmer bugs.
   (see [executor.md](executor.md) and [option.md](option.md))
5. **Middleware propagates through nesting via ctx accumulation.** Outer
   executor's middleware reaches every child at every level. (see [middleware.md](middleware.md))
6. **No `Step` abstraction.** If a sub-operation deserves observability, make
   it a task. (see [task.md](task.md))
7. **Errors are attributed via `*Error` carrying `Def.Name`** with `Unwrap`.
   (see [error.md](error.md))

## What's deliberately not in peanut

- **Retry, fallback, panic-recovery** — write these as middleware.
- **Step tracing** — promote sub-operations to tasks or use OpenTelemetry directly.
- **Concurrency limits on `Parallel`** — not yet; would be an option if/when needed.
- **Plugin system, registry, JSON dispatch** — peanut is Go-typed end-to-end.
