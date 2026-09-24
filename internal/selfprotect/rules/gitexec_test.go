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
