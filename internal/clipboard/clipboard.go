// Package clipboard copies text via X11/Wayland CLI tools (Phase 3 / P3-5).
// Wails runtime.ClipboardSetText is preferred at the App layer; this package is the OS fallback.
package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrUnavailable means no clipboard backend worked on this session.
var ErrUnavailable = errors.New("clipboard: жоден бекенд недоступний (wl-copy / xclip / xsel)")

// Copy writes text to the system clipboard using Wayland then X11 fallbacks.
func Copy(text string) error {
	session := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	wayland := session == "wayland" || os.Getenv("WAYLAND_DISPLAY") != ""

	var order []struct {
		name string
		args []string
	}
	xclip := struct {
		name string
		args []string
	}{"xclip", []string{"-selection", "clipboard"}}
	xsel := struct {
		name string
		args []string
	}{"xsel", []string{"--clipboard", "--input"}}
	wl := struct {
		name string
		args []string
	}{"wl-copy", nil}

	if wayland {
		order = []struct {
			name string
			args []string
		}{wl, xclip, xsel}
	} else {
		order = []struct {
			name string
			args []string
		}{xclip, xsel, wl}
	}

	var errs []string
	for _, b := range order {
		if err := pipeTo(b.name, text, b.args...); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", b.name, err))
			continue
		}
		return nil
	}
	if len(errs) == 0 {
		return ErrUnavailable
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, strings.Join(errs, "; "))
}

// pipeTo runs cmd with args, writing text to stdin.
func pipeTo(bin, text string, args ...string) error {
	path, err := exec.LookPath(bin)
	if err != nil {
		return err
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = bytes.NewBufferString(text)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
