package main

import (
	"context"

	appapi "snippetshow/internal/app"
)

// App is the Wails-bound application root.
// Phase 0: thin facade; Core logic arrives in Phase 1+.
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

// AppInfo returns brand metadata for the Phase 0 shell UI.
func (a *App) AppInfo() appapi.Info {
	return a.api.Info()
}

// PhaseStatus reports which implementation phase the binary represents.
func (a *App) PhaseStatus() string {
	return a.api.PhaseStatus()
}
