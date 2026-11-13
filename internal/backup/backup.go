// Package backup exports/imports user_snippets and custom_scripts as versioned JSON (P4-8…P4-9).
package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"snippetshow/internal/db"
)

// CurrentVersion is the backup document schema version.
const CurrentVersion = 1

// Document is the on-disk JSON backup envelope.
type Document struct {
	Version    int             `json:"version"`
	ExportedAt string          `json:"exportedAt"`
	Snippets   []SnippetRecord `json:"snippets"`
	Scripts    []ScriptRecord  `json:"scripts"`
}

// SnippetRecord is a portable user snippet (keyed by program name, not id).
type SnippetRecord struct {
	ProgramName string `json:"programName"`
	Title       string `json:"title"`
	Command     string `json:"command"`
	Tags        string `json:"tags,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

// ScriptRecord is a portable custom_scripts row.
type ScriptRecord struct {
	Name       string `json:"name"`
	ScriptBody string `json:"scriptBody"`
	UpdatedAt  string `json:"updatedAt,omitempty"`
}

// Result summarizes an import merge.
type Result struct {
	SnippetsAdded  int `json:"snippetsAdded"`
	SnippetsSkipped int `json:"snippetsSkipped"`
	ScriptsAdded   int `json:"scriptsAdded"`
	ScriptsSkipped int `json:"scriptsSkipped"`
	ProgramsCreated int `json:"programsCreated"`
}

// Export writes user_snippets + custom_scripts to path (versioned schema).
// Does not include programs/man_commands — those stay local.
func Export(store *db.Store, path string) error {
	if store == nil {
		return fmt.Errorf("backup: nil store")
	}
	sns, err := store.ListAllUserSnippets()
	if err != nil {
		return err
	}
	scripts, err := store.ListCustomScripts()
	if err != nil {
		return err
	}
	doc := Document{
		Version:    CurrentVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Snippets:   make([]SnippetRecord, 0, len(sns)),
		Scripts:    make([]ScriptRecord, 0, len(scripts)),
	}
	for _, sn := range sns {
		doc.Snippets = append(doc.Snippets, SnippetRecord{
			ProgramName: sn.ProgramName,
			Title:       sn.Title,
			Command:     sn.Command,
			Tags:        sn.Tags,
			CreatedAt:   sn.CreatedAt,
		})
	}
	for _, cs := range scripts {
		doc.Scripts = append(doc.Scripts, ScriptRecord{
			Name:       cs.Name,
			ScriptBody: cs.ScriptBody,
			UpdatedAt:  cs.UpdatedAt,
		})
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("backup: marshal: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("backup: write %s: %w", path, err)
	}
	return nil
}

// Import merges a JSON backup into the local DB without deleting programs/man.
//
// Merge policy (documented in README):
//   - Snippets: skip if identical (programName + title + command) already exists;
//     if the program is missing, create a stub program (is_installed=0).
//   - Scripts: skip if identical (name + scriptBody) already exists; otherwise insert.
//   - Re-import is idempotent for identical records.
func Import(store *db.Store, path string) (Result, error) {
	var res Result
	if store == nil {
		return res, fmt.Errorf("backup: nil store")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return res, fmt.Errorf("backup: read %s: %w", path, err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return res, fmt.Errorf("backup: parse JSON: %w", err)
	}
	if doc.Version != CurrentVersion {
		return res, fmt.Errorf("backup: unsupported version %d (want %d)", doc.Version, CurrentVersion)
	}

	for _, sn := range doc.Snippets {
		name := strings.TrimSpace(sn.ProgramName)
		title := strings.TrimSpace(sn.Title)
		command := strings.TrimSpace(sn.Command)
		if name == "" || title == "" || command == "" {
			res.SnippetsSkipped++
			continue
		}
		prog, err := store.GetProgramByName(name)
		if err != nil {
			id, upErr := store.UpsertProgram(db.Program{
				Name: name, Summary: "imported stub", IsInstalled: false,
			}, false)
			if upErr != nil {
				return res, upErr
			}
			prog, err = store.GetProgramByID(id)
			if err != nil {
				return res, err
			}
			res.ProgramsCreated++
		}
		dup, err := store.FindSnippetDuplicate(prog.ID, title, command)
		if err != nil {
			return res, err
		}
		if dup {
			res.SnippetsSkipped++
			continue
		}
		created := sn.CreatedAt
		if strings.TrimSpace(created) == "" {
			created = time.Now().UTC().Format(time.RFC3339)
		}
		if _, err := store.InsertUserSnippet(db.UserSnippet{
			ProgramID: prog.ID, Title: title, Command: command, Tags: sn.Tags, CreatedAt: created,
		}); err != nil {
			return res, err
		}
		res.SnippetsAdded++
	}

	for _, sc := range doc.Scripts {
		name := strings.TrimSpace(sc.Name)
		body := strings.TrimSpace(sc.ScriptBody)
		if name == "" || body == "" {
			res.ScriptsSkipped++
			continue
		}
		dup, err := store.FindScriptDuplicate(name, body)
		if err != nil {
			return res, err
		}
		if dup {
			res.ScriptsSkipped++
			continue
		}
		if _, err := store.CreateCustomScript(name, body); err != nil {
			return res, err
		}
		res.ScriptsAdded++
	}
	return res, nil
}
