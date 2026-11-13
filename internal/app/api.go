// Package app exposes Wails-facing application services.
// Phase 0 provides metadata only; scanner/index/search bindings land in Phase 1–2.

package app

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

// API is the application service facade bound through the Wails App root.
type API struct{}

// New returns a Phase 0 API stub.
func New() *API {
	return &API{}
}

// Info returns fixed brand and path decisions (D1–D4).
func (a *API) Info() Info {
	return Info{
		Name:     "SnippetShow",
		Subtitle: "Linux Command Wiki & Script Builder",
		Module:   "snippetshow",
		DBPath:   "~/.local/share/linux-cmd-wiki/wiki.db",
		UILocale: "uk",
		Phase:    "0",
		Frontend: "vanilla-js",
		License:  "MIT",
	}
}

// PhaseStatus is a human-readable build stage marker for the shell UI.
func (a *API) PhaseStatus() string {
	return "Фаза 0: скелет Wails/Go без Core-логіки"
}
