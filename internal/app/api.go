// Package app exposes Wails-facing application services.
// Phase 4: Script Builder, JSON backup, polish on top of Wiki + Snippets.
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

// ProgramDTO is a JSON-friendly navigator row (command or package — SS-29 / D10 / D13).
type ProgramDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	IsInstalled bool   `json:"isInstalled"`
	ManParsed   bool   `json:"manParsed"`
	IsFavorite  bool   `json:"isFavorite"`
	Category    string `json:"category"`
	ManStatus   string `json:"manStatus"`
	// Kind is "command" (PATH / programs) or "package" (repo catalog).
	Kind string `json:"kind"`
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
	// ParseWarning is set when on-demand man parse failed (M2); detail still returns.
	ParseWarning string `json:"parseWarning,omitempty"`
	// NeedsFavorite is set when man is gated until the user favorites the command (D13.4).
	NeedsFavorite bool `json:"needsFavorite,omitempty"`
}

// SearchResultDTO is an FTS hit for the UI.
type SearchResultDTO struct {
	ProgramID   int64  `json:"programId"`
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Rank        float64 `json:"rank"`
	IsInstalled bool   `json:"isInstalled"`
}

// ListQuery is the Wiki navigator filter contract (P2-4…P2-6 / D13).
type ListQuery struct {
	Mode     string `json:"mode"`     // "installed" (default) | "favorites" | "all"
	Letter   string `json:"letter"`   // "all" | "A"–"Z" | "0-9"
	Query    string `json:"query"`    // FTS text; empty → list
	Category string `json:"category"` // raw PM Section filter; empty → all
	NoMan    bool   `json:"noMan"`    // man_status=absent filter (D13.10)
	Limit    int    `json:"limit"`
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
	opCancel   context.CancelFunc // cancels in-flight UpdateIndex / SyncRepoCatalog (M3)
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
		Phase:    "4",
		Frontend: "vanilla-js",
		License:  "MIT",
	}
}

