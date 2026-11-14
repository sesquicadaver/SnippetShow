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

// UpsertProgram inserts or updates a program by unique name.
// On conflict, metadata and install flags are refreshed; man_parsed is reset
// only when the caller sets ResetManParsed (version change / reparse needed).
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
	_, err := ex.Exec(`
INSERT INTO programs(name, package_name, summary, version, is_installed, man_parsed)
VALUES(?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  package_name = excluded.package_name,
  summary = CASE
    WHEN excluded.summary IS NOT NULL AND excluded.summary != '' THEN excluded.summary
    ELSE programs.summary
  END,
  version = excluded.version,
  is_installed = excluded.is_installed,
  man_parsed = CASE WHEN ? THEN 0 ELSE programs.man_parsed END
`,
		p.Name, nullStr(p.PackageName), nullStr(p.Summary), nullStr(p.Version),
		boolToInt(p.IsInstalled), manParsed, boolToInt(resetManParsed),
	)
	if err != nil {
		return 0, fmt.Errorf("db: upsert program %q: %w", p.Name, err)
	}
	return programIDByNameExec(ex, p.Name)
}

// GetProgramByID loads a program by primary key.
func (s *Store) GetProgramByID(id int64) (*Program, error) {
	row := s.db.QueryRow(`
SELECT id, name, COALESCE(package_name,''), COALESCE(summary,''), COALESCE(version,''),
       is_installed, man_parsed
FROM programs WHERE id = ?`, id)
	return scanProgram(row)
}

// GetProgramByName loads a program by unique command name.
func (s *Store) GetProgramByName(name string) (*Program, error) {
	return getProgramByNameExec(s.db, name)
}

func getProgramByNameExec(ex sqlExec, name string) (*Program, error) {
	row := ex.QueryRow(`
SELECT id, name, COALESCE(package_name,''), COALESCE(summary,''), COALESCE(version,''),
       is_installed, man_parsed
FROM programs WHERE name = ?`, name)
	return scanProgram(row)
}

// ListPrograms returns programs, optionally only installed ones.
func (s *Store) ListPrograms(installedOnly bool) ([]Program, error) {
	return s.ListProgramsFiltered(ProgramFilter{InstalledOnly: installedOnly})
}

// ProgramFilter selects programs for the Wiki navigator (Phase 2).
type ProgramFilter struct {
	InstalledOnly bool
	// Letter is "", "all", a single A–Z letter, or "0-9" for digit-leading names.
	Letter string
	Limit  int
}

// ListProgramsFiltered returns programs matching install + alphabet filters.
func (s *Store) ListProgramsFiltered(f ProgramFilter) ([]Program, error) {
	q := `
SELECT id, name, COALESCE(package_name,''), COALESCE(summary,''), COALESCE(version,''),
       is_installed, man_parsed
FROM programs`
	var args []any
	var where []string
	if f.InstalledOnly {
		where = append(where, `is_installed = 1`)
	}
	letter := strings.TrimSpace(f.Letter)
	switch {
	case letter == "" || strings.EqualFold(letter, "all"):
		// no letter filter
	case letter == "0-9":
		where = append(where, `substr(name, 1, 1) GLOB '[0-9]'`)
	default:
		ch := strings.ToLower(letter)
		if len([]rune(ch)) == 1 {
			where = append(where, `lower(substr(name, 1, 1)) = ?`)
			args = append(args, ch)
		}
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
	var installed, parsed int
	if err := row.Scan(&p.ID, &p.Name, &p.PackageName, &p.Summary, &p.Version, &installed, &parsed); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	p.IsInstalled = installed == 1
	p.ManParsed = parsed == 1
	return &p, nil
}

func scanProgramRows(rows *sql.Rows) (*Program, error) {
	var p Program
	var installed, parsed int
	if err := rows.Scan(&p.ID, &p.Name, &p.PackageName, &p.Summary, &p.Version, &installed, &parsed); err != nil {
		return nil, err
	}
	p.IsInstalled = installed == 1
	p.ManParsed = parsed == 1
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
