package generate_test

import (
	"os/exec"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/generate"
)

func TestYAMLValidator_Valid(t *testing.T) {
	v := &generate.YAMLValidator{}
	result := v.Validate([]byte("key: value\nlist:\n  - one\n  - two\n"))
	if !result.Valid {
		t.Errorf("expected valid YAML, got error: %v", result.Error)
	}
}

func TestYAMLValidator_Invalid(t *testing.T) {
	v := &generate.YAMLValidator{}
	result := v.Validate([]byte("key: [unclosed"))
	if result.Valid {
		t.Error("expected invalid YAML to fail validation")
	}
	if result.Error == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestJSONValidator_Valid(t *testing.T) {
	v := &generate.JSONValidator{}
	result := v.Validate([]byte(`{"key": "value", "num": 42}`))
	if !result.Valid {
		t.Errorf("expected valid JSON, got error: %v", result.Error)
	}
}

func TestJSONValidator_Invalid(t *testing.T) {
	v := &generate.JSONValidator{}
	result := v.Validate([]byte(`{"key": "value",}`))
	if result.Valid {
		t.Error("expected invalid JSON to fail validation")
	}
	if result.Error == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestShellValidator_Valid(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	v := generate.NewShellValidator()
	result := v.Validate([]byte("#!/bin/bash\necho hello\nif true; then\n  echo yes\nfi\n"))
	if !result.Valid {
		t.Errorf("expected valid shell, got error: %v", result.Error)
	}
}

func TestShellValidator_Invalid(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	v := generate.NewShellValidator()
	result := v.Validate([]byte("#!/bin/bash\nif true; then\n"))
	if result.Valid {
		t.Error("expected invalid shell to fail validation")
	}
	if result.Error == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestNixValidator_SkipsWhenNotFound(t *testing.T) {
	// This test verifies the skip behavior. On systems with nix-instantiate
	// it will actually validate; on systems without it, it should skip.
	registry := generate.NewValidatorRegistry()
	result := registry.Validate("test.nix", []byte("{ pkgs }: pkgs"))

	if _, err := exec.LookPath("nix-instantiate"); err != nil {
		// nix-instantiate not available: expect skip
		if !result.Skipped {
			t.Error("expected Nix validation to be skipped when nix-instantiate not found")
		}
		if result.Warning == "" {
			t.Error("expected warning when nix-instantiate not found")
		}
	} else {
		// nix-instantiate available: should validate
		if result.Skipped {
			t.Error("expected Nix validation to run when nix-instantiate is available")
		}
	}
}

func TestValidatorRegistry_ExtensionDispatch(t *testing.T) {
	registry := generate.NewValidatorRegistry()

	tests := []struct {
		path  string
		valid bool
	}{
		{"config.yaml", true},
		{"config.yml", true},
		{"data.json", true},
		{"script.sh", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Use trivially valid content for each type.
			var content []byte
			switch tt.path {
			case "config.yaml", "config.yml":
				content = []byte("key: value\n")
			case "data.json":
				content = []byte(`{"key": "value"}`)
			case "script.sh":
				if _, err := exec.LookPath("bash"); err != nil {
					t.Skip("bash not available")
				}
				content = []byte("#!/bin/bash\necho hello\n")
			}

			result := registry.Validate(tt.path, content)
			if !result.Valid && !result.Skipped {
				t.Errorf("expected valid result for %s, got error: %v", tt.path, result.Error)
			}
		})
	}
}

func TestValidatorRegistry_UnknownExtensionReturnsSkipped(t *testing.T) {
	registry := generate.NewValidatorRegistry()

	tests := []string{"README.md", "file.txt", "Makefile", "image.png", ".gitignore"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			result := registry.Validate(path, []byte("anything"))
			if !result.Skipped {
				t.Errorf("expected skipped for %s, got valid=%v, error=%v", path, result.Valid, result.Error)
			}
		})
	}
}
