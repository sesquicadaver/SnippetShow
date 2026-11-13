// Package manparser extracts summaries and OPTIONS/FLAGS from man pages.
// Phase 0: skeleton only — whatis/raw/regex/worker pool land in Phase 1 (P1-9…P1-12).
// Policy D6: man parse is for installed programs in MVP.

package manparser

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("manparser: not implemented (phase 1)")

// Option is a parsed flag/command pair from a man page.
type Option struct {
	Flag        string
	Description string
}

// Summary returns a short description via whatis/man -f. Phase 0 stub.
func Summary(cmd string) (string, error) {
	_ = cmd
	return "", ErrNotImplemented
}

// ParseOptions extracts OPTIONS/COMMANDS/FLAGS. Phase 0 stub.
func ParseOptions(cmd string) ([]Option, error) {
	_ = cmd
	return nil, ErrNotImplemented
}
