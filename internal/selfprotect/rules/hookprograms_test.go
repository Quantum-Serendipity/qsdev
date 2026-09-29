package rules

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// hookFixture is a session whose settings register hooks by program name
// (qsdev), by script path (a Python hook with an env shebang), and inline
// (gofmt in a case statement), with a PATH of an empty early directory, the
// directory the programs resolve to, and a later one holding a second python3.
type hookFixture struct {
	env                       hookEnv
	project, early, bin, late string
}

func newHookFixture(t *testing.T) hookFixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := hookFixture{
		project: filepath.Join(root, "project"),
		early:   filepath.Join(root, "early"),
		bin:     filepath.Join(root, "bin"),
		late:    filepath.Join(root, "late"),
	}
	settings := `{"hooks": {
	  "PreToolUse": [{"matcher": "*", "hooks": [
	    {"type": "command", "command": "qsdev selfprotect"},
	    {"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/guard.py"}
	  ]}],
	  "PostToolUse": [{"matcher": "Write", "hooks": [
	    {"type": "command", "command": "case \"$F\" in *.go) gofmt -w \"$F\" ;; esac; exit 0"}
	  ]}]
	}}`
	files := map[string]string{
		filepath.Join(f.project, ".claude", "settings.json"):     settings,
		filepath.Join(f.project, ".claude", "hooks", "guard.py"): "#!/usr/bin/env python3\nprint('ok')\n",
		filepath.Join(f.bin, "qsdev"):                            "binary",
		filepath.Join(f.bin, "python3"):                          "binary",
		filepath.Join(f.bin, "gofmt"):                            "binary",
		filepath.Join(f.late, "python3"):                         "binary",
	}
	for p, content := range files {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(f.early, 0o755); err != nil {
		t.Fatal(err)
	}
	f.env = hookEnv{
		projectDir:    f.project,
		settingsFiles: []string{filepath.Join(f.project, ".claude", "settings.json")},
		pathDirs:      []string{f.early, f.bin, f.late},
	}
	return f
}

func (f hookFixture) eval(tool, target, command string) (Verdict, string) {
	env := f.env
	ctx := EvalContext{ToolName: tool, CanonicalPath: target, FilePath: target, Command: command, CWD: f.project, hookEnv: &env}
	return sp011.Evaluate(&ctx)
}

// TestSP011_HookCommandHijack covers F150: SP-011 guards the command path of
// every registered hook — a file of a hook program's name in a PATH directory
// ahead of the real one, the file it resolves to, and a hook script's shebang
// interpreter — against Write/Edit and shell writes.
func TestSP011_HookCommandHijack(t *testing.T) {
	t.Parallel()
	f := newHookFixture(t)
	shadowQsdev := filepath.Join(f.early, "qsdev")
	realPython := filepath.Join(f.bin, "python3")

	writes := []struct {
		name, target string
		want         Verdict
	}{
		{"shadow hook program in earlier PATH dir", shadowQsdev, Deny},
		{"rewrite resolved interpreter", realPython, Deny},
		{"shadow inline hook program", filepath.Join(f.early, "gofmt"), Deny},
		{"program later in PATH is not run", filepath.Join(f.late, "python3"), Allow},
		{"unrelated file in PATH dir", filepath.Join(f.early, "other"), Allow},
		{"project source", filepath.Join(f.project, "main.go"), Allow},
	}
	for _, tt := range writes {
		t.Run("Write/"+tt.name, func(t *testing.T) {
			t.Parallel()
			if v, reason := f.eval("Write", tt.target, ""); v != tt.want {
				t.Errorf("sp011 Write %s = %v (%s), want %v", tt.target, v, reason, tt.want)
			}
		})
	}

	// Commands spell paths the way a shell reads them: with forward slashes.
	// Claude Code's Bash tool runs Git Bash on Windows, where an unquoted
	// backslash is an escape, so C:\Users\x\qsdev names C:Usersxqsdev.
	early, shadowSh := filepath.ToSlash(f.early), filepath.ToSlash(shadowQsdev)
	sh := func(elem ...string) string { return filepath.ToSlash(filepath.Join(elem...)) }
	commands := []struct {
		name, command string
		want          Verdict
	}{
		{"copy over shadow path", "cp /tmp/evil " + shadowSh, Deny},
		{"copy over quoted native shadow path", "cp /tmp/evil '" + shadowQsdev + "'", Deny},
		{"redirect into shadow path", "echo x > " + sh(f.early, "gofmt"), Deny},
		{"symlink shadow", "ln -s /tmp/evil " + shadowSh, Deny},
		{"move over resolved program", "mv /tmp/evil " + filepath.ToSlash(realPython), Deny},
		{"chmod resolved program", "chmod -x " + sh(f.bin, "qsdev"), Deny},
		{"copy into PATH directory", "cp /tmp/evil/qsdev " + early + "/", Deny},
		{"copy with target directory option", "cp -t " + early + " /tmp/evil/qsdev", Deny},
		{"install with target directory option", "install --target-directory=" + early + " /tmp/evil/gofmt", Deny},
		{"dd output operand", "dd if=/tmp/evil of=" + shadowSh, Deny},
		{"relative after cd", "cd " + early + " && cp /tmp/e python3", Deny},
		{"brace expansion", "cp /tmp/e " + early + "/{qsdev,other}", Deny},
		{"glob", "cp /tmp/e " + early + "/qsd?v", Deny},
		{"directory from expansion", `cp /tmp/e "$D/python3"`, Deny},
		{"sh -c script", "sh -c 'cp /tmp/e " + shadowSh + "'", Deny},
		{"inline python program", `python3 -c "open('` + shadowSh + `','w')"`, Deny},
		{"unparseable naming target", "cp /tmp/e " + shadowSh + ` "unterminated`, Deny},
		{"read program", "cat " + sh(f.bin, "qsdev"), Allow},
		{"list PATH dir", "ls " + early, Allow},
		{"copy unrelated into PATH dir", "cp /tmp/tool " + early + "/", Allow},
		{"copy unrelated with target directory option", "cp -t " + early + " /tmp/tool", Allow},
		{"rsync times flag is not a target directory", "rsync -t " + early + " /tmp/qsdev", Allow},
		{"build into non-PATH dir", "go build -o ./bin/qsdev ./cmd/qsdev", Allow},
		{"expansion to other name", `cp /tmp/e "$D/unrelated"`, Allow},
		{"run hook program", "qsdev status", Allow},
	}
	for _, tt := range commands {
		t.Run("Bash/"+tt.name, func(t *testing.T) {
			t.Parallel()
			if v, reason := f.eval("Bash", "", tt.command); v != tt.want {
				t.Errorf("sp011(%q) = %v (%s), want %v", tt.command, v, reason, tt.want)
			}
		})
	}
}

