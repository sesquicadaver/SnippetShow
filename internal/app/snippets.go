package app

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"snippetshow/internal/clipboard"
	"snippetshow/internal/db"
	"snippetshow/internal/safety"
	"snippetshow/internal/terminal"
)

// SnippetDTO is a JSON-friendly user snippet row.
type SnippetDTO struct {
	ID        int64  `json:"id"`
	ProgramID int64  `json:"programId"`
	Title     string `json:"title"`
	Command   string `json:"command"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"createdAt"`
}

// SnippetInput is the create/update form payload (P3-2…P3-4).
type SnippetInput struct {
	ProgramID int64  `json:"programId"`
	Title     string `json:"title"`
	Command   string `json:"command"`
	Tags      string `json:"tags"`
}

// SafetyDTO mirrors safety.Result for the UI confirm dialog.
type SafetyDTO struct {
	Dangerous bool     `json:"dangerous"`
	Reasons   []string `json:"reasons"`
}

// RunRequest is the Run button payload.
type RunRequest struct {
	Command   string `json:"command"`
	Confirmed bool   `json:"confirmed"`
}

// RunResponse tells the UI whether the command started or needs confirmation.
type RunResponse struct {
	Started      bool     `json:"started"`
	NeedsConfirm bool     `json:"needsConfirm"`
	Reasons      []string `json:"reasons,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// CopyResponse is returned after a clipboard write.
type CopyResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// ListSnippets returns user snippets for a program (P3-1).
func (a *API) ListSnippets(programID int64) ([]SnippetDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	list, err := a.store.ListUserSnippets(programID)
	if err != nil {
		return nil, err
	}
	out := make([]SnippetDTO, 0, len(list))
	for _, sn := range list {
		out = append(out, toSnippetDTO(sn))
	}
	return out, nil
}

// CreateSnippet validates and inserts a snippet (P3-2, P3-4).
func (a *API) CreateSnippet(in SnippetInput) (*SnippetDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	if err := validateSnippet(in.Title, in.Command); err != nil {
		return nil, err
	}
	sn, err := a.store.CreateUserSnippet(in.ProgramID, in.Title, in.Command, in.Tags)
	if err != nil {
		return nil, err
	}
	dto := toSnippetDTO(*sn)
	return &dto, nil
}

// UpdateSnippet edits an existing snippet (P3-3, P3-4).
func (a *API) UpdateSnippet(id int64, in SnippetInput) (*SnippetDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	if err := validateSnippet(in.Title, in.Command); err != nil {
		return nil, err
	}
	sn, err := a.store.UpdateUserSnippet(id, in.Title, in.Command, in.Tags)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("сніпет #%d не знайдено", id)
		}
		return nil, err
	}
	dto := toSnippetDTO(*sn)
	return &dto, nil
}

// DeleteSnippet removes a snippet (P3-3).
func (a *API) DeleteSnippet(id int64) error {
	if err := a.ensure(); err != nil {
		return err
	}
	if err := a.store.DeleteUserSnippet(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("сніпет #%d не знайдено", id)
		}
		return err
	}
	return nil
}

// AnalyzeCommand returns safety info without starting anything (P3-9).
func (a *API) AnalyzeCommand(command string) SafetyDTO {
	r := safety.Analyze(command)
	return SafetyDTO{Dangerous: r.Dangerous, Reasons: r.Reasons}
}

// CopyText copies to clipboard via OS backends (App layer may wrap with Wails first).
func (a *API) CopyText(text string) (CopyResponse, error) {
	if strings.TrimSpace(text) == "" {
		return CopyResponse{}, fmt.Errorf("немає тексту для копіювання")
	}
	if err := clipboard.Copy(text); err != nil {
		return CopyResponse{OK: false, Message: err.Error()}, err
	}
	return CopyResponse{OK: true, Message: "Скопійовано"}, nil
}

// RunCommand starts a terminal run, requiring confirm for dangerous constructs (P3-8…P3-10).
func (a *API) RunCommand(req RunRequest) (RunResponse, error) {
	cmd := strings.TrimSpace(req.Command)
	if cmd == "" {
		return RunResponse{Error: "порожня команда"}, fmt.Errorf("порожня команда")
	}
	r := safety.Analyze(cmd)
	if r.Dangerous && !req.Confirmed {
		return RunResponse{NeedsConfirm: true, Reasons: r.Reasons}, nil
	}
	if err := terminal.Run(cmd); err != nil {
		return RunResponse{Error: err.Error()}, err
	}
	return RunResponse{Started: true}, nil
}

func validateSnippet(title, command string) error {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(command) == "" {
		return fmt.Errorf("назва і команда обовʼязкові")
	}
	return nil
}

func toSnippetDTO(sn db.UserSnippet) SnippetDTO {
	return SnippetDTO{
		ID: sn.ID, ProgramID: sn.ProgramID, Title: sn.Title,
		Command: sn.Command, Tags: sn.Tags, CreatedAt: sn.CreatedAt,
	}
}
