// Package scanner detects the package manager and inventories binaries / repo catalogs.
// Phase 0: skeleton only — adapters land in Phase 1 (P1-5…P1-8). Policy D5 applies to empty cache.

package scanner

import "errors"

// ErrNotImplemented marks Phase 0 stubs. Documented in docs/TODO_FIXME.md.
var ErrNotImplemented = errors.New("scanner: not implemented (phase 1)")

// PackageManager is a detected distribution package manager.
type PackageManager string

const (
	PMUnknown PackageManager = ""
	PMDpkg    PackageManager = "dpkg"
	PMPacman  PackageManager = "pacman"
	PMRpmDnf  PackageManager = "rpm+dnf"
)

// DetectPM identifies the host package manager. Phase 0 stub.
func DetectPM() (PackageManager, error) {
	return PMUnknown, ErrNotImplemented
}

// ScanInstalled lists installed executables. Phase 0 stub.
func ScanInstalled() ([]string, error) {
	return nil, ErrNotImplemented
}
