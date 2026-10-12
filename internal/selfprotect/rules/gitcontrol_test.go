package rules

import (
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// TestGitControl_HookVerdicts pins the verdicts on the git control files
// (U18-09, U18-V01, XS-N4): the hooks that run the commit gate, the repository
// and global git config that can point core.hooksPath elsewhere, and the
// attributes that bind filter drivers. The Write/Edit tools and every shell
// tool are denied; reading them, ordinary git use and writes to refs and logs
// (a branch named config or hooks/x) stay allowed.
func TestGitControl_HookVerdicts(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")

	writes := []struct {
		path string
		want Verdict
	}{
		{filepath.Join(project, ".git", "hooks", "pre-commit"), Deny},
		{filepath.Join(project, ".git", "config"), Deny},
		{filepath.Join(project, ".git", "config.worktree"), Deny},
		{filepath.Join(project, ".git", "info", "attributes"), Deny},
		{filepath.Join(project, ".git", "modules", "sub", "hooks", "pre-commit"), Deny},
		{filepath.Join(project, ".git", "modules", "a", "b", "config"), Deny},
		{filepath.Join(project, ".git", "worktrees", "w", "config.worktree"), Deny},
		{filepath.Join(home, ".gitconfig"), Deny},
		{filepath.Join(home, ".config", "git", "config"), Deny},
		{filepath.Join(home, ".config", "git", "attributes"), Deny},
		{filepath.Join(project, ".git", "refs", "heads", "config"), Allow},
		{filepath.Join(project, ".git", "logs", "refs", "heads", "hooks", "x"), Allow},
		{filepath.Join(project, ".git", "info", "exclude"), Allow},
		{filepath.Join(project, "hooks", "pre-commit"), Allow},
		{filepath.Join(project, "src", "main.go"), Allow},
		{filepath.Join(project, ".gitignore"), Allow},
		{filepath.Join(project, ".gitattributes"), Allow},
	}
	for _, tool := range []string{"Write", "Edit"} {
		for _, w := range writes {
			t.Run(tool+" "+w.path, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, FilePath: w.path, CanonicalPath: w.path, CWD: project}
				got, matches := Tier1Rules.EvaluateAll(&ctx)
				if got != w.want {
					t.Fatalf("verdict = %v (first: %s), want %v", got, firstRuleID(matches), w.want)
				}
				if got == Deny && firstRuleID(matches) != "SP-001" {
					t.Errorf("first match = %s, want SP-001", firstRuleID(matches))
				}
			})
		}
	}

	// Each denied command has a PowerShell spelling ("" reuses the POSIX
	// one, which PowerShell runs through its aliases or as a program).
	denied := []struct{ posix, ps string }{
		{"rm .git/hooks/pre-commit", `Remove-Item .git\hooks\pre-commit`},
		{"echo 'exit 0' > .git/hooks/pre-commit", `Set-Content .git\hooks\pre-commit 'exit 0'`},
		{"chmod -x .git/hooks/pre-commit", ""},
		{"mv /tmp/x .git/hooks/pre-commit", `Move-Item C:\tmp\x .git\hooks\pre-commit`},
		{"ln -sf /bin/true .git/hooks/pre-commit", `New-Item -ItemType SymbolicLink -Path .git\hooks\pre-commit -Target C:\x.exe`},
		{"cd .git && rm hooks/pre-commit", `Set-Location .git; Remove-Item hooks\pre-commit`},
		{"cd .git/hooks && rm pre-commit", `Set-Location .git\hooks; Remove-Item pre-commit`},
		{"rm -rf .git/hooks", `Remove-Item -Recurse -Force .git\hooks`},
		{"find . -name pre-commit -delete", ""},
		{`printf '[core]\n\thooksPath=/tmp/h\n' >> .git/config`, `Add-Content .git\config '[core]'`},
		{"echo '* filter=x' > .git/info/attributes", `Set-Content .git\info\attributes '* filter=x'`},
		{"cp /tmp/c .git/modules/sub/config", `Copy-Item C:\tmp\c .git\modules\sub\config`},
		{`printf '[core]\n\thooksPath=/tmp/h\n' >> ~/.gitconfig`, `Add-Content ~\.gitconfig '[core]'`},
		{"echo x >> ~/.config/git/config", `Add-Content ~/.config/git/config x`},
		// Spellings that never write the control path literally: a glob
		// segment, the git directory moved, replaced or changed as a whole,
		// a find started inside it, and a variable holding it.
		{"rm .gi?/hooks/pre-commit", ""},
		{"rm .git/*/pre-commit", ""},
		{"rm .git/hoo*/pre-commit", ""},
		{"rm .[g]it/hooks/pre-commit", ""},
		{"rm .g{i,x}t/hooks/pre-commit", ""},
		{"mv .git .g", `Move-Item .git .g`},
		{"mv .git .g && rm .g/hooks/pre-commit && mv .g .git", `Move-Item .git .g; Remove-Item .g\hooks\pre-commit`},
		{"mv .gi? .g", ""},
		{"mv /tmp/g .git", `Move-Item C:\tmp\g .git`},
		{"cp -a .git .g && rm -rf .git && rm .g/hooks/pre-commit && mv .g .git", ""},
		{"rm -rf .git/modules/sub", `Remove-Item -Recurse .git\modules\sub`},
		{"chmod -R a-x .git", ""},
		{"chown -R nobody .git", ""},
		{"cd .git && chmod -R a-x .", ""},
		{`find .git -name "pre-*" -delete`, ""},
		{"find .git -perm -u+x -type f -delete", ""},
		{"find .git -path '.git/hooks/*' -delete", ""},
		{"cd .git && find . -path './hooks/*' -delete", ""},
		{"d=.git; rm $d/hooks/pre-commit", `$d = '.git'; Remove-Item "$d\hooks\pre-commit"`},
		{`rm "$(printf .git)/hooks/pre-commit"`, ""},
	}
	for _, tool := range cmdscan.ShellTools {
		for _, d := range denied {
			command := d.posix
			if tool == "PowerShell" && d.ps != "" {
				command = d.ps
			}
			t.Run(tool+" deny "+command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, Command: command, CWD: project}
				if got, matches := Tier1Rules.EvaluateAll(&ctx); got != Deny {
					t.Errorf("EvaluateAll(%q) = %v (first: %s), want Deny", command, got, firstRuleID(matches))
				}
			})
		}
	}

	posixAllowed := []string{
		"cat .git/config",
		"ls .git/hooks",
		"ls .git",
		"cat .git/HEAD",
		"du -sh .git",
		"rm -f .git/index.lock",
		"rm -f .git/*.lock",
		"rm -rf .git/rebase-merge",
		"find .git -name '*.lock'",
		"find .git -name '*.lock' -delete",
		"ls -la .gi?",
		"cp -a .git /tmp/backup",
		"rm -rf .github/workflows/old.yml",
		"mv .gitignore .gitignore.bak",
		"echo build/ >> .git/info/exclude",
		"cat ~/.gitconfig",
		"git --git-dir=.git status",
		"git status",
		"git log",
		"git diff",
		"git add -A",
		"git fetch origin",
		"git commit -m x",
		`git commit -m "handle sh -c and --no-verify"`,
		"git checkout -b feature",
		"git switch main",
		"pre-commit run --all-files",
		"SKIP=golangci-lint pre-commit run --all-files",
		"pre-commit install",
		"prek install",
	}
	psAllowed := []string{
		`Get-Content .git\config`,
		`Get-ChildItem .git\hooks`,
		`Get-ChildItem .git`,
		`Get-Content .git\HEAD`,
		`Remove-Item -Force .git\index.lock`,
		`Add-Content .git\info\exclude 'build/'`,
		`Get-Content ~\.gitconfig`,
		"git status",
		"git log",
		"git commit -m x",
		"pre-commit run --all-files",
	}
	for _, tool := range cmdscan.ShellTools {
		allowed := posixAllowed
		if tool == "PowerShell" {
			allowed = psAllowed
		}
		for _, command := range allowed {
			t.Run(tool+" allow "+command, func(t *testing.T) {
				t.Parallel()
				// `git switch main` asks git what it rewrites: a repository
				// whose protected files main leaves unchanged.
				ctx := EvalContext{ToolName: tool, Command: command, CWD: project, Ctx: t.Context(), gitRunner: rewriteRepo(project, false).run}
				if got, matches := Tier1Rules.EvaluateAll(&ctx); got != Allow {
					t.Errorf("EvaluateAll(%q) = %v (first: %s), want Allow", command, got, firstRuleID(matches))
				}
			})
		}
	}
}

