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
		wantField string
	}{
		{"nothing configured", types.InfraConfig{RegistryProxy: "https://repo.corp.internal"}, "", nil, ""},
		{"opted out", types.InfraConfig{NixCache: types.InfraDisabled}, types.InfraDisabled, nil, ""},
		{"cachix name", types.InfraConfig{NixCache: "corp", NixCachePublicKey: testCacheKey}, "https://corp.cachix.org", nil, ""},
		{"cache URL", types.InfraConfig{NixCache: "https://cache.corp.internal", NixCachePublicKey: testCacheKey}, "https://cache.corp.internal", nil, ""},
		{"key missing", types.InfraConfig{NixCache: "corp"}, "", ErrEndpointNotConfigured, "infrastructure.nix_cache"},
		{"placeholder cache", types.InfraConfig{NixCache: "https://myorg.cachix.org", NixCachePublicKey: testCacheKey}, "", ErrPlaceholderEndpoint, "infrastructure.nix_cache"},
		{"placeholder cache trailing dot", types.InfraConfig{NixCache: "https://myorg.cachix.org.", NixCachePublicKey: testCacheKey}, "", ErrPlaceholderEndpoint, "infrastructure.nix_cache"},
		{"cache port out of range", types.InfraConfig{NixCache: "https://cache.corp.internal:70000", NixCachePublicKey: testCacheKey}, "", ErrInvalidEndpoint, "infrastructure.nix_cache"},
		{"all-zero key", types.InfraConfig{NixCache: "corp", NixCachePublicKey: placeholderZeroKey}, "", ErrPlaceholderEndpoint, "infrastructure.nix_cache"},
		{"plain http", types.InfraConfig{NixCache: "http://cache.corp.internal", NixCachePublicKey: testCacheKey}, "", ErrInvalidEndpoint, "infrastructure.nix_cache"},
		{"registry proxy plain http", types.InfraConfig{RegistryProxy: "http://alice:s3cret@proxy.corp.lan:8081"}, "", ErrInvalidEndpoint, "infrastructure.registry_proxy"},
		{"registry proxy credentials", types.InfraConfig{RegistryProxy: "https://alice:s3cret@proxy.corp.lan"}, "", ErrInvalidEndpoint, "infrastructure.registry_proxy"},
		{"registry proxy newline", types.InfraConfig{RegistryProxy: "https://proxy.corp.lan\nregistry=https://evil.io"}, "", ErrInvalidEndpoint, "infrastructure.registry_proxy"},
		{"registry path rewrites host", types.InfraConfig{
			RegistryProxy: "https://artifactory.corp.io", RegistryProxyPaths: map[string]string{"npm": "@attacker.io/npm/"},
		}, "", ErrInvalidEndpoint, "infrastructure.registry_proxy_paths.npm"},
		{"loopback http registry proxy", types.InfraConfig{RegistryProxy: "http://127.0.0.1:8081"}, "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveProjectInfrastructure(tt.infra)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || !strings.Contains(err.Error(), tt.wantField) {
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

// TestValidateInfra_NoProfile checks every endpoint of a project's
// infrastructure is validated without an infra profile selected.
func TestValidateInfra_NoProfile(t *testing.T) {
	t.Parallel()
	fields := []struct {
		name string
		set  func(*types.InfraConfig, string)
	}{
		{"infrastructure.registry_proxy", func(c *types.InfraConfig, v string) { c.RegistryProxy = v }},
		{"infrastructure.registry_proxy_overrides.npm", func(c *types.InfraConfig, v string) {
			c.RegistryProxyOverrides = map[string]string{"npm": v}
		}},
		{"infrastructure.build_cache_url", func(c *types.InfraConfig, v string) { c.BuildCacheURL = v }},
	}
	values := []struct {
		name    string
		value   string
		wantErr error
		wantMsg string
	}{
		{"plain http to a remote host", "http://proxy.corp.lan:8081", ErrInvalidEndpoint, "uses plain http"},
		{"plain http with credentials", "http://alice:s3cret@proxy.corp.lan:8081", ErrInvalidEndpoint, "uses plain http"},
		{"embedded credentials", "https://alice:s3cret@proxy.corp.lan", ErrInvalidEndpoint, "environment"},
		{"placeholder host", "https://proxy.example.com", ErrPlaceholderEndpoint, "example host"},
		{"embedded newline", "https://proxy.corp.lan\nregistry=https://evil.io", ErrInvalidEndpoint, "not an absolute http(s) URL"},
		{"port above 65535", "https://proxy.corp.lan:65536", ErrInvalidEndpoint, "port"},
		{"port far out of range", "https://proxy.corp.lan:99999999999", ErrInvalidEndpoint, "port"},
		{"port zero", "https://proxy.corp.lan:0", ErrInvalidEndpoint, "port"},
		{"example cachix cache with trailing dot", "https://myorg.cachix.org.", ErrPlaceholderEndpoint, "example Cachix cache"},
		{"https", "https://proxy.corp.lan/artifactory", nil, ""},
		{"highest port", "https://proxy.corp.lan:65535", nil, ""},
		{"loopback http", "http://localhost:8081", nil, ""},
	}
	for _, f := range fields {
		for _, v := range values {
			t.Run(f.name+"/"+v.name, func(t *testing.T) {
				t.Parallel()
				var infra types.InfraConfig
				f.set(&infra, v.value)
				err := errors.Join(ValidateInfra(infra)...)
				if v.wantErr == nil {
					if err != nil {
						t.Fatalf("ValidateInfra() error = %v", err)
					}
					return
				}
				if !errors.Is(err, v.wantErr) {
					t.Fatalf("ValidateInfra() error = %v, want %v", err, v.wantErr)
				}
				for _, want := range []string{f.name, v.wantMsg} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q lacks %q", err, want)
					}
				}
				if strings.Contains(err.Error(), "s3cret") {
					t.Errorf("error %q echoes the embedded password", err)
				}
			})
		}
	}

	t.Run("registry_proxy none", func(t *testing.T) {
		t.Parallel()
		if errs := ValidateInfra(types.InfraConfig{RegistryProxy: types.InfraDisabled, NixCache: types.InfraDisabled}); len(errs) != 0 {
			t.Errorf("ValidateInfra() = %v, want none", errs)
		}
	})
	t.Run("nix cache without key", func(t *testing.T) {
		t.Parallel()
		err := errors.Join(ValidateInfra(types.InfraConfig{NixCache: "corp"})...)
		if !errors.Is(err, ErrEndpointNotConfigured) || !strings.Contains(err.Error(), "nix_cache_public_key") {
			t.Errorf("ValidateInfra() error = %v, want %v naming nix_cache_public_key", err, ErrEndpointNotConfigured)
		}
	})
}

