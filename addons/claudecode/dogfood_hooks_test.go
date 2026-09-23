package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDogfoodHooksMatchTemplates guards against the repository's own
// committed .claude/hooks drifting from the embedded templates they are
// generated from. A stale dogfooded package-guard.py once kept regex-based
// detection long after the template moved to argv parsing, so maintainers'
// sessions were denied on ordinary grep/commit commands while users got the
// fixed hook. Regenerate with `qsdev init --update` (or copy the template)
// when this fails.
func TestDogfoodHooksMatchTemplates(t *testing.T) {
	t.Parallel()

	// Union of every hook the generator can emit. audit-log.sh and
	// soc2-audit-log.py are mutually exclusive, so two answer sets are needed.
	all := types.HookChoices{
		SafetyBlock: true, CredentialScan: true, DestructivePrevention: true,
		FileBoundary: true, ToolGates: true,
	}
	withSOC2, withAudit := all, all
	withSOC2.SOC2Audit = true
	withAudit.AuditLog = true

	want := map[string][]byte{}
	for _, hooks := range []types.HookChoices{withSOC2, withAudit} {
		files, err := claudecode.GenerateHookFiles(types.WizardAnswers{
			Hooks:      hooks,
			AgentTools: types.AgentToolsAnswers{SembleEnabled: true},
		})
		if err != nil {
			t.Fatalf("GenerateHookFiles: %v", err)
		}
		for _, f := range files {
			want[filepath.Base(f.Path)] = f.Content
		}
	}

	hookDir := filepath.Join("..", "..", ".claude", "hooks")
	entries, err := os.ReadDir(hookDir)
	if err != nil {
		t.Fatalf("reading dogfooded hooks: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("%s has no hooks; expected at least package-guard.py", hookDir)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			t.Parallel()
			tmpl, ok := want[e.Name()]
			if !ok {
				t.Fatalf("%s is not produced by any hook template", e.Name())
			}
			got, err := os.ReadFile(filepath.Join(hookDir, e.Name()))
			if err != nil {
				t.Fatalf("reading %s: %v", e.Name(), err)
			}
			if !bytes.Equal(got, tmpl) {
				t.Errorf(".claude/hooks/%s differs from its embedded template; regenerate it with `qsdev init --update`", e.Name())
			}
		})
	}
}
