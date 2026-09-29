package devenv_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const testNixCacheKey = "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="

// generateInfra runs the full devenv generator with the real ecosystem
// modules and the built-in infra profiles, keyed by path.
func generateInfra(t *testing.T, answers types.WizardAnswers) (map[string]string, error) {
	t.Helper()
	gen := devenv.NewDevenvGenerator(ecosystem.DefaultRegistry(), devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
	files, err := gen.Generate(answers)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(files))
	for _, f := range files {
		out[f.Path] = string(f.Content)
	}
	return out, nil
}

func infraAnswers(profileName string, infra types.InfraConfig, langs ...types.LanguageChoice) types.WizardAnswers {
	return types.WizardAnswers{
		ProjectName:       "infra",
		Languages:         langs,
		Tier:              "standard",
		ProfileName:       profileName,
		Infrastructure:    infra,
		NixHardeningGuide: true,
	}
}

// TestGenerate_ExplicitInfraProfileRequiresEndpoints is the regression test
// for --infra-profile enterprise configuring no registry proxy while the
// security overview claimed the profile was in place: without real
// endpoints the generator now refuses and names the missing settings.
func TestGenerate_ExplicitInfraProfileRequiresEndpoints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		infra   types.InfraConfig
		wantErr error
		want    []string
	}{
		{"nothing configured", types.InfraConfig{}, profile.ErrEndpointNotConfigured,
			[]string{`"enterprise"`, "infrastructure.registry_proxy is not set", "infrastructure.nix_cache is not set", "--registry-proxy"}},
		{"placeholder endpoints", types.InfraConfig{
			RegistryProxy: "https://nexus.example.com", NixCache: "myorg",
			NixCachePublicKey: "myorg.cachix.org-1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		}, profile.ErrPlaceholderEndpoint, []string{"example host", "example Cachix cache", "all-zero"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := generateInfra(t, infraAnswers("enterprise", tt.infra, types.LanguageChoice{Name: "go", Version: "1.24"}))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Generate() error = %v, want %v", err, tt.wantErr)
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error lacks %q:\n%v", want, err)
				}
			}
		})
	}
}

// TestGenerate_ExplicitInfraProfileIsApplied checks the resolved profile
// reaches the package-manager configs, devenv.nix, the nix.conf guide and
// the security overview.
func TestGenerate_ExplicitInfraProfileIsApplied(t *testing.T) {
	t.Parallel()
	infra := types.InfraConfig{
		RegistryProxy:     "https://repo.corp.internal/artifactory",
		NixCache:          "corp",
		NixCachePublicKey: testNixCacheKey,
	}
	files, err := generateInfra(t, infraAnswers("enterprise", infra,
		types.LanguageChoice{Name: "javascript", PackageManager: "npm"},
		types.LanguageChoice{Name: "rust"}))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	checks := []struct{ path, want string }{
		{".npmrc", "registry=https://repo.corp.internal/artifactory/api/npm/npm-virtual/"},
		{".cargo/config.toml", "sparse+https://repo.corp.internal/artifactory/api/cargo/cargo-virtual/index/"},
		{".cargo/config.toml", `rustc-wrapper = "sccache"`},
		{"devenv.nix", `cachix.pull = [ "corp" ];`},
		{"devenv.nix", "pkgs.sccache"},
		{"docs/nix-conf-hardening.md", "https://corp.cachix.org"},
		{"docs/nix-conf-hardening.md", testNixCacheKey},
		{"docs/security-overview.md", "npm via https://repo.corp.internal/artifactory/api/npm/npm-virtual/"},
	}
	for _, c := range checks {
		if !strings.Contains(files[c.path], c.want) {
			t.Errorf("%s lacks %q:\n%s", c.path, c.want, files[c.path])
		}
	}
	if strings.Contains(files["devenv.nix"], "AWS_ACCESS_KEY_ID = ") {
		t.Error("devenv.nix sets a credential variable; credentials must come from the environment")
	}
}

func TestGenerate_InfraProfileEnvironment(t *testing.T) {
	t.Parallel()
	infra := types.InfraConfig{
		NixCache:          types.InfraDisabled,
		BuildCacheURL:     "https://turbo.corp.internal",
		RegistryProxy:     types.InfraDisabled,
		NixCachePublicKey: "",
	}
	answers := infraAnswers("startup-github", infra, types.LanguageChoice{Name: "javascript", PackageManager: "npm"})
	answers.EnvVars = map[string]string{"KEEP": "1"}
	files, err := generateInfra(t, answers)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	for _, want := range []string{`TURBO_API = "https://turbo.corp.internal";`, `KEEP = "1";`} {
		if !strings.Contains(files["devenv.nix"], want) {
			t.Errorf("devenv.nix lacks %q", want)
		}
	}
	if strings.Contains(files["devenv.nix"], "cachix.pull") {
		t.Error("nix_cache none still emitted cachix.pull")
	}
	// A user env var wins over the profile's.
	answers.EnvVars = map[string]string{"TURBO_API": "https://mine.corp.internal"}
	files, err = generateInfra(t, answers)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["devenv.nix"], `TURBO_API = "https://mine.corp.internal";`) {
		t.Error("user TURBO_API did not override the profile's")
	}
}

// TestGenerate_ImplicitInfraProfileAppliesNothing keeps projects that never
// chose an infra profile on exactly the endpoints they configured.
func TestGenerate_ImplicitInfraProfileAppliesNothing(t *testing.T) {
	t.Parallel()
	files, err := generateInfra(t, infraAnswers("", types.InfraConfig{}, types.LanguageChoice{Name: "rust"}))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if strings.Contains(files[".cargo/config.toml"], "sccache") || strings.Contains(files["devenv.nix"], "cachix.pull") {
		t.Error("implicit default profile applied its build or Nix cache")
	}
	if !strings.Contains(files["docs/security-overview.md"], "**Registry proxy**: not configured") {
		t.Error("security overview should state the registry proxy is not configured")
	}
	if strings.Contains(files["docs/security-overview.md"], "Credentials read from the environment") {
		t.Error("security overview lists credentials for components the implicit default never applied")
	}
}

// TestGenerate_ProjectNixCacheWithoutProfile checks a Nix cache configured
// without an infra profile is applied when real and refused when a
// placeholder, instead of reaching devenv.nix unchecked or being dropped.
func TestGenerate_ProjectNixCacheWithoutProfile(t *testing.T) {
	t.Parallel()
	files, err := generateInfra(t, infraAnswers("", types.InfraConfig{NixCache: "corp", NixCachePublicKey: testNixCacheKey},
		types.LanguageChoice{Name: "go", Version: "1.24"}))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	for _, c := range []struct{ path, want string }{
		{"devenv.nix", `cachix.pull = [ "corp" ];`},
		{"docs/nix-conf-hardening.md", "https://corp.cachix.org"},
		{"docs/security-overview.md", "**Nix binary cache**: https://corp.cachix.org"},
	} {
		if !strings.Contains(files[c.path], c.want) {
			t.Errorf("%s lacks %q", c.path, c.want)
		}
	}

	_, err = generateInfra(t, infraAnswers("", types.InfraConfig{
		NixCache: "https://myorg.cachix.org", NixCachePublicKey: "myorg.cachix.org-1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}, types.LanguageChoice{Name: "go", Version: "1.24"}))
	if !errors.Is(err, profile.ErrPlaceholderEndpoint) {
		t.Errorf("placeholder Nix cache without a profile: err = %v, want %v", err, profile.ErrPlaceholderEndpoint)
	}
}
