package check

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// fakeProber returns a ToolProber that reports info for every binary and
// records the binaries it was asked about.
func fakeProber(info toolcheck.Info, probed *[]string) ToolProber {
	return func(binary, _ string) toolcheck.Info {
		*probed = append(*probed, binary)
		return info
	}
}

// TestCheckSecurityHardening_ToolchainRequirements verifies W054: an npm
// project fails the check when the npm on PATH predates min-release-age (npm
// 10 reads the .npmrc key as an unknown string and ignores it), and the probe
// is derived from the module, so pnpm projects are not probed for npm.
func TestCheckSecurityHardening_ToolchainRequirements(t *testing.T) {
	t.Parallel()
	npm := func(version string) toolcheck.Info {
		return toolcheck.Info{Found: true, Path: "/nix/store/x-npm/bin/npm", Version: version, Output: version + "\n"}
	}
	tests := []struct {
		name         string
		lang         types.LanguageConfig
		noProber     bool
		info         toolcheck.Info
		wantStatus   CheckStatus // "" means no toolchain result expected
		wantSeverity CheckSeverity
	}{
		{"npm 10 ignores the age gate", types.LanguageConfig{Name: "javascript", PackageManager: "npm"}, false, npm("10.9.8"), StatusFail, SeverityHigh},
		{"npm 11 before min-release-age", types.LanguageConfig{Name: "javascript", PackageManager: "npm"}, false, npm("11.9.0"), StatusFail, SeverityHigh},
		{"npm 11.10 knows the age gate", types.LanguageConfig{Name: "javascript", PackageManager: "npm"}, false, npm("11.10.0"), StatusPass, SeverityInfo},
		{"npm 11.19", types.LanguageConfig{Name: "javascript", PackageManager: "npm"}, false, npm("11.19.0"), StatusPass, SeverityInfo},
		{"default package manager is npm", types.LanguageConfig{Name: "javascript"}, false, npm("10.9.8"), StatusFail, SeverityHigh},
		{"npm not on PATH", types.LanguageConfig{Name: "javascript"}, false, toolcheck.Info{}, StatusSkip, SeverityInfo},
		{"unreadable version", types.LanguageConfig{Name: "javascript"}, false, npm("unknown"), StatusWarn, SeverityLow},
		{"no prober configured", types.LanguageConfig{Name: "javascript"}, true, toolcheck.Info{}, StatusSkip, SeverityInfo},
		{"pnpm is not probed for npm", types.LanguageConfig{Name: "javascript", PackageManager: "pnpm"}, false, npm("10.9.8"), "", ""},
		{"module without requirements", types.LanguageConfig{Name: "go"}, false, npm("10.9.8"), "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var probed []string
			ctx := CheckContext{
				ProjectRoot: t.TempDir(),
				QsdevConfig: &types.QsdevConfig{Languages: []types.LanguageConfig{tt.lang}},
			}
			if !tt.noProber {
				ctx.ProbeTool = fakeProber(tt.info, &probed)
			}

			r := findResult(CheckSecurityHardening(ctx), "toolchain_javascript_npm")
			if tt.wantStatus == "" {
				if r != nil || len(probed) > 0 {
					t.Fatalf("expected no npm probe, got result %+v (probed %v)", r, probed)
				}
				return
			}
			if r == nil {
				t.Fatal("toolchain_javascript_npm result missing")
			}
			if r.Status != tt.wantStatus || r.Severity != tt.wantSeverity {
				t.Errorf("status/severity = %s/%s, want %s/%s (%s)", r.Status, r.Severity, tt.wantStatus, tt.wantSeverity, r.Message)
			}
			if r.Category != CategorySecurityHarden {
				t.Errorf("category = %s, want %s", r.Category, CategorySecurityHarden)
			}
			if r.Status == StatusFail && r.Remediation == "" {
				t.Error("failing result has no remediation")
			}
		})
	}
}
