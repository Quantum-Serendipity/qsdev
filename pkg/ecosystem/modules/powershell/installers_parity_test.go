package powershell

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// packageGuardInstallersRE captures the body of the package-guard hook's
// _PS_INSTALLER_CMDLETS tuple.
var packageGuardInstallersRE = regexp.MustCompile(`(?s)\n_PS_INSTALLER_CMDLETS: tuple = \((.*?)\n\)`)

// TestInstallersMatchPackageGuard keeps the two layers that block PowerShell
// installers in step: every cmdlet and alias these deny rules cover is one
// the package-guard hook denies as a command, and the reverse, so a name
// added to one layer (inmo, upmo, Save-Package) cannot be missing from the
// other.
func TestInstallersMatchPackageGuard(t *testing.T) {
	t.Parallel()

	hook := filepath.Join("..", "..", "..", "..", "addons", "claudecode", "templates", "hooks", "package-guard.py")
	src, err := os.ReadFile(hook)
	if err != nil {
		t.Fatalf("reading the package-guard hook: %v", err)
	}
	m := packageGuardInstallersRE.FindSubmatch(src)
	if m == nil {
		t.Fatalf("%s: _PS_INSTALLER_CMDLETS tuple not found", hook)
	}
	var hookNames []string
	for _, q := range regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(m[1], -1) {
		hookNames = append(hookNames, string(q[1]))
	}
	var ruleNames []string
	for _, name := range slices.Concat(psInstallCmdlets, psInstallAliases) {
		ruleNames = append(ruleNames, strings.ToLower(name))
	}
	slices.Sort(hookNames)
	slices.Sort(ruleNames)
	if !slices.Equal(hookNames, ruleNames) {
		t.Errorf("package-guard installers %q differ from the deny rules' %q", hookNames, ruleNames)
	}
}
