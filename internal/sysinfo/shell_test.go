package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveShellRCFile(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	tests := []struct {
		name  string
		shell string
		want  string
	}{
		{"bash", "bash", filepath.Join(home, ".bashrc")},
		{"zsh", "zsh", filepath.Join(home, ".zshrc")},
		{"fish", "fish", filepath.Join(home, ".config", "fish", "config.fish")},
		{"nushell", "nu", filepath.Join(home, ".config", "nushell", "config.nu")},
		{"cmd", "cmd", ""},
		{"empty", "", ""},
		{"unknown", "tcsh", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveShellRCFile(tt.shell)
			if got != tt.want {
				t.Errorf("resolveShellRCFile(%q) = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}

func TestResolveShellRCFile_Pwsh(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	got := resolveShellRCFile("pwsh")

	if runtime.GOOS == "windows" {
		want := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
		if got != want {
			t.Errorf("resolveShellRCFile(\"pwsh\") on Windows = %q, want %q", got, want)
		}
	} else {
		want := filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
		if got != want {
			t.Errorf("resolveShellRCFile(\"pwsh\") on Unix = %q, want %q", got, want)
		}
	}
}

func TestResolveShellRCFile_Powershell(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	got := resolveShellRCFile("powershell")
	want := filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	if got != want {
		t.Errorf("resolveShellRCFile(\"powershell\") = %q, want %q", got, want)
	}
}

func TestDetectCurrentShell(t *testing.T) {
	name, _ := detectCurrentShell()
	if name == "" {
		t.Error("detectCurrentShell() returned empty name")
	}
}
