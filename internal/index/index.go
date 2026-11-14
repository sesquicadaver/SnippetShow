// Package index orchestrates incremental program index updates (Phase 1 / ТЗ §2.3 + D13).
package index

import (
	"context"
	"fmt"
	"time"

	"snippetshow/internal/db"
	"snippetshow/internal/manparser"
	"snippetshow/internal/scanner"
)

// Progress is emitted during UpdateIndex / bulk parse.
type Progress struct {
	Phase   string `json:"phase"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

// UpdateResult summarises an incremental update.
type UpdateResult struct {
	Added         int `json:"added"`
	Updated       int `json:"updated"`
	MarkedGone    int `json:"markedGone"`
	Parsed        int `json:"parsed"`
	InstalledSeen int `json:"installedSeen"`
}

// Updater coordinates scanner + manparser + db.
type Updater struct {
	Store   *db.Store
	Scanner *scanner.Scanner
	Parser  *manparser.Parser
	// ParseOnUpdate, when true, reparses man for favorite programs on version change (D13.7).
	ParseOnUpdate bool
	Workers       int
}

// New builds an Updater with default OS-backed scanner/parser.
func New(store *db.Store) *Updater {
	return &Updater{
		Store:         store,
		Scanner:       scanner.New(),
		Parser:        manparser.New(),
		ParseOnUpdate: true,
		Workers:       4,
	}
}

// FirstRun performs a fast names-only installed scan (D13.1 / P1-13).
func (u *Updater) FirstRun(ctx context.Context) (UpdateResult, error) {
	return u.syncInstalled(ctx, false, nil)
}

// UpdateIndex diffs installed programs vs DB: add, reparse favorites on version change,
// mark missing as is_installed=0 without deleting snippets (P1-15 / D13.7).
func (u *Updater) UpdateIndex(ctx context.Context) (UpdateResult, error) {
	return u.UpdateIndexWithProgress(ctx, nil)
}

// UpdateIndexWithProgress is UpdateIndex with optional progress callbacks.
// After the names pass it runs bulk metadata enrich (D13.9); favorite man reparse
// happens inside EnrichMetadata on version change (D13.7).
func (u *Updater) UpdateIndexWithProgress(ctx context.Context, progress chan<- Progress) (UpdateResult, error) {
	res, err := u.syncInstalled(ctx, false, progress)
	if err != nil {
		return res, err
	}
	if err := u.EnrichMetadata(ctx, progress); err != nil {
		return res, err
	}
	return res, nil
}

func (u *Updater) syncInstalled(ctx context.Context, parseFavorites bool, progress chan<- Progress) (UpdateResult, error) {
	if u.Store == nil {
		return UpdateResult{}, fmt.Errorf("index: nil store")
	}
	send := func(p Progress) {
		if progress == nil {
			return
		}
		select {
		case progress <- p:
		case <-ctx.Done():
		}
	}
	send(Progress{Phase: "scan", Message: "Сканування встановлених програм"})

	names, err := u.Scanner.ScanExecutableNames()
	if err != nil {
		return UpdateResult{}, err
	}
	res := UpdateResult{InstalledSeen: len(names)}
	keep := make([]string, 0, len(names))
	var toParse []string

	for i, name := range names {
		keep = append(keep, name)
		existing, err := u.Store.GetProgramByName(name)
		resetMan := false
		isNew := false
		wasFavorite := false
		if err != nil {
			isNew = true
		} else {
			wasFavorite = existing.IsFavorite
			if existing.Version != "" {
				// Version compare happens after background enrich; here we only upsert presence.
			}
			if !existing.IsInstalled {
				res.Updated++
			}
		}

		_, err = u.Store.UpsertProgram(db.Program{
			Name:        name,
			IsInstalled: true,
			ManStatus:   db.ManStatusUnknown,
		}, resetMan)
		if err != nil {
			return res, err
		}
		if isNew {
			res.Added++
		}
		if parseFavorites && wasFavorite && resetMan {
			toParse = append(toParse, name)
		}
		send(Progress{Phase: "upsert", Done: i + 1, Total: len(names), Message: name})
	}

	n, err := u.Store.MarkNotInstalledExcept(keep)
	if err != nil {
		return res, err
	}
	res.MarkedGone = int(n)

	if parseFavorites && len(toParse) > 0 {
		parsed, err := u.parseMany(ctx, toParse, send)
		res.Parsed = parsed
		if err != nil {
			return res, err
		}
	}
	send(Progress{Phase: "done", Message: fmt.Sprintf("Оновлено %d пакетів", res.Added+res.Updated+res.MarkedGone)})
	return res, nil
}

// EnrichMetadata applies bulk PM Section / summary / man presence (D13.8–D13.9).
func (u *Updater) EnrichMetadata(ctx context.Context, progress chan<- Progress) error {
	if u.Store == nil {
		return fmt.Errorf("index: nil store")
	}
	send := func(p Progress) {
		if progress == nil {
			return
		}
		select {
		case progress <- p:
		case <-ctx.Done():
		}
	}
	send(Progress{Phase: "enrich", Message: "Збагачення метаданих (Section / man)"})
	rows, err := u.Scanner.EnrichInstalled(ctx)
	if err != nil {
		return err
	}
	var reparse []string
	for i, e := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		prev, _ := u.Store.GetProgramByName(e.Name)
		status := db.ManStatusAbsent
		if e.HasMan {
			if prev != nil && prev.ManStatus == db.ManStatusParsed {
				status = db.ManStatusParsed
			} else {
				status = db.ManStatusUnknown
			}
		}
		_ = u.Store.UpdateProgramMeta(e.Name, e.PackageName, e.Summary, e.Version, e.Category, status)
		if !e.HasMan {
			if p, err := u.Store.GetProgramByName(e.Name); err == nil {
				_ = u.Store.SetManStatus(p.ID, db.ManStatusAbsent, false)
			}
		}
		// D13.7: version change on a favorite → reparse man.
		if prev != nil && prev.IsFavorite && e.Version != "" && prev.Version != "" && prev.Version != e.Version {
			reparse = append(reparse, e.Name)
		}
		send(Progress{Phase: "enrich", Done: i + 1, Total: len(rows), Message: e.Name})
	}
	if len(reparse) > 0 {
		_, _ = u.parseMany(ctx, reparse, send)
	}
	send(Progress{Phase: "done", Message: "Метадані оновлено"})
	return nil
}

// SetFavorite marks/unmarks a PATH command and parses or purges man (D13.3–D13.5).
func (u *Updater) SetFavorite(ctx context.Context, name string, on bool) (*db.Program, error) {
	if u.Store == nil {
		return nil, fmt.Errorf("index: nil store")
	}
	p, err := u.Store.GetProgramByName(name)
	if err != nil {
		return nil, err
	}
	if !p.IsInstalled {
		return nil, fmt.Errorf("обрані лише для встановлених PATH-команд (D13.6)")
	}
	if !on {
		if err := u.Store.ClearManCommands(p.ID); err != nil {
			return nil, err
		}
		return u.Store.SetFavorite(name, false)
	}
	if _, err := u.Store.SetFavorite(name, true); err != nil {
		return nil, err
	}
	// Parse man into DB on favorite (D13.3).
	if err := u.ParseProgram(ctx, name); err != nil {
		if p2, gerr := u.Store.GetProgramByName(name); gerr == nil {
			_ = u.Store.SetManStatus(p2.ID, db.ManStatusAbsent, false)
			p2.ManStatus = db.ManStatusAbsent
			p2.IsFavorite = true
			return p2, nil // soft-fail: favorite kept without man (D13.11)
		}
		return nil, err
	}
	return u.Store.GetProgramByName(name)
}

// parseMany parses man pages via the worker pool (M1 / ТЗ §2.2) and persists results.
func (u *Updater) parseMany(ctx context.Context, names []string, send func(Progress)) (int, error) {
	send(Progress{Phase: "parse", Total: len(names), Message: "Парсинг man"})
	workers := u.Workers
	if workers < 1 {
		workers = 4
	}
	manProg := make(chan manparser.Progress, len(names))
	bridgeDone := make(chan struct{})
	go func() {
		defer close(bridgeDone)
		for pr := range manProg {
			msg := pr.Command
			if pr.Err != nil {
				msg = pr.Command + ": " + pr.Err.Error()
			}
			send(Progress{Phase: "parse", Done: pr.Done, Total: pr.Total, Message: msg})
		}
	}()
	results := u.Parser.ParseMany(ctx, names, workers, manProg)
	close(manProg)
	<-bridgeDone

	parsed := 0
	for _, pr := range results {
		if pr.Err != nil {
			if p, err := u.Store.GetProgramByName(pr.Command); err == nil {
				_ = u.Store.SetManStatus(p.ID, db.ManStatusAbsent, false)
			}
			continue
		}
		if err := u.applyParsedOptions(ctx, pr.Command, pr.Options); err != nil {
			send(Progress{Phase: "parse", Done: pr.Done, Total: pr.Total, Message: pr.Command + ": " + err.Error()})
			continue
		}
		parsed++
	}
	return parsed, nil
}

// ParseProgram parses man on-demand and stores man_commands (P1-14).
func (u *Updater) ParseProgram(ctx context.Context, name string) error {
	opts, err := u.Parser.ParseOptions(ctx, name)
	if err != nil {
		return err
	}
	return u.applyParsedOptions(ctx, name, opts)
}

// applyParsedOptions writes parsed flags into the store for an installed program.
func (u *Updater) applyParsedOptions(ctx context.Context, name string, opts []manparser.Option) error {
	p, err := u.Store.GetProgramByName(name)
	if err != nil {
		return err
	}
	if !p.IsInstalled {
		return fmt.Errorf("index: man parse skipped for uninstalled %q (D6)", name)
	}
	if len(opts) == 0 {
		_ = u.Store.SetManStatus(p.ID, db.ManStatusAbsent, false)
		return fmt.Errorf("man без прапорців для %q", name)
	}
	cmds := make([]db.ManCommand, 0, len(opts))
	for _, o := range opts {
		cmds = append(cmds, db.ManCommand{Flag: o.Flag, Description: o.Description})
	}
	if sum, err := u.Parser.Summary(ctx, name); err == nil && sum != "" {
		_, _ = u.Store.UpsertProgram(db.Program{
			Name:        p.Name,
			PackageName: p.PackageName,
			Summary:     sum,
			Version:     p.Version,
			IsInstalled: true,
			ManParsed:   true,
			Category:    p.Category,
			ManStatus:   db.ManStatusParsed,
		}, false)
	}
	return u.Store.ReplaceManCommands(p.ID, cmds)
}

// EnsureIndexed runs FirstRun when the programs table is empty.
func (u *Updater) EnsureIndexed(ctx context.Context) (UpdateResult, error) {
	list, err := u.Store.ListPrograms(false)
	if err != nil {
		return UpdateResult{}, err
	}
	if len(list) > 0 {
		return UpdateResult{InstalledSeen: len(list)}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return u.FirstRun(ctx)
}

// CatalogSyncResult summarises a repo-catalog sync into packages (Phase 2 / D5 / SS-29).
type CatalogSyncResult struct {
	Added  int                 `json:"added"`
	Status scanner.CacheStatus `json:"status"`
	Hint   string              `json:"hint"`
	Total  int                 `json:"total"`
}

// SyncRepoCatalog upserts distribution packages into the packages table (SS-29 / D10).
func (u *Updater) SyncRepoCatalog(ctx context.Context, progress chan<- Progress) (CatalogSyncResult, error) {
	if u.Store == nil {
		return CatalogSyncResult{}, fmt.Errorf("index: nil store")
	}
	send := func(p Progress) {
		if progress == nil {
			return
		}
		select {
		case progress <- p:
		case <-ctx.Done():
		}
	}
	send(Progress{Phase: "catalog", Message: "Читання каталогу репозиторію"})
	cat, err := u.Scanner.ListRepoCatalog(ctx)
	out := CatalogSyncResult{Status: cat.Status, Hint: cat.Hint, Total: len(cat.Packages)}
	if err != nil {
		return out, err
	}
	if cat.Status == scanner.CacheEmpty || len(cat.Packages) == 0 {
		send(Progress{Phase: "catalog", Message: cat.Hint})
		return out, nil
	}
	entries := make([]db.CatalogEntry, 0, len(cat.Packages))
	for _, pkg := range cat.Packages {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		default:
		}
		entries = append(entries, db.CatalogEntry{
			Name:        pkg.Name,
			PackageName: pkg.Name,
			Summary:     pkg.Summary,
		})
	}
	added, err := u.Store.UpsertCatalogProgramsBatch(entries, db.DefaultCatalogChunkSize, func(done, total int) {
		msg := ""
		if done > 0 && done <= len(cat.Packages) {
			msg = cat.Packages[done-1].Name
		}
		send(Progress{Phase: "catalog", Done: done, Total: total, Message: msg})
	})
	if err != nil {
		return out, err
	}
	out.Added = added
	send(Progress{Phase: "done", Message: fmt.Sprintf("Каталог: додано %d пакетів", out.Added)})
	return out, nil
}
