package devinit

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestLifecycle_LocalAnswersNeverReachCommittedConfig is the U28-WS1
// regression for the enable/disable half of the settle step: a hand-edited
// `claude_code: false` or always-on off in the local, gitignored answers file
// is settled against the committed .qsdev.yaml before the command syncs it,
// so the committed claude_code block stays as the team committed it, and the
// dropped always-on off is warned about as on every other path.
func TestLifecycle_LocalAnswersNeverReachCommittedConfig(t *testing.T) {
	const tool = "commitlint"
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		run   func(t *testing.T, dir string, args ...string) (string, error)
	}{
		{name: "enable", run: enableTool},
		{
			name:  "disable",
			setup: func(t *testing.T, dir string) { t.Helper(); mustEnable(t, dir, tool) },
			run:   disableTool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			before := committedConfig(t, dir)
			if !qsdevconfig.ClaudeCodeEnabled(before) || len(before.ClaudeCode.MCPServers) == 0 {
				t.Fatalf("init committed claude_code = %+v, want enabled with MCP servers", before.ClaudeCode)
			}

			local := loadProjectAnswers(t, dir)
			local.ClaudeCode = false
			local.EnabledTools["attach-guard"] = false
			if err := saveAnswers(dir, local); err != nil {
				t.Fatalf("saving answers: %v", err)
			}

			out, err := tt.run(t, dir, tool)
			if err != nil {
				t.Fatalf("%s %s: %v\n%s", tt.name, tool, err, out)
			}

			after := committedConfig(t, dir)
			if !qsdevconfig.ClaudeCodeEnabled(after) {
				t.Errorf("committed claude_code.enabled turned off by a local answers edit:\n%+v", after.ClaudeCode)
			}
			if !equalClaudeCode(before.ClaudeCode, after.ClaudeCode) {
				t.Errorf("committed claude_code changed:\nbefore %+v\nafter  %+v", before.ClaudeCode, after.ClaudeCode)
			}
			if !strings.Contains(out, `always-on tool "attach-guard" kept enabled`) {
				t.Errorf("no warning for the dropped attach-guard off; output:\n%s", out)
			}
			if got := loadProjectAnswers(t, dir); !got.ClaudeCode || !got.EnabledTools["attach-guard"] {
				t.Errorf("saved answers claude_code=%v attach-guard=%v, want both settled on", got.ClaudeCode, got.EnabledTools["attach-guard"])
			}
		})
	}
}

func committedConfig(t *testing.T, dir string) *types.QsdevConfig {
	t.Helper()
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(dir, branding.Get().ConfigFile))
	if err != nil {
		t.Fatalf("parsing committed config: %v", err)
	}
	return cfg
}

func equalClaudeCode(a, b types.ClaudeCodeConfig) bool {
	return slices.Equal(a.Skills, b.Skills) && slices.Equal(a.MCPServers, b.MCPServers) &&
		a.PermissionLevel == b.PermissionLevel
}
