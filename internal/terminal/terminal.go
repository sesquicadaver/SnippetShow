// Package terminal detects a user terminal (D7) and runs commands with a held shell (Phase 3 / P3-7…P3-8).
package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotFound means no terminal from the D7 fallback list is available.
var ErrNotFound = errors.New("terminal: не знайдено підтримуваний термінал (D7)")

// FallbackOrder is the D7 terminal discovery sequence.
var FallbackOrder = []string{
	"$TERMINAL",
	"x-terminal-emulator",
	"gnome-terminal",
	"konsole",
	"xfce4-terminal",
}

// Detected describes a resolved terminal binary and how to pass a command.
type Detected struct {
	Name string // logical name from FallbackOrder
	Path string // absolute or PATH-resolved binary
	Kind string // argv template kind
}

// Detect returns the preferred terminal binary using D7 order.
func Detect() (Detected, error) {
	// 1) $TERMINAL — may be a path or bare name.
	if raw := strings.TrimSpace(os.Getenv("TERMINAL")); raw != "" {
		path, err := look(raw)
		if err == nil {
			return Detected{Name: "$TERMINAL", Path: path, Kind: kindFor(filepath.Base(path))}, nil
		}
	}
	for _, name := range FallbackOrder[1:] {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		return Detected{Name: name, Path: path, Kind: kindFor(name)}, nil
	}
	return Detected{}, ErrNotFound
}

func look(raw string) (string, error) {
	if strings.Contains(raw, "/") {
		if st, err := os.Stat(raw); err == nil && !st.IsDir() {
			return raw, nil
		}
		return "", fmt.Errorf("not found: %s", raw)
	}
	return exec.LookPath(raw)
}

func kindFor(base string) string {
	base = strings.ToLower(base)
	switch {
	case strings.Contains(base, "gnome-terminal"):
		return "gnome"
	case strings.Contains(base, "konsole"):
		return "konsole"
	case strings.Contains(base, "xfce4-terminal"):
		return "xfce"
	default:
		return "e" // x-terminal-emulator and most others: -e
	}
}

// BuildArgv constructs argv for launching heldShell(command) in the detected terminal.
func BuildArgv(d Detected, command string) []string {
	shell := heldShell(command)
	switch d.Kind {
	case "gnome":
		return []string{d.Path, "--", "bash", "-c", shell}
	case "konsole":
		return []string{d.Path, "-e", "bash", "-c", shell}
	case "xfce":
		return []string{d.Path, "-e", "bash -c " + shellQuote(shell)}
	default:
		return []string{d.Path, "-e", "bash", "-c", shell}
	}
}

// heldShell wraps the user command so the terminal stays open (ТЗ §4.3).
func heldShell(command string) string {
	return command + "; exec bash"
}

func shellQuote(s string) string {
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}

// Run opens the detected terminal and executes command, keeping the window open.
func Run(command string) error {
	d, err := Detect()
	if err != nil {
		return err
	}
	argv := BuildArgv(d, command)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("terminal: запуск %s: %w", d.Name, err)
	}
	// Detach: do not wait for the terminal process.
	_ = cmd.Process.Release()
	return nil
}
