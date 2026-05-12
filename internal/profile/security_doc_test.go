package profile

import (
	"strings"
	"testing"
)

func TestGenerateSecurityDoc_ConsultingDefault(t *testing.T) {
	f := ConsultingDefault.generateSecurityDoc()

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
			f := p.generateSecurityDoc()
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
			f := p.generateSecurityDoc()
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
	f := Enterprise.generateSecurityDoc()
	content := string(f.Content)

	if !strings.Contains(content, "enterprise") {
		t.Error("enterprise doc should mention enterprise profile name")
	}
	if !strings.Contains(content, "snyk") {
		t.Error("enterprise doc should mention snyk as vulnerability scanner")
	}
}

func TestGenerateSecurityDoc_DefenseLayersTable(t *testing.T) {
	f := ConsultingDefault.generateSecurityDoc()
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
