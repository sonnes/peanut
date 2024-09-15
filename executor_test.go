package peanut

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor(t *testing.T) {
	e := New()
	require.NotNil(t, e)

	stage := func(name string) HandlerFunc {
		return func(ctx context.Context, req Request) error {
			return nil
		}
	}

	middleware := func(name string) MiddlewareFunc {
		return func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, req Request) error {
				return next.Run(ctx, req)
			})
		}
	}

	t.Run("Add Stages", func(t *testing.T) {
		e.Add(
			stage("stage 1"),
			stage("stage 2"),
		)

		assert.Len(t, e.stages, 2)
	})

	t.Run("Add Middlewares", func(t *testing.T) {
		e.Use(
			middleware("middleware 1"),
			middleware("middleware 2"),
		)

		assert.Len(t, e.middleware, 2)
	})

	t.Run("Run", func(t *testing.T) {
		err := e.Run(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("Run after cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := e.Run(ctx, nil)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Run with error", func(t *testing.T) {
		e.Add(HandlerFunc(func(ctx context.Context, req Request) error {
			return assert.AnError
		}))

		err := e.Run(context.Background(), nil)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}
