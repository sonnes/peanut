package peanut_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefineTask_DefCarriesName(t *testing.T) {
	task, err := peanut.DefineTask("fetch", noopIntBody)
	require.NoError(t, err)
	assert.Equal(t, peanut.Def{Name: "fetch"}, task.Def())
}

func TestDefineTask_DescriptionOption(t *testing.T) {
	task, err := peanut.DefineTask("fetch", noopIntBody,
		peanut.Description("loads things"),
	)
	require.NoError(t, err)
	assert.Equal(t, peanut.Def{Name: "fetch", Description: "loads things"}, task.Def())
}

func TestDefineTask_RunsBody(t *testing.T) {
	task, err := peanut.DefineTask("set", func(_ context.Context, s *int) error {
		*s = 42
		return nil
	})
	require.NoError(t, err)

	state := 0
	require.NoError(t, task.Run(context.Background(), &state))
	assert.Equal(t, 42, state)
}

func TestDefineTask_PropagatesBodyError(t *testing.T) {
	boom := errors.New("boom")
	task, err := peanut.DefineTask("err", func(context.Context, *int) error { return boom })
	require.NoError(t, err)

	assert.ErrorIs(t, task.Run(context.Background(), new(int)), boom)
}

func TestDefineTask_NegativeTimeoutFails(t *testing.T) {
	_, err := peanut.DefineTask("bad", noopIntBody, peanut.Timeout(-time.Second))
	require.Error(t, err)
}

func TestDefineTask_TimeoutOnDef(t *testing.T) {
	task, err := peanut.DefineTask("fetch", noopIntBody, peanut.Timeout(2*time.Second))
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, task.Def().Timeout)
}

func TestWithDef_PreservesIdentity(t *testing.T) {
	def := peanut.Def{Name: "wrapped", Description: "by middleware"}
	task := peanut.WithDef[*int](def, noopIntBody)
	assert.Equal(t, def, task.Def())
}

func TestMustDefine_ReturnsTask(t *testing.T) {
	task := peanut.MustDefine[*int]("ok", noopIntBody, peanut.Description("desc"))
	assert.Equal(t, "ok", task.Def().Name)
	assert.Equal(t, "desc", task.Def().Description)
}

func TestMustDefine_PanicsOnBadOption(t *testing.T) {
	assert.Panics(t, func() {
		peanut.MustDefine[*int]("bad", noopIntBody, peanut.Timeout(-time.Second))
	})
}

func TestDefineFunc_DerivesNameFromFunc(t *testing.T) {
	task := peanut.DefineFunc[*int](reflectedTestBody)
	assert.Contains(t, task.Def().Name, "reflectedTestBody",
		"reflected name should include the source function name")
}

func TestDefineFunc_NameOptionOverrides(t *testing.T) {
	task := peanut.DefineFunc[*int](reflectedTestBody, peanut.Name("custom"))
	assert.Equal(t, "custom", task.Def().Name)
}

func TestDefineFunc_SupportsDescriptionAndTimeout(t *testing.T) {
	task := peanut.DefineFunc[*int](reflectedTestBody,
		peanut.Description("desc"),
		peanut.Timeout(500*time.Millisecond),
	)
	assert.Equal(t, "desc", task.Def().Description)
	assert.Equal(t, 500*time.Millisecond, task.Def().Timeout)
}

func TestDefineFunc_PanicsOnBadOption(t *testing.T) {
	assert.Panics(t, func() {
		peanut.DefineFunc[*int](reflectedTestBody, peanut.Timeout(-time.Second))
	})
}

func TestName_OverridesInDefineTask(t *testing.T) {
	task, err := peanut.DefineTask("first", noopIntBody, peanut.Name("override"))
	require.NoError(t, err)
	assert.Equal(t, "override", task.Def().Name)
}

func TestDefineFunc_RunsBody(t *testing.T) {
	state := 0
	task := peanut.DefineFunc[*int](func(_ context.Context, s *int) error {
		*s = 99
		return nil
	})
	require.NoError(t, task.Run(context.Background(), &state))
	assert.Equal(t, 99, state)
}

// reflectedTestBody is package-level so runtime.FuncForPC has a stable name
// to recover. Used by the DefineFunc reflection tests.
func reflectedTestBody(context.Context, *int) error { return nil }

func noopIntBody(context.Context, *int) error { return nil }
