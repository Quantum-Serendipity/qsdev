package check

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestCheckDevenvSecurityFloor pins that check fails a devenv.nix that no
// longer enables a security hook, or strips a credential variable, the
// generator emits for the project (G-01 independent check).
func TestCheckDevenvSecurityFloor(t *testing.T) {
	t.Parallel()

	expectedHooks := []string{"check-added-large-files", "lock-file-audit", "ripsecrets"}
	expectedVars := []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN"}
	without := func(list []string, drop string) []string {
		return slices.DeleteFunc(slices.Clone(list), func(s string) bool { return s == drop })
	}
	tests := []struct {
		name       string
		ctx        CheckContext
		status     CheckStatus
		severity   CheckSeverity
		wantInText []string
	}{
		{
			name: "clean generated devenv.nix",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: append(slices.Clone(expectedHooks), "gofmt"), DevenvUnsetVars: append(slices.Clone(expectedVars), "MY_TEAM_TOKEN"),
			},
			status: StatusPass,
		},
		{
			name: "ripsecrets removed or disabled",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: without(expectedHooks, "ripsecrets"), DevenvUnsetVars: expectedVars,
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"ripsecrets"},
		},
		{
			name: "AWS_SECRET_ACCESS_KEY no longer stripped",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: without(expectedVars, "AWS_SECRET_ACCESS_KEY"),
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"AWS_SECRET_ACCESS_KEY"},
		},
		{
			// The command layer scopes the expected hooks to the security
			// ones, so a hand-disabled formatter is not in them.
			name: "disabled non-security hook",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: expectedVars,
			},
			status: StatusPass,
		},
		{
			name: "deleted devenv.nix fails closed",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"ripsecrets", "GITHUB_TOKEN"},
		},
		{
			name: "unreadable devenv module fails closed",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: expectedVars,
				DevenvSecurityErr: errors.New("parsing devenv.local.nix: unsupported Nix syntax"),
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"devenv.local.nix"},
		},
		{
			name: "security hook entry replaced",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: expectedVars,
				ExpectedDevenvHookSettings: map[string]string{"git-hooks.hooks.lock-file-audit.entry": `"audit"`},
				DevenvHookSettings:         map[string]string{"git-hooks.hooks.lock-file-audit.entry": `"true"`},
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"git-hooks.hooks.lock-file-audit.entry"},
		},
		{
			name: "security hook excludes added",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: expectedVars,
				DevenvHookSettings: map[string]string{"git-hooks.hooks.ripsecrets.excludes": `[ ".*" ]`},
			},
			status:     StatusFail,
			severity:   SeverityHigh,
			wantInText: []string{"git-hooks.hooks.ripsecrets.excludes"},
		},
		{
			name: "matching hook settings",
			ctx: CheckContext{
				ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars,
				DevenvHooks: expectedHooks, DevenvUnsetVars: expectedVars,
				ExpectedDevenvHookSettings: map[string]string{"git-hooks.hooks.lock-file-audit.entry": `"audit"`},
				DevenvHookSettings:         map[string]string{"git-hooks.hooks.lock-file-audit.entry": `"audit"`},
			},
			status: StatusPass,
		},
		{
			name:   "no devenv generated",
			ctx:    CheckContext{},
			status: StatusSkip,
		},
		{
			name:   "unknown expected output",
			ctx:    CheckContext{ExpectedGenerationErr: errors.New("generator failed")},
			status: StatusWarn,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CheckDevenvSecurityFloor(tt.ctx)
			if got.Name != "devenv_security_floor" || got.Category != CategorySecurityHarden {
				t.Errorf("result %s/%s, want security_hardening/devenv_security_floor", got.Category, got.Name)
			}
			if got.Status != tt.status {
				t.Fatalf("status = %s, want %s (%s)", got.Status, tt.status, got.Message)
			}
			if tt.severity != "" && got.Severity != tt.severity {
				t.Errorf("severity = %s, want %s", got.Severity, tt.severity)
			}
			for _, want := range tt.wantInText {
				if !strings.Contains(got.Message, want) {
					t.Errorf("message %q does not name %s", got.Message, want)
				}
			}
			// 'init --update' only writes a sidecar beside a hand-edited
			// devenv.nix, so the remediation must name the overwriting update.
			if got.Status == StatusFail && !strings.Contains(got.Remediation, "update --configs-only --overwrite-modified") {
				t.Errorf("remediation %q does not say how to restore the generated file", got.Remediation)
			}
		})
	}

	report := RunAllChecks(CheckContext{ExpectedDevenvHooks: expectedHooks, ExpectedUnsetVars: expectedVars})
	found := false
	for _, r := range report.Checks {
		found = found || (r.Name == "devenv_security_floor" && r.Status == StatusFail)
	}
	if !found {
		t.Error("RunAllChecks does not report the weakened devenv.nix")
	}
}
