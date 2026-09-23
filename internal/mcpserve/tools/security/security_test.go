package security

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan/vulnscantest"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// writeConfig writes a .qsdev.yaml fixture into dir and returns its path.
func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, ".qsdev.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

// structuredMap unwraps a result's structured payload as a map.
func structuredMap(t *testing.T, res *spi.ToolResult) map[string]any {
	t.Helper()
	if res == nil {
		t.Fatal("nil result")
	}
	m, ok := res.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured is %T, want map[string]any", res.Structured)
	}
	return m
}

func call(t *testing.T, h spi.ToolHandler, args map[string]any) *spi.ToolResult {
	t.Helper()
	res, err := h(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{Arguments: args})
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	return res
}

const policyFixture = `version: 1
claude_code:
  permission_level: minimal
tools:
  enabled:
    - semgrep
  disabled:
    - dangerous_tool
`

func TestPolicyCheckEvaluatesDenyRule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir, nil)

	t.Run("denied tool", func(t *testing.T) {
		// A denied verdict is a normal evaluation, not a tool error: IsError stays
		// false (only graceful-degradation results set it).
		res := call(t, pc.handle, map[string]any{"tool_name": "dangerous_tool"})
		if res.IsError {
			t.Errorf("a normal deny verdict must not set IsError")
		}
		eval := structuredMap(t, res)["evaluation"].(policyDecision)
		if eval.Decision != decisionDenied {
			t.Errorf("decision = %q, want denied", eval.Decision)
		}
		if eval.Source != sourceProject || eval.Rule != "tools.disabled" {
			t.Errorf("source/rule = %q/%q, want project/tools.disabled", eval.Source, eval.Rule)
		}
	})

	t.Run("allowed tool", func(t *testing.T) {
		eval := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "semgrep"}))["evaluation"].(policyDecision)
		if eval.Decision != decisionAllowed || eval.Source != sourceProject {
			t.Errorf("got %+v, want allowed/project", eval)
		}
	})

	t.Run("unknown tool falls through to default", func(t *testing.T) {
		eval := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "mystery"}))["evaluation"].(policyDecision)
		// permission_level: minimal (a plan-mode preset) escalates the default to "ask".
		if eval.Decision != decisionAsk || eval.Source != sourceDefault {
			t.Errorf("got %+v, want ask/default", eval)
		}
	})
}

// TestPolicyCheckReportsEnforcedDenySet proves qsdev_policy_check reports the MCP
// Guardrail's enforced deny set from the Policy object the running server
// installed (via projectPolicy/chainForMode), so a tool "reported denied" and a
// tool "actually blocked on an MCP call" cannot diverge (BL-P1-3, S7).
func TestPolicyCheckReportsEnforcedDenySet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, "version: 1\ntools:\n  disabled:\n    - qsdev_security_scan\n    - qsdev_nix_run\n")
	cfg, err := config.ParseQsdevConfig(filepath.Join(dir, ".qsdev.yaml"))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	installed := middleware.PolicyFromConfig(cfg)
	pc := newPolicyChecker(dir, installed)

	// Inventory mode (no tool_name) surfaces the enforced deny set.
	m := structuredMap(t, call(t, pc.handle, nil))
	reported, ok := m["mcp_enforced_deny"].([]string)
	if !ok {
		t.Fatalf("mcp_enforced_deny missing or wrong type")
	}
	if want := installed.DenyToolSet(); !reflect.DeepEqual(reported, want) {
		t.Errorf("reported enforced deny = %v, want %v (reported must equal enforced)", reported, want)
	}
	// Guard against a vacuous pass where both sides are empty.
	if len(reported) != 2 {
		t.Fatalf("enforced deny set = %v, want the 2 disabled tools", reported)
	}
	if w := m["warnings"]; w != nil {
		t.Errorf("file and enforced policy agree; want no warnings, got %v", w)
	}
}