// TestGitOutputIntoGitConfigStillMatchesGIT001 pins that `git log --output`
// into .git/config, which the Tier-1 mutation rule now denies first (SP-007)
// because .git/config is a git-control file, still denies under GIT-001: the
// git option check keeps its coverage of the output-file spelling. GIT-001
// is not a Tier-1 rule (EvaluateAll never lists it); judge.Evaluate runs it
// after the Tier-1 rules, so it is asserted by calling GitCodeExecution.
func TestGitOutputIntoGitConfigStillMatchesGIT001(t *testing.T) {
	t.Parallel()
	project := filepath.Join(homeDir(t), "project")
	for _, command := range []string{
		"git log -1 --format=%B --output .git/config",
		"git log -1 --format=%B --output=.git/config",
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: command, CWD: project}
			got, matches := Tier1Rules.EvaluateAll(&ctx)
			if got != Deny {
				t.Fatalf("verdict = %v, want Deny", got)
			}
			if first := firstRuleID(matches); first != "SP-007" {
				t.Errorf("first match = %s, want SP-007", first)
			}
			fresh := EvalContext{ToolName: "Bash", Command: command, CWD: project}
			if reason, deny := GitCodeExecution(&fresh); !deny {
				t.Errorf("GitCodeExecution(%q) = (%q, false), want %s to still deny it", command, reason, GitCodeExecutionRuleID)
			}
		})
	}
}
