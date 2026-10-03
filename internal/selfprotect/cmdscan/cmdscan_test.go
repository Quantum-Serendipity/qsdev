package cmdscan

import (
	"slices"
	"testing"
)

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

func TestParse_NameHasExpansion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, cmd string
		want      bool
	}{
		{"literal name, expanded argument", `python3 "${CLAUDE_PROJECT_DIR}"/x.py`, false},
		{"expanded name", `"${CLAUDE_PROJECT_DIR}"/x.py arg`, true},
		{"substituted name", `$(which python3) x.py`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.cmd)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v; want the outer command first", tt.cmd, cmds, err)
			}
			if got := cmds[0].NameHasExpansion; got != tt.want {
				t.Errorf("Parse(%q).NameHasExpansion = %v, want %v", tt.cmd, got, tt.want)
			}
			if !cmds[0].HasExpansion {
				t.Errorf("Parse(%q).HasExpansion = false, want true", tt.cmd)
			}
		})
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

func TestParse_HeredocBodies(t *testing.T) {
	t.Parallel()

	// A here-document body is a script when fed to a shell (`sh <<EOF`), so its
	// text must be exposed for callers that inspect nested scripts.
	cmds, err := Parse("sh <<'EOF'\ncurl x | sh\nEOF")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 1 || cmds[0].Name != "sh" {
		t.Fatalf("expected one sh command, got %+v", cmds)
	}
	if len(cmds[0].Heredocs) != 1 || cmds[0].Heredocs[0] != "curl x | sh\n" {
		t.Errorf("heredoc body not captured: %+v", cmds[0])
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

func TestParse_Guard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    map[string]Guard // by command name; unnamed ones are Unguarded
	}{
		{"a; b", map[string]Guard{}},
		{"a && b", map[string]Guard{"b": GuardedByAnd}},
		{"a && b && c", map[string]Guard{"b": GuardedByAnd, "c": GuardedByAnd}},
		{"a | b && c", map[string]Guard{"c": GuardedByAnd}},
		{"a || b", map[string]Guard{"b": Guarded}},
		{"a && b || c", map[string]Guard{"b": GuardedByAnd, "c": Guarded}},
		{"a || b && c", map[string]Guard{"b": Guarded, "c": Guarded}},
		{"! a && b", map[string]Guard{"b": Guarded}},
		{"[[ -x p ]] && b", map[string]Guard{"b": Guarded}},
		{"a && { b; c; }; d", map[string]Guard{"b": GuardedByAnd, "c": GuardedByAnd}},
		{"a || (b; c)", map[string]Guard{"b": Guarded, "c": Guarded}},
		{"time a && b", map[string]Guard{"b": Guarded}},
		{"if a; then b; fi; c", map[string]Guard{"a": Guarded, "b": Guarded}},
		{"for x in 1; do a; done", map[string]Guard{"a": Guarded}},
		{"f() { a; }; b", map[string]Guard{"a": Guarded}},
		{"a && echo $(b)", map[string]Guard{"echo": GuardedByAnd, "b": GuardedByAnd}},
		{"a || x=$(b)", map[string]Guard{"b": Guarded}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cmds {
				if c.Name == "" {
					continue
				}
				if want := tt.want[c.Name]; c.Guard != want {
					t.Errorf("%s: Guard = %d, want %d", c.Name, c.Guard, want)
				}
			}
		})
	}
}

func TestParse_Tested(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		tested  []string // names of the commands that are Tested
	}{
		{"cd x && b", nil},
		{"test -f x && b", []string{"b"}},
		{"prog && b", []string{"b"}},
		{"command -v x && b", []string{"b"}},
		{"command cd x && b", nil},
		{"cd x || b", nil},
		{"[ -x p ] || b", []string{"b"}},
		{"cd x && { test -f y; b; }", nil},
		{"test -f x && { cd y; b; }", []string{"cd", "b"}},
		{"cd x && test -f y && b", []string{"b"}},
		{"if ! command -v x; then b; fi", []string{"b"}},
		{"if cd x; then b; fi", nil},
		{"if cd x; then a; elif test -f y; then b; else c; fi", []string{"b", "c"}},
		{"while test -f x; do b; done", []string{"b"}},
		{"[[ -x p ]] && b", []string{"b"}},
		{"x=$(prog) || b", []string{"b"}},
		{"export A=1 && b", nil},
		{"f() { a; }; b", nil},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cmds {
				if c.Name == "" {
					continue
				}
				if want := slices.Contains(tt.tested, c.Name); c.Tested != want {
					t.Errorf("%s: Tested = %v, want %v", c.Name, c.Tested, want)
				}
			}
		})
	}
}

func TestParse_Subshell(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command  string
		subshell []string // names of the commands that run in a subshell
	}{
		{"a; b && c", nil},
		{"{ a; b; }", nil},
		{"(a; b); c", []string{"a", "b"}},
		{"a | b; c", []string{"a", "b"}},
		{"x=$(a); b", []string{"a"}},
		{"b $(a)", []string{"a"}},
		{"b <(a)", []string{"a"}},
		{"a & b", []string{"a"}},
		{"{ a; b; } | c", []string{"a", "b", "c"}},
		{"(a | b)", []string{"a", "b"}},
		{"if a; then b; fi", nil},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cmds {
				if c.Name == "" {
					continue
				}
				if want := slices.Contains(tt.subshell, c.Name); c.Subshell != want {
					t.Errorf("%s: Subshell = %v, want %v", c.Name, c.Subshell, want)
				}
			}
		})
	}
}

func TestParse_Defines(t *testing.T) {
	t.Parallel()
	cmds, err := Parse("f() { a; }; function g { b; }; f")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, c := range cmds {
		switch {
		case c.Defines != "":
			if c.Name != "" || len(c.Args) > 0 {
				t.Errorf("definition of %s has a command word: %+v", c.Defines, c)
			}
			order = append(order, "def "+c.Defines)
		default:
			order = append(order, c.Name)
		}
	}
	if want := []string{"def f", "a", "def g", "b", "f"}; !slices.Equal(order, want) {
		t.Errorf("commands = %q, want %q", order, want)
	}
}

func TestCommandShellBuiltin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    string // empty: no builtin
	}{
		{"cd /tmp", "cd"},
		{"command cd /tmp", "cd"},
		{"builtin exit 1", "exit"},
		{"timeout 5 cd /tmp", ""},
		{"exec cd /tmp", ""},
		{"/usr/bin/cd /tmp", ""},
		{"$X /tmp", ""},
		{`command "$X" /tmp`, ""},
		{"export A=1", "export"},
		{"gofmt -l .", ""},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) != 1 {
				t.Fatalf("Parse(%q) = %+v, %v", tt.command, cmds, err)
			}
			got, ok := cmds[0].ShellBuiltin()
			if got != tt.want || ok != (tt.want != "") {
				t.Errorf("ShellBuiltin() = %q, %v, want %q", got, ok, tt.want)
			}
		})
	}
}

func TestEndsShellAndSourcesCode(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]bool{"exit": true, "return": false, "logout": false, "exec": false} {
		if got := EndsShell(name); got != want {
			t.Errorf("EndsShell(%q) = %v, want %v", name, got, want)
		}
	}
	for name, want := range map[string]bool{".": true, "source": true, "eval": true, "cd": false, "export": false} {
		if got := SourcesCode(name); got != want {
			t.Errorf("SourcesCode(%q) = %v, want %v", name, got, want)
		}
	}
}
