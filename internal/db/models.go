package db

// Man status values (D13.10 / D13.11).
const (
	ManStatusUnknown = "unknown"
	ManStatusAbsent  = "absent"
	ManStatusParsed  = "parsed"
)

// Program is a row in the programs table (ТЗ §3 + D13).
type Program struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	IsInstalled bool   `json:"isInstalled"`
	ManParsed   bool   `json:"manParsed"`
	IsFavorite  bool   `json:"isFavorite"`
	// Category is the raw PM Section/Group (D13.8).
	Category string `json:"category"`
	// ManStatus is unknown | absent | parsed (D13.10).
	ManStatus string `json:"manStatus"`
}

// ManCommand is a parsed man flag/command row.
type ManCommand struct {
	ID          int64  `json:"id"`
	ProgramID   int64  `json:"programId"`
	Flag        string `json:"flag"`
	Description string `json:"description"`
}

// UserSnippet is a user-owned command snippet (CRUD in Phase 3).
type UserSnippet struct {
	ID        int64  `json:"id"`
	ProgramID int64  `json:"programId"`
	Title     string `json:"title"`
	Command   string `json:"command"`
	Tags      string `json:"tags"`
	CreatedAt string `json:"createdAt"`
}

// CustomScript is a saved bash script row (Phase 4 / P4-7).
type CustomScript struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ScriptBody string `json:"scriptBody"`
	UpdatedAt  string `json:"updatedAt"`
}

// SearchHit is an FTS5 match with program columns needed by the navigator (M4: no N+1).
type SearchHit struct {
	ProgramID   int64   `json:"programId"`
	Name        string  `json:"name"`
	PackageName string  `json:"packageName"`
	Summary     string  `json:"summary"`
	Version     string  `json:"version"`
	IsInstalled bool    `json:"isInstalled"`
	ManParsed   bool    `json:"manParsed"`
	IsFavorite  bool    `json:"isFavorite"`
	Category    string  `json:"category"`
	ManStatus   string  `json:"manStatus"`
	Rank        float64 `json:"rank"`
}
