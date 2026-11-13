package db

import (
	"fmt"
	"strings"
)

// syncFTS rebuilds the contentless FTS5 row for one program (name, summary, flags, snippets).
func (s *Store) syncFTS(programID int64) error {
	p, err := s.GetProgramByID(programID)
	if err != nil {
		return err
	}

	flags, err := s.collectFlags(programID)
	if err != nil {
		return err
	}
	snippets, err := s.collectSnippets(programID)
	if err != nil {
		return err
	}

	// Contentless FTS5 delete (SQLite 'delete' command) then re-insert under same rowid.
	if _, err := s.db.Exec(
		`INSERT INTO programs_fts(programs_fts, rowid) VALUES('delete', ?)`,
		programID,
	); err != nil {
		// First sync: row may not exist yet — ignore missing-row style errors by continuing.
		_ = err
	}
	// Always attempt a hard delete for engines that support contentless_delete; ignore failure.
	_, _ = s.db.Exec(`DELETE FROM programs_fts WHERE rowid = ?`, programID)

	_, err = s.db.Exec(
		`INSERT INTO programs_fts(rowid, name, summary, flags, snippets) VALUES(?, ?, ?, ?, ?)`,
		programID, p.Name, p.Summary, flags, snippets,
	)
	if err != nil {
		return fmt.Errorf("db: fts insert program %d: %w", programID, err)
	}
	return nil
}

func (s *Store) collectFlags(programID int64) (string, error) {
	rows, err := s.db.Query(`SELECT flag, COALESCE(description,'') FROM man_commands WHERE program_id = ?`, programID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var flag, desc string
		if err := rows.Scan(&flag, &desc); err != nil {
			return "", err
		}
		parts = append(parts, flag, desc)
	}
	return strings.Join(parts, " "), rows.Err()
}

func (s *Store) collectSnippets(programID int64) (string, error) {
	rows, err := s.db.Query(
		`SELECT title, command, COALESCE(tags,'') FROM user_snippets WHERE program_id = ?`,
		programID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var title, cmd, tags string
		if err := rows.Scan(&title, &cmd, &tags); err != nil {
			return "", err
		}
		parts = append(parts, title, cmd, tags)
	}
	return strings.Join(parts, " "), rows.Err()
}

// Search runs an FTS5 query and returns matching programs ordered by rank.
// Empty query returns an empty slice (caller may fall back to ListPrograms).
func (s *Store) Search(query string) ([]SearchHit, error) {
	return s.SearchFiltered(query, ProgramFilter{})
}

// SearchFiltered runs FTS5 with optional installed-only and alphabet filters (Phase 2).
func (s *Store) SearchFiltered(query string, f ProgramFilter) ([]SearchHit, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	ftsQuery := buildFTSQuery(q)
	sqlQ := `
SELECT p.id, p.name, COALESCE(p.summary,''), COALESCE(bm25(programs_fts), 0) AS rank
FROM programs_fts
JOIN programs p ON p.id = programs_fts.rowid
WHERE programs_fts MATCH ?`
	args := []any{ftsQuery}
	if f.InstalledOnly {
		sqlQ += ` AND p.is_installed = 1`
	}
	letter := strings.TrimSpace(f.Letter)
	switch {
	case letter == "" || strings.EqualFold(letter, "all"):
	case letter == "0-9":
		sqlQ += ` AND substr(p.name, 1, 1) GLOB '[0-9]'`
	default:
		ch := strings.ToLower(letter)
		if len([]rune(ch)) == 1 {
			sqlQ += ` AND lower(substr(p.name, 1, 1)) = ?`
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
		return nil, fmt.Errorf("db: fts search: %w", err)
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.ProgramID, &h.Name, &h.Summary, &h.Rank); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// buildFTSQuery turns a user string into a safe FTS5 MATCH expression (prefix tokens).
func buildFTSQuery(q string) string {
	fields := strings.Fields(q)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		safe := strings.ReplaceAll(f, `"`, "")
		if safe == "" {
			continue
		}
		parts = append(parts, `"`+safe+`"*`)
	}
	if len(parts) == 0 {
		return `""`
	}
	return strings.Join(parts, " ")
}
