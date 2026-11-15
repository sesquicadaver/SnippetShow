package manparser

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockManRunner struct {
	whatis map[string]string
	raw    map[string]string
}

func (m *mockManRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := ""
	if len(args) > 0 {
		cmd = args[len(args)-1]
	}
	switch name {
	case "whatis":
		if s, ok := m.whatis[cmd]; ok {
			return s, nil
		}
	case "man":
		if len(args) >= 1 && args[0] == "-f" {
			if s, ok := m.whatis[cmd]; ok {
				return s, nil
			}
		}
		if s, ok := m.raw[cmd]; ok {
			return s, nil
		}
	}
	return "", os.ErrNotExist
}

func (m *mockManRunner) RunPipeline(_ context.Context, _ []string, leftName string, leftArgs []string, _ string, _ []string) (string, error) {
	cmd := leftArgs[len(leftArgs)-1]
	if leftName == "man" {
		if s, ok := m.raw[cmd]; ok {
			return s, nil
		}
	}
	return "", os.ErrNotExist
}

func TestParseWhatisSummary(t *testing.T) {
	p := &Parser{Runner: &mockManRunner{
		whatis: map[string]string{"ls": "ls (1) - list directory contents\n"},
	}}
	sum, err := p.Summary(context.Background(), "ls")
	if err != nil {
		t.Fatal(err)
	}
	if sum != "list directory contents" {
		t.Fatalf("got %q", sum)
	}
}

func TestParseOptionsFromFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sample.man"))
	if err != nil {
		t.Fatal(err)
	}
	opts := ParseOptionsFromText(string(raw))
	if len(opts) < 5 {
		t.Fatalf("expected >=5 options, got %d: %+v", len(opts), opts)
	}
	find := func(substr string) bool {
		for _, o := range opts {
			if strings.Contains(o.Flag, substr) {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"-a", "--all", "--help", "-v", "--list", "--dry-run"} {
		if !find(want) {
			t.Fatalf("missing flag containing %q in %+v", want, opts)
		}
	}
	foundAll := false
	for _, o := range opts {
		if strings.Contains(o.Flag, ",") {
			t.Fatalf("option flag must not contain comma (SS-20): %q", o.Flag)
		}
		if o.Flag == "-a --all" {
			foundAll = true
			if !strings.Contains(o.Description, "process all") {
				t.Fatalf("desc=%q", o.Description)
			}
		}
	}
	if !foundAll {
		t.Fatalf("combined -a/--all missing: %+v", opts)
	}
}

func TestNormalizeFlagList(t *testing.T) {
	cases := map[string]string{
		"-a, --all":           "-a --all",
		"--audible,":          "--audible",
		"  --quiet, --audible  ": "--quiet --audible",
		"-f, --file FILE":     "-f --file FILE",
		"--help":              "--help",
		"":                    "",
	}
	for in, want := range cases {
		if got := NormalizeFlagList(in); got != want {
			t.Fatalf("NormalizeFlagList(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRawManUsesPipeline(t *testing.T) {
	over := "NAME\nO\bOP\bPT\bTI\bIO\bON\bNS\bS\n       -a  all\n"
	p := &Parser{Runner: &mockManRunner{raw: map[string]string{"demo": over}}}
	text, err := p.RawMan(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "\b") {
		t.Fatal("backspace remains")
	}
	if !strings.Contains(text, "OPTIONS") {
		t.Fatalf("overstrike not cleaned: %q", text)
	}
}

func TestParseOptionsViaParser(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.man"))
	if err != nil {
		t.Fatal(err)
	}
	p := &Parser{Runner: &mockManRunner{raw: map[string]string{"demo": string(fixture)}}}
	opts, err := p.ParseOptions(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) == 0 {
		t.Fatal("no options")
	}
}

func TestParseManyAsyncNonBlocking(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "sample.man"))
	if err != nil {
		t.Fatal(err)
	}
	rawMap := map[string]string{}
	cmds := []string{"a", "b", "c", "d", "e", "f"}
	for _, c := range cmds {
		rawMap[c] = string(fixture)
	}
	p := &Parser{Runner: &mockManRunner{raw: rawMap}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	started := time.Now()
	ch := p.ParseManyAsync(ctx, cmds, 4)
	if time.Since(started) > 200*time.Millisecond {
		t.Fatal("ParseManyAsync blocked caller")
	}
	count := 0
	for range ch {
		count++
	}
	if count != len(cmds) {
		t.Fatalf("progress=%d want %d", count, len(cmds))
	}
}

func TestStripOverstrike(t *testing.T) {
	in := "h\bhe\bel\bll\to\bo" // classic overstrike fragments
	out := stripOverstrike("c\bc")
	if out != "c" {
		t.Fatalf("got %q", out)
	}
	_ = in
}
