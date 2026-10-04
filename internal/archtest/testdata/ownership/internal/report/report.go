package report

// Log is a private SARIF document.
type Log struct {
	Schema string `json:"$schema,omitempty"`
	Runs   []any  `json:"runs"`
}

// Renovate has a $schema but is not SARIF.
type Renovate struct {
	Schema string `json:"$schema"`
}
