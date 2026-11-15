package scanner

import (
	"bufio"
	"context"
	"strings"
)

// ListRepoCatalog returns available packages from the local PM cache (P1-8 / D5).
func (s *Scanner) ListRepoCatalog(ctx context.Context) (CatalogResult, error) {
	pm, err := s.DetectPM()
	if err != nil {
		return CatalogResult{Status: CacheError, Hint: CacheHint(PMUnknown)}, err
	}
	switch pm {
	case PMDpkg:
		return s.catalogApt(ctx)
	case PMPacman:
		return s.catalogPacman(ctx)
	case PMRpmDnf:
		return s.catalogDnf(ctx)
	default:
		return CatalogResult{Status: CacheError, Hint: CacheHint(pm)}, ErrUnsupportedPM
	}
}

func (s *Scanner) catalogApt(ctx context.Context) (CatalogResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()
	out, err := r.Run(ctx, "apt-cache", "pkgnames")
	if err != nil {
		// Soft-fail: empty cache / apt missing lists.
		return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMDpkg)}, nil
	}
	names := strings.Fields(out)
	if len(names) == 0 {
		return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMDpkg)}, nil
	}

	// Summaries via apt-cache search . (may be large); prefer dumpavail short parse when available.
	sumOut, sumErr := r.Run(ctx, "apt-cache", "search", ".")
	summaries := map[string]string{}
	if sumErr == nil {
		sc := bufio.NewScanner(strings.NewReader(sumOut))
		for sc.Scan() {
			line := sc.Text()
			// "name - summary"
			if i := strings.Index(line, " - "); i > 0 {
				summaries[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+3:])
			}
		}
	}

	pkgs := make([]RepoPackage, 0, len(names))
	for _, n := range names {
		pkgs = append(pkgs, RepoPackage{Name: n, Summary: summaries[n]})
	}
	return CatalogResult{Packages: pkgs, Status: CacheOK}, nil
}

func (s *Scanner) catalogPacman(ctx context.Context) (CatalogResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	out, err := s.runner().Run(ctx, "pacman", "-Sl")
	if err != nil {
		return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMPacman)}, nil
	}
	var pkgs []RepoPackage
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		// repo name version [installed]
		if len(fields) < 3 {
			continue
		}
		name := fields[1]
		summary := ""
		pkgs = append(pkgs, RepoPackage{Name: name, Summary: summary})
	}
	if len(pkgs) == 0 {
		return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMPacman)}, nil
	}
	return CatalogResult{Packages: pkgs, Status: CacheOK}, nil
}

func (s *Scanner) catalogDnf(ctx context.Context) (CatalogResult, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()
	out, err := r.Run(ctx, "dnf", "list", "--available", "-q")
	if err != nil {
		// Fallback rpm-only hosts without dnf cache.
		out, err = r.Run(ctx, "dnf", "list", "-q")
		if err != nil {
			return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMRpmDnf)}, nil
		}
	}
	var pkgs []RepoPackage
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasSuffix(line, "Packages") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		name := fields[0]
		if i := strings.IndexByte(name, '.'); i > 0 {
			name = name[:i] // strip arch suffix name.x86_64
		}
		pkgs = append(pkgs, RepoPackage{Name: name})
	}
	if len(pkgs) == 0 {
		return CatalogResult{Status: CacheEmpty, Hint: CacheHint(PMRpmDnf)}, nil
	}
	return CatalogResult{Packages: pkgs, Status: CacheOK}, nil
}
