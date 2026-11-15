package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// Package is a distribution repo-catalog row (SS-29 / D10) — not a PATH command.
type Package struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	IsInstalled bool   `json:"isInstalled"` // derived: matching installed program exists
}

// PackageFilter selects packages for navigator mode «all».
type PackageFilter struct {
	Letter string
	Limit  int
}

// UpsertCatalogProgramsBatch upserts repo packages into the packages table (SS-29).
// Name kept for call-site compatibility; writes no longer touch programs rows.
func (s *Store) UpsertCatalogProgramsBatch(entries []CatalogEntry, chunkSize int, onChunk func(done, total int)) (added int, err error) {
	return s.UpsertPackagesBatch(entries, chunkSize, onChunk)
}

// UpsertCatalogProgram upserts one package row (SS-29).
func (s *Store) UpsertCatalogProgram(name, packageName, summary string) (int64, error) {
	_ = packageName
	id, _, changed, err := s.applyPackageEntry(s.db, name, summary, "")
	if err != nil {
		return id, err
	}
	if changed {
		if err := s.syncPackageFTS(id); err != nil {
			return id, err
		}
	}
	return id, nil
}

// UpsertPackagesBatch upserts many packages in transactional chunks with deferred FTS.
func (s *Store) UpsertPackagesBatch(entries []CatalogEntry, chunkSize int, onChunk func(done, total int)) (added int, err error) {
	if chunkSize <= 0 {
		chunkSize = DefaultCatalogChunkSize
	}
	total := len(entries)
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		tx, err := s.db.Begin()
		if err != nil {
			return added, err
		}
		var touched []int64
		s.catalogBatchWriting.Store(true)
		for _, e := range entries[i:end] {
			id, isNew, changed, err := s.applyPackageEntry(tx, e.Name, e.Summary, "")
			if err != nil {
				s.catalogBatchWriting.Store(false)
				_ = tx.Rollback()
				return added, err
			}
			if isNew {
				added++
			}
			if changed {
				touched = append(touched, id)
			}
		}
		if err := tx.Commit(); err != nil {
			s.catalogBatchWriting.Store(false)
			return added, err
		}
		s.catalogBatchWriting.Store(false)
		for _, id := range touched {
			if err := s.syncPackageFTS(id); err != nil {
				return added, err
			}
		}
		if onChunk != nil {
			onChunk(end, total)
		}
	}
	return added, nil
}

func (s *Store) applyPackageEntry(ex sqlExec, name, summary, version string) (id int64, isNew bool, changed bool, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, false, false, fmt.Errorf("db: empty package name")
	}
	existing, err := getPackageByNameExec(ex, name)
	if err == nil {
		need := false
		sum := existing.Summary
		ver := existing.Version
		if sum == "" && strings.TrimSpace(summary) != "" {
			sum = strings.TrimSpace(summary)
			need = true
		}
		if ver == "" && strings.TrimSpace(version) != "" {
			ver = strings.TrimSpace(version)
			need = true
		}
		if !need {
			return existing.ID, false, false, nil
		}
		_, err = ex.Exec(`UPDATE packages SET summary = ?, version = ? WHERE id = ?`, nullStr(sum), nullStr(ver), existing.ID)
		return existing.ID, false, true, err
	}
	if err != sql.ErrNoRows {
		return 0, false, false, err
	}
	_, err = ex.Exec(`INSERT INTO packages(name, summary, version) VALUES(?, ?, ?)`,
		name, nullStr(summary), nullStr(version))
	if err != nil {
		return 0, false, false, err
	}
	id, err = packageIDByNameExec(ex, name)
	return id, true, true, err
}

func getPackageByNameExec(ex sqlExec, name string) (*Package, error) {
	row := ex.QueryRow(`
SELECT id, name, COALESCE(summary,''), COALESCE(version,'')
FROM packages WHERE name = ?`, name)
	var p Package
	if err := row.Scan(&p.ID, &p.Name, &p.Summary, &p.Version); err != nil {
		return nil, err
	}
	return &p, nil
}

func packageIDByNameExec(ex sqlExec, name string) (int64, error) {
	var id int64
	err := ex.QueryRow(`SELECT id FROM packages WHERE name = ?`, name).Scan(&id)
	return id, err
}

// GetPackageByName loads a package and derives install status from programs.
func (s *Store) GetPackageByName(name string) (*Package, error) {
	p, err := getPackageByNameExec(s.db, name)
	if err != nil {
		return nil, err
	}
	p.IsInstalled = s.packageIsInstalled(p.Name)
	return p, nil
}

