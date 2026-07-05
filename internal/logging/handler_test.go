package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestRedactingHandler_RedactsMessage proves the wrapping handler scrubs a secret
// embedded in a record's MESSAGE (not just its attributes) before forwarding to
// the inner handler — the third credential-leak path (handler copied r.Message
// verbatim before this fix).
func TestRedactingHandler_RedactsMessage(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewRedactingHandler(inner))

	logger.Info("connecting with PGPASSWORD=supersecret to db")

	out := buf.String()
	if strings.Contains(out, "supersecret") {
		t.Errorf("handler leaked secret embedded in message: %s", out)
	}
	if !strings.Contains(out, redacted) {
		t.Errorf("expected %s marker in message output: %s", redacted, out)
	}
}
