package shebang

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	t.Parallel()
	long := "#!/bin/" + strings.Repeat("x", MaxLen) + "\n"
	tests := []struct {
		name, content string
		want          Line
	}{
		{"env", "#!/usr/bin/env python3\nprint(1)\n", Line{"/usr/bin/env", "python3"}},
		{"env -S", "#!/usr/bin/env -S python3 -u\n", Line{"/usr/bin/env", "-S python3 -u"}},
		{"env assignment", "#!/usr/bin/env PYTHONSAFEPATH=1 python3\n", Line{"/usr/bin/env", "PYTHONSAFEPATH=1 python3"}},
		{"absolute with arg", "#!/bin/bash -e\n", Line{"/bin/bash", "-e"}},
		{"tab separator", "#!/bin/sh\t-x  \n", Line{"/bin/sh", "-x"}},
		{"space after bang", "#! /usr/bin/python3\n", Line{"/usr/bin/python3", ""}},
		// The kernel does not strip '\r': env looks up "python3\r".
		{"CRLF line ending", "#!/usr/bin/env python3\r\nprint(1)\r\n", Line{"/usr/bin/env", "python3\r"}},
		{"CRLF without argument", "#!/bin/sh\r\n", Line{"/bin/sh\r", ""}},
		{"argument with blanks", "#!/usr/bin/env python3 -u\n", Line{"/usr/bin/env", "python3 -u"}},
		{"no newline", "#!/bin/sh", Line{"/bin/sh", ""}},
		{"over 256 bytes", long, Line{long[2:MaxLen], ""}},
		{"blank interpreter line", "#!   \n", Line{}},
		{"no shebang", "print('x')\n", Line{}},
		{"empty", "", Line{}},
	}
	dir := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "-"))
			if err := os.WriteFile(p, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := Read(p)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tt.want {
				t.Errorf("Read(%q) = %+v, want %+v", tt.content, got, tt.want)
			}
		})
	}
}

func TestRead_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := Read(filepath.Join(t.TempDir(), "absent"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read(absent) error = %v, want fs.ErrNotExist", err)
	}
}

func TestLine_Program(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		line       Line
		want       string
		wantViaEnv bool
	}{
		{"env", Line{"/usr/bin/env", "python3"}, "python3", true},
		{"env -S", Line{"/usr/bin/env", "-S python3 -u"}, "python3", true},
		{"env VAR=x prog", Line{"/usr/bin/env", "PYTHONSAFEPATH=1 python3"}, "python3", true},
		{"env without program", Line{"/usr/bin/env", "-i"}, "", true},
		{"bare env", Line{"env", "node"}, "node", true},
		{"absolute", Line{"/bin/bash", "-e"}, "/bin/bash", false},
		{"interpreter named like env", Line{"/opt/envoy", "x"}, "/opt/envoy", false},
		{"CRLF interpreter", Line{"/bin/sh\r", ""}, "/bin/sh", false},
		{"CRLF env argument", Line{"/usr/bin/env", "python3\r"}, "python3", true},
		{"no shebang", Line{}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.line.Program(); got != tt.want {
				t.Errorf("%+v.Program() = %q, want %q", tt.line, got, tt.want)
			}
			if got := tt.line.ViaEnv(); got != tt.wantViaEnv {
				t.Errorf("%+v.ViaEnv() = %v, want %v", tt.line, got, tt.wantViaEnv)
			}
		})
	}
}

func TestLine_EnvProgram(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		line   Line
		goos   string
		want   string
		wantOK bool
	}{
		{"single word", Line{"/usr/bin/env", "python3"}, "linux", "python3", true},
		{"single word macOS", Line{"/usr/bin/env", "python3"}, "darwin", "python3", true},
		// Linux passes "python3 -u" as one word: env looks up that name.
		{"env with unsplit option", Line{"/usr/bin/env", "python3 -u"}, "linux", "python3 -u", true},
		{"env with option macOS", Line{"/usr/bin/env", "python3 -u"}, "darwin", "python3", true},
		{"env with unsplit option windows", Line{"/usr/bin/env", "python3 -u"}, "windows", "python3 -u", true},
		// "-i python3" is one option word to GNU env, which exits 125.
		{"env option without -S", Line{"/usr/bin/env", "-i python3"}, "linux", "", false},
		{"env option macOS", Line{"/usr/bin/env", "-i python3"}, "darwin", "python3", true},
		{"lone option", Line{"/usr/bin/env", "-i"}, "linux", "", false},
		{"-S", Line{"/usr/bin/env", "-S python3 -u"}, "linux", "python3", true},
		{"-S attached", Line{"/usr/bin/env", "-Spython3 -u"}, "linux", "python3", true},
		{"-vS", Line{"/usr/bin/env", "-vS python3"}, "linux", "python3", true},
		{"--split-string=", Line{"/usr/bin/env", "--split-string=PYTHONSAFEPATH=1 python3"}, "linux", "python3", true},
		{"-S assignment only", Line{"/usr/bin/env", "-S A=1"}, "linux", "", false},
		{"assignment is one word", Line{"/usr/bin/env", "PYTHONSAFEPATH=1 python3"}, "linux", "", false},
		{"assignment macOS", Line{"/usr/bin/env", "PYTHONSAFEPATH=1 python3"}, "darwin", "python3", true},
		{"CRLF linux", Line{"/usr/bin/env", "python3\r"}, "linux", "python3\r", true},
		{"CRLF windows", Line{"/usr/bin/env", "python3\r"}, "windows", "python3", true},
		{"no argument", Line{"/usr/bin/env", ""}, "linux", "", false},
		{"not env", Line{"/usr/bin/python3", ""}, "linux", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.line.EnvProgram(tt.goos)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("%+v.EnvProgram(%s) = %q, %v; want %q, %v", tt.line, tt.goos, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
