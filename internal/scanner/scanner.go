// Package scanner detects the package manager and inventories binaries / repo catalogs.
// Phase 1: P1-5…P1-8. Policy D5 applies to empty cache.
package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrUnsupportedPM is returned when no known package manager is available.
var ErrUnsupportedPM = errors.New("scanner: unsupported package manager")

// PackageManager is a detected distribution package manager.
type PackageManager string

const (
	PMUnknown PackageManager = ""
	PMDpkg    PackageManager = "dpkg"
	PMPacman  PackageManager = "pacman"
	PMRpmDnf  PackageManager = "rpm+dnf"
)

// CacheStatus describes repo catalog availability (D5).
type CacheStatus string

const (
	CacheOK    CacheStatus = "ok"
	CacheEmpty CacheStatus = "cache_empty"
	CacheError CacheStatus = "error"
)

// InstalledProgram is an executable enriched with PM metadata when available.
type InstalledProgram struct {
	Name        string
	Path        string
	PackageName string
	Version     string
	Summary     string
}

// RepoPackage is a name+summary from the distribution package cache.
type RepoPackage struct {
	Name    string
	Summary string
}

// CatalogResult is the outcome of listing the full repo catalog.
type CatalogResult struct {
	Packages []RepoPackage
	Status   CacheStatus
	Hint     string
}

// Scanner performs PM detection, PATH scans and catalog reads.
type Scanner struct {
	Runner Runner
	// PathEnv overrides PATH for tests; empty uses os.Getenv("PATH").
	PathEnv string
	// SkipCanonical, when true, scans only PathEnv directories (unit tests).
	SkipCanonical bool
	// ExtraDirs are always scanned (canonical TOR paths) unless SkipCanonical.
	ExtraDirs []string
	// Home overrides os.UserHomeDir for ~/.local/bin.
	Home string
}

// New returns a Scanner bound to the real OS runner.
func New() *Scanner {
	return &Scanner{Runner: OSRunner{}}
}

func (s *Scanner) runner() Runner {
	if s == nil || s.Runner == nil {
		return OSRunner{}
	}
	return s.Runner
}

// DetectPM identifies the host package manager (dpkg / pacman / rpm+dnf).
func DetectPM() (PackageManager, error) {
	return New().DetectPM()
}

// DetectPM identifies the package manager using LookPath probes.
func (s *Scanner) DetectPM() (PackageManager, error) {
	r := s.runner()
	if _, err := r.LookPath("dpkg-query"); err == nil {
		return PMDpkg, nil
	}
	if _, err := r.LookPath("pacman"); err == nil {
		return PMPacman, nil
	}
	if _, err := r.LookPath("rpm"); err == nil {
		// Prefer dnf for catalog; rpm alone is enough to classify as rpm family.
		return PMRpmDnf, nil
	}
	return PMUnknown, ErrUnsupportedPM
}

// ScanInstalled lists unique executable basenames from PATH + canonical dirs,
// enriched with package metadata when the PM is available.
func ScanInstalled() ([]InstalledProgram, error) {
	return New().ScanInstalled(context.Background())
}

// ScanInstalled lists installed executables with optional PM enrichment.
func (s *Scanner) ScanInstalled(ctx context.Context) ([]InstalledProgram, error) {
	names, paths, err := s.scanExecutables()
	if err != nil {
		return nil, err
	}
	pm, pmErr := s.DetectPM()
	out := make([]InstalledProgram, 0, len(names))
	for _, name := range names {
		p := InstalledProgram{Name: name, Path: paths[name]}
		if pmErr == nil {
			meta, err := s.lookupPackage(ctx, pm, name, paths[name])
			if err == nil && meta != nil {
				p.PackageName = meta.PackageName
				p.Version = meta.Version
				p.Summary = meta.Summary
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// ScanExecutableNames returns deduplicated binary names (P1-6).
func (s *Scanner) ScanExecutableNames() ([]string, error) {
	names, _, err := s.scanExecutables()
	return names, err
}

func (s *Scanner) scanExecutables() ([]string, map[string]string, error) {
	dirs := s.pathDirs()
	seen := map[string]string{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			full := filepath.Join(dir, name)
			info, err := e.Info()
			if err != nil {
				continue
			}
			mode := info.Mode()
			if mode&0o111 == 0 {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = full
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, seen, nil
}

func (s *Scanner) pathDirs() []string {
	pathEnv := s.PathEnv
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	var dirs []string
	seen := map[string]struct{}{}
	add := func(d string) {
		d = filepath.Clean(d)
		if d == "" || d == "." {
			return
		}
		if _, ok := seen[d]; ok {
			return
		}
		seen[d] = struct{}{}
		dirs = append(dirs, d)
	}
	for _, p := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		if p != "" {
			add(p)
		}
	}
	if s.SkipCanonical {
		for _, e := range s.ExtraDirs {
			add(e)
		}
		return dirs
	}
	canonical := []string{"/usr/bin", "/bin", "/usr/local/bin"}
	home := s.Home
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}
	if home != "" {
		canonical = append(canonical, filepath.Join(home, ".local", "bin"))
	}
	for _, c := range canonical {
		add(c)
	}
	for _, e := range s.ExtraDirs {
		add(e)
	}
	return dirs
}

type pkgMeta struct {
	PackageName string
	Version     string
	Summary     string
	Section     string // raw dpkg Section / pacman group (D13.8)
}

func (s *Scanner) lookupPackage(ctx context.Context, pm PackageManager, binName, binPath string) (*pkgMeta, error) {
	switch pm {
	case PMDpkg:
		return s.lookupDpkg(ctx, binPath, binName)
	case PMPacman:
		return s.lookupPacman(ctx, binPath, binName)
	case PMRpmDnf:
		return s.lookupRpm(ctx, binPath, binName)
	default:
		return nil, fmt.Errorf("scanner: no PM for lookup")
	}
}

// InstallCommand returns a suggested install command for an uninstalled package (D6 UI).
func InstallCommand(pm PackageManager, pkg string) string {
	switch pm {
	case PMDpkg:
		return "sudo apt install " + pkg
	case PMPacman:
		return "sudo pacman -S " + pkg
	case PMRpmDnf:
		return "sudo dnf install " + pkg
	default:
		return ""
	}
}

// CacheHint returns the D5 user hint for an empty package cache.
func CacheHint(pm PackageManager) string {
	switch pm {
	case PMDpkg:
		return "Кеш пакетів порожній. Оновіть: sudo apt update"
	case PMPacman:
		return "Кеш пакетів порожній. Оновіть: sudo pacman -Sy"
	case PMRpmDnf:
		return "Кеш пакетів порожній. Оновіть: sudo dnf makecache"
	default:
		return "Кеш пакетів порожній або пакетний менеджер невідомий."
	}
}
