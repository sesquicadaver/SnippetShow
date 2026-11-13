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
	if info.Phase != "4" {
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

func TestListProgramsFilters(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	for _, p := range []db.Program{
		{Name: "ffmpeg", Summary: "video", IsInstalled: true},
		{Name: "curl", Summary: "http", IsInstalled: true},
		{Name: "figlet", Summary: "banners", IsInstalled: false},
		{Name: "7z", Summary: "archive", IsInstalled: true},
	} {
		if _, err := store.UpsertProgram(p, false); err != nil {
			t.Fatal(err)
		}
	}

	api := NewWithStore(store)
	installed, err := api.ListPrograms(ListQuery{Mode: "installed"})
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 3 {
		t.Fatalf("installed=%d", len(installed))
	}
	letterF, err := api.ListPrograms(ListQuery{Mode: "all", Letter: "f"})
	if err != nil {
		t.Fatal(err)
	}
	if len(letterF) != 2 { // ffmpeg, figlet
		t.Fatalf("letter f: %+v", letterF)
	}
	digits, err := api.ListPrograms(ListQuery{Mode: "installed", Letter: "0-9"})
	if err != nil {
		t.Fatal(err)
	}
	if len(digits) != 1 || digits[0].Name != "7z" {
		t.Fatalf("digits: %+v", digits)
	}
	fts, err := api.ListPrograms(ListQuery{Mode: "installed", Query: "video"})
	if err != nil || len(fts) != 1 || fts[0].Name != "ffmpeg" {
		t.Fatalf("fts: %+v err=%v", fts, err)
	}
}

func TestUpsertCatalogDoesNotDemoteInstalled(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.UpsertProgram(db.Program{Name: "curl", Summary: "http", IsInstalled: true}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertCatalogProgram("curl", "curl", "from repo"); err != nil {
		t.Fatal(err)
	}
	p, err := store.GetProgramByName("curl")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsInstalled {
		t.Fatal("catalog upsert demoted installed program")
	}
	if _, err := store.UpsertCatalogProgram("htop", "htop", "process viewer"); err != nil {
		t.Fatal(err)
	}
	h, err := store.GetProgramByName("htop")
	if err != nil || h.IsInstalled || h.Summary != "process viewer" {
		t.Fatalf("%+v err=%v", h, err)
	}
}

func TestSnippetAPIAndScriptDraft(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	id, err := store.UpsertProgram(db.Program{Name: "ffmpeg", Summary: "video", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	api := NewWithStore(store)
	if _, err := api.CreateSnippet(SnippetInput{ProgramID: id, Title: "", Command: "x"}); err == nil {
		t.Fatal("validation expected")
	}
	sn, err := api.CreateSnippet(SnippetInput{
		ProgramID: id, Title: "encode", Command: "ffmpeg -i a b", Tags: "video",
	})
	if err != nil {
		t.Fatal(err)
	}
	detail, err := api.GetProgram("ffmpeg", false)
	if err != nil || len(detail.Snippets) != 1 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	_, err = api.UpdateSnippet(sn.ID, SnippetInput{Title: "encode2", Command: "ffmpeg -y -i a b", Tags: "t"})
	if err != nil {
		t.Fatal(err)
	}
	safety := api.AnalyzeCommand("sudo rm -rf /")
	if !safety.Dangerous || len(safety.Reasons) < 2 {
		t.Fatalf("%+v", safety)
	}
	run, err := api.RunCommand(RunRequest{Command: "sudo true", Confirmed: false})
	if err != nil {
		t.Fatal(err)
	}
	if !run.NeedsConfirm || run.Started {
		t.Fatalf("%+v", run)
	}
	// H2: path-prefixed / env wrappers must still gate Run (NeedsConfirm).
	bypass, err := api.RunCommand(RunRequest{Command: "/usr/bin/sudo true", Confirmed: false})
	if err != nil {
		t.Fatal(err)
	}
	if !bypass.NeedsConfirm || bypass.Started {
		t.Fatalf("path sudo: %+v", bypass)
	}
	envBypass, err := api.RunCommand(RunRequest{Command: "env sudo true", Confirmed: false})
	if err != nil {
		t.Fatal(err)
	}
	if !envBypass.NeedsConfirm || envBypass.Started {
		t.Fatalf("env sudo: %+v", envBypass)
	}
	draft, err := api.AddToScript("ffmpeg -i a b")
	if err != nil || draft.Count != 1 {
		t.Fatalf("%+v err=%v", draft, err)
	}
	draft2, err := api.AddToScript("echo done")
	if err != nil || draft2.Count != 2 {
		t.Fatalf("draft must persist: %+v", draft2)
	}
	if err := api.DeleteSnippet(sn.ID); err != nil {
		t.Fatal(err)
	}
	list, err := api.ListSnippets(id)
	if err != nil || len(list) != 0 {
		t.Fatalf("%v err=%v", list, err)
	}
}

func TestScriptBuilderAndBackupAPI(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	pid, err := store.UpsertProgram(db.Program{Name: "curl", Summary: "http", IsInstalled: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	api := NewWithStore(store)
	if _, err := api.AddToScript("echo one"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.AddToScript("echo two"); err != nil {
		t.Fatal(err)
	}
	draft, err := api.MoveScriptLine(1, "up")
	if err != nil || draft.Lines[0] != "echo two" {
		t.Fatalf("%+v err=%v", draft, err)
	}
	body := api.GenerateScript()
	if body == "" || draft.Body == "" {
		t.Fatal("empty body")
	}
	cs, err := api.SaveCustomScript(0, CustomScriptInput{Name: "demo", ScriptBody: body})
	if err != nil || cs.ID == 0 {
		t.Fatalf("%+v err=%v", cs, err)
	}
	api.ClearScriptDraft()
	if api.ScriptDraft().Count != 0 {
		t.Fatal("clear failed")
	}
	loaded, err := api.LoadCustomScript(cs.ID)
	if err != nil || loaded.Count == 0 {
		t.Fatalf("%+v err=%v", loaded, err)
	}
	scripts, err := api.ListCustomScripts()
	if err != nil || len(scripts) != 1 {
		t.Fatalf("%v err=%v", scripts, err)
	}
	shPath := filepath.Join(t.TempDir(), "out.sh")
	if err := api.ExportScriptFile(shPath, ""); err != nil {
		t.Fatal(err)
	}
	_, err = api.CreateSnippet(SnippetInput{ProgramID: pid, Title: "get", Command: "curl -I https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	bak := filepath.Join(t.TempDir(), "bak.json")
	if err := api.ExportBackup(bak); err != nil {
		t.Fatal(err)
	}
	res, err := api.ImportBackup(bak)
	if err != nil {
		t.Fatal(err)
	}
	if res.SnippetsSkipped < 1 || res.ScriptsSkipped < 1 {
		t.Fatalf("expected skip on re-import: %+v", res)
	}
	run, err := api.RunScript(RunRequest{Command: "sudo true", Confirmed: false})
	if err != nil {
		t.Fatal(err)
	}
	if !run.NeedsConfirm {
		t.Fatalf("%+v", run)
	}
	if err := api.DeleteCustomScript(cs.ID); err != nil {
		t.Fatal(err)
	}
}

func TestGetProgramInstallHintForUninstalled(t *testing.T) {
	store, err := db.Open(filepath.Join(t.TempDir(), "wiki.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.UpsertProgram(db.Program{
		Name: "figlet", PackageName: "figlet", Summary: "banners", IsInstalled: false,
	}, false); err != nil {
		t.Fatal(err)
	}
	api := NewWithStore(store)
	detail, err := api.GetProgram("figlet", false)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Program.IsInstalled {
		t.Fatal("expected uninstalled")
	}
	// InstallCommand may be empty if PM undetectable in CI; do not require it.
	if len(detail.Commands) != 0 {
		t.Fatalf("D6: unexpected man cmds for uninstalled: %+v", detail.Commands)
	}
}
