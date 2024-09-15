package peanut

import (
	"context"

	"github.com/sourcegraph/conc/pool"
)

// Parallel creates a meta stage that executes stages concurrently.
func Parallel(stages ...Handler) HandlerFunc {
	return HandlerFunc(func(ctx context.Context, req Request) error {
		p := pool.New().WithContext(ctx)

		for _, stage := range stages {
			stage := stage

			p.Go(func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					err := RunWithContext(ctx, stage, req)
					if err != nil {
						return NewError(err, NameOf(stage))
					}

					return nil
				}
			})
		}

		return p.Wait()
	})
}
