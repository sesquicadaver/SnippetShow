package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	appapi "snippetshow/internal/app"
	"snippetshow/internal/index"
)

// App is the Wails-bound application root.
// Phase 4: Script Builder, JSON backup, dist polish.
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

// ListSnippets returns user snippets for a program (P3-1).
func (a *App) ListSnippets(programID int64) ([]appapi.SnippetDTO, error) {
	return a.api.ListSnippets(programID)
}

// CreateSnippet inserts a user snippet (P3-2).
func (a *App) CreateSnippet(in appapi.SnippetInput) (*appapi.SnippetDTO, error) {
	return a.api.CreateSnippet(in)
}

// UpdateSnippet edits a user snippet (P3-3).
func (a *App) UpdateSnippet(id int64, in appapi.SnippetInput) (*appapi.SnippetDTO, error) {
	return a.api.UpdateSnippet(id, in)
}

// DeleteSnippet removes a user snippet (P3-3).
func (a *App) DeleteSnippet(id int64) error {
	return a.api.DeleteSnippet(id)
}

// AnalyzeCommand returns safety analysis for the confirm dialog (P3-9).
func (a *App) AnalyzeCommand(command string) appapi.SafetyDTO {
	return a.api.AnalyzeCommand(command)
}

// CopyText copies text via Wails clipboard first, then X11/Wayland CLI (P3-5).
func (a *App) CopyText(text string) (appapi.CopyResponse, error) {
	if strings.TrimSpace(text) == "" {
		return appapi.CopyResponse{}, fmt.Errorf("немає тексту для копіювання")
	}
	if a.ctx != nil {
		if err := runtime.ClipboardSetText(a.ctx, text); err == nil {
			return appapi.CopyResponse{OK: true, Message: "Скопійовано"}, nil
		}
	}
	return a.api.CopyText(text)
}

// RunCommand launches the command in a system terminal with safety gate (P3-8…P3-10).
func (a *App) RunCommand(req appapi.RunRequest) (appapi.RunResponse, error) {
	return a.api.RunCommand(req)
}

// AddToScript appends a line to the in-memory Script Builder draft (P3-11).
func (a *App) AddToScript(command string) (appapi.ScriptDraftDTO, error) {
	return a.api.AddToScript(command)
}

// ScriptDraft returns the current draft snapshot.
func (a *App) ScriptDraft() appapi.ScriptDraftDTO {
	return a.api.ScriptDraft()
}

// MoveScriptLine reorders a draft line (P4-2).
func (a *App) MoveScriptLine(index int, direction string) (appapi.ScriptDraftDTO, error) {
	return a.api.MoveScriptLine(index, direction)
}

// RemoveScriptLine deletes a draft line.
func (a *App) RemoveScriptLine(index int) (appapi.ScriptDraftDTO, error) {
	return a.api.RemoveScriptLine(index)
}

// ClearScriptDraft empties the draft.
func (a *App) ClearScriptDraft() appapi.ScriptDraftDTO {
	return a.api.ClearScriptDraft()
}

// SetScriptBody replaces the draft from the editor body (P4-3).
func (a *App) SetScriptBody(body string) appapi.ScriptDraftDTO {
	return a.api.SetScriptBody(body)
}

// GenerateScript returns the current script text with shebang.
func (a *App) GenerateScript() string {
	return a.api.GenerateScript()
}

// ExportScriptFile writes a .sh with chmod +x (P4-6).
func (a *App) ExportScriptFile(path, body string) error {
	return a.api.ExportScriptFile(path, body)
}

// ExportScriptDialog opens a save dialog and exports the script (P4-6).
func (a *App) ExportScriptDialog(body string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("runtime context недоступний")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Експорт скрипту (.sh)",
		DefaultFilename: "snippetshow-script.sh",
		Filters: []runtime.FileFilter{
			{DisplayName: "Shell scripts", Pattern: "*.sh"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := a.api.ExportScriptFile(path, body); err != nil {
		return "", err
	}
	return path, nil
}

// RunScript runs the script body with safety confirm (P4-5).
func (a *App) RunScript(req appapi.RunRequest) (appapi.RunResponse, error) {
	return a.api.RunScript(req)
}

// ListCustomScripts returns saved scripts (P4-7).
func (a *App) ListCustomScripts() ([]appapi.CustomScriptDTO, error) {
	return a.api.ListCustomScripts()
}

// SaveCustomScript creates or updates a DB script (P4-7).
func (a *App) SaveCustomScript(id int64, in appapi.CustomScriptInput) (*appapi.CustomScriptDTO, error) {
	return a.api.SaveCustomScript(id, in)
}

// LoadCustomScript loads a saved script into the draft.
func (a *App) LoadCustomScript(id int64) (appapi.ScriptDraftDTO, error) {
	return a.api.LoadCustomScript(id)
}

// DeleteCustomScript removes a saved script.
func (a *App) DeleteCustomScript(id int64) error {
	return a.api.DeleteCustomScript(id)
}

// ExportBackup writes JSON backup to path (P4-8).
func (a *App) ExportBackup(path string) error {
	return a.api.ExportBackup(path)
}

// ImportBackup merges JSON backup from path (P4-9).
func (a *App) ImportBackup(path string) (appapi.BackupResultDTO, error) {
	return a.api.ImportBackup(path)
}

// ExportBackupDialog opens a save dialog and exports JSON backup (P4-10).
func (a *App) ExportBackupDialog() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("runtime context недоступний")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Експорт JSON-бекапу",
		DefaultFilename: "snippetshow-backup.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := a.api.ExportBackup(path); err != nil {
		return "", err
	}
	return path, nil
}

// ImportBackupDialog opens an open dialog and imports JSON backup (P4-10).
func (a *App) ImportBackupDialog() (appapi.BackupResultDTO, error) {
	if a.ctx == nil {
		return appapi.BackupResultDTO{}, fmt.Errorf("runtime context недоступний")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Імпорт JSON-бекапу",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})
	if err != nil {
		return appapi.BackupResultDTO{}, err
	}
	if strings.TrimSpace(path) == "" {
		return appapi.BackupResultDTO{}, nil
	}
	return a.api.ImportBackup(path)
}
