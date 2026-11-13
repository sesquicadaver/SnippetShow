package terminal

import "testing"

func TestFallbackOrderD7(t *testing.T) {
	want := []string{
		"$TERMINAL",
		"x-terminal-emulator",
		"gnome-terminal",
		"konsole",
		"xfce4-terminal",
	}
	if len(FallbackOrder) != len(want) {
		t.Fatalf("len: got %d want %d", len(FallbackOrder), len(want))
	}
	for i := range want {
		if FallbackOrder[i] != want[i] {
			t.Fatalf("order[%d]: got %q want %q", i, FallbackOrder[i], want[i])
		}
	}
}

func TestDetectStub(t *testing.T) {
	_, err := Detect()
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
