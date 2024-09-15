package peanut

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func panicWith(msg any) HandlerFunc {
	return func(ctx context.Context, req Request) error {
		panic(msg)
	}
}

func TestRecover(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		mw := Recover()
		require.NotNil(t, mw)

		h := mw.Apply(panicWith("test"))
		require.NotNil(t, h)

		err := h.Run(context.Background(), nil)
		assert.Error(t, err)
		assert.Equal(t, "test", err.Error())
	})

	t.Run("Error", func(t *testing.T) {
		mw := Recover()
		require.NotNil(t, mw)

		h := mw.Apply(panicWith(assert.AnError))
		require.NotNil(t, h)

		err := h.Run(context.Background(), nil)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}
