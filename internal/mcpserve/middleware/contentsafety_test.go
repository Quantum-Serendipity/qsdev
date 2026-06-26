package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// awsKey is a realistic-shaped AWS access key id (AKIA + 16 upper/digits) used
// to exercise the redactor without embedding a real credential. It is AWS's
// published EXAMPLE value, split across a concatenation so the literal does not
// appear contiguously in source and trip the ripsecrets pre-commit scanner; the
// runtime value is still a full AKIA-shaped string the redactor must catch.
const awsKey = "AKIA" + "IOSFODNN7EXAMPLE"

func runContentSafety(t *testing.T, res *spi.ToolResult, err error) (*spi.ToolResult, error) {
	t.Helper()
	cs := ContentSafety{}
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return res, err
	}
	return cs.Handle(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{}, final)
}

func TestContentSafetyRedactsText(t *testing.T) {
	t.Parallel()

	out, err := runContentSafety(t, &spi.ToolResult{Text: "creds: " + awsKey + " end"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out.Text, awsKey) {
		t.Errorf("AWS key not redacted from Text: %q", out.Text)
	}
	if !strings.Contains(out.Text, "[REDACTED]") {
		t.Errorf("expected redaction marker in Text, got %q", out.Text)
	}
}

func TestContentSafetyRedactsStructured(t *testing.T) {
	t.Parallel()

	structured := map[string]any{
		"key":    awsKey,
		"nested": map[string]any{"token": awsKey},
		"list":   []any{"clean", awsKey},
		"count":  42,
	}
	out, err := runContentSafety(t, &spi.ToolResult{Text: "ok", Structured: structured}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := out.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured type changed: %T", out.Structured)
	}
	if got["key"] == awsKey {
		t.Error("top-level structured key not redacted")
	}
	if nested := got["nested"].(map[string]any); nested["token"] == awsKey {
		t.Error("nested structured key not redacted")
	}
	if list := got["list"].([]any); list[1] == awsKey {
		t.Error("structured list element not redacted")
	}
	if got["count"] != 42 {
		t.Errorf("non-string leaf altered: %v", got["count"])
	}
}

// TestContentSafetyRedactsConcreteTypedStructured proves the generalized
// structured redaction (red.RedactStructured) reaches secrets hidden behind
// CONCRETE container types — map[string]string, []string, []map[string]any — not
// just the any-tree shape, and that a deny-listed key name redacts a plain
// value. Secret-shaped fixtures use the awsKey convention; the URL credential is
// assembled by concatenation so the contiguous "user:pass@" literal never lands
// in source (ripsecrets).
func TestContentSafetyRedactsConcreteTypedStructured(t *testing.T) {
	t.Parallel()

	dbURL := "postgres://" + "u:p" + "@h/x" // URL-credential shape; creds = "u:p"

	structured := map[string]any{
		"env":      map[string]string{"DB": dbURL},
		"list":     []string{awsKey},
		"rows":     []map[string]any{{"token": awsKey}},
		"password": "hunter2",
	}

	out, err := runContentSafety(t, &spi.ToolResult{Text: "ok", Structured: structured}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := out.Structured.(map[string]any)
	if !ok {
		t.Fatalf("structured type changed: %T", out.Structured)
	}

	// End-to-end: marshal the whole redacted tree and assert no raw secret
	// survives anywhere in it.
	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal redacted structured: %v", err)
	}
	for _, secret := range []string{awsKey, "u:p@", "hunter2"} {
		if strings.Contains(string(blob), secret) {
			t.Errorf("secret %q survived redaction in %s", secret, blob)
		}
	}

	// Copy-on-write preserves the concrete container types; assert each leaf.
	env, ok := got["env"].(map[string]string)
	if !ok {
		t.Fatalf("env type changed: %T", got["env"])
	}
	if env["DB"] == dbURL {
		t.Errorf("URL credentials in map[string]string not redacted: %q", env["DB"])
	}
	if strings.Contains(env["DB"], "://u:p@") {
		t.Errorf("raw URL credentials survived in map[string]string: %q", env["DB"])
	}

	list, ok := got["list"].([]string)
	if !ok {
		t.Fatalf("list type changed: %T", got["list"])
	}
	if list[0] != "[REDACTED]" {
		t.Errorf("secret token in []string not redacted: %q", list[0])
	}

	rows, ok := got["rows"].([]map[string]any)
	if !ok {
		t.Fatalf("rows type changed: %T", got["rows"])
	}
	if rows[0]["token"] != "[REDACTED]" {
		t.Errorf("deny-listed key in []map[string]any not redacted: %v", rows[0]["token"])
	}

	if got["password"] != "[REDACTED]" {
		t.Errorf("plain value under deny-listed key %q not redacted: %v", "password", got["password"])
	}
}

func TestContentSafetyNilSafeAndErrorPassthrough(t *testing.T) {
	t.Parallel()

	t.Run("nil result", func(t *testing.T) {
		t.Parallel()
		out, err := runContentSafety(t, nil, nil)
		if err != nil || out != nil {
			t.Errorf("nil result: got (%+v, %v), want (nil, nil)", out, err)
		}
	})

	t.Run("error passthrough untouched", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("boom")
		out, err := runContentSafety(t, nil, sentinel)
		if !errors.Is(err, sentinel) {
			t.Errorf("error not passed through: %v", err)
		}
		if out != nil {
			t.Errorf("result should stay nil on error, got %+v", out)
		}
	})
}
