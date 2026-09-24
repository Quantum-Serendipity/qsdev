package powershell_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/powershell"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: both installer cmdlets are denied bare or inside
// any pwsh/powershell invocation; routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&powershell.Module{}).DenyRules(ecosystem.ModuleConfig{})
	denied := func(cmd string) bool {
		for _, r := range rules {
			if denyutil.MatchesDenyRule(r, "Bash("+cmd+")") {
				return true
			}
		}
		return false
	}

	for _, cmd := range []string{
		"Install-Module Evil",
		"Install-PSResource Evil",
		"pwsh -c 'Install-Module Evil'",
		"pwsh -NoProfile -Command Install-Module Evil",
		"pwsh -Command \"Install-PSResource Evil\"",
		"powershell -Command Install-Module Evil",
		"pwsh.exe -c Install-Module Evil",
		"powershell.exe -NoProfile -Command Install-PSResource Evil",
		"install-module Evil",
		"pwsh -c 'install-psresource Evil'",
	} {
		if !denied(cmd) {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"pwsh -Command \"Invoke-ScriptAnalyzer -Path .\"",
		"Get-Module -ListAvailable",
	} {
		if denied(cmd) {
			t.Errorf("%q is unexpectedly denied by %v", cmd, rules)
		}
	}
}
