package scriptbuilder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShebang(t *testing.T) {
	if Shebang != "#!/usr/bin/env bash" {
		t.Fatalf("unexpected shebang: %q", Shebang)
	}
}

func TestDraftAddGenerateReorder(t *testing.T) {
	d := NewDraft()
	d.Add("  echo one  ")
	d.Add("")
	d.Add("echo two")
	d.Add("echo three")
	if d.Count() != 3 {
		t.Fatalf("count=%d", d.Count())
	}
	if err := d.MoveUp(1); err != nil {
		t.Fatal(err)
	}
	snap := d.Snapshot()
	if snap[0] != "echo two" || snap[1] != "echo one" {
		t.Fatalf("%v", snap)
	}
	if err := d.MoveDown(0); err != nil {
		t.Fatal(err)
	}
	snap = d.Snapshot()
	if snap[0] != "echo one" || snap[1] != "echo two" {
		t.Fatalf("%v", snap)
	}
	if err := d.Remove(1); err != nil {
		t.Fatal(err)
	}
	if d.Count() != 2 {
		t.Fatalf("after remove: %d", d.Count())
	}
	body := d.Generate()
	if !strings.HasPrefix(body, Shebang+"\n") {
		t.Fatalf("body=%q", body)
	}
	if !strings.Contains(body, "echo one\n") || !strings.Contains(body, "echo three\n") {
		t.Fatalf("body=%q", body)
	}
	d.Clear()
	if d.Count() != 0 {
		t.Fatal("clear failed")
	}
}

func TestApplyBodyAndEnsureShebang(t *testing.T) {
	d := NewDraft()
	d.ApplyBody(Shebang + "\necho a\n\necho b\n")
	if d.Count() != 2 {
		t.Fatalf("%v", d.Snapshot())
	}
	got := EnsureShebang("echo x\n")
	if !strings.HasPrefix(got, Shebang+"\n") {
		t.Fatalf("%q", got)
	}
	got2 := EnsureShebang("#!/bin/sh\necho y\n")
	if !strings.HasPrefix(got2, Shebang+"\n") || strings.Contains(got2, "#!/bin/sh") {
		t.Fatalf("%q", got2)
	}
}

func TestWriteExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.sh")
	if err := WriteExecutable(path, "echo hi\n"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("not executable: %v", info.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), Shebang+"\n") {
		t.Fatalf("%q", data)
	}
}
