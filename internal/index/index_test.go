package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"snippetshow/internal/db"
	"snippetshow/internal/manparser"
	"snippetshow/internal/scanner"
)

type mockExec struct {
	bins map[string]string
}

func (m *mockExec) LookPath(file string) (string, error) {
	if p, ok := m.bins[file]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func (m *mockExec) Run(context.Context, string, ...string) (string, error) {
	return "", os.ErrNotExist
}

type mockMan struct {
	summary map[string]string
	raw     map[string]string
}

func (m *mockMan) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := args[len(args)-1]
	if name == "whatis" || (name == "man" && len(args) > 0 && args[0] == "-f") {
		if s, ok := m.summary[cmd]; ok {
			return cmd + " (1) - " + s + "\n", nil
		}
	}
	if s, ok := m.raw[cmd]; ok {
		return s, nil
	}
	return "", os.ErrNotExist
}

func (m *mockMan) RunPipeline(_ context.Context, _ []string, _ string, leftArgs []string, _ string, _ []string) (string, error) {
	cmd := leftArgs[len(leftArgs)-1]
	if s, ok := m.raw[cmd]; ok {
		return s, nil
	}
	return "", os.ErrNotExist
}

const sampleMan = `NAME
       tool - sample

OPTIONS
       -h, --help
              show help
`

func openStore(t *testing.T) *db.Store {
	t.Helper()
	s, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func writeExec(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFirstRunFillsProgramsWithoutMan(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "tool")

	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        &mockExec{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}},
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{
			summary: map[string]string{"tool": "sample tool"},
			raw:     map[string]string{"tool": sampleMan},
		}},
	}
	res, err := u.FirstRun(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.InstalledSeen != 1 {
		t.Fatalf("%+v", res)
	}
	p, err := store.GetProgramByName("tool")
	if err != nil {
		t.Fatal(err)
	}
	if p.ManParsed {
		t.Fatal("man should not be parsed on first run")
	}
	if !strings.Contains(p.Summary, "sample") {
		t.Fatalf("summary=%q", p.Summary)
	}
}

func TestParseProgramOnDemand(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "tool")
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        &mockExec{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}},
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{
			summary: map[string]string{"tool": "sample tool"},
			raw:     map[string]string{"tool": sampleMan},
		}},
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.ParseProgram(context.Background(), "tool"); err != nil {
		t.Fatal(err)
	}
	p, err := store.GetProgramByName("tool")
	if err != nil {
		t.Fatal(err)
	}
	if !p.ManParsed {
		t.Fatal("expected man_parsed")
	}
	cmds, err := store.ListManCommands(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) == 0 {
		t.Fatal("no man commands")
	}
}

func TestUpdateIndexMarksGoneKeepsSnippets(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "tool")
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        &mockExec{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}},
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{
			summary: map[string]string{"tool": "sample tool"},
			raw:     map[string]string{"tool": sampleMan},
		}},
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, _ := store.GetProgramByName("tool")
	_, err := store.InsertUserSnippet(db.UserSnippet{
		ProgramID: p.ID,
		Title:     "keep",
		Command:   "tool -h",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Binary disappears.
	if err := os.Remove(filepath.Join(binDir, "tool")); err != nil {
		t.Fatal(err)
	}
	res, err := u.UpdateIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.MarkedGone != 1 {
		t.Fatalf("%+v", res)
	}
	p2, err := store.GetProgramByName("tool")
	if err != nil {
		t.Fatal(err)
	}
	if p2.IsInstalled {
		t.Fatal("still installed")
	}
	n, err := store.CountUserSnippets(p2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("snippets=%d", n)
	}
}
