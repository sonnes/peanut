package peanut_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
)

func successHandler(name string, seq chan string) peanut.Handler {
	return peanut.HandlerFunc(func(ctx context.Context, req peanut.Request) error {
		<-time.After(10 * time.Millisecond)
		seq <- name

		return nil

	})
}

func errorHandler(name string, seq chan string, err error) peanut.Handler {
	return peanut.HandlerFunc(func(ctx context.Context, req peanut.Request) error {
		<-time.After(10 * time.Millisecond)
		seq <- name

		return err
	})
}

func TestSeries(t *testing.T) {
	execOrder := make(chan string, 3)

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := successHandler("h3", execOrder)

	series := peanut.Series(h1, h2, h3)

	err := series.Run(context.TODO(), nil)
	assert.NoError(t, err)

	want := []string{"h1", "h2", "h3"}
	got := []string{<-execOrder, <-execOrder, <-execOrder}

	assert.Equal(t, want, got)
}

func TestSeriesError(t *testing.T) {
	execOrder := make(chan string, 3)
	execErr := errors.New("execution error")

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := peanut.HandlerFunc(func(ctx context.Context, req peanut.Request) error {
		execOrder <- "h3"

		return execErr
	})

	series := peanut.Series(h1, h2, h3)

	err := series.Run(context.TODO(), nil)
	assert.Error(t, err)

	se, ok := err.(*peanut.Error)
	assert.True(t, ok)
	assert.ErrorIs(t, se, execErr)

	want := []string{"h1", "h2", "h3"}
	got := []string{<-execOrder, <-execOrder, <-execOrder}

	assert.Equal(t, want, got)
}

func TestSeriesCancel(t *testing.T) {
	execOrder := make(chan string, 3)

	h1 := successHandler("h1", execOrder)
	h2 := successHandler("h2", execOrder)
	h3 := successHandler("h3", execOrder)

	series := peanut.Series(h1, h2, h3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := series.Run(ctx, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func Handler1() peanut.HandlerFunc {
	return peanut.HandlerFunc(func(ctx context.Context, req peanut.Request) error {
		return nil
	})
}

type Handler2 struct{}

func (h *Handler2) Run(ctx context.Context, req peanut.Request) error {
	return nil
}

func (h *Handler2) Name() string {
	return "Handler2"
}

func TestNameOf(t *testing.T) {
	testcases := []struct {
		name  string
		stage peanut.Handler
		want  string
	}{
		{
			name:  "HandlerFunc",
			stage: Handler1(),
			want:  "github.com/sonnes/peanut_test.TestNameOf.Handler1.<HandlerFunc>",
		},
		{
			name:  "Inline HandlerFunc",
			stage: peanut.HandlerFunc(func(ctx context.Context, req peanut.Request) error { return nil }),
			want:  "github.com/sonnes/peanut_test.TestNameOf.<HandlerFunc>",
		},
		{
			name:  "Named Handler",
			stage: &Handler2{},
			want:  "Handler2",
		},
	}

	for _, tc := range testcases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			got := peanut.NameOf(tc.stage)
			assert.Equal(t, tc.want, got)
		})
	}
}
