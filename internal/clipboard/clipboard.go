// Package clipboard copies text via Wails runtime with X11/Wayland fallbacks.
// Phase 0: skeleton only — implementation lands in Phase 3 (P3-5…P3-6).

package clipboard

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("clipboard: not implemented (phase 3)")

// Copy writes text to the system clipboard. Phase 0 stub.
func Copy(text string) error {
	_ = text
	return ErrNotImplemented
}
