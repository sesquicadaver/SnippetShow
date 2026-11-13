package scriptbuilder

import "testing"

func TestShebang(t *testing.T) {
	if Shebang != "#!/usr/bin/env bash" {
		t.Fatalf("unexpected shebang: %q", Shebang)
	}
}

func TestGenerateStub(t *testing.T) {
	_, err := NewDraft().Generate()
	if err != ErrNotImplemented {
		t.Fatalf("want ErrNotImplemented, got %v", err)
	}
}
