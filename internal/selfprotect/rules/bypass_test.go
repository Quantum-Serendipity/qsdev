package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// bashCase is one Bash command evaluated against the full Tier-1 rule set.
type bashCase struct {
	name    string
	command string
	cwd     string // defaults to <home>/project
}

// evalBash evaluates a Bash command against the full Tier-1 rule set.
func evalBash(t *testing.T, c bashCase) (Verdict, []RuleMatch) {
	t.Helper()
	cwd := c.cwd
	if cwd == "" {
		cwd = filepath.Join(homeDir(t), "project")
	}
	ctx := EvalContext{ToolName: "Bash", Command: c.command, CWD: cwd}
	return Tier1Rules.EvaluateAll(&ctx)
}

func runBashCases(t *testing.T, cases []bashCase, want Verdict) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if v, matches := evalBash(t, c); v != want {
				t.Errorf("EvaluateAll(%q) = %v (first: %s), want %v", c.command, v, firstRuleID(matches), want)
			}
		})
	}
}

// TestBash_CwdTrackingBypasses covers F134: a cd into a protected directory
// followed by a relative write, in one call or across calls (the session cwd
// persists, and arrives in the hook envelope).
func TestBash_CwdTrackingBypasses(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")

	runBashCases(t, []bashCase{
		{name: "cd home claude then clobber settings", command: `cd ~/.claude && echo '{}' > settings.json`},
		{name: "cd project claude then write hook", command: "cd .claude && echo evil > hooks/tool-gates.py"},
		{name: "cd hooks then append hook", command: "cd .claude/hooks; echo evil >> tool-gates.py"},
		{name: "pushd claude then delete settings", command: "pushd .claude && rm settings.json"},
		{name: "cd through expansion into claude", command: `cd "$HOME/.claude" && echo x > settings.json`},
		{name: "cd via glob into claude", command: "cd .c*e && echo x > settings.json"},
		{name: "cd hooks then remove parent", command: "cd .claude/hooks && rm -rf .."},
		{name: "cd found through CDPATH", command: "CDPATH=.claude; cd hooks && echo evil > tool-gates.py"},
		{name: "pushd rotates back into claude", command: "pushd .claude; pushd /tmp; pushd +1; echo x > settings.json"},
		{name: "session cwd is project claude dir", command: "echo evil > hooks/x.py", cwd: filepath.Join(project, ".claude")},
		{name: "session cwd is hooks dir", command: "echo evil >> tool-gates.py", cwd: filepath.Join(project, ".claude", "hooks")},
		{name: "session cwd is home claude dir", command: "echo '{}' > settings.json", cwd: filepath.Join(home, ".claude")},
		{name: "session cwd is home qsdev bin", command: "cp /tmp/evil qsdev", cwd: filepath.Join(home, ".qsdev", "bin")},
	}, Deny)

	runBashCases(t, []bashCase{
		{name: "cd claude then read settings", command: "cd .claude && cat settings.json"},
		{name: "cd claude then read back out", command: "cd .claude && cat ../README.md"},
		{name: "cd src then write", command: "cd src && echo x > main.go"},
		{name: "cd claude then back out and write", command: "cd .claude && cd .. && echo x > notes.txt"},
		// A Claude Code worktree under .claude/worktrees/ is an ordinary
		// checkout: working inside it is not working in a protected directory.
		{name: "work inside a claude worktree", command: "echo x > main.go", cwd: filepath.Join(project, ".claude", "worktrees", "wt")},
		{name: "cd into a claude worktree and build", command: "cd " + filepath.ToSlash(filepath.Join(project, ".claude", "worktrees", "wt")) + " && make build"},
	}, Allow)
}

