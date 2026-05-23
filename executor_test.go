package peanut_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_AcceptsTimeout(t *testing.T) {
	exec := peanut.New[*int]("pipe", peanut.Timeout(time.Second))
	require.Equal(t, time.Second, exec.Def().Timeout)
}

func TestNew_PanicsOnBadOption(t *testing.T) {
	require.Panics(t, func() {
		peanut.New[*int]("pipe", peanut.Timeout(-time.Second))
	})
}

func TestExecutor_SatisfiesTaskForNesting(t *testing.T) {
	inner := peanut.New[*int]("inner")
	outer := peanut.New[*int]("outer")

	// Compiles only if *Executor[*int] satisfies Task[*int].
	var _ peanut.Task[*int] = inner
	outer.Add(inner)

	require.Equal(t, "inner", inner.Def().Name)
}

func TestNew_NameAndDescription(t *testing.T) {
	exec := peanut.New[*int]("timeline",
		peanut.Description("loads and renders a timeline"),
	)
	def := exec.Def()
	require.Equal(t, "timeline", def.Name)
	require.Equal(t, "loads and renders a timeline", def.Description)
}

func TestExecutor_RunsTasksInOrder(t *testing.T) {
	exec := peanut.New[*[]string]("e")
	exec.Add(appendTask("a"), appendTask("b"), appendTask("c"))

	state := &[]string{}
	require.NoError(t, exec.Run(context.Background(), state))
	assert.Equal(t, []string{"a", "b", "c"}, *state)
}

func TestExecutor_RunStopsOnFirstError(t *testing.T) {
	boom := errors.New("boom")
	called := 0
	count := func(name string, fail bool) peanut.Task[*int] {
		task, _ := peanut.DefineTask(name, func(context.Context, *int) error {
			called++
			if fail {
				return boom
			}
			return nil
		})
		return task
	}

	exec := peanut.New[*int]("e")
	exec.Add(count("a", false), count("b", true), count("c", false))

	err := exec.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, 2, called, "should not run task after failing one")
}

func TestExecutor_NoTasksReturnsNil(t *testing.T) {
	exec := peanut.New[*int]("e")
	require.NoError(t, exec.Run(context.Background(), new(int)))
}

func TestExecutor_AddAndUseChainAndReturnReceiver(t *testing.T) {
	exec := peanut.New[*int]("e")
	require.Same(t, exec, exec.Add())
	require.Same(t, exec, exec.Use())
}

func TestExecutor_UseAppliesMiddlewareToEveryTask(t *testing.T) {
	var order []string
	exec := peanut.New[*int]("e")

	mw := peanut.Middleware[*int](func(next peanut.Task[*int]) peanut.Task[*int] {
		return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
			order = append(order, "mw:"+next.Def().Name)
			return next.Run(ctx, s)
		})
	})
	exec.Use(mw).Add(
		recordTask("t1", &order),
		recordTask("t2", &order),
	)

	require.NoError(t, exec.Run(context.Background(), new(int)))
	assert.Equal(t, []string{"mw:t1", "t1", "mw:t2", "t2"}, order)
}

func TestExecutor_RunRespectsExecutorTimeout(t *testing.T) {
	exec := peanut.New[*int]("e", peanut.Timeout(5*time.Millisecond))
	exec.Add(slowTask(100 * time.Millisecond))

	err := exec.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestExecutor_RunRespectsPerTaskTimeout(t *testing.T) {
	exec := peanut.New[*int]("e")
	slow, _ := peanut.DefineTask("slow", func(ctx context.Context, _ *int) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, peanut.Timeout(5*time.Millisecond))
	exec.Add(slow)

	err := exec.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestExecutor_MiddlewareAppliesToNestedSequential(t *testing.T) {
	var (
		mu      sync.Mutex
		visited []string
	)
	mw := peanut.Middleware[*int](func(next peanut.Task[*int]) peanut.Task[*int] {
		return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
			mu.Lock()
			visited = append(visited, "mw:"+next.Def().Name)
			mu.Unlock()
			return next.Run(ctx, s)
		})
	})

	mk := func(name string) peanut.Task[*int] {
		task, _ := peanut.DefineTask(name, func(context.Context, *int) error {
			mu.Lock()
			visited = append(visited, name)
			mu.Unlock()
			return nil
		})
		return task
	}

	exec := peanut.New[*int]("e")
	exec.Use(mw).Add(
		mk("a"),
		peanut.Sequential[*int]("sequential").Add(mk("b"), mk("c")),
	)
	require.NoError(t, exec.Run(context.Background(), new(int)))

	assert.Equal(t,
		[]string{"mw:a", "a", "mw:sequential", "mw:b", "b", "mw:c", "c"},
		visited,
		"middleware should wrap every task including children inside Sequential",
	)
}

