package doctor

import (
	"context"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sysinfo"
)

func TestRunAllChecksReturns15Results(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	osInfo := &sysinfo.OSInfo{
		OS:             "linux",
		Arch:           "amd64",
		Family:         "linux",
		PackageManager: "nix",
		HasNix:         true,
	}

	results := RunAllChecks(ctx, osInfo)
	if len(results) != 15 {
		t.Errorf("RunAllChecks returned %d results, want 15", len(results))
	}
}

func TestRunAllChecksPopulatesToolNames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	osInfo := &sysinfo.OSInfo{
		OS:     "linux",
		Arch:   "amd64",
		Family: "linux",
	}

	results := RunAllChecks(ctx, osInfo)
	names := make(map[string]bool)
	for _, r := range results {
		if r.Name == "" {
			t.Error("got result with empty Name")
		}
		names[r.Name] = true
	}

	// Verify a few expected names are present
	for _, expected := range []string{"git", "go", "node", "npm", "nix"} {
		if !names[expected] {
			t.Errorf("expected tool %q not in results", expected)
		}
	}
}

func TestRunSingleCheckNotFound(t *testing.T) {
	ctx := context.Background()
	tc := ToolCheck{
		Name:        "nonexistent-tool-xyz",
		Binary:      "nonexistent-tool-xyz-12345",
		VersionFlag: "--version",
		Required:    true,
		ParseVersion: func(raw string) string {
			return raw
		},
		AutoInstall: func(_ *sysinfo.OSInfo) bool { return true },
	}

	osInfo := &sysinfo.OSInfo{OS: "linux", Family: "linux"}
	status := runSingleCheck(ctx, tc, osInfo)

	if status.Installed {
		t.Error("expected nonexistent tool to not be installed")
	}
	if !status.AutoInstallable {
		t.Error("expected AutoInstallable to be true")
	}
	if status.Name != "nonexistent-tool-xyz" {
		t.Errorf("Name = %q, want %q", status.Name, "nonexistent-tool-xyz")
	}
}
