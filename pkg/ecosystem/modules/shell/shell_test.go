package shell_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/shell"
)

// newModule returns a fresh Module for testing.
func newModule() *shell.Module {
	return &shell.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*shell.Module)(nil)
	var _ ecosystem.PackageProvider = (*shell.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "shell", "Bash/Shell", 2)
}

// --- Detection tests ---

func TestDetect_ShFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash\necho hello\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence < ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want >= Probable", r.Confidence)
	}
	if len(r.Evidence) == 0 {
		t.Error("expected non-empty Evidence")
	}

	foundSh := false
	for _, e := range r.Evidence {
		if strings.Contains(e, ".sh") {
			foundSh = true
		}
	}
	if !foundSh {
		t.Error("Evidence should mention .sh files")
	}
}

// TestDetect_ScriptsDir verifies scripts/ only indicates shell when it holds
// shell scripts: the directory commonly contains Python or JS helpers.
func TestDetect_ScriptsDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		files        []string
		wantDetected bool
	}{
		{"empty scripts dir", nil, false},
		{"python helper only", []string{"x.py"}, false},
		{"sh script", []string{"build.sh"}, true},
		{"bash script", []string{"setup.bash"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, "scripts", f), []byte("#!/bin/sh\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			r := newModule().Detect(dir)
			if r.Detected != tt.wantDetected {
				t.Fatalf("Detected = %v, want %v (evidence %v)", r.Detected, tt.wantDetected, r.Evidence)
			}
			if !tt.wantDetected {
				return
			}
			if r.Confidence != ecosystem.ConfidenceProbable {
				t.Errorf("Confidence = %v, want Probable", r.Confidence)
			}
			found := false
			for _, e := range r.Evidence {
				if strings.Contains(e, "scripts/") {
					found = true
				}
			}
			if !found {
				t.Errorf("Evidence %v should mention scripts/", r.Evidence)
			}
		})
	}
}

func TestDetect_Envrc(t *testing.T) {
	dir := t.TempDir()
	// .envrc alone is evidence but not a detection trigger per the code;
	// it only adds evidence if something else is also detected.
	// Create a .sh file to trigger detection, then verify .envrc evidence.
	if err := os.WriteFile(filepath.Join(dir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".envrc"), []byte("use nix\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}

	foundEnvrc := false
	for _, e := range r.Evidence {
		if strings.Contains(e, ".envrc") {
			foundEnvrc = true
		}
	}
	if !foundEnvrc {
		t.Error("Evidence should mention .envrc")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages(t *testing.T) {
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	expected := []string{"shellcheck", "shfmt"}
	if len(pkgs) != len(expected) {
		t.Fatalf("DevenvPackages() returned %d packages, want %d", len(pkgs), len(expected))
	}
	for i, pkg := range pkgs {
		if pkg != expected[i] {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkg, expected[i])
		}
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_Empty(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag != "" {
		t.Errorf("DevenvNixFragment() = %q, want empty string (packages moved to DevenvPackages)", frag)
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}

	ids := make(map[string]bool)
	for _, h := range hooks {
		ids[h.ID] = true
	}
	if !ids["shellcheck"] {
		t.Error("missing hook with ID shellcheck")
	}
	if !ids["shfmt"] {
		t.Error("missing hook with ID shfmt")
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 4 {
		t.Fatalf("DenyRules() returned %d rules, want 4", len(rules))
	}
}

// --- CICommands tests ---

func TestCICommands(t *testing.T) {
	m := newModule()
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 2 {
		t.Fatalf("CICommands() returned %d commands, want 2", len(cmds))
	}
}

// TestCICommands_BashSyntaxCheckFailsOnAnyFile runs the bash-syntax-check
// command: it must fail when any script, not only the first one find lists,
// has a syntax error (F436: `bash -n a.sh b.sh` checks only a.sh).
func TestCICommands_BashSyntaxCheckFailsOnAnyFile(t *testing.T) {
	t.Parallel()

	var command string
	for _, c := range newModule().CICommands(ecosystem.ModuleConfig{}) {
		if c.Name == "bash-syntax-check" {
			command = c.Command
		}
	}
	if command == "" {
		t.Fatal("no bash-syntax-check CI command")
	}

	tests := []struct {
		name     string
		files    map[string]string
		wantExit bool
	}{
		{name: "all valid", files: map[string]string{"a.sh": "echo a\n", "sub/b.sh": "echo b\n"}},
		{name: "no scripts", files: map[string]string{"README": "x\n"}},
		{name: "broken first", files: map[string]string{"a.sh": "if then\n", "b.sh": "echo b\n", "c.sh": "echo c\n"}, wantExit: true},
		{name: "broken last", files: map[string]string{"a.sh": "echo a\n", "b.sh": "echo b\n", "z/zz.sh": "if then\n"}, wantExit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := shelltest.WriteTree(t, t.TempDir(), tt.files)
			res := shelltest.Run(t, dir, command, nil)
			if (res.Exit != 0) != tt.wantExit {
				t.Errorf("exit = %d, want failure %v; output:\n%s", res.Exit, tt.wantExit, res.Output)
			}
		})
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if pms != nil {
		t.Errorf("PackageManagers() = %v, want nil (shell has no package manager)", pms)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs(t *testing.T) {
	m := newModule()
	files := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if files != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", files)
	}
}
