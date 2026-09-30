package vsentinel

import "time"

// VersionReport is the check_versions result: the dependencies each manifest in
// the project root declares. It reports declarations only; it makes no claim
// about whether a dependency is stale.
type VersionReport struct {
	Manifests     []ManifestStatus `json:"manifests"`
	LastCheckTime time.Time        `json:"last_check_time"`
}

type ManifestStatus struct {
	Path         string      `json:"path"`
	Ecosystem    string      `json:"ecosystem"`
	Dependencies []DepStatus `json:"dependencies"`
}

// DepStatus is one declared dependency and the version requirement the
// manifest states for it (empty when it states none).
type DepStatus struct {
	Name            string `json:"name"`
	DeclaredVersion string `json:"declared_version"`
}

// DriftReport is the detect_drift result. A zero drift count is a verified
// clean only when Coverage is CoverageComplete; NotVerified names every
// manifest whose dependencies were not version-diffed against a lockfile.
type DriftReport struct {
	Manifests   []DriftManifestStatus `json:"manifests"`
	Coverage    string                `json:"coverage"`
	NotVerified []string              `json:"not_verified,omitempty"`
}

// Report coverage verdicts. CoverageNone means no manifest was found to
// check, which is neither a verified clean nor a partial check.
const (
	CoverageComplete = "complete"
	CoveragePartial  = "partial"
	CoverageNone     = "none"
)

// Verification is how thoroughly a manifest was checked.
type Verification string

const (
	// VerificationDiffed: a parser read the manifest and either diffed it
	// against its lockfile or found it declares no dependencies.
	VerificationDiffed Verification = "diffed"
	// VerificationPresenceOnly: only the presence (or absence) of a lockfile
	// was checked; declared versions were not compared with locked ones.
	VerificationPresenceOnly Verification = "presence-only"
	// VerificationUncovered: no drift check exists for the manifest; neither
	// its versions nor its lockfile's presence are verified.
	VerificationUncovered Verification = "uncovered"
)

type DriftManifestStatus struct {
	Path         string       `json:"path"`
	Ecosystem    string       `json:"ecosystem"`
	Verification Verification `json:"verification"`
	DriftCount   int          `json:"drift_count"`
	Drifted      []DriftEntry `json:"drifted"`
}

type DriftEntry struct {
	Name            string `json:"name"`
	DeclaredVersion string `json:"declared_version"`
	LockedVersion   string `json:"locked_version"`
}

type VersionEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Ecosystem  string    `json:"ecosystem"`
	Package    string    `json:"package"`
	OldVersion string    `json:"old_version"`
	NewVersion string    `json:"new_version"`
	Source     string    `json:"source"`
}
