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

func TestListProgramsFilteredAndSearchLetter(t *testing.T) {
	s := openTemp(t)
	for _, p := range []Program{
		{Name: "alpha", Summary: "a tool", IsInstalled: true},
		{Name: "beta", Summary: "b tool", IsInstalled: false},
		{Name: "9base", Summary: "digit", IsInstalled: true},
	} {
		if _, err := s.UpsertProgram(p, false); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.ListProgramsFiltered(ProgramFilter{})
	if err != nil || len(all) != 3 {
		t.Fatalf("all=%v err=%v", all, err)
	}
	inst, err := s.ListProgramsFiltered(ProgramFilter{InstalledOnly: true})
	if err != nil || len(inst) != 2 {
		t.Fatalf("installed=%v err=%v", inst, err)
	}
	letterA, err := s.ListProgramsFiltered(ProgramFilter{Letter: "A"})
	if err != nil || len(letterA) != 1 || letterA[0].Name != "alpha" {
		t.Fatalf("letter A: %+v err=%v", letterA, err)
	}
	digits, err := s.ListProgramsFiltered(ProgramFilter{Letter: "0-9", InstalledOnly: true})
	if err != nil || len(digits) != 1 || digits[0].Name != "9base" {
		t.Fatalf("digits: %+v err=%v", digits, err)
	}
	hits, err := s.SearchFiltered("tool", ProgramFilter{InstalledOnly: true, Letter: "a"})
	if err != nil || len(hits) != 1 || hits[0].Name != "alpha" {
		t.Fatalf("fts filter: %+v err=%v", hits, err)
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

func TestSnippetCRUDAndValidation(t *testing.T) {
	s := openTemp(t)
	id, err := s.UpsertProgram(Program{Name: "ffmpeg", Summary: "video", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUserSnippet(id, "", "ffmpeg -i a b", ""); err == nil {
		t.Fatal("empty title should fail")
	}
	if _, err := s.CreateUserSnippet(id, "encode", "  ", ""); err == nil {
		t.Fatal("empty command should fail")
	}
	sn, err := s.CreateUserSnippet(id, "encode h265", "ffmpeg -i in.mp4 -c:v libx265 out.mkv", "video h265")
	if err != nil {
		t.Fatal(err)
	}
	list, err := s.ListUserSnippets(id)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	updated, err := s.UpdateUserSnippet(sn.ID, "encode av1", "ffmpeg -i in.mp4 -c:v libsvtav1 out.mkv", "av1")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "encode av1" || updated.Tags != "av1" {
		t.Fatalf("%+v", updated)
	}
	hits, err := s.Search("av1")
	if err != nil || len(hits) == 0 {
		t.Fatalf("fts after update: %+v err=%v", hits, err)
	}
	if err := s.DeleteUserSnippet(sn.ID); err != nil {
		t.Fatal(err)
	}
	list, err = s.ListUserSnippets(id)
	if err != nil || len(list) != 0 {
		t.Fatalf("after delete list=%v err=%v", list, err)
	}
}

func TestCustomScriptsCRUD(t *testing.T) {
	s := openTemp(t)
	cs, err := s.CreateCustomScript("demo", "#!/usr/bin/env bash\necho hi\n")
	if err != nil {
		t.Fatal(err)
	}
	if cs.ID == 0 || cs.UpdatedAt == "" {
		t.Fatalf("%+v", cs)
	}
	got, err := s.GetCustomScript(cs.ID)
	if err != nil || got.Name != "demo" {
		t.Fatalf("%+v err=%v", got, err)
	}
	upd, err := s.UpdateCustomScript(cs.ID, "demo2", "#!/usr/bin/env bash\necho bye\n")
	if err != nil || upd.Name != "demo2" {
		t.Fatalf("%+v err=%v", upd, err)
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

func TestUpsertCatalogProgramsBatchFTS(t *testing.T) {
	s := openTemp(t)
	// Installed row must not be demoted by catalog upsert.
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
	// pkg0005 already existed → 1199 new.
	if added != n-1 {
		t.Fatalf("added=%d want %d", added, n-1)
	}
	if chunks != 3 {
		t.Fatalf("chunks=%d want 3", chunks)
	}
	ftsDelta := s.FTSSyncCount.Load() - baseFTS
	// Touched = all new + summary fill skipped for installed (already had summary).
	// Installed pkg0005 unchanged → no extra FTS; 1199 new rows synced after chunks.
	if ftsDelta != int64(n-1) {
		t.Fatalf("FTS syncs=%d want %d (deferred per touched, not interleaved mid-write)", ftsDelta, n-1)
	}
	for _, q := range []string{"pkg0000", "pkg0500", "pkg1199"} {
		hits, err := s.Search(q)
		if err != nil || len(hits) == 0 {
			t.Fatalf("search %q: %+v err=%v", q, hits, err)
		}
	}
	p, err := s.GetProgramByName("pkg0005")
	if err != nil || !p.IsInstalled || p.Summary != "installed" {
		t.Fatalf("installed retention: %+v err=%v", p, err)
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
