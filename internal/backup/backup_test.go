package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"snippetshow/internal/db"
)

func TestExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	pid, err := store.UpsertProgram(db.Program{Name: "ffmpeg", Summary: "video", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.InsertUserSnippet(db.UserSnippet{
		ProgramID: pid, Title: "encode", Command: "ffmpeg -i a b", Tags: "video",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCustomScript("demo", "#!/usr/bin/env bash\necho hi\n"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "backup.json")
	if err := Export(store, path); err != nil {
		t.Fatal(err)
	}

	// Second store simulates another machine; re-import twice for idempotency.
	store2, err := db.Open(filepath.Join(dir, "wiki2.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	res, err := Import(store2, path)
	if err != nil {
		t.Fatal(err)
	}
	if res.SnippetsAdded != 1 || res.ScriptsAdded != 1 || res.ProgramsCreated != 1 {
		t.Fatalf("first import: %+v", res)
	}
	res2, err := Import(store2, path)
	if err != nil {
		t.Fatal(err)
	}
	if res2.SnippetsAdded != 0 || res2.ScriptsAdded != 0 || res2.SnippetsSkipped != 1 || res2.ScriptsSkipped != 1 {
		t.Fatalf("idempotent import: %+v", res2)
	}
	sns, err := store2.ListAllUserSnippets()
	if err != nil || len(sns) != 1 || sns[0].ProgramName != "ffmpeg" {
		t.Fatalf("%+v err=%v", sns, err)
	}
	scripts, err := store2.ListCustomScripts()
	if err != nil || len(scripts) != 1 {
		t.Fatalf("%+v err=%v", scripts, err)
	}
	// programs/man must not be wiped — stub program exists, no man required.
	p, err := store2.GetProgramByName("ffmpeg")
	if err != nil || p.IsInstalled {
		t.Fatalf("stub program: %+v err=%v", p, err)
	}
}

func TestImportRejectsBadVersion(t *testing.T) {
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"snippets":[],"scripts":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(store, path); err == nil {
		t.Fatal("expected version error")
	}
}
