// Package app exposes Wails-facing application services.
// Phase 1: Core read/index bindings (P1-16).
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"snippetshow/internal/db"
	"snippetshow/internal/index"
	"snippetshow/internal/manparser"
	"snippetshow/internal/scanner"
)

// Info describes product identity shown in the UI shell.
type Info struct {
	Name     string `json:"name"`
	Subtitle string `json:"subtitle"`
	Module   string `json:"module"`
	DBPath   string `json:"dbPath"`
	UILocale string `json:"uiLocale"`
	Phase    string `json:"phase"`
	Frontend string `json:"frontend"`
	License  string `json:"license"`
}

// ProgramDTO is a JSON-friendly program row for the frontend.
type ProgramDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	IsInstalled bool   `json:"isInstalled"`
	ManParsed   bool   `json:"manParsed"`
}

// ManCommandDTO is a JSON-friendly man flag row.
type ManCommandDTO struct {
	ID          int64  `json:"id"`
	Flag        string `json:"flag"`
	Description string `json:"description"`
}

// ProgramDetail bundles program metadata with man commands.
type ProgramDetail struct {
	Program  ProgramDTO      `json:"program"`
	Commands []ManCommandDTO `json:"commands"`
	RawMan   string          `json:"rawMan,omitempty"`
}

// SearchResultDTO is an FTS hit for the UI.
type SearchResultDTO struct {
	ProgramID int64   `json:"programId"`
	Name      string  `json:"name"`
	Summary   string  `json:"summary"`
	Rank      float64 `json:"rank"`
}

// API is the application service facade bound through the Wails App root.
type API struct {
	mu      sync.Mutex
	store   *db.Store
	updater *index.Updater
	parser  *manparser.Parser
	dbPath  string
}

// New returns an API that opens the default DB path lazily.
func New() *API {
	return &API{dbPath: db.DefaultPath, parser: manparser.New()}
}

// NewWithStore wires a pre-opened store (tests).
func NewWithStore(store *db.Store) *API {
	u := index.New(store)
	return &API{store: store, updater: u, parser: u.Parser, dbPath: store.Path()}
}

// Info returns fixed brand and path decisions (D1–D4).
func (a *API) Info() Info {
	return Info{
		Name:     "SnippetShow",
		Subtitle: "Linux Command Wiki & Script Builder",
		Module:   "snippetshow",
		DBPath:   db.DefaultPath,
		UILocale: "uk",
		Phase:    "1",
		Frontend: "vanilla-js",
		License:  "MIT",
	}
}

// PhaseStatus is a human-readable build stage marker for the shell UI.
func (a *API) PhaseStatus() string {
	return "Фаза 1: Core & DB (scanner, manparser, SQLite/FTS5, індекс)"
}

func (a *API) ensure() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		return nil
	}
	store, err := db.Open(a.dbPath)
	if err != nil {
		return err
	}
	a.store = store
	a.updater = index.New(store)
	a.parser = a.updater.Parser
	return nil
}

// Close releases the database (tests / shutdown).
func (a *API) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		return nil
	}
	err := a.store.Close()
	a.store = nil
	a.updater = nil
	return err
}

// ScanInstalled runs a PATH/PM scan and upserts installed programs (P1-16).
func (a *API) ScanInstalled() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return a.updater.FirstRun(ctx)
}

// UpdateIndex performs incremental index refresh (P1-15/P1-16).
func (a *API) UpdateIndex() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return a.updater.UpdateIndex(ctx)
}

// GetProgram returns program detail; optionally parses man on demand (P1-14).
func (a *API) GetProgram(name string, parseMan bool) (*ProgramDetail, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	p, err := a.store.GetProgramByName(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("program %q not found", name)
		}
		return nil, err
	}
	if parseMan && p.IsInstalled && !p.ManParsed {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.updater.ParseProgram(ctx, name); err == nil {
			p, err = a.store.GetProgramByName(name)
			if err != nil {
				return nil, err
			}
		}
	}
	cmds, err := a.store.ListManCommands(p.ID)
	if err != nil {
		return nil, err
	}
	dto := ProgramDetail{
		Program: toDTO(p),
	}
	for _, c := range cmds {
		dto.Commands = append(dto.Commands, ManCommandDTO{ID: c.ID, Flag: c.Flag, Description: c.Description})
	}
	return &dto, nil
}

// Search runs FTS5 search (P1-16).
func (a *API) Search(query string) ([]SearchResultDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	hits, err := a.store.Search(query)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResultDTO, 0, len(hits))
	for _, h := range hits {
		out = append(out, SearchResultDTO{
			ProgramID: h.ProgramID,
			Name:      h.Name,
			Summary:   h.Summary,
			Rank:      h.Rank,
		})
	}
	return out, nil
}

// ListInstalled returns installed programs from the DB.
func (a *API) ListInstalled() ([]ProgramDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	list, err := a.store.ListPrograms(true)
	if err != nil {
		return nil, err
	}
	out := make([]ProgramDTO, 0, len(list))
	for i := range list {
		out = append(out, toDTO(&list[i]))
	}
	return out, nil
}

// DetectPackageManager exposes scanner PM detection for the UI.
func (a *API) DetectPackageManager() (string, error) {
	pm, err := scanner.DetectPM()
	if err != nil {
		return string(scanner.PMUnknown), err
	}
	return string(pm), nil
}

func toDTO(p *db.Program) ProgramDTO {
	return ProgramDTO{
		ID:          p.ID,
		Name:        p.Name,
		PackageName: p.PackageName,
		Summary:     p.Summary,
		Version:     p.Version,
		IsInstalled: p.IsInstalled,
		ManParsed:   p.ManParsed,
	}
}
