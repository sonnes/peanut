package peanut

import (
	"fmt"
	"time"
)

// Option configures a task (via DefineTask) or an executor (via New) at
// construction time. The same options apply to both; tasks and executors
// share the same Def.
type Option interface {
	apply(*Def) error
}

// Name overrides the Def's Name. Primarily useful with DefineFunc when the
// reflection-derived default isn't what you want; with other constructors it
// overrides the name passed as the first argument (last write wins).
func Name(s string) Option {
	return nameOpt(s)
}

type nameOpt string

func (n nameOpt) apply(def *Def) error {
	def.Name = string(n)
	return nil
}

// Description attaches a human-readable description.
func Description(s string) Option {
	return descriptionOpt(s)
}

type descriptionOpt string

func (d descriptionOpt) apply(def *Def) error {
	def.Description = string(d)
	return nil
}

// Timeout bounds execution. On a task it bounds that task's Run; on an
// Executor it bounds the whole Run.
func Timeout(d time.Duration) Option {
	return timeoutOpt(d)
}

type timeoutOpt time.Duration

func (t timeoutOpt) apply(def *Def) error {
	if time.Duration(t) < 0 {
		return fmt.Errorf("peanut: timeout must be non-negative, got %v", time.Duration(t))
	}
	def.Timeout = time.Duration(t)
	return nil
}
