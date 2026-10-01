package cmdscan

import (
	"slices"
	"testing"
)

func TestCommandWordIndexes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  []int
	}{
		{"empty", nil, nil},
		{"plain program", []string{"python3", "x.py"}, []int{0}},
		{"wrapper", []string{"timeout", "30", "x.py"}, []int{0, 1, 2}},
		{"wrapper by path", []string{"/usr/bin/env", "x.py"}, []int{0, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CommandWordIndexes(tt.words); !slices.Equal(got, tt.want) {
				t.Errorf("CommandWordIndexes(%q) = %v, want %v", tt.words, got, tt.want)
			}
		})
	}
}

func TestProgramWordIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		words []string
		want  int
	}{
		{"empty", nil, -1},
		{"plain program", []string{"python3", "x.py"}, 0},
		{"timeout duration", []string{"timeout", "30", "python3", "x.py"}, 2},
		{"timeout option with argument", []string{"timeout", "-s", "KILL", "1.5m", "python3", "x.py"}, 4},
		{"env assignment", []string{"env", "PYTHONSAFEPATH=1", "python3", "x.py"}, 2},
		{"env by path with flag", []string{"/usr/bin/env", "-i", "A=1", "x.py"}, 3},
		{"wrapper chain", []string{"env", "A=1", "nice", "-n", "5", "nohup", "x.py", "arg"}, 6},
		{"double dash", []string{"nice", "--", "-weird"}, 2},
		{"sudo user", []string{"sudo", "-u", "root", "x.py"}, 3},
		{"exec", []string{"exec", "x.py"}, 1},
		{"env split string", []string{"env", "-S", "python3 -u", "x.py"}, -1},
		{"wrapper only", []string{"timeout", "30"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ProgramWordIndex(tt.words); got != tt.want {
				t.Errorf("ProgramWordIndex(%q) = %d, want %d", tt.words, got, tt.want)
			}
		})
	}
}

func TestShellScript(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		words  []string
		want   string
		wantOK bool
	}{
		{"sh -c", []string{"sh", "-c", "echo hi"}, "echo hi", true},
		{"bash -ec", []string{"bash", "-ec", "true"}, "true", true},
		{"through a wrapper", []string{"timeout", "5", "bash", "-c", "true"}, "true", true},
		{"eval", []string{"eval", "echo", "hi"}, "echo hi", true},
		{"script file", []string{"sh", "x.sh"}, "", false},
		{"not a shell", []string{"python3", "-c", "print()"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := ShellScript(tt.words)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("ShellScript(%q) = %q, %v; want %q, %v", tt.words, got, ok, tt.want, tt.wantOK)
			}
		})
	}
	if !IsScriptShell("/bin/bash") || IsScriptShell("python3") {
		t.Error("IsScriptShell misclassifies bash or python3")
	}
}
