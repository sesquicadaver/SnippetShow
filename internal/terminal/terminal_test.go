package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestKindFor(t *testing.T) {
	cases := map[string]string{
		"gnome-terminal":  "gnome",
		"konsole":         "konsole",
		"xfce4-terminal":  "xfce",
		"x-terminal-emulator": "e",
		"alacritty":       "e",
	}
	for in, want := range cases {
		if got := kindFor(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestBuildArgvTemplates(t *testing.T) {
	cmd := "echo hi"
	gnome := BuildArgv(Detected{Path: "/usr/bin/gnome-terminal", Kind: "gnome"}, cmd)
	if !contains(gnome, "--") || !contains(gnome, "bash") {
		t.Fatalf("gnome argv: %v", gnome)
	}
	if !strings.Contains(strings.Join(gnome, " "), "echo hi; exec bash") {
		t.Fatalf("held shell missing: %v", gnome)
	}
	konsole := BuildArgv(Detected{Path: "/usr/bin/konsole", Kind: "konsole"}, cmd)
	if konsole[1] != "-e" {
		t.Fatalf("konsole: %v", konsole)
	}
	xfce := BuildArgv(Detected{Path: "/usr/bin/xfce4-terminal", Kind: "xfce"}, cmd)
	if xfce[1] != "-e" || !strings.Contains(xfce[2], "bash -c") {
		t.Fatalf("xfce: %v", xfce)
	}
	generic := BuildArgv(Detected{Path: "/usr/bin/x-terminal-emulator", Kind: "e"}, cmd)
	if generic[1] != "-e" {
		t.Fatalf("generic: %v", generic)
	}
}

func TestDetectUsesTERMINALEnv(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "myterm")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TERMINAL", bin)
	d, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "$TERMINAL" || d.Path != bin {
		t.Fatalf("%+v", d)
	}
}

func TestDetectMissing(t *testing.T) {
	t.Setenv("TERMINAL", "")
	t.Setenv("PATH", t.TempDir()) // empty PATH → no terminals
	_, err := Detect()
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func contains(argv []string, s string) bool {
	for _, a := range argv {
		if a == s {
			return true
		}
	}
	return false
}
