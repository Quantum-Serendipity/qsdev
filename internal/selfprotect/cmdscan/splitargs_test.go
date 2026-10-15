package cmdscan

import (
	"slices"
	"testing"
)

// TestParse_ExpandedAndSplitArgs pins which argument words hold an
// expansion (ExpandedArgIndexes) and which of them the shell may turn into
// several words or none (SplitArgs, maySplit).
func TestParse_ExpandedAndSplitArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command         string
		expanded, split []int
	}{
		{`git commit -am "$MSG"`, []int{2}, nil},
		{`git commit -am "fix: $MSG ($(date))"`, []int{2}, nil},
		{`git commit -m "$((1+2))" "${!ref}" "${#x}"`, []int{2, 3, 4}, nil},
		{`git commit $'a\tb'`, nil, nil},
		{`git commit -am $MSG`, []int{2}, []int{2}},
		{`git commit -m x $(cat opts)`, []int{3}, []int{3}},
		{`git commit -m "$@"`, []int{2}, []int{2}},
		{`git commit -m "${a[@]}"`, []int{2}, []int{2}},
		{`git commit -m "${a[0]}"`, []int{2}, []int{2}},
		{`git commit -m "${!pre@}"`, []int{2}, []int{2}},
		{`git commit -m "${x:-"$@"}"`, []int{2}, []int{2}},
		{`git commit -m "${x/a/b}"`, []int{2}, []int{2}},
		{`git commit -m x"$Y"$Z`, []int{2}, []int{2}},
		{`git commit -F <(echo x)`, []int{2}, []int{2}},
		{`git commit -m 'literal $X'`, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v", tt.command, cmds, err)
			}
			c := cmds[0]
			if !slices.Equal(c.ExpandedArgIndexes, tt.expanded) {
				t.Errorf("Parse(%q).ExpandedArgIndexes = %v, want %v", tt.command, c.ExpandedArgIndexes, tt.expanded)
			}
			if !slices.Equal(c.SplitArgs, tt.split) {
				t.Errorf("Parse(%q).SplitArgs = %v, want %v", tt.command, c.SplitArgs, tt.split)
			}
			if len(c.ExpandedArgIndexes) != len(c.ExpandedArgs) {
				t.Errorf("Parse(%q): %d expanded indexes for %d expanded args", tt.command, len(c.ExpandedArgIndexes), len(c.ExpandedArgs))
			}
		})
	}
}

// TestParse_ExpandedArgPrefixes pins the literal text before the first
// expansion of each expanded argument word (ExpandedArgPrefixes), which
// tells a value attached to its option (`-m"$MSG"`) from an expanded option.
func TestParse_ExpandedArgPrefixes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command  string
		prefixes []string
	}{
		{`git commit -am "$MSG"`, []string{""}},
		{`git commit -m"$MSG" -a`, []string{"-m"}},
		{`git commit --message="$MSG" -a`, []string{"--message="}},
		{`git commit "--message=$MSG"`, []string{"--message="}},
		{`git commit -m$MSG`, []string{"-m"}},
		{`git commit -m"a\"b$X"c`, []string{`-ma"b`}},
		{`git commit --mess"$X"=y`, []string{"--mess"}},
		{`git commit -m'lit'$'\t'"$X"`, []string{"-mlit\t"}},
		{`git commit -m "$(date)" --author=$'\x2e'"${W}"`, []string{"", "--author=."}},
		{`git commit -m 'literal $X'`, nil},
		{`git commit -C HEAD -$'\x00m'"$X"`, []string{"-"}},
		{`git commit .claude$'\x00'x`, []string{".claude"}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v", tt.command, cmds, err)
			}
			i := slices.IndexFunc(cmds, func(c Command) bool { return c.Name == "git" })
			if i < 0 {
				t.Fatalf("Parse(%q) = %+v, want a git command", tt.command, cmds)
			}
			c := cmds[i]
			if !slices.Equal(c.ExpandedArgPrefixes, tt.prefixes) {
				t.Errorf("Parse(%q).ExpandedArgPrefixes = %q, want %q", tt.command, c.ExpandedArgPrefixes, tt.prefixes)
			}
		})
	}
}
