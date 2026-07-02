package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
  permission_level: strict
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
	pc := newPolicyChecker(dir)

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
		// permission_level: strict escalates the default to "ask".
		if eval.Decision != decisionAsk || eval.Source != sourceDefault {
			t.Errorf("got %+v, want ask/default", eval)
		}
	})
}

func TestPolicyCheckNotConfigured(t *testing.T) {
	t.Parallel()
	pc := newPolicyChecker(t.TempDir()) // no .qsdev.yaml present
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
	pc := newPolicyChecker(dir)

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

// TestPolicyCheckRejectsPathTraversal proves the policy_path argument is confined
// to the project root: a relative or absolute path that escapes the root degrades
// to a structured not_configured result (rather than reading an arbitrary host
// file), while an in-root path still loads normally.
func TestPolicyCheckRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeConfig(t, dir, policyFixture)
	pc := newPolicyChecker(dir)

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
	pc := newPolicyChecker(dir)

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
	// response reports zero vulnerabilities.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(osvBatchResponse{})
	}))
	// t.Cleanup (not defer): the parallel subtests below run after this function
	// returns, so a defer would close the stub before they make their request.
	t.Cleanup(srv.Close)
	scanner := &securityScanner{projectRoot: dir, baseURL: srv.URL, httpClient: srv.Client()}

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

// TestSecurityScanFetchDetailsDeterministic proves that when more than
// maxVulnDetailFetches unique vulnerabilities are present, the subset whose
// details are fetched is deterministic (the lexicographically smallest ids) and
// stable across repeated runs, so the downstream severity-floor filtering does
// not vary run-to-run.
func TestSecurityScanFetchDetailsDeterministic(t *testing.T) {
	t.Parallel()

	const total = maxVulnDetailFetches + 50
	oneQuery := make([]string, total)
	for i := range oneQuery {
		oneQuery[i] = fmt.Sprintf("VULN-%04d", i)
	}
	idsByQuery := [][]string{oneQuery}

	var mu sync.Mutex
	requested := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/vulns/")
		mu.Lock()
		requested[id]++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(osvVuln{ID: id})
	}))
	defer srv.Close()

	s := &securityScanner{baseURL: srv.URL, httpClient: srv.Client()}

	first := s.fetchDetails(context.Background(), idsByQuery)
	if len(first) != maxVulnDetailFetches {
		t.Fatalf("fetched %d details, want %d", len(first), maxVulnDetailFetches)
	}

	// The fetched subset must be exactly the lexicographically smallest ids.
	want := append([]string(nil), oneQuery...)
	sort.Strings(want)
	want = want[:maxVulnDetailFetches]
	for _, id := range want {
		if _, ok := first[id]; !ok {
			t.Fatalf("expected smallest id %q to be fetched", id)
		}
	}

	second := s.fetchDetails(context.Background(), idsByQuery)
	if len(second) != len(first) {
		t.Fatalf("second fetch size %d != first %d", len(second), len(first))
	}
	for id := range first {
		if _, ok := second[id]; !ok {
			t.Errorf("nondeterministic selection: id %q fetched first run but not second", id)
		}
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
