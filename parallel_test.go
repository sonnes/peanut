package peanut_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
)

func TestParallel(t *testing.T) {
	execOrder := make(chan string, 3)

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := successHandler("h3", execOrder)

	parallel := peanut.Parallel(h1, h2, h3)

	err := parallel.Run(context.TODO(), nil)
	assert.NoError(t, err)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		want := []string{"h1", "h2", "h3"}
		got := []string{<-execOrder, <-execOrder, <-execOrder}

		assert.ElementsMatch(t, want, got)

		wg.Done()
	}()

	wg.Wait()
}

func TestParallel_Error(t *testing.T) {
	execOrder := make(chan string, 3)
	execErr := errors.New("execution error")

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := errorHandler("h3", execOrder, execErr)

	parallel := peanut.Parallel(h1, h2, h3)

	err := parallel.Run(context.TODO(), nil)
	assert.Error(t, err)

	se := &peanut.Error{}
	ok := errors.As(err, &se)
	assert.True(t, ok)

	assert.ErrorIs(t, err, execErr)
}

func TestParallel_Cancel(t *testing.T) {
	execOrder := make(chan string, 3)

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := successHandler("h3", execOrder)

	parallel := peanut.Parallel(h1, h2, h3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := parallel.Run(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