// TestValidateInfra_PathRewritesHost is the regression test for a
// registry_proxy_paths value that moved the computed proxy URL to another
// host (https://artifactory.corp.io + "@attacker.io/npm/").
func TestValidateInfra_PathRewritesHost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		ok   bool
	}{
		{"@evil/", false},
		{"@attacker.io/npm/", false},
		{"evil.com/x", false},
		{"//evil.com/x", false},
		{"npm/", false},
		{"/repository/npm/", true},
		{"/api/npm/npm-virtual/", true},
	}
	for _, base := range []string{"https://artifactory.corp.io", ""} {
		for _, tt := range tests {
			t.Run(base+" "+tt.path, func(t *testing.T) {
				t.Parallel()
				err := errors.Join(ValidateInfra(types.InfraConfig{
					RegistryProxy:      base,
					RegistryProxyPaths: map[string]string{"npm": tt.path},
				})...)
				if tt.ok {
					if err != nil {
						t.Fatalf("ValidateInfra() error = %v", err)
					}
					return
				}
				if !errors.Is(err, ErrInvalidEndpoint) || !strings.Contains(err.Error(), "infrastructure.registry_proxy_paths.npm") {
					t.Fatalf("ValidateInfra() error = %v, want %v naming infrastructure.registry_proxy_paths.npm", err, ErrInvalidEndpoint)
				}
			})
		}
	}
}

// TestResolve_RejectsBadRegistryPath checks an explicitly selected profile
// validates registry_proxy_paths through the same shared checks.
func TestResolve_RejectsBadRegistryPath(t *testing.T) {
	t.Parallel()
	infra := validInfra()
	infra.RegistryProxyPaths = map[string]string{"npm": "@attacker.io/npm/"}
	_, err := Enterprise.Resolve(infra, ProjectInputs{Ecosystems: []string{"npm"}})
	if !errors.Is(err, ErrInvalidEndpoint) || !strings.Contains(err.Error(), "infrastructure.registry_proxy_paths.npm") {
		t.Fatalf("Resolve() error = %v, want %v naming infrastructure.registry_proxy_paths.npm", err, ErrInvalidEndpoint)
	}

	infra.RegistryProxyPaths = map[string]string{"npm": "/custom/npm/"}
	if _, err := Enterprise.Resolve(infra, ProjectInputs{Ecosystems: []string{"npm"}}); err != nil {
		t.Fatalf("Resolve() with a valid path error = %v", err)
	}
}
