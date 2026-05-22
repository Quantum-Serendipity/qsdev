package shellintegration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathExportLine(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		shell    string
		expected string
	}{
		{
			name:     "bash",
			dir:      "/usr/local/bin",
			shell:    "bash",
			expected: `export PATH="/usr/local/bin:$PATH"`,
		},
		{
			name:     "zsh",
			dir:      "/home/user/.local/bin",
			shell:    "zsh",
			expected: `export PATH="/home/user/.local/bin:$PATH"`,
		},
		{
			name:     "fish",
			dir:      "/opt/bin",
			shell:    "fish",
			expected: "fish_add_path /opt/bin",
		},
		{
			name:     "pwsh",
			dir:      "/usr/local/bin",
			shell:    "pwsh",
			expected: `$env:PATH = "/usr/local/bin" + [IO.Path]::PathSeparator + $env:PATH`,
		},
		{
			name:     "powershell",
			dir:      "/usr/local/bin",
			shell:    "powershell",
			expected: `$env:PATH = "/usr/local/bin" + [IO.Path]::PathSeparator + $env:PATH`,
		},
		{
			name:     "full path shell",
			dir:      "/usr/local/bin",
			shell:    "/usr/bin/zsh",
			expected: `export PATH="/usr/local/bin:$PATH"`,
		},
		{
			name:     "unknown shell defaults to POSIX",
			dir:      "/usr/local/bin",
			shell:    "sh",
			expected: `export PATH="/usr/local/bin:$PATH"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathExportLine(tt.dir, tt.shell)
			if got != tt.expected {
				t.Errorf("pathExportLine(%q, %q) = %q, want %q", tt.dir, tt.shell, got, tt.expected)
			}
		})
	}
}

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

func TestEnsurePath_CreatesNewRCFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")
	dir := "/home/user/.local/bin"

	err := EnsurePath(dir, "bash", rcFile)
	if err != nil {
		t.Fatalf("EnsurePath failed: %v", err)
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("reading RC file: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, pathMarkerStart()) {
		t.Error("RC file should contain start marker")
	}
	if !strings.Contains(s, pathMarkerEnd()) {
		t.Error("RC file should contain end marker")
	}
	if !strings.Contains(s, `export PATH="/home/user/.local/bin:$PATH"`) {
		t.Error("RC file should contain PATH export line")
	}
}

func TestEnsurePath_IdempotentOnExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	// Write an initial RC file with some existing content.
	initialContent := "# existing config\nalias ll='ls -la'\n"
	if err := os.WriteFile(rcFile, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := "/opt/qsdev/bin"

	// First call.
	if err := EnsurePath(dir, "bash", rcFile); err != nil {
		t.Fatalf("first EnsurePath failed: %v", err)
	}

	first, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	// Second call should be idempotent.
	if err := EnsurePath(dir, "bash", rcFile); err != nil {
		t.Fatalf("second EnsurePath failed: %v", err)
	}

	second, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Errorf("second call changed the file.\nFirst:\n%s\nSecond:\n%s", first, second)
	}

	// Verify existing content is preserved.
	if !strings.Contains(string(second), "alias ll='ls -la'") {
		t.Error("existing config should be preserved")
	}
}

func TestEnsurePath_UpdatesExistingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".zshrc")

	// Write RC file with an existing qsdev PATH block.
	existing := "# some config\n" +
		pathMarkerStart() + "\n" +
		`export PATH="/old/path:$PATH"` + "\n" +
		pathMarkerEnd() + "\n" +
		"# more config\n"
	if err := os.WriteFile(rcFile, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	// Update with new path.
	if err := EnsurePath("/new/path", "zsh", rcFile); err != nil {
		t.Fatalf("EnsurePath failed: %v", err)
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if strings.Contains(s, "/old/path") {
		t.Error("old path should have been replaced")
	}
	if !strings.Contains(s, "/new/path") {
		t.Error("new path should be present")
	}
	if !strings.Contains(s, "# some config") {
		t.Error("surrounding config should be preserved")
	}
	if !strings.Contains(s, "# more config") {
		t.Error("surrounding config should be preserved")
	}
}

func TestEnsurePath_FishSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, "config.fish")

	if err := EnsurePath("/opt/bin", "fish", rcFile); err != nil {
		t.Fatalf("EnsurePath failed: %v", err)
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "fish_add_path /opt/bin") {
		t.Error("fish config should use fish_add_path")
	}
}

func TestEnsurePath_PowershellSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, "profile.ps1")

	if err := EnsurePath("/opt/bin", "pwsh", rcFile); err != nil {
		t.Fatalf("EnsurePath failed: %v", err)
	}

	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), `$env:PATH = "/opt/bin"`) {
		t.Error("powershell config should use $env:PATH syntax")
	}
}

func TestEnsurePath_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, ".bashrc")

	err := EnsurePath("", "bash", rcFile)
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
	if !strings.Contains(err.Error(), "dir must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnsurePath_EmptyRCFile(t *testing.T) {
	err := EnsurePath("/opt/bin", "bash", "")
	if err == nil {
		t.Fatal("expected error for empty rcFile")
	}
	if !strings.Contains(err.Error(), "rcFile must not be empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEnsurePath_CreatesParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	rcFile := filepath.Join(tmpDir, "subdir", "nested", ".bashrc")

	err := EnsurePath("/opt/bin", "bash", rcFile)
	if err != nil {
		t.Fatalf("EnsurePath failed: %v", err)
	}

	if _, err := os.Stat(rcFile); err != nil {
		t.Errorf("RC file should have been created: %v", err)
	}
}
