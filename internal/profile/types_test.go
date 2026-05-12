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

func TestEcosystemURL_Artifactory(t *testing.T) {
	r := RegistryConfig{
		Type: RegistryArtifactory,
		URL:  "https://artifactory.example.com",
	}

	tests := []struct {
		ecosystem string
		want      string
	}{
		{"npm", "https://artifactory.example.com/api/npm/npm-virtual/"},
		{"pypi", "https://artifactory.example.com/api/pypi/pypi-virtual/simple"},
		{"go", "https://artifactory.example.com/api/go/go-virtual"},
		{"cargo", "sparse+https://artifactory.example.com/api/cargo/cargo-virtual/index/"},
		{"maven", "https://artifactory.example.com/maven-virtual"},
		{"nuget", "https://artifactory.example.com/api/nuget/nuget-virtual"},
	}

	for _, tt := range tests {
		t.Run(tt.ecosystem, func(t *testing.T) {
			got := r.EcosystemURL(tt.ecosystem)
			if got != tt.want {
				t.Errorf("EcosystemURL(%q) = %q, want %q", tt.ecosystem, got, tt.want)
			}
		})
	}
}

func TestEcosystemURL_Nexus(t *testing.T) {
	r := RegistryConfig{
		Type: RegistryNexus,
		URL:  "https://nexus.example.com",
	}

	tests := []struct {
		ecosystem string
		want      string
	}{
		{"npm", "https://nexus.example.com/repository/npm-group/"},
		{"pypi", "https://nexus.example.com/repository/pypi-group/simple"},
		{"go", "https://nexus.example.com/repository/go-group/"},
		{"maven", "https://nexus.example.com/repository/maven-group/"},
		{"nuget", "https://nexus.example.com/repository/nuget-group/"},
	}

	for _, tt := range tests {
		t.Run(tt.ecosystem, func(t *testing.T) {
			got := r.EcosystemURL(tt.ecosystem)
			if got != tt.want {
				t.Errorf("EcosystemURL(%q) = %q, want %q", tt.ecosystem, got, tt.want)
			}
		})
	}
}

func TestEcosystemURL_GitHub(t *testing.T) {
	r := RegistryConfig{
		Type: RegistryGitHub,
	}

	tests := []struct {
		ecosystem string
		want      string
	}{
		{"npm", "https://npm.pkg.github.com/"},
		{"maven", "https://maven.pkg.github.com/"},
	}

	for _, tt := range tests {
		t.Run(tt.ecosystem, func(t *testing.T) {
			got := r.EcosystemURL(tt.ecosystem)
			if got != tt.want {
				t.Errorf("EcosystemURL(%q) = %q, want %q", tt.ecosystem, got, tt.want)
			}
		})
	}
}

func TestEcosystemURL_None(t *testing.T) {
	r := RegistryConfig{
		Type: RegistryNone,
	}

	if got := r.EcosystemURL("npm"); got != "" {
		t.Errorf("EcosystemURL(npm) with none = %q, want empty", got)
	}
}

func TestEcosystemURL_UnsupportedCombination(t *testing.T) {
	tests := []struct {
		name      string
		regType   RegistryType
		ecosystem string
	}{
		{"nexus-cargo", RegistryNexus, "cargo"},
		{"github-pypi", RegistryGitHub, "pypi"},
		{"github-go", RegistryGitHub, "go"},
		{"github-cargo", RegistryGitHub, "cargo"},
		{"artifactory-unknown", RegistryArtifactory, "haskell"},
		{"nexus-unknown", RegistryNexus, "haskell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RegistryConfig{Type: tt.regType, URL: "https://example.com"}
			if got := r.EcosystemURL(tt.ecosystem); got != "" {
				t.Errorf("EcosystemURL(%q) for %q = %q, want empty", tt.ecosystem, tt.regType, got)
			}
		})
	}
}

func TestEcosystemURL_EmptyURL(t *testing.T) {
	// Artifactory and Nexus require a URL; should return "" when URL is empty.
	for _, regType := range []RegistryType{RegistryArtifactory, RegistryNexus} {
		r := RegistryConfig{Type: regType, URL: ""}
		if got := r.EcosystemURL("npm"); got != "" {
			t.Errorf("EcosystemURL(npm) for %q with empty URL = %q, want empty", regType, got)
		}
	}
}

func TestEcosystemURL_TrailingSlash(t *testing.T) {
	// URLs with trailing slashes should not produce double slashes.
	r := RegistryConfig{
		Type: RegistryArtifactory,
		URL:  "https://artifactory.example.com/",
	}
	got := r.EcosystemURL("npm")
	want := "https://artifactory.example.com/api/npm/npm-virtual/"
	if got != want {
		t.Errorf("EcosystemURL(npm) with trailing slash = %q, want %q", got, want)
	}
}
