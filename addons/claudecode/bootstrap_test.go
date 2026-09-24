package claudecode_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
)

// bootstrapNow is the fixed clock the bootstrap tests run at.
var bootstrapNow = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

// bootstrapCutoff is bootstrapNow minus installer.NpmMinReleaseAge.
const bootstrapCutoff = "--before=2026-09-21T12:00:00Z"

func TestInstallClaudeStep_NotNil(t *testing.T) {
	t.Parallel()
	step := claudecode.InstallClaudeStep()
	if step == nil {
		t.Fatal("InstallClaudeStep() returned nil")
	}
}

// TestClaudeSpec_EmbeddedPin covers F110: the bootstrap installs the exact
// Claude Code release the catalog pins, age-gated with npm --before, never
// the registry's latest release.
func TestClaudeSpec_EmbeddedPin(t *testing.T) {
	t.Parallel()

	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	def, ok := cat.BootstrapTool("claude-code")
	if !ok {
		t.Fatal("embedded catalog has no bootstrap_tools.claude-code pin")
	}
	if !installer.IsExactSemver(def.Version) {
		t.Fatalf("bootstrap_tools.claude-code.version = %q, want an exact release", def.Version)
	}

	spec, err := claudecode.ExportClaudeSpec(cat, bootstrapNow)
	if err != nil {
		t.Fatalf("claudeSpec: %v", err)
	}
	want := []string{"npm", "install", "-g"}
	if !def.AllowInstallScripts {
		want = append(want, "--ignore-scripts")
	}
	want = append(want, bootstrapCutoff, "@anthropic-ai/claude-code@"+def.Version)
	if !slices.Equal(spec.InstallCmd, want) {
		t.Errorf("InstallCmd = %q, want %q", spec.InstallCmd, want)
	}
	if spec.Binary != "claude" || spec.ManagerBinary != "npm" {
		t.Errorf("Binary, ManagerBinary = %q, %q; want claude, npm", spec.Binary, spec.ManagerBinary)
	}
}

// TestClaudeSpec_Overlay checks the pin a user defaults overlay sets: a
// different exact release and the script policy are honoured, and an
// unpinned or unsupported entry is refused instead of installing whatever
// the registry serves.
func TestClaudeSpec_Overlay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		overlay string
		wantCmd []string
		wantErr []error
	}{
		{
			name:    "bumped exact release",
			overlay: "bootstrap_tools:\n    claude-code:\n        version: \"2.1.200\"\n        allow_install_scripts: true\n",
			wantCmd: []string{"npm", "install", "-g", bootstrapCutoff, "@anthropic-ai/claude-code@2.1.200"},
		},
		{
			name:    "scripts disallowed",
			overlay: "bootstrap_tools:\n    claude-code:\n        version: \"2.1.200\"\n        allow_install_scripts: false\n",
			wantCmd: []string{"npm", "install", "-g", "--ignore-scripts", bootstrapCutoff, "@anthropic-ai/claude-code@2.1.200"},
		},
		{
			name:    "dist-tag refused",
			overlay: "bootstrap_tools:\n    claude-code:\n        version: latest\n",
			wantErr: []error{installer.ErrBootstrapPin, installer.ErrUnpinned},
		},
		{
			name:    "range refused",
			overlay: "bootstrap_tools:\n    claude-code:\n        version: \"^2.1.0\"\n",
			wantErr: []error{installer.ErrBootstrapPin, installer.ErrUnpinned},
		},
		{
			name:    "unsupported install method",
			overlay: "bootstrap_tools:\n    claude-code:\n        install_method: curl-script\n",
			wantErr: []error{installer.ErrBootstrapPin},
		},
		{
			name:    "option-like package name refused",
			overlay: "bootstrap_tools:\n    claude-code:\n        package_name: \"--registry=https://evil.example/x\"\n",
			wantErr: []error{installer.ErrBootstrapPin},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "defaults.yaml")
			if err := os.WriteFile(path, []byte(tt.overlay), 0o600); err != nil {
				t.Fatalf("writing overlay: %v", err)
			}
			cat, err := catalog.Load(catalog.WithOrgConfigFile(path))
			if err != nil {
				t.Fatalf("catalog.Load: %v", err)
			}

			spec, err := claudecode.ExportClaudeSpec(cat, bootstrapNow)
			if len(tt.wantErr) > 0 {
				for _, want := range tt.wantErr {
					if !errors.Is(err, want) {
						t.Errorf("claudeSpec error = %v, want %v", err, want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("claudeSpec: %v", err)
			}
			if tt.wantCmd != nil && !slices.Equal(spec.InstallCmd, tt.wantCmd) {
				t.Errorf("InstallCmd = %q, want %q", spec.InstallCmd, tt.wantCmd)
			}
			def, _ := cat.BootstrapTool("claude-code")
			if got := slices.Contains(spec.InstallCmd, "--ignore-scripts"); got == def.AllowInstallScripts {
				t.Errorf("--ignore-scripts present = %v with allow_install_scripts = %v", got, def.AllowInstallScripts)
			}
		})
	}
}
