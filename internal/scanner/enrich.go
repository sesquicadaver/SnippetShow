package scanner

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
)

// MetaEnrichment is bulk PM metadata for one executable basename (D13.9).
type MetaEnrichment struct {
	Name        string
	PackageName string
	Version     string
	Summary     string
	Category    string // raw Section / Group
	HasMan      bool
}

// EnrichInstalled builds package/section/man-presence maps without per-binary -S (D13).
func (s *Scanner) EnrichInstalled(ctx context.Context) ([]MetaEnrichment, error) {
	names, paths, err := s.scanExecutables()
	if err != nil {
		return nil, err
	}
	pm, pmErr := s.DetectPM()
	pkgMeta := map[string]pkgMeta{} // package name -> meta
	owner := map[string]string{}    // abs path -> package

	if pmErr == nil {
		switch pm {
		case PMDpkg:
			pkgMeta, _ = s.loadDpkgPackageMeta(ctx)
			owner, _ = loadDpkgPathOwners()
		case PMPacman:
			pkgMeta, owner, _ = s.loadPacmanMeta(ctx)
		case PMRpmDnf:
			// Best-effort: leave category empty; man check still runs.
		}
	}

	out := make([]MetaEnrichment, 0, len(names))
	for _, name := range names {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		e := MetaEnrichment{Name: name, HasMan: ManPageExists(name)}
		if path := paths[name]; path != "" {
			if pkg := owner[path]; pkg != "" {
				e.PackageName = pkg
				if m, ok := pkgMeta[pkg]; ok {
					e.Version = m.Version
					e.Summary = m.Summary
					e.Category = m.Section
				}
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// ManPageExists reports whether a man page file is present for cmd (no subprocess).
func ManPageExists(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	roots := []string{"/usr/share/man", "/usr/local/share/man"}
	sections := []string{"1", "8", "6", "5", "7"}
	for _, root := range roots {
		for _, loc := range []string{"", "en", "uk"} {
			base := root
			if loc != "" {
				base = filepath.Join(root, loc)
			}
			for _, num := range sections {
				dir := filepath.Join(base, "man"+num)
				matches, _ := filepath.Glob(filepath.Join(dir, cmd+"."+num+"*"))
				if len(matches) > 0 {
					return true
				}
			}
		}
	}
	return false
}

func (s *Scanner) loadDpkgPackageMeta(ctx context.Context) (map[string]pkgMeta, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	out, err := s.runner().Run(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\t${Section}\t${binary:Package}\t${Description}\n")
	if err != nil {
		return nil, err
	}
	meta := map[string]pkgMeta{}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		parts := strings.Split(sc.Text(), "\t")
		if len(parts) < 1 || strings.TrimSpace(parts[0]) == "" {
			continue
		}
		pkg := strings.TrimSpace(parts[0])
		m := pkgMeta{PackageName: pkg}
		if len(parts) >= 2 {
			m.Version = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			m.Section = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 5 {
			m.Summary = shortDesc(parts[4])
		} else if len(parts) >= 4 {
			m.Summary = shortDesc(parts[3])
		}
		meta[pkg] = m
	}
	return meta, sc.Err()
}

func loadDpkgPathOwners() (map[string]string, error) {
	const dir = "/var/lib/dpkg/info"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	owner := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".list") {
			continue
		}
		pkg := strings.TrimSuffix(name, ".list")
		if i := strings.IndexByte(pkg, ':'); i > 0 {
			pkg = pkg[:i]
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			path := sc.Text()
			if path == "" || strings.HasSuffix(path, "/") {
				continue
			}
			owner[path] = pkg
		}
		_ = f.Close()
	}
	return owner, nil
}

func (s *Scanner) loadPacmanMeta(ctx context.Context) (map[string]pkgMeta, map[string]string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()
	ql, err := r.Run(ctx, "pacman", "-Ql")
	if err != nil {
		return nil, nil, err
	}
	owner := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(ql))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		owner[fields[1]] = fields[0]
	}
	// Groups as raw category: pacman -Qg → "group pkg"
	groups := map[string]string{}
	if gOut, err := r.Run(ctx, "pacman", "-Qg"); err == nil {
		gsc := bufio.NewScanner(strings.NewReader(gOut))
		for gsc.Scan() {
			fields := strings.Fields(gsc.Text())
			if len(fields) >= 2 {
				// Keep first group seen as category.
				if _, ok := groups[fields[1]]; !ok {
					groups[fields[1]] = fields[0]
				}
			}
		}
	}
	meta := map[string]pkgMeta{}
	if qOut, err := r.Run(ctx, "pacman", "-Q"); err == nil {
		qsc := bufio.NewScanner(strings.NewReader(qOut))
		for qsc.Scan() {
			fields := strings.Fields(qsc.Text())
			if len(fields) >= 1 {
				pkg := fields[0]
				m := pkgMeta{PackageName: pkg, Section: groups[pkg]}
				if len(fields) >= 2 {
					m.Version = fields[1]
				}
				meta[pkg] = m
			}
		}
	}
	return meta, owner, nil
}
