package cmdscan

import (
	"slices"
	"testing"
)

// TestWrittenOperands pins which operands the modelled commands write
// (U18-WS1 round 3): source and sed -n read their files, ln writes only the
// link it creates, and anything that may write or run elsewhere is not
// modelled, so callers keep treating every operand as written.
func TestWrittenOperands(t *testing.T) {
	t.Parallel()
	dirs := map[string]bool{"/home/u": true, "dir": true}
	isDir := func(w string) bool { return dirs[w] }
	tests := []struct {
		line    string
		want    []string
		modeled bool
	}{
		{"source ~/.bashrc", nil, true},
		{". ./env.sh", nil, true},
		{"source /dev/stdin", nil, false},
		{"source $F", nil, false},
		{"source <(echo x)", nil, false},
		{"sed -n 1p .bashrc", nil, true},
		{"sed -n -e 1,5p --quiet .bashrc", nil, true},
		{"sed --expression=1p .bashrc", nil, true},
		{"sed -i s/a/b/ .bashrc", nil, false},
		{"sed -ni p .bashrc", nil, false},
		{"sed -ie p .bashrc", nil, false},
		{"sed --in-place s/a/b/ .bashrc", nil, false},
		{"sed --in s/a/b/ .bashrc", nil, false},
		{"sed -f script.sed .bashrc", nil, false},
		{"sed -n 'w out' .bashrc", nil, false},
		{"sed 's/a/b/e' .bashrc", nil, false},
		{"sed -n $S .bashrc", nil, false},
		{"X=1 sed -n p .bashrc", nil, false},
		{"ln -s ~/.bashrc notes.txt", []string{"notes.txt"}, true},
		{"ln -sf /tmp/x .bashrc", []string{".bashrc"}, true},
		{"ln -s /tmp/.bashrc", []string{"/tmp/.bashrc"}, true},
		{"ln -s /tmp/.bashrc dir", []string{"dir", "dir/.bashrc"}, true},
		{"ln -s /tmp/.bashrc out/", []string{"out/", "out/.bashrc"}, true},
		{"ln -s a b c", []string{"c", "c/a", "c/b"}, true},
		{"ln -st /home/u /tmp/.bashrc", []string{"/home/u", "/home/u/.bashrc"}, true},
		{"ln --target-directory=d -s x", []string{"d", "d/x"}, true},
		{"ln -S .bak -s x y", []string{"y"}, true},
		{"link a b", []string{"b"}, true},
		{"cp a b", nil, false},
		{"rm .bashrc", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.line)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %v, %v", tt.line, cmds, err)
			}
			got, ok := WrittenOperands(cmds[len(cmds)-1], isDir)
			if ok != tt.modeled || (ok && !slices.Equal(got, tt.want)) {
				t.Errorf("WrittenOperands = %q, %v; want %q, %v", got, ok, tt.want, tt.modeled)
			}
		})
	}
}

// TestWrittenOperands_Gh pins the gh subcommands that write no file (U18-WS3):
// creating, commenting on, editing, viewing or listing a pull request or
// issue takes text (--title, --body) and numbers, not paths. A subcommand
// word built from an expansion, an assignment that could change what gh
// does, and every other subcommand are not modelled.
func TestWrittenOperands_Gh(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line    string
		modeled bool
	}{
		{`gh pr create --title x --body "$(cat <<'EOF'` + "\nSee .qsdev.yaml\nEOF\n)\"", true},
		{"gh issue comment 1 --body x", true},
		{"gh pr view 5", true},
		{"gh pr edit 5 --title y", true},
		{"gh issue list", true},
		{"gh $x create", false},
		{`gh "pr$x" create`, false},
		{"gh pr $sub", false},
		{"GH_HOST=x gh pr create", false},
		{"gh release download", false},
		{"gh pr checkout 5", false},
		{"gh api repos/o/r", false},
		{"gh pr", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.line, err)
			}
			i := slices.IndexFunc(cmds, func(c Command) bool { return c.Name == "gh" })
			if i < 0 {
				t.Fatalf("Parse(%q) = %+v, no gh command", tt.line, cmds)
			}
			got, ok := WrittenOperands(cmds[i], func(string) bool { return true })
			if ok != tt.modeled || got != nil {
				t.Errorf("WrittenOperands = %q, %v; want nil, %v", got, ok, tt.modeled)
			}
		})
	}
}

// TestRedefinitions pins which command names a line may give another
// meaning (U18-WS3 review): an alias or function it defines anywhere on the
// line, and any name when an alias command uses an expansion.
func TestRedefinitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		line       string
		redefined  []string
		notDefined []string
	}{
		{line: "alias gh='rm -f'; gh pr create .npmrc", redefined: []string{"gh"}, notDefined: []string{"rm", "cat"}},
		{line: "shopt -s expand_aliases\nalias gh='tee -a'\necho x | gh pr create .npmrc", redefined: []string{"gh"}, notDefined: []string{"tee"}},
		{line: "builtin alias cat='tee -a' ll='ls -l'", redefined: []string{"cat", "ll"}, notDefined: []string{"gh"}},
		{line: `gh() { command rm -f "$@"; }; gh pr create .npmrc`, redefined: []string{"gh"}, notDefined: []string{"rm"}},
		{line: "gh pr create .npmrc; alias gh=rm", redefined: []string{"gh"}},
		{line: `alias "$n"='rm -f'`, redefined: []string{"gh", "cat"}},
		// A sourced script may define any name; the one source command
		// still names itself.
		{line: ". ./defs.sh; gh pr create .npmrc", redefined: []string{"gh", "cat"}, notDefined: []string{".", "source"}},
		{line: "source x && cat .npmrc", redefined: []string{"gh", "cat"}, notDefined: []string{"source"}},
		{line: "builtin source x; gh pr view 1", redefined: []string{"gh"}},
		{line: ". a; source .npmrc", redefined: []string{"source", ".", "gh"}},
		{line: "source .npmrc", redefined: []string{"gh"}, notDefined: []string{"source", "."}},
		{line: "git add .", notDefined: []string{"gh", "cat"}},
		{line: "alias", notDefined: []string{"gh"}},
		{line: "alias gh", notDefined: []string{"gh"}},
		{line: "gh pr create --title alias=x", notDefined: []string{"gh", "alias"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.line, err)
			}
			redefined := Redefinitions(cmds)
			for _, name := range tt.redefined {
				if !redefined(name) {
					t.Errorf("Redefinitions(%q)(%q) = false, want true", tt.line, name)
				}
			}
			for _, name := range tt.notDefined {
				if redefined(name) {
					t.Errorf("Redefinitions(%q)(%q) = true, want false", tt.line, name)
				}
			}
		})
	}
}