// TestPolicyCheckEnforcedDenyIgnoresFileDrift is the regression test for
// reporting a deny set the server does not enforce: after .qsdev.yaml is edited
// mid-session (the Guardrail keeps its startup snapshot until restart), or when
// policy_path points at another file, mcp_enforced_deny must still be the
// installed policy's set, and the gap must be surfaced as a warning.
func TestPolicyCheckEnforcedDenyIgnoresFileDrift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Server started with nothing disabled: the installed Guardrail policy is nil.
	pc := newPolicyChecker(dir, nil)
	// The config now disables a tool (edited after startup).
	writeConfig(t, dir, "version: 1\ntools:\n  disabled:\n    - qsdev_credential_vend\n")
	other := filepath.Join(dir, "other.yaml")
	if err := os.WriteFile(other, []byte("version: 1\ntools:\n  disabled:\n    - qsdev_nix_run\n"), 0o644); err != nil {
		t.Fatalf("write other policy: %v", err)
	}

	tests := []struct {
		name     string
		args     map[string]any
		wantWarn string
	}{
		{"edited default config", map[string]any{}, "restart"},
		{"non-default policy_path", map[string]any{"policy_path": "other.yaml"}, "hypothetical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := structuredMap(t, call(t, pc.handle, tt.args))
			if got, _ := m["mcp_enforced_deny"].([]string); len(got) != 0 {
				t.Errorf("mcp_enforced_deny = %v, want empty (nothing is enforced until restart)", got)
			}
			warnings, _ := m["warnings"].([]string)
			if len(warnings) == 0 || !strings.Contains(warnings[0], tt.wantWarn) {
				t.Errorf("warnings = %v, want one mentioning %q", warnings, tt.wantWarn)
			}
		})
	}
}

// TestPolicyCheckOverlayMatchesCanonicalResolver is the regression test for
// policy_check's private overlay cascade: a project deny must win over a local
// enable (union semantics, as the Guardrail enforces), and a local security.level
// must not lower the project floor when deriving the default decision.
func TestPolicyCheckOverlayMatchesCanonicalResolver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		project        string
		local          string
		tool           string
		wantDecision   string
		wantSource     string
		wantViolations bool
	}{
		{
			name:         "project deny beats local enable",
			project:      "version: 1\ntools:\n  disabled:\n    - semgrep\n",
			local:        "tools:\n  enabled:\n    - semgrep\n",
			tool:         "semgrep",
			wantDecision: decisionDenied,
			wantSource:   sourceProject,
		},
		{
			name:         "local deny beats project enable",
			project:      "version: 1\ntools:\n  enabled:\n    - semgrep\n",
			local:        "tools:\n  disabled:\n    - semgrep\n",
			tool:         "semgrep",
			wantDecision: decisionDenied,
			wantSource:   sourceLocal,
		},
		{
			name:           "local security level cannot lower the floor",
			project:        "version: 1\nsecurity:\n  level: strict\n",
			local:          "security:\n  level: baseline\n",
			tool:           "unlisted",
			wantDecision:   decisionAsk,
			wantSource:     sourceDefault,
			wantViolations: true,
		},
		{
			name:         "local security level may raise it",
			project:      "version: 1\nsecurity:\n  level: baseline\n",
			local:        "security:\n  level: strict\n",
			tool:         "unlisted",
			wantDecision: decisionAsk,
			wantSource:   sourceDefault,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeConfig(t, dir, tt.project)
			if err := os.WriteFile(filepath.Join(dir, branding.Get().LocalConfig), []byte(tt.local), 0o644); err != nil {
				t.Fatalf("write local overlay: %v", err)
			}
			m := structuredMap(t, call(t, newPolicyChecker(dir, nil).handle, map[string]any{"tool_name": tt.tool}))
			eval := m["evaluation"].(policyDecision)
			if eval.Decision != tt.wantDecision || eval.Source != tt.wantSource {
				t.Errorf("evaluation = %+v, want %s/%s", eval, tt.wantDecision, tt.wantSource)
			}
			violations, _ := m["floor_violations"].([]floorViolation)
			levelViolation := slices.ContainsFunc(violations, func(v floorViolation) bool { return v.Field == "security.level" })
			if levelViolation != tt.wantViolations {
				t.Errorf("security.level floor violation = %v, want %v (%v)", levelViolation, tt.wantViolations, violations)
			}
		})
	}
}

