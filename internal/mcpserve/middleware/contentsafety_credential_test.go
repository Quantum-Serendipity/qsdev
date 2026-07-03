package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

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
