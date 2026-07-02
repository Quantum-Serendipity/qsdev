package shellintegration

import "testing"

func TestNormalizeShellName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/usr/bin/zsh", "zsh"},
		{"/bin/bash", "bash"},
		{"fish", "fish"},
		{"ZSH", "zsh"},
		{"/usr/local/bin/Fish", "fish"},
		{"pwsh", "pwsh"},
		{`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, "powershell"},
		{`C:\Program Files\PowerShell\7\pwsh.exe`, "pwsh"},
		{"bash.exe", "bash"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeShellName(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeShellName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