// TestDefaultDecisionForRealVocabulary is the regression test for a switch on
// permission/security levels that do not exist: every valid preset and security
// level must map to the intended default, in particular the restrictive
// plan-mode "minimal" preset and the "strict" security level escalate to ask.
func TestDefaultDecisionForRealVocabulary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		permission string
		security   string
		want       string
	}{
		{"", "", decisionAllowed},
		{"minimal", "", decisionAsk},
		{"standard", "", decisionAllowed},
		{"permissive", "", decisionAllowed},
		{"custom", "", decisionAllowed},
		{"supply-chain-only", "", decisionAllowed},
		{"", "baseline", decisionAllowed},
		{"", "enhanced", decisionAllowed},
		{"", "strict", decisionAsk},
		{"standard", "strict", decisionAsk},
		{"minimal", "baseline", decisionAsk},
	}
	for _, tt := range tests {
		t.Run(tt.permission+"/"+tt.security, func(t *testing.T) {
			t.Parallel()
			cfg := &types.QsdevConfig{}
			cfg.ClaudeCode.PermissionLevel = tt.permission
			cfg.Security.Level = tt.security
			if got := defaultDecisionFor(cfg); got != tt.want {
				t.Errorf("defaultDecisionFor(%q, %q) = %q, want %q", tt.permission, tt.security, got, tt.want)
			}
		})
	}
}

func TestPolicyCheckNotConfigured(t *testing.T) {
	t.Parallel()
	pc := newPolicyChecker(t.TempDir(), nil) // no .qsdev.yaml present
	res := call(t, pc.handle, map[string]any{"tool_name": "anything"})
	if !res.IsError {
		t.Fatal("expected IsError for missing policy")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}

// TestPolicyCheckWarnsOnMalformedLocalOverlay proves a malformed .qsdev.local.yaml
// overlay does not fail the evaluation (it still runs on the project policy) but
// surfaces a warning — the overlay is the highest-precedence layer, so silently
// dropping its denies would be a fail-open.
func TestPolicyCheckWarnsOnMalformedLocalOverlay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	// Unterminated flow sequence: invalid YAML, so ParseLocalConfig errors.
	localPath := filepath.Join(dir, branding.Get().LocalConfig)
	if err := os.WriteFile(localPath, []byte("tools:\n  enabled: [a, b\n"), 0o644); err != nil {
		t.Fatalf("write local overlay: %v", err)
	}
	pc := newPolicyChecker(dir, nil)

	res := call(t, pc.handle, map[string]any{"tool_name": "semgrep"})
	if res.IsError {
		t.Errorf("malformed overlay must degrade, not fail: %+v", res.Structured)
	}
	warnings, ok := structuredMap(t, res)["warnings"].([]string)
	if !ok || len(warnings) == 0 {
		t.Fatalf("expected a warnings entry for the dropped overlay, got %v", structuredMap(t, res)["warnings"])
	}
	if !strings.Contains(warnings[0], branding.Get().LocalConfig) {
		t.Errorf("warning should name the overlay file, got %q", warnings[0])
	}
}

// TestSecurityScanReportsUnparseableLockFile proves a present-but-corrupt lock
// file is reported distinctly from "no lock file", so a caller cannot misread a
// failed scan as a clean (zero-vulnerability) result.
func TestSecurityScanReportsUnparseableLockFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{ this is not valid json"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	scanner := newSecurityScanner(dir)

	res := call(t, scanner.handle, map[string]any{"manifest_path": "package-lock.json"})
	if !res.IsError {
		t.Fatal("expected IsError for an unparseable lock file")
	}
	m := structuredMap(t, res)
	if m["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", m["status"])
	}
	if reason, _ := m["reason"].(string); !strings.Contains(reason, "could not be parsed") {
		t.Errorf("reason should distinguish a present-but-unparseable lock file, got %q", reason)
	}
}

