package profile

import (
	"strings"
	"testing"
)

func TestConsultingDefault_EnvironmentVars(t *testing.T) {
	env := ConsultingDefault.EnvironmentVars()

	required := []string{
		"NPM_CONFIG_REGISTRY",
		"PIP_INDEX_URL",
		"GOPROXY",
		"RUSTC_WRAPPER",
		"SOCKET_SECURITY_API_KEY",
	}

	for _, key := range required {
		if _, ok := env[key]; !ok {
			t.Errorf("missing env var %q in ConsultingDefault", key)
		}
	}

	// GOPROXY must end with ,direct
	if gp, ok := env["GOPROXY"]; ok {
		if !strings.HasSuffix(gp, ",direct") {
			t.Errorf("GOPROXY = %q, want suffix ,direct", gp)
		}
	}

	// RUSTC_WRAPPER should be sccache
	if rw := env["RUSTC_WRAPPER"]; rw != "sccache" {
		t.Errorf("RUSTC_WRAPPER = %q, want sccache", rw)
	}
}

func TestEnterprise_EnvironmentVars(t *testing.T) {
	env := Enterprise.EnvironmentVars()

	// Artifactory URL patterns
	if npm := env["NPM_CONFIG_REGISTRY"]; !strings.Contains(npm, "artifactory.example.com") {
		t.Errorf("NPM_CONFIG_REGISTRY = %q, want artifactory URL", npm)
	}
	if pip := env["PIP_INDEX_URL"]; !strings.Contains(pip, "pypi-virtual") {
		t.Errorf("PIP_INDEX_URL = %q, want pypi-virtual in URL", pip)
	}
	if nuget, ok := env["NUGET_SOURCE_URL"]; !ok || !strings.Contains(nuget, "nuget-virtual") {
		t.Errorf("NUGET_SOURCE_URL = %q, want nuget-virtual in URL", nuget)
	}

	// SNYK_TOKEN reference
	if snyk := env["SNYK_TOKEN"]; snyk != "${SNYK_TOKEN}" {
		t.Errorf("SNYK_TOKEN = %q, want ${SNYK_TOKEN}", snyk)
	}
}

func TestStartupGitHub_EnvironmentVars(t *testing.T) {
	env := StartupGitHub.EnvironmentVars()

	// GitHub Packages URLs
	if npm := env["NPM_CONFIG_REGISTRY"]; npm != "https://npm.pkg.github.com/" {
		t.Errorf("NPM_CONFIG_REGISTRY = %q, want https://npm.pkg.github.com/", npm)
	}
	if maven := env["MAVEN_REPO_URL"]; maven != "https://maven.pkg.github.com/" {
		t.Errorf("MAVEN_REPO_URL = %q, want https://maven.pkg.github.com/", maven)
	}

	// Turborepo
	if turbo, ok := env["TURBO_API"]; !ok || turbo == "" {
		t.Error("TURBO_API should be set for StartupGitHub")
	}
}

func TestNoHardcodedSecrets(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	// Patterns that would indicate a hardcoded secret.
	forbidden := []string{
		"sk-", "ghp_", "gho_", "Bearer ", "Basic ",
	}

	for _, p := range profiles {
		env := p.EnvironmentVars()
		for k, v := range env {
			for _, pattern := range forbidden {
				if strings.Contains(v, pattern) {
					t.Errorf("profile %q env var %q contains forbidden pattern %q: %q",
						p.Name, k, pattern, v)
				}
			}
		}
	}
}

func TestConsultingDefault_ConfigFiles_Renovate(t *testing.T) {
	files := ConsultingDefault.ConfigFiles()
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
	files := StartupGitHub.ConfigFiles()
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
	files := ConsultingDefault.ConfigFiles()
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
	files := ConsultingDefault.ConfigFiles()
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
	files := Enterprise.ConfigFiles()
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

func TestNixCacheNixConfig_Cachix(t *testing.T) {
	subs, keys := ConsultingDefault.NixCacheNixConfig()

	if subs == "" {
		t.Error("NixCacheNixConfig substituters should not be empty for cachix")
	}
	if !strings.Contains(subs, "cachix.org") {
		t.Errorf("substituters = %q, want to contain cachix.org", subs)
	}
	if keys == "" {
		t.Error("NixCacheNixConfig trustedKeys should not be empty when PublicKey is set")
	}
}

func TestNixCacheNixConfig_CachixFromCacheName(t *testing.T) {
	p := &InfraProfile{
		NixCache: NixCacheConfig{
			Type:      NixCacheCachix,
			CacheName: "testcache",
			PublicKey: "testcache.cachix.org-1:key=",
		},
	}

	subs, keys := p.NixCacheNixConfig()
	if subs != "https://testcache.cachix.org" {
		t.Errorf("substituters = %q, want https://testcache.cachix.org", subs)
	}
	if keys != "testcache.cachix.org-1:key=" {
		t.Errorf("trustedKeys = %q, want testcache.cachix.org-1:key=", keys)
	}
}

func TestNixCacheNixConfig_None(t *testing.T) {
	p := &InfraProfile{
		NixCache: NixCacheConfig{Type: NixCacheNone},
	}
	subs, keys := p.NixCacheNixConfig()
	if subs != "" || keys != "" {
		t.Errorf("NixCacheNixConfig for none: subs=%q, keys=%q; want both empty", subs, keys)
	}
}
