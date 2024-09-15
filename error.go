package peanut

import "fmt"

// Error is a wrapper for errors that are returned by stages.
// It contains the stage name and the error.
type Error struct {
	name string
	err  error
}

// NewError creates a new Error.
func NewError(err error, name string) *Error {
	return &Error{
		name: name,
		err:  err,
	}
}

// Name returns the stage name where the error occurred.
func (e *Error) Name() string {
	return e.name
}

// Error returns the error message.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.name, e.err.Error())
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	return e.err
}
