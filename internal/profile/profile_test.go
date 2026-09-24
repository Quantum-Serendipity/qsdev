package profile

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestEnvironmentVars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    *InfraProfile
		want map[string]string
	}{
		{"turborepo remote cache", &InfraProfile{BuildCache: BuildCacheConfig{Type: BuildCacheTurborepo, URL: "https://turbo.corp.internal"}},
			map[string]string{"TURBO_API": "https://turbo.corp.internal"}},
		{"turborepo without a URL uses the default cache", &InfraProfile{BuildCache: BuildCacheConfig{Type: BuildCacheTurborepo}}, map[string]string{}},
		// Credentials are never set: a devenv env entry would replace the
		// developer's real secret with a literal "${VAR}" string.
		{"sccache credentials are not set", Enterprise, map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.p.EnvironmentVars(); !maps.Equal(got, tt.want) {
				t.Errorf("EnvironmentVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCredentialEnvVars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		p    *InfraProfile
		want []string
	}{
		{"consulting-default", ConsultingDefault, []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "CACHIX_AUTH_TOKEN", "NEXUS_TOKEN", "SCCACHE_BUCKET"}},
		{"enterprise adds snyk", Enterprise, []string{"ARTIFACTORY_TOKEN", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "CACHIX_AUTH_TOKEN", "SCCACHE_BUCKET", "SNYK_TOKEN"}},
		{"components switched off", &InfraProfile{Registry: RegistryConfig{Type: RegistryNone, AuthEnvVar: "NEXUS_TOKEN"},
			NixCache: NixCacheConfig{Type: NixCacheNone, PushTokenEnvVar: "CACHIX_AUTH_TOKEN"}, BuildCache: BuildCacheConfig{Type: BuildCacheNone, AuthEnvVars: []string{"X"}}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.p.CredentialEnvVars(); !slices.Equal(got, tt.want) {
				t.Errorf("CredentialEnvVars() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsultingDefault_ConfigFiles_Renovate(t *testing.T) {
	files := mustConfigFiles(t, ConsultingDefault, ProjectInputs{})
	found := false
	for _, f := range files {
		if f.Path == "renovate.json" {
			found = true
			content := string(f.Content)
			if !strings.Contains(content, "config:recommended") {
				t.Error("renovate.json missing extends config:recommended")
			}
			if !strings.Contains(content, "3 days") {
				t.Error("renovate.json missing 3-day age gate")
			}
			if !strings.Contains(content, `"automerge"`) {
				t.Error("renovate.json missing automerge rule")
			}
		}
	}
	if !found {
		t.Error("ConsultingDefault.ConfigFiles() did not produce renovate.json")
	}
}

func TestStartupGitHub_ConfigFiles_Dependabot(t *testing.T) {
	files := mustConfigFiles(t, StartupGitHub, ProjectInputs{Ecosystems: []string{"npm"}})
	foundDependabot := false
	foundRenovate := false
	for _, f := range files {
		if f.Path == ".github/dependabot.yml" {
			foundDependabot = true
			content := string(f.Content)
			if !strings.Contains(content, "version: 2") {
				t.Error("dependabot.yml missing version: 2")
			}
			if !strings.Contains(content, "npm") {
				t.Error("dependabot.yml missing npm ecosystem")
			}
		}
		if f.Path == "renovate.json" {
			foundRenovate = true
		}
	}
	if !foundDependabot {
		t.Error("StartupGitHub.ConfigFiles() did not produce .github/dependabot.yml")
	}
	if foundRenovate {
		t.Error("StartupGitHub.ConfigFiles() should not produce renovate.json")
	}
}

func TestConsultingDefault_ConfigFiles_IncludesWorkflow(t *testing.T) {
	files := mustConfigFiles(t, ConsultingDefault, ProjectInputs{})
	found := false
	for _, f := range files {
		if f.Path == ".github/workflows/security-scan.yml" {
			found = true
			content := string(f.Content)
			if !strings.Contains(content, "Security Scan") {
				t.Error("workflow should contain 'Security Scan' name")
			}
			if !strings.Contains(content, "OSV Scanner") {
				t.Error("consulting-default workflow should contain OSV Scanner")
			}
		}
	}
	if !found {
		t.Error("ConsultingDefault.ConfigFiles() did not produce .github/workflows/security-scan.yml")
	}
}

func TestConsultingDefault_ConfigFiles_IncludesSecurityDoc(t *testing.T) {
	files := mustConfigFiles(t, ConsultingDefault, ProjectInputs{})
	found := false
	for _, f := range files {
		if f.Path == "docs/security-overview.md" {
			found = true
			content := string(f.Content)
			if !strings.Contains(content, "consulting-default") {
				t.Error("security doc should mention consulting-default profile")
			}
		}
	}
	if !found {
		t.Error("ConsultingDefault.ConfigFiles() did not produce docs/security-overview.md")
	}
}

func TestEnterprise_ConfigFiles_IncludesWorkflow(t *testing.T) {
	files := mustConfigFiles(t, Enterprise, ProjectInputs{})
	foundWorkflow := false
	foundSecDoc := false
	for _, f := range files {
		if f.Path == ".github/workflows/security-scan.yml" {
			foundWorkflow = true
			content := string(f.Content)
			if !strings.Contains(content, "Snyk") {
				t.Error("enterprise workflow should contain Snyk")
			}
		}
		if f.Path == "docs/security-overview.md" {
			foundSecDoc = true
		}
	}
	if !foundWorkflow {
		t.Error("Enterprise.ConfigFiles() did not produce .github/workflows/security-scan.yml")
	}
	if !foundSecDoc {
		t.Error("Enterprise.ConfigFiles() did not produce docs/security-overview.md")
	}
}

func TestNixCacheNixConfig(t *testing.T) {
	t.Parallel()
	const key = "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
	tests := []struct {
		name      string
		cache     NixCacheConfig
		wantSubst string
		wantKey   string
	}{
		{"cachix URL", NixCacheConfig{Type: NixCacheCachix, URL: "https://corp.cachix.org", PublicKey: key}, "https://corp.cachix.org", key},
		{"cachix name", NixCacheConfig{Type: NixCacheCachix, CacheName: "corp", PublicKey: key}, "https://corp.cachix.org", key},
		{"attic", NixCacheConfig{Type: NixCacheAttic, URL: "https://attic.corp.internal/main", PublicKey: key}, "https://attic.corp.internal/main", key},
		{"unconfigured cachix", NixCacheConfig{Type: NixCacheCachix, PublicKey: key}, "", ""},
		{"none", NixCacheConfig{Type: NixCacheNone, URL: "https://corp.cachix.org", PublicKey: key}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &InfraProfile{NixCache: tt.cache}
			subst, k := p.NixCacheNixConfig()
			if subst != tt.wantSubst || k != tt.wantKey {
				t.Errorf("NixCacheNixConfig() = %q, %q; want %q, %q", subst, k, tt.wantSubst, tt.wantKey)
			}
		})
	}
}