func TestExecutor_MiddlewareAppliesToNestedParallel(t *testing.T) {
	var seen sync.Map
	mw := peanut.Middleware[*int](func(next peanut.Task[*int]) peanut.Task[*int] {
		return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
			seen.Store("mw:"+next.Def().Name, true)
			return next.Run(ctx, s)
		})
	})

	mk := func(name string) peanut.Task[*int] {
		task, _ := peanut.DefineTask(name, func(context.Context, *int) error { return nil })
		return task
	}

	exec := peanut.New[*int]("e")
	exec.Use(mw).Add(peanut.Parallel[*int]("parallel").Add(mk("a"), mk("b"), mk("c")))
	require.NoError(t, exec.Run(context.Background(), new(int)))

	for _, name := range []string{"mw:parallel", "mw:a", "mw:b", "mw:c"} {
		_, ok := seen.Load(name)
		assert.True(t, ok, "middleware missed %s", name)
	}
}

func TestExecutor_NestedExecutorAccumulatesMiddleware(t *testing.T) {
	var (
		mu   sync.Mutex
		seen []string
	)
	record := func(prefix string) peanut.Middleware[*int] {
		return func(next peanut.Task[*int]) peanut.Task[*int] {
			return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
				mu.Lock()
				seen = append(seen, prefix+":"+next.Def().Name)
				mu.Unlock()
				return next.Run(ctx, s)
			})
		}
	}

	child, _ := peanut.DefineTask("child", func(context.Context, *int) error { return nil })

	inner := peanut.New[*int]("inner")
	inner.Use(record("B")).Add(child)

	outer := peanut.New[*int]("outer")
	outer.Use(record("A")).Add(inner)

	require.NoError(t, outer.Run(context.Background(), new(int)))

	assert.Equal(t, []string{
		"A:inner", // outer mw wraps inner-as-task
		"A:child", // outer mw propagates to inner's children
		"B:child", // inner mw additionally wraps inner's children
	}, seen)
}


func TestExecutor_RunHonorsCanceledContext(t *testing.T) {
	exec := peanut.New[*int]("e")
	exec.Add(slowTask(100 * time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	err := exec.Run(ctx, new(int))
	assert.ErrorIs(t, err, context.Canceled)
}

func TestExecutor_RunIsRepeatable(t *testing.T) {
	out := &[]string{}
	exec := peanut.New[*[]string]("e")
	exec.Add(appendTask("first"))

	require.NoError(t, exec.Run(context.Background(), out))
	assert.Equal(t, []string{"first"}, *out)

	exec.Add(appendTask("second"))
	require.NoError(t, exec.Run(context.Background(), out))

	assert.Equal(t, []string{"first", "first", "second"}, *out,
		"second Run replays original tasks plus the task added between runs")
}

func TestExecutor_NestedAttributionUnwrapChain(t *testing.T) {
	boom := errors.New("boom")
	leaf, _ := peanut.DefineTask("leaf", func(context.Context, *int) error { return boom })

	inner := peanut.Sequential[*int]("inner").Add(leaf)
	outer := peanut.New[*int]("outer")
	outer.Add(inner)

	err := outer.Run(context.Background(), new(int))

	var top *peanut.Error
	require.ErrorAs(t, err, &top)
	assert.Equal(t, "inner", top.Name, "outer attributes to its child (inner)")

	nested, ok := top.Err.(*peanut.Error)
	require.True(t, ok, "should unwrap into a nested *Error from the inner composer")
	assert.Equal(t, "leaf", nested.Name)
	assert.ErrorIs(t, err, boom)
}

func TestRunWithMiddleware_WithoutAmbientChainCallsTaskDirectly(t *testing.T) {
	task, _ := peanut.DefineTask("solo", func(_ context.Context, s *int) error {
		*s = 7
		return nil
	})
	state := 0
	require.NoError(t, peanut.RunWithMiddleware(context.Background(), task, &state))
	assert.Equal(t, 7, state)
}

func TestRunWithMiddleware_AppliesAmbientChain(t *testing.T) {
	var seen []string
	mw := peanut.Middleware[*int](func(next peanut.Task[*int]) peanut.Task[*int] {
		return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
			seen = append(seen, "mw:"+next.Def().Name)
			return next.Run(ctx, s)
		})
	})

	// Build a custom composer that uses RunWithMiddleware so executor-registered
	// middleware reaches its children. Tests the public contract.
	customComposer := peanut.WithDef[*int](peanut.Def{Name: "custom"}, func(ctx context.Context, s *int) error {
		child, _ := peanut.DefineTask("inner", func(context.Context, *int) error { return nil })
		return peanut.RunWithMiddleware(ctx, child, s)
	})

	exec := peanut.New[*int]("e")
	exec.Use(mw).Add(customComposer)
	require.NoError(t, exec.Run(context.Background(), new(int)))

	assert.Equal(t, []string{"mw:custom", "mw:inner"}, seen,
		"middleware should wrap both the custom composer and its children via RunWithMiddleware")
}

// helpers

func appendTask(name string) peanut.Task[*[]string] {
	t, _ := peanut.DefineTask(name, func(_ context.Context, s *[]string) error {
		*s = append(*s, name)
		return nil
	})
	return t
}

func recordTask(name string, out *[]string) peanut.Task[*int] {
	t, _ := peanut.DefineTask(name, func(context.Context, *int) error {
		*out = append(*out, name)
		return nil
	})
	return t
}

func slowTask(d time.Duration) peanut.Task[*int] {
	t, _ := peanut.DefineTask("slow", func(ctx context.Context, _ *int) error {
		select {
		case <-time.After(d):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	return t
}