// TestSecurityScanSeverityMatchesSharedMapping is the M11 regression: the MCP
// security_scan tool must report the SAME normalized severity the CLI scan does.
// Before consolidation, a GHSA "MODERATE" advisory folded to "unknown" via the
// tool's private table (which only knew "medium"), while the CLI reported
// "moderate" — the two entry points disagreed on the same advisory. Both the
// "moderate" and the "medium" (alias) thresholds must admit it.
func TestSecurityScanSeverityMatchesSharedMapping(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goSum := "example.com/mod v1.0.0 h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\n" +
		"example.com/mod v1.0.0/go.mod h1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb=\n"
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(goSum), 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}

	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-MOD-1"}},
		map[string]string{"GHSA-MOD-1": "MODERATE"},
	)
	scanner := &securityScanner{
		projectRoot: dir,
		scanner:     &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()},
	}

	for _, threshold := range []string{"moderate", "medium"} {
		t.Run("threshold_"+threshold, func(t *testing.T) {
			res := call(t, scanner.handle, map[string]any{"severity_threshold": threshold})
			if res.IsError {
				t.Fatalf("scan returned error: %+v", res.Structured)
			}
			vulns, ok := structuredMap(t, res)["vulnerabilities"].([]vulnReport)
			if !ok || len(vulns) != 1 {
				t.Fatalf("vulnerabilities = %v, want exactly one at/above %q", structuredMap(t, res)["vulnerabilities"], threshold)
			}
			if vulns[0].Severity != "moderate" {
				t.Errorf("severity = %q, want \"moderate\" (must match the shared CLI mapping, not fold to unknown)", vulns[0].Severity)
			}
		})
	}
}

// TestPolicyCheckRejectsPathTraversal proves the policy_path argument is confined
// to the project root: a relative or absolute path that escapes the root degrades
// to a structured not_configured result (rather than reading an arbitrary host
// file), while an in-root path still loads normally.
func TestPolicyCheckRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir, nil)

	cases := []struct {
		name       string
		policyPath string
		wantError  bool
	}{
		{name: "relative escape", policyPath: "../../../../etc/passwd", wantError: true},
		{name: "absolute outside root", policyPath: "/etc/passwd", wantError: true},
		{name: "sneaky middle escape", policyPath: "sub/../../outside.yaml", wantError: true},
		{name: "in-root relative path", policyPath: ".qsdev.yaml", wantError: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := call(t, pc.handle, map[string]any{
				"tool_name":   "semgrep",
				"policy_path": tc.policyPath,
			})
			if tc.wantError {
				if !res.IsError {
					t.Fatalf("expected IsError for path %q", tc.policyPath)
				}
				if got := structuredMap(t, res)["status"]; got != "not_configured" {
					t.Errorf("status = %v, want not_configured", got)
				}
				return
			}
			if res.IsError {
				t.Fatalf("in-root path %q should be allowed, got error: %+v", tc.policyPath, res.Structured)
			}
		})
	}
}

// TestPolicyCheckFastPath proves the fast-path cache directly: once the policy is
// parsed, a subsequent call with the file's mtime unchanged returns the cached
// verdict WITHOUT re-parsing the YAML (the property that keeps repeated calls
// cheap), and a call after the mtime advances re-parses and reflects the new
// policy.
//
// This asserts the caching mechanism rather than wall-clock latency: a
// millisecond budget measured under parallel `go test ./...` load is dominated by
// scheduler and GC jitter on shared CI hosts, so it flakes without catching the
// real regression — a broken cache that re-parses on every call, which the
// stale-content check below catches deterministically.
func TestPolicyCheckFastPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir, nil)

	// Warm the cache: semgrep is enabled in the fixture, so the verdict is allowed.
	warm := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "semgrep"}))["evaluation"].(policyDecision)
	if warm.Decision != decisionAllowed {
		t.Fatalf("warm-up decision = %q, want allowed", warm.Decision)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat policy: %v", err)
	}
	orig := info.ModTime()

	// Rewrite the policy so semgrep would now evaluate to denied, but restore the
	// original mtime. A cache hit (fast path) must skip the re-parse and still
	// return the cached allowed verdict; a re-parse would surface the new deny.
	flipped := "version: 1\ntools:\n  disabled:\n    - semgrep\n"
	if err := os.WriteFile(path, []byte(flipped), 0o644); err != nil {
		t.Fatalf("rewrite policy: %v", err)
	}
	if err := os.Chtimes(path, orig, orig); err != nil {
		t.Fatalf("reset mtime: %v", err)
	}
	cached := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "semgrep"}))["evaluation"].(policyDecision)
	if cached.Decision != decisionAllowed {
		t.Errorf("cached decision = %q, want allowed (fast path must not re-parse while mtime is unchanged)", cached.Decision)
	}

	// Advance the mtime: the cache must invalidate and re-parse, now returning denied.
	changed := orig.Add(2 * time.Second)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatalf("advance mtime: %v", err)
	}
	fresh := structuredMap(t, call(t, pc.handle, map[string]any{"tool_name": "semgrep"}))["evaluation"].(policyDecision)
	if fresh.Decision != decisionDenied {
		t.Errorf("post-invalidation decision = %q, want denied (mtime change must trigger re-parse)", fresh.Decision)
	}
}