// PhaseStatus is a human-readable build stage marker for the shell UI.
func (a *API) PhaseStatus() string {
	return "Фаза 4: Script Builder, JSON backup, збірка Ubuntu/Arch/Fedora"
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

// EnsureIndexed runs a fast first-run scan when the DB has no programs (D13.1).
// Caller should ListPrograms first, then EnrichMetadata (SS-40) — do not race enrich
// against the first list paint on a single-writer SQLite connection.
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

// beginLongOp marks the API busy and returns a cancellable context (M3).
func (a *API) beginLongOp(timeout time.Duration) (context.Context, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.updating {
		return nil, fmt.Errorf("оновлення індексу вже виконується")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.updating = true
	a.opCancel = cancel
	return ctx, nil
}

// endLongOp clears the busy flag and cancel handle.
func (a *API) endLongOp() {
	a.mu.Lock()
	cancel := a.opCancel
	a.opCancel = nil
	a.updating = false
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// CancelIndex cancels an in-flight UpdateIndex or SyncRepoCatalog (M3 / P2-7).
// Returns true if a cancel signal was delivered.
func (a *API) CancelIndex() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.opCancel == nil {
		return false
	}
	a.opCancel()
	return true
}

// IndexBusy reports whether UpdateIndex / SyncRepoCatalog is running.
func (a *API) IndexBusy() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updating
}

func isCancelErr(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

// UpdateIndex performs a fast PATH names-only refresh (SS-36b / P1-15 / P2-7).
// EnrichMetadata is NOT started here — the UI must reload the list first (SS-40),
// then call EnrichMetadata so Section enrich cannot starve ListPrograms.
func (a *API) UpdateIndex() (index.UpdateResult, error) {
	if err := a.ensure(); err != nil {
		return index.UpdateResult{}, err
	}
	ctx, err := a.beginLongOp(2 * time.Minute)
	if err != nil {
		return index.UpdateResult{}, err
	}

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
	a.endLongOp()

	if isCancelErr(err) || isCancelErr(ctx.Err()) {
		a.emit(ProgressEvent{Phase: "cancelled", Message: "Скасовано"})
		return res, fmt.Errorf("скасовано")
	}
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
	ctx, err := a.beginLongOp(5 * time.Minute)
	if err != nil {
		return index.CatalogSyncResult{}, err
	}
	defer a.endLongOp()

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
	if isCancelErr(err) || isCancelErr(ctx.Err()) {
		a.emit(ProgressEvent{Phase: "cancelled", Message: "Скасовано"})
		return res, fmt.Errorf("скасовано")
	}
	return res, err
}

// GetProgram returns command or package detail (P1-14 / P2-14 / D13).
// Man is loaded only for favorites (D13.3–D13.4); parseMan is ignored unless is_favorite.
// Uninstalled repo packages (SS-29) resolve from the packages table with install hints (D6).
func (a *API) GetProgram(name string, parseMan bool) (*ProgramDetail, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	p, err := a.store.GetProgramByName(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return a.getPackageDetail(name)
		}
		return nil, err
	}
	var parseWarning string
	if p.IsInstalled && !p.IsFavorite {
		dto := ProgramDetail{
			Program:       toDTO(p),
			Snippets:      []SnippetDTO{},
			NeedsFavorite: true,
		}
		pm, _ := scanner.DetectPM()
		dto.PackageManager = string(pm)
		return &dto, nil
	}
	if parseMan && p.IsInstalled && p.IsFavorite && !p.ManParsed && p.ManStatus != db.ManStatusAbsent {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a.emit(ProgressEvent{Phase: "parse", Message: "Парсинг man: " + name})
		if err := a.updater.ParseProgram(ctx, name); err != nil {
			parseWarning = err.Error()
			_ = a.store.SetManStatus(p.ID, db.ManStatusAbsent, false)
		} else {
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
	dto := ProgramDetail{Program: toDTO(p), Snippets: []SnippetDTO{}, ParseWarning: parseWarning}
	if p.IsFavorite && (p.ManStatus == db.ManStatusAbsent || (parseWarning != "" && len(cmds) == 0)) {
		if parseWarning == "" {
			parseWarning = "Man-сторінка відсутня або не розпарсена"
		}
		dto.ParseWarning = parseWarning
	}
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

// SetFavorite toggles Обрані and parses or purges man (D13.3–D13.5).
func (a *API) SetFavorite(name string, on bool) (*ProgramDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	p, err := a.updater.SetFavorite(ctx, name, on)
	if err != nil {
		return nil, err
	}
	dto := toDTO(p)
	return &dto, nil
}

// ListCategories returns distinct raw PM Section values (D13.8).
func (a *API) ListCategories() ([]string, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	return a.store.ListCategories()
}

// EnrichMetadata runs bulk Section / man-status enrich (D13.9); usable from UI/tests.
func (a *API) EnrichMetadata() error {
	if err := a.ensure(); err != nil {
		return err
	}
	ctx, err := a.beginLongOp(10 * time.Minute)
	if err != nil {
		return err
	}
	defer a.endLongOp()
	progress := make(chan index.Progress, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for p := range progress {
			a.emit(ProgressEvent{Phase: p.Phase, Done: p.Done, Total: p.Total, Message: p.Message})
		}
	}()
	err = a.updater.EnrichMetadata(ctx, progress)
	close(progress)
	<-done
	return err
}

// GetRawMan returns cleaned raw man text for an installed favorite (P2-13 / D13).
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
	if !p.IsFavorite {
		return "", fmt.Errorf("man лише для обраних програм (D13)")
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
// Mode "installed" lists PATH commands (programs); mode "all" lists repo packages (SS-29).
func (a *API) ListPrograms(q ListQuery) ([]ProgramDTO, error) {
	if err := a.ensure(); err != nil {
		return nil, err
	}
	mode := strings.ToLower(strings.TrimSpace(q.Mode))
	if mode == "" {
		mode = "installed"
	}
	query := strings.TrimSpace(q.Query)
	letter := strings.TrimSpace(q.Letter)
	if letter == "" {
		letter = "all"
	}

	if mode == "all" {
		pf := db.PackageFilter{Letter: letter, Limit: q.Limit}
		if query != "" {
			hits, err := a.store.SearchPackagesFiltered(query, pf)
			if err != nil {
				return nil, err
			}
			out := make([]ProgramDTO, 0, len(hits))
			for i := range hits {
				out = append(out, packageToDTO(&hits[i]))
			}
			return out, nil
		}
		list, err := a.store.ListPackagesFiltered(pf)
		if err != nil {
			return nil, err
		}
		out := make([]ProgramDTO, 0, len(list))
		for i := range list {
			out = append(out, packageToDTO(&list[i]))
		}
		return out, nil
	}

	filter := toFilter(q)
	if query != "" {
		hits, err := a.store.SearchFiltered(query, filter)
		if err != nil {
			return nil, err
		}
		out := make([]ProgramDTO, 0, len(hits))
		for _, h := range hits {
			out = append(out, hitToDTO(h))
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
		out = append(out, SearchResultDTO{
			ProgramID: h.ProgramID, Name: h.Name, Summary: h.Summary, Rank: h.Rank, IsInstalled: h.IsInstalled,
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
		InstalledOnly: mode == "installed" || mode == "favorites",
		FavoritesOnly: mode == "favorites",
		Letter:        letter,
		Category:      strings.TrimSpace(q.Category),
		NoManOnly:     q.NoMan,
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
		IsFavorite:  p.IsFavorite,
		Category:    p.Category,
		ManStatus:   p.ManStatus,
		Kind:        "command",
	}
}

func hitToDTO(h db.SearchHit) ProgramDTO {
	return ProgramDTO{
		ID:          h.ProgramID,
		Name:        h.Name,
		PackageName: h.PackageName,
		Summary:     h.Summary,
		Version:     h.Version,
		IsInstalled: h.IsInstalled,
		ManParsed:   h.ManParsed,
		IsFavorite:  h.IsFavorite,
		Category:    h.Category,
		ManStatus:   h.ManStatus,
		Kind:        "command",
	}
}

func packageToDTO(p *db.Package) ProgramDTO {
	return ProgramDTO{
		ID:          p.ID,
		Name:        p.Name,
		PackageName: p.Name,
		Summary:     p.Summary,
		Version:     p.Version,
		IsInstalled: p.IsInstalled,
		ManParsed:   false,
		Kind:        "package",
	}
}

func (a *API) getPackageDetail(name string) (*ProgramDetail, error) {
	pkg, err := a.store.GetPackageByName(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("program %q not found", name)
		}
		return nil, err
	}
	dto := ProgramDetail{
		Program:  packageToDTO(pkg),
		Commands: nil,
		Snippets: []SnippetDTO{},
	}
	pm, _ := scanner.DetectPM()
	dto.PackageManager = string(pm)
	if !pkg.IsInstalled {
		dto.InstallCommand = scanner.InstallCommand(pm, pkg.Name)
		dto.CacheHint = scanner.CacheHint(pm)
	}
	return &dto, nil
}
