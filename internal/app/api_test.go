package app

import (
	"path/filepath"
	"testing"
	"time"

	"snippetshow/internal/db"
)

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
	if info.Phase != "1" {
		t.Fatalf("phase: got %q", info.Phase)
	}
}

func TestPhaseStatus(t *testing.T) {
	if New().PhaseStatus() == "" {
		t.Fatal("empty phase status")
	}
}

func TestSearchAndGetProgramAPI(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	id, err := store.UpsertProgram(db.Program{
		Name: "ffmpeg", Summary: "video transcoder", IsInstalled: true,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceManCommands(id, []db.ManCommand{
		{Flag: "-i", Description: "input file"},
	}); err != nil {
		t.Fatal(err)
	}
	_, err = store.InsertUserSnippet(db.UserSnippet{
		ProgramID: id, Title: "encode", Command: "ffmpeg -i in.mp4 out.mkv",
		Tags: "video", CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := NewWithStore(store)
	hits, err := api.Search("video")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected FTS hits")
	}
	detail, err := api.GetProgram("ffmpeg", false)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Program.Name != "ffmpeg" || len(detail.Commands) != 1 {
		t.Fatalf("%+v", detail)
	}
	list, err := api.ListInstalled()
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}
