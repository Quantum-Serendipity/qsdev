package powershell_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/powershell"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: the installer cmdlets and their aliases are denied
// as whole words anywhere in a Claude Code PowerShell tool call (after `;`,
// `|`, `&&`, a newline, inside a scriptblock, subexpression or string,
// module-qualified); in the Bash tool the cmdlets are denied bare or inside
// any pwsh/powershell invocation, also behind env. Routine commands, and
// longer cmdlets that start with an installer's name (Update-ModuleManifest),
// stay allowed. Operations are Claude Code tool calls ("Bash(...)",
// "PowerShell(...)").
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&powershell.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, op := range []string{
		"Bash(Install-Module Evil)",
		"Bash(Install-PSResource Evil)",
		"Bash(pwsh -c 'Install-Module Evil')",
		"Bash(pwsh -NoProfile -Command Install-Module Evil)",
		"Bash(pwsh -Command \"Install-PSResource Evil\")",
		"Bash(powershell -Command Install-Module Evil)",
		"Bash(pwsh.exe -c Install-Module Evil)",
		"Bash(powershell.exe -NoProfile -Command Install-PSResource Evil)",
		"Bash(install-module Evil)",
		"Bash(pwsh -c 'install-psresource Evil')",
		"Bash(pwsh -c 'Get-Date; Install-Module Evil')",
		"PowerShell(Install-Module Evil)",
		"PowerShell(INSTALL-MODULE Evil)",
		"PowerShell(install-psresource Evil)",
		"PowerShell(Get-Date; Install-Module Evil)",
		"PowerShell(Get-Date;Save-PSResource Evil -Path .)",
		"PowerShell($ErrorActionPreference = 'Stop'; install-module Evil)",
		"Bash(env X=1 pwsh -c Install-Module Evil)",
		"Bash(env X=1 powershell -Command install-psresource Evil)",
		"PowerShell(Find-Module Evil | Install-Module)",
		"PowerShell(Find-Module Evil | Install-Module -Force)",
		"PowerShell(Get-Date && Install-Module Evil)",
		"PowerShell(Get-Date\nInstall-Module Evil)",
		"PowerShell(& { Install-Module Evil })",
		"PowerShell(PowerShellGet\\Install-Module Evil)",
		"PowerShell(Invoke-Expression 'Install-Module Evil')",
		"PowerShell(isres Evil)",
		"PowerShell(ISRES Evil)",
		"PowerShell(udres)",
		"PowerShell(Get-Date; isres Evil)",
		"PowerShell(Get-Date;isres Evil)",
		"PowerShell(Find-PSResource Evil | isres)",
		"PowerShell(Find-PSResource Evil|udres -Force)",
		"PowerShell(Install-Module)",
		"PowerShell(Get-Date;Install-Module;Get-Date)",
		"PowerShell(Invoke-Expression \"Install-Module\")",
		"PowerShell(& {Install-Module})",
		"PowerShell((Install-Module Evil))",
		"PowerShell(Install-Module\nGet-Date)",
		"PowerShell(Get-Date\nisres Evil)",
		"PowerShell({isres Evil})",
		"PowerShell(& {isres})",
		"PowerShell((isres Evil))",
		"PowerShell(Get-Date&&isres Evil)",
		"PowerShell(isres;Get-Date)",
	} {
		if _, ok := denyutil.FirstMatch(rules, op); !ok {
			t.Errorf("%q is not denied", op)
		}
	}
	for _, op := range []string{
		"Bash(pwsh -Command \"Invoke-ScriptAnalyzer -Path .\")",
		"Bash(Get-Module -ListAvailable)",
		"PowerShell(Get-Module -ListAvailable)",
		"PowerShell(Get-Date; Invoke-Pester)",
		"PowerShell(Get-Content thisresult.txt)",
		"PowerShell(Get-Content isresult.txt)",
		"PowerShell(Find-PSResource Evil)",
		"PowerShell(Update-ModuleManifest -Path ./My.psd1 -ModuleVersion 1.2.0)",
		"PowerShell(Update-ScriptFileInfo -Path ./x.ps1 -Version 1.1)",
		"PowerShell(New-ModuleManifest -Path ./My.psd1)",
		"PowerShell(Get-Date; Update-ModuleManifest -Path ./My.psd1)",
		"PowerShell(Get-Content isres.txt)",
	} {
		if rule, ok := denyutil.FirstMatch(rules, op); ok {
			t.Errorf("%s is unexpectedly denied by %q", op, rule)
		}
	}
}

// TestDenyRules_NoCaseFoldedPowerShellDuplicates pins that every
// PowerShell(...) rule is emitted once: the PowerShell tool matches rules
// case-insensitively, so a lowercase copy of a rule adds nothing. Bash rules
// are case-sensitive and keep their lowercase spelling.
func TestDenyRules_NoCaseFoldedPowerShellDuplicates(t *testing.T) {
	t.Parallel()

	seen := map[string]string{}
	for _, rule := range (&powershell.Module{}).DenyRules(ecosystem.ModuleConfig{}) {
		if tool, _ := denyutil.ParseToolPattern(rule); tool != "PowerShell" {
			continue
		}
		key := strings.ToLower(rule)
		if prev, ok := seen[key]; ok {
			t.Errorf("PowerShell rule %q duplicates %q up to case", rule, prev)
		}
		seen[key] = rule
	}
}
