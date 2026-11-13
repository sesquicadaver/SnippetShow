package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ListUserSnippets returns all snippets for a program ordered by created_at, id.
func (s *Store) ListUserSnippets(programID int64) ([]UserSnippet, error) {
	rows, err := s.db.Query(`
SELECT id, program_id, title, command, COALESCE(tags,''), created_at
FROM user_snippets
WHERE program_id = ?
ORDER BY created_at ASC, id ASC`, programID)
	if err != nil {
		return nil, fmt.Errorf("db: list snippets: %w", err)
	}
	defer rows.Close()
	var out []UserSnippet
	for rows.Next() {
		var sn UserSnippet
		if err := rows.Scan(&sn.ID, &sn.ProgramID, &sn.Title, &sn.Command, &sn.Tags, &sn.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

// GetUserSnippet loads one snippet by id.
func (s *Store) GetUserSnippet(id int64) (*UserSnippet, error) {
	row := s.db.QueryRow(`
SELECT id, program_id, title, command, COALESCE(tags,''), created_at
FROM user_snippets WHERE id = ?`, id)
	var sn UserSnippet
	if err := row.Scan(&sn.ID, &sn.ProgramID, &sn.Title, &sn.Command, &sn.Tags, &sn.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &sn, nil
}

// CreateUserSnippet validates and inserts a snippet, refreshing FTS (P3-2, P3-4).
func (s *Store) CreateUserSnippet(programID int64, title, command, tags string) (*UserSnippet, error) {
	title = strings.TrimSpace(title)
	command = strings.TrimSpace(command)
	tags = strings.TrimSpace(tags)
	if title == "" || command == "" {
		return nil, fmt.Errorf("db: title і command обовʼязкові")
	}
	created := time.Now().UTC().Format(time.RFC3339)
	id, err := s.InsertUserSnippet(UserSnippet{
		ProgramID: programID,
		Title:     title,
		Command:   command,
		Tags:      tags,
		CreatedAt: created,
	})
	if err != nil {
		return nil, err
	}
	return &UserSnippet{
		ID: id, ProgramID: programID, Title: title, Command: command, Tags: tags, CreatedAt: created,
	}, nil
}

// UpdateUserSnippet updates title/command/tags and refreshes FTS (P3-3, P3-4).
func (s *Store) UpdateUserSnippet(id int64, title, command, tags string) (*UserSnippet, error) {
	title = strings.TrimSpace(title)
	command = strings.TrimSpace(command)
	tags = strings.TrimSpace(tags)
	if title == "" || command == "" {
		return nil, fmt.Errorf("db: title і command обовʼязкові")
	}
	existing, err := s.GetUserSnippet(id)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec(`
UPDATE user_snippets SET title = ?, command = ?, tags = ? WHERE id = ?`,
		title, command, nullStr(tags), id,
	)
	if err != nil {
		return nil, fmt.Errorf("db: update snippet: %w", err)
	}
	if err := s.syncFTS(existing.ProgramID); err != nil {
		return nil, err
	}
	existing.Title = title
	existing.Command = command
	existing.Tags = tags
	return existing, nil
}

// DeleteUserSnippet removes a snippet and refreshes FTS (P3-3).
func (s *Store) DeleteUserSnippet(id int64) error {
	existing, err := s.GetUserSnippet(id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM user_snippets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete snippet: %w", err)
	}
	return s.syncFTS(existing.ProgramID)
}
