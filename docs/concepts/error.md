---
title: Error
description: Why failures wrap in *Error, why attribution stacks at every level, why the empty-name guard exists.
package: peanut
---

# Error

For signatures and usage, see godoc. This doc is for the design calls.

## Decision: structured `*Error` rather than string concatenation

`Error{Name, Err}` is an exported struct with `Error()` and `Unwrap()`,
not a `fmt.Errorf("%s: %w", ...)` string.

**Considered:**

1. *Plain string wrap* with `%w`. Preserves `errors.Is`, but the name is
   only recoverable by parsing the formatted message — fragile.
2. *No wrapping at all*. Executor returns the raw error; only composers
   (`Sequential`, `Parallel`) attribute. v1 peanut's compromise.
3. *Rich error type* with stack traces, action keys, retry counts.
   Heavyweight; useful in big frameworks (genkit), overkill here.

**Why structured:** `errors.As(err, &perr)` is the right way to recover the
failing task's name programmatically. `Unwrap` preserves `errors.Is`
against the underlying error. Two exported fields, methods only for the
standard error interfaces — simplest thing that wins both. Caller doesn't
have to choose between "human-readable string" and "structured access."

**Costs:** users have to know `*Error` exists to extract the name. The
`Error()` format makes the chain visible by default, so naive code that
just prints the error still gets useful output.

## Decision: attribution at every executor level

Both `runSequential` and `runParallel` wrap child errors with the child's
`Def.Name`. Nested executors stack the wraps.

**Considered:** attribute only at the deepest level (just the leaf name);
attribute only at the outermost level (just the entry-point task); skip
attribution at the executor level entirely (only composers wrap, v1's
asymmetry).

**Why uniform:** the full path is genuinely useful when reading logs for
complex pipelines. "outer-pipeline: fetch: call-api: connection refused"
tells a story that "connection refused" doesn't. The cost is one struct
allocation per executor frame on the error path — negligible compared to
the value at debug time.

**Costs:** for very deep nesting, the formatted `Error()` string can get
long. The underlying chain is intact; if a caller wants only the leaf,
they walk `Unwrap`.

## Decision: `attribute` helper is internal

The `attribute(name, err)` function that constructs `*Error` is
unexported. Users return raw errors from task bodies; the executor wraps
them.

**Why internal:** `*Error` is the executor-to-caller contract. Pre-wrapping
inside a task body would be redundant (executor wraps anyway) and risk
double-wrapping. The struct fields are exported in case a user really wants
to construct one manually (rare).

**Costs:** if a user wants to attribute a synthetic name inside their task
body, they write `&peanut.Error{Name: ..., Err: ...}` directly. We don't
provide a constructor function. This stays rare and that's fine.

## Decision: empty-name guard in `attribute`

```go
if err == nil || name == "" { return err }
```

No normal path produces an empty name — `DefineTask` requires a name,
composers set `"sequential"`/`"parallel"`/the user-given name.

**Why the guard exists anyway:** defense in depth. A user constructing a
task via `WithDef(Def{}, fn)` with an empty name shouldn't end up with
errors that format as `": connection refused"`. Cheap to guard, expensive
to debug if you trip it without one.

**Costs:** technically unreachable code in the happy path. Trivial.
