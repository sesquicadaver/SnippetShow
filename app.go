package main

import (
	"context"

	appapi "snippetshow/internal/app"
	"snippetshow/internal/index"
)

// App is the Wails-bound application root.
// Phase 1: Core & DB bindings (ScanInstalled, UpdateIndex, GetProgram, Search).
type App struct {
	ctx context.Context
	api *appapi.API
}

// NewApp constructs the desktop application shell.
func NewApp() *App {
	return &App{api: appapi.New()}
}

// startup stores the runtime context for later Wails runtime calls.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AppInfo returns brand metadata for the shell UI.
func (a *App) AppInfo() appapi.Info {
	return a.api.Info()
}

// PhaseStatus reports which implementation phase the binary represents.
func (a *App) PhaseStatus() string {
	return a.api.PhaseStatus()
}

// ScanInstalled performs a fast installed-binaries scan into SQLite.
func (a *App) ScanInstalled() (index.UpdateResult, error) {
	return a.api.ScanInstalled()
}

// UpdateIndex runs incremental index refresh.
func (a *App) UpdateIndex() (index.UpdateResult, error) {
	return a.api.UpdateIndex()
}

// GetProgram returns program detail; parseMan triggers on-demand man parse.
func (a *App) GetProgram(name string, parseMan bool) (*appapi.ProgramDetail, error) {
	return a.api.GetProgram(name, parseMan)
}

// Search runs FTS5 search over programs / flags / snippets.
func (a *App) Search(query string) ([]appapi.SearchResultDTO, error) {
	return a.api.Search(query)
}

// ListInstalled returns installed programs from the local DB.
func (a *App) ListInstalled() ([]appapi.ProgramDTO, error) {
	return a.api.ListInstalled()
}
