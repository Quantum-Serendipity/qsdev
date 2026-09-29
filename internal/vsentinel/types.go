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

type DriftReport struct {
	Manifests []DriftManifestStatus `json:"manifests"`
}

type DriftManifestStatus struct {
	Path       string       `json:"path"`
	Ecosystem  string       `json:"ecosystem"`
	DriftCount int          `json:"drift_count"`
	Drifted    []DriftEntry `json:"drifted"`
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
