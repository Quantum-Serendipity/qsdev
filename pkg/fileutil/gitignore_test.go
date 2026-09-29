package fileutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestEnsureGitignoreEntry(t *testing.T) {
	t.Parallel()

	comment := fileutil.GitignoreSectionComment()
	tests := []struct {
		name     string
		existing *string // nil means no .gitignore
		entry    string
		want     string
	}{
		{"creates file", nil, ".qsdev.local.yaml", comment + "\n.qsdev.local.yaml\n"},
		{"appends with section", ptr("node_modules/\n"), ".qsdev.local.yaml", "node_modules/\n\n" + comment + "\n.qsdev.local.yaml\n"},
		{"adds missing trailing newline", ptr("dist"), ".devinit/", "dist\n\n" + comment + "\n.devinit/\n"},
		{"exact entry is a no-op", ptr(".qsdev.local.yaml\n"), ".qsdev.local.yaml", ".qsdev.local.yaml\n"},
		{"root-anchored equivalent is a no-op", ptr("/.qsdev.local.yaml\n"), ".qsdev.local.yaml", "/.qsdev.local.yaml\n"},
		{"slash-less directory equivalent is a no-op", ptr(".devinit\n"), ".devinit/", ".devinit\n"},
		{"negation is not equivalent", ptr("!.devinit/\n"), ".devinit/", "!.devinit/\n\n" + comment + "\n.devinit/\n"},
		{"section comment added once", ptr(comment + "\n.qsdev.local.yaml\n"), ".devinit/", comment + "\n.qsdev.local.yaml\n.devinit/\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitignore")
			if tt.existing != nil {
				if err := os.WriteFile(path, []byte(*tt.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := fileutil.EnsureGitignoreEntry(dir, tt.entry); err != nil {
				t.Fatalf("EnsureGitignoreEntry: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf(".gitignore = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitignoreSectionComment_UsesBranding(t *testing.T) {
	t.Parallel()
	if got := fileutil.GitignoreSectionComment(); !strings.HasPrefix(got, "# ") || !strings.Contains(got, "local configuration") {
		t.Errorf("GitignoreSectionComment() = %q", got)
	}
}

func TestEnsureGitignoreEntry_WrapsReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory named .gitignore makes ReadFile fail with a non-NotExist error.
	if err := os.Mkdir(filepath.Join(dir, ".gitignore"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := fileutil.EnsureGitignoreEntry(dir, ".devinit/")
	if err == nil || !strings.Contains(err.Error(), ".gitignore") {
		t.Errorf("error = %v, want wrapped error naming .gitignore", err)
	}
}

func ptr(s string) *string { return &s }
