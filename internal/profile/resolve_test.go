package profile

import (
	"errors"
	"maps"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const (
	testCacheKey       = "corp.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
	placeholderZeroKey = "myorg.cachix.org-1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

// validInfra is a complete, real-looking infrastructure configuration.
func validInfra() types.InfraConfig {
	return types.InfraConfig{
		RegistryProxy:     "https://repo.corp.internal/artifactory",
		NixCache:          "https://corp.cachix.org",
		NixCachePublicKey: testCacheKey,
	}
}

func TestResolve_Errors(t *testing.T) {
	t.Parallel()
	goProject := ProjectInputs{Ecosystems: []string{"go"}}
	tests := []struct {
		name    string
		p       *InfraProfile
		infra   func(*types.InfraConfig)
		in      ProjectInputs
		wantErr error
		wantMsg string
	}{
		{"registry proxy missing for a proxied ecosystem", Enterprise, func(c *types.InfraConfig) { c.RegistryProxy = "" }, goProject,
			ErrEndpointNotConfigured, "infrastructure.registry_proxy is not set"},
		{"nix cache missing", Enterprise, func(c *types.InfraConfig) { c.NixCache = "" }, goProject,
			ErrEndpointNotConfigured, "infrastructure.nix_cache is not set"},
		{"nix cache key missing", ConsultingDefault, func(c *types.InfraConfig) { c.NixCachePublicKey = "" }, goProject,
			ErrEndpointNotConfigured, "needs infrastructure.nix_cache_public_key"},
		{"placeholder registry host", ConsultingDefault, func(c *types.InfraConfig) { c.RegistryProxy = "https://nexus.example.com" }, goProject,
			ErrPlaceholderEndpoint, "infrastructure.registry_proxy"},
		{"placeholder override host", Enterprise, func(c *types.InfraConfig) {
			c.RegistryProxyOverrides = map[string]string{"npm": "https://npm.corp.example"}
		}, goProject, ErrPlaceholderEndpoint, "registry_proxy_overrides.npm"},
		{"placeholder cachix cache URL", Enterprise, func(c *types.InfraConfig) { c.NixCache = "https://myorg.cachix.org" }, goProject,
			ErrPlaceholderEndpoint, "example Cachix cache"},
		{"placeholder cachix cache name", Enterprise, func(c *types.InfraConfig) { c.NixCache = "myorg" }, goProject,
			ErrPlaceholderEndpoint, "example Cachix cache"},
		{"all-zero public key", Enterprise, func(c *types.InfraConfig) { c.NixCachePublicKey = placeholderZeroKey }, goProject,
			ErrPlaceholderEndpoint, "all-zero"},
		{"malformed public key", Enterprise, func(c *types.InfraConfig) { c.NixCachePublicKey = "not-a-key" }, goProject,
			ErrInvalidEndpoint, "nix_cache_public_key"},
		{"plain http to a remote host", Enterprise, func(c *types.InfraConfig) { c.RegistryProxy = "http://repo.corp.internal" }, goProject,
			ErrInvalidEndpoint, "plain http"},
		{"credentials in the URL", Enterprise, func(c *types.InfraConfig) { c.RegistryProxy = "https://u:p@repo.corp.internal" }, goProject,
			ErrInvalidEndpoint, "must not embed credentials"},
		{"placeholder turborepo URL", StartupGitHub, func(c *types.InfraConfig) { c.BuildCacheURL = "https://turbo.example.com" }, goProject,
			ErrPlaceholderEndpoint, "build_cache_url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			infra := validInfra()
			tt.infra(&infra)
			_, err := tt.p.Resolve(infra, tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve() error = %v, want %v", err, tt.wantErr)
			}
			for _, want := range []string{tt.wantMsg, tt.p.Name, "infrastructure:"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %q", err, want)
				}
			}
		})
	}
}

