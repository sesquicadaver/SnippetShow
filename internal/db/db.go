// Package db owns SQLite schema, migrations, FTS5 and CRUD.
// Phase 0: skeleton only — Open/schema land in Phase 1 (P1-1…P1-4).

package db

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("db: not implemented (phase 1)")

// DefaultPath is the canonical database location from TOR / D1.
const DefaultPath = "~/.local/share/linux-cmd-wiki/wiki.db"

// Store will hold an open SQLite connection after Phase 1.
type Store struct{}

// Open opens (or creates) the wiki database. Phase 0 stub.
func Open(path string) (*Store, error) {
	_ = path
	return nil, ErrNotImplemented
}
