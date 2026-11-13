// Package db owns SQLite schema, migrations, FTS5 and CRUD (Phase 1 / ТЗ §3).
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	_ "modernc.org/sqlite"
)

// DefaultPath is the canonical database location from TOR / D1.
const DefaultPath = "~/.local/share/linux-cmd-wiki/wiki.db"

// SchemaVersion is the current migration revision applied by Open.
const SchemaVersion = 1

// Store holds an open SQLite connection with schema ready for use.
type Store struct {
	db   *sql.DB
	path string
	// FTSSyncCount counts syncFTS invocations (H4 tests / diagnostics).
	FTSSyncCount atomic.Int64
	// catalogBatchWriting is set while catalog chunk rows are inserted (H4 invariant).
	catalogBatchWriting atomic.Bool
}

// Open opens (or creates) the wiki database at path, creating parent dirs and
// applying the v1 schema (programs, man_commands, user_snippets, custom_scripts, FTS5).
// Empty path or DefaultPath expands to the D1 location under the user home.
func Open(path string) (*Store, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return nil, fmt.Errorf("db: create dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", resolved)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)

	s := &Store{db: sqlDB, path: resolved}
	if err := s.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return s, nil
}

// ResolvePath expands "~" / DefaultPath to an absolute filesystem path.
func ResolvePath(path string) (string, error) {
	if path == "" || path == DefaultPath {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("db: home dir: %w", err)
		}
		return filepath.Join(home, ".local", "share", "linux-cmd-wiki", "wiki.db"), nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("db: home dir: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return filepath.Clean(path), nil
}

// Path returns the resolved on-disk database path.
func (s *Store) Path() string {
	return s.path
}

// DB exposes the underlying *sql.DB for packages that need transactions.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// migrate applies schema v1 idempotently and records schema_version.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("db: pragma foreign_keys: %w", err)
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS schema_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("db: schema_meta: %w", err)
	}

	var ver int
	err := s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key = 'schema_version'`).Scan(&ver)
	if err == sql.ErrNoRows {
		if err := s.applyV1(); err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO schema_meta(key, value) VALUES('schema_version', ?)`,
			fmt.Sprintf("%d", SchemaVersion),
		); err != nil {
			return fmt.Errorf("db: set schema_version: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("db: read schema_version: %w", err)
	}
	if ver < SchemaVersion {
		if err := s.applyV1(); err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`UPDATE schema_meta SET value = ? WHERE key = 'schema_version'`,
			fmt.Sprintf("%d", SchemaVersion),
		); err != nil {
			return fmt.Errorf("db: bump schema_version: %w", err)
		}
	}
	return nil
}

// applyV1 creates TOR §3 tables and contentless FTS5 index.
func (s *Store) applyV1() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS programs (
  id           INTEGER PRIMARY KEY,
  name         TEXT    NOT NULL UNIQUE,
  package_name TEXT,
  summary      TEXT,
  version      TEXT,
  is_installed INTEGER NOT NULL DEFAULT 0 CHECK (is_installed IN (0, 1)),
  man_parsed   INTEGER NOT NULL DEFAULT 0 CHECK (man_parsed IN (0, 1))
)`,
		`CREATE TABLE IF NOT EXISTS man_commands (
  id          INTEGER PRIMARY KEY,
  program_id  INTEGER NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
  flag        TEXT    NOT NULL,
  description TEXT
)`,
		`CREATE INDEX IF NOT EXISTS idx_man_commands_program ON man_commands(program_id)`,
		`CREATE TABLE IF NOT EXISTS user_snippets (
  id         INTEGER PRIMARY KEY,
  program_id INTEGER NOT NULL REFERENCES programs(id) ON DELETE CASCADE,
  title      TEXT    NOT NULL,
  command    TEXT    NOT NULL,
  tags       TEXT,
  created_at TEXT    NOT NULL
)`,
		`CREATE INDEX IF NOT EXISTS idx_user_snippets_program ON user_snippets(program_id)`,
		`CREATE TABLE IF NOT EXISTS custom_scripts (
  id          INTEGER PRIMARY KEY,
  name        TEXT NOT NULL,
  script_body TEXT NOT NULL,
  updated_at  TEXT NOT NULL
)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS programs_fts USING fts5(
  name,
  summary,
  flags,
  snippets,
  content='',
  tokenize = 'unicode61'
)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("db: migrate v1: %w\nstmt: %s", err, stmt)
		}
	}
	return nil
}

// HasTable reports whether a user table exists (used by schema tests).
func (s *Store) HasTable(name string) (bool, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`,
		name,
	).Scan(&n)
	return n > 0, err
}

// ColumnNames returns ordered column names for a table (schema tests).
func (s *Store) ColumnNames(table string) ([]string, error) {
	rows, err := s.db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, quoteIdent(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
