package claudecode_test

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// runLoggingHook runs a shipped hook script with payload on stdin through
// interpreter (bash, sh or python3) and returns its stdout. It skips when a
// required binary is missing.
func runLoggingHook(t *testing.T, interpreter, script string, payload any, env []string, args ...string) string {
	t.Helper()
	for _, bin := range []string{interpreter, "python3"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available; skipping %s test", bin, script)
		}
	}
	path, err := filepath.Abs(filepath.Join("templates", "hooks", script))
	if err != nil {
		t.Fatal(err)
	}
	in, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(interpreter, append([]string{path}, args...)...)
	cmd.Stdin = strings.NewReader(string(in))
	cmd.Env = hookEnv(t, env...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s failed: %v (stdout %q)", script, err, out)
	}
	return string(out)
}

// readJSONLines parses every line of the file at path as a JSON object and
// checks the file is private to its owner.
func readJSONLines(t *testing.T, path string) (string, []map[string]any) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("log not written: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("%s mode = %o, want 600", filepath.Base(path), info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []map[string]any
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		var e map[string]any
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("invalid JSON line %q: %v", sc.Text(), err)
		}
		entries = append(entries, e)
	}
	return string(data), entries
}

// TestAuditLogHook_MetadataOnly verifies audit-log.sh records what a tool
// touched but never its content or command text (W047): a .env write or a
// bearer token typed into Bash must not end up in a committable log.
func TestAuditLogHook_MetadataOnly(t *testing.T) {
	t.Parallel()
	project := t.TempDir()
	env := []string{"CLAUDE_PROJECT_DIR=" + project}
	secret := "hunter2" + "Xq9vLp4Zr"
	token := "ghp_" + strings.Repeat("Zq8", 12)

	payloads := []map[string]any{
		{"tool_name": "Write", "session_id": "s1", "tool_input": map[string]any{
			"file_path": filepath.Join(project, ".env"), "content": "DB_PASSWORD=" + secret,
		}},
		{"tool_name": "Bash", "session_id": "s1", "tool_input": map[string]any{
			"command": `curl -H "Authorization: Bearer ` + token + `" https://api.example.com`, "run_in_background": false,
		}},
	}
	for _, p := range payloads {
		if out := runLoggingHook(t, "bash", "audit-log.sh", p, env); strings.TrimSpace(out) != "" {
			t.Errorf("PostToolUse hook printed %q, want nothing", out)
		}
	}

	matches, err := filepath.Glob(filepath.Join(project, filepath.FromSlash(canon.HookLogDir), "audit-*.jsonl"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("audit log files = %v (%v), want one", matches, err)
	}
	raw, entries := readJSONLines(t, matches[0])
	if strings.Contains(raw, secret) || strings.Contains(raw, token) {
		t.Fatalf("audit log leaks tool input:\n%s", raw)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	write := entries[0]["input"].(map[string]any)
	if write["file_path"] != filepath.Join(project, ".env") {
		t.Errorf("file_path not recorded: %v", write)
	}
	content, ok := write["content"].(map[string]any)
	if !ok || content["sha256"] == "" || content["len"] == nil {
		t.Errorf("content fingerprint missing: %v", write["content"])
	}
	if entries[1]["tool"] != "Bash" || entries[1]["session_id"] != "s1" {
		t.Errorf("bash entry = %v", entries[1])
	}
}

// TestSembleAnalyticsHook reads the tool call from stdin, the only place
// Claude Code provides it (W048), and keeps queries with quotes as valid JSON.
func TestSembleAnalyticsHook(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not available; skipping semble analytics test")
	}
	project := t.TempDir()
	env := []string{"CLAUDE_PROJECT_DIR=" + project, "CLAUDE_TOOL_NAME="}
	queries := []string{"auth middleware", `say "hi\" x`}
	for _, q := range queries {
		p := map[string]any{"tool_name": "mcp__semble__search", "session_id": "s9", "tool_input": map[string]any{"query": q}}
		if out := runLoggingHook(t, "sh", "semble-analytics.sh", p, env); strings.TrimSpace(out) != "" {
			t.Errorf("PostToolUse hook printed %q, want nothing", out)
		}
	}
	other := map[string]any{"tool_name": "Read", "tool_input": map[string]any{"file_path": "x"}}
	runLoggingHook(t, "sh", "semble-analytics.sh", other, env)

	_, entries := readJSONLines(t, filepath.Join(project, filepath.FromSlash(canon.HookLogDir), "semble-searches.jsonl"))
	if len(entries) != len(queries) {
		t.Fatalf("entries = %d, want %d", len(entries), len(queries))
	}
	for i, q := range queries {
		e := entries[i]
		if e["query"] != q || e["tool"] != "mcp__semble__search" || e["sessionId"] != "s9" || e["projectRoot"] != project {
			t.Errorf("entry %d = %v", i, e)
		}
	}
}

// TestSOC2AuditHook records the session end reason instead of cost fields the
// hook input never carries, plus failed and denied tool calls (W049).
func TestSOC2AuditHook(t *testing.T) {
	t.Parallel()
	auditDir := t.TempDir()
	env := []string{"CLAUDE_AUDIT_DIR=" + auditDir, "CLAUDE_PROJECT_DIR=" + t.TempDir()}
	calls := []struct {
		event   string
		payload map[string]any
	}{
		{"session_start", map[string]any{"session_id": "s1", "source": "clear"}},
		{"tool_failure", map[string]any{"session_id": "s1", "tool_name": "Bash", "error": "exit 1", "is_interrupt": false}},
		{"permission_denied", map[string]any{"session_id": "s1", "tool_name": "WebFetch", "reason": "denied"}},
		{"session_end", map[string]any{"session_id": "s1", "reason": "clear", "transcript_path": "/x"}},
	}
	for _, c := range calls {
		runLoggingHook(t, "python3", "soc2-audit-log.py", c.payload, env, c.event)
	}
	matches, err := filepath.Glob(filepath.Join(auditDir, "claude-sessions-*.jsonl"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("audit files = %v (%v), want one", matches, err)
	}
	_, entries := readJSONLines(t, matches[0])
	if len(entries) != len(calls) {
		t.Fatalf("entries = %d, want %d", len(entries), len(calls))
	}
	for i, c := range calls {
		if entries[i]["event"] != c.event {
			t.Errorf("entry %d event = %v, want %s", i, entries[i]["event"], c.event)
		}
	}
	if entries[0]["start_source"] != "clear" {
		t.Errorf("session_start source = %v", entries[0]["start_source"])
	}
	if entries[1]["tool_name"] != "Bash" || entries[2]["tool_name"] != "WebFetch" {
		t.Errorf("tool entries = %v / %v", entries[1], entries[2])
	}
	end := entries[3]
	if end["end_reason"] != "clear" {
		t.Errorf("session_end reason = %v", end["end_reason"])
	}
	if _, ok := end["estimated_cost_usd"]; ok {
		t.Errorf("session_end reports a cost the hook input never carries: %v", end)
	}
}

// TestSOC2AuditHook_Registration covers every session start source and the
// failed/denied tool events (W049).
func TestSOC2AuditHook_Registration(t *testing.T) {
	t.Parallel()
	hooks := claudecode.ExportDefaultHookRegistry().BuildHooksMap(types.WizardAnswers{Hooks: types.HookChoices{SOC2Audit: true}})
	find := func(event, arg string) *claudecode.HookMatcher {
		for i, m := range hooks[event] {
			for _, h := range m.Hooks {
				if strings.HasSuffix(h.Command, "soc2-audit-log.py "+arg) {
					return &hooks[event][i]
				}
			}
		}
		return nil
	}
	start := find("SessionStart", "session_start")
	if start == nil || start.Matcher != "" {
		t.Errorf("SessionStart must match every source, got %+v", start)
	}
	if find("PostToolUseFailure", "tool_failure") == nil {
		t.Error("PostToolUseFailure not logged")
	}
	if find("PermissionDenied", "permission_denied") == nil {
		t.Error("PermissionDenied not logged")
	}
}

// TestHookTemplates_LogUnderHookLogDir pins that every generated hook that
// writes a project-local log writes it under canon.HookLogDir: the hook
// sandbox keeps only that directory writable inside the read-only .claude and
// project data directory, so a log anywhere else would silently stop being
// written once hooks are sandboxed.
func TestHookTemplates_LogUnderHookLogDir(t *testing.T) {
	t.Parallel()

	shellLogDir := regexp.MustCompile(`(?m)^LOG_DIR="\$\{CLAUDE_PROJECT_DIR:-\.\}/([^"]+)"`)
	pyParts := `"` + strings.Join(strings.Split(canon.HookLogDir, "/"), `", "`) + `"`
	// The shipped templates, not the source tree: the hook tests leave a
	// __pycache__ directory there that is never embedded.
	files, err := fs.Glob(claudecode.ExportTemplateFS, "templates/hooks/*")
	if err != nil {
		t.Fatal(err)
	}
	var shellLogs int
	for _, f := range files {
		data, err := fs.ReadFile(claudecode.ExportTemplateFS, f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range shellLogDir.FindAllStringSubmatch(string(data), -1) {
			shellLogs++
			if m[1] != canon.HookLogDir {
				t.Errorf("%s logs to %s, want %s", f, m[1], canon.HookLogDir)
			}
		}
	}
	if shellLogs < 2 {
		t.Errorf("found %d shell LOG_DIR assignments, want the audit-log and semble-analytics ones", shellLogs)
	}
	lib, err := fs.ReadFile(claudecode.ExportTemplateFS, "templates/hooks/_qsdev_hooklib.py")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lib), "os.path.join(base, "+pyParts+", AUDIT_LOG_NAME)") {
		t.Errorf("_qsdev_hooklib.py audit_log_path does not join %s", pyParts)
	}
}

// TestLoggingHooks_RefusePlantedSymlink: any sandboxed hook with a writable
// project can write the hook log directory, so the logging hooks must never
// append through a symlink planted there (or in its place), which would let
// that hook aim the next append at any file the user can write.
func TestLoggingHooks_RefusePlantedSymlink(t *testing.T) {
	t.Parallel()
	semble := map[string]any{"tool_name": "mcp__semble__search", "tool_input": map[string]any{"query": "q"}}
	audit := map[string]any{"tool_name": "Bash", "tool_input": map[string]any{"command": "ls"}}
	now := time.Now()
	auditNames := []string{
		"audit-" + now.Format("2006-01-02") + ".jsonl",
		"audit-" + now.AddDate(0, 0, 1).Format("2006-01-02") + ".jsonl",
	}
	tests := []struct {
		name, interpreter, script string
		needs                     string
		payload                   any
		files                     []string // log names planted as symlinks; none: the directory is
	}{
		{"audit-log file", "bash", "audit-log.sh", "", audit, auditNames},
		{"audit-log directory", "bash", "audit-log.sh", "", audit, nil},
		{"semble file", "sh", "semble-analytics.sh", "jq", semble, []string{"semble-searches.jsonl"}},
		{"semble directory", "sh", "semble-analytics.sh", "jq", semble, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.needs != "" {
				if _, err := exec.LookPath(tt.needs); err != nil {
					t.Skipf("%s not available", tt.needs)
				}
			}
			project, elsewhere := t.TempDir(), t.TempDir()
			logDir := filepath.Join(project, filepath.FromSlash(canon.HookLogDir))
			if err := os.MkdirAll(filepath.Dir(logDir), 0o755); err != nil {
				t.Fatal(err)
			}
			victim := filepath.Join(elsewhere, "victim")
			if err := os.WriteFile(victim, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if tt.files == nil {
				if err := os.Symlink(elsewhere, logDir); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			} else {
				if err := os.Mkdir(logDir, 0o700); err != nil {
					t.Fatal(err)
				}
				for _, f := range tt.files {
					if err := os.Symlink(victim, filepath.Join(logDir, f)); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				}
			}
			runLoggingHook(t, tt.interpreter, tt.script, tt.payload, []string{"CLAUDE_PROJECT_DIR=" + project})

			if data, err := os.ReadFile(victim); err != nil || len(data) != 0 {
				t.Errorf("victim = %q (err %v): the hook appended through a planted symlink", data, err)
			}
			if entries, err := os.ReadDir(elsewhere); err != nil || len(entries) != 1 {
				t.Errorf("the hook wrote through the symlinked log directory: %v (err %v)", entries, err)
			}
		})
	}
}
