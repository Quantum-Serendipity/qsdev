// Package detect scans a project directory to identify programming languages,
// build systems, and existing configuration state. The results are returned
// as a [types.DetectedProject] that drives the rest of the init wizard.
package detect

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Detect scans projectRoot for language markers, lockfiles, configuration
// files, and git metadata, returning a fully populated DetectedProject.
func Detect(projectRoot string) types.DetectedProject {
	registry := ecosystem.DefaultRegistry()
	summary := registry.DetectWithEnvironment(projectRoot)
	return summary.Project
}
