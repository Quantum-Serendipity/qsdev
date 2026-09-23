package devenv_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGenerateDevenvNix_EnabledToolSections is the F030 regression: the
// devenv.nix contribution of an enabled tool must come from the generator, so
// init/update keep it instead of dropping what `enable` inserted. The result
// must also be valid Nix (the hook scripts once used \" escapes, which Nix
// rejects in expression context).
func TestGenerateDevenvNix_EnabledToolSections(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t, goMock())

	tests := []struct {
		name    string
		enabled map[string]bool
		envVars map[string]string
		want    map[string]string // attribute path -> value ("" = any value)
		notWant []string          // attribute paths that must be undefined
	}{
		{
			name:    "no tools enabled emits no tool sections",
			enabled: map[string]bool{},
			notWant: []string{"env.STARSHIP_CONFIG", "git-hooks.hooks.commit-ticket", "git-hooks.hooks.branch-naming", "env.OTEL_SERVICE_NAME"},
		},
		{
			name: "enabled tools emit their sections",
			enabled: map[string]bool{
				"starship-integration": true,
				"commit-ticket":        true,
				"branch-naming":        true,
				"otel-config":          true,
			},
			want: map[string]string{
				"env.STARSHIP_CONFIG":                  `"${config.devenv.root}/.starship.toml"`,
				"git-hooks.hooks.commit-ticket.enable": "true",
				"git-hooks.hooks.branch-naming.enable": "true",
				"env.OTEL_SERVICE_NAME":                "",
			},
		},
		{
			name:    "disabled tools emit nothing",
			enabled: map[string]bool{"starship-integration": false, "commit-ticket": false},
			notWant: []string{"env.STARSHIP_CONFIG", "git-hooks.hooks.commit-ticket"},
		},
		{
			name:    "user env var is not redefined by a tool section",
			enabled: map[string]bool{"otel-config": true},
			envVars: map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://collector:4317"},
			want:    map[string]string{"env.OTEL_EXPORTER_OTLP_ENDPOINT": `"http://collector:4317"`, "env.OTEL_SERVICE_NAME": ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{
				ProjectName:  "demo",
				Languages:    []types.LanguageChoice{{Name: "go"}},
				EnabledTools: tt.enabled,
				EnvVars:      tt.envVars,
			}
			got, err := devenv.GenerateDevenvNix(answers, reg)
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}
			attrs := nixAttrs(t, got.Content)
			for path, value := range tt.want {
				requireNixAttr(t, attrs, path)
				if got := attrs[path]; value != "" && got != value {
					t.Errorf("%s = %s, want %s", path, got, value)
				}
			}
			for _, path := range tt.notWant {
				if hasNixAttr(attrs, path) {
					t.Errorf("devenv.nix unexpectedly defines %s", path)
				}
			}
			requireNixParses(t, got.Content)
		})
	}
}

// requireNixParses checks content is syntactically valid Nix when
// nix-instantiate is available.
func requireNixParses(t *testing.T, content []byte) {
	t.Helper()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		return
	}
	path := filepath.Join(t.TempDir(), "devenv.nix")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(nixInstantiate, "--parse", path).CombinedOutput()
	if err != nil {
		t.Fatalf("generated devenv.nix does not parse: %v\n%s\n--- content ---\n%s", err, out, content)
	}
}
