//go:build windows

package sysinfo

import "testing"

func TestShellFromImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		image string
		want  string
	}{
		{"pwsh.exe", "pwsh"},
		{"PowerShell.exe", "powershell"},
		{"cmd.exe", "cmd"},
		{"bash.exe", "bash"},
		{`C:\Program Files\Git\usr\bin\bash.exe`, "bash"},
		{"WindowsTerminal.exe", ""},
		{"Code.exe", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			t.Parallel()
			if got := shellFromImage(tt.image); got != tt.want {
				t.Errorf("shellFromImage(%q) = %q, want %q", tt.image, got, tt.want)
			}
		})
	}
}

// TestShellFromEnv verifies that the machine-wide PSModulePath inherited by
// cmd.exe and Git Bash is not mistaken for a PowerShell session.
func TestShellFromEnv(t *testing.T) {
	t.Parallel()

	const machinePSModulePath = `C:\Program Files\WindowsPowerShell\Modules;C:\WINDOWS\system32\WindowsPowerShell\v1.0\Modules`

	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "git bash inherits machine PSModulePath",
			env:  map[string]string{"MSYSTEM": "MINGW64", "PSModulePath": machinePSModulePath},
			want: "bash",
		},
		{
			name: "cmd inherits machine PSModulePath",
			env:  map[string]string{"PSModulePath": machinePSModulePath},
			want: "cmd",
		},
		{
			name: "powershell core session",
			env:  map[string]string{"PSModulePath": `C:\Users\dev\Documents\PowerShell\Modules;C:\Program Files\PowerShell\Modules;` + machinePSModulePath},
			want: "pwsh",
		},
		{
			name: "windows powershell session",
			env:  map[string]string{"PSModulePath": `C:\Users\dev\Documents\WindowsPowerShell\Modules;` + machinePSModulePath},
			want: "powershell",
		},
		{
			name: "SHELL set by a unix-like environment",
			env:  map[string]string{"SHELL": "/usr/bin/zsh", "PSModulePath": machinePSModulePath},
			want: "zsh",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getenv := func(k string) string { return tt.env[k] }
			if got := shellFromEnv(getenv); got != tt.want {
				t.Errorf("shellFromEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
