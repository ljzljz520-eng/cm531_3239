package domain

type DDLStatement struct {
	Kind       string   `json:"kind"`
	Table      string   `json:"table"`
	SQL        string   `json:"sql"`
	Warnings   []string `json:"warnings"`
	SourceHash string   `json:"source_hash"`
}

type MappingSummary struct {
	Tables          int `json:"tables"`
	Columns         int `json:"columns"`
	Indexes         int `json:"indexes"`
	Warnings        int `json:"warnings"`
	PersistedOrders int `json:"persisted_orders"`
	PersistedAssets int `json:"persisted_assets"`
	PersistedChecks int `json:"persisted_checks"`
	PersistedStock  int `json:"persisted_stock"`
}

type Preview struct {
	RequestKey string         `json:"request_key"`
	Statements []DDLStatement `json:"statements"`
	Summary    MappingSummary `json:"summary"`
	Warnings   []string       `json:"warnings"`
}

type Download struct {
	ID        string `json:"id"`
	RecordID  string `json:"record_id"`
	Format    string `json:"format"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"created_at"`
}

type AuditEvent struct {
	ID         string            `json:"id"`
	RecordID   string            `json:"record_id"`
	Action     string            `json:"action"`
	Actor      string            `json:"actor"`
	Metadata   map[string]string `json:"metadata"`
	OccurredAt string            `json:"occurred_at"`
}
