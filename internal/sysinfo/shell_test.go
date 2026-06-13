package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsKnownShell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want bool
	}{
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"pwsh", true},
		{"powershell", true},
		{"nu", true},
		{"sh", true},
		{"dash", true},
		{"ksh", true},
		{"tcsh", true},
		{"csh", true},
		{"cmd", false},
		{"python", false},
		{"node", false},
		{"", false},
		{"BASH", false},
		{"Zsh", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isKnownShell(tc.name)
			if got != tc.want {
				t.Errorf("isKnownShell(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestResolveShellRCFile(t *testing.T) {
	t.Parallel()

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
		{"unknown shell dash", "dash", ""},
		{"unknown shell ksh", "ksh", ""},
		{"unknown shell tcsh", "tcsh", ""},
		{"unknown shell csh", "csh", ""},
		{"unknown shell sh", "sh", ""},
		{"nonsense", "notashell", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveShellRCFile(tt.shell)
			if got != tt.want {
				t.Errorf("resolveShellRCFile(%q) = %q, want %q", tt.shell, got, tt.want)
			}
		})
	}
}

func TestResolveShellRCFile_Pwsh(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

func TestDetectShell_PopulatesAllFields(t *testing.T) {
	info := &OSInfo{}
	detectShell(info)

	if info.Shell == "" {
		t.Error("detectShell did not set Shell")
	}
	// ShellPath may be empty in some CI environments, but Shell should
	// correspond to a known shell or "unknown".
	if info.Shell != "unknown" && info.ShellPath == "" {
		t.Errorf("Shell=%q but ShellPath is empty", info.Shell)
	}
}
