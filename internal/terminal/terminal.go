// Package terminal detects a user terminal and runs commands with a held shell.
// Phase 0: skeleton only — detection order is fixed by D7; implementation in Phase 3 (P3-7…P3-8).

package terminal

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("terminal: not implemented (phase 3)")

// FallbackOrder is the D7 terminal discovery sequence.
var FallbackOrder = []string{
	"$TERMINAL",
	"x-terminal-emulator",
	"gnome-terminal",
	"konsole",
	"xfce4-terminal",
}

// Detect returns the preferred terminal binary. Phase 0 stub.
func Detect() (string, error) {
	return "", ErrNotImplemented
}

// Run opens the detected terminal and executes command. Phase 0 stub.
func Run(command string) error {
	_ = command
	return ErrNotImplemented
}
