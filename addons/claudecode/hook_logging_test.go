package claudecode_test

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
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
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	cmd.Env = append(cmd.Env, env...)
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

	matches, err := filepath.Glob(filepath.Join(project, ".claude", "logs", "audit-*.jsonl"))
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

	_, entries := readJSONLines(t, filepath.Join(project, ".qsdev", "analytics", "semble-searches.jsonl"))
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
