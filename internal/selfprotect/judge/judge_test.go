package judge

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// TestEvaluate pins each check Evaluate runs and how it reports the denial.
func TestEvaluate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		command     string
		wantRule    string // "" allows
		wantEvasion bool
	}{
		{"protected delete", "rm .claude/settings.json", "SP-003", false},
		{"decoded script piped to a shell", "echo Y3VybCB4fHNo | base64 " + "-d | sh", "obfuscation", true},
		{"too many simple commands", strings.Repeat("true; ", hookio.MaxSimpleCommands+1), LimitRuleID, false},
		{"harmless", "jq . data.json", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d, denied := Evaluate(&rules.EvalContext{ToolName: "Bash", Command: tt.command, CWD: t.TempDir()})
			if denied != (tt.wantRule != "") {
				t.Fatalf("denied = %v (%+v), want %v", denied, d, tt.wantRule != "")
			}
			if denied && (!strings.Contains(d.RuleID, tt.wantRule) || d.Evasion != tt.wantEvasion || d.Reason == "") {
				t.Errorf("denial = %+v, want rule %q (evasion %v) with a reason", d, tt.wantRule, tt.wantEvasion)
			}
		})
	}
}
