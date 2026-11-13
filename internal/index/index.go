// Package index orchestrates incremental program index updates (Phase 1 / ТЗ §2.3).
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
	// ParseOnUpdate, when true, parses man for new/changed programs during UpdateIndex.
	ParseOnUpdate bool
	Workers       int
}

// New builds an Updater with default OS-backed scanner/parser.
// ParseOnUpdate is enabled so «Оновити індекс» reparses man on version change (H1 / ТЗ §2.3).
func New(store *db.Store) *Updater {
	return &Updater{
		Store:         store,
		Scanner:       scanner.New(),
		Parser:        manparser.New(),
		ParseOnUpdate: true,
		Workers:       4,
	}
}

// FirstRun performs a fast installed scan into programs with man_parsed=0 (P1-13).
func (u *Updater) FirstRun(ctx context.Context) (UpdateResult, error) {
	return u.syncInstalled(ctx, false, nil)
}

// UpdateIndex diffs installed programs vs DB: add, reparse on version change,
// mark missing as is_installed=0 without deleting snippets (P1-15).
func (u *Updater) UpdateIndex(ctx context.Context) (UpdateResult, error) {
	return u.UpdateIndexWithProgress(ctx, nil)
}

// UpdateIndexWithProgress is UpdateIndex with optional progress callbacks.
func (u *Updater) UpdateIndexWithProgress(ctx context.Context, progress chan<- Progress) (UpdateResult, error) {
	return u.syncInstalled(ctx, u.ParseOnUpdate, progress)
}

func (u *Updater) syncInstalled(ctx context.Context, parseChanged bool, progress chan<- Progress) (UpdateResult, error) {
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

	progs, err := u.Scanner.ScanInstalled(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	res := UpdateResult{InstalledSeen: len(progs)}
	keep := make([]string, 0, len(progs))
	var toParse []string

	for i, ip := range progs {
		keep = append(keep, ip.Name)
		existing, err := u.Store.GetProgramByName(ip.Name)
		resetMan := false
		isNew := false
		if err != nil {
			isNew = true
		} else if existing.Version != "" && ip.Version != "" && existing.Version != ip.Version {
			resetMan = true
			res.Updated++
		} else if !existing.IsInstalled {
			res.Updated++
		}

		summary := ip.Summary
		if summary == "" && isNew {
			if s, err := u.Parser.Summary(ctx, ip.Name); err == nil {
				summary = s
			}
		}

		id, err := u.Store.UpsertProgram(db.Program{
			Name:        ip.Name,
			PackageName: ip.PackageName,
			Summary:     summary,
			Version:     ip.Version,
			IsInstalled: true,
		}, resetMan)
		if err != nil {
			return res, err
		}
		_ = id
		if isNew {
			res.Added++
		}
		if parseChanged && (isNew || resetMan) {
			toParse = append(toParse, ip.Name)
		}
		send(Progress{Phase: "upsert", Done: i + 1, Total: len(progs), Message: ip.Name})
	}

	n, err := u.Store.MarkNotInstalledExcept(keep)
	if err != nil {
		return res, err
	}
	res.MarkedGone = int(n)

	if parseChanged && len(toParse) > 0 {
		parsed, err := u.parseMany(ctx, toParse, send)
		res.Parsed = parsed
		if err != nil {
			return res, err
		}
	}
	send(Progress{Phase: "done", Message: fmt.Sprintf("Оновлено %d пакетів", res.Added+res.Updated+res.MarkedGone)})
	return res, nil
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
			continue
		}
		if err := u.applyParsedOptions(ctx, pr.Command, pr.Options); err != nil {
			// Soft-fail individual man pages (same policy as sequential ParseProgram).
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

// CatalogSyncResult summarises a repo-catalog sync into programs (Phase 2 / D5).
type CatalogSyncResult struct {
	Added  int                 `json:"added"`
	Status scanner.CacheStatus `json:"status"`
	Hint   string              `json:"hint"`
	Total  int                 `json:"total"`
}

// SyncRepoCatalog upserts distribution packages as programs without demoting installed rows.
// Writes go through transactional batch upsert with deferred FTS (H4).
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
