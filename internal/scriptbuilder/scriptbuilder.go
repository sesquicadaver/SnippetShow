// Package scriptbuilder holds an in-memory script draft (Phase 3 hook P3-11; full UI in Phase 4).
package scriptbuilder

import (
	"errors"
	"strings"
	"sync"
)

// ErrNotImplemented marks unfinished Phase 4 surfaces (Generate polish / export).
var ErrNotImplemented = errors.New("scriptbuilder: not implemented (phase 4)")

// Shebang is the default script header.
const Shebang = "#!/usr/bin/env bash"

// Draft is an in-memory ordered list of commands.
type Draft struct {
	mu    sync.Mutex
	Lines []string
}

// NewDraft returns an empty draft.
func NewDraft() *Draft {
	return &Draft{Lines: nil}
}

// Add appends a non-empty command line (P3-11). Preserved across program switches via shared Draft.
func (d *Draft) Add(line string) {
	line = strings.TrimSpace(line)
	if line == "" || d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Lines = append(d.Lines, line)
}

// Count returns the number of draft lines.
func (d *Draft) Count() int {
	if d == nil {
		return 0
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.Lines)
}

// Snapshot returns a copy of current lines.
func (d *Draft) Snapshot() []string {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.Lines))
	copy(out, d.Lines)
	return out
}

// Clear removes all draft lines.
func (d *Draft) Clear() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Lines = nil
}

// Generate produces a bash script body with shebang (usable for Phase 3 draft preview).
func (d *Draft) Generate() (string, error) {
	if d == nil {
		return Shebang + "\n", nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	var b strings.Builder
	b.WriteString(Shebang)
	b.WriteByte('\n')
	for _, line := range d.Lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
