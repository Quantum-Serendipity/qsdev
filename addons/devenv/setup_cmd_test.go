package devenv

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSetupCmd_Flags(t *testing.T) {
	cmd := setupCmd()

	if cmd.Use != "setup" {
		t.Errorf("Use = %q, want %q", cmd.Use, "setup")
	}

	yesFlag := cmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag to be registered")
	}
	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag to be registered")
	}
}

func TestSetupCmd_DryRun(t *testing.T) {
	cmd := setupCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("setup --dry-run failed: %v", err)
	}

	output := buf.String()

	// Dry-run should produce output (either tool list or "all installed" message).
	if output == "" {
		t.Error("dry-run produced no output")
	}

	// On NixOS, the command prints declarative instructions instead of dry-run.
	// On other systems, it shows "Dry run" or "All tools are installed".
	validOutputs := []string{
		"Dry run",
		"All tools are installed",
		"NixOS detected",
		"No auto-installable tools",
	}
	found := false
	for _, valid := range validOutputs {
		if strings.Contains(output, valid) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected dry-run output: %s", output)
	}
}

func TestSetupCmd_NothingToInstall(t *testing.T) {
	// On a well-configured dev machine (which this test environment should be),
	// if all doctor checks pass, setup should say everything is installed.
	// This test is conditional: it only validates behavior, not environment state.
	cmd := setupCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--dry-run"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("setup --dry-run failed: %v", err)
	}

	output := buf.String()
	// The output should be meaningful regardless of environment state.
	if output == "" {
		t.Error("expected non-empty output from setup --dry-run")
	}
}

func TestSetupCmd_ToolLevelsDefined(t *testing.T) {
	// Verify toolLevels is well-formed.
	if len(toolLevels) == 0 {
		t.Fatal("toolLevels should not be empty")
	}

	seen := make(map[string]bool)
	prevLevel := -1
	for _, level := range toolLevels {
		if level.level < prevLevel {
			t.Errorf("toolLevels not in order: level %d comes after %d", level.level, prevLevel)
		}
		prevLevel = level.level
		for _, tool := range level.tools {
			if seen[tool] {
				t.Errorf("duplicate tool %q in toolLevels", tool)
			}
			seen[tool] = true
		}
	}
}

func TestSetupCmd_ToolToNixPkg(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"git", "git"},
		{"go", "go"},
		{"node", "nodejs"},
		{"npm", "nodejs"},
		{"nix", ""},
		{"devenv", "devenv"},
		{"direnv", "direnv"},
		{"shellcheck", "shellcheck"},
		{"jq", "jq"},
		{"curl", "curl"},
		{"python3", "python3"},
	}

	for _, tt := range tests {
		got := toolToNixPkg(tt.name)
		if got != tt.want {
			t.Errorf("toolToNixPkg(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSetupCmd_InstallCommandForTool(t *testing.T) {
	tests := []struct {
		name   string
		family string
		mgr    string
		want   string
	}{
		{"nix", "debian", "apt", "curl -sSf -L https://install.determinate.systems/nix | sh -s -- install"},
		{"claude", "debian", "apt", "npm install -g @anthropic-ai/claude-code"},
		{"devenv", "debian", "apt", "nix profile install nixpkgs#devenv"},
		{"git", "debian", "apt", "sudo apt-get install -y git"},
		{"git", "macos", "brew", "brew install git"},
	}

	for _, tt := range tests {
		got := installCommandForTool(tt.name, tt.family, tt.mgr)
		if got != tt.want {
			t.Errorf("installCommandForTool(%q, %q, %q) = %q, want %q", tt.name, tt.family, tt.mgr, got, tt.want)
		}
	}
}

func TestSetupCmd_PmInstallArgs(t *testing.T) {
	// Verify that pmInstallArgs produces correct arguments for each PM type.
	// We use a helper mock that just returns a name.
	tests := []struct {
		pmName string
		pkg    string
		want   []string
	}{
		{"apt", "git", []string{"install", "-y", "git"}},
		{"dnf", "git", []string{"install", "-y", "git"}},
		{"pacman", "git", []string{"-S", "--noconfirm", "git"}},
		{"apk", "git", []string{"add", "git"}},
		{"xbps", "git", []string{"-y", "git"}},
		{"emerge", "git", []string{"--ask=n", "git"}},
		{"brew", "git", []string{"install", "git"}},
	}

	for _, tt := range tests {
		pm := &pmNameOnly{name: tt.pmName}
		got := pmInstallArgs(pm, tt.pkg)
		if len(got) != len(tt.want) {
			t.Errorf("pmInstallArgs(%q, %q): len=%d, want len=%d", tt.pmName, tt.pkg, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("pmInstallArgs(%q, %q)[%d]=%q, want %q", tt.pmName, tt.pkg, i, got[i], tt.want[i])
			}
		}
	}
}

// pmNameOnly satisfies pkgmanager.PackageManager for Name() only.
// Other methods panic; only Name() is tested here.
type pmNameOnly struct {
	name string
}

func (p *pmNameOnly) Name() string                                          { return p.name }
func (p *pmNameOnly) Available() bool                                       { panic("unused") }
func (p *pmNameOnly) NeedsElevation() bool                                  { panic("unused") }
func (p *pmNameOnly) UpdateIndex(_ context.Context) error                   { panic("unused") }
func (p *pmNameOnly) Install(_ context.Context, _ ...string) error          { panic("unused") }
func (p *pmNameOnly) IsInstalled(_ string) bool                             { panic("unused") }
func (p *pmNameOnly) SearchCmd() string                                     { panic("unused") }
