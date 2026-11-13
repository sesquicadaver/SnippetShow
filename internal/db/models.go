package db

// Program is a row in the programs table (ТЗ §3).
type Program struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
	Summary     string `json:"summary"`
	Version     string `json:"version"`
	IsInstalled bool   `json:"isInstalled"`
	ManParsed   bool   `json:"manParsed"`
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

// SearchHit is an FTS5 match with the owning program id.
type SearchHit struct {
	ProgramID int64   `json:"programId"`
	Name      string  `json:"name"`
	Summary   string  `json:"summary"`
	Rank      float64 `json:"rank"`
}
