# Peanut - A Composition Framework

Peanut is a strongly opinionated composition framework in Go. Peanut is built on the foundation of a UNIX pipeline-like architecture, inspired by Reddit's []"From Service to Platform: A Ranking System in Go"](https://www.reddit.com/r/RedditEng/comments/z137m3/from_service_to_platform_a_ranking_system_in_go/). It is designed to help implement and maintain complex multi-step APIs, jobs, workflows, etc.

## Design

At its core, Peanut is a small library that implements a small set of building blocks called stages. These building blocks are simple & powerful.

## Separation of Concerns

Peanut has been designed to achieve a clear separation between WHAT a stage does, and WHEN & WHICH stages are required in a workflow. Peanut stages are designed to declare their read & write dependencies via interfaces. This separation enables the creation of simple and concise stages, each dedicated to performing just one specific task.

Hence, the name - Peanut.

Peanut builds on three fundamental mental models:

### Domain

Domain defines the shape of the the feature, business or job to be performed. Domain is essentially Go struct(s) defined by a stage or a shared library/package.

### State

State is instantiation of one or more domain objects necessary for the stage. Think of state as a request scoped database that enables one or more stages to read & write data.

### Stage

Stages perform the actual work. The work itself could be varied, as long as the stage has a consise definition of what it does.

## Peanut API

### Request

Request is a generalization of HTTP, gRPC, Kafka Message or any other message that triggers execution of Peanut workflow.

Request is an interface of common methods to read payload, headers, etc.

```go
type Request interface {
    // GetHeader returns the header with the given key.
    GetHeader(key string) string
    // GetHeaders returns all headers.
    GetHeaders() map[string]string
    // UnmarshalBody unmarshals the body into the given struct.
    UnmarshalBody(v interface{}) error
}
```

Peanut provides default implementations for HTTP, gRPC and Kafka.

### Stage

Stage is the fundamental unit of work in Peanut. A Stage is any implementation of the Handler interface.

```go
type Handler interface {
    Handle(ctx context.Context, req Request) error
}
type HandlerFunc func(ctx context.Context, req Request) error
```

### Meta Stage

Meta stages handle the execution of other stages. Some examples are:

```go
// Series creates a meta stage that executes stages sequentially.
func Series(stages ...peanut.Handler) peanut.Handler {
// ...
}
// Parallel creates a meta stage that executes stages concurrently.
func Parallel(stages ...peanut.Handler) peanut.Handler {
// ...
}
// If executes the given stage if the condition is true.
func If(condition func() bool, stage peanut.Handler) peanut.Handler {
// ...
}
// IfElse executes the given stage if the condition is true,
// otherwise it executes the else stage.
func IfElse(condition func() bool, stage, elseStage peanut.Handler) peanut.Handler {
// ...
}
// IgnoreError executes the given stage and ignores any error returned.
func IgnoreError(stage peanut.Handler) peanut.Handler {
// ...
}
// Retry executes the given stage and retries it if it
// returns a ErrRetryable error.
func Retry(policy RetryPolicy, stage peanut.Handler) peanut.Handler {
// ...
}
```

### Middleware

Middlewares are like meta-stages that are applied to every stage automatically. Middlewares are designed to mimic HTTP middlewares.

Middlewares are useful for things like logging, tracing, error handling, etc.

git filter-repo --email-callback '
return b"raviatluri@gmail.com"
'
