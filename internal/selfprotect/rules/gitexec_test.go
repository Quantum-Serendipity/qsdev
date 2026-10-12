package rules

import "testing"

// TestGitCodeExecution covers the git spellings that run a program no rule or
// hook inspects, or skip the hooks, which prefix-glob permission rules cannot
// express, and the ordinary git commands (commit messages that mention the
// same options included) that must stay allowed.
func TestGitCodeExecution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		deny    bool
	}{
		// Per-invocation configuration, wherever the global option sits.
		{`git -c alias.x='!npm i evil-pkg' x`, true},
		{`git -C . -c alias.x='!sh' x`, true},
		{`git --no-pager -c core.pager='sh -c id' log`, true},
		{`git --git-dir .git -c core.fsmonitor=evil status`, true},
		{`git --config-env=alias.x=CMD x`, true},
		{`git --config-env alias.x=CMD x`, true},
		{`git --exec-path=/tmp/evil status`, true},
		{`/usr/bin/git -c alias.x='!sh' x`, true},
		{`sudo git -c alias.x='!sh' x`, true},
		{`command git -c alias.x='!sh' x`, true},
		// Configuration and programs from the environment.
		{`GIT_CONFIG_PARAMETERS="'core.pager=sh'" git log`, true},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.fsmonitor GIT_CONFIG_VALUE_0=evil git status`, true},
		{`env GIT_CONFIG_COUNT=1 git status`, true},
		{`export GIT_CONFIG_GLOBAL=/tmp/evil && git status`, true},
		{`GIT_EXTERNAL_DIFF=/tmp/evil git diff`, true},
		{`GIT_EXEC_PATH=/tmp/evil git status`, true},
		// Persistent configuration.
		{`git config core.hooksPath /tmp/x`, true},
		{`git -C . config alias.x '!sh'`, true},
		// Hook skips: long form, abbreviations, clusters.
		{`git commit -m wip --no-verify`, true},
		{`git commit -m wip --no-veri`, true},
		{`git commit -m wip --no-verif`, true},
		{`git commit -m wip -n`, true},
		{`git commit -m wip -qn`, true},
		{`git commit -nm wip`, true},
		{`git commit -anm wip`, true},
		{`git am -n fix.patch`, true},
		{`git merge --no-verify feature`, true},
		{`git push --no-verify origin main`, true},
		// Output to an arbitrary file.
		{`git log -1 --format=%B --output=.git/config`, true},
		{`git log -1 --format=%B --output .git/config`, true},
		{`git diff HEAD~1 --output=patch.txt`, true},
		// Git inside a shell script or eval.
		{`sh -c "git -c alias.x='!sh' x"`, true},
		{`bash -ec 'git commit -n -m wip'`, true},
		{`eval "git commit --no-verify -m wip"`, true},
		{`git commit -m 'unterminated`, true},

		// Ordinary git use stays allowed.
		{`git status`, false},
		{`git diff --stat`, false},
		{`git log --oneline -n 5`, false},
		{`git show HEAD`, false},
		{`git add -A`, false},
		{`git commit -m "Handle sh -c wrappers, the -n flag and --no-verify"`, false},
		{`git commit -am "fix -c parsing"`, false},
		{`git commit -m -n`, false},
		{`git commit -mfix-nav`, false},
		{`git commit -C HEAD --amend`, false},
		{`git commit -c HEAD`, false},
		{"git commit -m \"$(cat <<'EOF'\nfix: skip --no-verify\nEOF\n)\"", false},
		{`git commit -m wip -- -n`, false},
		{`git push -u origin feature`, false},
		{`git pull --rebase`, false},
		{`git merge --no-ff feature`, false},
		{`git -C sub status`, false},
		{`git --no-pager log -1`, false},
		{`GIT_PAGER=cat git log -1`, false},
		{`GIT_EDITOR=true git rebase --continue`, false},
		{`echo git -c alias.x=y x`, false},
		{`gh pr create --body "never use git commit --no-verify"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := &EvalContext{ToolName: "Bash", Command: tt.command}
			reason, deny := GitCodeExecution(ctx)
			if deny != tt.deny {
				t.Errorf("GitCodeExecution(%q) = (%q, %v), want deny=%v", tt.command, reason, deny, tt.deny)
			}
		})
	}
}

// TestGIT001_SkipEnvAndUninstall covers the commit gate's own integrity
// (U18-09): a commit-gate variable (canon.CommitGateEnvVars) set for a git
// command that runs hooks, a hook manager's uninstall or an install that
// points it at another config or lets it run without one, and a template
// directory that plants hooks in a new repository. A manual hook run with
// SKIP, a plain install and ordinary commits stay allowed.
func TestGIT001_SkipEnvAndUninstall(t *testing.T) {
	t.Parallel()
	shells := []struct {
		command string
		deny    bool
	}{
		// Hook managers.
		{`pre-commit uninstall`, true},
		{`prek uninstall`, true},
		{`pre-commit uninstall -t pre-push`, true},
		{`/usr/bin/pre-commit uninstall`, true},
		{`pre-commit install -c /tmp/empty.yaml`, true},
		{`pre-commit install --config /tmp/empty.yaml`, true},
		{`pre-commit install --conf=/tmp/empty.yaml`, true},
		{`pre-commit install -f -c/tmp/empty.yaml`, true},
		{`pre-commit install --allow-missing-config`, true},
		{`pre-commit install --allow-missing`, true},
		{`prek install --config=/tmp/e.yaml`, true},
		{`prek -c /tmp/e.yaml install`, true},
		{`sh -c 'pre-commit uninstall'`, true},
		// Commit-gate variables for a git command that runs hooks.
		{`SKIP=ripsecrets,gitleaks git commit -m x`, true},
		{`env SKIP=gitleaks git commit -m x`, true},
		{`export SKIP=gitleaks; git commit -m x`, true},
		{`sh -c 'SKIP=gitleaks git push'`, true},
		{`SKIP=gitleaks sh -c 'git push'`, true},
		{`PREK_SKIP=gitleaks git commit -m x`, true},
		{`PRE_COMMIT_ALLOW_NO_CONFIG=1 git commit -m x`, true},
		{`PREK_ALLOW_NO_CONFIG=1 git commit -m x`, true},
		{`SKIP=gitleaks git -C . merge feature`, true},
		{`SKIP=gitleaks git rebase main`, true},
		// Template directories plant hooks.
		{`git init --template=/tmp/t`, true},
		{`git init --template /tmp/t`, true},
		{`git clone --templ=/tmp/t url`, true},
		{`git clone --te=/tmp/t url`, true},
		{`GIT_TEMPLATE_DIR=/tmp/t git init`, true},
		// clone's configuration options set core.hooksPath in the clone.
		{`git clone -c core.hooksPath=/tmp url`, true},
		{`git clone --config core.hooksPath=/tmp url`, true},
		{`git clone --config=core.hooksPath=/tmp url`, true},
		{`git clone --conf=core.hooksPath=/tmp url`, true},
		{`git clone -qc core.hooksPath=/tmp url`, true},
		{`git clone -ccore.hooksPath=/tmp url`, true},
		{`git clone -b main url dir`, false},
		{`git clone -bc url dir`, false}, // -b takes "c" as the branch name
		{`git clone --depth 1 url`, false},

		{`SKIP=golangci-lint pre-commit run --all-files`, false},
		{`SKIP=x pre-commit run --all-files`, false},
		{`pre-commit run --all-files`, false},
		{`pre-commit install`, false},
		{`pre-commit install --install-hooks -t pre-push`, false},
		{`prek install`, false},
		{`prek run --all-files`, false},
		{`git commit -m x`, false},
		{`git commit -m "handle sh -c and --no-verify"`, false},
		{`git commit -m "SKIP=gitleaks pre-commit uninstall"`, false},
		{`SKIP=gitleaks git status`, false},
		{`SKIP=gitleaks git log -1`, false},
		{`git init`, false},
		{`git clone url dir`, false},
		{`git clone --tags url`, false},
		{`echo SKIP=x`, false},
		// pre-commit reads SKIP case-sensitively except where environment
		// names ignore case (Windows).
		{`skip=gitleaks git commit -m x`, envNamesFoldCase},
		{`export Prek_Skip=x; git commit -m x`, envNamesFoldCase},
	}
	for _, tool := range []string{"Bash", "Monitor"} {
		for _, tt := range shells {
			t.Run(tool+"/"+tt.command, func(t *testing.T) {
				t.Parallel()
				reason, deny := GitCodeExecution(&EvalContext{ToolName: tool, Command: tt.command})
				if deny != tt.deny {
					t.Errorf("GitCodeExecution(%q) = (%q, %v), want deny=%v", tt.command, reason, deny, tt.deny)
				}
			})
		}
	}
	// A PowerShell session keeps what it sets, so a write of a commit-gate
	// variable is denied with or without git on the line.
	powershell := []struct {
		command string
		deny    bool
	}{
		{`$env:SKIP='gitleaks'`, true},
		{`$env:SKIP = 'gitleaks'; git commit -m x`, true},
		{`$Env:PREK_SKIP='x'`, true},
		{`$env:PRE_COMMIT_ALLOW_NO_CONFIG=1`, true},
		{`Set-Item Env:PREK_ALLOW_NO_CONFIG 1`, true},
		{`Remove-Item Env:\SKIP`, true},
		{`[Environment]::SetEnvironmentVariable('SKIP', 'gitleaks', 'User')`, true},
		{`pre-commit uninstall`, true},
		{`prek.exe uninstall`, true},
		{`pre-commit install --allow-missing-config`, true},
		{`git init --template=C:\t`, true},
		{`$env:SKIPPED='x'`, false},
		{`Write-Host $env:SKIP`, false},
		{`if ($env:SKIP -eq 'x') { git status }`, false},
		{`pre-commit install`, false},
		{`pre-commit run --all-files`, false},
		{`git commit -m x`, false},
	}
	for _, tt := range powershell {
		t.Run("PowerShell/"+tt.command, func(t *testing.T) {
			t.Parallel()
			reason, deny := GitCodeExecution(&EvalContext{ToolName: "PowerShell", Command: tt.command})
			if deny != tt.deny {
				t.Errorf("GitCodeExecution(%q) = (%q, %v), want deny=%v", tt.command, reason, deny, tt.deny)
			}
		})
	}
}
