package evasion

import "testing"

// TestCheck_UnparseableEvalFailsClosed covers F565: when the eval trigger fires
// but the command cannot be parsed, the obfuscation check must block rather
// than fall open.
func TestCheck_UnparseableEvalFailsClosed(t *testing.T) {
	t.Parallel()

	command := `eval "$PAYLOAD" 'unterminated`
	if blocked, category, _ := Check("Bash", command, ""); !blocked || category != "obfuscation" {
		t.Errorf("Check(%q) = (%v, %q), want blocked as obfuscation (fail closed)", command, blocked, category)
	}
}
