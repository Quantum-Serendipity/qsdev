package devinit_test

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// builtinRegistry builds the built-in profile registry from the default
// catalog, failing the test on any error.
func builtinRegistry(t *testing.T) *devinit.ExportProjectProfileRegistry {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	r, err := devinit.ExportCatalogProjectProfiles(cat)
	if err != nil {
		t.Fatalf("building built-in profiles: %v", err)
	}
	return r
}

// builtinProfile returns a built-in profile from the default registry.
func builtinProfile(t *testing.T, name string) devinit.ExportProfile {
	t.Helper()
	p, ok := builtinRegistry(t).Get(name)
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

func TestCatalogProjectProfiles_AllBuiltinsRegistered(t *testing.T) {
	r := builtinRegistry(t)

	builtins := []string{
		"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
		"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
	}
	for _, name := range builtins {
		p, ok := r.Get(name)
		if !ok {
			t.Errorf("built-in profile %q not found in the catalog profile registry", name)
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

// TestProfileNamesDerivedFromCatalog verifies the built-in profile set is the
// catalog's project_profiles keys in name order, not a curated list that can
// drift from the catalog.
func TestProfileNamesDerivedFromCatalog(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	r, err := devinit.ExportCatalogProjectProfiles(cat)
	if err != nil {
		t.Fatal(err)
	}
	want := slices.Sorted(maps.Keys(cat.ProjectProfiles()))
	if got := r.Names(); !slices.Equal(got, want) {
		t.Errorf("Names = %q, want %q", got, want)
	}
}

// TestOverlayProjectProfileSelectable verifies a profile added by an org
// overlay is registered and listed, so --profile can select it.
func TestOverlayProjectProfileSelectable(t *testing.T) {
	t.Parallel()
	overlay := `project_profiles:
  go-custom:
    description: 'Custom Go service'
    tier: full
    languages:
      - name: go
    services: []
    direnv: true
    claude_code: true
    permission_level: standard
`
	f := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(f, []byte(overlay), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := catalog.Load(catalog.WithOrgConfigFile(f))
	if err != nil {
		t.Fatalf("loading catalog with overlay: %v", err)
	}
	r, err := devinit.ExportCatalogProjectProfiles(cat)
	if err != nil {
		t.Fatalf("building profiles: %v", err)
	}
	p, ok := r.Get("go-custom")
	if !ok {
		t.Fatalf("overlay profile go-custom not registered; names = %q", r.Names())
	}
	if p.Description != "Custom Go service" {
		t.Errorf("go-custom description = %q", p.Description)
	}
	if !slices.Contains(r.Names(), "go-custom") {
		t.Errorf("Names = %q, want go-custom listed", r.Names())
	}
	if _, ok := r.Get("go-web"); !ok {
		t.Error("built-in go-web missing next to the overlay profile")
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

// TestNewProjectProfilesCatalogErrorReturnedNotLogged: a catalog that fails
// to load is returned by newProjectProfiles to the command that needs the
// profiles, and nothing is logged. It mutates the process-global catalog, so
// it is not parallel.
func TestNewProjectProfilesCatalogErrorReturnedNotLogged(t *testing.T) {
	dir := t.TempDir()
	bad := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(bad), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("keep_vars: [AWS_SECRET_ACCESS_KEY]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(dir); err != nil {
		t.Fatal(err)
	}
	prev := slog.Default()
	h := &recordingHandler{}
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() {
		slog.SetDefault(prev)
		catalog.ResetDefault()
	})

	r, err := devinit.ExportNewProjectProfiles(nil)
	if err == nil || !strings.Contains(err.Error(), "loading built-in project profiles") {
		t.Fatalf("newProjectProfiles err = %v, want a built-in profile loading error", err)
	}
	if r != nil {
		t.Errorf("registry = %q, want nil on a catalog failure", r.Names())
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, rec := range h.records {
		t.Errorf("catalog failure logged at %s: %s", rec.Level, rec.Message)
	}
}

// TestEmbedderProfileCollisionReported verifies an embedder profile whose
// name collides with a built-in is returned as an error (not logged and
// skipped), while non-colliding embedder profiles register next to the
// built-ins. TestRegisterProfiles checks the built-in itself is kept.
func TestEmbedderProfileCollisionReported(t *testing.T) {
	t.Parallel()
	custom := devinit.ExportProfile{Description: "custom"}

	reg, err := devinit.ExportNewProjectProfiles(map[string]devinit.ExportProfile{"my-team": custom})
	if err != nil {
		t.Fatalf("non-colliding embedder profile: %v", err)
	}
	if p, ok := reg.Get("my-team"); !ok || p.Description != "custom" {
		t.Errorf("embedder profile my-team = %+v, %v; want registered", p, ok)
	}
	if _, ok := reg.Get("go-web"); !ok {
		t.Error("built-in go-web missing next to an embedder profile")
	}

	_, err = devinit.ExportNewProjectProfiles(map[string]devinit.ExportProfile{"go-web": custom, "my-team": custom})
	if err == nil || !strings.Contains(err.Error(), `registering profile "go-web"`) {
		t.Fatalf("colliding embedder profile: err = %v, want a go-web registration error", err)
	}
}

const lazyProfilesHelperEnv = "QSDEV_DEVINIT_LAZY_PROFILES_HELPER"

// TestProjectProfilesLazyHelper is not a real test: TestProjectProfilesLazy
// runs it in a fresh process, where nothing has loaded the catalog yet.
func TestProjectProfilesLazyHelper(t *testing.T) {
	if os.Getenv(lazyProfilesHelperEnv) != "1" {
		t.Skip("helper process for TestProjectProfilesLazy")
	}
	if err := devinit.ExportInitialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	root := t.TempDir()
	if err := catalog.SetProjectRoot(root); err != nil {
		t.Fatalf("initialize loaded the catalog: SetProjectRoot = %v", err)
	}
	if _, err := devinit.ExportProjectProfiles(); err != nil {
		t.Fatalf("projectProfiles: %v", err)
	}
	if err := catalog.SetProjectRoot(t.TempDir()); !errors.Is(err, catalog.ErrCatalogAlreadyLoaded) {
		t.Fatalf("projectProfiles did not load the catalog: SetProjectRoot = %v", err)
	}
}

// TestProjectProfilesLazy verifies building the devinit commands does not
// load the catalog: the project-type profiles load on first use, after the
// command's project root is known.
func TestProjectProfilesLazy(t *testing.T) {
	t.Parallel()
	cmd := exec.Command(os.Args[0], "-test.run=^TestProjectProfilesLazyHelper$", "-test.v")
	cmd.Env = append(os.Environ(), lazyProfilesHelperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--- PASS: TestProjectProfilesLazyHelper") {
		t.Fatalf("helper did not run:\n%s", out)
	}
}
