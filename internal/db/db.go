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
// v2 (SS-29 / D10): separate packages (repo catalog) from programs (PATH commands).
// v3 (SS-32 / D13): favorites, raw PM category (Section), man_status.
const SchemaVersion = 3

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
	// SS-40: WAL + a few conns so ListPrograms can paint while EnrichMetadata writes.
	// Single-conn mode blocked reads for the whole enrich pass → stuck «Завантаження списку…».
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)
	if _, err := sqlDB.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: WAL: %w", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA busy_timeout=8000`); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: busy_timeout: %w", err)
	}

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
		if err := s.applyV2(); err != nil {
			return err
		}
		if err := s.applyV3(); err != nil {
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
	if ver < 1 {
		if err := s.applyV1(); err != nil {
			return err
		}
		ver = 1
	}
	if ver < 2 {
		if err := s.applyV2(); err != nil {
			return err
		}
		ver = 2
	}
	if ver < 3 {
		if err := s.applyV3(); err != nil {
			return err
		}
		ver = 3
	}
	if ver != SchemaVersion {
		return fmt.Errorf("db: unsupported schema_version %d (want %d)", ver, SchemaVersion)
	}
	if _, err := s.db.Exec(
		`INSERT INTO schema_meta(key, value) VALUES('schema_version', ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		fmt.Sprintf("%d", SchemaVersion),
	); err != nil {
		return fmt.Errorf("db: set schema_version: %w", err)
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

// applyV2 introduces packages (+ FTS) and moves catalog-only rows out of programs (SS-29 / D10).
func (s *Store) applyV2() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS packages (
  id      INTEGER PRIMARY KEY,
  name    TEXT NOT NULL UNIQUE,
  summary TEXT,
  version TEXT
)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS packages_fts USING fts5(
  name,
  summary,
  content='',
  tokenize = 'unicode61'
)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("db: migrate v2: %w\nstmt: %s", err, stmt)
		}
	}
	return s.migrateCatalogRowsToPackages()
}

// applyV3 adds favorites / raw category / man_status (SS-32 / D13).
func (s *Store) applyV3() error {
	stmts := []string{
		`ALTER TABLE programs ADD COLUMN is_favorite INTEGER NOT NULL DEFAULT 0 CHECK (is_favorite IN (0, 1))`,
		`ALTER TABLE programs ADD COLUMN category TEXT`,
		`ALTER TABLE programs ADD COLUMN man_status TEXT NOT NULL DEFAULT 'unknown'`,
		`CREATE INDEX IF NOT EXISTS idx_programs_favorite ON programs(is_favorite)`,
		`CREATE INDEX IF NOT EXISTS idx_programs_category ON programs(category)`,
		`CREATE INDEX IF NOT EXISTS idx_programs_man_status ON programs(man_status)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			// Fresh DB after v1 already has no columns — ALTER is fine.
			// Re-run / duplicate column: ignore "duplicate column name".
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				continue
			}
			return fmt.Errorf("db: migrate v3: %w\nstmt: %s", err, stmt)
		}
	}
	return nil
}

// migrateCatalogRowsToPackages copies uninstalled catalog stubs into packages and
// removes them from programs when they have no snippets (PATH commands stay put).
func (s *Store) migrateCatalogRowsToPackages() error {
	rows, err := s.db.Query(`
SELECT p.id, p.name, COALESCE(p.summary,''), COALESCE(p.version,'')
FROM programs p
WHERE p.is_installed = 0
  AND NOT EXISTS (SELECT 1 FROM user_snippets s WHERE s.program_id = p.id)`)
	if err != nil {
		return fmt.Errorf("db: v2 select catalog stubs: %w", err)
	}
	defer rows.Close()
	type stub struct {
		id      int64
		name    string
		summary string
		version string
	}
	var stubs []stub
	for rows.Next() {
		var st stub
		if err := rows.Scan(&st.id, &st.name, &st.summary, &st.version); err != nil {
			return err
		}
		stubs = append(stubs, st)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, st := range stubs {
		if _, err := s.db.Exec(
			`INSERT INTO packages(name, summary, version) VALUES(?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  summary = CASE
    WHEN excluded.summary IS NOT NULL AND excluded.summary != '' THEN excluded.summary
    ELSE packages.summary
  END,
  version = CASE
    WHEN excluded.version IS NOT NULL AND excluded.version != '' THEN excluded.version
    ELSE packages.version
  END`,
			st.name, nullStr(st.summary), nullStr(st.version),
		); err != nil {
			return fmt.Errorf("db: v2 insert package %q: %w", st.name, err)
		}
		var pkgID int64
		if err := s.db.QueryRow(`SELECT id FROM packages WHERE name = ?`, st.name).Scan(&pkgID); err != nil {
			return err
		}
		if err := s.syncPackageFTS(pkgID); err != nil {
			return err
		}
		if _, err := s.db.Exec(`DELETE FROM programs WHERE id = ?`, st.id); err != nil {
			return fmt.Errorf("db: v2 delete stub program %d: %w", st.id, err)
		}
	}
	// Mirror installed command names into packages so mode «all» can mark them installed.
	if _, err := s.db.Exec(`
INSERT INTO packages(name, summary, version)
SELECT name, summary, version FROM programs WHERE is_installed = 1
ON CONFLICT(name) DO NOTHING`); err != nil {
		return fmt.Errorf("db: v2 mirror installed: %w", err)
	}
	rows2, err := s.db.Query(`SELECT id FROM packages`)
	if err != nil {
		return err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id int64
		if err := rows2.Scan(&id); err != nil {
			return err
		}
		if err := s.syncPackageFTS(id); err != nil {
			return err
		}
	}
	return rows2.Err()
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