// TestHookEnvFromProcess checks that the session description comes from the
// environment Claude Code gives its hooks.
func TestHookEnvFromProcess(t *testing.T) {
	project := filepath.FromSlash("/work/project")
	configDir := filepath.FromSlash("/home/alice/.config/claude")
	dirs := []string{filepath.FromSlash("/opt/a"), filepath.FromSlash("/opt/b")}
	t.Setenv("CLAUDE_PROJECT_DIR", project)
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	t.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))

	env := hookEnvFromProcess(filepath.FromSlash("/elsewhere"))
	if env.projectDir != project {
		t.Errorf("projectDir = %q, want %q", env.projectDir, project)
	}
	if !slices.Equal(env.pathDirs, dirs) {
		t.Errorf("pathDirs = %q, want %q", env.pathDirs, dirs)
	}
	for _, want := range []string{
		filepath.Join(project, ".claude", "settings.json"),
		filepath.Join(project, ".claude", "settings.local.json"),
		filepath.Join(configDir, "settings.json"),
	} {
		if !slices.Contains(env.settingsFiles, want) {
			t.Errorf("settingsFiles = %q, missing %q", env.settingsFiles, want)
		}
	}
}

func TestShebangProgram(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tests := []struct {
		name, content, want string
	}{
		{"env lookup", "#!/usr/bin/env python3\n", "python3"},
		{"env split string", "#!/usr/bin/env -S python3 -u\n", "python3"},
		{"env assignment", "#!/usr/bin/env PYTHONSAFEPATH=1 python3\n", "python3"},
		{"direct interpreter", "#!/bin/bash -e\n", "/bin/bash"},
		{"space after bang", "#! /usr/bin/python3\n", "/usr/bin/python3"},
		{"no newline", "#!/bin/sh", "/bin/sh"},
		{"no shebang", "print('x')\n", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "-"))
			if err := os.WriteFile(p, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := shebangProgram(p); got != tt.want {
				t.Errorf("shebangProgram(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestHookCommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tests := []struct {
		name, content string
		want          []string
	}{
		{"command hooks", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"a"},{"type":"prompt","prompt":"p"},{"command":"b"}]}]}}`, []string{"a", "b"}},
		{"no hooks", `{"permissions":{}}`, nil},
		{"invalid JSON", `{`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "-")+".json")
			if err := os.WriteFile(p, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := hookCommands(p); !slices.Equal(got, tt.want) {
				t.Errorf("hookCommands = %q, want %q", got, tt.want)
			}
		})
	}
	if got := hookCommands(filepath.Join(dir, "missing.json")); got != nil {
		t.Errorf("hookCommands(missing) = %q, want none", got)
	}
}
