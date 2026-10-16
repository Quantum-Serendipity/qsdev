package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets/secretstest"
)

// canonSampleCase is one secrets canon sample embedded in context and the tool
// result that carries it in Text and in nested Structured leaves.
type canonSampleCase struct {
	name, sample, input, prefix, suffix string
}

// canonSampleCases lists every secretstest sample of every secrets.ValuePatterns
// entry, failing the test for a canon entry that has no sample.
func canonSampleCases(t *testing.T) []canonSampleCase {
	t.Helper()
	samples := secretstest.ValuePatternSamples()
	var cases []canonSampleCase
	for _, vp := range secrets.ValuePatterns {
		if len(samples[vp.Name]) == 0 {
			t.Errorf("canon entry %q has no secretstest sample", vp.Name)
		}
		for i, sample := range samples[vp.Name] {
			input, prefix, suffix := secretstest.InContext(sample)
			cases = append(cases, canonSampleCase{
				name: fmt.Sprintf("%s/%d", vp.Name, i), sample: sample,
				input: input, prefix: prefix, suffix: suffix,
			})
		}
	}
	return cases
}

// result returns a fresh tool result carrying the case input in Text and in a
// map leaf and a nested slice leaf of Structured.
func (c canonSampleCase) result() *spi.ToolResult {
	return &spi.ToolResult{
		Text:       c.input,
		Structured: map[string]any{"detail": c.input, "nested": []any{c.input}},
	}
}

// stringLeaves collects every string leaf of a JSON-shaped value.
func stringLeaves(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case map[string]any:
		var out []string
		for _, e := range x {
			out = append(out, stringLeaves(e)...)
		}
		return out
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, stringLeaves(e)...)
		}
		return out
	default:
		return nil
	}
}

// TestContentSafety_RedactsEveryCanonSample proves the MCP tool-result surface
// covers every token shape in the secrets canon: for a non-vend tool, each
// sample is scrubbed from Text and from every Structured string leaf, while the
// text around it survives.
func TestContentSafety_RedactsEveryCanonSample(t *testing.T) {
	t.Parallel()
	for _, c := range canonSampleCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := runContentSafetyWithTool(t, "qsdev_security_scan", CategorySecurity, c.result())

			t.Run("text", func(t *testing.T) {
				if strings.Contains(out.Text, c.sample) {
					t.Errorf("ContentSafety left the sample in Text: %q", out.Text)
				}
				if !strings.HasPrefix(out.Text, c.prefix) || !strings.HasSuffix(out.Text, c.suffix) {
					t.Errorf("ContentSafety lost the surrounding text: %q", out.Text)
				}
			})
			t.Run("structured", func(t *testing.T) {
				leaves := stringLeaves(out.Structured)
				if len(leaves) != 2 {
					t.Fatalf("Structured has %d string leaves, want 2: %#v", len(leaves), out.Structured)
				}
				for _, leaf := range leaves {
					if strings.Contains(leaf, c.sample) {
						t.Errorf("ContentSafety left the sample in a Structured leaf: %q", leaf)
					}
				}
			})
		})
	}
}

// TestContentSafety_VendToolPassesCanonSamples guards the credential-vend
// exemption as on base: the trusted vend tool's output carries every canon
// sample through unchanged, in Text and in Structured.
func TestContentSafety_VendToolPassesCanonSamples(t *testing.T) {
	t.Parallel()
	for _, c := range canonSampleCases(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := runContentSafetyWithTool(t, CredentialVendToolName, CategoryCredential, c.result())
			if out.Text != c.input {
				t.Errorf("vend tool Text = %q, want unchanged %q", out.Text, c.input)
			}
			leaves := stringLeaves(out.Structured)
			if len(leaves) != 2 {
				t.Fatalf("Structured has %d string leaves, want 2: %#v", len(leaves), out.Structured)
			}
			for _, leaf := range leaves {
				if leaf != c.input {
					t.Errorf("vend tool Structured leaf = %q, want unchanged %q", leaf, c.input)
				}
			}
		})
	}
}

