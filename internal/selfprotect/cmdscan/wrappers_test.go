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
		{"windows wrapper", []string{`C:\Git\usr\bin\env.exe`, "x.py"}, []int{0, 1}},
		{"case-folded wrapper", []string{"ENV", "x.py"}, []int{0, 1}},
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
		{"env attached split string", []string{"env", "-vS", "python3 -u"}, -1},
		{"env abbreviated split string", []string{"env", "--split", "python3 -u"}, -1},
		{"env cluster with argument", []string{"env", "-iu", "HOME", "x.py"}, 3},
		{"env chdir then unset", []string{"env", "-C", "/tmp", "-u", "HOME", "x.py"}, 5},
		{"env abbreviated chdir", []string{"env", "--ch", "/tmp", "x.py"}, 3},
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

func TestInvokesProgram(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    bool
	}{
		{"qsdev init --update", true},
		{"HOME=/tmp/e ./bin/qsdev status", true},
		{`sh -c "qsdev claude update"`, true},
		{`C:\tools\QSDEV.exe status`, true},
		{"q''sdev status", true},
		{"go test ./...", false},
		{"cat qsdev.yaml", false},
		{"ls ~/.config/qsdevx", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			if got := InvokesProgram(tt.command, "qsdev"); got != tt.want {
				t.Errorf("InvokesProgram(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// TestWrapperOptions pins how a wrapper's options are read: each letter of a
// cluster, an argument from the rest of the cluster, after "=", or from the
// next word, and options ending at the first operand.
func TestWrapperOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want []WrapperOption
	}{
		{"unset", []string{"-u", "HOME", "x"}, []WrapperOption{{"-u", "HOME"}}},
		{"attached unset", []string{"-uHOME", "x"}, []WrapperOption{{"-u", "HOME"}}},
		{"cluster", []string{"-iu", "HOME", "x"}, []WrapperOption{{"-i", ""}, {"-u", "HOME"}}},
		{"chdir then unset", []string{"-C", "/tmp", "-u", "HOME"}, []WrapperOption{{"-C", "/tmp"}, {"-u", "HOME"}}},
		{"long with equals", []string{"--unset=HOME", "x"}, []WrapperOption{{"--unset", "HOME"}}},
		{"abbreviated long with argument", []string{"--un", "HOME", "x"}, []WrapperOption{{"--un", "HOME"}}},
		{"long without argument", []string{"--ignore-env", "x"}, []WrapperOption{{"--ignore-env", ""}}},
		{"lone dash", []string{"-", "x"}, []WrapperOption{{"-", ""}}},
		{"stops at operand", []string{"A=1", "-u", "HOME"}, nil},
		{"stops at double dash", []string{"--", "-u", "HOME"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := WrapperOptions("env", tt.args); !slices.Equal(got, tt.want) {
				t.Errorf("WrapperOptions(env, %q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestWrapperOptionIs pins option matching: short options exactly, long ones
// by any getopt_long abbreviation.
func TestWrapperOptionIs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		spellings []string
		want      bool
	}{
		{"-i", []string{"-i", "--ignore-environment"}, true},
		{"--ignore-env", []string{"-i", "--ignore-environment"}, true},
		{"--ignore-environment", []string{"--ignore-environment"}, true},
		{"--ignore-environments", []string{"--ignore-environment"}, false},
		{"--", []string{"--ignore-environment"}, false},
		{"-u", []string{"-i", "--ignore-environment"}, false},
		{"-", []string{"-"}, true},
	}
	for _, tt := range tests {
		if got := (WrapperOption{Name: tt.name}).Is(tt.spellings...); got != tt.want {
			t.Errorf("WrapperOption{%q}.Is(%q) = %v, want %v", tt.name, tt.spellings, got, tt.want)
		}
	}
}
