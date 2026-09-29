package outdated

// OutdatedOptions configures the outdated check.
type OutdatedOptions struct {
	Ecosystem string // Filter to a single ecosystem, empty means all
	// PackageManagers maps an ecosystem to the package manager the project is
	// configured with (e.g. javascript -> pnpm), so the matching tool runs even
	// when another manager for the same ecosystem comes first on PATH.
	PackageManagers map[string]string
}

// EcosystemCommand maps an ecosystem to its native outdated command.
type EcosystemCommand struct {
	Ecosystem       string
	Binary          string
	Args            []string
	OutdatedOnExit1 bool // true if exit code 1 means "outdated found" (npm behavior)
	// Markers are project-root files (lockfiles, build files) whose presence
	// shows the project is managed by this command's tool.
	Markers []string
	// Unsupported, when set, returns why the tool cannot check the project at
	// projectRoot, or "" when it can.
	Unsupported func(projectRoot string) string
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

// FailedEcosystems returns the ecosystems whose outdated command failed (a
// non-zero exit that is not the tool's "outdated found" signal, a timeout, or a
// command that could not start). Their outdated status is unknown.
func (r *OutdatedResult) FailedEcosystems() []string {
	var failed []string
	for _, e := range r.Ecosystems {
		if e.Error != nil {
			failed = append(failed, e.Name)
		}
	}
	return failed
}
