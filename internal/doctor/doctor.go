// Package doctor provides tool prerequisite detection, version checking,
// and formatted reporting for the "qsdev doctor" command.
package doctor

import (
	"context"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// RunAllChecks runs all 15 tool checks in parallel and returns the results.
func RunAllChecks(ctx context.Context, osInfo *sysinfo.OSInfo) []ToolStatus {
	checks := DefaultChecks()
	results := make([]ToolStatus, len(checks))

	var wg sync.WaitGroup
	wg.Add(len(checks))
	for i, tc := range checks {
		go func(idx int, check ToolCheck) {
			defer wg.Done()
			results[idx] = runSingleCheck(ctx, check, osInfo)
		}(i, tc)
	}
	wg.Wait()

	return results
}

// runSingleCheck detects one tool and builds a ToolStatus from the result.
func runSingleCheck(ctx context.Context, tc ToolCheck, osInfo *sysinfo.OSInfo) ToolStatus {
	info := toolcheck.Detect(ctx, tc.Binary, tc.VersionFlag)

	// Try alternative binaries if the primary was not found
	if !info.Found && len(tc.AltBinaries) > 0 {
		for _, alt := range tc.AltBinaries {
			info = toolcheck.Detect(ctx, alt, tc.VersionFlag)
			if info.Found {
				break
			}
		}
	}

	status := ToolStatus{
		Name:     tc.Name,
		Required: tc.Required,
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

	// Parse version from raw output
	if tc.ParseVersion != nil && info.Version != "" {
		status.Version = tc.ParseVersion(info.Version)
	}

	// Check minimum version
	if tc.MinVersion != "" {
		status.MinVersion = tc.MinVersion
		if status.Version != "" {
			status.VersionOK = MeetsMinimum(status.Version, tc.MinVersion)
		}
	} else {
		// No minimum version requirement — if installed, version is OK
		status.VersionOK = true
	}

	if tc.AutoInstall != nil {
		status.AutoInstallable = tc.AutoInstall(osInfo)
	}
	if tc.Notes != nil {
		status.Notes = tc.Notes(osInfo)
	}

	return status
}
