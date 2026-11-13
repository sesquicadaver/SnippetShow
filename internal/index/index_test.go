package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"snippetshow/internal/db"
	"snippetshow/internal/manparser"
	"snippetshow/internal/scanner"
)

type mockExec struct {
	bins map[string]string
	outs map[string]string
}

func (m *mockExec) LookPath(file string) (string, error) {
	if p, ok := m.bins[file]; ok {
		return p, nil
	}
	return "", os.ErrNotExist
}

func (m *mockExec) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if m.outs != nil {
		if out, ok := m.outs[key]; ok {
			return out, nil
		}
	}
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

// concurrentMan tracks peak in-flight man lookups to prove the worker pool is used (M1).
type concurrentMan struct {
	mockMan
	mu       sync.Mutex
	inFlight int
	maxFlight int
}

func (m *concurrentMan) Run(ctx context.Context, name string, args ...string) (string, error) {
	m.mu.Lock()
	m.inFlight++
	if m.inFlight > m.maxFlight {
		m.maxFlight = m.inFlight
	}
	m.mu.Unlock()
	time.Sleep(25 * time.Millisecond)
	out, err := m.mockMan.Run(ctx, name, args...)
	m.mu.Lock()
	m.inFlight--
	m.mu.Unlock()
	return out, err
}

func (m *concurrentMan) RunPipeline(ctx context.Context, pathEnv []string, left string, leftArgs []string, right string, rightArgs []string) (string, error) {
	m.mu.Lock()
	m.inFlight++
	if m.inFlight > m.maxFlight {
		m.maxFlight = m.inFlight
	}
	m.mu.Unlock()
	time.Sleep(25 * time.Millisecond)
	out, err := m.mockMan.RunPipeline(ctx, pathEnv, left, leftArgs, right, rightArgs)
	m.mu.Lock()
	m.inFlight--
	m.mu.Unlock()
	return out, err
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

func dpkgOuts(binPath, pkg, version, summary string) map[string]string {
	return map[string]string{
		"dpkg-query -S " + binPath: pkg + ":amd64: " + binPath,
		"dpkg-query -W -f=${Package}\t${Version}\t${binary:Package}\t${Description}\n " + pkg: pkg + "\t" + version + "\t" + pkg + "\t" + summary + "\n",
	}
}

func TestNewEnablesParseOnUpdate(t *testing.T) {
	u := New(openStore(t))
	if !u.ParseOnUpdate {
		t.Fatal("index.New must set ParseOnUpdate=true (H1)")
	}
	if u.Workers < 4 {
		t.Fatalf("workers=%d", u.Workers)
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
	if res.Parsed != 0 {
		t.Fatalf("FirstRun must not mass-parse: %+v", res)
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

func TestUpdateIndexReparsesOnVersionChange(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "tool")
	binPath := filepath.Join(binDir, "tool")
	exec := &mockExec{
		bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		outs: dpkgOuts(binPath, "tool", "1.0", "sample tool"),
	}
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        exec,
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{
			summary: map[string]string{"tool": "sample tool"},
			raw:     map[string]string{"tool": sampleMan},
		}},
		ParseOnUpdate: true,
		Workers:       4,
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.ParseProgram(context.Background(), "tool"); err != nil {
		t.Fatal(err)
	}
	p, err := store.GetProgramByName("tool")
	if err != nil || !p.ManParsed || p.Version != "1.0" {
		t.Fatalf("precondition: %+v err=%v", p, err)
	}

	// Bump package version → UpdateIndex must reparse, not only reset man_parsed.
	exec.outs = dpkgOuts(binPath, "tool", "2.0", "sample tool v2")
	prog := make(chan Progress, 64)
	var phases []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range prog {
			phases = append(phases, p.Phase)
		}
	}()
	res, err := u.UpdateIndexWithProgress(context.Background(), prog)
	close(prog)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated < 1 || res.Parsed < 1 {
		t.Fatalf("want version update + reparse: %+v", res)
	}
	p2, err := store.GetProgramByName("tool")
	if err != nil {
		t.Fatal(err)
	}
	if p2.Version != "2.0" || !p2.ManParsed {
		t.Fatalf("after update: %+v", p2)
	}
	cmds, err := store.ListManCommands(p2.ID)
	if err != nil || len(cmds) == 0 {
		t.Fatalf("man commands missing after reparse: %v err=%v", cmds, err)
	}
	sawParse := false
	for _, ph := range phases {
		if ph == "parse" {
			sawParse = true
			break
		}
	}
	if !sawParse {
		t.Fatalf("missing parse progress phase: %v", phases)
	}
}

func TestUpdateIndexParsesNewInstall(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "alpha")
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        &mockExec{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}},
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{
			summary: map[string]string{"alpha": "a", "beta": "b"},
			raw:     map[string]string{"alpha": sampleMan, "beta": sampleMan},
		}},
		ParseOnUpdate: true,
		Workers:       4,
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	writeExec(t, binDir, "beta")
	res, err := u.UpdateIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Added < 1 || res.Parsed < 1 {
		t.Fatalf("new install must parse: %+v", res)
	}
	p, err := store.GetProgramByName("beta")
	if err != nil || !p.ManParsed {
		t.Fatalf("beta not parsed: %+v err=%v", p, err)
	}
}

func TestUpdateIndexUsesParseManyPool(t *testing.T) {
	store := openStore(t)
	binDir := t.TempDir()
	writeExec(t, binDir, "alpha")
	writeExec(t, binDir, "beta")
	cm := &concurrentMan{mockMan: mockMan{
		summary: map[string]string{"alpha": "a", "beta": "b"},
		raw:     map[string]string{"alpha": sampleMan, "beta": sampleMan},
	}}
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        &mockExec{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}},
			PathEnv:       binDir,
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser:        &manparser.Parser{Runner: cm},
		ParseOnUpdate: true,
		Workers:       4,
	}
	res, err := u.UpdateIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Parsed < 2 {
		t.Fatalf("want ≥2 parsed via pool: %+v", res)
	}
	cm.mu.Lock()
	maxFlight := cm.maxFlight
	cm.mu.Unlock()
	if maxFlight < 2 {
		t.Fatalf("ParseMany pool not concurrent: maxFlight=%d (want ≥2)", maxFlight)
	}
}