func TestResolve_Accepts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		p     *InfraProfile
		infra types.InfraConfig
		in    ProjectInputs
	}{
		{"complete configuration", Enterprise, validInfra(), ProjectInputs{Ecosystems: []string{"go", "npm"}}},
		{"registry proxy not needed without a proxied ecosystem", ConsultingDefault,
			types.InfraConfig{NixCache: "corp", NixCachePublicKey: testCacheKey}, ProjectInputs{Ecosystems: []string{"container"}}},
		{"per-ecosystem overrides instead of a base URL", Enterprise,
			types.InfraConfig{RegistryProxyOverrides: map[string]string{"go": "https://goproxy.corp.internal"}, NixCache: "corp", NixCachePublicKey: testCacheKey},
			ProjectInputs{Ecosystems: []string{"go"}}},
		{"components opted out with none", Enterprise,
			types.InfraConfig{RegistryProxy: types.InfraDisabled, NixCache: types.InfraDisabled}, ProjectInputs{Ecosystems: []string{"go"}}},
		{"github packages needs no proxy endpoint", StartupGitHub,
			types.InfraConfig{NixCache: "corp", NixCachePublicKey: testCacheKey}, ProjectInputs{Ecosystems: []string{"npm"}}},
		{"loopback http registry", ConsultingDefault,
			types.InfraConfig{RegistryProxy: "http://localhost:8081", NixCache: types.InfraDisabled}, ProjectInputs{Ecosystems: []string{"npm"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tt.p.Resolve(tt.infra, tt.in); err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
		})
	}
}

