package devinit_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// builtinProfile returns a built-in profile from the default registry.
func builtinProfile(t *testing.T, name string) devinit.ExportProfile {
	t.Helper()
	p, ok := devinit.ExportDefaultProjectProfileRegistry().Get(name)
	if !ok {
		t.Fatalf("built-in profile %q not registered", name)
	}
	return p
}

func TestBuiltinProfiles_HaveRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		profile devinit.ExportProfile
	}{
		{"go-web", builtinProfile(t, "go-web")},
		{"ts-fullstack", builtinProfile(t, "ts-fullstack")},
		{"python-data", builtinProfile(t, "python-data")},
		{"rust-cli", builtinProfile(t, "rust-cli")},
		{"java-web", builtinProfile(t, "java-web")},
		{"python-web", builtinProfile(t, "python-web")},
		{"ts-backend", builtinProfile(t, "ts-backend")},
		{"elixir-web", builtinProfile(t, "elixir-web")},
		{"rust-web", builtinProfile(t, "rust-web")},
		{"dotnet-web", builtinProfile(t, "dotnet-web")},
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
	p := builtinProfile(t, "go-web")

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
	p := builtinProfile(t, "ts-fullstack")

	if len(p.Languages) != 1 || p.Languages[0].Name != "javascript" {
		t.Errorf("Languages = %v, want [{javascript pnpm}]", p.Languages)
	}
	if p.Languages[0].PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q", p.Languages[0].PackageManager, "pnpm")
	}
	if p.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "standard")
	}
	// Should have the safety-block hook.
	if len(p.Hooks) != 1 {
		t.Errorf("Hooks length = %d, want 1", len(p.Hooks))
	}
}

func TestBuiltinProfiles_PythonData(t *testing.T) {
	p := builtinProfile(t, "python-data")

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
	p := builtinProfile(t, "rust-cli")

	if len(p.Languages) != 1 || p.Languages[0].Name != "rust" {
		t.Errorf("Languages = %v, want [{rust}]", p.Languages)
	}
	if len(p.Services) != 0 {
		t.Errorf("Services length = %d, want 0", len(p.Services))
	}
	if p.PermissionLevel != "minimal" {
		t.Errorf("PermissionLevel = %q, want %q", p.PermissionLevel, "minimal")
	}
	// Should have the safety-block hook.
	if len(p.Hooks) != 1 {
		t.Errorf("Hooks length = %d, want 1", len(p.Hooks))
	}
}

func TestBuiltinProfiles_WebProfiles_HaveServices(t *testing.T) {
	webProfiles := []struct {
		name    string
		profile devinit.ExportProfile
	}{
		{"java-web", builtinProfile(t, "java-web")},
		{"python-web", builtinProfile(t, "python-web")},
		{"ts-backend", builtinProfile(t, "ts-backend")},
		{"elixir-web", builtinProfile(t, "elixir-web")},
		{"rust-web", builtinProfile(t, "rust-web")},
		{"dotnet-web", builtinProfile(t, "dotnet-web")},
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

// recordingHandler is a slog handler that keeps every record it handles.
type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

// TestDefaultProjectProfileRegistry_CatalogErrorNotLoggedAsError: a catalog
// that fails to load is reported to the user by the root catalog gate or by
// check, so building the profile registry (which every command does at
// addon initialization) logs it at debug, not as a startup ERROR line on
// every command. It mutates the process-global catalog, so it is not
// parallel.
func TestDefaultProjectProfileRegistry_CatalogErrorNotLoggedAsError(t *testing.T) {
	dir := t.TempDir()
	bad := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("keep_vars: [AWS_SECRET_ACCESS_KEY]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.ResetDefault()
	catalog.SetProjectRoot(dir)
	prev := slog.Default()
	h := &recordingHandler{}
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() {
		slog.SetDefault(prev)
		catalog.ResetDefault()
	})

	if r := devinit.ExportDefaultProjectProfileRegistry(); len(r.Names()) != 0 {
		t.Errorf("registered %q without a catalog", r.Names())
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var debug int
	for _, rec := range h.records {
		switch {
		case rec.Level >= slog.LevelWarn:
			t.Errorf("catalog failure logged at %s: %s", rec.Level, rec.Message)
		case rec.Level == slog.LevelDebug:
			debug++
		}
	}
	if debug == 0 {
		t.Error("catalog failure not logged at debug")
	}
}
