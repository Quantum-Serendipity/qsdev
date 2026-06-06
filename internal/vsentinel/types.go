package vsentinel

import "time"

type VersionReport struct {
	Manifests     []ManifestStatus
	StaleCount    int
	LastCheckTime time.Time
}

type ManifestStatus struct {
	Path         string
	Ecosystem    string
	Dependencies []DepStatus
}

type DepStatus struct {
	Name            string
	DeclaredVersion string
	LockedVersion   string
	LatestKnown     string
	StaleDays       int
	DriftDetected   bool
}

type DriftReport struct {
	Manifests []DriftManifestStatus
}

type DriftManifestStatus struct {
	Path       string
	Ecosystem  string
	DriftCount int
	Drifted    []DriftEntry
}

type DriftEntry struct {
	Name            string
	DeclaredVersion string
	LockedVersion   string
}

type VersionEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Ecosystem  string    `json:"ecosystem"`
	Package    string    `json:"package"`
	OldVersion string    `json:"old_version"`
	NewVersion string    `json:"new_version"`
	Source     string    `json:"source"`
}
