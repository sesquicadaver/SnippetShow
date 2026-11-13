// Package backup exports/imports user_snippets and custom_scripts as JSON.
// Phase 0: skeleton only — versioned schema lands in Phase 4 (P4-8…P4-9).

package backup

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("backup: not implemented (phase 4)")

// Export writes snippets and scripts to path. Phase 0 stub.
func Export(path string) error {
	_ = path
	return ErrNotImplemented
}

// Import merges a JSON backup into the local DB. Phase 0 stub.
func Import(path string) error {
	_ = path
	return ErrNotImplemented
}
