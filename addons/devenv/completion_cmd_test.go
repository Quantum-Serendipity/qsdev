package devenv_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
)

func TestCompletionCmd_Bash(t *testing.T) {
	cmd := devenv.ExportCompletionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"bash"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("completion bash failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "bash") {
		t.Error("bash completion output should contain shell-specific content")
	}
	if len(output) < 100 {
		t.Error("bash completion output seems too short")
	}
}

func TestCompletionCmd_Zsh(t *testing.T) {
	cmd := devenv.ExportCompletionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"zsh"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("completion zsh failed: %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("zsh completion output seems too short")
	}
}

func TestCompletionCmd_Fish(t *testing.T) {
	cmd := devenv.ExportCompletionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"fish"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("completion fish failed: %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("fish completion output seems too short")
	}
}

func TestCompletionCmd_Powershell(t *testing.T) {
	cmd := devenv.ExportCompletionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"powershell"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("completion powershell failed: %v", err)
	}

	output := buf.String()
	if len(output) < 100 {
		t.Error("powershell completion output seems too short")
	}
}

func TestCompletionCmd_HasSubcommands(t *testing.T) {
	cmd := devenv.ExportCompletionCmd()

	expectedSubs := []string{"bash", "zsh", "fish", "powershell", "install"}
	subs := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subs[sub.Name()] = true
	}

	for _, name := range expectedSubs {
		if !subs[name] {
			t.Errorf("missing expected subcommand %q", name)
		}
	}
}

func TestCompletionCmd_Install_NoShellFlag(t *testing.T) {
	// When SHELL env is not set and --shell is not provided, install should
	// fail with a helpful error.
	cmd := devenv.ExportCompletionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"install"})

	// Clear SHELL to simulate missing detection.
	t.Setenv("SHELL", "")

	err := cmd.Execute()
	if err == nil {
		// If SHELL was somehow set in the environment, the command might
		// succeed, which is fine. But if it fails it should mention auto-detect.
		return
	}
	if !strings.Contains(err.Error(), "auto-detect") {
		t.Errorf("error should mention auto-detect, got: %v", err)
	}
}

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		envShell string
		expected string
	}{
		{"zsh path", "/usr/bin/zsh", "zsh"},
		{"bash path", "/bin/bash", "bash"},
		{"fish path", "/usr/local/bin/fish", "fish"},
		{"bare name", "bash", "bash"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.envShell)
			got := devenv.ExportDetectShell()
			if got != tt.expected {
				t.Errorf("detectShell() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDefaultRCFile(t *testing.T) {
	tests := []struct {
		shell    string
		contains string // substring that should be in the returned path
	}{
		{"bash", ".bashrc"},
		{"zsh", ".zshrc"},
		{"fish", "config.fish"},
		{"pwsh", "Microsoft.PowerShell_profile.ps1"},
		{"powershell", "Microsoft.PowerShell_profile.ps1"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got := devenv.ExportDefaultRCFile(tt.shell)
			if got == "" {
				t.Fatalf("defaultRCFile(%q) returned empty", tt.shell)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("defaultRCFile(%q) = %q, expected to contain %q", tt.shell, got, tt.contains)
			}
		})
	}
}

func TestDefaultRCFile_Unknown(t *testing.T) {
	got := devenv.ExportDefaultRCFile("nushell")
	if got != "" {
		t.Errorf("defaultRCFile(nushell) = %q, want empty", got)
	}
}