// runContentSafetyWithTool drives ContentSafety with a ToolCallContext and
// request whose tool name and category are set, so the credential-vend exemption
// (keyed on the trusted tool identity, not the declared category) can be
// exercised. It mirrors the bridge, which sets the tool name on BOTH cc.ToolName
// and req.Name from the registration.
func runContentSafetyWithTool(t *testing.T, toolName, category string, res *spi.ToolResult) *spi.ToolResult {
	t.Helper()
	cs := ContentSafety{}
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return res, nil
	}
	cc := &spi.ToolCallContext{ToolName: toolName, Category: category}
	out, err := cs.Handle(context.Background(), cc, &spi.ToolRequest{Name: toolName}, final)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

// TestContentSafetyRedactsNonCredentialCategory confirms a normal (non-credential)
// tool still has secret-shaped output redacted.
func TestContentSafetyRedactsNonCredentialCategory(t *testing.T) {
	t.Parallel()
	out := runContentSafetyWithTool(t, "qsdev_security_scan", CategorySecurity,
		&spi.ToolResult{
			Text:       "key: " + awsKey,
			Structured: map[string]any{"secret_access_key": awsKey},
		})

	if strings.Contains(out.Text, awsKey) {
		t.Errorf("non-credential Text was not redacted: %q", out.Text)
	}
	if got := out.Structured.(map[string]any)["secret_access_key"]; got == awsKey {
		t.Error("non-credential structured value was not redacted")
	}
}

// TestContentSafetyRedactsSpoofedCredentialCategory is the BUG B regression: a
// tool that is NOT the trusted credential-vend tool but DECLARES itself
// CategoryCredential must still have its secret-shaped output redacted. Before
// the fix the exemption keyed on the self-declared category, so any tool could
// assert "credential" and pass its output through un-redacted.
func TestContentSafetyRedactsSpoofedCredentialCategory(t *testing.T) {
	t.Parallel()
	out := runContentSafetyWithTool(t, "evil_tool", CategoryCredential,
		&spi.ToolResult{
			Text:       "session token: " + awsKey,
			Structured: map[string]any{"access_key_id": awsKey},
		})

	if strings.Contains(out.Text, awsKey) {
		t.Errorf("spoofed-credential Text was NOT redacted (category-declared exemption leaked): %q", out.Text)
	}
	if got := out.Structured.(map[string]any)["access_key_id"]; got == awsKey {
		t.Error("spoofed-credential structured value was NOT redacted (category-declared exemption leaked)")
	}
}

// TestContentSafetyExemptsCredentialVendTool confirms the genuine credential-vend
// tool — identified by its trusted, server-registered name — passes through
// unredacted, so the vended token reaches the agent intact.
func TestContentSafetyExemptsCredentialVendTool(t *testing.T) {
	t.Parallel()
	out := runContentSafetyWithTool(t, CredentialVendToolName, CategoryCredential,
		&spi.ToolResult{
			Text:       "session token: " + awsKey,
			Structured: map[string]any{"access_key_id": awsKey},
		})

	if !strings.Contains(out.Text, awsKey) {
		t.Errorf("credential-vend Text was redacted but must pass through: %q", out.Text)
	}
	if got := out.Structured.(map[string]any)["access_key_id"]; got != awsKey {
		t.Errorf("credential-vend structured value was redacted: %v", got)
	}
}

// TestContentSafetyRedactsPaddedBasicAuth confirms a tool result carrying a
// padded Basic credential (whose trailing '==' once looked like a new NAME= key
// and cut the redaction short) is fully redacted on the tool-result path, while
// a benign {"pass":3,"fail":0} check/posture summary passes through unchanged.
func TestContentSafetyRedactsPaddedBasicAuth(t *testing.T) {
	t.Parallel()
	const cred = "dXNlcjpwYXNzd29yZA"
	summary := `{"pass":3,"fail":0}`
	out := runContentSafetyWithTool(t, "qsdev_check", CategorySecurity,
		&spi.ToolResult{
			Text:       "request failed\nAuthorization: Basic " + cred + "==\n" + summary,
			Structured: map[string]any{"pass": 3, "fail": 0},
		})

	if strings.Contains(out.Text, cred) {
		t.Errorf("padded Basic credential survived redaction: %q", out.Text)
	}
	if !strings.Contains(out.Text, summary) {
		t.Errorf("benign pass/fail summary was altered: %q", out.Text)
	}
	got := out.Structured.(map[string]any)
	if got["pass"] != 3 || got["fail"] != 0 {
		t.Errorf("benign structured pass/fail was altered: %v", got)
	}
}
