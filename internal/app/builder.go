package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"snippetshow/internal/backup"
	"snippetshow/internal/db"
	"snippetshow/internal/safety"
	"snippetshow/internal/scriptbuilder"
)

// ScriptDraftDTO is a view of the in-memory builder draft (P3-11 / P4-1).
type ScriptDraftDTO struct {
	Lines  []string `json:"lines"`
	Count  int      `json:"count"`
	Body   string   `json:"body"`
	Shebang string  `json:"shebang"`
}

// CustomScriptDTO is a saved custom_scripts row (P4-7).
type CustomScriptDTO struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScriptBody string `json:"scriptBody"`
	UpdatedAt  string `json:"updatedAt"`
}

// CustomScriptInput is create/update payload for saved scripts.
type CustomScriptInput struct {
	Name       string `json:"name"`
	ScriptBody string `json:"scriptBody"`
}

// BackupResultDTO mirrors backup.Result for the UI.
type BackupResultDTO struct {
	SnippetsAdded   int `json:"snippetsAdded"`
	SnippetsSkipped int `json:"snippetsSkipped"`
	ScriptsAdded    int `json:"scriptsAdded"`
	ScriptsSkipped  int `json:"scriptsSkipped"`
	ProgramsCreated int `json:"programsCreated"`
}

// AddToScript appends a command to the in-memory Script Builder draft (P3-11).
func (a *API) AddToScript(command string) (ScriptDraftDTO, error) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return ScriptDraftDTO{}, fmt.Errorf("порожня команда")
	}
	a.draft.Add(cmd)
	return a.ScriptDraft(), nil
}

// ScriptDraft returns the current in-memory draft snapshot with generated body.
func (a *API) ScriptDraft() ScriptDraftDTO {
	lines := a.draft.Snapshot()
	return ScriptDraftDTO{
		Lines:   lines,
		Count:   len(lines),
		Body:    a.draft.Generate(),
		Shebang: scriptbuilder.Shebang,
	}
}

// MoveScriptLine moves a draft line up or down (P4-2). direction: "up" | "down".
func (a *API) MoveScriptLine(index int, direction string) (ScriptDraftDTO, error) {
	dir := strings.ToLower(strings.TrimSpace(direction))
	var err error
	switch dir {
	case "up":
		err = a.draft.MoveUp(index)
	case "down":
		err = a.draft.MoveDown(index)
	default:
		return ScriptDraftDTO{}, fmt.Errorf("direction must be up|down")
	}
	if err != nil {
		return ScriptDraftDTO{}, err
	}
	return a.ScriptDraft(), nil
}

// RemoveScriptLine deletes a draft line by index.
func (a *API) RemoveScriptLine(index int) (ScriptDraftDTO, error) {
	if err := a.draft.Remove(index); err != nil {
		return ScriptDraftDTO{}, err
	}
	return a.ScriptDraft(), nil
}

// ClearScriptDraft empties the in-memory draft.
func (a *API) ClearScriptDraft() ScriptDraftDTO {
	a.draft.Clear()
	return a.ScriptDraft()
}

// SetScriptBody replaces the draft from an editor body (P4-3).
func (a *API) SetScriptBody(body string) ScriptDraftDTO {
	a.draft.ApplyBody(body)
	return a.ScriptDraft()
}

// GenerateScript returns the current script text with shebang.
func (a *API) GenerateScript() string {
	return a.draft.Generate()
}

// ExportScriptFile writes the script (or provided body) to path with chmod +x (P4-6).
func (a *API) ExportScriptFile(path, body string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("шлях файлу порожній")
	}
	if strings.TrimSpace(body) == "" {
		body = a.draft.Generate()
	}
	return scriptbuilder.WriteExecutable(path, body)
}

