package generate

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestLFAttributeLines covers which generated files are pinned to LF line
// endings: scripts (content starting with "#!") and files that ask for it
// with PinLF, each as one root-anchored, glob-escaped pattern. A path that
// leaves the project or holds whitespace gets no pattern.
func TestLFAttributeLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file types.GeneratedFile
		want []string
	}{
		{"script", types.GeneratedFile{Path: ".claude/hooks/guard.py", Content: []byte("#!/usr/bin/env python3\n")},
			[]string{"/.claude/hooks/guard.py text eol=lf"}},
		{"PinLF without shebang", types.GeneratedFile{Path: ".github/nix-release.env", Content: []byte("NIX_VERSION=2.0.0\n"), PinLF: true},
			[]string{"/.github/nix-release.env text eol=lf"}},
		{"PinLF script is pinned once", types.GeneratedFile{Path: "run.sh", Content: []byte("#!/bin/sh\n"), PinLF: true},
			[]string{"/run.sh text eol=lf"}},
		{"plain file", types.GeneratedFile{Path: "README.md", Content: []byte("# readme\n")}, nil},
		{"glob characters escaped", types.GeneratedFile{Path: "odd[1]*?.env", Content: []byte("A=1\n"), PinLF: true},
			[]string{`/odd\[1]\*\?.env text eol=lf`}},
		{"backslash path normalised", types.GeneratedFile{Path: `.github\nix-release.env`, Content: []byte("A=1\n"), PinLF: true},
			[]string{"/.github/nix-release.env text eol=lf"}},
		{"non-local path", types.GeneratedFile{Path: "../outside.env", Content: []byte("A=1\n"), PinLF: true}, nil},
		{"whitespace path", types.GeneratedFile{Path: "my pins.env", Content: []byte("A=1\n"), PinLF: true}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := lfAttributeLines([]types.GeneratedFile{tt.file}); !slices.Equal(got, tt.want) {
				t.Errorf("lfAttributeLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestWriteGeneratedFile_PinsLF checks the single-file writer pins a PinLF
// file through the same lines as WriteFiles.
func TestWriteGeneratedFile_PinsLF(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f := types.GeneratedFile{Path: ".github/nix-release.env", Content: []byte("NIX_VERSION=2.0.0\n"), PinLF: true}
	if err := WriteGeneratedFile(dir, f); err != nil {
		t.Fatalf("WriteGeneratedFile: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "/.github/nix-release.env text eol=lf\n") {
		t.Errorf(".gitattributes does not pin the env file:\n%s", got)
	}
}
