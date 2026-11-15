package manparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandBraceAlternatives(t *testing.T) {
	opts := ExpandBraceAlternatives("-ao{a|s|t|u}", "set overwrite mode")
	if len(opts) != 4 {
		t.Fatalf("got %d opts: %+v", len(opts), opts)
	}
	wantFlags := []string{"-aoa", "-aos", "-aot", "-aou"}
	for i, want := range wantFlags {
		if opts[i].Flag != want {
			t.Fatalf("flag[%d]=%q want %q", i, opts[i].Flag, want)
		}
		if !strings.Contains(opts[i].Description, "set overwrite mode") {
			t.Fatalf("desc missing base: %q", opts[i].Description)
		}
		if !strings.Contains(opts[i].Description, "(варіант: "+want[len(want)-1:]+")") {
			t.Fatalf("desc missing variant ann: %q", opts[i].Description)
		}
	}
}

func TestExpandBraceAlternativesNoBrace(t *testing.T) {
	opts := ExpandBraceAlternatives("-v", "verbose")
	if len(opts) != 1 || opts[0].Flag != "-v" || opts[0].Description != "verbose" {
		t.Fatalf("%+v", opts)
	}
}

func TestExpandBraceAlternativesEqualsForm(t *testing.T) {
	opts := ExpandBraceAlternatives("--mode={fast|slow}", "pick speed")
	if len(opts) != 2 {
		t.Fatalf("%+v", opts)
	}
	if opts[0].Flag != "--mode=fast" || opts[1].Flag != "--mode=slow" {
		t.Fatalf("%+v", opts)
	}
}

func TestParseOptionsExpandsBracesFromFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sample.man"))
	if err != nil {
		t.Fatal(err)
	}
	opts := ParseOptionsFromText(string(raw))
	found := map[string]bool{}
	for _, o := range opts {
		if strings.Contains(o.Flag, "{") {
			t.Fatalf("brace left in flag after SS-26 expand: %q", o.Flag)
		}
		found[o.Flag] = true
	}
	for _, want := range []string{"-aoa", "-aos", "-aot", "-aou"} {
		if !found[want] {
			t.Fatalf("missing expanded %q in %+v", want, opts)
		}
	}
}
