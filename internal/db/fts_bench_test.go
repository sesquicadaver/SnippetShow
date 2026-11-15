package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FTS@~3000 NFR (ТЗ §5 / FIX-FTS-3k): search ≤ 50 ms on a ~3000-program fixture.
const (
	ftsBenchProgramCount = 3000
	ftsBenchBudget       = 50 * time.Millisecond
	ftsBenchNeedle       = "ffmpegbench"
)

// seedFTSScaleFixture inserts ~n catalog programs (+ a few man flags / snippets on needles)
// into a fresh Store. Uses batch upsert so setup stays seconds-scale, not minutes.
func seedFTSScaleFixture(t testing.TB, n int) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "wiki-fts-scale.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// Seed PATH-style programs (not packages): FTS@3k gate covers programs_fts + man/snippets.
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("pkg%04d", i)
		summary := fmt.Sprintf("utility package number %d for scale search", i)
		switch {
		case i == 42:
			name = ftsBenchNeedle
			summary = "video transcoder bench needle " + ftsBenchNeedle
		case i%200 == 0:
			summary += " network transfer helper"
		case i%137 == 0:
			summary += " archive compression tool"
		}
		if _, err := s.UpsertProgram(Program{
			Name: name, PackageName: name, Summary: summary, IsInstalled: true,
		}, false); err != nil {
			t.Fatalf("seed program %q: %v", name, err)
		}
	}

	// Enrich the needle with man flags + snippet so FTS covers all columns.
	p, err := s.GetProgramByName(ftsBenchNeedle)
	if err != nil {
		t.Fatalf("needle program: %v", err)
	}
	if err := s.ReplaceManCommands(p.ID, []ManCommand{
		{Flag: "-i", Description: "input file for " + ftsBenchNeedle},
		{Flag: "--bench-flag", Description: "synthetic flag token ftsflagtoken"},
	}); err != nil {
		t.Fatalf("man: %v", err)
	}
	if _, err := s.InsertUserSnippet(UserSnippet{
		ProgramID: p.ID,
		Title:     "bench encode",
		Command:   ftsBenchNeedle + " -i in.mp4 out.mkv",
		Tags:      "video benchsnippet",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("snippet: %v", err)
	}

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM programs`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count < n {
		t.Fatalf("programs=%d want ≥%d", count, n)
	}
	return s
}

// TestFTSLatencyAt3000 proves ТЗ §5 search budget on a ~3000-row fixture.
// Override with FTS_LATENCY_BUDGET_MS (milliseconds) if a host needs a documented deviation.
func TestFTSLatencyAt3000(t *testing.T) {
	s := seedFTSScaleFixture(t, ftsBenchProgramCount)
	budget := ftsBenchBudget
	if v := os.Getenv("FTS_LATENCY_BUDGET_MS"); v != "" {
		ms, err := time.ParseDuration(v + "ms")
		if err != nil {
			t.Fatalf("FTS_LATENCY_BUDGET_MS: %v", err)
		}
		budget = ms
	}

	// Warm-up (ignore timing): first MATCH may pay sqlite/page-cache cost.
	if _, err := s.Search(ftsBenchNeedle); err != nil {
		t.Fatal(err)
	}

	const rounds = 20
	var total time.Duration
	var max time.Duration
	for i := 0; i < rounds; i++ {
		start := time.Now()
		hits, err := s.Search(ftsBenchNeedle)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) == 0 {
			t.Fatal("expected needle hit")
		}
		total += elapsed
		if elapsed > max {
			max = elapsed
		}
	}
	avg := total / rounds
	t.Logf("FTS@%d Search(%q): avg=%s max=%s budget=%s (rounds=%d)",
		ftsBenchProgramCount, ftsBenchNeedle, avg, max, budget, rounds)
	if avg > budget {
		t.Fatalf("avg latency %s exceeds budget %s (max=%s)", avg, budget, max)
	}
	if max > budget*3 {
		// Soft guard: a single spike ≫ budget still indicates a problem even if avg is OK.
		t.Fatalf("max latency %s exceeds 3× budget %s", max, budget)
	}

	// Secondary queries (flags / snippet tags) must also stay under budget on avg.
	for _, q := range []string{"ftsflagtoken", "benchsnippet"} {
		start := time.Now()
		hits, err := s.Search(q)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) == 0 {
			t.Fatalf("expected hits for %q", q)
		}
		if elapsed > budget {
			t.Fatalf("Search(%q)=%s exceeds budget %s", q, elapsed, budget)
		}
	}
}

func BenchmarkSearchAt3000(b *testing.B) {
	s := seedFTSScaleFixture(b, ftsBenchProgramCount)
	if _, err := s.Search(ftsBenchNeedle); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hits, err := s.Search(ftsBenchNeedle)
		if err != nil {
			b.Fatal(err)
		}
		if len(hits) == 0 {
			b.Fatal("no hits")
		}
	}
}