// TestResolve_Infrastructure checks the effective settings a resolved
// profile hands the generators.
func TestResolve_Infrastructure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		p     *InfraProfile
		infra types.InfraConfig
		want  types.InfraConfig
	}{
		{"enterprise artifactory layout, cachix and sccache", Enterprise, validInfra(), types.InfraConfig{
			RegistryProxy: "https://repo.corp.internal/artifactory",
			RegistryProxyPaths: map[string]string{
				"npm": "/api/npm/npm-virtual/", "pypi": "/api/pypi/pypi-virtual/simple", "go": "/api/go/go-virtual",
				"cargo": "/api/cargo/cargo-virtual/index/", "maven": "/maven-virtual", "gradle": "/maven-virtual",
				"nuget": "/api/nuget/v3/nuget-virtual/index.json",
			},
			NixCache: "https://corp.cachix.org", NixCachePublicKey: testCacheKey, BuildCache: "sccache",
		}},
		{"user paths win and a cachix name becomes its URL", ConsultingDefault, types.InfraConfig{
			RegistryProxy: "https://nexus.corp.internal", RegistryProxyPaths: map[string]string{"npm": "/repository/npm-all/"},
			NixCache: "corp", NixCachePublicKey: testCacheKey, BuildCache: "none",
		}, types.InfraConfig{
			RegistryProxy: "https://nexus.corp.internal",
			RegistryProxyPaths: map[string]string{
				"npm": "/repository/npm-all/", "pypi": "/repository/pypi-group/simple", "go": "/repository/go-group/",
				"maven": "/repository/maven-group/", "gradle": "/repository/maven-group/",
			},
			NixCache: "https://corp.cachix.org", NixCachePublicKey: testCacheKey,
		}},
		{"github packages adds no layout", StartupGitHub, types.InfraConfig{
			NixCache: "corp", NixCachePublicKey: testCacheKey, BuildCacheURL: "https://turbo.corp.internal",
		}, types.InfraConfig{
			NixCache: "https://corp.cachix.org", NixCachePublicKey: testCacheKey, BuildCache: "turborepo", BuildCacheURL: "https://turbo.corp.internal",
		}},
		{"opted out", Enterprise, types.InfraConfig{RegistryProxy: types.InfraDisabled, NixCache: types.InfraDisabled},
			types.InfraConfig{BuildCache: "sccache"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, err := tt.p.Resolve(tt.infra, ProjectInputs{})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			got := r.Infrastructure()
			if got.RegistryProxy != tt.want.RegistryProxy || !maps.Equal(got.RegistryProxyPaths, tt.want.RegistryProxyPaths) ||
				!maps.Equal(got.RegistryProxyOverrides, tt.want.RegistryProxyOverrides) || got.NixCache != tt.want.NixCache ||
				got.NixCachePublicKey != tt.want.NixCachePublicKey || got.BuildCache != tt.want.BuildCache || got.BuildCacheURL != tt.want.BuildCacheURL {
				t.Errorf("Infrastructure() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}

// TestResolve_DoesNotMutateProfile guards the shared built-in profiles.
func TestResolve_DoesNotMutateProfile(t *testing.T) {
	t.Parallel()
	infra := validInfra()
	infra.RegistryProxyOverrides = map[string]string{"npm": "https://npm.corp.internal"}
	if _, err := Enterprise.Resolve(infra, ProjectInputs{}); err != nil {
		t.Fatal(err)
	}
	if Enterprise.Registry.URL != "" || Enterprise.Registry.Overrides != nil || Enterprise.NixCache.PublicKey != "" {
		t.Errorf("Resolve mutated the built-in profile: %+v %+v", Enterprise.Registry, Enterprise.NixCache)
	}
}

func TestSecurityDoc_DescribesAppliedInfrastructure(t *testing.T) {
	t.Parallel()
	r, err := Enterprise.Resolve(validInfra(), ProjectInputs{})
	if err != nil {
		t.Fatal(err)
	}
	in := ProjectInputs{Ecosystems: []string{"go"}, Infrastructure: r.Infrastructure()}
	doc := string(mustFile(t, r, in, "docs/security-overview.md").Content)
	for _, want := range []string{
		"**Registry proxy**: go via https://repo.corp.internal/artifactory/api/go/go-virtual",
		"**Nix binary cache**: https://corp.cachix.org",
		"**Build cache**: sccache (s3 backend)",
		"`ARTIFACTORY_TOKEN`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("security overview lacks %q", want)
		}
	}

	// The implicit default with nothing configured must not claim a proxy.
	doc = string(mustFile(t, ConsultingDefault, ProjectInputs{Ecosystems: []string{"go"}}, "docs/security-overview.md").Content)
	for _, want := range []string{"**Registry proxy**: not configured", "**Nix binary cache**: not configured"} {
		if !strings.Contains(doc, want) {
			t.Errorf("unconfigured security overview lacks %q", want)
		}
	}
}

func mustFile(t *testing.T, p *InfraProfile, in ProjectInputs, path string) types.GeneratedFile {
	t.Helper()
	f, ok := findFile(mustConfigFiles(t, p, in), path)
	if !ok {
		t.Fatalf("%s not generated", path)
	}
	return f
}

// TestResolveProjectInfrastructure covers a Nix cache configured without an
// infra profile: it is written into devenv.nix, so it is checked like a
// profile's and a bare Cachix name becomes its substituter URL.
func TestResolveProjectInfrastructure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		infra     types.InfraConfig
		wantCache string
		wantErr   error
	}{
		{"nothing configured", types.InfraConfig{RegistryProxy: "https://repo.corp.internal"}, "", nil},
		{"opted out", types.InfraConfig{NixCache: types.InfraDisabled}, types.InfraDisabled, nil},
		{"cachix name", types.InfraConfig{NixCache: "corp", NixCachePublicKey: testCacheKey}, "https://corp.cachix.org", nil},
		{"cache URL", types.InfraConfig{NixCache: "https://cache.corp.internal", NixCachePublicKey: testCacheKey}, "https://cache.corp.internal", nil},
		{"key missing", types.InfraConfig{NixCache: "corp"}, "", ErrEndpointNotConfigured},
		{"placeholder cache", types.InfraConfig{NixCache: "https://myorg.cachix.org", NixCachePublicKey: testCacheKey}, "", ErrPlaceholderEndpoint},
		{"all-zero key", types.InfraConfig{NixCache: "corp", NixCachePublicKey: placeholderZeroKey}, "", ErrPlaceholderEndpoint},
		{"plain http", types.InfraConfig{NixCache: "http://cache.corp.internal", NixCachePublicKey: testCacheKey}, "", ErrInvalidEndpoint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveProjectInfrastructure(tt.infra)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), "infrastructure.nix_cache") {
					t.Fatalf("ResolveProjectInfrastructure() error = %v, want %v naming the setting", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveProjectInfrastructure() error = %v", err)
			}
			if got.NixCache != tt.wantCache || got.RegistryProxy != tt.infra.RegistryProxy {
				t.Errorf("ResolveProjectInfrastructure() = %+v, want nix_cache %q", got, tt.wantCache)
			}
		})
	}
}

// TestConfigOnly_DescribesNoComponents keeps the implicit default's security
// overview from listing credentials or a build cache it never applied.
func TestConfigOnly_DescribesNoComponents(t *testing.T) {
	t.Parallel()
	p := ConsultingDefault.ConfigOnly()
	if creds := p.CredentialEnvVars(); len(creds) != 0 {
		t.Errorf("ConfigOnly().CredentialEnvVars() = %v, want none", creds)
	}
	if ConsultingDefault.Registry.Type != RegistryNexus || ConsultingDefault.BuildCache.Type != BuildCacheSccache {
		t.Error("ConfigOnly mutated the built-in profile")
	}
	doc := string(mustFile(t, p, ProjectInputs{Ecosystems: []string{"rust"}}, "docs/security-overview.md").Content)
	for _, unwanted := range []string{"NEXUS_TOKEN", "AWS_ACCESS_KEY_ID", "CACHIX_AUTH_TOKEN"} {
		if strings.Contains(doc, unwanted) {
			t.Errorf("implicit default security overview lists %s", unwanted)
		}
	}
}
