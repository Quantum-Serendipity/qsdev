package cmdscan

import (
	"slices"
	"testing"
)

// TestParse_QuoteAndEscapeRemoval covers the shell's quote removal in
// argument words (F135): a protected path spelled with split quotes,
// backslash escapes, or an ANSI-C string must reach the rules as the literal
// path the command receives.
func TestParse_QuoteAndEscapeRemoval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    string
		wantExp bool
	}{
		{"empty double quotes", `rm .cl""aude/settings.json`, ".claude/settings.json", false},
		{"empty single quotes", `rm .cl''aude/settings.json`, ".claude/settings.json", false},
		{"quoted tail", `rm .cl'aude'`, ".claude", false},
		{"backslash escape", `rm .cl\aude/settings.json`, ".claude/settings.json", false},
		{"escaped space", `rm my\ file`, "my file", false},
		{"double-quoted escaped quote", `rm "a\"b"`, `a"b`, false},
		{"double-quoted literal backslash", `rm "a\qb"`, `a\qb`, false},
		{"ansi-c hex escape", `rm $'\x2e'claude/settings.json`, ".claude/settings.json", false},
		{"ansi-c octal escape", `rm $'\056claude'`, ".claude", false},
		{"ansi-c unicode escape", `rm $'.claude'`, ".claude", false},
		{"ansi-c simple escapes", `rm $'a\tb\\c'`, "a\tb\\c", false},
		{"ansi-c unmodelled escape is opaque", `rm $'\cA.claude'`, `\cA.claude`, true},
		// Bash ends an ANSI-C string at its first NUL; the word goes on.
		{"ansi-c hex NUL ends the string", `rm -$'\x00m'x`, "-x", false},
		{"ansi-c octal NUL ends the string", `rm a$'b\0c'd`, "abd", false},
		{"ansi-c wrapped octal NUL ends the string", `rm a$'b\400c'd`, "abd", false},
		{"ansi-c unicode NUL ends the string", `rm a$'b\u0000c'd`, "abd", false},
		{"ansi-c long unicode NUL ends the string", `rm a$'b\U00000000c'd`, "abd", false},
		{"ansi-c NUL drops an unmodelled escape after it", `rm a$'\x00\cA'd`, "ad", false},
		{"ansi-c octal 1 is kept", `rm a$'\401'd`, "a\x01d", false},
		{"glob characters are kept", `rm .c*e/settings.json`, ".c*e/settings.json", false},
		{"braces are kept", `rm .{claude,x}`, ".{claude,x}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.command, err)
			}
			if len(cmds) != 1 || len(cmds[0].Args) != 1 {
				t.Fatalf("Parse(%q) = %+v, want one command with one argument", tt.command, cmds)
			}
			if got := cmds[0].Args[0]; got != tt.want {
				t.Errorf("Parse(%q) argument = %q, want %q", tt.command, got, tt.want)
			}
			if cmds[0].HasExpansion != tt.wantExp {
				t.Errorf("Parse(%q) HasExpansion = %v, want %v", tt.command, cmds[0].HasExpansion, tt.wantExp)
			}
		})
	}
}

