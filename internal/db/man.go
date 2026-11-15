package db

import (
	"fmt"
)

// ReplaceManCommands replaces all man_commands for a program and marks man_parsed=1.
// FTS flags column is rebuilt afterwards.
func (s *Store) ReplaceManCommands(programID int64, cmds []ManCommand) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM man_commands WHERE program_id = ?`, programID); err != nil {
		return fmt.Errorf("db: clear man_commands: %w", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO man_commands(program_id, flag, description) VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, c := range cmds {
		if _, err := stmt.Exec(programID, c.Flag, nullStr(c.Description)); err != nil {
			return fmt.Errorf("db: insert man_command: %w", err)
		}
	}
	if _, err := tx.Exec(
		`UPDATE programs SET man_parsed = 1, man_status = ? WHERE id = ?`,
		ManStatusParsed, programID,
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.syncFTS(programID)
}

// ClearManCommands deletes man rows and resets parse flags (D13.5 unfavorite purge).
func (s *Store) ClearManCommands(programID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM man_commands WHERE program_id = ?`, programID); err != nil {
		return fmt.Errorf("db: clear man_commands: %w", err)
	}
	if _, err := tx.Exec(
		`UPDATE programs SET man_parsed = 0, man_status = ? WHERE id = ?`,
		ManStatusUnknown, programID,
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.syncFTS(programID)
}

// ListManCommands returns man flag rows for a program.
func (s *Store) ListManCommands(programID int64) ([]ManCommand, error) {
	rows, err := s.db.Query(`
SELECT id, program_id, flag, COALESCE(description,'')
FROM man_commands WHERE program_id = ? ORDER BY id`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ManCommand
	for rows.Next() {
		var c ManCommand
		if err := rows.Scan(&c.ID, &c.ProgramID, &c.Flag, &c.Description); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
