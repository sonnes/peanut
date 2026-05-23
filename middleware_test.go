package peanut_test

import (
	"context"
	"testing"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrap_ChainsAndPreservesDef(t *testing.T) {
	var order []string

	inner, err := peanut.DefineTask("inner", func(context.Context, *int) error {
		order = append(order, "inner")
		return nil
	})
	require.NoError(t, err)

	mw := func(label string) peanut.Middleware[*int] {
		return func(next peanut.Task[*int]) peanut.Task[*int] {
			return peanut.WithDef(next.Def(), func(ctx context.Context, s *int) error {
				order = append(order, label+":before")
				err := next.Run(ctx, s)
				order = append(order, label+":after")
				return err
			})
		}
	}

	task := peanut.Wrap(inner, mw("a"), mw("b"))
	require.NoError(t, task.Run(context.Background(), new(int)))

	assert.Equal(t, []string{"a:before", "b:before", "inner", "b:after", "a:after"}, order)
	assert.Equal(t, "inner", task.Def().Name, "outermost wrapper should carry the inner task's identity")
}