// TestParse_KnownSubstitutionOutput pins the command substitutions whose
// output is fixed by their argv (W1bN-02, W1bN-07): the git directory
// rev-parse prints, and the name command -v, which and type -P print. Any
// other substitution, including one with an extra option, a redirect, an
// assignment or a second statement, renders nothing. Every such word is
// still an expansion.
func TestParse_KnownSubstitutionOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"git dir", `rm "$(git rev-parse --git-dir)/hooks/pre-commit"`, ".git/hooks/pre-commit"},
		{"absolute git dir", `rm "$(git rev-parse --absolute-git-dir)/hooks/x"`, ".git/hooks/x"},
		{"common git dir", `rm "$(git rev-parse --git-common-dir)/config"`, ".git/config"},
		{"git path", `rm "$(git rev-parse --git-path hooks)/pre-commit"`, ".git/hooks/pre-commit"},
		{"git path to a file", "rm `git rev-parse --git-path hooks/pre-commit`", ".git/hooks/pre-commit"},
		{"git dir in backticks", "rm `git rev-parse --git-dir`/hooks/pre-commit", ".git/hooks/pre-commit"},
		{"git dir with -C", `rm "$(git -C sub rev-parse --git-dir)/hooks/x"`, "sub/.git/hooks/x"},
		{"git dir with two -C", `rm "$(git -C a -C b rev-parse --git-dir)/x"`, "a/b/.git/x"},
		{"git dir with rooted -C", `rm "$(git -C a -C /r rev-parse --git-dir)/x"`, "/r/.git/x"},
		{"command -v", "x $(command -v git)", "git"},
		{"which", "x $(which rm)", "rm"},
		{"type -P", "x $(type -P rm)", "rm"},
		{"type -p", "x `type -p rm`", "rm"},
		{"echo still renders", "x $(echo .git)/y", ".git/y"},
		// Discarding another stream, and rev-parse's quiet and path-format
		// options, leave the path printed; "--" may end the lookup's options.
		{"stderr discarded", `rm "$(git rev-parse --git-dir 2>/dev/null)/x"`, ".git/x"},
		{"stderr appended to null", `rm "$(git rev-parse --git-path hooks/x 2>>/dev/null)"`, ".git/hooks/x"},
		{"quiet", `rm "$(git rev-parse -q --git-dir)/x"`, ".git/x"},
		{"long quiet", `rm "$(git rev-parse --quiet --git-path hooks)/x"`, ".git/hooks/x"},
		{"path format", `rm "$(git rev-parse --path-format=absolute --git-dir)/x"`, ".git/x"},
		{"command -v --", "x $(command -v -- git)", "git"},
		{"which --", "x $(which -- rm 2>/dev/null)", "rm"},
		{"type -P --", "x $(type -P -- rm)", "rm"},
		// Anything else prints output the scan does not know.
		{"two rev-parse options", `rm "$(git rev-parse --git-dir --show-toplevel)/x"`, "/x"},
		{"another rev-parse option", `rm "$(git rev-parse --show-toplevel)/x"`, "/x"},
		{"git path without a value", `rm "$(git rev-parse --git-path)/x"`, "/x"},
		{"git path with two values", `rm "$(git rev-parse --git-path a b)/x"`, "/x"},
		{"another git option", `rm "$(git -c a=b rev-parse --git-dir)/x"`, "/x"},
		{"another git subcommand", `rm "$(git config --get x)/x"`, "/x"},
		{"output discarded", `rm "$(git rev-parse --git-dir >/dev/null)/x"`, "/x"},
		{"output fd discarded", `rm "$(git rev-parse --git-dir 1>/dev/null)/x"`, "/x"},
		{"both streams discarded", `rm "$(git rev-parse --git-dir &>/dev/null)/x"`, "/x"},
		{"stderr to a file", `rm "$(git rev-parse --git-dir 2>/tmp/e)/x"`, "/x"},
		{"stderr merged", `rm "$(git rev-parse --git-dir 2>&1)/x"`, "/x"},
		{"input redirect", `rm "$(git rev-parse --git-dir </dev/null)/x"`, "/x"},
		{"pipeline", `rm "$(git rev-parse --git-dir | cat)/x"`, "/x"},
		{"another quiet-looking option", `rm "$(git rev-parse --verify --git-dir)/x"`, "/x"},
		{"lookup of an option", "x $(command -v -x)", ""},
		{"which alone", "x $(which)", ""},
		{"assignment", `rm "$(GIT_DIR=/o git rev-parse --git-dir)/x"`, "/x"},
		{"second statement", `rm "$(cd /; git rev-parse --git-dir)/x"`, "/x"},
		{"quoted operand", `rm "$(git rev-parse --git-path 'hooks')/x"`, "/x"},
		{"command without -v", "x $(command git)", ""},
		{"command -V", "x $(command -V git)", ""},
		{"which with an option", "x $(which -a rm)", ""},
		{"which of two names", "x $(which rm cp)", ""},
		{"type without -P", "x $(type rm)", ""},
		{"type -t", "x $(type -t rm)", ""},
		{"type -P of an option", "x $(type -P -a)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v; want a command", tt.command, cmds, err)
			}
			c := cmds[0] // the outer command; the substitution's own follow it
			if len(c.Args) == 0 {
				t.Fatalf("Parse(%q) = %+v, want an argument", tt.command, c)
			}
			if got := c.Args[len(c.Args)-1]; got != tt.want {
				t.Errorf("Parse(%q) last argument = %q, want %q", tt.command, got, tt.want)
			}
			if !c.HasExpansion || !slices.Contains(c.ExpandedArgs, tt.want) {
				t.Errorf("Parse(%q) HasExpansion = %v, ExpandedArgs = %q; want an expanded argument", tt.command, c.HasExpansion, c.ExpandedArgs)
			}
		})
	}

	// A substitution in the command word names the program it prints.
	cmds, err := Parse("$(command -v git) reset --hard x")
	if err != nil || len(cmds) == 0 {
		t.Fatalf("Parse = %+v, %v", cmds, err)
	}
	if c := cmds[0]; c.Name != "git" || !c.NameHasExpansion || !c.HasExpansion {
		t.Errorf("Name = %q, NameHasExpansion = %v, HasExpansion = %v; want git, true, true", c.Name, c.NameHasExpansion, c.HasExpansion)
	}

	// An assignment carries the rendered output for a later $NAME.
	cmds, err = Parse("d=$(git rev-parse --git-dir) x; P=`which rm` y")
	if err != nil || len(cmds) == 0 {
		t.Fatalf("Parse = %+v, %v", cmds, err)
	}
	var texts []string
	for _, c := range cmds {
		texts = append(texts, c.AssignTexts...)
	}
	if want := []string{"d=.git", "P=rm"}; !slices.Equal(texts, want) {
		t.Errorf("AssignTexts = %q, want %q", texts, want)
	}
}

