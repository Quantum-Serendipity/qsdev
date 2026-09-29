package profile

import (
	"strings"
	"testing"
)

func TestGenerateSecurityDoc_ConsultingDefault(t *testing.T) {
	f := mustSecurityDoc(t, ConsultingDefault)

	if f.Path != "docs/security-overview.md" {
		t.Errorf("Path = %q, want docs/security-overview.md", f.Path)
	}

	content := string(f.Content)

	// ConsultingDefault uses OSV + Socket.
	if !strings.Contains(content, "osv") {
		t.Error("consulting-default doc should mention osv")
	}
	if !strings.Contains(content, "socket") {
		t.Error("consulting-default doc should mention socket")
	}
}

func TestGenerateSecurityDoc_TrivyCompromiseWarning(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			f := mustSecurityDoc(t, p)
			content := string(f.Content)

			if !strings.Contains(content, "Trivy") {
				t.Error("security doc should include Trivy compromise warning")
			}
			if !strings.Contains(content, "trivy-action") {
				t.Error("security doc should mention trivy-action compromise")
			}
		})
	}
}

func TestGenerateSecurityDoc_SocketNotPhylum(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			f := mustSecurityDoc(t, p)
			content := string(f.Content)

			if !strings.Contains(content, "Socket") {
				t.Error("security doc should mention Socket.dev")
			}
			if strings.Contains(content, "Phylum") {
				t.Error("security doc should not mention Phylum (acquired by Veracode Jan 2025)")
			}
		})
	}
}

func TestGenerateSecurityDoc_ProfileSpecificContent(t *testing.T) {
	f := mustSecurityDoc(t, Enterprise)
	content := string(f.Content)

	if !strings.Contains(content, "enterprise") {
		t.Error("enterprise doc should mention enterprise profile name")
	}
	if !strings.Contains(content, "snyk") {
		t.Error("enterprise doc should mention snyk as vulnerability scanner")
	}
}

// Regression: the doc labelled the Renovate/Dependabot PR delay as the
// install-time age gate (so startup-github, with no PR delay, was documented
// as having age gating disabled) and asserted project-level controls as
// Enabled regardless of the project's security settings.
func TestGenerateSecurityDoc_DoesNotConflateProfileAndProjectControls(t *testing.T) {
	t.Parallel()
	for _, p := range []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise} {
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			content := string(mustSecurityDoc(t, p).Content)
			for _, stale := range []string{
				"Block packages newer than",
				"| Enabled (per-ecosystem config) |",
				"integrity | Enabled |",
			} {
				if strings.Contains(content, stale) {
					t.Errorf("doc still contains %q", stale)
				}
			}
			for _, setting := range []string{"security.age_gating", "security.script_blocking", "security.lock_enforcement"} {
				if !strings.Contains(content, setting) {
					t.Errorf("doc should attribute a layer to the project setting %s", setting)
				}
			}
			if !strings.Contains(content, "Dependency update PR delay") {
				t.Error("doc should report the update-tool delay as a dependency update PR delay")
			}
		})
	}
}

func TestGenerateSecurityDoc_PRDelayReflectsProfile(t *testing.T) {
	t.Parallel()
	enterprise := string(mustSecurityDoc(t, Enterprise).Content)
	if !strings.Contains(enterprise, "waits 7 day(s)") {
		t.Error("enterprise doc should report its 7-day update PR delay")
	}
	startup := string(mustSecurityDoc(t, StartupGitHub).Content)
	if !strings.Contains(startup, "none (dependabot proposes releases immediately)") {
		t.Error("startup-github doc should report no update PR delay")
	}
}

// Regression: template failures used to become comment-only files written
// with the Overwrite strategy. Every built-in profile must render real files.
func TestConfigFiles_BuiltinsRenderWithoutErrorStubs(t *testing.T) {
	t.Parallel()
	for _, p := range []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise} {
		t.Run(p.Name, func(t *testing.T) {
			t.Parallel()
			for _, f := range mustConfigFiles(t, p, ProjectInputs{Ecosystems: []string{"go"}}) {
				if strings.HasPrefix(string(f.Content), "# Error") {
					t.Errorf("%s rendered as an error stub: %q", f.Path, f.Content)
				}
			}
		})
	}
}

func TestSecurityTemplates_ExecuteErrorsAreReturned(t *testing.T) {
	t.Parallel()
	// Executing against data without the expected fields must fail loudly;
	// the generators propagate this as an error rather than writing a stub.
	var buf strings.Builder
	if err := securityDocTmpl.Execute(&buf, struct{}{}); err == nil {
		t.Error("security overview template executed against empty data without error")
	}
	buf.Reset()
	if err := securityScanWorkflowTmpl.Execute(&buf, struct{}{}); err == nil {
		t.Error("security-scan workflow template executed against empty data without error")
	}
}

func TestGenerateSecurityDoc_DefenseLayersTable(t *testing.T) {
	f := mustSecurityDoc(t, ConsultingDefault)
	content := string(f.Content)

	requiredLayers := []string{
		"Age-Gating",
		"Script Blocking",
		"Lock Files",
		"Vulnerability Scanning",
		"CI Protection",
		"Behavioral Analysis",
	}

	for _, layer := range requiredLayers {
		if !strings.Contains(content, layer) {
			t.Errorf("security doc should mention defense layer %q", layer)
		}
	}
}
