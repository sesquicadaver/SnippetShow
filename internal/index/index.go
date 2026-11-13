// Package index orchestrates incremental program index updates.
// Phase 0: skeleton only — first-run / UpdateIndex land in Phase 1 (P1-13…P1-16).

package index

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("index: not implemented (phase 1)")

// Updater will coordinate scanner + manparser + db after Phase 1.
type Updater struct{}

// UpdateIndex refreshes the local program index. Phase 0 stub.
func (u *Updater) UpdateIndex() error {
	return ErrNotImplemented
}
