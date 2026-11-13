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
func New(store *db.Store) *Updater {
	return &Updater{
		Store:   store,
		Scanner: scanner.New(),
		Parser:  manparser.New(),
		Workers: 4,
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
		send(Progress{Phase: "parse", Total: len(toParse), Message: "Парсинг man"})
		for i, name := range toParse {
			if err := u.ParseProgram(ctx, name); err != nil {
				// Soft-fail individual man pages.
				send(Progress{Phase: "parse", Done: i + 1, Total: len(toParse), Message: name + ": " + err.Error()})
				continue
			}
			res.Parsed++
			send(Progress{Phase: "parse", Done: i + 1, Total: len(toParse), Message: name})
		}
	}
	send(Progress{Phase: "done", Message: fmt.Sprintf("Оновлено %d пакетів", res.Added+res.Updated+res.MarkedGone)})
	return res, nil
}

// ParseProgram parses man on-demand and stores man_commands (P1-14).
func (u *Updater) ParseProgram(ctx context.Context, name string) error {
	p, err := u.Store.GetProgramByName(name)
	if err != nil {
		return err
	}
	if !p.IsInstalled {
		return fmt.Errorf("index: man parse skipped for uninstalled %q (D6)", name)
	}
	opts, err := u.Parser.ParseOptions(ctx, name)
	if err != nil {
		return err
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
