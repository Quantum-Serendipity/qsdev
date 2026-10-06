package rules

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestPowerShellNative verifies the PowerShell tool is judged in its own
// dialect (U18-08): backslash paths, cmdlet names and case-insensitive
// spellings that the POSIX analysis misreads. A line that names a protected
// path and runs anything but a read cmdlet denies (fail closed); a read of a
// protected path, or a mutation of an unprotected one, is allowed.
func TestPowerShellNative(t *testing.T) {
	t.Parallel()
	project := filepath.Join(homeDir(t), "project")
	app := branding.Get().AppName
	cases := []struct {
		name    string
		command string
		cwd     string
		want    Verdict
		rule    string // a rule that must be among the matches, when set
	}{
		{name: "Remove-Item", command: `Remove-Item .claude\settings.json`, want: Deny, rule: "SP-003"},
		{name: "Set-Content", command: `Set-Content -Path .claude\settings.json -Value '{}'`, want: Deny, rule: "SP-007"},
		{name: "Out-File", command: `'{}' | Out-File .claude\settings.json`, want: Deny, rule: "SP-007"},
		{name: "ri hooks", command: `ri .claude\hooks -Recurse`, want: Deny, rule: "SP-010"},
		{name: "Copy-Item onto settings", command: `Copy-Item C:\evil.json .claude\settings.json`, want: Deny, rule: "SP-007"},
		{name: "redirect onto settings", command: `Get-Content x > .claude\settings.json`, want: Deny, rule: "SP-007"},
		{name: "pipe to Set-Content", command: `Get-Content .claude\settings.json | Set-Content x.json`, want: Deny},
		{name: "concatenated name", command: `$p='.cl'+'aude'; ri "$p\settings.json"`, want: Deny, rule: "SP-003"},
		{name: "Set-Location then relative delete", command: `Set-Location .claude; ri settings.json`, want: Deny, rule: "SP-003"},
		{name: "cwd inside .claude", command: `ri settings.json`, cwd: filepath.Join(project, ".claude"), want: Deny},
		{name: "upper case", command: `REMOVE-ITEM .CLAUDE\SETTINGS.JSON`, want: Deny, rule: "SP-003"},
		{name: "audit trail", command: `Clear-Content .qsdev\audit\log.jsonl`, want: Deny, rule: "SP-013"},
		{name: "security binary", command: `Set-Content .qsdev\bin\qsdev.exe x`, want: Deny, rule: "INT-001"},
		{name: "subexpression in a read", command: `Get-Content "$(Remove-Item .claude\settings.json)"`, want: Deny},
		{name: "MCP config", command: `Set-Content .cursor\mcp.json '{}'`, want: Deny, rule: "MCP-005"},
		{name: "Stop-Process by name", command: `Stop-Process -Name ` + app, want: Deny, rule: "SP-009"},
		{name: "Get-Process piped to Stop-Process", command: `Get-Process ` + app + ` | Stop-Process`, want: Deny, rule: "SP-009"},
		{name: "taskkill image", command: `taskkill /IM ` + app + `.exe /F`, want: Deny, rule: "SP-009"},
		{name: "Stop-Process glob", command: `Stop-Process -Name q*`, want: Deny, rule: "SP-009"},
		{name: "spps claude", command: `spps -Name:claude -Force`, want: Deny, rule: "SP-009"},
		{name: "STOP-PROCESS gdev", command: `STOP-PROCESS -Name GDEV,node`, want: Deny, rule: "SP-009"},
		{name: "kill by variable", command: `$n = 'qs' + 'dev'; kill -Name $n`, want: Deny, rule: "SP-009"},
		{name: "Kill method", command: `(Get-Process ` + app + `).Kill()`, want: Deny, rule: "SP-009"},
		{name: "Stop-Process concatenated name", command: `Stop-Process -Name ('` + app[:2] + `'+'` + app[2:] + `')`, want: Deny, rule: "SP-009"},
		{name: "Stop-Process name after an id", command: `Stop-Process -Id 1 -Name ` + app, want: Deny, rule: "SP-009"},
		{name: "glob segment", command: `Remove-Item .cla?de/settings.json`, want: Deny, rule: "SP-003"},
		{name: "glob segment backslash", command: `Remove-Item .cla?de\settings.json`, want: Deny, rule: "SP-003"},
		{name: "star glob backslash", command: `Remove-Item .c*\settings.json`, want: Deny, rule: "SP-003"},
		{name: "bracket glob", command: `Remove-Item .cl[a]ude/settings.json`, want: Deny, rule: "SP-003"},
		{name: "upper-case glob", command: `Remove-Item .CLA?DE\SETTINGS.JSON`, want: Deny, rule: "SP-003"},
		{name: "Set-Content glob", command: `Set-Content .cla?de/settings.json '{}'`, want: Deny, rule: "SP-007"},
		{name: "home glob", command: `Remove-Item ~/.cla*`, want: Deny, rule: "SP-003"},
		{name: "empty subexpression split", command: `ri .cl$()aude/settings.json`, want: Deny, rule: "SP-003"},
		{name: "quoted empty subexpression split", command: `ri ".cl$()aude/settings.json"`, want: Deny, rule: "SP-003"},
		{name: "Push-Location then relative write", command: `Push-Location .claude; Set-Content settings.json '{}'`, want: Deny, rule: "SP-007"},
		{name: "Remove-Item home", command: `Remove-Item ~ -Recurse -Force`, want: Deny, rule: "SP-003"},
		{name: "Remove-Item $HOME", command: `Remove-Item $HOME -Recurse -Force`, want: Deny, rule: "SP-003"},
		{name: "Remove-Item $env:USERPROFILE", command: `Remove-Item $env:USERPROFILE -Recurse -Force`, want: Deny, rule: "SP-003"},
		{name: "Remove-Item parent", command: `Remove-Item .. -Recurse -Force`, want: Deny, rule: "SP-003"},
		{name: "Move-Item home", command: `Move-Item ~ C:\old-home`, want: Deny},
		{name: "git diff output onto settings", command: `git diff --output=.claude\settings.json`, want: Deny, rule: "SP-007"},

		{name: "Get-Content", command: `Get-Content .claude\settings.json`, want: Allow},
		{name: "Get-ChildItem", command: `Get-ChildItem .claude`, want: Allow},
		{name: "git status", command: `git status`, want: Allow},
		{name: "Remove-Item unprotected", command: `Remove-Item node_modules -Recurse`, want: Allow},
		{name: "Get-Content stderr to null", command: `Get-Content .claude\settings.json 2>$null`, want: Allow},
		{name: "read in cwd inside .claude", command: `Get-Content settings.json`, cwd: filepath.Join(project, ".claude"), want: Allow},
		{name: "MCP config read", command: `Get-Content .cursor\mcp.json`, want: Allow},
		{name: "Stop-Process by id", command: `Stop-Process -Id 1234`, want: Allow},
		{name: "taskkill other image", command: `taskkill /IM node.exe /F`, want: Allow},
		{name: "Get-Process of the CLI", command: `Get-Process ` + app, want: Allow},
		{name: "Stop-Process by id variable", command: `Stop-Process -Id $proc.Id`, want: Allow},
		{name: "Stop-Process by pid", command: `Stop-Process -Id:$pid -Force`, want: Allow},
		{name: "git diff backslash", command: `git diff .claude\settings.json`, want: Allow},
		{name: "git diff slash", command: `git diff .claude/settings.json`, want: Allow},
		{name: "git log", command: `git log -- .claude`, want: Allow},
		{name: "Remove-Item project subdirectory", command: `Remove-Item build -Recurse -Force`, want: Allow},
		{name: "kill named in a path", command: `Stop-Process -Id 1234; Get-Content docs\` + app + `-notes.md`, want: Allow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cwd := tc.cwd
			if cwd == "" {
				cwd = project
			}
			ctx := EvalContext{ToolName: "PowerShell", Command: tc.command, CWD: cwd}
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != tc.want {
				t.Fatalf("verdict = %v, want %v (matches %v)", got, tc.want, matchIDs(matches))
			}
			if tc.rule != "" && !slices.Contains(matchIDs(matches), tc.rule) {
				t.Errorf("matches = %v, want %s among them", matchIDs(matches), tc.rule)
			}
		})
	}
}

func matchIDs(matches []RuleMatch) []string {
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.Rule.ID
	}
	return ids
}

// TestPowerShellResolvedPaths verifies a PowerShell word is resolved like a
// POSIX one: a symlink into a protected directory, and a protected
// directory reached through Set-Location, cd or the session directory, are
// protected even though the line never spells the protected name.
func TestPowerShellResolvedPaths(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(project, "link")
	if err := os.Symlink(filepath.Join(project, ".claude"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cases := []struct {
		name, command, cwd string
		want               Verdict
	}{
		{name: "Remove-Item through link", command: `Remove-Item link/settings.json`, want: Deny},
		{name: "Remove-Item through link backslash", command: `Remove-Item link\settings.json`, want: Deny},
		{name: "Set-Content through link", command: `Set-Content link/settings.json '{}'`, want: Deny},
		{name: "Copy-Item onto link", command: `Copy-Item x link/settings.json`, want: Deny},
		{name: "cd link", command: `cd link; Remove-Item settings.json`, want: Deny},
		{name: "Set-Location link", command: `Set-Location link; Remove-Item settings.json`, want: Deny},
		{name: "sl link backslash", command: `sl .\link; ri settings.json`, want: Deny},
		{name: "cwd is the link", command: `Remove-Item settings.json`, cwd: link, want: Deny},
		{name: "read through link", command: `Get-Content link\settings.json`, want: Allow},
		{name: "unrelated delete", command: `Remove-Item other/settings.json`, want: Allow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cwd := tc.cwd
			if cwd == "" {
				cwd = project
			}
			ctx := EvalContext{ToolName: "PowerShell", Command: tc.command, CWD: cwd}
			if got, matches := Tier1Rules.EvaluateAll(&ctx); got != tc.want {
				t.Fatalf("verdict = %v, want %v (matches %v)", got, tc.want, matchIDs(matches))
			}
		})
	}
}
