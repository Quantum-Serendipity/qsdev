package devinit_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
)

func TestBuiltinProfiles_HaveRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		profile devinit.ExportProfile
	}{
		{"go-web", devinit.ExportGoWeb},
		{"ts-fullstack", devinit.ExportTSFullstack},
		{"python-data", devinit.ExportPythonData},
		{"rust-cli", devinit.ExportRustCLI},
		{"java-web", devinit.ExportJavaWeb},
		{"python-web", devinit.ExportPythonWeb},
		{"ts-backend", devinit.ExportTSBackend},
		{"elixir-web", devinit.ExportElixirWeb},
		{"rust-web", devinit.ExportRustWeb},
		{"dotnet-web", devinit.ExportDotnetWeb},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.profile.Description == "" {
				t.Error("Description is empty")
			}
			if len(tt.profile.Languages) == 0 {
				t.Error("Languages is empty")
			}
			for _, lang := range tt.profile.Languages {
				if lang.Name == "" {
					t.Error("Language has empty Name")
				}
			}
		})
	}
}

func TestBuiltinProfiles_GoWeb(t *testing.T) {
	p := devinit.ExportGoWeb

	if len(p.Languages) != 1 || p.Languages[0].Name != "go" {
		t.Errorf("Languages = %v, want [{go 1.24}]", p.Languages)
	}
	if p.Languages[0].Version != "1.24" {
		t.Errorf("Go version = %q, want %q", p.Languages[0].Version, "1.24")
	}
	if len(p.Services) != 2 {
		t.Errorf("Services length = %d, want 2", len(p.Services))
	}
	if !p.Direnv {
		t.Error("Direnv should be true")
	}
	if !p.ClaudeCode {
		t.Error("ClaudeCode should be true")
	}
	if p.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "standard")
	}
}

func TestBuiltinProfiles_TSFullstack(t *testing.T) {
	p := devinit.ExportTSFullstack

	if len(p.Languages) != 1 || p.Languages[0].Name != "javascript" {
		t.Errorf("Languages = %v, want [{javascript pnpm}]", p.Languages)
	}
	if p.Languages[0].PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q", p.Languages[0].PackageManager, "pnpm")
	}
	if p.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "standard")
	}
	// Should have auto-format, safety-block, pre-commit hooks.
	if len(p.Hooks) != 3 {
		t.Errorf("Hooks length = %d, want 3", len(p.Hooks))
	}
}

func TestBuiltinProfiles_PythonData(t *testing.T) {
	p := devinit.ExportPythonData

	if len(p.Languages) != 1 || p.Languages[0].Name != "python" {
		t.Errorf("Languages = %v, want [{python 3.12 uv}]", p.Languages)
	}
	if p.Languages[0].Version != "3.12" {
		t.Errorf("Python version = %q, want %q", p.Languages[0].Version, "3.12")
	}
	if p.Languages[0].PackageManager != "uv" {
		t.Errorf("PackageManager = %q, want %q", p.Languages[0].PackageManager, "uv")
	}
	if len(p.Services) != 0 {
		t.Errorf("Services length = %d, want 0", len(p.Services))
	}
	if p.PermissionLevel != "minimal" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "minimal")
	}
}

func TestBuiltinProfiles_RustCLI(t *testing.T) {
	p := devinit.ExportRustCLI

	if len(p.Languages) != 1 || p.Languages[0].Name != "rust" {
		t.Errorf("Languages = %v, want [{rust}]", p.Languages)
	}
	if len(p.Services) != 0 {
		t.Errorf("Services length = %d, want 0", len(p.Services))
	}
	if p.PermissionLevel != "minimal" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "minimal")
	}
	// Should have safety-block, pre-commit hooks.
	if len(p.Hooks) != 2 {
		t.Errorf("Hooks length = %d, want 2", len(p.Hooks))
	}
}

func TestBuiltinProfiles_WebProfiles_HaveServices(t *testing.T) {
	webProfiles := []struct {
		name    string
		profile devinit.ExportProfile
	}{
		{"java-web", devinit.ExportJavaWeb},
		{"python-web", devinit.ExportPythonWeb},
		{"ts-backend", devinit.ExportTSBackend},
		{"elixir-web", devinit.ExportElixirWeb},
		{"rust-web", devinit.ExportRustWeb},
		{"dotnet-web", devinit.ExportDotnetWeb},
	}

	for _, tt := range webProfiles {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.profile.Services) != 2 {
				t.Errorf("Services length = %d, want 2 (postgres, redis)", len(tt.profile.Services))
			}
			if tt.profile.PermissionLevel != "standard" {
				t.Errorf("PermissionLevel = %q, want %q", tt.profile.PermissionLevel, "standard")
			}
			if !tt.profile.Direnv {
				t.Error("Direnv should be true")
			}
			if !tt.profile.ClaudeCode {
				t.Error("ClaudeCode should be true")
			}
		})
	}
}

func TestDefaultProjectProfileRegistry_AllBuiltinsRegistered(t *testing.T) {
	r := devinit.ExportDefaultProjectProfileRegistry()

	builtins := []string{
		"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
		"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
	}
	for _, name := range builtins {
		p, ok := r.Get(name)
		if !ok {
			t.Errorf("built-in profile %q not found in DefaultProjectProfileRegistry", name)
			continue
		}
		if p.Description == "" {
			t.Errorf("built-in profile %q has empty description", name)
		}
		if len(p.Languages) == 0 {
			t.Errorf("built-in profile %q has no languages", name)
		}
	}

	names := r.Names()
	if len(names) != len(builtins) {
		t.Errorf("Names length = %d, want %d", len(names), len(builtins))
	}
}

func TestDefaultProjectProfileRegistry_InsertionOrder(t *testing.T) {
	r := devinit.ExportDefaultProjectProfileRegistry()

	want := []string{
		"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
		"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
	}
	names := r.Names()
	if len(names) != len(want) {
		t.Fatalf("Names length = %d, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("Names[%d] = %q, want %q", i, n, want[i])
		}
	}
}
