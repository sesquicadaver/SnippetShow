// Package app exposes Wails-facing application services.
// Phase 2: Read-Only Wiki UI bindings on top of Core & DB (P1-16 + P2).
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"snippetshow/internal/db"
	"snippetshow/internal/index"
	"snippetshow/internal/manparser"
	"snippetshow/internal/scanner"
	"snippetshow/internal/scriptbuilder"
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

// ProgramDetail bundles program metadata with man commands, snippets and install hint.
type ProgramDetail struct {
	Program        ProgramDTO      `json:"program"`
	Commands       []ManCommandDTO `json:"commands"`
	Snippets       []SnippetDTO    `json:"snippets"`
	RawMan         string          `json:"rawMan,omitempty"`
	InstallCommand string          `json:"installCommand,omitempty"`
	PackageManager string          `json:"packageManager,omitempty"`
	CacheHint      string          `json:"cacheHint,omitempty"`
}

// SearchResultDTO is an FTS hit for the UI.
type SearchResultDTO struct {
	ProgramID   int64  `json:"programId"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Rank        float64 `json:"rank"`
	IsInstalled bool   `json:"isInstalled"`
}

// ListQuery is the Wiki navigator filter contract (P2-4…P2-6).
type ListQuery struct {
	Mode   string `json:"mode"`   // "installed" (default) | "all"
	Letter string `json:"letter"` // "all" | "A"–"Z" | "0-9"
	Query  string `json:"query"`  // FTS text; empty → list
	Limit  int    `json:"limit"`
}

// ProgressEvent is emitted to the UI during index/catalog work (P2-7).
type ProgressEvent struct {
	Phase   string `json:"phase"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

// ProgressHandler receives progress for Wails EventsEmit wiring.
type ProgressHandler func(ProgressEvent)

// API is the application service facade bound through the Wails App root.
type API struct {
	mu         sync.Mutex
	store      *db.Store
	updater    *index.Updater
	parser     *manparser.Parser
	dbPath     string
	onProgress ProgressHandler
	updating   bool
	draft      *scriptbuilder.Draft
}

// New returns an API that opens the default DB path lazily.
func New() *API {
	return &API{
		dbPath: db.DefaultPath,
		parser: manparser.New(),
		draft:  scriptbuilder.NewDraft(),
	}
}

// NewWithStore wires a pre-opened store (tests).
func NewWithStore(store *db.Store) *API {
	u := index.New(store)
	return &API{
		store: store, updater: u, parser: u.Parser, dbPath: store.Path(),
		draft: scriptbuilder.NewDraft(),
	}
}

// SetProgressHandler registers a callback for index/catalog progress (UI).
func (a *API) SetProgressHandler(h ProgressHandler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onProgress = h
}

// Info returns fixed brand and path decisions (D1–D4).
func (a *API) Info() Info {
	return Info{
		Name:     "SnippetShow",
		Subtitle: "Linux Command Wiki & Script Builder",
		Module:   "snippetshow",
		DBPath:   db.DefaultPath,
		UILocale: "uk",
		Phase:    "3",
		Frontend: "vanilla-js",
		License:  "MIT",
	}
}

// PhaseStatus is a human-readable build stage marker for the shell UI.
func (a *API) PhaseStatus() string {
	return "Фаза 3: User Snippets & Actions (CRUD, Copy, Run + safety)"
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

func (a *API) emit(p ProgressEvent) {
	a.mu.Lock()
	h := a.onProgress
	a.mu.Unlock()
	if h != nil {
		h(p)
	}
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

// EnsureIndexed runs a fast first-run scan when the DB has no programs.
func (a *API) EnsureIndexed() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return a.updater.EnsureIndexed(ctx)
}

// ScanInstalled runs a PATH/PM scan and upserts installed programs (P1-16).
func (a *API) ScanInstalled() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	a.emit(ProgressEvent{Phase: "scan", Message: "Сканування встановлених програм"})
	res, err := a.updater.FirstRun(ctx)
	if err != nil {
		return res, err
	}
	a.emit(ProgressEvent{
		Phase:   "done",
		Message: fmt.Sprintf("Оновлено %d пакетів", res.Added+res.Updated+res.MarkedGone),
	})
	return res, nil
}

// UpdateIndex performs incremental index refresh with progress events (P1-15/P2-7).
func (a *API) UpdateIndex() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	a.mu.Lock()
	if a.updating {
		a.mu.Unlock()
		return index.UpdateResult{}, fmt.Errorf("оновлення індексу вже виконується")
	}
	a.updating = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.updating = false
		a.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	progress := make(chan index.Progress, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range progress {
			a.emit(ProgressEvent{Phase: p.Phase, Done: p.Done, Total: p.Total, Message: p.Message})
		}
	}()

	res, err := a.updater.UpdateIndexWithProgress(ctx, progress)
	close(progress)
	<-done
	if err != nil {
		return res, err
	}
	a.emit(ProgressEvent{
		Phase:   "done",
		Message: fmt.Sprintf("Оновлено %d пакетів", res.Added+res.Updated+res.MarkedGone),
	})
	return res, nil
}

