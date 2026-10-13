package claudecode_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
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

// scanSecretsHook returns the python3 interpreter and the absolute path of
// the scan-secrets.py template, skipping the test when python3 is missing.
func scanSecretsHook(t *testing.T) (python, hook string) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping scan-secrets hook test")
	}
	hook, err = filepath.Abs(filepath.Join("templates", "hooks", "scan-secrets.py"))
	if err != nil {
		t.Fatal(err)
	}
	return python, hook
}

// runScanSecretsWrite runs the real hook on a Write of content to filePath
// and returns its permission decision ("" when it allowed silently) and the
// decision reason.
func runScanSecretsWrite(t *testing.T, python, hook, filePath, content string) (decision, reason string) {
	t.Helper()
	input, err := json.Marshal(map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]string{"file_path": filePath, "content": content},
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
	var resp struct {
		HookSpecificOutput struct {
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return "", ""
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("hook output %q is not JSON: %v", out, err)
	}
	return resp.HookSpecificOutput.PermissionDecision, resp.HookSpecificOutput.PermissionDecisionReason
}

// TestScanSecretsHook_PlaceholderIndicatorsTakeEffect runs the real hook to
// show that each indicator suppresses a match regardless of its case.
func TestScanSecretsHook_PlaceholderIndicatorsTakeEffect(t *testing.T) {
	t.Parallel()
	python, hook := scanSecretsHook(t)

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
			decision, reason := runScanSecretsWrite(t, python, hook, "config.py", tt.content)
			if denied := decision == "deny"; denied != tt.wantDeny {
				t.Errorf("denied = %v, want %v (reason %q)", denied, tt.wantDeny, reason)
			}
		})
	}
}

// TestScanSecretsHook_FixedLengthTokenBeforePlaceholderWord pins R2: an npm
// token is exactly 36 characters after npm_, so a placeholder word written
// straight after a real token (`npm_<36>TODO`) must not be swallowed into the
// match, where the placeholder filter would skip the real token with it.
func TestScanSecretsHook_FixedLengthTokenBeforePlaceholderWord(t *testing.T) {
	t.Parallel()
	python, hook := scanSecretsHook(t)
	token := "npm_" + strings.Repeat("Ab1", 12)
	tests := []struct {
		name     string
		content  string
		wantDeny bool
	}{
		{"token then XXXX", token + "XXXX", true},
		{"token then TODO", token + "TODO", true},
		{"token then DUMMY", token + "DUMMY", true},
		{"token then SAMPLE", token + "SAMPLE", true},
		{"token then Example", token + "Example", true},
		{"token alone", token, true},
		{"placeholder token", "npm_" + strings.Repeat("X", 36), false},
		{"placeholder body then real tail", "npm_" + strings.Repeat("x", 32) + "Ab1c", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decision, reason := runScanSecretsWrite(t, python, hook, "main.go", "value := \""+tt.content+"\"\n")
			if denied := decision == "deny"; denied != tt.wantDeny {
				t.Errorf("denied = %v, want %v (reason %q)", denied, tt.wantDeny, reason)
			}
		})
	}
}

// TestScanSecretsHook_DeniesEveryCanonSample runs the real hook on a Write of
// every canon sample. It catches a canon entry Python re cannot compile:
// get_patterns would drop it with only an audit-log line, and the shape would
// go unscanned. The file is source code, so only DEFAULT_PATTERNS apply.
func TestScanSecretsHook_DeniesEveryCanonSample(t *testing.T) {
	t.Parallel()
	python, hook := scanSecretsHook(t)
	for _, vp := range secrets.ValuePatterns {
		for i, sample := range secretstest.ValuePatternSamples()[vp.Name] {
			t.Run(fmt.Sprintf("%s/%d", vp.Name, i), func(t *testing.T) {
				t.Parallel()
				decision, reason := runScanSecretsWrite(t, python, hook, "main.go", "value := \""+sample+"\"\n")
				if decision != "deny" {
					t.Fatalf("scan-secrets.py decision %q on canon %q sample %d, want deny", decision, vp.Name, i)
				}
				// The deny must come from this entry's own pattern, or a
				// dropped entry could hide behind an overlapping one.
				if !strings.Contains(reason, "Matched pattern: "+vp.Regex+" (") {
					t.Errorf("canon %q sample %d denied by another pattern: %s", vp.Name, i, reason)
				}
			})
		}
	}
}

// TestValuePatternSamples_NoPlaceholderIndicators pins that no canon sample
// contains a placeholder indicator: the hook compares indicators against the
// upper-cased match, so a sample such as "x"*24 (XXXX) would be skipped as a
// placeholder and the deny test above would prove nothing for it.
func TestValuePatternSamples_NoPlaceholderIndicators(t *testing.T) {
	t.Parallel()
	for name, samples := range secretstest.ValuePatternSamples() {
		for i, sample := range samples {
			upper := strings.ToUpper(sample)
			for _, ind := range claudecode.ExportPlaceholderIndicators {
				if strings.Contains(upper, ind) {
					t.Errorf("canon %q sample %d contains placeholder indicator %q", name, i, ind)
				}
			}
		}
	}
}
