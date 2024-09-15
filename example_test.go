package peanut_test

import (
	"context"
	"fmt"

	"github.com/sonnes/peanut"
)

func Stage(name string) peanut.HandlerFunc {
	return func(ctx context.Context, req peanut.Request) error {
		fmt.Printf("Running stage=%s\n", name)

		return nil
	}
}

func ExampleNew() {
	p := peanut.New()

	// add stages
	p.Add(
		Stage("1"),
		Stage("2"),
		peanut.Parallel(
			Stage("3"),
			Stage("4"),
			peanut.Series(
				Stage("5"),
				Stage("6"),
			),
		),
	)

	// always register middlewares before running the
	// executor
	p.Use(peanut.Recover())

	// run the stages
	err := p.Run(context.Background(), nil)
	if err != nil {
		panic(err)
	}
}
