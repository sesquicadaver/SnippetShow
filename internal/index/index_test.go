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
	// D13.1: names-only — summary comes from later EnrichMetadata, not FirstRun.
	if p.Summary != "" {
		t.Fatalf("first-run summary must be empty, got %q", p.Summary)
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

func TestSetFavoriteParsesAndPurgesMan(t *testing.T) {
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
		Workers: 4,
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := store.GetProgramByName("tool")
	if err != nil || p.ManParsed || p.IsFavorite {
		t.Fatalf("first-run must be names-only: %+v err=%v", p, err)
	}
	fav, err := u.SetFavorite(context.Background(), "tool", true)
	if err != nil {
		t.Fatal(err)
	}
	if !fav.IsFavorite || !fav.ManParsed {
		t.Fatalf("favorite must parse man: %+v", fav)
	}
	cmds, err := store.ListManCommands(fav.ID)
	if err != nil || len(cmds) == 0 {
		t.Fatalf("man missing: %v err=%v", cmds, err)
	}
	un, err := u.SetFavorite(context.Background(), "tool", false)
	if err != nil {
		t.Fatal(err)
	}
	if un.IsFavorite || un.ManParsed {
		t.Fatalf("unfavorite must purge man: %+v", un)
	}
	cmds, err = store.ListManCommands(un.ID)
	if err != nil || len(cmds) != 0 {
		t.Fatalf("man_commands must be empty after purge: %v err=%v", cmds, err)
	}
}

func TestUpdateIndexDoesNotParseNewInstall(t *testing.T) {
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
	res, err := u.syncInstalled(context.Background(), false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added < 1 {
		t.Fatalf("expected new name: %+v", res)
	}
	if res.Parsed != 0 {
		t.Fatalf("D13: new install must not auto-parse: %+v", res)
	}
	p, err := store.GetProgramByName("beta")
	if err != nil || p.ManParsed || p.IsFavorite {
		t.Fatalf("beta must stay unparsed: %+v err=%v", p, err)
	}
}

func TestSyncRepoCatalogBatchAndCacheEmpty(t *testing.T) {
	store := openStore(t)
	_, err := store.UpsertProgram(db.Program{Name: "keepme", Summary: "local", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	baseFTS := store.FTSSyncCount.Load()

	const n = 1100
	var names []string
	for i := 0; i < n; i++ {
		names = append(names, "catpkg"+itoaPad(i, 4))
	}
	exec := &mockExec{
		bins: map[string]string{
			"dpkg-query": "/usr/bin/dpkg-query",
			"apt-cache":  "/usr/bin/apt-cache",
		},
		outs: map[string]string{
			"apt-cache pkgnames": strings.Join(names, "\n"),
			"apt-cache search .": names[0] + " - first\n" + names[n-1] + " - last\n",
		},
	}
	u := &Updater{
		Store: store,
		Scanner: &scanner.Scanner{
			Runner:        exec,
			PathEnv:       t.TempDir(),
			Home:          t.TempDir(),
			SkipCanonical: true,
		},
		Parser: &manparser.Parser{Runner: &mockMan{}},
	}
	prog := make(chan Progress, 32)
	var sawCatalog bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range prog {
			if p.Phase == "catalog" && p.Total == n {
				sawCatalog = true
			}
		}
	}()
	res, err := u.SyncRepoCatalog(context.Background(), prog)
	close(prog)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != scanner.CacheOK || res.Added != n || res.Total != n {
		t.Fatalf("%+v", res)
	}
	if !sawCatalog {
		t.Fatal("missing catalog progress")
	}
	ftsDelta := store.FTSSyncCount.Load() - baseFTS
	if ftsDelta != int64(n) {
		t.Fatalf("FTS syncs=%d want %d", ftsDelta, n)
	}
	hits, err := store.SearchPackagesFiltered(names[0], db.PackageFilter{})
	if err != nil || len(hits) == 0 {
		t.Fatalf("packages fts missing first: %v err=%v", hits, err)
	}
	keep, err := store.GetProgramByName("keepme")
	if err != nil || !keep.IsInstalled {
		t.Fatalf("installed demoted: %+v err=%v", keep, err)
	}

	// D5: empty apt cache soft-fail.
	u.Scanner.Runner = &mockExec{
		bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query", "apt-cache": "/usr/bin/apt-cache"},
		outs: map[string]string{},
	}
	empty, err := u.SyncRepoCatalog(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Status != scanner.CacheEmpty {
		t.Fatalf("cache_empty: %+v", empty)
	}
}

func itoaPad(i, width int) string {
	s := itoa(i)
	for len(s) < width {
		s = "0" + s
	}
	return s
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}

func TestParseManyPoolStillConcurrent(t *testing.T) {
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
		Parser:  &manparser.Parser{Runner: cm},
		Workers: 4,
	}
	if _, err := u.FirstRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	parsed, err := u.parseMany(context.Background(), []string{"alpha", "beta"}, func(Progress) {})
	if err != nil {
		t.Fatal(err)
	}
	if parsed < 2 {
		t.Fatalf("want ≥2 parsed via pool: %d", parsed)
	}
	cm.mu.Lock()
	maxFlight := cm.maxFlight
	cm.mu.Unlock()
	if maxFlight < 2 {
		t.Fatalf("ParseMany pool not concurrent: maxFlight=%d (want ≥2)", maxFlight)
	}
}

func TestUpdateIndexNamesOnlyDoesNotEnrich(t *testing.T) {
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
		ParseOnUpdate: true,
		Workers:       4,
	}
	res, err := u.UpdateIndex(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Added < 1 || res.Parsed != 0 {
		t.Fatalf("names-only UpdateIndex: %+v", res)
	}
	p, err := store.GetProgramByName("tool")
	if err != nil {
		t.Fatal(err)
	}
	if p.Category != "" || p.Summary != "" || p.ManParsed {
		t.Fatalf("SS-36b: UpdateIndex must not enrich/parse: %+v", p)
	}
}
