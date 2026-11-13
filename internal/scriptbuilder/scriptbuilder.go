// Package scriptbuilder holds an in-memory script draft and shebang helpers.
// Phase 0: skeleton only — UI/export land in Phase 4 (P4-1…P4-7).

package scriptbuilder

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("scriptbuilder: not implemented (phase 4)")

// Shebang is the default script header.
const Shebang = "#!/usr/bin/env bash"

// Draft is an in-memory ordered list of commands.
type Draft struct {
	Lines []string
}

// NewDraft returns an empty draft.
func NewDraft() *Draft {
	return &Draft{Lines: nil}
}

// Generate produces a bash script body with shebang. Phase 0 stub.
func (d *Draft) Generate() (string, error) {
	return "", ErrNotImplemented
}
