package outdated

// OutdatedOptions configures the outdated check.
type OutdatedOptions struct {
	Ecosystem string // Filter to a single ecosystem, empty means all
}

// EcosystemCommand maps an ecosystem to its native outdated command.
type EcosystemCommand struct {
	Ecosystem       string
	Binary          string
	Args            []string
	OutdatedOnExit1 bool // true if exit code 1 means "outdated found" (npm behavior)
}

// EcosystemCheck holds the result of running one ecosystem's outdated command.
type EcosystemCheck struct {
	Name        string
	Command     string
	HasOutdated bool
	Skipped     bool
	SkipReason  string
	ExitCode    int
	Error       error
}

// OutdatedResult holds the results for all checked ecosystems.
type OutdatedResult struct {
	Ecosystems []EcosystemCheck
}

// HasAnyOutdated returns true if any ecosystem has outdated packages.
func (r *OutdatedResult) HasAnyOutdated() bool {
	for _, e := range r.Ecosystems {
		if e.HasOutdated {
			return true
		}
	}
	return false
}
