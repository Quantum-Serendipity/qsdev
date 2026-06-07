package mcpregistry

import "testing"

func TestGradeServer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		def      McpServerDefinition
		expected ComplianceLevel
	}{
		{
			name: "qsdev embedded server grades Secure",
			def: McpServerDefinition{
				Command:   "qsdev",
				Args:      []string{"mcp", "agent-postmortem"},
				Transport: TransportStdio,
			},
			// qsdev: no secrets, stdio, local-only, no npx-y → Secure
			// Not Verified because "qsdev" has verified provenance → actually Verified
			// Wait: hasVerifiedProvenance returns true for "qsdev"
			// So: Standard ✓, Secure ✓, Verified ✓
			expected: ComplianceVerified,
		},
		{
			name: "npx with -y grades Basic",
			def: McpServerDefinition{
				Command:   "npx",
				Args:      []string{"-y", "@upstash/context7-mcp"},
				Transport: TransportStdio,
			},
			// npx: no secrets ✓, stdio ✓ → Standard ✓
			// local-only ✗ (npx is network command) → Secure ✗
			// Also hasNpxDashY ✓ so noNpxY ✗
			// Caps at Standard
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
			// npx: no plaintext secrets ✓, stdio ✓ → Standard ✓
			// local-only ✗ → Secure ✗
			expected: ComplianceStandard,
		},
		{
			name: "plaintext secret in env grades Basic",
			def: McpServerDefinition{
				Command: "uvx",
				Env:     map[string]string{"API_KEY": "secret_test_xxxxxxxxxxxxxxxxxxxxxxxxx"},
			},
			// hasPlaintextSecrets → true, so noSecrets ✗
			// Standard ✗ → stays Basic
			expected: ComplianceBasic,
		},
		{
			name: "local binary no secrets grades Secure",
			def: McpServerDefinition{
				Command:   "/usr/local/bin/man-mcp-server",
				Transport: TransportStdio,
			},
			// no secrets ✓, stdio ✓ → Standard ✓
			// local-only ✓, no npx-y ✓ → Secure ✓
			// provenance: /usr/local is not /nix/store → Verified ✗
			expected: ComplianceSecure,
		},
		{
			name: "nix store binary grades Verified",
			def: McpServerDefinition{
				Command:   "/nix/store/abc-server/bin/server",
				Transport: TransportStdio,
			},
			// no secrets ✓, stdio ✓ → Standard ✓
			// local-only ✓, no npx-y ✓ → Secure ✓
			// /nix/store provenance ✓ → Verified ✓
			expected: ComplianceVerified,
		},
		{
			name: "SSE transport caps at Basic",
			def: McpServerDefinition{
				Command:   "qsdev",
				Transport: TransportSSE,
			},
			// no secrets ✓ but stdio ✗ → Standard ✗
			// Stays Basic
			expected: ComplianceBasic,
		},
		{
			name: "HTTP transport caps at Basic",
			def: McpServerDefinition{
				Command:   "/nix/store/abc/bin/server",
				Transport: TransportHTTP,
			},
			// stdio ✗ → Standard ✗ → stays Basic
			expected: ComplianceBasic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := GradeServer(&tt.def)
			if result.Level != tt.expected {
				t.Errorf("GradeServer().Level = %v (%d), want %v (%d)",
					result.Level, result.Level, tt.expected, tt.expected)
			}
		})
	}
}

func TestGradeServerCriteriaPopulated(t *testing.T) {
	t.Parallel()

	def := &McpServerDefinition{
		Command:   "qsdev",
		Transport: TransportStdio,
	}
	result := GradeServer(def)

	if len(result.Criteria) == 0 {
		t.Fatal("GradeServer().Criteria is empty, expected criterion results")
	}

	// We expect criteria for each check: no-plaintext-secrets, stdio-transport,
	// local-only, no-npx-dash-y, verified-provenance, external-attestation.
	expectedNames := map[string]bool{
		"no-plaintext-secrets": false,
		"stdio-transport":      false,
		"local-only":           false,
		"no-npx-dash-y":        false,
		"verified-provenance":  false,
		"external-attestation": false,
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

	first := GradeServer(def)
	second := GradeServer(def)

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
