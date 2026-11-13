package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultPath(t *testing.T) {
	if DefaultPath != "~/.local/share/linux-cmd-wiki/wiki.db" {
		t.Fatalf("unexpected DefaultPath: %q", DefaultPath)
	}
}

func TestResolvePathDefault(t *testing.T) {
	got, err := ResolvePath(DefaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "wiki.db" {
		t.Fatalf("base: %q", filepath.Base(got))
	}
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wiki.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenCreatesSchema(t *testing.T) {
	s := openTemp(t)
	if _, err := os.Stat(s.Path()); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
	for _, table := range []string{"programs", "man_commands", "user_snippets", "custom_scripts", "programs_fts"} {
		ok, err := s.HasTable(table)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("missing table %s", table)
		}
	}
	cols, err := s.ColumnNames("programs")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"id", "name", "package_name", "summary", "version", "is_installed", "man_parsed"}
	if len(cols) != len(want) {
		t.Fatalf("programs cols=%v want=%v", cols, want)
	}
	for i := range want {
		if cols[i] != want[i] {
			t.Fatalf("col[%d]=%q want %q", i, cols[i], want[i])
		}
	}
}

func TestUpsertAndGetProgram(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{
		Name: "ffmpeg", PackageName: "ffmpeg", Summary: "video converter",
		Version: "6.1", IsInstalled: true,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("id=%d", id)
	}
	p, err := s.GetProgramByName("ffmpeg")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsInstalled || p.Summary != "video converter" || p.Version != "6.1" {
		t.Fatalf("unexpected program: %+v", p)
	}
	byID, err := s.GetProgramByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if byID.Name != "ffmpeg" {
		t.Fatalf("by id: %+v", byID)
	}
}

func TestManCommandsAndFTS(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{
		Name: "curl", Summary: "transfer data", IsInstalled: true,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = s.ReplaceManCommands(id, []ManCommand{
		{Flag: "-o", Description: "write output to file"},
		{Flag: "--silent", Description: "silent mode"},
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.GetProgramByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if !p.ManParsed {
		t.Fatal("expected man_parsed=1")
	}
	cmds, err := s.ListManCommands(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 2 {
		t.Fatalf("cmds=%d", len(cmds))
	}

	_, err = s.InsertUserSnippet(UserSnippet{
		ProgramID: id,
		Title:     "quiet download",
		Command:   "curl --silent https://example.com",
		Tags:      "http network",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search("silent")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].ProgramID != id {
		t.Fatalf("search silent: %+v", hits)
	}
	hits, err = s.Search("network")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected snippet tag hit")
	}
	hits, err = s.Search("curl")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected name hit")
	}
}

func TestMarkNotInstalledKeepsSnippets(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{Name: "gone", Summary: "x", Version: "1", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.InsertUserSnippet(UserSnippet{
		ProgramID: id, Title: "keep me", Command: "gone -h",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.MarkNotInstalledExcept(nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("affected=%d", n)
	}
	p, err := s.GetProgramByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if p.IsInstalled {
		t.Fatal("still installed")
	}
	count, err := s.CountUserSnippets(id)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("snippets=%d", count)
	}
}

func TestUpsertResetsManParsedOnVersionChange(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{Name: "jq", Version: "1.6", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceManCommands(id, []ManCommand{{Flag: "-r", Description: "raw"}}); err != nil {
		t.Fatal(err)
	}
	_, err = s.UpsertProgram(Program{Name: "jq", Version: "1.7", IsInstalled: true}, true)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.GetProgramByName("jq")
	if err != nil {
		t.Fatal(err)
	}
	if p.ManParsed {
		t.Fatal("expected man_parsed reset after version change")
	}
	if p.Version != "1.7" {
		t.Fatalf("version=%q", p.Version)
	}
}
