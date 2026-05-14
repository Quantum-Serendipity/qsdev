package claudecode

import "testing"

func TestMatchesDenyRule_ExactMatch(t *testing.T) {
	// "Bash(npm install *)" should match "Bash(npm install lodash)"
	// because "npm install lodash" starts with "npm install ".
	if !matchesDenyRule("Bash(npm install *)", "Bash(npm install lodash)") {
		t.Error("Bash(npm install *) should match Bash(npm install lodash)")
	}
}

func TestMatchesDenyRule_NoMatch_DifferentCommand(t *testing.T) {
	// "Bash(npm install *)" should NOT match "Bash(npm test *)" because
	// the command prefix "npm install " is not a prefix of "npm test ".
	if matchesDenyRule("Bash(npm install *)", "Bash(npm test *)") {
		t.Error("Bash(npm install *) should NOT match Bash(npm test *)")
	}
}

func TestMatchesDenyRule_WildcardAll(t *testing.T) {
	// "Bash(npm *)" DOES match "Bash(npm test *)" — overly broad deny rule.
	if !matchesDenyRule("Bash(npm *)", "Bash(npm test *)") {
		t.Error("Bash(npm *) should match Bash(npm test *) — overly broad rule")
	}
}

func TestMatchesDenyRule_DifferentTool(t *testing.T) {
	// "Bash(npm *)" should NOT match "Read(.env)".
	if matchesDenyRule("Bash(npm *)", "Read(.env)") {
		t.Error("Bash(npm *) should NOT match Read(.env)")
	}
}

func TestMatchesDenyRule_ReadPattern(t *testing.T) {
	// "Read(./.env)" should match "Read(./.env)".
	if !matchesDenyRule("Read(./.env)", "Read(./.env)") {
		t.Error("Read(./.env) should match Read(./.env)")
	}
}

func TestMatchesDenyRule_ReadNoMatch(t *testing.T) {
	// "Read(./.env)" should NOT match "Read(./README.md)".
	if matchesDenyRule("Read(./.env)", "Read(./README.md)") {
		t.Error("Read(./.env) should NOT match Read(./README.md)")
	}
}

func TestMatchesDenyRule_NoParens(t *testing.T) {
	// Tool patterns without parentheses: only exact match on tool name
	// with empty args on both sides.
	if matchesDenyRule("Read", "Read(anything)") {
		t.Error("Read (no parens) should NOT match Read(anything)")
	}
	if !matchesDenyRule("Read", "Read") {
		t.Error("Read should match Read (exact)")
	}
}

func TestMatchesDenyRule_EmptyArgs(t *testing.T) {
	// "Bash(npm install)" (no wildcard) should match "Bash(npm install)" exactly.
	if !matchesDenyRule("Bash(npm install)", "Bash(npm install)") {
		t.Error("Bash(npm install) should match Bash(npm install) exactly")
	}
	// "Bash(npm install)" should NOT match "Bash(npm install lodash)".
	if matchesDenyRule("Bash(npm install)", "Bash(npm install lodash)") {
		t.Error("Bash(npm install) without wildcard should NOT match Bash(npm install lodash)")
	}
}

func TestMatchesDenyRule_TrailingWildcardPrefixMatch(t *testing.T) {
	tests := []struct {
		deny   string
		op     string
		expect bool
	}{
		// Deny rule prefix matches operation prefix.
		{"Bash(go install *)", "Bash(go install example.com/pkg)", true},
		{"Bash(pip install *)", "Bash(pip install requests)", true},
		{"Bash(cargo install *)", "Bash(cargo install ripgrep)", true},
		// Deny rule prefix does NOT match different operations.
		{"Bash(go install *)", "Bash(go build ./...)", false},
		{"Bash(pip install *)", "Bash(pip list)", false},
		{"Bash(cargo install *)", "Bash(cargo build)", false},
		// Wildcard-to-wildcard matching.
		{"Bash(git *)", "Bash(git log *)", true},
		{"Bash(git push --force *)", "Bash(git push *)", false},
		// Read patterns with wildcards.
		{"Read(./.env.*)", "Read(./.env.local)", true},
		{"Read(./.env.*)", "Read(./README.md)", false},
	}

	for _, tc := range tests {
		got := matchesDenyRule(tc.deny, tc.op)
		if got != tc.expect {
			t.Errorf("matchesDenyRule(%q, %q) = %v, want %v", tc.deny, tc.op, got, tc.expect)
		}
	}
}

func TestMatchesDenyRule_EmbeddedWildcard(t *testing.T) {
	// Embedded wildcards like "bash -c *npm install*" should not match simple operations.
	if matchesDenyRule("Bash(bash -c *npm install*)", "Bash(npm install lodash)") {
		t.Error("embedded wildcard rule should not match simple operation")
	}
}

func TestParseToolPattern(t *testing.T) {
	tests := []struct {
		input    string
		wantTool string
		wantArgs string
	}{
		{"Bash(npm install *)", "Bash", "npm install *"},
		{"Read(./.env)", "Read", "./.env"},
		{"Edit(*)", "Edit", "*"},
		{"Write", "Write", ""},
		{"Bash()", "Bash", ""},
	}

	for _, tc := range tests {
		tool, args := parseToolPattern(tc.input)
		if tool != tc.wantTool || args != tc.wantArgs {
			t.Errorf("parseToolPattern(%q) = (%q, %q), want (%q, %q)",
				tc.input, tool, args, tc.wantTool, tc.wantArgs)
		}
	}
}
