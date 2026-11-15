package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// DefaultCatalogChunkSize is how many catalog packages are written per transaction
// before deferred FTS sync (H4). Keeps SQLite locks short while avoiding per-row FTS.
const DefaultCatalogChunkSize = 500

// CatalogEntry is one repo-catalog package for batch upsert (H4).
type CatalogEntry struct {
	Name        string
	PackageName string
	Summary     string
}

type sqlExec interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

const programSelectCols = `id, name, COALESCE(package_name,''), COALESCE(summary,''), COALESCE(version,''),
       is_installed, man_parsed, is_favorite, COALESCE(category,''), COALESCE(man_status,'unknown')`

// DefaultInstalledChunkSize is the transactional chunk for PATH name upserts (SS-36b).
const DefaultInstalledChunkSize = 500

// UpsertInstalledNamesBatch marks PATH command names as installed in chunks with
// deferred FTS (+ light packages mirror). Skips no-op rows that are already installed
// to keep UpdateIndex names-pass fast (SS-36b / D13.1).
func (s *Store) UpsertInstalledNamesBatch(names []string, chunkSize int, onChunk func(done, total int)) (added, updated int, err error) {
	if chunkSize <= 0 {
		chunkSize = DefaultInstalledChunkSize
	}
	total := len(names)
	for i := 0; i < total; i += chunkSize {
		end := i + chunkSize
		if end > total {
			end = total
		}
		tx, err := s.db.Begin()
		if err != nil {
			return added, updated, err
		}
		var touched []int64
		var mirror []CatalogEntry
		for _, name := range names[i:end] {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			existing, gerr := getProgramByNameExec(tx, name)
			if gerr == nil && existing.IsInstalled {
				continue
			}
			isNew := gerr == sql.ErrNoRows
			if gerr != nil && !isNew {
				_ = tx.Rollback()
				return added, updated, gerr
			}
			id, uerr := s.upsertProgramExec(tx, Program{
				Name:        name,
				IsInstalled: true,
				ManStatus:   ManStatusUnknown,
			}, false)
			if uerr != nil {
				_ = tx.Rollback()
				return added, updated, uerr
			}
			touched = append(touched, id)
			mirror = append(mirror, CatalogEntry{Name: name, PackageName: name})
			if isNew {
				added++
			} else {
				updated++
			}
		}
		if err := tx.Commit(); err != nil {
			return added, updated, err
		}
		for _, id := range touched {
			if err := s.syncFTS(id); err != nil {
				return added, updated, err
			}
		}
		if len(mirror) > 0 {
			if _, err := s.UpsertPackagesBatch(mirror, chunkSize, nil); err != nil {
				return added, updated, err
			}
		}
		if onChunk != nil {
			onChunk(end, total)
		}
	}
	return added, updated, nil
}

// UpsertProgram inserts or updates a program by unique name.
// On conflict, metadata and install flags are refreshed; man_parsed is reset
// only when the caller sets ResetManParsed (version change / reparse needed).
// Favorite flag is preserved unless p.IsFavorite is explicitly set via SetFavorite.
func (s *Store) UpsertProgram(p Program, resetManParsed bool) (int64, error) {
	id, err := s.upsertProgramExec(s.db, p, resetManParsed)
	if err != nil {
		return id, err
	}
	if err := s.syncFTS(id); err != nil {
		return id, err
	}
	// SS-29: mirror installed PATH commands into packages so mode «all» can mark them.
	if p.IsInstalled {
		pkg := strings.TrimSpace(p.PackageName)
		if pkg == "" {
			pkg = p.Name
		}
		if _, err := s.UpsertCatalogProgram(p.Name, pkg, p.Summary); err != nil {
			return id, err
		}
	}
	return id, nil
}

