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
