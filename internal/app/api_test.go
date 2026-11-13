package app

import "testing"

func TestInfoDecisions(t *testing.T) {
	info := New().Info()
	if info.Name != "SnippetShow" {
		t.Fatalf("name: got %q", info.Name)
	}
	if info.DBPath != "~/.local/share/linux-cmd-wiki/wiki.db" {
		t.Fatalf("db path: got %q", info.DBPath)
	}
	if info.UILocale != "uk" {
		t.Fatalf("locale: got %q", info.UILocale)
	}
	if info.Frontend != "vanilla-js" {
		t.Fatalf("frontend: got %q", info.Frontend)
	}
}

func TestPhaseStatus(t *testing.T) {
	if New().PhaseStatus() == "" {
		t.Fatal("empty phase status")
	}
}
