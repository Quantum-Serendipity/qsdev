package check

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
)

func TestHookHostPrograms(t *testing.T) {
	t.Parallel()
	const guard = `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`
	envPython := map[string]string{".claude/hooks/package-guard.py": "#!/usr/bin/env python3\nprint()\n"}
	tests := []struct {
		name    string
		project []string          // hook commands in settings.json
		local   []string          // hook commands in settings.local.json
		disable bool              // settings.json sets disableAllHooks
		scripts map[string]string // project files to create (executable), by content
		want    []string
	}{
		{name: "env-python3 script run as a program", project: []string{guard}, scripts: envPython, want: []string{"python3"}},
		{name: "fail-closed env-python3 script", project: []string{claudesettings.FailClosedCommand("package-guard", guard)}, scripts: envPython, want: []string{"python3"}},
		{name: "absolute interpreter script", project: []string{guard}, scripts: map[string]string{".claude/hooks/package-guard.py": "#!/usr/bin/python3\n"}},
		{name: "script without interpreter line", project: []string{guard}, scripts: map[string]string{".claude/hooks/package-guard.py": "echo hi\n"}},
		{name: "missing script", project: []string{guard}},
		{name: "python3 interpreter of a project script", project: []string{`python3 "$CLAUDE_PROJECT_DIR"/x.py`}, want: []string{"python3"}},
		{name: "fail-closed qsdev selfprotect", project: []string{claudesettings.FailClosedCommand("self-protection", "qsdev selfprotect")}, want: []string{"qsdev"}},
		{name: "guarded optional program", project: []string{`command -v x >/dev/null || exit 0; x`}},
		{name: "absolute-path program", project: []string{"/opt/tools/guard check"}},
		{name: "disableAllHooks", project: []string{"qsdev selfprotect", guard}, disable: true, scripts: envPython},
		{name: "local settings hooks included", project: []string{"qsdev selfprotect"}, local: []string{"gofmt -l ."}, want: []string{"gofmt", "qsdev"}},
		{name: "sorted and deduped", project: []string{guard, "qsdev selfprotect", `python3 "$CLAUDE_PROJECT_DIR"/x.py`}, scripts: envPython, want: []string{"python3", "qsdev"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for rel, content := range tt.scripts {
				writeTestScript(t, dir, rel, content)
			}
			writeTestFile(t, dir, claudesettings.ProjectRelPath, hookSettingsJSON(t, tt.project, tt.disable))
			if tt.local != nil {
				writeTestFile(t, dir, claudesettings.LocalRelPath, hookSettingsJSON(t, tt.local, false))
			}
			got, err := HookHostPrograms(dir)
			if err != nil {
				t.Fatalf("HookHostPrograms: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("HookHostPrograms = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHookHostPrograms_NoSettings(t *testing.T) {
	t.Parallel()
	got, err := HookHostPrograms(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Errorf("HookHostPrograms(empty project) = %q, %v; want nothing", got, err)
	}
}

// hookSettingsJSON renders a settings.json registering each command as a
// PreToolUse hook.
func hookSettingsJSON(t *testing.T, commands []string, disable bool) string {
	t.Helper()
	hooks := make([]map[string]string, 0, len(commands))
	for _, c := range commands {
		hooks = append(hooks, map[string]string{"type": "command", "command": c})
	}
	doc := map[string]any{
		"hooks": map[string]any{"PreToolUse": []any{map[string]any{"matcher": "*", "hooks": hooks}}},
	}
	if disable {
		doc[claudesettings.KeyDisableAllHooks] = true
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