func (s *Store) upsertProgramExec(ex sqlExec, p Program, resetManParsed bool) (int64, error) {
	manParsed := boolToInt(p.ManParsed)
	if resetManParsed {
		manParsed = 0
	}
	manStatus := strings.TrimSpace(p.ManStatus)
	if manStatus == "" {
		manStatus = ManStatusUnknown
	}
	_, err := ex.Exec(`
INSERT INTO programs(name, package_name, summary, version, is_installed, man_parsed, is_favorite, category, man_status)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  package_name = CASE
    WHEN excluded.package_name IS NOT NULL AND excluded.package_name != '' THEN excluded.package_name
    ELSE programs.package_name
  END,
  summary = CASE
    WHEN excluded.summary IS NOT NULL AND excluded.summary != '' THEN excluded.summary
    ELSE programs.summary
  END,
  version = CASE
    WHEN excluded.version IS NOT NULL AND excluded.version != '' THEN excluded.version
    ELSE programs.version
  END,
  is_installed = excluded.is_installed,
  man_parsed = CASE WHEN ? THEN 0 ELSE programs.man_parsed END,
  category = CASE
    WHEN excluded.category IS NOT NULL AND excluded.category != '' THEN excluded.category
    ELSE programs.category
  END,
  man_status = CASE
    WHEN excluded.man_status IS NOT NULL AND excluded.man_status != '' AND excluded.man_status != 'unknown'
      THEN excluded.man_status
    ELSE programs.man_status
  END
`,
		p.Name, nullStr(p.PackageName), nullStr(p.Summary), nullStr(p.Version),
		boolToInt(p.IsInstalled), manParsed, boolToInt(p.IsFavorite), nullStr(p.Category), manStatus,
		boolToInt(resetManParsed),
	)
	if err != nil {
		return 0, fmt.Errorf("db: upsert program %q: %w", p.Name, err)
	}
	return programIDByNameExec(ex, p.Name)
}

// GetProgramByID loads a program by primary key.
func (s *Store) GetProgramByID(id int64) (*Program, error) {
	row := s.db.QueryRow(`SELECT `+programSelectCols+` FROM programs WHERE id = ?`, id)
	return scanProgram(row)
}

// GetProgramByName loads a program by unique command name.
func (s *Store) GetProgramByName(name string) (*Program, error) {
	return getProgramByNameExec(s.db, name)
}

func getProgramByNameExec(ex sqlExec, name string) (*Program, error) {
	row := ex.QueryRow(`SELECT `+programSelectCols+` FROM programs WHERE name = ?`, name)
	return scanProgram(row)
}

// ListPrograms returns programs, optionally only installed ones.
func (s *Store) ListPrograms(installedOnly bool) ([]Program, error) {
	return s.ListProgramsFiltered(ProgramFilter{InstalledOnly: installedOnly})
}

// ProgramFilter selects programs for the Wiki navigator (Phase 2 + D13).
type ProgramFilter struct {
	InstalledOnly bool
	FavoritesOnly bool
	// Letter is "", "all", a single A–Z letter, or "0-9" for digit-leading names.
	Letter string
	// Category filters by raw PM Section (exact match); empty = all.
	Category string
	// NoManOnly keeps rows with man_status = absent (D13.10).
	NoManOnly bool
	Limit     int
}

