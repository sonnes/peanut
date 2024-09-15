package peanut

import (
	"context"
	"reflect"
	"runtime"
	"strings"
)

// Series creates a meta stage that executes stages sequentially.
func Series(stages ...Handler) HandlerFunc {
	return HandlerFunc(func(ctx context.Context, req Request) error {
		for _, stage := range stages {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := RunWithContext(ctx, stage, req)
				if err != nil {
					return NewError(err, NameOf(stage))
				}
			}
		}

		return nil
	})
}

// NameOf returns the name of the stage using:
// - `Name()` method if the stage implements `Named` interface
// - function name if the stage is a function, using `runtime.FuncForPC`
func NameOf(stage Handler) string {
	if named, ok := stage.(Named); ok {
		return named.Name()
	}

	name := runtime.FuncForPC(reflect.ValueOf(stage).Pointer()).Name()

	// remove the `func[\d]+` part of the name
	// TODO: find a way to get the name of the method or struct
	// returning the Handler interface
	parts := strings.Split(name, ".")
	parts[len(parts)-1] = "<HandlerFunc>"

	return strings.Join(parts, ".")
}