// TestBash_SymlinkedPathsResolve covers the canonicalization half of F134: a
// path that reaches a protected directory through a symlink is protected.
func TestBash_SymlinkedPathsResolve(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	claudeDir := filepath.Join(project, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "c")
	if err := os.Symlink(claudeDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(claudeDir, filepath.Join(project, "cfg")); err != nil {
		t.Fatal(err)
	}

	runBashCases(t, []bashCase{
		{name: "absolute path through symlink", command: "echo '{}' > " + filepath.ToSlash(filepath.Join(link, "settings.json")), cwd: project},
		{name: "relative path through symlink", command: "sed -i d cfg/settings.json", cwd: project},
	}, Deny)
	runBashCases(t, []bashCase{
		{name: "read through symlink", command: "cat cfg/settings.json", cwd: project},
	}, Allow)
}

// TestBash_ShellSpellingBypasses covers F135/F560: quote-splitting, escapes,
// globs, braces, ANSI-C strings, doubled slashes and find patterns that hide a
// protected path from a raw-substring trigger.
func TestBash_ShellSpellingBypasses(t *testing.T) {
	t.Parallel()

	runBashCases(t, []bashCase{
		{name: "empty double quotes in home path", command: `rm ~/.cl""aude/settings.json`},
		{name: "backslash escape in home path", command: `rm ~/.cl\aude/settings.json`},
		{name: "question-mark glob", command: "rm ~/.clau?e/settings.json"},
		{name: "star glob", command: "rm ~/.c*e/settings.json"},
		{name: "empty single quotes in redirect", command: `echo x > ~/.cl''aude/settings.json`},
		{name: "doubled slash system path", command: "rm /etc//claude-code/managed-settings.json"},
		{name: "find by name and path pattern", command: "find ~ -name settings.json -path '*claude*' -delete"},
		{name: "find from the filesystem root", command: "find / -name settings.json -delete"},
		{name: "find in project by pattern", command: "find . -name settings.json -path '*claude*' -delete"},
		{name: "ansi-c string after trigger", command: `grep x .claude/settings.json; rm ~/$'\x2e'claude/settings.json`},
		{name: "ansi-c string alone", command: `rm ~/$'\x2e'claude/settings.json`},
		{name: "quoted tail of project dir", command: "rm -rf .cl'aude'"},
		{name: "quote-split project settings", command: `rm -rf ./.cla""ude/settings.json`},
		{name: "glob of project dir", command: "rm -rf .clau*"},
		{name: "quoted tail of home dir", command: "rm -rf ~/.cl'aude'/settings.json"},
		{name: "brace expansion", command: "rm ~/.{claude,x}/settings.json"},
		{name: "quote-split verb", command: "r''m .claude/settings.json"},
		{name: "glob system dir", command: "rm /etc/g?ev/policy.yaml"},
	}, Deny)

	runBashCases(t, []bashCase{
		{name: "glob of build outputs", command: "rm -rf ./build*"},
		{name: "glob of log files", command: "rm -f *.log"},
		{name: "find temp files", command: "find . -name '*.tmp' -delete"},
		{name: "read through glob", command: "ls ~/.cl*"},
		{name: "brace expansion of unrelated files", command: "rm {a,b}.txt"},
		{name: "embedded token is not the protected dir", command: "rm -f my.claude.bak"},
	}, Allow)
}

// TestBash_ArbitraryMutatorBypasses covers F136/F561: any command that writes a
// protected path is denied, not just a fixed list of delete/copy verbs.
func TestBash_ArbitraryMutatorBypasses(t *testing.T) {
	t.Parallel()

	runBashCases(t, []bashCase{
		{name: "sed in place on home settings", command: "sed -i 's/selfprotect/true/' ~/.claude/settings.json"},
		{name: "sed delete on project settings", command: "sed -i '/selfprotect/d' .claude/settings.json"},
		{name: "perl in place", command: "perl -pi -e 's/a/b/' ~/.claude/settings.json"},
		{name: "perl in place on qsdev policy", command: "perl -pi -e 's/deny/x/' ~/.qsdev/policy.yaml"},
		{name: "python open for write", command: `python3 -c "open('.claude/settings.json','w').write('{}')"`},
		{name: "ex script", command: "ex -sc '%d|x' ~/.claude/settings.json"},
		{name: "vim batch mode", command: "vim -es -c ':wq' ~/.claude/settings.json"},
		{name: "install over home settings", command: "install -m644 /tmp/x ~/.claude/settings.json"},
		{name: "install over project settings", command: "install -m 644 /tmp/evil .claude/settings.json"},
		{name: "sed on audit log", command: "sed -i d ~/.qsdev/audit/log.jsonl"},
		{name: "patch", command: "patch .claude/settings.json < /tmp/p"},
		{name: "unzip over settings", command: "unzip -o /tmp/x.zip .claude/settings.json"},
		{name: "chmod settings", command: "chmod 666 .claude/settings.json"},
		{name: "chown settings", command: "chown nobody .claude/settings.json"},
		{name: "awk inplace", command: "awk -i inplace '{print}' .claude/settings.json"},
		{name: "tar extract into claude", command: "tar -xf /tmp/x.tar -C .claude"},
		{name: "dd onto settings", command: "dd if=/tmp/x of=.claude/settings.json"},
		{name: "unknown binary on settings", command: "./tool --write .claude/settings.json"},
		{name: "piped protected path into xargs sed", command: "echo .claude/settings.json | xargs sed -i d"},
	}, Deny)

	runBashCases(t, []bashCase{
		{name: "cat settings", command: "cat .claude/settings.json"},
		{name: "grep claude dir", command: "grep -r deny .claude/"},
		{name: "jq settings", command: "jq . .claude/settings.json"},
		{name: "diff settings", command: "diff .claude/settings.json /tmp/x"},
		{name: "sed on another file while reading settings", command: "sed -n 1p README.md && cat .claude/settings.json"},
		{name: "cat settings through a filter", command: "cat .claude/settings.json | grep foo"},
		{name: "run the installed binary", command: "~/.qsdev/bin/qsdev status"},
		// An interpreter reads and runs its script file; it does not write it.
		{name: "run a hook script", command: "bash .claude/hooks/test.sh"},
		// --exclude values are patterns, not operands.
		{name: "tar excluding claude dir", command: "tar czf backup.tgz --exclude=.claude ."},
	}, Allow)

	runBashCases(t, []bashCase{
		{name: "script given a protected argument", command: "python3 tool.py .claude/settings.json"},
		{name: "inline program", command: "bash -c 'echo x > .claude/settings.json'"},
		{name: "protected read piped into a shell", command: "cat .claude/settings.json | sh"},
		// Only tar and rsync have modelled pattern options; elsewhere an
		// --include= value is an opaque, potentially written, path.
		{name: "pattern-looking option of an opaque command", command: `python3 -c "import sys" --include=.claude/settings.json`},
		{name: "shell program in a here-document", command: "bash <<'EOF'\necho x > .claude/settings.json\nEOF"},
		{name: "shell program in a here-string", command: "sh <<< 'rm .claude/settings.json'"},
		{name: "protected paths fed to xargs by redirect", command: "xargs rm < .claude/list.txt"},
		// A linking copy makes a writable alias of a protected file.
		{name: "hard-link copy of settings", command: "cp -l .claude/settings.json alias.json"},
		{name: "clustered link copy of settings", command: "cp -al .claude settings-alias"},
		{name: "symbolic-link copy of settings", command: "cp --symbolic-link .claude/settings.json alias.json"},
	}, Deny)
}

// TestBash_FlagEmbeddedPathBypasses covers F138: protected paths carried in an
// option value (--opt=value, -t<dir>, -t <dir>) are destinations too.
func TestBash_FlagEmbeddedPathBypasses(t *testing.T) {
	t.Parallel()

	runBashCases(t, []bashCase{
		{name: "cp long target directory", command: "cp --target-directory=.claude/hooks /tmp/evil.py"},
		{name: "mv long target directory", command: "mv --target-directory=.claude /tmp/settings.json"},
		{name: "tar long directory", command: "tar -xf /tmp/evil.tar --directory=.claude/hooks"},
		{name: "cp attached short target", command: "cp -t.claude/hooks /tmp/evil.py"},
		{name: "cp separate short target", command: "cp -t .claude/hooks /tmp/evil.py"},
		{name: "cp protected source to outside target dir", command: "cp -t /tmp/out .claude/settings.json"},
	}, Deny)

	runBashCases(t, []bashCase{
		{name: "cp protected source to in-repo target dir", command: "cp -t backup .claude/settings.json"},
		{name: "rsync excluding claude dir", command: "rsync -a --exclude .claude/ src/ dst/"},
		// rsync's -t preserves times; it is not cp's target-directory option.
		{name: "rsync -t backup of protected config in repo", command: "rsync -t .claude/settings.json settings.bak"},
	}, Allow)
}

// TestBash_AreaRulesCoverWholeDirs covers F146: the hook, audit, and binary
// rules deny mutations of the directory itself (no trailing slash) and of its
// ancestors, with any chmod mode, and through wrappers.
func TestBash_AreaRulesCoverWholeDirs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rule    *Rule
		command string
		verdict Verdict
	}{
		{&sp010, "chmod -R a-x .claude/hooks", Deny},
		{&sp010, "chmod a-x .claude/hooks", Deny},
		{&sp010, "chmod -R 000 .claude", Deny},
		{&sp010, "sh -c 'chmod 0 .claude/hooks/pre.sh'", Deny},
		{&sp010, "cd .claude/hooks && sed -i s/deny/allow/ pre.sh", Deny},
		{&sp010, "chmod 644 .claude/settings.json", Allow}, // not a hook (SP-007's concern)
		{&sp010, "cat .claude/hooks/pre.sh", Allow},
		{&int001, "chmod a-x ~/.qsdev/bin/qsdev", Deny},
		{&int001, "chmod 000 ~/.qsdev/bin", Deny},
		{&int001, "chmod -R 000 ~/.qsdev", Deny},
		{&int001, "cp /tmp/evil ~/.qsdev/bin/qsdev", Deny},
		{&int001, "sh -c 'chmod 0 ~/.qsdev/bin/qsdev'", Deny},
		{&int001, "ls -l ~/.qsdev/bin", Allow},
		{&int001, "~/.qsdev/bin/qsdev status", Allow},
		{&sp013, "sed -i d ~/.qsdev/audit/log.jsonl", Deny},
		{&sp013, "rm -rf ~/.qsdev", Deny},
		{&sp013, "tail ~/.qsdev/audit/log.jsonl", Allow},
	}
	for _, tt := range tests {
		t.Run(tt.rule.ID+" "+tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: filepath.Join(homeDir(t), "project")}
			if v, _ := tt.rule.Evaluate(&ctx); v != tt.verdict {
				t.Errorf("%s(%q) = %v, want %v", tt.rule.ID, tt.command, v, tt.verdict)
			}
		})
	}
}

