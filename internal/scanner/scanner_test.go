package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockRunner struct {
	bins map[string]string
	outs map[string]string
	errs map[string]error
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if p, ok := m.bins[file]; ok {
		return p, nil
	}
	return "", errors.New("not found: " + file)
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if err, ok := m.errs[key]; ok {
		return m.outs[key], err
	}
	// Prefix match for flexible fixtures.
	for k, v := range m.outs {
		if strings.HasPrefix(key, k) || k == name {
			if err, ok := m.errs[k]; ok {
				return v, err
			}
			return v, nil
		}
	}
	return "", errors.New("unexpected command: " + key)
}

func TestDetectPMDpkg(t *testing.T) {
	s := &Scanner{Runner: &mockRunner{bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"}}}
	pm, err := s.DetectPM()
	if err != nil || pm != PMDpkg {
		t.Fatalf("got %q %v", pm, err)
	}
}

func TestDetectPMPacman(t *testing.T) {
	s := &Scanner{Runner: &mockRunner{bins: map[string]string{"pacman": "/usr/bin/pacman"}}}
	pm, err := s.DetectPM()
	if err != nil || pm != PMPacman {
		t.Fatalf("got %q %v", pm, err)
	}
}

func TestDetectPMRpm(t *testing.T) {
	s := &Scanner{Runner: &mockRunner{bins: map[string]string{"rpm": "/usr/bin/rpm"}}}
	pm, err := s.DetectPM()
	if err != nil || pm != PMRpmDnf {
		t.Fatalf("got %q %v", pm, err)
	}
}

func TestDetectPMUnknown(t *testing.T) {
	s := &Scanner{Runner: &mockRunner{bins: map[string]string{}}}
	_, err := s.DetectPM()
	if !errors.Is(err, ErrUnsupportedPM) {
		t.Fatalf("want ErrUnsupportedPM, got %v", err)
	}
}

func TestScanExecutableNamesDedup(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	writeExec(t, filepath.Join(dir1, "alpha"))
	writeExec(t, filepath.Join(dir2, "alpha"))
	writeExec(t, filepath.Join(dir1, "beta"))
	_ = os.WriteFile(filepath.Join(dir1, "noexec"), []byte("x"), 0o644)

	s := &Scanner{
		Runner:        &mockRunner{bins: map[string]string{}},
		PathEnv:       dir1 + string(os.PathListSeparator) + dir2,
		Home:          t.TempDir(),
		SkipCanonical: true,
	}
	names, err := s.ScanExecutableNames()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, n := range names {
		if n == "alpha" || n == "beta" {
			count++
		}
		if n == "noexec" {
			t.Fatal("non-executable included")
		}
	}
	if count != 2 {
		t.Fatalf("names=%v", names)
	}
}

func TestScanInstalledWithDpkgMeta(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "curl")
	writeExec(t, bin)
	r := &mockRunner{
		bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		outs: map[string]string{
			"dpkg-query -S " + bin: "curl:amd64: " + bin,
			"dpkg-query -W -f=${Package}\t${Version}\t${binary:Package}\t${Description}\n curl": "curl\t8.5.0\tcurl\tcommand line tool for transferring data\n",
		},
	}
	s := &Scanner{Runner: r, PathEnv: dir, Home: t.TempDir(), SkipCanonical: true}
	progs, err := s.ScanInstalled(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(progs) != 1 || progs[0].Name != "curl" {
		t.Fatalf("%+v", progs)
	}
	if progs[0].PackageName != "curl" || progs[0].Version != "8.5.0" {
		t.Fatalf("meta=%+v", progs[0])
	}
	if !strings.Contains(progs[0].Summary, "transferring") {
		t.Fatalf("summary=%q", progs[0].Summary)
	}
}

func TestCatalogEmptyCacheD5(t *testing.T) {
	r := &mockRunner{
		bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		errs: map[string]error{"apt-cache": errors.New("fail")},
		outs: map[string]string{},
	}
	s := &Scanner{Runner: r}
	res, err := s.ListRepoCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != CacheEmpty {
		t.Fatalf("status=%s", res.Status)
	}
	if res.Hint == "" || len(res.Packages) != 0 {
		t.Fatalf("%+v", res)
	}
}

func TestCatalogAptOK(t *testing.T) {
	r := &mockRunner{
		bins: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		outs: map[string]string{
			"apt-cache pkgnames": "ffmpeg\ncurl\n",
			"apt-cache search .": "ffmpeg - video tools\ncurl - http client\n",
		},
	}
	s := &Scanner{Runner: r}
	res, err := s.ListRepoCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != CacheOK || len(res.Packages) != 2 {
		t.Fatalf("%+v", res)
	}
}

func TestInstallCommand(t *testing.T) {
	if InstallCommand(PMDpkg, "ffmpeg") != "sudo apt install ffmpeg" {
		t.Fatal(InstallCommand(PMDpkg, "ffmpeg"))
	}
}

func writeExec(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
