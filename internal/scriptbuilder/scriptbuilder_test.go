package scriptbuilder

import (
	"strings"
	"testing"
)

func TestShebang(t *testing.T) {
	if Shebang != "#!/usr/bin/env bash" {
		t.Fatalf("unexpected shebang: %q", Shebang)
	}
}

func TestDraftAddAndGenerate(t *testing.T) {
	d := NewDraft()
	d.Add("  echo one  ")
	d.Add("")
	d.Add("echo two")
	if d.Count() != 2 {
		t.Fatalf("count=%d", d.Count())
	}
	body, err := d.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(body, Shebang+"\n") {
		t.Fatalf("body=%q", body)
	}
	if !strings.Contains(body, "echo one\n") || !strings.Contains(body, "echo two\n") {
		t.Fatalf("body=%q", body)
	}
	snap := d.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("%v", snap)
	}
	d.Clear()
	if d.Count() != 0 {
		t.Fatal("clear failed")
	}
}