// RunScript runs a full script body in the terminal with the same safety gate (P4-5).
// The body is written to a temp executable and launched as `bash <path>` so newlines stay intact.
func (a *API) RunScript(req RunRequest) (RunResponse, error) {
	body := strings.TrimSpace(req.Command)
	if body == "" {
		body = strings.TrimSpace(a.draft.Generate())
	}
	body = scriptbuilder.EnsureShebang(body)
	r := safety.Analyze(body)
	if r.Dangerous && !req.Confirmed {
		return RunResponse{NeedsConfirm: true, Reasons: r.Reasons}, nil
	}
	dir, err := os.MkdirTemp("", "snippetshow-script-*")
	if err != nil {
		return RunResponse{Error: err.Error()}, err
	}
	path := filepath.Join(dir, "run.sh")
	if err := scriptbuilder.WriteExecutable(path, body); err != nil {
		return RunResponse{Error: err.Error()}, err
	}
	return a.RunCommand(RunRequest{
		Command:   "bash " + shellQuote(path),
		Confirmed: true, // already gated above on full body
	})
}

func shellQuote(path string) string {
	return `'` + strings.ReplaceAll(path, `'`, `'\''`) + `'`
}

// ListCustomScripts returns saved scripts from DB (P4-7).
func (a *API) ListCustomScripts() ([]CustomScriptDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	list, err := a.store.ListCustomScripts()
	if err != nil {
		return nil, err
	}
	out := make([]CustomScriptDTO, 0, len(list))
	for _, cs := range list {
		out = append(out, toScriptDTO(cs))
	}
	return out, nil
}

// SaveCustomScript creates or updates a named script (P4-7).
// If id > 0 — update; otherwise create. Body defaults to current draft.
func (a *API) SaveCustomScript(id int64, in CustomScriptInput) (*CustomScriptDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	body := strings.TrimSpace(in.ScriptBody)
	if body == "" {
		body = strings.TrimSpace(a.draft.Generate())
	}
	body = scriptbuilder.EnsureShebang(body)
	if name == "" || body == "" {
		return nil, fmt.Errorf("назва і тіло скрипту обовʼязкові")
	}
	var (
		cs  *db.CustomScript
		err error
	)
	if id > 0 {
		cs, err = a.store.UpdateCustomScript(id, name, body)
	} else {
		cs, err = a.store.CreateCustomScript(name, body)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("скрипт #%d не знайдено", id)
		}
		return nil, err
	}
	dto := toScriptDTO(*cs)
	return &dto, nil
}

// LoadCustomScript loads a saved script into the draft and returns it.
func (a *API) LoadCustomScript(id int64) (ScriptDraftDTO, error) {
	if err := a.ensure(); err != nil {
		return ScriptDraftDTO{}, err
	}
	cs, err := a.store.GetCustomScript(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ScriptDraftDTO{}, fmt.Errorf("скрипт #%d не знайдено", id)
		}
		return ScriptDraftDTO{}, err
	}
	a.draft.ApplyBody(cs.ScriptBody)
	return a.ScriptDraft(), nil
}

// DeleteCustomScript removes a saved script.
func (a *API) DeleteCustomScript(id int64) error {
	if err := a.ensure(); err != nil {
		return err
	}
	if err := a.store.DeleteCustomScript(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("скрипт #%d не знайдено", id)
		}
		return err
	}
	return nil
}

// ExportBackup writes snippets+scripts JSON to path (P4-8).
func (a *API) ExportBackup(path string) error {
	if err := a.ensure(); err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("шлях файлу порожній")
	}
	return backup.Export(a.store, path)
}

// ImportBackup merges a JSON backup (P4-9).
func (a *API) ImportBackup(path string) (BackupResultDTO, error) {
	if err := a.ensure(); err != nil {
		return BackupResultDTO{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return BackupResultDTO{}, fmt.Errorf("шлях файлу порожній")
	}
	res, err := backup.Import(a.store, path)
	if err != nil {
		return BackupResultDTO{}, err
	}
	return BackupResultDTO{
		SnippetsAdded:   res.SnippetsAdded,
		SnippetsSkipped: res.SnippetsSkipped,
		ScriptsAdded:    res.ScriptsAdded,
		ScriptsSkipped:  res.ScriptsSkipped,
		ProgramsCreated: res.ProgramsCreated,
	}, nil
}

func toScriptDTO(cs db.CustomScript) CustomScriptDTO {
	return CustomScriptDTO{
		ID: cs.ID, Name: cs.Name, ScriptBody: cs.ScriptBody, UpdatedAt: cs.UpdatedAt,
	}
}
