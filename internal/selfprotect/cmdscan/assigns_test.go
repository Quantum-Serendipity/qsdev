package cmdscan

import (
	"slices"
	"testing"
)

// TestParse_OtherAssignForms pins that every bash form that sets or clears a
// variable is reported in Command.Assigns, or as AssignsDynamic when the name
// comes from an expansion. Each of these set the HOME a later `qsdev init
// --update` saw while the self-protection hook allowed the line (U18-WS1).
func TestParse_OtherAssignForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command     string
		wantAssigns []string
		wantDynamic bool
	}{
		{"for HOME in /tmp/e; do qsdev init --update; done", []string{"HOME"}, false},
		{"select HOME in /tmp/e; do break; done", []string{"HOME"}, false},
		{"read HOME <<< /tmp/e", []string{"HOME"}, false},
		{"read -r -p prompt HOME X", []string{"HOME", "X"}, false},
		{"read -a HOME", []string{"HOME"}, false},
		{"read -raHOME", []string{"HOME"}, false},
		{"read -d '' -n 3 HOME", []string{"HOME"}, false},
		{"printf -v HOME %s /tmp/e", []string{"HOME"}, false},
		{"printf -vHOME %s /tmp/e", []string{"HOME"}, false},
		{"builtin printf -v HOME %s /tmp/e", []string{"HOME"}, false},
		{"mapfile -t HOME <<< /tmp/e", []string{"HOME"}, false},
		{"readarray -t -u 3 HOME", []string{"HOME"}, false},
		{"getopts e HOME -e", []string{"HOME"}, false},
		{"wait -n -p HOME", []string{"HOME"}, false},
		{"unset -v HOME", []string{"HOME"}, false},
		{"((HOME=5))", []string{"HOME"}, false},
		{"((HOME++))", []string{"HOME"}, false},
		{"let HOME=5", []string{"HOME"}, false},
		{"let 'HOME += 1'", []string{"HOME"}, false},
		{"echo $((HOME=5))", []string{"HOME"}, false},
		{": ${HOME:=/tmp/e}", []string{"HOME"}, false},
		{"export HOME=/tmp/e", []string{"HOME"}, false},
		{"export 'HOME=/tmp/e'", []string{"HOME"}, false},
		{"export -n HOME", []string{"HOME"}, false},
		{"local HOME", []string{"HOME"}, false},
		{"command export HOME=/tmp/e", []string{"HOME"}, false},
		{"declare -n R=HOME", []string{"R", "HOME"}, false},
		{"typeset -gn R=HOME", []string{"R", "HOME"}, false},
		{"coproc HOME { cat; }", []string{"HOME"}, false},
		{"exec {HOME}>/dev/null", []string{"HOME"}, false},
		{"X=HOME; export $X=/tmp/e", []string{"X"}, true},
		{"export HO$X=1", nil, true},
		{"declare -n R=$X", []string{"R"}, true},
		{"read \"$V\"", nil, true},
		{"printf -v \"$V\" %s x", nil, true},
		{"let \"$X=1\"", nil, true},
		{": ${!X:=1}", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.command, err)
			}
			var assigns []string
			dynamic := false
			for _, c := range cmds {
				assigns = append(assigns, c.Assigns...)
				dynamic = dynamic || c.AssignsDynamic
			}
			for _, want := range tt.wantAssigns {
				if !slices.Contains(assigns, want) {
					t.Errorf("Parse(%q) assigns %q, want it to include %q", tt.command, assigns, want)
				}
			}
			if dynamic != tt.wantDynamic {
				t.Errorf("Parse(%q) AssignsDynamic = %v, want %v", tt.command, dynamic, tt.wantDynamic)
			}
			for _, c := range cmds {
				if (len(c.Assigns) > 0 || c.AssignsDynamic) && IsSafeReadCommand(c) {
					t.Errorf("IsSafeReadCommand(%+v) = true for a command that sets variables", c)
				}
			}
		})
	}
}

// TestParse_NoSpuriousAssigns pins that ordinary commands report no
// variables: an option argument, a format operand or a comparison is not an
// assignment.
func TestParse_NoSpuriousAssigns(t *testing.T) {
	t.Parallel()

	for _, command := range []string{
		"printf '%s\\n' HOME",
		"echo HOME=1",
		"read -p HOME",
		"((HOME == 5))",
		"cat \"$HOME/x\"",
		"echo ${HOME:-/tmp}",
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(command)
			if err != nil {
				t.Fatalf("Parse(%q): %v", command, err)
			}
			for _, c := range cmds {
				if len(c.Assigns) > 0 || c.AssignsDynamic {
					t.Errorf("Parse(%q) = assigns %q dynamic %v, want none", command, c.Assigns, c.AssignsDynamic)
				}
			}
		})
	}
}
