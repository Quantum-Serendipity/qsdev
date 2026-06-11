package evasion

import "testing"

func TestCheck_Obfuscation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tool     string
		command  string
		file     string
		blocked  bool
		category string
	}{
		{
			name:     "base64 decode piped to bash",
			tool:     "Bash",
			command:  "echo aGVsbG8= | base64 -d | bash",
			blocked:  true,
			category: "obfuscation",
		},
		{
			name:     "base64 decode piped to sh",
			tool:     "Bash",
			command:  "base64 --decode payload.b64 | sh",
			blocked:  true,
			category: "obfuscation",
		},
		{
			name:     "printf hex piped to sh",
			tool:     "Bash",
			command:  `printf '\x68\x65\x6c' | sh`,
			blocked:  true,
			category: "obfuscation",
		},
		{
			name:     "eval with variable expansion",
			tool:     "Bash",
			command:  `eval "$HIDDEN_CMD"`,
			blocked:  true,
			category: "obfuscation",
		},
		{
			name:    "base64 encoding not decoding",
			tool:    "Bash",
			command: "echo hello | base64",
			blocked: false,
		},
		{
			name:    "base64 decode without pipe to shell",
			tool:    "Bash",
			command: "base64 -d file.txt",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, tt.file)
			if blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.blocked, reason)
			}
			if tt.blocked && category != tt.category {
				t.Errorf("category = %q, want %q", category, tt.category)
			}
		})
	}
}

func TestCheck_Hardlink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tool     string
		command  string
		file     string
		blocked  bool
		category string
	}{
		{
			name:     "hardlink targeting .claude/ path",
			tool:     "Bash",
			command:  "ln ~/.claude/settings.json /tmp/copy",
			blocked:  true,
			category: "hardlink",
		},
		{
			name:     "hardlink targeting .qsdev/ path",
			tool:     "Bash",
			command:  "ln .qsdev/config.yaml /tmp/grab",
			blocked:  true,
			category: "hardlink",
		},
		{
			name:    "symlink is allowed",
			tool:    "Bash",
			command: "ln -s /tmp/a /tmp/b",
			blocked: false,
		},
		{
			name:    "hardlink to non-protected path",
			tool:    "Bash",
			command: "ln /tmp/a /tmp/b",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, tt.file)
			if blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.blocked, reason)
			}
			if tt.blocked && category != tt.category {
				t.Errorf("category = %q, want %q", category, tt.category)
			}
		})
	}
}

func TestCheck_FDTricks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tool     string
		command  string
		file     string
		blocked  bool
		category string
	}{
		{
			name:     "filePath with /dev/fd/",
			tool:     "Write",
			file:     "/dev/fd/3",
			blocked:  true,
			category: "fdtricks",
		},
		{
			name:     "command with /proc/self/fd/",
			tool:     "Bash",
			command:  "cat /proc/self/fd/5",
			blocked:  true,
			category: "fdtricks",
		},
		{
			name:     "process substitution with protected path",
			tool:     "Bash",
			command:  "diff <(cat .claude/settings.json) /tmp/other",
			blocked:  true,
			category: "fdtricks",
		},
		{
			name:     "exec fd redirect to protected path",
			tool:     "Bash",
			command:  "exec 3> .qsdev/audit/log",
			blocked:  true,
			category: "fdtricks",
		},
		{
			name:    "normal file path",
			tool:    "Write",
			file:    "/tmp/normal",
			blocked: false,
		},
		{
			name:    "process substitution without protected path",
			tool:    "Bash",
			command: "diff <(cat /tmp/a) /tmp/b",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, tt.file)
			if blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.blocked, reason)
			}
			if tt.blocked && category != tt.category {
				t.Errorf("category = %q, want %q", category, tt.category)
			}
		})
	}
}

func TestCheck_ProcRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tool     string
		command  string
		file     string
		blocked  bool
		category string
	}{
		{
			name:     "filePath with /proc/self/root/",
			tool:     "Read",
			file:     "/proc/self/root/home/user/.claude/settings.json",
			blocked:  true,
			category: "procroot",
		},
		{
			name:     "filePath with /proc/<pid>/root/",
			tool:     "Read",
			file:     "/proc/1234/root/etc/passwd",
			blocked:  true,
			category: "procroot",
		},
		{
			name:     "command with /proc/self/environ",
			tool:     "Bash",
			command:  "cat /proc/self/environ",
			blocked:  true,
			category: "procroot",
		},
		{
			name:     "command with /proc/self/cmdline",
			tool:     "Bash",
			command:  "strings /proc/self/cmdline",
			blocked:  true,
			category: "procroot",
		},
		{
			name:    "safe proc path",
			tool:    "Bash",
			command: "cat /proc/cpuinfo",
			blocked: false,
		},
		{
			name:    "safe proc meminfo path",
			tool:    "Read",
			file:    "/proc/meminfo",
			blocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, tt.file)
			if blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.blocked, reason)
			}
			if tt.blocked && category != tt.category {
				t.Errorf("category = %q, want %q", category, tt.category)
			}
		})
	}
}

func TestCheck_NoEvasion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		command string
		file    string
	}{
		{
			name:    "simple ls command",
			tool:    "Bash",
			command: "ls -la /tmp",
		},
		{
			name:    "git status",
			tool:    "Bash",
			command: "git status",
		},
		{
			name: "write to normal file",
			tool: "Write",
			file: "/home/user/project/main.go",
		},
		{
			name:    "read normal file",
			tool:    "Read",
			file:    "/etc/hosts",
			command: "",
		},
		{
			name:    "empty everything",
			tool:    "",
			command: "",
			file:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, category, reason := Check(tt.tool, tt.command, tt.file)
			if blocked {
				t.Errorf("expected no evasion but got blocked: category=%q reason=%q", category, reason)
			}
			if category != "" {
				t.Errorf("category = %q, want empty", category)
			}
			if reason != "" {
				t.Errorf("reason = %q, want empty", reason)
			}
		})
	}
}

func TestCheck_NonBashToolSkipsCommandChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tool    string
		command string
		file    string
		blocked bool
	}{
		{
			name:    "Write tool with obfuscation in command field is ignored",
			tool:    "Write",
			command: "echo aGVsbG8= | base64 -d | bash",
			file:    "/tmp/safe.txt",
			blocked: false,
		},
		{
			name:    "Read tool with eval in command field is ignored",
			tool:    "Read",
			command: `eval "$HIDDEN_CMD"`,
			file:    "/tmp/safe.txt",
			blocked: false,
		},
		{
			name:    "Write tool still checks filePath for fd tricks",
			tool:    "Write",
			command: "some content",
			file:    "/dev/fd/3",
			blocked: true,
		},
		{
			name:    "Read tool still checks filePath for proc root",
			tool:    "Read",
			command: "",
			file:    "/proc/self/root/etc/shadow",
			blocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, _, reason := Check(tt.tool, tt.command, tt.file)
			if blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.blocked, reason)
			}
		})
	}
}
