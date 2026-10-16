package mcpregistry

import (
	"context"
	"testing"
)

// testStorePath is a well-formed Nix store path (32-char nix-base32 hash).
const testStorePath = "/nix/store/0123456789abcdfghijklmnpqrsvwxyz-server/bin/server"

// fakeProvenance returns a resolver that sees selfPath as both the running
// executable and the "qsdev" found on PATH, and treats every path as existing
// with no symlinks.
func fakeProvenance(t *testing.T, selfPath string) provenanceResolver {
	t.Helper()
	return provenanceResolver{
		lookPath:     func(string) (string, error) { return selfPath, nil },
		evalSymlinks: func(p string) (string, error) { return p, nil },
		executable:   func() (string, error) { return selfPath, nil },
	}
}

func TestGradeServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		def      McpServerDefinition
		expected ComplianceLevel
	}{
		{
			name: "qsdev single-module server grades Verified",
			def: McpServerDefinition{
				Command:   "qsdev",
				Args:      []string{"mcp", "serve", "--module", "agent-postmortem"},
				Transport: TransportStdio,
			},
			// no secrets, stdio, local-only, no auto-install, and "qsdev"
			// resolves to the running binary → Verified.
			expected: ComplianceVerified,
		},
		{
			name: "npx with -y grades Standard",
			def: McpServerDefinition{
				Command:   "npx",
				Args:      []string{"-y", "@upstash/context7-mcp"},
				Transport: TransportStdio,
			},
			// no secrets ✓, stdio ✓ → Standard; npx fetches → Secure ✗.
			expected: ComplianceStandard,
		},
		{
			name: "npx without -y no secrets grades Standard",
			def: McpServerDefinition{
				Command:   "npx",
				Args:      []string{"@anthropic-ai/mcp-github"},
				Transport: TransportStdio,
				Env:       map[string]string{"TOKEN": "${GITHUB_TOKEN}"},
			},
			expected: ComplianceStandard,
		},
		{
			name: "offline npx without pinned version grades Standard",
			def: McpServerDefinition{
				Command:   "npx",
				Args:      []string{"--offline", "@anthropic-ai/mcp-github"},
				Transport: TransportStdio,
			},
			// local-only ✓ (offline) but the unpinned package is auto-installed
			// from the cache → no-runtime-auto-install ✗.
			expected: ComplianceStandard,
		},
		{
			name: "offline npx with exact version grades Secure",
			def: McpServerDefinition{
				Command:   "npx",
				Args:      []string{"--offline", "@anthropic-ai/mcp-github@1.2.3"},
				Transport: TransportStdio,
			},
			expected: ComplianceSecure,
		},
		{
			name: "path-qualified npx grades Standard",
			def: McpServerDefinition{
				Command:   "/run/current-system/sw/bin/npx",
				Args:      []string{"-y", "pkg"},
				Transport: TransportStdio,
			},
			expected: ComplianceStandard,
		},
		{
			name: "plaintext secret in env grades Basic",
			def: McpServerDefinition{
				Command: "uvx",
				Env:     map[string]string{"API_KEY": "secret_test_xxxxxxxxxxxxxxxxxxxxxxxxx"},
			},
			expected: ComplianceBasic,
		},
		{
			name: "plaintext secret in args grades Basic",
			def: McpServerDefinition{
				Command:   "/usr/local/bin/server",
				Args:      []string{"--api-key", "sk-ant-api03-abcdefghijklmnopqrstuvwxyz"},
				Transport: TransportStdio,
			},
			expected: ComplianceBasic,
		},
		{
			name: "local binary no secrets grades Secure",
			def: McpServerDefinition{
				Command:   "/usr/local/bin/man-mcp-server",
				Transport: TransportStdio,
			},
			// provenance: /usr/local is not /nix/store → Verified ✗
			expected: ComplianceSecure,
		},
		{
			name: "nix store binary grades Verified",
			def: McpServerDefinition{
				Command:   testStorePath,
				Transport: TransportStdio,
			},
			expected: ComplianceVerified,
		},
		{
			name: "nix store traversal grades Secure",
			def: McpServerDefinition{
				Command:   "/nix/store/../../tmp/evil",
				Transport: TransportStdio,
			},
			expected: ComplianceSecure,
		},
		{
			name: "SSE transport caps at Basic",
			def: McpServerDefinition{
				Command:   "qsdev",
				Transport: TransportSSE,
			},
			expected: ComplianceBasic,
		},
		{
			name: "HTTP transport caps at Basic",
			def: McpServerDefinition{
				Command:   testStorePath,
				Transport: TransportHTTP,
			},
			expected: ComplianceBasic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := gradeServer(t.Context(), &tt.def, fakeProvenance(t, "/opt/qsdev/bin/qsdev"))
			if result.Level != tt.expected {
				t.Errorf("gradeServer().Level = %v (%d), want %v (%d)",
					result.Level, result.Level, tt.expected, tt.expected)
			}
		})
	}
}

