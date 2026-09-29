package privilege

import (
	"context"
	"runtime"
	"testing"
)

func TestIsElevated_InverseOfNeedsElevation(t *testing.T) {
	if NeedsElevation() == IsElevated() {
		t.Error("NeedsElevation() and IsElevated() must be inverses")
	}
}

func TestDetectElevationTool_ReturnsRecognized(t *testing.T) {
	tool := DetectElevationTool()
	valid := map[string]bool{
		"": true, "sudo": true, "doas": true, "pkexec": true,
		"gsudo": true, "powershell": true,
	}
	if !valid[tool] {
		t.Errorf("DetectElevationTool() = %q, not recognized", tool)
	}
}

func TestBatchElevatedInstall_EmptyPackages(t *testing.T) {
	err := BatchElevatedInstall(context.Background(), "apt-get", []string{"install", "-y"}, nil)
	if err != nil {
		t.Errorf("BatchElevatedInstall with nil packages = %v, want nil", err)
	}
}

func TestBatchElevatedInstall_EmptySlice(t *testing.T) {
	err := BatchElevatedInstall(context.Background(), "apt-get", []string{"install", "-y"}, []string{})
	if err != nil {
		t.Errorf("BatchElevatedInstall with empty packages = %v, want nil", err)
	}
}

func TestQuotePowerShellArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean", "package-name", "'package-name'"},
		{"single quote", "test'inject", "'test''inject'"},
		{"injection attempt", "test'; Get-Process; '", "'test''; Get-Process; '''"},
		{"multiple quotes", "a'b'c", "'a''b''c'"},
		{"empty", "", "''"},
		{"left single quotation mark", "a\u2018; Get-Process", "'a\u2018\u2018; Get-Process'"},
		{"right single quotation mark", "a\u2019; Get-Process", "'a\u2019\u2019; Get-Process'"},
		{"single low-9 quotation mark", "a\u201A; Get-Process", "'a\u201A\u201A; Get-Process'"},
		{"single high-reversed-9 quotation mark", "a\u201B; Get-Process", "'a\u201B\u201B; Get-Process'"},
		{"mixed quote kinds", "'\u2019", "'''\u2019\u2019'"},
		{"double quotes are literal", `say "hi"`, `'say "hi"'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := quotePowerShellArg(tt.input); got != tt.expected {
				t.Errorf("quotePowerShellArg(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWindowsCommandLineArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain", "install", "install"},
		{"empty", "", `""`},
		{"space", "my pkg", `"my pkg"`},
		{"tab", "a\tb", "\"a\tb\""},
		{"backslashes only are literal", `C:\tools\bin`, `C:\tools\bin`},
		{"trailing backslash with space", `C:\Program Files\x\`, `"C:\Program Files\x\\"`},
		{"embedded quotes", `say "hi"`, `"say \"hi\""`},
		{"backslash before quote", `a\"b`, `"a\\\"b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := windowsCommandLineArg(tt.input); got != tt.want {
				t.Errorf("windowsCommandLineArg(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestPowerShellElevationCommand(t *testing.T) {
	t.Parallel()

	const prefix = "$ErrorActionPreference = 'Stop'; $p = Start-Process -Verb RunAs -Wait -PassThru -FilePath "
	const suffix = "; exit $p.ExitCode"

	tests := []struct {
		name string
		pm   string
		args []string
		want string
	}{
		{
			name: "no arguments omits -ArgumentList",
			pm:   "choco",
			want: prefix + "'choco'" + suffix,
		},
		{
			name: "arguments are quoted for PowerShell and the Windows command line",
			pm:   "choco",
			args: []string{"install", "-y", "my pkg"},
			want: prefix + `'choco' -ArgumentList 'install','-y','"my pkg"'` + suffix,
		},
		{
			name: "injection through a quote stays inside the literal",
			pm:   `C:\Program Files\pm.exe`,
			args: []string{"x'; Get-Process; '"},
			want: prefix + `'C:\Program Files\pm.exe' -ArgumentList '"x''; Get-Process; ''"'` + suffix,
		},
		{
			name: "injection through a typographic quote stays inside the literal",
			pm:   "choco",
			args: []string{"x\u2019;Get-Process"},
			want: prefix + "'choco' -ArgumentList 'x\u2019\u2019;Get-Process'" + suffix,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := powerShellElevationCommand(tt.pm, tt.args); got != tt.want {
				t.Errorf("powerShellElevationCommand() =\n  %s\nwant\n  %s", got, tt.want)
			}
		})
	}
}

func TestDetectElevationTool_Linux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}
	tool := DetectElevationTool()
	// On most Linux systems, sudo should be available
	if tool == "" {
		t.Log("No elevation tool found — expected on minimal systems")
	}
}
