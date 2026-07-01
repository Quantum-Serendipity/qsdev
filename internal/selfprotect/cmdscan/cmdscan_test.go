package cmdscan

import "testing"

func TestParse_SegmentsAndArgs(t *testing.T) {
	t.Parallel()

	cmds, err := Parse("rm -rf /tmp/build && grep secret .claude/settings.json")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d: %+v", len(cmds), cmds)
	}
	if cmds[0].Name != "rm" || len(cmds[0].Args) != 2 || cmds[0].Args[1] != "/tmp/build" {
		t.Errorf("first command parsed wrong: %+v", cmds[0])
	}
	if cmds[1].Name != "grep" || cmds[1].Args[1] != ".claude/settings.json" {
		t.Errorf("second command parsed wrong: %+v", cmds[1])
	}
}

func TestParse_ExpansionDetection(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		`eval "$X"`:          true,  // parameter expansion inside double quotes
		`eval $(whoami)`:     true,  // command substitution
		`eval "cleanup"`:     false, // pure literal
		`grep 'eval "$("' f`: false, // $ is inside a single-quoted literal arg
	}
	for cmd, wantExp := range cases {
		cmds, err := Parse(cmd)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", cmd, err)
		}
		var got bool
		for _, c := range cmds {
			if c.Name == "eval" && c.HasExpansion {
				got = true
			}
		}
		if got != wantExp {
			t.Errorf("Parse(%q): eval-expansion = %v, want %v (%+v)", cmd, got, wantExp, cmds)
		}
	}
}

func TestParse_Redirects(t *testing.T) {
	t.Parallel()

	cmds, err := Parse("echo '{}' > .mcp.json")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 1 || len(cmds[0].Redirects) != 1 || cmds[0].Redirects[0] != ".mcp.json" {
		t.Errorf("redirect target not captured: %+v", cmds)
	}
}

func TestParse_MalformedReturnsError(t *testing.T) {
	t.Parallel()

	// Unterminated quote — callers must fail closed on this error.
	if _, err := Parse(`rm "unterminated`); err == nil {
		t.Error("expected parse error for malformed command, got nil")
	}
}