// ListProgramsFiltered returns programs matching install + alphabet filters.
func (s *Store) ListProgramsFiltered(f ProgramFilter) ([]Program, error) {
	q := `SELECT ` + programSelectCols + ` FROM programs`
	var args []any
	var where []string
	if f.FavoritesOnly {
		where = append(where, `is_favorite = 1`)
	} else if f.InstalledOnly {
		where = append(where, `is_installed = 1`)
	}
	letter := strings.TrimSpace(f.Letter)
	switch {
	case letter == "" || strings.EqualFold(letter, "all"):
	case letter == "0-9":
		where = append(where, `substr(name, 1, 1) GLOB '[0-9]'`)
	default:
		ch := strings.ToLower(letter)
		if len([]rune(ch)) == 1 {
			where = append(where, `lower(substr(name, 1, 1)) = ?`)
			args = append(args, ch)
		}
	}
	if cat := strings.TrimSpace(f.Category); cat != "" {
		where = append(where, `category = ?`)
		args = append(args, cat)
	}
	if f.NoManOnly {
		where = append(where, `man_status = ?`)
		args = append(args, ManStatusAbsent)
	}
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, ` AND `)
	}
	q += ` ORDER BY name COLLATE NOCASE`
	if f.Limit > 0 {
		q += ` LIMIT ?`
		args = append(args, f.Limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Program
	for rows.Next() {
		p, err := scanProgramRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListCategories returns distinct raw category values for installed programs.
func (s *Store) ListCategories() ([]string, error) {
	rows, err := s.db.Query(`
SELECT DISTINCT category FROM programs
WHERE is_installed = 1 AND category IS NOT NULL AND category != ''
ORDER BY category COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MarkNotInstalledExcept sets is_installed=0 for names not in keepNames.
// Snippets and man_commands are preserved (ТЗ §2.3).
func (s *Store) MarkNotInstalledExcept(keepNames []string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	keep := map[string]struct{}{}
	for _, n := range keepNames {
		keep[n] = struct{}{}
	}

	rows, err := tx.Query(`SELECT id, name FROM programs WHERE is_installed = 1`)
	if err != nil {
		return 0, err
	}
	type pair struct {
		id   int64
		name string
	}
	var gone []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.id, &p.name); err != nil {
			rows.Close()
			return 0, err
		}
		if _, ok := keep[p.name]; !ok {
			gone = append(gone, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var affected int64
	for _, p := range gone {
		res, err := tx.Exec(`UPDATE programs SET is_installed = 0 WHERE id = ?`, p.id)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		affected += n
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	for _, p := range gone {
		if err := s.syncFTS(p.id); err != nil {
			return affected, err
		}
	}
	return affected, nil
}

// SetFavorite toggles is_favorite (D13). Does not parse man — caller does.
func (s *Store) SetFavorite(name string, on bool) (*Program, error) {
	p, err := s.GetProgramByName(name)
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(`UPDATE programs SET is_favorite = ? WHERE id = ?`, boolToInt(on), p.ID); err != nil {
		return nil, err
	}
	return s.GetProgramByName(name)
}

// SetManStatus updates man_status (+ man_parsed when parsed).
func (s *Store) SetManStatus(programID int64, status string, manParsed bool) error {
	status = strings.TrimSpace(status)
	if status == "" {
		status = ManStatusUnknown
	}
	_, err := s.db.Exec(
		`UPDATE programs SET man_status = ?, man_parsed = ? WHERE id = ?`,
		status, boolToInt(manParsed), programID,
	)
	return err
}

// UpdateProgramMeta patches package metadata / category / man_status without touching favorite.
func (s *Store) UpdateProgramMeta(name, packageName, summary, version, category, manStatus string) error {
	p, err := s.GetProgramByName(name)
	if err != nil {
		return err
	}
	if manStatus == "" {
		manStatus = p.ManStatus
	}
	_, err = s.db.Exec(`
UPDATE programs SET
  package_name = CASE WHEN ? != '' THEN ? ELSE package_name END,
  summary = CASE WHEN ? != '' THEN ? ELSE summary END,
  version = CASE WHEN ? != '' THEN ? ELSE version END,
  category = CASE WHEN ? != '' THEN ? ELSE category END,
  man_status = CASE WHEN ? != '' THEN ? ELSE man_status END
WHERE id = ?`,
		packageName, packageName,
		summary, summary,
		version, version,
		category, category,
		manStatus, manStatus,
		p.ID,
	)
	if err != nil {
		return err
	}
	return s.syncFTS(p.ID)
}

// SetManParsed updates the man_parsed flag for a program.
func (s *Store) SetManParsed(programID int64, parsed bool) error {
	_, err := s.db.Exec(`UPDATE programs SET man_parsed = ? WHERE id = ?`, boolToInt(parsed), programID)
	return err
}

// InsertUserSnippet inserts a snippet (used by index tests / Phase 3 prep) and refreshes FTS.
func (s *Store) InsertUserSnippet(sn UserSnippet) (int64, error) {
	res, err := s.db.Exec(`
INSERT INTO user_snippets(program_id, title, command, tags, created_at)
VALUES(?, ?, ?, ?, ?)`,
		sn.ProgramID, sn.Title, sn.Command, nullStr(sn.Tags), sn.CreatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.syncFTS(sn.ProgramID); err != nil {
		return id, err
	}
	return id, nil
}

// CountUserSnippets returns snippet count for a program (retention checks).
func (s *Store) CountUserSnippets(programID int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM user_snippets WHERE program_id = ?`, programID).Scan(&n)
	return n, err
}

func programIDByNameExec(ex sqlExec, name string) (int64, error) {
	var id int64
	err := ex.QueryRow(`SELECT id FROM programs WHERE name = ?`, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("db: program id for %q: %w", name, err)
	}
	return id, nil
}

func scanProgram(row *sql.Row) (*Program, error) {
	var p Program
	var installed, parsed, favorite int
	if err := row.Scan(
		&p.ID, &p.Name, &p.PackageName, &p.Summary, &p.Version,
		&installed, &parsed, &favorite, &p.Category, &p.ManStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	p.IsInstalled = installed == 1
	p.ManParsed = parsed == 1
	p.IsFavorite = favorite == 1
	if p.ManStatus == "" {
		p.ManStatus = ManStatusUnknown
	}
	return &p, nil
}

func scanProgramRows(rows *sql.Rows) (*Program, error) {
	var p Program
	var installed, parsed, favorite int
	if err := rows.Scan(
		&p.ID, &p.Name, &p.PackageName, &p.Summary, &p.Version,
		&installed, &parsed, &favorite, &p.Category, &p.ManStatus,
	); err != nil {
		return nil, err
	}
	p.IsInstalled = installed == 1
	p.ManParsed = parsed == 1
	p.IsFavorite = favorite == 1
	if p.ManStatus == "" {
		p.ManStatus = ManStatusUnknown
	}
	return &p, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
