package fileutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestEnsureGitattributesLines(t *testing.T) {
	t.Parallel()

	comment := fileutil.GitattributesSectionComment()
	const pin = "/a.sh text eol=lf"
	tests := []struct {
		name     string
		existing *string // nil means no .gitattributes
		lines    []string
		want     *string // nil means no .gitattributes
	}{
		{"nothing to add creates no file", nil, nil, nil},
		{"creates file", nil, []string{pin, "/b.py text eol=lf"}, ptr(comment + "\n" + pin + "\n/b.py text eol=lf\n")},
		{"appends with section", ptr("* text=auto\n"), []string{pin}, ptr("* text=auto\n\n" + comment + "\n" + pin + "\n")},
		{"adds missing trailing newline", ptr("* text=auto"), []string{pin}, ptr("* text=auto\n\n" + comment + "\n" + pin + "\n")},
		{"present line is a no-op", ptr(pin + "\n"), []string{pin}, ptr(pin + "\n")},
		{"CRLF and spacing are the same line", ptr("/a.sh   text eol=lf\r\n"), []string{pin}, ptr("/a.sh   text eol=lf\r\n")},
		{"other attributes are not the line", ptr("/a.sh text eol=crlf\n"), []string{pin}, ptr("/a.sh text eol=crlf\n\n" + comment + "\n" + pin + "\n")},
		{"section comment added once", ptr(comment + "\n" + pin + "\n"), []string{"/b.py text eol=lf"}, ptr(comment + "\n" + pin + "\n/b.py text eol=lf\n")},
		{"duplicates added once", nil, []string{pin, pin}, ptr(comment + "\n" + pin + "\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, ".gitattributes")
			if tt.existing != nil {
				if err := os.WriteFile(path, []byte(*tt.existing), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := fileutil.EnsureGitattributesLines(dir, tt.lines); err != nil {
				t.Fatalf("EnsureGitattributesLines: %v", err)
			}
			got, err := os.ReadFile(path)
			switch {
			case tt.want == nil && err == nil:
				t.Errorf(".gitattributes created: %q", got)
			case tt.want == nil:
			case err != nil:
				t.Fatal(err)
			case string(got) != *tt.want:
				t.Errorf(".gitattributes = %q, want %q", got, *tt.want)
			}
		})
	}
}