// SyncRepoCatalog loads the PM cache into programs for «Всі програми» (D5).
func (a *API) SyncRepoCatalog() (index.CatalogSyncResult, error) {
	if err := a.ensure(); err != nil {
		return index.CatalogSyncResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	progress := make(chan index.Progress, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range progress {
			a.emit(ProgressEvent{Phase: p.Phase, Done: p.Done, Total: p.Total, Message: p.Message})
		}
	}()
	res, err := a.updater.SyncRepoCatalog(ctx, progress)
	close(progress)
	<-done
	return res, err
}

// GetProgram returns program detail; optionally parses man on demand (P1-14 / P2-14).
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
		a.emit(ProgressEvent{Phase: "parse", Message: "Парсинг man: " + name})
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
	sns, err := a.store.ListUserSnippets(p.ID)
	if err != nil {
		return nil, err
	}
	dto := ProgramDetail{Program: toDTO(p), Snippets: []SnippetDTO{}}
	for _, c := range cmds {
		dto.Commands = append(dto.Commands, ManCommandDTO{ID: c.ID, Flag: c.Flag, Description: c.Description})
	}
	for _, sn := range sns {
		dto.Snippets = append(dto.Snippets, toSnippetDTO(sn))
	}
	pm, _ := scanner.DetectPM()
	dto.PackageManager = string(pm)
	if !p.IsInstalled {
		pkg := p.PackageName
		if pkg == "" {
			pkg = p.Name
		}
		dto.InstallCommand = scanner.InstallCommand(pm, pkg)
		dto.CacheHint = scanner.CacheHint(pm)
	}
	return &dto, nil
}

// GetRawMan returns cleaned raw man text for an installed program (P2-13).
func (a *API) GetRawMan(name string) (string, error) {
	if err := a.ensure(); err != nil {
		return "", err
	}
	p, err := a.store.GetProgramByName(name)
	if err != nil {
		return "", err
	}
	if !p.IsInstalled {
		return "", fmt.Errorf("man недоступний для невстановленої програми %q (D6)", name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.parser.RawMan(ctx, name)
}

// Search runs FTS5 search (P1-16).
func (a *API) Search(query string) ([]SearchResultDTO, error) {
	return a.SearchPrograms(ListQuery{Query: query, Mode: "all"})
}

// ListInstalled returns installed programs from the DB.
func (a *API) ListInstalled() ([]ProgramDTO, error) {
	return a.ListPrograms(ListQuery{Mode: "installed"})
}

// ListPrograms returns navigator rows for mode/letter/query (P2-4…P2-6, P2-8).
func (a *API) ListPrograms(q ListQuery) ([]ProgramDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	filter := toFilter(q)
	query := strings.TrimSpace(q.Query)
	if query != "" {
		hits, err := a.store.SearchFiltered(query, filter)
		if err != nil {
			return nil, err
		}
		out := make([]ProgramDTO, 0, len(hits))
		for _, h := range hits {
			p, err := a.store.GetProgramByID(h.ProgramID)
			if err != nil {
				continue
			}
			out = append(out, toDTO(p))
		}
		return out, nil
	}
	list, err := a.store.ListProgramsFiltered(filter)
	if err != nil {
		return nil, err
	}
	out := make([]ProgramDTO, 0, len(list))
	for i := range list {
		out = append(out, toDTO(&list[i]))
	}
	return out, nil
}

// SearchPrograms is ListPrograms with FTS emphasis (same filter contract).
func (a *API) SearchPrograms(q ListQuery) ([]SearchResultDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	filter := toFilter(q)
	query := strings.TrimSpace(q.Query)
	if query == "" {
		list, err := a.store.ListProgramsFiltered(filter)
		if err != nil {
			return nil, err
		}
		out := make([]SearchResultDTO, 0, len(list))
		for _, p := range list {
			out = append(out, SearchResultDTO{
				ProgramID: p.ID, Name: p.Name, Summary: p.Summary, IsInstalled: p.IsInstalled,
			})
		}
		return out, nil
	}
	hits, err := a.store.SearchFiltered(query, filter)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResultDTO, 0, len(hits))
	for _, h := range hits {
		p, err := a.store.GetProgramByID(h.ProgramID)
		installed := false
		if err == nil {
			installed = p.IsInstalled
		}
		out = append(out, SearchResultDTO{
			ProgramID: h.ProgramID, Name: h.Name, Summary: h.Summary, Rank: h.Rank, IsInstalled: installed,
		})
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

// InstallCommandFor returns a suggested install command for a package/binary name.
func (a *API) InstallCommandFor(pkg string) (string, error) {
	pm, err := scanner.DetectPM()
	if err != nil {
		return "", err
	}
	cmd := scanner.InstallCommand(pm, strings.TrimSpace(pkg))
	if cmd == "" {
		return "", fmt.Errorf("невідомий пакетний менеджер")
	}
	return cmd, nil
}

func toFilter(q ListQuery) db.ProgramFilter {
	mode := strings.ToLower(strings.TrimSpace(q.Mode))
	if mode == "" {
		mode = "installed"
	}
	letter := strings.TrimSpace(q.Letter)
	if letter == "" {
		letter = "all"
	}
	return db.ProgramFilter{
		InstalledOnly: mode != "all",
		Letter:        letter,
		Limit:         q.Limit,
	}
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
