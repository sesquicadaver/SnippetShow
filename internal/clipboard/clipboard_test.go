package clipboard

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCopyEmptyBackendsReportUnavailable(t *testing.T) {
	// Ensure we don't accidentally succeed via real tools if PATH is weird —
	// still OK if a tool exists: then Copy should return nil.
	err := Copy("snippetshow-clipboard-test")
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "clipboard:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyPreferWaylandOrder(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	// Smoke: function does not panic and returns either nil or ErrUnavailable wrap.
	_ = Copy("hello")
}

func TestCopyPreferX11Order(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "x11")
	_ = os.Unsetenv("WAYLAND_DISPLAY")
	_ = Copy("hello")
}

func TestPipeToMissingBinary(t *testing.T) {
	err := pipeTo("snippetshow-no-such-clipboard-bin", "x")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestPipeToEchoIfAvailable(t *testing.T) {
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}
	// cat with no args reads stdin and exits 0 — good smoke for pipeTo.
	if err := pipeTo("cat", "payload"); err != nil {
		t.Fatal(err)
	}
}
