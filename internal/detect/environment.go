package detect

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
)

// EnvironmentState captures existing configuration and tooling state
// found in a project directory. This is a type alias for the canonical
// definition in the ecosystem package.
type EnvironmentState = ecosystem.EnvironmentState

// detectEnvironment scans the project root for configuration files,
// tooling state, and git metadata. It delegates to the ecosystem package.
func detectEnvironment(root string) EnvironmentState {
	return ecosystem.DetectEnvironment(root)
}
