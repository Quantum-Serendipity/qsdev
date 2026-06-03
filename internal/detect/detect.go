// Package detect scans a project directory to identify programming languages,
// build systems, and existing configuration state. The results are returned
// as a [types.DetectedProject] that drives the rest of the init wizard.
package detect

import (
	"context"
	"log/slog"

	"github.com/Quantum-Serendipity/qsdev/internal/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Detect scans projectRoot for language markers, lockfiles, configuration
// files, and git metadata, returning a fully populated DetectedProject.
func Detect(projectRoot string) types.DetectedProject {
	registry := ecosystem.DefaultRegistry()
	summary := registry.DetectWithEnvironment(projectRoot)

	if rtInfo, err := container.DetectDefault(context.Background()); err == nil && rtInfo.Active != container.RuntimeNone {
		summary.Project.ContainerRuntime = string(rtInfo.Active)
	}

	slog.Debug("project detection complete",
		"ecosystems", len(summary.Project.Ecosystems),
		"has_go", summary.Project.HasGoMod,
		"has_node", summary.Project.HasPackageJSON,
		"has_container", summary.Project.HasDockerfile,
		"container_runtime", summary.Project.ContainerRuntime)
	return summary.Project
}
