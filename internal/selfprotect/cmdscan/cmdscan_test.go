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
	if len(cmds) != 1 || len(cmds[0].WriteRedirects) != 1 || cmds[0].WriteRedirects[0] != ".mcp.json" {
		t.Errorf("write redirect target not captured: %+v", cmds)
	}
	if len(cmds[0].ReadRedirects) != 0 {
		t.Errorf("output redirect misclassified as read: %+v", cmds)
	}
}

func TestParse_RedirectOpSplit(t *testing.T) {
	t.Parallel()

	// An input redirect is a read, not a mutation: it must not land in
	// WriteRedirects (a bug there falsely denies `cat < .mcp.json`).
	cmds, err := Parse("cat < .mcp.json && echo hi")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	var reads, writes []string
	for _, c := range cmds {
		reads = append(reads, c.ReadRedirects...)
		writes = append(writes, c.WriteRedirects...)
	}
	if len(writes) != 0 {
		t.Errorf("input redirect classified as write: %+v", cmds)
	}
	if len(reads) != 1 || reads[0] != ".mcp.json" {
		t.Errorf("input redirect not captured as read: %+v", cmds)
	}
}

func TestParse_CompoundRedirects(t *testing.T) {
	t.Parallel()

	// A redirect on a compound command (block or subshell) attaches to the
	// outer statement, whose Cmd is not a CallExpr. It must still be captured
	// (as a nameless command) or `{ echo evil; } > .mcp.json` bypasses the
	// mutation rules entirely.
	for _, cmd := range []string{"{ echo evil; } > .mcp.json", "( echo evil ) > .mcp.json"} {
		cmds, err := Parse(cmd)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", cmd, err)
		}
		var writes []string
		for _, c := range cmds {
			writes = append(writes, c.WriteRedirects...)
		}
		if len(writes) != 1 || writes[0] != ".mcp.json" {
			t.Errorf("Parse(%q): compound redirect not captured: %+v", cmd, cmds)
		}
	}
}

func TestParse_PipelineGrouping(t *testing.T) {
	t.Parallel()

	cmds, err := Parse("cat .claude/settings.json | grep foo | tee /tmp/out")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d: %+v", len(cmds), cmds)
	}
	// All three stages share one non-zero pipeline id, in order.
	id := cmds[0].Pipeline
	if id == 0 {
		t.Fatalf("pipeline id not assigned: %+v", cmds)
	}
	for i, c := range cmds {
		if c.Pipeline != id {
			t.Errorf("stage %d (%s) pipeline id = %d, want %d", i, c.Name, c.Pipeline, id)
		}
	}
	if cmds[0].Name != "cat" || cmds[2].Name != "tee" {
		t.Errorf("pipeline order wrong: %+v", cmds)
	}

	// A standalone command has no pipeline id.
	solo, err := Parse("rm -rf /tmp/build")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(solo) != 1 || solo[0].Pipeline != 0 {
		t.Errorf("standalone command should have pipeline id 0: %+v", solo)
	}
}

func TestParse_MalformedReturnsError(t *testing.T) {
	t.Parallel()

	// Unterminated quote — callers must fail closed on this error.
	if _, err := Parse(`rm "unterminated`); err == nil {
		t.Error("expected parse error for malformed command, got nil")
	}
}
