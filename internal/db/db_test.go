package db

import "testing"

func TestDefaultPath(t *testing.T) {
	if DefaultPath != "~/.local/share/linux-cmd-wiki/wiki.db" {
		t.Fatalf("unexpected DefaultPath: %q", DefaultPath)
	}
}

func TestOpenStub(t *testing.T) {
	_, err := Open(DefaultPath)
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