// TestParse_DeclarationAssignTexts pins that a declaration builtin's
// assignments are carried in AssignTexts as rendered, like a bare
// assignment's, and its options are not.
func TestParse_DeclarationAssignTexts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    []string
	}{
		{"export d=$(git rev-parse --git-dir)", []string{"d=.git"}},
		{"declare -x d=`git rev-parse --git-path hooks` e", []string{"d=.git/hooks", "e"}},
		{"local d=$(git rev-parse --git-dir | cat)", []string{"d="}},
		{"readonly -p", nil},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v", tt.command, cmds, err)
			}
			if got := cmds[0].AssignTexts; !slices.Equal(got, tt.want) {
				t.Errorf("Parse(%q).AssignTexts = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

// TestParse_OutputRendered pins Command.OutputRendered: it marks the
// statement of a command substitution whose output the word holding it
// renders, and no other statement.
func TestParse_OutputRendered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    []bool // OutputRendered of each command, in order
	}{
		{`rm "$(git rev-parse --git-dir)/x"`, []bool{false, true}},
		{`rm "$(git rev-parse --git-dir 2>/dev/null)/x"`, []bool{false, true}},
		{`rm "$(git rev-parse --git-dir | cat)/x"`, []bool{false, false, false}},
		{`rm "$(cd . && git rev-parse --git-dir)/x"`, []bool{false, false, false}},
		{`read d < <(git rev-parse --git-dir)`, []bool{false, false}},
		{`git rev-parse --git-dir`, []bool{false}},
		// Rendered into an assignment value, a declaration or a redirect.
		{`d=$(git rev-parse --git-dir)`, []bool{false, true}},
		{`local d=$(git rev-parse --git-dir)`, []bool{false, true}},
		{`cat > "$(git rev-parse --git-dir)/x"`, []bool{false, true}},
		// A substitution no rendered word holds (W1bN-02): in the operand
		// of a parameter expansion, a for-loop word list or an array.
		{`rm "${GIT_DIR:-$(git rev-parse --git-dir)}/x"`, []bool{false, false}},
		{`rm "${GIT_DIR-$(git rev-parse --git-dir)}/x"`, []bool{false, false}},
		{`rm "${d:=$(git rev-parse --git-dir)}/x"`, []bool{false, false}},
		{`for d in $(git rev-parse --git-dir); do rm "$d/x"; done`, []bool{false, false, false}},
		{`a=($(git rev-parse --git-dir))`, []bool{false, false}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.command, err)
			}
			got := make([]bool, len(cmds))
			for i, c := range cmds {
				got[i] = c.OutputRendered
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("Parse(%q) OutputRendered = %v, want %v (commands %+v)", tt.command, got, tt.want, cmds)
			}
		})
	}
}

func TestParse_EscapedCommandWord(t *testing.T) {
	t.Parallel()

	cmds, err := Parse(`r\m -f x && r''m -f y`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cmds) != 2 || cmds[0].Name != "rm" || cmds[1].Name != "rm" {
		t.Errorf("escaped/quote-split command words not decoded: %+v", cmds)
	}
}

// TestParse_TildeWords pins that only a ~ the shell expands, unquoted and
// unescaped at the start of a word, is reported.
func TestParse_TildeWords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    []int
	}{
		{`~/bin/guard ~ x`, []int{0, 1}},
		{`exec ~/bin/guard`, []int{1}},
		{`"~/bin/guard"`, nil},
		{`'~'/bin/guard`, nil},
		{`\~/bin/guard`, nil},
		{`a~/b x~`, nil},
		{`~user/bin/guard`, []int{0}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) != 1 {
				t.Fatalf("Parse(%q) = %+v, %v", tt.command, cmds, err)
			}
			if !slices.Equal(cmds[0].TildeWords, tt.want) {
				t.Errorf("Parse(%q).TildeWords = %v, want %v", tt.command, cmds[0].TildeWords, tt.want)
			}
		})
	}
}