// TestBash_ProcSpellings covers F148: every pid spelling of the sensitive /proc
// entries is denied.
func TestBash_ProcSpellings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ctx     EvalContext
		verdict Verdict
	}{
		{"shell pid", EvalContext{ToolName: "Bash", Command: "cat /proc/$$/environ"}, Deny},
		{"thread-self", EvalContext{ToolName: "Bash", Command: "cat /proc/thread-self/environ"}, Deny},
		{"task entry", EvalContext{ToolName: "Bash", Command: "cat /proc/self/task/1/environ"}, Deny},
		{"parent pid variable", EvalContext{ToolName: "Bash", Command: "cat /proc/$PPID/environ"}, Deny},
		{"braced pid variable", EvalContext{ToolName: "Bash", Command: "cat /proc/${PPID}/cmdline"}, Deny},
		{"command substitution pid", EvalContext{ToolName: "Bash", Command: `cat "/proc/$(pgrep claude)/environ"`}, Deny},
		{"root traversal", EvalContext{ToolName: "Bash", Command: "ls /proc/thread-self/root/etc"}, Deny},
		{"read thread-self environ", EvalContext{ToolName: "Read", CanonicalPath: "/proc/thread-self/environ"}, Deny},
		{"meminfo", EvalContext{ToolName: "Bash", Command: "cat /proc/meminfo"}, Allow},
		{"fdinfo is not fd", EvalContext{ToolName: "Bash", Command: "ls /proc/self/fdinfo"}, Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if v, _ := sp006.Evaluate(&tt.ctx); v != tt.verdict {
				t.Errorf("sp006(%+v) = %v, want %v", tt.ctx, v, tt.verdict)
			}
		})
	}
}

