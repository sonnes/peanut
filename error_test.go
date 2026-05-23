package peanut_test

import (
	"errors"
	"testing"

	"github.com/sonnes/peanut"
	"github.com/stretchr/testify/assert"
)

func TestError_FormatAndUnwrap(t *testing.T) {
	root := errors.New("connection refused")
	err := &peanut.Error{Name: "fetch-tweets", Err: root}

	assert.Equal(t, "fetch-tweets: connection refused", err.Error())
	assert.Same(t, root, err.Unwrap())
	assert.ErrorIs(t, err, root)
}

func TestError_NestedUnwrapChain(t *testing.T) {
	root := errors.New("disk full")
	inner := &peanut.Error{Name: "save", Err: root}
	outer := &peanut.Error{Name: "checkpoint", Err: inner}

	assert.Equal(t, "checkpoint: save: disk full", outer.Error())
	assert.ErrorIs(t, outer, root)

	var leaf *peanut.Error
	assert.True(t, errors.As(outer.Unwrap(), &leaf))
	assert.Equal(t, "save", leaf.Name)
}
