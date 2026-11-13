package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	appapi "snippetshow/internal/app"
	"snippetshow/internal/index"
)

// App is the Wails-bound application root.
// Phase 2: Read-Only Wiki UI bindings.
type App struct {
	ctx context.Context
	api *appapi.API
}

// NewApp constructs the desktop application shell.
func NewApp() *App {
	return &App{api: appapi.New()}
}

// startup stores the runtime context and wires index progress → EventsEmit.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.api.SetProgressHandler(func(p appapi.ProgressEvent) {
		runtime.EventsEmit(ctx, "index:progress", p)
	})
}

// AppInfo returns brand metadata for the shell UI.
func (a *App) AppInfo() appapi.Info {
	return a.api.Info()
}

// PhaseStatus reports which implementation phase the binary represents.
func (a *App) PhaseStatus() string {
	return a.api.PhaseStatus()
}

// EnsureIndexed runs a first-run scan when the local DB is empty.
func (a *App) EnsureIndexed() (index.UpdateResult, error) {
	return a.api.EnsureIndexed()
}

// ScanInstalled performs a fast installed-binaries scan into SQLite.
func (a *App) ScanInstalled() (index.UpdateResult, error) {
	return a.api.ScanInstalled()
}

// UpdateIndex runs incremental index refresh (emits index:progress).
func (a *App) UpdateIndex() (index.UpdateResult, error) {
	return a.api.UpdateIndex()
}

// SyncRepoCatalog loads the distribution package cache into programs (D5).
func (a *App) SyncRepoCatalog() (index.CatalogSyncResult, error) {
	return a.api.SyncRepoCatalog()
}

// GetProgram returns program detail; parseMan triggers on-demand man parse.
func (a *App) GetProgram(name string, parseMan bool) (*appapi.ProgramDetail, error) {
	return a.api.GetProgram(name, parseMan)
}

// GetRawMan returns cleaned man text for an installed program.
func (a *App) GetRawMan(name string) (string, error) {
	return a.api.GetRawMan(name)
}

// Search runs FTS5 search over programs / flags / snippets.
func (a *App) Search(query string) ([]appapi.SearchResultDTO, error) {
	return a.api.Search(query)
}

// ListInstalled returns installed programs from the local DB.
func (a *App) ListInstalled() ([]appapi.ProgramDTO, error) {
	return a.api.ListInstalled()
}

// ListPrograms returns navigator rows for mode/letter/query filters.
func (a *App) ListPrograms(q appapi.ListQuery) ([]appapi.ProgramDTO, error) {
	return a.api.ListPrograms(q)
}

// SearchPrograms runs filtered FTS / list for the Wiki toolbar.
func (a *App) SearchPrograms(q appapi.ListQuery) ([]appapi.SearchResultDTO, error) {
	return a.api.SearchPrograms(q)
}

// DetectPackageManager returns dpkg / pacman / rpm+dnf.
func (a *App) DetectPackageManager() (string, error) {
	return a.api.DetectPackageManager()
}

// InstallCommandFor returns a suggested install command for a package name.
func (a *App) InstallCommandFor(pkg string) (string, error) {
	return a.api.InstallCommandFor(pkg)
}