func TestSecurityScanNotConfiguredWithoutLockFile(t *testing.T) {
	t.Parallel()
	scanner := newSecurityScanner(t.TempDir()) // empty dir, no lock file
	res := call(t, scanner.handle, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when no lock file found")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}

// TestSecurityScanRejectsPathTraversal proves the manifest_path argument is
// confined to the project root: a relative or absolute path that escapes the
// root degrades to a structured not_configured result (rather than reading an
// out-of-tree lock file), while an in-root path still scans. The OSV endpoint is
// stubbed so the in-root happy path completes offline.
func TestSecurityScanRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("requests==2.31.0\n"), 0o644); err != nil {
		t.Fatalf("write requirements.txt: %v", err)
	}
	// outsideAbs is an absolute path in a sibling temp dir — outside the project
	// root on every platform. A literal "/etc/passwd" is not portable here:
	// filepath.IsAbs treats it as rooted-but-relative on Windows, so it would be
	// joined under the root rather than rejected as an escape.
	outsideAbs := filepath.Join(t.TempDir(), "go.sum")

	// Stub OSV so the in-root scan succeeds without network access: an empty
	// response reports zero vulnerabilities. The stub closes via t.Cleanup (not
	// defer): the parallel subtests below run after this function returns, so a
	// defer would close it before they make their request.
	srv := vulnscantest.NewServer(t, nil, nil)
	scanner := &securityScanner{
		projectRoot: dir,
		scanner:     &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()},
	}

	cases := []struct {
		name      string
		manifest  string
		wantError bool
	}{
		{name: "relative escape", manifest: "../../../../etc/passwd", wantError: true},
		{name: "absolute outside root", manifest: outsideAbs, wantError: true},
		{name: "sneaky middle escape", manifest: "sub/../../outside/requirements.txt", wantError: true},
		{name: "in-root relative path", manifest: "requirements.txt", wantError: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := call(t, scanner.handle, map[string]any{"manifest_path": tc.manifest})
			if tc.wantError {
				if !res.IsError {
					t.Fatalf("expected IsError for escaping path %q", tc.manifest)
				}
				m := structuredMap(t, res)
				if m["status"] != "not_configured" {
					t.Errorf("status = %v, want not_configured", m["status"])
				}
				if m["reason"] != "manifest_path escapes the project root" {
					t.Errorf("reason = %v, want manifest_path escape reason", m["reason"])
				}
				return
			}
			if res.IsError {
				t.Fatalf("in-root path %q should scan, got error: %+v", tc.manifest, res.Structured)
			}
		})
	}
}