func TestGradeServerAttestationLiftsVerifiedToAttested(t *testing.T) {
	// Not parallel: it mutates the package-level AttestationChecker.
	t.Cleanup(func() {
		AttestationChecker = func(context.Context, *McpServerDefinition) bool { return false }
	})

	// A definition that already satisfies every Verified criterion: stdio
	// transport, no plaintext secrets, local-only command with verified
	// provenance, and no runtime auto-install.
	def := &McpServerDefinition{
		Command:   "qsdev",
		Args:      []string{"mcp", "serve", "--module", "agent-postmortem"},
		Transport: TransportStdio,
	}
	prov := fakeProvenance(t, "/opt/qsdev/bin/qsdev")

	// With the default checker (false) the definition grades to Verified, one
	// below Attested.
	AttestationChecker = func(context.Context, *McpServerDefinition) bool { return false }
	if got := gradeServer(t.Context(), def, prov); got.Level != ComplianceVerified {
		t.Fatalf("default checker: Level = %v, want %v", got.Level, ComplianceVerified)
	}

	// With an injected checker returning true, the same definition reaches
	// Attested and the external-attestation criterion passes.
	AttestationChecker = func(context.Context, *McpServerDefinition) bool { return true }
	result := gradeServer(t.Context(), def, prov)
	if result.Level != ComplianceAttested {
		t.Errorf("attested checker: Level = %v, want %v", result.Level, ComplianceAttested)
	}

	var found bool
	for _, c := range result.Criteria {
		if c.Name == "external-attestation" {
			found = true
			if !c.Passed {
				t.Errorf("external-attestation criterion Passed = false, want true")
			}
		}
	}
	if !found {
		t.Error("external-attestation criterion not found in results")
	}
}

// TestGradeServerNpxReportsAutoInstall is the F268 regression: the old
// no-npx-dash-y criterion passed for `npx pkg`, although npx assumes --yes when
// stdin is not a TTY and so installs the package at launch exactly like -y.
func TestGradeServerNpxReportsAutoInstall(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"pkg"}, {"-y", "pkg"}} {
		def := &McpServerDefinition{Command: "npx", Args: args, Transport: TransportStdio}
		result := gradeServer(t.Context(), def, fakeProvenance(t, "/opt/qsdev/bin/qsdev"))
		var found bool
		for _, c := range result.Criteria {
			if c.Name == "no-runtime-auto-install" {
				found = true
				if c.Passed {
					t.Errorf("npx %v: no-runtime-auto-install passed, want failed", args)
				}
			}
		}
		if !found {
			t.Errorf("npx %v: no-runtime-auto-install criterion missing", args)
		}
	}
}

