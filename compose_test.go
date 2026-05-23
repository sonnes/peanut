package peanut_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSequential_RunsInOrder(t *testing.T) {
	out := &[]string{}
	seq := peanut.Sequential[*[]string]("seq").Add(
		appendTask("a"),
		appendTask("b"),
		appendTask("c"),
	)
	require.NoError(t, seq.Run(context.Background(), out))
	assert.Equal(t, []string{"a", "b", "c"}, *out)
}

func TestSequential_StopsOnError(t *testing.T) {
	boom := errors.New("boom")
	var executed []string
	mk := func(name string, fail bool) peanut.Task[*int] {
		task, _ := peanut.DefineTask(name, func(context.Context, *int) error {
			executed = append(executed, name)
			if fail {
				return boom
			}
			return nil
		})
		return task
	}
	seq := peanut.Sequential[*int]("seq").Add(mk("a", false), mk("b", true), mk("c", false))
	err := seq.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"a", "b"}, executed)
}

func TestSequential_HonorsChildTimeout(t *testing.T) {
	slow, _ := peanut.DefineTask("slow", func(ctx context.Context, _ *int) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, peanut.Timeout(5*time.Millisecond))

	seq := peanut.Sequential[*int]("seq").Add(slow)
	err := seq.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestParallel_RunsAllTasks(t *testing.T) {
	var counter atomic.Int32
	mk := func(name string) peanut.Task[*int] {
		task, _ := peanut.DefineTask(name, func(context.Context, *int) error {
			counter.Add(1)
			return nil
		})
		return task
	}
	par := peanut.Parallel[*int]("par").Add(mk("p1"), mk("p2"), mk("p3"))
	require.NoError(t, par.Run(context.Background(), new(int)))
	assert.Equal(t, int32(3), counter.Load())
}

func TestParallel_RunsConcurrently(t *testing.T) {
	const n = 3
	started := make(chan struct{}, n)
	release := make(chan struct{})
	mk := func() peanut.Task[*int] {
		task, _ := peanut.DefineTask("p", func(context.Context, *int) error {
			started <- struct{}{}
			<-release
			return nil
		})
		return task
	}

	par := peanut.Parallel[*int]("par").Add(mk(), mk(), mk())
	done := make(chan error, 1)
	go func() { done <- par.Run(context.Background(), new(int)) }()

	for i := 0; i < n; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("only %d of %d tasks started — Parallel is not concurrent", i, n)
		}
	}
	close(release)
	require.NoError(t, <-done)
}

func TestParallel_FirstErrorReturnedAndSiblingsCanceled(t *testing.T) {
	boom := errors.New("boom")
	fail, _ := peanut.DefineTask("fail", func(context.Context, *int) error {
		return boom
	})

	var siblingErr atomic.Value
	var siblingDone sync.WaitGroup
	siblingDone.Add(1)
	sibling, _ := peanut.DefineTask("sibling", func(ctx context.Context, _ *int) error {
		defer siblingDone.Done()
		select {
		case <-time.After(time.Second):
			return nil
		case <-ctx.Done():
			siblingErr.Store(ctx.Err())
			return ctx.Err()
		}
	})

	par := peanut.Parallel[*int]("par").Add(fail, sibling)
	err := par.Run(context.Background(), new(int))
	assert.ErrorIs(t, err, boom)

	siblingDone.Wait()
	assert.ErrorIs(t, siblingErr.Load().(error), context.Canceled)
}

func TestParallel_NoTasksReturnsNil(t *testing.T) {
	par := peanut.Parallel[*int]("par")
	require.NoError(t, par.Run(context.Background(), new(int)))
}

func TestParallel_AcceptsOptions(t *testing.T) {
	par := peanut.Parallel[*int]("fetch",
		peanut.Description("fetches a thing"),
		peanut.Timeout(time.Second),
	)
	def := par.Def()
	assert.Equal(t, "fetch", def.Name)
	assert.Equal(t, "fetches a thing", def.Description)
	assert.Equal(t, time.Second, def.Timeout)
}

func TestSequential_AcceptsOptions(t *testing.T) {
	seq := peanut.Sequential[*int]("pipeline",
		peanut.Description("a pipeline"),
		peanut.Timeout(2*time.Second),
	)
	def := seq.Def()
	assert.Equal(t, "pipeline", def.Name)
	assert.Equal(t, "a pipeline", def.Description)
	assert.Equal(t, 2*time.Second, def.Timeout)
}

func TestSequential_WrapsErrorWithTaskName(t *testing.T) {
	boom := errors.New("boom")
	failing, _ := peanut.DefineTask("failing", func(context.Context, *int) error {
		return boom
	})

	err := peanut.Sequential[*int]("seq").Add(failing).Run(context.Background(), new(int))

	var perr *peanut.Error
	require.ErrorAs(t, err, &perr)
	assert.Equal(t, "failing", perr.Name)
	assert.ErrorIs(t, err, boom)
}

func TestParallel_WrapsErrorWithTaskName(t *testing.T) {
	boom := errors.New("boom")
	failing, _ := peanut.DefineTask("failing", func(context.Context, *int) error {
		return boom
	})
	ok, _ := peanut.DefineTask("ok", func(context.Context, *int) error { return nil })

	err := peanut.Parallel[*int]("par").Add(failing, ok).Run(context.Background(), new(int))

	var perr *peanut.Error
	require.ErrorAs(t, err, &perr)
	assert.Equal(t, "failing", perr.Name)
	assert.ErrorIs(t, err, boom)
}

func TestParallel_PanicsOnBadOption(t *testing.T) {
	assert.Panics(t, func() {
		peanut.Parallel[*int]("bad", peanut.Timeout(-1))
	})
}
