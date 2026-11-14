package scanner

import (
	"context"
	"strings"
)

func (s *Scanner) lookupDpkg(ctx context.Context, binPath, binName string) (*pkgMeta, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()

	pkgOut, err := r.Run(ctx, "dpkg-query", "-S", binPath)
	if err != nil {
		// Fallback: query by basename pattern.
		pkgOut, err = r.Run(ctx, "dpkg-query", "-S", "*/"+binName)
		if err != nil {
			return nil, err
		}
	}
	pkg := parseDpkgOwner(pkgOut)
	if pkg == "" {
		return nil, ErrUnsupportedPM
	}
	metaOut, err := r.Run(ctx, "dpkg-query", "-W", "-f=${Package}\t${Version}\t${binary:Package}\t${Description}\n", pkg)
	if err != nil {
		return &pkgMeta{PackageName: pkg}, nil
	}
	line := firstLine(metaOut)
	parts := strings.Split(line, "\t")
	m := &pkgMeta{PackageName: pkg}
	if len(parts) >= 2 {
		m.PackageName = firstNonEmpty(parts[0], pkg)
		m.Version = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 4 {
		m.Summary = shortDesc(parts[3])
	} else if len(parts) >= 3 {
		m.Summary = shortDesc(parts[2])
	}
	// Per-binary path does not load Section; bulk enrich fills category (D13.9).
	return m, nil
}

func parseDpkgOwner(out string) string {
	line := firstLine(out)
	if i := strings.IndexByte(line, ':'); i > 0 {
		return strings.TrimSpace(line[:i])
	}
	return ""
}

func (s *Scanner) lookupPacman(ctx context.Context, binPath, binName string) (*pkgMeta, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()
	ownerOut, err := r.Run(ctx, "pacman", "-Qo", binPath)
	if err != nil {
		ownerOut, err = r.Run(ctx, "pacman", "-Qo", binName)
		if err != nil {
			return nil, err
		}
	}
	// " /usr/bin/ls is owned by coreutils 9.4-2"
	pkg, ver := parsePacmanOwner(ownerOut)
	if pkg == "" {
		return nil, ErrUnsupportedPM
	}
	m := &pkgMeta{PackageName: pkg, Version: ver}
	info, err := r.Run(ctx, "pacman", "-Qi", pkg)
	if err == nil {
		m.Summary = pacmanField(info, "Description")
		if m.Version == "" {
			m.Version = pacmanField(info, "Version")
		}
	}
	return m, nil
}

func parsePacmanOwner(out string) (pkg, ver string) {
	line := firstLine(out)
	const marker = " is owned by "
	i := strings.Index(line, marker)
	if i < 0 {
		return "", ""
	}
	rest := strings.TrimSpace(line[i+len(marker):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", ""
	}
	pkg = fields[0]
	if len(fields) > 1 {
		ver = fields[1]
	}
	return pkg, ver
}

func pacmanField(info, key string) string {
	prefix := key + " "
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key) {
			// "Description     : GNU core utilities"
			if idx := strings.Index(line, ":"); idx >= 0 {
				return strings.TrimSpace(line[idx+1:])
			}
			_ = prefix
		}
	}
	return ""
}

func (s *Scanner) lookupRpm(ctx context.Context, binPath, binName string) (*pkgMeta, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()
	r := s.runner()
	ownerOut, err := r.Run(ctx, "rpm", "-qf", binPath)
	if err != nil {
		ownerOut, err = r.Run(ctx, "rpm", "-qf", binName)
		if err != nil {
			return nil, err
		}
	}
	pkg := strings.TrimSpace(firstLine(ownerOut))
	if pkg == "" {
		return nil, ErrUnsupportedPM
	}
	m := &pkgMeta{PackageName: pkg}
	qi, err := r.Run(ctx, "rpm", "-q", "--qf", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{SUMMARY}\n", pkg)
	if err == nil {
		parts := strings.Split(firstLine(qi), "\t")
		if len(parts) >= 1 {
			m.PackageName = parts[0]
		}
		if len(parts) >= 2 {
			m.Version = parts[1]
		}
		if len(parts) >= 3 {
			m.Summary = parts[2]
		}
	}
	return m, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func shortDesc(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