func TestGradeServerCriteriaPopulated(t *testing.T) {
	t.Parallel()

	def := &McpServerDefinition{
		Command:   "qsdev",
		Transport: TransportStdio,
	}
	result := GradeServer(t.Context(), def)

	if len(result.Criteria) == 0 {
		t.Fatal("GradeServer().Criteria is empty, expected criterion results")
	}

	// We expect criteria for each check: no-plaintext-secrets, stdio-transport,
	// local-only, no-runtime-auto-install, verified-provenance,
	// external-attestation.
	expectedNames := map[string]bool{
		"no-plaintext-secrets":    false,
		"stdio-transport":         false,
		"local-only":              false,
		"no-runtime-auto-install": false,
		"verified-provenance":     false,
		"external-attestation":    false,
	}

	for _, c := range result.Criteria {
		if _, ok := expectedNames[c.Name]; ok {
			expectedNames[c.Name] = true
		}
		if c.Detail == "" {
			t.Errorf("criterion %q has empty Detail", c.Name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected criterion %q not found in results", name)
		}
	}
}

func TestGradeServerDeterministic(t *testing.T) {
	t.Parallel()

	def := &McpServerDefinition{
		Command:   "npx",
		Args:      []string{"-y", "some-package"},
		Transport: TransportStdio,
		Env:       map[string]string{"FOO": "${BAR}"},
	}

	first := GradeServer(t.Context(), def)
	second := GradeServer(t.Context(), def)

	if first.Level != second.Level {
		t.Errorf("non-deterministic: first=%v, second=%v", first.Level, second.Level)
	}

	if len(first.Criteria) != len(second.Criteria) {
		t.Fatalf("non-deterministic criteria count: first=%d, second=%d",
			len(first.Criteria), len(second.Criteria))
	}

	for i := range first.Criteria {
		if first.Criteria[i] != second.Criteria[i] {
			t.Errorf("non-deterministic criterion[%d]: first=%+v, second=%+v",
				i, first.Criteria[i], second.Criteria[i])
		}
	}
}

// TestGradeConfigured_CommandlessEntryIsNotStdio pins that a remote
// .mcp.json entry (no command) is not graded as a local stdio server.
func TestGradeConfigured_CommandlessEntryIsNotStdio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		def       McpServerDefinition
		wantStdio bool
	}{
		{name: "remote entry parsed as stdio", def: McpServerDefinition{Name: "socket", Transport: TransportStdio}},
		{name: "http entry", def: McpServerDefinition{Name: "socket", URL: "https://mcp.example/", Transport: TransportHTTP}},
		{name: "local stdio entry", def: McpServerDefinition{Name: "vs", Command: "qsdev", Args: []string{"mcp", "serve", "--module", "version-sentinel"}, Transport: TransportStdio}, wantStdio: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			def := configuredDefinition(tt.def)
			if got := def.Transport == TransportStdio; got != tt.wantStdio {
				t.Errorf("transport = %q, want stdio=%v", def.Transport, tt.wantStdio)
			}
			if got := criterionPassed(t, GradeConfigured(t.Context(), tt.def), "stdio-transport"); got != tt.wantStdio {
				t.Errorf("GradeConfigured stdio-transport passed = %v, want %v", got, tt.wantStdio)
			}
		})
	}
}

// criterionPassed returns whether gr's criterion name passed, failing the
// test when gr has no such criterion.
func criterionPassed(t *testing.T, gr GradeResult, name string) bool {
	t.Helper()
	for _, c := range gr.Criteria {
		if c.Name == name {
			return c.Passed
		}
	}
	t.Fatalf("grade has no %q criterion: %+v", name, gr.Criteria)
	return false
}

func TestSuggestsPinnedInstall(t *testing.T) {
	t.Parallel()

	pinned := &McpServerDefinition{Name: "context7", Command: "npx", Bin: "context7-mcp", Version: "4.1.1"}
	tests := []struct {
		name       string
		known      *McpServerDefinition
		configured *McpServerDefinition
		want       bool
	}{
		{name: "package launcher", known: pinned, configured: &McpServerDefinition{Command: "npx"}, want: true},
		{name: "installed binary", known: pinned, configured: &McpServerDefinition{Command: "context7-mcp"}},
		{name: "installed binary shim", known: pinned, configured: &McpServerDefinition{Command: `C:\npm\context7-mcp.cmd`}},
		{name: "no binary", known: &McpServerDefinition{Command: "qsdev", Version: "1"}, configured: &McpServerDefinition{Command: "qsdev"}},
		{name: "no pinned version", known: &McpServerDefinition{Command: "npx", Bin: "context7-mcp"}, configured: &McpServerDefinition{Command: "npx"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SuggestsPinnedInstall(tt.known, tt.configured); got != tt.want {
				t.Errorf("SuggestsPinnedInstall = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGradeServer_ThreadsContext pins U16-16: the attestation checker runs
// under the caller's context (so a cancelled `mcp grade` stops verifying
// signatures), not a context.Background() of its own.
func TestGradeServer_ThreadsContext(t *testing.T) {
	// Not parallel: it mutates the package-level AttestationChecker.
	t.Cleanup(func() {
		AttestationChecker = func(context.Context, *McpServerDefinition) bool { return false }
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var got context.Context
	AttestationChecker = func(c context.Context, _ *McpServerDefinition) bool {
		got = c
		return false
	}
	def := &McpServerDefinition{Command: "qsdev", Args: []string{"mcp", "serve"}, Transport: TransportStdio}
	gradeServer(ctx, def, fakeProvenance(t, "/opt/qsdev/bin/qsdev"))

	if got == nil {
		t.Fatal("AttestationChecker was not called for a Verified server")
	}
	if got.Err() == nil {
		t.Error("AttestationChecker context is not the caller's cancelled context")
	}
}
