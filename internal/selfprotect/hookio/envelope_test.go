package hookio

import (
	"context"
	"strings"
	"testing"
)

// TestParseToolCall_DecodesSessionCwd covers F134: the envelope's cwd (the
// session directory, which persists across Bash calls) reaches the rules.
func TestParseToolCall_DecodesSessionCwd(t *testing.T) {
	t.Parallel()

	envelope := `{"tool_name":"Bash","tool_input":{"command":"echo x > settings.json"},"cwd":"/home/u/project/.claude"}`
	call, err := ParseToolCall(context.Background(), strings.NewReader(envelope))
	if err != nil {
		t.Fatalf("ParseToolCall error: %v", err)
	}
	if call.CWD != "/home/u/project/.claude" {
		t.Errorf("CWD = %q, want the envelope cwd", call.CWD)
	}
}

// TestParseInput_DecodesReplaceAll ensures Edit/MultiEdit replace_all flags are
// decoded, so the resulting file can be reconstructed faithfully.
func TestParseInput_DecodesReplaceAll(t *testing.T) {
	t.Parallel()

	edit := ParseInput([]byte(`{"file_path":"f","old_string":"a","new_string":"b","replace_all":true}`))
	if !edit.ReplaceAll {
		t.Error("Edit replace_all not decoded")
	}
	multi := ParseInput([]byte(`{"file_path":"f","edits":[{"old_string":"a","new_string":"b","replace_all":true},{"old_string":"c","new_string":"d"}]}`))
	if len(multi.Edits) != 2 || !multi.Edits[0].ReplaceAll || multi.Edits[1].ReplaceAll {
		t.Errorf("MultiEdit replace_all not decoded per edit: %+v", multi.Edits)
	}
}
