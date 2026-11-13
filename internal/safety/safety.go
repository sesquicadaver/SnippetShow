// Package safety detects dangerous shell constructs before Run.
// Phase 0: skeleton only — pattern matrix lands in Phase 3 (P3-9…P3-10).

package safety

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("safety: not implemented (phase 3)")

// Result describes whether a command needs user confirmation.
type Result struct {
	Dangerous bool
	Reasons   []string
}

// Analyze inspects a command for rm -rf / dd / mkfs / sudo patterns. Phase 0 stub.
func Analyze(command string) (Result, error) {
	_ = command
	return Result{}, ErrNotImplemented
}
