package peanut

import "fmt"

// Error attributes a task failure to the task that produced it. Composers
// (Sequential, Parallel) wrap a child's error in this so callers see the
// failing task's name without losing the underlying error via errors.Is /
// errors.As.
type Error struct {
	Name string
	Err  error
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Name, e.Err.Error()) }
func (e *Error) Unwrap() error { return e.Err }

// attribute wraps err with name unless err is nil or name is empty.
func attribute(name string, err error) error {
	if err == nil || name == "" {
		return err
	}
	return &Error{Name: name, Err: err}
}