// TestAWSCredsResultNilCredentials proves the AWS vend path degrades to an error
// result instead of panicking when STS returns a nil Credentials block. The
// populated case uses secret-shaped values assembled by concatenation so the
// ripsecrets scanner does not flag the fixture.
func TestAWSCredsResultNilCredentials(t *testing.T) {
	t.Parallel()

	t.Run("nil credentials degrades to error", func(t *testing.T) {
		t.Parallel()
		res := awsCredsResult(nil)
		if res == nil || !res.IsError {
			t.Fatalf("expected error result, got %+v", res)
		}
		if got := structuredMap(t, res)["status"]; got != "error" {
			t.Errorf("status = %v, want error", got)
		}
	})

	t.Run("populated credentials returns success", func(t *testing.T) {
		t.Parallel()
		exp := time.Now().Add(time.Hour)
		creds := &ststypes.Credentials{
			AccessKeyId:     awssdk.String("AKIA" + "EXAMPLEFIXTURE"),
			SecretAccessKey: awssdk.String("not" + "-a-real-secret"),
			SessionToken:    awssdk.String("session" + "-fixture-token"),
			Expiration:      &exp,
		}
		res := awsCredsResult(creds)
		if res.IsError {
			t.Fatalf("unexpected error result: %+v", res)
		}
		m := structuredMap(t, res)
		if m["access_key_id"] != "AKIA"+"EXAMPLEFIXTURE" {
			t.Errorf("access_key_id = %v", m["access_key_id"])
		}
		if m["expiration"] != exp.UTC().Format(time.RFC3339) {
			t.Errorf("expiration = %v", m["expiration"])
		}
	})
}

func TestCredentialVendNotConfigured(t *testing.T) {
	t.Parallel()
	cv := newCredentialVendor()

	t.Run("missing provider", func(t *testing.T) {
		res := call(t, cv.handle, map[string]any{})
		if !res.IsError || structuredMap(t, res)["status"] != "not_configured" {
			t.Errorf("got IsError=%t status=%v, want not_configured", res.IsError, structuredMap(t, res)["status"])
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		res := call(t, cv.handle, map[string]any{"provider": "digitalocean"})
		if !res.IsError || structuredMap(t, res)["status"] != "not_configured" {
			t.Errorf("got IsError=%t status=%v, want not_configured", res.IsError, structuredMap(t, res)["status"])
		}
	})
}

// TestCredentialVendAWSNoCredentials proves the AWS path degrades to a structured
// not_configured result (rather than crashing) when no credentials can be
// resolved. IMDS is disabled and every credential source is pointed at a
// nonexistent file so the test is fully offline and deterministic.
func TestCredentialVendAWSNoCredentials(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", missing)
	t.Setenv("AWS_CONFIG_FILE", missing)

	cv := newCredentialVendor()
	res := call(t, cv.handle, map[string]any{"provider": "aws"})
	if !res.IsError {
		t.Fatal("expected IsError when AWS has no credentials")
	}
	if structuredMap(t, res)["status"] != "not_configured" {
		t.Errorf("status = %v, want not_configured", structuredMap(t, res)["status"])
	}
}

// writePolyglotProject writes a go.sum and a package-lock.json, one pinned
// dependency each, into a fresh directory.
func writePolyglotProject(t *testing.T, npmLock string) string {
	t.Helper()
	dir := t.TempDir()
	goSum := "example.com/mod v1.0.0 h1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=\n"
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(goSum), 0o644); err != nil {
		t.Fatalf("write go.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(npmLock), 0o644); err != nil {
		t.Fatalf("write package-lock.json: %v", err)
	}
	return dir
}

// TestSecurityScanCoversEveryEcosystem is the regression test for polyglot
// repos: with both go.sum and package-lock.json present, the auto-detected scan
// must query both ecosystems and report a vulnerability in either, rather than
// scanning only the first lock file and returning a clean-looking result.
func TestSecurityScanCoversEveryEcosystem(t *testing.T) {
	t.Parallel()
	dir := writePolyglotProject(t,
		`{"lockfileVersion":3,"packages":{"":{"name":"app"},"node_modules/left-pad":{"version":"1.3.0"}}}`)

	srv := vulnscantest.NewServer(t,
		map[int][]string{0: {"GHSA-VULN-0"}, 1: {"GHSA-VULN-1"}},
		map[string]string{"GHSA-VULN-0": "HIGH", "GHSA-VULN-1": "HIGH"},
	)
	scanner := &securityScanner{
		projectRoot: dir,
		scanner:     &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()},
	}

	res := call(t, scanner.handle, map[string]any{})
	if res.IsError {
		t.Fatalf("scan returned error: %+v", res.Structured)
	}
	m := structuredMap(t, res)
	if got := m["ecosystems"]; !reflect.DeepEqual(got, []string{"Go", "npm"}) {
		t.Errorf("ecosystems = %v, want [Go npm]", got)
	}
	if lockFiles, _ := m["lock_files"].([]string); len(lockFiles) != 2 {
		t.Errorf("lock_files = %v, want both lock files", m["lock_files"])
	}
	if m["coverage"] != "complete" {
		t.Errorf("coverage = %v, want complete", m["coverage"])
	}
	vulns, _ := m["vulnerabilities"].([]vulnReport)
	ecos := map[string]bool{}
	for _, v := range vulns {
		ecos[v.Ecosystem] = true
	}
	if !ecos["Go"] || !ecos["npm"] {
		t.Errorf("vulnerabilities = %+v, want findings from both Go and npm", vulns)
	}
}

