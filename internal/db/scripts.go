package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ListCustomScripts returns all saved scripts ordered by updated_at DESC, id DESC.
func (s *Store) ListCustomScripts() ([]CustomScript, error) {
	rows, err := s.db.Query(`
SELECT id, name, script_body, updated_at
FROM custom_scripts
ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("db: list custom_scripts: %w", err)
	}
	defer rows.Close()
	var out []CustomScript
	for rows.Next() {
		var cs CustomScript
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.ScriptBody, &cs.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

// GetCustomScript loads one script by id.
func (s *Store) GetCustomScript(id int64) (*CustomScript, error) {
	row := s.db.QueryRow(`
SELECT id, name, script_body, updated_at FROM custom_scripts WHERE id = ?`, id)
	var cs CustomScript
	if err := row.Scan(&cs.ID, &cs.Name, &cs.ScriptBody, &cs.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &cs, nil
}

// CreateCustomScript inserts a named script body with updated_at (P4-7).
func (s *Store) CreateCustomScript(name, body string) (*CustomScript, error) {
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	if name == "" || body == "" {
		return nil, fmt.Errorf("db: name і script_body обовʼязкові")
	}
	updated := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`
INSERT INTO custom_scripts(name, script_body, updated_at) VALUES(?, ?, ?)`,
		name, body, updated,
	)
	if err != nil {
		return nil, fmt.Errorf("db: insert custom_script: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &CustomScript{ID: id, Name: name, ScriptBody: body, UpdatedAt: updated}, nil
}

// UpdateCustomScript updates name/body and bumps updated_at.
func (s *Store) UpdateCustomScript(id int64, name, body string) (*CustomScript, error) {
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	if name == "" || body == "" {
		return nil, fmt.Errorf("db: name і script_body обовʼязкові")
	}
	if _, err := s.GetCustomScript(id); err != nil {
		return nil, err
	}
	updated := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`
UPDATE custom_scripts SET name = ?, script_body = ?, updated_at = ? WHERE id = ?`,
		name, body, updated, id,
	)
	if err != nil {
		return nil, fmt.Errorf("db: update custom_script: %w", err)
	}
	return &CustomScript{ID: id, Name: name, ScriptBody: body, UpdatedAt: updated}, nil
}

// DeleteCustomScript removes a saved script by id.
func (s *Store) DeleteCustomScript(id int64) error {
	res, err := s.db.Exec(`DELETE FROM custom_scripts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete custom_script: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListAllUserSnippets returns every snippet with owning program name (backup export).
func (s *Store) ListAllUserSnippets() ([]UserSnippetExport, error) {
	rows, err := s.db.Query(`
SELECT s.id, s.program_id, p.name, s.title, s.command, COALESCE(s.tags,''), s.created_at
FROM user_snippets s
JOIN programs p ON p.id = s.program_id
ORDER BY s.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("db: list all snippets: %w", err)
	}
	defer rows.Close()
	var out []UserSnippetExport
	for rows.Next() {
		var sn UserSnippetExport
		if err := rows.Scan(&sn.ID, &sn.ProgramID, &sn.ProgramName, &sn.Title, &sn.Command, &sn.Tags, &sn.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

// UserSnippetExport is a snippet row annotated with program name for JSON backup.
type UserSnippetExport struct {
	ID          int64  `json:"id"`
	ProgramID   int64  `json:"programId"`
	ProgramName string `json:"programName"`
	Title       string `json:"title"`
	Command     string `json:"command"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"createdAt"`
}

// FindSnippetDuplicate reports whether an identical snippet already exists.
func (s *Store) FindSnippetDuplicate(programID int64, title, command string) (bool, error) {
	var n int
	err := s.db.QueryRow(`
SELECT COUNT(*) FROM user_snippets
WHERE program_id = ? AND title = ? AND command = ?`,
		programID, title, command,
	).Scan(&n)
	return n > 0, err
}

// FindScriptDuplicate reports whether an identical script (name+body) already exists.
func (s *Store) FindScriptDuplicate(name, body string) (bool, error) {
	var n int
	err := s.db.QueryRow(`
SELECT COUNT(*) FROM custom_scripts WHERE name = ? AND script_body = ?`,
		name, body,
	).Scan(&n)
	return n > 0, err
}
