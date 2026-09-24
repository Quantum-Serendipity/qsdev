package claudecode_test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

// pythonPlaceholderIndicators extracts PLACEHOLDER_INDICATORS from the
// embedded scan-secrets.py hook.
func pythonPlaceholderIndicators(t *testing.T) []string {
	t.Helper()
	src, err := claudecode.ExportTemplateFS.ReadFile("templates/hooks/scan-secrets.py")
	if err != nil {
		t.Fatalf("reading scan-secrets.py: %v", err)
	}
	block := regexp.MustCompile(`(?s)PLACEHOLDER_INDICATORS[^=]*=\s*\((.*?)\)`).FindSubmatch(src)
	if block == nil {
		t.Fatal("PLACEHOLDER_INDICATORS tuple not found in scan-secrets.py")
	}
	var got []string
	for _, m := range regexp.MustCompile(`'([^']*)'`).FindAllSubmatch(block[1], -1) {
		got = append(got, string(m[1]))
	}
	return got
}

// TestPlaceholderIndicators_MatchPythonHook pins F109: the Go list mirrors
// the Python hook exactly, and every entry is upper case because the hook
// compares against the upper-cased match (a lower-case entry never matches).
func TestPlaceholderIndicators_MatchPythonHook(t *testing.T) {
	t.Parallel()
	py := pythonPlaceholderIndicators(t)
	if !slices.Equal(py, claudecode.ExportPlaceholderIndicators) {
		t.Errorf("scan-secrets.py PLACEHOLDER_INDICATORS = %q, Go PlaceholderIndicators = %q", py, claudecode.ExportPlaceholderIndicators)
	}
	for _, ind := range py {
		if ind != strings.ToUpper(ind) {
			t.Errorf("indicator %q is not upper case, so the hook can never match it", ind)
		}
	}
}

// TestScanSecretsHook_PlaceholderIndicatorsTakeEffect runs the real hook to
// show that each indicator suppresses a match regardless of its case.
func TestScanSecretsHook_PlaceholderIndicatorsTakeEffect(t *testing.T) {
	t.Parallel()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping scan-secrets hook test")
	}
	hook, err := filepath.Abs(filepath.Join("templates", "hooks", "scan-secrets.py"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		content  string
		wantDeny bool
	}{
		{"lower-case dummy placeholder", `password = "dummy_password_value"`, false},
		{"lower-case sample placeholder", `secret: "sample-secret-value"`, false},
		{"test_key placeholder", `token = "my_test_key_value"`, false},
		{"real-looking secret", `password = "hunter2hunter2hunter2"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input, err := json.Marshal(map[string]any{
				"tool_name":  "Write",
				"tool_input": map[string]string{"file_path": "config.py", "content": tt.content},
			})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(python, hook)
			cmd.Env = append(cmd.Environ(), "CLAUDE_PROJECT_DIR="+t.TempDir())
			cmd.Stdin = bytes.NewReader(input)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("hook failed: %v", err)
			}
			denied := strings.Contains(string(out), `"deny"`)
			if denied != tt.wantDeny {
				t.Errorf("denied = %v, want %v (output %q)", denied, tt.wantDeny, out)
			}
		})
	}
}