// TestSecurityScanReportsPartialCoverage verifies that when one of several lock
// files cannot be parsed, the others are still scanned and the result says the
// coverage is partial instead of implying a complete scan.
func TestSecurityScanReportsPartialCoverage(t *testing.T) {
	t.Parallel()
	dir := writePolyglotProject(t, "{ not valid json")

	srv := vulnscantest.NewServer(t, nil, nil)
	scanner := &securityScanner{
		projectRoot: dir,
		scanner:     &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()},
	}

	res := call(t, scanner.handle, map[string]any{})
	if res.IsError {
		t.Fatalf("scan returned error: %+v", res.Structured)
	}
	m := structuredMap(t, res)
	if m["coverage"] != "partial" {
		t.Errorf("coverage = %v, want partial", m["coverage"])
	}
	unscanned, _ := m["unscanned_lock_files"].([]unscannedLockFile)
	if len(unscanned) != 1 || filepath.Base(unscanned[0].Path) != "package-lock.json" {
		t.Errorf("unscanned_lock_files = %+v, want the broken package-lock.json", m["unscanned_lock_files"])
	}
	if !strings.Contains(res.Text, "PARTIAL coverage") {
		t.Errorf("text = %q, want it to flag partial coverage", res.Text)
	}
}

// TestCredentialVendGCPRejectsInvalidServiceAccount is the regression test for
// the unvalidated IAM Credentials URL path: a service_account carrying URL
// syntax must be rejected before any credential lookup or request, so it cannot
// retarget the ADC-authorized call to a different IAM method. Valid emails and
// numeric unique IDs are accepted.
func TestCredentialVendGCPRejectsInvalidServiceAccount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		sa    string
		valid bool
	}{
		{"user-managed email", "deployer@my-project.iam.gserviceaccount.com", true},
		{"compute default email", "123456789012-compute@developer.gserviceaccount.com", true},
		{"appspot email", "my-project@appspot.gserviceaccount.com", true},
		{"numeric unique id", "112233445566778899001", true},
		{"method suffix and fragment", "x@y.iam.gserviceaccount.com:signJwt#", false},
		{"query string", "x@y.iam.gserviceaccount.com:signJwt?alt=json", false},
		{"path traversal", "../../projects/p/serviceAccounts/x@y.iam.gserviceaccount.com", false},
		{"slash in local part", "a/b@y.iam.gserviceaccount.com", false},
		{"not an email", "deployer", false},
		{"whitespace", "x@y.iam.gserviceaccount.com ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gcpServiceAccountPattern.MatchString(tt.sa); got != tt.valid {
				t.Errorf("gcpServiceAccountPattern.MatchString(%q) = %v, want %v", tt.sa, got, tt.valid)
			}
		})
	}

	cv := newCredentialVendor()
	res := call(t, cv.handle, map[string]any{"provider": "gcp", "service_account": "x@y.iam.gserviceaccount.com:signJwt#"})
	m := structuredMap(t, res)
	if !res.IsError || m["status"] != "not_configured" {
		t.Fatalf("got IsError=%t status=%v, want not_configured", res.IsError, m["status"])
	}
	if reason, _ := m["reason"].(string); !strings.Contains(reason, "invalid service_account") {
		t.Errorf("reason = %q, want the invalid service_account rejection", reason)
	}
}
