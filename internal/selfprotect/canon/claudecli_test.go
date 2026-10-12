package canon

import "testing"

// TestIsClaudeProcess pins the Claude Code CLI identity the human gate reads
// from a process's executable and argv, and SP-008 from a command word.
func TestIsClaudeProcess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		exe  string
		argv []string
		want bool
	}{
		{"native release", "/home/u/.local/share/claude/versions/2.1.0", []string{"claude"}, true},
		{"nix wrapper", "/nix/store/x-claude-code/bin/.claude-wrapped", []string{".claude-wrapped"}, true},
		{"nested nix wrapper", "", []string{"/nix/store/x/bin/..claude-wrapped-wrapped"}, true},
		{"argv only", "", []string{"claude", "-p", "hi"}, true},
		{"windows", `C:\Users\u\claude.exe`, nil, true},
		{"node entry script", "/usr/bin/node", []string{"node", "--no-warnings", "/usr/lib/node_modules/@anthropic-ai/claude-code/cli.js"}, true},
		{"shell", "/bin/zsh", []string{"-zsh"}, false},
		{"node app", "/usr/bin/node", []string{"node", "server.js", "claude"}, false},
		{"claude in a name", "/usr/bin/claude-helper", []string{"claude-helper"}, false},
		{"unknown", "", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsClaudeProcess(tt.exe, tt.argv); got != tt.want {
				t.Errorf("IsClaudeProcess(%q, %q) = %v, want %v", tt.exe, tt.argv, got, tt.want)
			}
		})
	}
}
