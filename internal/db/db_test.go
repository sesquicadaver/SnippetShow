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
	for _, table := range []string{"programs", "packages", "man_commands", "user_snippets", "custom_scripts", "programs_fts", "packages_fts"} {
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
	if p.Summary != "video converter" || !p.IsInstalled {
		t.Fatalf("%+v", p)
	}
	pkg, err := s.GetPackageByName("ffmpeg")
	if err != nil || !pkg.IsInstalled {
		t.Fatalf("SS-29 mirror package: %+v err=%v", pkg, err)
	}
}

func TestReplaceManCommandsAndSnippets(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{Name: "curl", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceManCommands(id, []ManCommand{
		{Flag: "-f", Description: "fail"},
		{Flag: "-s", Description: "silent"},
	}); err != nil {
		t.Fatal(err)
	}
	cmds, err := s.ListManCommands(id)
	if err != nil || len(cmds) != 2 {
		t.Fatalf("%v %v", cmds, err)
	}
	_, err = s.InsertUserSnippet(UserSnippet{
		ProgramID: id, Title: "get", Command: "curl -fsSL https://x", Tags: "net",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	sns, err := s.ListUserSnippets(id)
	if err != nil || len(sns) != 1 {
		t.Fatalf("%v %v", sns, err)
	}
}

func TestListProgramsFiltered(t *testing.T) {
	s := openTemp(t)
	for _, name := range []string{"alpha", "beta", "1pass"} {
		if _, err := s.UpsertProgram(Program{Name: name, IsInstalled: true}, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.UpsertProgram(Program{Name: "gone", IsInstalled: false}, false); err != nil {
		t.Fatal(err)
	}
	all, err := s.ListProgramsFiltered(ProgramFilter{})
	if err != nil {
		t.Fatal(err)
	}
	// Uninstalled stub stays in programs only if UpsertProgram without install — still a program row.
	if len(all) < 4 {
		t.Fatalf("all=%d", len(all))
	}
	inst, err := s.ListProgramsFiltered(ProgramFilter{InstalledOnly: true})
	if err != nil || len(inst) != 3 {
		t.Fatalf("installed=%d err=%v", len(inst), err)
	}
	letterA, err := s.ListProgramsFiltered(ProgramFilter{InstalledOnly: true, Letter: "a"})
	if err != nil || len(letterA) != 1 || letterA[0].Name != "alpha" {
		t.Fatalf("%+v err=%v", letterA, err)
	}
	digits, err := s.ListProgramsFiltered(ProgramFilter{InstalledOnly: true, Letter: "0-9"})
	if err != nil || len(digits) != 1 || digits[0].Name != "1pass" {
		t.Fatalf("%+v err=%v", digits, err)
	}
}

func TestSearchFTS(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{Name: "ripgrep", Summary: "fast grep", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceManCommands(id, []ManCommand{{Flag: "--json", Description: "json out"}}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search("json")
	if err != nil || len(hits) == 0 {
		t.Fatalf("%+v err=%v", hits, err)
	}
}

func TestCustomScriptsCRUD(t *testing.T) {
	s := openTemp(t)
	cs, err := s.CreateCustomScript("n", "echo 1")
	if err != nil || cs == nil || cs.ID <= 0 {
		t.Fatalf("%+v err=%v", cs, err)
	}
	all, err := s.ListCustomScripts()
	if err != nil || len(all) != 1 {
		t.Fatalf("%v err=%v", all, err)
	}
	if err := s.DeleteCustomScript(cs.ID); err != nil {
		t.Fatal(err)
	}
	all, err = s.ListCustomScripts()
	if err != nil || len(all) != 0 {
		t.Fatalf("after delete %v err=%v", all, err)
	}
}

func TestUpsertPackagesBatchFTS(t *testing.T) {
	s := openTemp(t)
	_, err := s.UpsertProgram(Program{Name: "pkg0005", Summary: "installed", Version: "1", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	baseFTS := s.FTSSyncCount.Load()

	const n = 1200
	entries := make([]CatalogEntry, 0, n)
	for i := 0; i < n; i++ {
		name := "pkg" + pad4(i)
		entries = append(entries, CatalogEntry{Name: name, PackageName: name, Summary: "sum-" + name})
	}
	chunks := 0
	added, err := s.UpsertCatalogProgramsBatch(entries, 500, func(done, total int) {
		chunks++
		if done%500 != 0 && done != total {
			t.Fatalf("unexpected progress done=%d total=%d", done, total)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	// pkg0005 already mirrored into packages by UpsertProgram → 1199 new.
	if added != n-1 {
		t.Fatalf("added=%d want %d", added, n-1)
	}
	if chunks != 3 {
		t.Fatalf("chunks=%d want 3", chunks)
	}
	ftsDelta := s.FTSSyncCount.Load() - baseFTS
	if ftsDelta != int64(n-1) {
		t.Fatalf("FTS syncs=%d want %d", ftsDelta, n-1)
	}
	for _, q := range []string{"pkg0000", "pkg0500", "pkg1199"} {
		hits, err := s.SearchPackagesFiltered(q, PackageFilter{})
		if err != nil || len(hits) == 0 {
			t.Fatalf("search %q: %+v err=%v", q, hits, err)
		}
	}
	p, err := s.GetProgramByName("pkg0005")
	if err != nil || !p.IsInstalled || p.Summary != "installed" {
		t.Fatalf("installed retention: %+v err=%v", p, err)
	}
	pkg, err := s.GetPackageByName("pkg0005")
	if err != nil || !pkg.IsInstalled {
		t.Fatalf("package install flag: %+v err=%v", pkg, err)
	}
}

func pad4(i int) string {
	s := "0000" + itoa(i)
	return s[len(s)-4:]
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
