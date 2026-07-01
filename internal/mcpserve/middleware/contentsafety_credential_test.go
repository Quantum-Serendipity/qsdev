package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// runContentSafetyWithCategory drives ContentSafety with a ToolCallContext whose
// Category is set, so the credential exemption can be exercised.
func runContentSafetyWithCategory(t *testing.T, category string, res *spi.ToolResult) *spi.ToolResult {
	t.Helper()
	cs := ContentSafety{}
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return res, nil
	}
	out, err := cs.Handle(context.Background(), &spi.ToolCallContext{Category: category}, &spi.ToolRequest{}, final)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return out
}

// TestContentSafetyRedactsNonCredentialCategory confirms a normal (non-credential)
// tool still has secret-shaped output redacted.
func TestContentSafetyRedactsNonCredentialCategory(t *testing.T) {
	t.Parallel()
	out := runContentSafetyWithCategory(t, CategorySecurity,
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

// TestContentSafetyExemptsCredentialCategory confirms the credential category —
// the sole tool sanctioned to emit credentials — passes through unredacted, so
// the vended token reaches the agent intact.
func TestContentSafetyExemptsCredentialCategory(t *testing.T) {
	t.Parallel()
	out := runContentSafetyWithCategory(t, CategoryCredential,
		&spi.ToolResult{
			Text:       "session token: " + awsKey,
			Structured: map[string]any{"access_key_id": awsKey},
		})

	if !strings.Contains(out.Text, awsKey) {
		t.Errorf("credential Text was redacted but must pass through: %q", out.Text)
	}
	if got := out.Structured.(map[string]any)["access_key_id"]; got != awsKey {
		t.Errorf("credential structured value was redacted: %v", got)
	}
}
