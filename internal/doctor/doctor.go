// Package doctor provides tool prerequisite detection, version checking,
// and formatted reporting for the "qsdev doctor" command.
package doctor

import (
	"context"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// RunChecks runs checks in parallel and returns one ToolStatus per check, in
// order.
func RunChecks(ctx context.Context, osInfo *sysinfo.OSInfo, checks []ToolCheck) []ToolStatus {
	results := make([]ToolStatus, len(checks))

	var wg sync.WaitGroup
	for i, tc := range checks {
		wg.Go(func() {
			results[i] = runSingleCheck(ctx, tc, osInfo)
		})
	}
	wg.Wait()

	return results
}

// runSingleCheck detects one tool and builds a ToolStatus from the result.
func runSingleCheck(ctx context.Context, tc ToolCheck, osInfo *sysinfo.OSInfo) ToolStatus {
	info := detect(ctx, tc.Binary, tc.VersionFlag)

	// Try alternative binaries if the primary was not found
	if !info.Found && len(tc.AltBinaries) > 0 {
		for _, alt := range tc.AltBinaries {
			info = detect(ctx, alt, tc.VersionFlag)
			if info.Found {
				break
			}
		}
	}

	status := ToolStatus{
		Name:       tc.Name,
		Required:   tc.Required,
		RequiredBy: tc.RequiredBy,
	}

	if !info.Found {
		if tc.AutoInstall != nil {
			status.AutoInstallable = tc.AutoInstall(osInfo)
		}
		if tc.Notes != nil {
			status.Notes = tc.Notes(osInfo)
		}
		return status
	}

	status.Installed = true
	status.Path = info.Path

	// Parse version from the full output; some tools print it on a later
	// line or as JSON.
	if tc.ParseVersion != nil && info.Output != "" {
		status.Version = tc.ParseVersion(info.Output)
	}
	// A binary inside the project was not run; metadata beside it may still
	// say its version.
	status.InProject = info.InProject
	if info.InProject && tc.ProjectVersion != nil {
		status.Version = tc.ProjectVersion(info.Path)
	}

	// With no floor any installed version is OK; with one, a version that
	// could not be determined is not.
	status.MinVersion = tc.MinVersion
	status.VersionOK = toolcheck.MeetsMinimum(status.Version, tc.MinVersion)

	if tc.AutoInstall != nil {
		status.AutoInstallable = tc.AutoInstall(osInfo)
	}
	if !status.VersionOK && tc.UpgradeHint != "" {
		// Setup would reinstall, not upgrade.
		status.AutoInstallable = false
		status.UpgradeHint = tc.UpgradeHint
	}
	if tc.Notes != nil {
		status.Notes = tc.Notes(osInfo)
	}

	return status
}

// detect finds binary on PATH and, unless versionFlag is empty (a
// lookup-only check), runs it with versionFlag for its version output.
func detect(ctx context.Context, binary, versionFlag string) toolcheck.Info {
	if versionFlag == "" {
		path, err := toolcheck.LookPath(binary)
		if err != nil {
			return toolcheck.Info{}
		}
		return toolcheck.Info{Found: true, Path: path}
	}
	return toolcheck.Detect(ctx, binary, versionFlag)
}
