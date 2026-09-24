package profile

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInfraProfile_YAMLRoundTrip(t *testing.T) {
	original := *ConsultingDefault

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}

	var decoded InfraProfile
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	// Verify key fields survive the round-trip.
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Registry.Type != original.Registry.Type {
		t.Errorf("Registry.Type: got %q, want %q", decoded.Registry.Type, original.Registry.Type)
	}
	if decoded.NixCache.Type != original.NixCache.Type {
		t.Errorf("NixCache.Type: got %q, want %q", decoded.NixCache.Type, original.NixCache.Type)
	}
	if decoded.BuildCache.Type != original.BuildCache.Type {
		t.Errorf("BuildCache.Type: got %q, want %q", decoded.BuildCache.Type, original.BuildCache.Type)
	}
	if decoded.Scanning.Vulnerability != original.Scanning.Vulnerability {
		t.Errorf("Scanning.Vulnerability: got %q, want %q", decoded.Scanning.Vulnerability, original.Scanning.Vulnerability)
	}
	if decoded.Updates.Type != original.Updates.Type {
		t.Errorf("Updates.Type: got %q, want %q", decoded.Updates.Type, original.Updates.Type)
	}
	if decoded.Updates.AgeGatingDays != original.Updates.AgeGatingDays {
		t.Errorf("Updates.AgeGatingDays: got %d, want %d", decoded.Updates.AgeGatingDays, original.Updates.AgeGatingDays)
	}
	if decoded.SBOM.Generator != original.SBOM.Generator {
		t.Errorf("SBOM.Generator: got %q, want %q", decoded.SBOM.Generator, original.SBOM.Generator)
	}
	if len(decoded.Registry.Ecosystems) != len(original.Registry.Ecosystems) {
		t.Errorf("Registry.Ecosystems length: got %d, want %d", len(decoded.Registry.Ecosystems), len(original.Registry.Ecosystems))
	}
}

func TestInfraProfile_JSONRoundTrip(t *testing.T) {
	original := *ConsultingDefault

	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded InfraProfile
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Registry.Type != original.Registry.Type {
		t.Errorf("Registry.Type: got %q, want %q", decoded.Registry.Type, original.Registry.Type)
	}
	if decoded.BuildCache.Backend != original.BuildCache.Backend {
		t.Errorf("BuildCache.Backend: got %q, want %q", decoded.BuildCache.Backend, original.BuildCache.Backend)
	}
	if decoded.Updates.AutomergePatches != original.Updates.AutomergePatches {
		t.Errorf("Updates.AutomergePatches: got %v, want %v", decoded.Updates.AutomergePatches, original.Updates.AutomergePatches)
	}
}

func TestEcosystemURL(t *testing.T) {
	t.Parallel()
	artifactory := RegistryConfig{
		Type:       RegistryArtifactory,
		URL:        "https://repo.corp.internal/artifactory/",
		Ecosystems: []string{"npm", "pypi", "go", "cargo", "maven", "nuget"},
	}
	nexus := RegistryConfig{
		Type:       RegistryNexus,
		URL:        "https://nexus.corp.internal",
		Ecosystems: []string{"npm", "pypi", "go", "maven", "nuget", "cargo"},
	}
	tests := []struct {
		name string
		reg  RegistryConfig
		eco  string
		want string
	}{
		{"artifactory npm", artifactory, "npm", "https://repo.corp.internal/artifactory/api/npm/npm-virtual/"},
		{"artifactory pypi", artifactory, "pypi", "https://repo.corp.internal/artifactory/api/pypi/pypi-virtual/simple"},
		{"artifactory go", artifactory, "go", "https://repo.corp.internal/artifactory/api/go/go-virtual"},
		{"artifactory cargo", artifactory, "cargo", "https://repo.corp.internal/artifactory/api/cargo/cargo-virtual/index/"},
		{"artifactory maven", artifactory, "maven", "https://repo.corp.internal/artifactory/maven-virtual"},
		{"artifactory gradle shares maven", artifactory, "gradle", "https://repo.corp.internal/artifactory/maven-virtual"},
		{"artifactory nuget v3", artifactory, "nuget", "https://repo.corp.internal/artifactory/api/nuget/v3/nuget-virtual/index.json"},
		{"nexus npm", nexus, "npm", "https://nexus.corp.internal/repository/npm-group/"},
		{"nexus pypi", nexus, "pypi", "https://nexus.corp.internal/repository/pypi-group/simple"},
		{"nexus go", nexus, "go", "https://nexus.corp.internal/repository/go-group/"},
		{"nexus maven", nexus, "maven", "https://nexus.corp.internal/repository/maven-group/"},
		{"nexus nuget v3", nexus, "nuget", "https://nexus.corp.internal/repository/nuget-group/index.json"},
		{"nexus layout gap falls back to the default path", nexus, "cargo", "https://nexus.corp.internal/repository/cargo-proxy/"},
		{"ecosystem not served", RegistryConfig{Type: RegistryNexus, URL: "https://nexus.corp.internal", Ecosystems: []string{"npm"}}, "pypi", ""},
		{"no endpoint", RegistryConfig{Type: RegistryNexus, Ecosystems: []string{"npm"}}, "npm", ""},
		{"override wins", RegistryConfig{Type: RegistryNexus, URL: "https://nexus.corp.internal", Ecosystems: []string{"npm"},
			Overrides: map[string]string{"npm": "https://npm.corp.internal/"}}, "npm", "https://npm.corp.internal/"},
		{"custom path wins over layout", RegistryConfig{Type: RegistryNexus, URL: "https://nexus.corp.internal", Ecosystems: []string{"npm"},
			Paths: map[string]string{"npm": "/repository/npm-all/"}}, "npm", "https://nexus.corp.internal/repository/npm-all/"},
		{"github packages is not a proxy", RegistryConfig{Type: RegistryGitHub, URL: "https://npm.pkg.github.com", Ecosystems: []string{"npm"}}, "npm", ""},
		{"none", RegistryConfig{Type: RegistryNone, URL: "https://nexus.corp.internal", Ecosystems: []string{"npm"}}, "npm", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.reg.EcosystemURL(tt.eco); got != tt.want {
				t.Errorf("EcosystemURL(%q) = %q, want %q", tt.eco, got, tt.want)
			}
		})
	}
}