// TestBash_UnparseableFailsClosed covers F565: every Bash rule denies an
// unparseable command (an unterminated quote) that mentions its protected
// target, rather than falling open when the argv analysis cannot run.
func TestBash_UnparseableFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rule    *Rule
		command string
	}{
		{&sp003, "rm .claude/settings.json 'unterminated"},
		{&sp004, "ln -s /tmp/x .claude/settings.json 'unterminated"},
		{&sp005, "cat ../../.claude/settings.json 'unterminated"},
		{&sp007, "echo x > .claude/settings.json 'unterminated"},
		{&sp010, "chmod 000 .claude/hooks/pre.sh 'unterminated"},
		{&sp013, "rm .qsdev/audit/log.jsonl 'unterminated"},
		{&int001, "chmod 000 ~/.qsdev/bin/qsdev 'unterminated"},
		{&mcp005, "echo x > .mcp.json 'unterminated"},
	}
	for _, tt := range tests {
		t.Run(tt.rule.ID, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: filepath.Join(homeDir(t), "project")}
			if _, err := ctx.ParsedCommands(); err == nil {
				t.Fatalf("test command %q unexpectedly parsed", tt.command)
			}
			if v, _ := tt.rule.Evaluate(&ctx); v != Deny {
				t.Errorf("%s(%q) = %v, want Deny (fail closed)", tt.rule.ID, tt.command, v)
			}
		})
	}
}

// TestMCP002_ToolPathArguments covers F147: MCP tools name paths through
// path/paths/source/destination, not only file_path.
func TestMCP002_ToolPathArguments(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")

	tests := []struct {
		name    string
		input   string
		verdict Verdict
	}{
		{"path to home claude settings", `{"path":"~/.claude/settings.json"}`, Deny},
		{"paths list with qsdev config", `{"paths":["/tmp/a","` + filepath.ToSlash(filepath.Join(home, ".qsdev", "config.yaml")) + `"]}`, Deny},
		{"relative source mcp config", `{"source":".mcp.json","destination":"/tmp/x"}`, Deny},
		{"destination hook script", `{"source":"/tmp/x","destination":".claude/hooks/pre.sh"}`, Deny},
		{"ordinary project file", `{"path":"src/main.go"}`, Allow},
		{"non-string path fields", `{"path":3,"paths":"x"}`, Allow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "mcp__filesystem__move_file", CWD: project, ToolInput: json.RawMessage(tt.input)}
			if v, _ := mcp002.Evaluate(&ctx); v != tt.verdict {
				t.Errorf("mcp002(%s) = %v, want %v", tt.input, v, tt.verdict)
			}
		})
	}
}
