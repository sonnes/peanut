package peanut

import (
	"context"
	"fmt"
)

// Recover is a middleware that recovers from panics.
func Recover() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (err error) {
			defer func() {
				if r := recover(); r != nil {
					switch r := r.(type) {
					case error:
						err = r
					default:
						err = fmt.Errorf("%v", r)
					}

					return
				}
			}()

			return next.Run(ctx, req)
		})
	}
}
