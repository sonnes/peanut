# peanut

A small task runner for Go where every unit of work has a name, every
composition is just another unit of work, and one line of middleware reaches
everything.

```bash
go get github.com/sonnes/peanut
```

## What it feels like

```go
fetch := peanut.Parallel[*Timeline]("fetch").Add(
    fetchProfile, fetchTweets, fetchFollowers, fetchCounts,
)

pipeline := peanut.New[*Timeline]("timeline").Use(timing)
pipeline.Add(fetch, renderTimeline)

err := pipeline.Run(ctx, &Timeline{Handle: "@gopher"})
```

That's the whole shape. One generic state type `*Timeline` threads through
everything. `fetch` is a parallel sub-executor. `pipeline` is a sequential
parent. Both are `Task`s. The `timing` middleware on `pipeline` wraps every
task at every level — `fetchProfile`, all four parallel children, the `fetch`
composer itself, and `renderTimeline`. One `Use`, everything covered.

When something fails inside `fetchTweets`, the error you get back is
`*peanut.Error{Name: "fetch", Err: *peanut.Error{Name: "fetch-tweets", Err: ...}}` —
attribution path intact, `errors.Is` against the original still works.

## What you can build

**A pipeline with parallel fan-out.** The example above. Tasks that hit
independent APIs in parallel, gathered into a single state struct, then
rendered downstream.

**Nested pipelines with their own middleware.** A retry-equipped sub-pipeline
inside a logged top-level pipeline. Inner middleware layers on top of outer —
both fire for inner-pipeline tasks.

**Cross-cutting behavior without per-task changes.** Logging, tracing,
timing, retries, panic recovery — write each one once as a `Middleware[S]`
function and attach via `Use`. It reaches every task and every level.

**Custom composers.** Need fan-out with a concurrency limit, or a
retry-with-backoff loop? Write a `Task[S]` whose body calls
`peanut.RunWithMiddleware(ctx, child, state)` on each invocation. Your
composer participates in middleware propagation like the built-in ones.

**Type-safe state, no runtime casts.** Pick whatever struct shape you want
for `S` — the executor enforces that every task in the pipeline operates on
the same type. Mismatches are compile errors, not panics.

## Dev experience choices

The library makes a few opinionated calls so things behave predictably:

- **Every task has a name.** No anonymous tasks, no reflection-based name
  recovery, no "what was that thing in the log" moments. `DefineTask` and
  `MustDefine` require a name as the first argument.
- **Bad options panic at construction.** `peanut.Timeout(-1)` is a typo, not
  a runtime condition you handle. Constructors panic; you fix the source.
- **Failures attribute themselves.** Errors from inside an executor come back
  wrapped with the failing task's name. `errors.As(err, &perr)` extracts it;
  `errors.Is(err, target)` still walks the chain.
- **`Executor` is a `Task`.** There's no second composition concept to learn.
  Sub-pipelines are just nested executors.
- **Middleware propagates.** Register once on the outermost executor and it
  applies everywhere below. Inner executors can add more on top.

## Reading further

- **godoc**: [pkg.go.dev/github.com/sonnes/peanut](https://pkg.go.dev/github.com/sonnes/peanut)
  — every function, type, and method, signatures and contracts.
- **[docs/concepts/](docs/concepts/)** — the design calls behind the API.
  Each abstraction has a doc covering what was considered, what was chosen,
  and what each choice trades off.
- **`example_test.go`** in the repo — a runnable end-to-end pipeline
  (Twitter-style timeline) covering parallel fan-out, middleware, and error
  rendering.

## Status

API has stabilized through several iterations. No formal v1.0 tag yet — pin
to a commit if that matters to you.
