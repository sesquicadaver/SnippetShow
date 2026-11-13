// Package scriptbuilder holds an in-memory script draft and shebang helpers (Phase 4 / P4-1…P4-7).
package scriptbuilder

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Shebang is the default script header (always first line of Generate).
const Shebang = "#!/usr/bin/env bash"

// Draft is an in-memory ordered list of commands shared across program switches.
type Draft struct {
	mu    sync.Mutex
	Lines []string
}

// NewDraft returns an empty draft.
func NewDraft() *Draft {
	return &Draft{Lines: nil}
}

// Add appends a non-empty command line (P3-11 / P4-1).
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

// SetLines replaces the draft with a copy of lines (empty entries skipped).
func (d *Draft) SetLines(lines []string) {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Lines = nil
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == Shebang || strings.HasPrefix(line, "#!") {
			continue
		}
		d.Lines = append(d.Lines, line)
	}
}

// MoveUp swaps line i with i-1. No-op if index is invalid.
func (d *Draft) MoveUp(i int) error {
	if d == nil {
		return fmt.Errorf("scriptbuilder: nil draft")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if i <= 0 || i >= len(d.Lines) {
		return fmt.Errorf("scriptbuilder: cannot move up index %d", i)
	}
	d.Lines[i-1], d.Lines[i] = d.Lines[i], d.Lines[i-1]
	return nil
}

// MoveDown swaps line i with i+1. No-op if index is invalid.
func (d *Draft) MoveDown(i int) error {
	if d == nil {
		return fmt.Errorf("scriptbuilder: nil draft")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if i < 0 || i >= len(d.Lines)-1 {
		return fmt.Errorf("scriptbuilder: cannot move down index %d", i)
	}
	d.Lines[i], d.Lines[i+1] = d.Lines[i+1], d.Lines[i]
	return nil
}

// Remove deletes the line at index i.
func (d *Draft) Remove(i int) error {
	if d == nil {
		return fmt.Errorf("scriptbuilder: nil draft")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if i < 0 || i >= len(d.Lines) {
		return fmt.Errorf("scriptbuilder: index %d out of range", i)
	}
	d.Lines = append(d.Lines[:i], d.Lines[i+1:]...)
	return nil
}

// ApplyBody parses an editor body into draft lines (shebang / blank lines skipped).
func (d *Draft) ApplyBody(body string) {
	if d == nil {
		return
	}
	raw := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	d.SetLines(raw)
}

// Generate produces a bash script body with shebang always as the first line (P4-3).
func (d *Draft) Generate() string {
	if d == nil {
		return Shebang + "\n"
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
	return b.String()
}

// EnsureShebang returns body with Shebang forced as the first line.
func EnsureShebang(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines)+1)
	out = append(out, Shebang)
	for i, line := range lines {
		if i == 0 && (line == Shebang || strings.HasPrefix(strings.TrimSpace(line), "#!")) {
			continue
		}
		out = append(out, line)
	}
	// Drop a single trailing empty split artifact only when body was empty.
	if len(out) == 1 {
		return Shebang + "\n"
	}
	return strings.Join(out, "\n")
}

// WriteExecutable writes body to path and sets mode 0755 (chmod +x) (P4-6).
func WriteExecutable(path, body string) error {
	body = EnsureShebang(body)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return fmt.Errorf("scriptbuilder: write %s: %w", path, err)
	}
	// Ensure +x even if umask stripped bits.
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("scriptbuilder: chmod +x %s: %w", path, err)
	}
	return nil
}