func (s *Store) packageIsInstalled(name string) bool {
	var n int
	err := s.db.QueryRow(`
SELECT COUNT(*) FROM programs
WHERE is_installed = 1 AND (name = ? OR package_name = ?)`, name, name).Scan(&n)
	return err == nil && n > 0
}

// ListPackagesFiltered returns catalog packages for mode «all» (SS-29).
func (s *Store) ListPackagesFiltered(f PackageFilter) ([]Package, error) {
	q := `
SELECT pkg.id, pkg.name, COALESCE(pkg.summary,''), COALESCE(pkg.version,''),
  CASE WHEN EXISTS (
    SELECT 1 FROM programs p
    WHERE p.is_installed = 1 AND (p.name = pkg.name OR p.package_name = pkg.name)
  ) THEN 1 ELSE 0 END AS is_installed
FROM packages pkg`
	var args []any
	var where []string
	letter := strings.TrimSpace(f.Letter)
	switch {
	case letter == "" || strings.EqualFold(letter, "all"):
	case letter == "0-9":
		where = append(where, `substr(pkg.name, 1, 1) GLOB '[0-9]'`)
	default:
		ch := strings.ToLower(letter)
		if len([]rune(ch)) == 1 {
			where = append(where, `lower(substr(pkg.name, 1, 1)) = ?`)
			args = append(args, ch)
		}
	}
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, ` AND `)
	}
	q += ` ORDER BY pkg.name COLLATE NOCASE`
	if f.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, f.Limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Package
	for rows.Next() {
		var p Package
		var installed int
		if err := rows.Scan(&p.ID, &p.Name, &p.Summary, &p.Version, &installed); err != nil {
			return nil, err
		}
		p.IsInstalled = installed == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// SearchPackagesFiltered runs packages_fts MATCH for mode «all».
func (s *Store) SearchPackagesFiltered(query string, f PackageFilter) ([]Package, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	ftsQuery := buildFTSQuery(q)
	sqlQ := `
SELECT pkg.id, pkg.name, COALESCE(pkg.summary,''), COALESCE(pkg.version,''),
  CASE WHEN EXISTS (
    SELECT 1 FROM programs p
    WHERE p.is_installed = 1 AND (p.name = pkg.name OR p.package_name = pkg.name)
  ) THEN 1 ELSE 0 END AS is_installed,
  COALESCE(bm25(packages_fts), 0) AS rank
FROM packages_fts
JOIN packages pkg ON pkg.id = packages_fts.rowid
WHERE packages_fts MATCH ?`
	args := []any{ftsQuery}
	letter := strings.TrimSpace(f.Letter)
	switch {
	case letter == "" || strings.EqualFold(letter, "all"):
	case letter == "0-9":
		sqlQ += ` AND substr(pkg.name, 1, 1) GLOB '[0-9]'`
	default:
		ch := strings.ToLower(letter)
		if len([]rune(ch)) == 1 {
			sqlQ += ` AND lower(substr(pkg.name, 1, 1)) = ?`
			args = append(args, ch)
		}
	}
	sqlQ += ` ORDER BY rank`
	if f.Limit > 0 {
		sqlQ += ` LIMIT ?`
		args = append(args, f.Limit)
	}
	rows, err := s.db.Query(sqlQ, args...)
	if err != nil {
		return nil, fmt.Errorf("db: packages fts: %w", err)
	}
	defer rows.Close()
	var out []Package
	for rows.Next() {
		var p Package
		var installed int
		var rank float64
		if err := rows.Scan(&p.ID, &p.Name, &p.Summary, &p.Version, &installed, &rank); err != nil {
			return nil, err
		}
		p.IsInstalled = installed == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) syncPackageFTS(packageID int64) error {
	if s.catalogBatchWriting.Load() {
		return fmt.Errorf("db: syncPackageFTS during catalog batch write (H4)")
	}
	s.FTSSyncCount.Add(1)
	var name, summary string
	err := s.db.QueryRow(
		`SELECT name, COALESCE(summary,'') FROM packages WHERE id = ?`, packageID,
	).Scan(&name, &summary)
	if err != nil {
		return err
	}
	_, _ = s.db.Exec(`INSERT INTO packages_fts(packages_fts, rowid) VALUES('delete', ?)`, packageID)
	_, _ = s.db.Exec(`DELETE FROM packages_fts WHERE rowid = ?`, packageID)
	_, err = s.db.Exec(
		`INSERT INTO packages_fts(rowid, name, summary) VALUES(?, ?, ?)`,
		packageID, name, summary,
	)
	if err != nil {
		return fmt.Errorf("db: packages fts insert %d: %w", packageID, err)
	}
	return nil
}
