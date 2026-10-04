package generate_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestWriteFiles_CRLFCheckout covers an existing file holding the generated
// content as a CRLF checkout (Git's core.autocrlf, the Git for Windows
// default) under every strategy: it is that content, so it is neither
// rewritten nor reported as a local change, and state records the generated
// content. A script whose interpreter line would then end in CR is rewritten
// with LF where the kernel cannot start it.
func TestWriteFiles_CRLFCheckout(t *testing.T) {
	t.Parallel()
	const text = "line one\nline two\n"
	const script = "#!/usr/bin/env bash\necho hi\n"
	scriptAction := generate.ActionUpdated
	if runtime.GOOS == "windows" {
		scriptAction = generate.ActionUnchanged // Git Bash tolerates the CR
	}
	tests := []struct {
		name       string
		path       string
		strategy   types.MergeStrategy
		generated  string
		wantAction generate.FileAction
	}{
		{"overwrite", "o.txt", types.Overwrite, text, generate.ActionUnchanged},
		{"library-managed", "l.txt", types.LibraryManaged, text, generate.ActionUnchanged},
		{"skip", "s.txt", types.Skip, text, generate.ActionUnchanged},
		{"manual merge", "m.txt", types.ManualMerge, text, generate.ActionUnchanged},
		{"section marker", "CLAUDE.md", types.SectionMarker, "<!-- BEGIN GENERATED SECTION -->\nx\n<!-- END GENERATED SECTION -->\n", generate.ActionUnchanged},
		{"three-way merge", "settings.json", types.ThreeWayMerge, "{\n  \"a\": 1\n}\n", generate.ActionUnchanged},
		{"script", "hook.sh", types.Overwrite, script, scriptAction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, tt.path)
			crlf := strings.ReplaceAll(tt.generated, "\n", "\r\n")
			if err := os.WriteFile(path, []byte(crlf), 0o644); err != nil {
				t.Fatal(err)
			}
			file := types.GeneratedFile{Path: tt.path, Content: []byte(tt.generated), Strategy: tt.strategy, Mode: 0o644}
			result, err := generate.WriteFiles([]types.GeneratedFile{file}, generate.PipelineOptions{ProjectRoot: dir})
			if err != nil {
				t.Fatalf("WriteFiles: %v", err)
			}
			fr := result.Files[0]
			if fr.Action != tt.wantAction || fr.SidecarPath != "" {
				t.Fatalf("action = %v (sidecar %q, error %v), want %v", fr.Action, fr.SidecarPath, fr.Error, tt.wantAction)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := crlf
			if tt.wantAction == generate.ActionUpdated {
				want = tt.generated
			}
			if string(got) != want {
				t.Errorf("file = %q, want %q", got, want)
			}
			recorded := state.RecordFiles(result.SuccessfulFiles([]types.GeneratedFile{file}))
			if h := recorded.Files[tt.path].Hash; h != state.ComputeHash(file.Content) {
				t.Errorf("recorded hash %s, want the generated content's", h)
			}
		})
	}
}

// TestWriteFiles_PinsScriptsToLF covers the .gitattributes lines that keep
// generated scripts checked out with LF line endings: one root-anchored,
// glob-escaped pattern per script, none for other files or a dry run.
func TestWriteFiles_PinsScriptsToLF(t *testing.T) {
	t.Parallel()
	files := []types.GeneratedFile{
		{Path: ".claude/hooks/guard.py", Content: []byte("#!/usr/bin/env python3\n"), Mode: 0o755},
		{Path: ".envrc", Content: []byte("#!/usr/bin/env bash\n"), Strategy: types.Skip},
		{Path: "odd[1].sh", Content: []byte("#!/bin/sh\n"), Mode: 0o755},
		{Path: "README.md", Content: []byte("# readme\n")},
	}

	dry := t.TempDir()
	if _, err := generate.WriteFiles(files, generate.PipelineOptions{ProjectRoot: dry, DryRun: true}); err != nil {
		t.Fatalf("WriteFiles (dry run): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dry, ".gitattributes")); err == nil {
		t.Error("a dry run wrote .gitattributes")
	}

	dir := t.TempDir()
	if _, err := generate.WriteFiles(files, generate.PipelineOptions{ProjectRoot: dir}); err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\n/.claude/hooks/guard.py text eol=lf\n",
		"\n/.envrc text eol=lf\n",
		"\n/odd\\[1].sh text eol=lf\n",
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf(".gitattributes lacks %q:\n%s", want, got)
		}
	}
	if bytes.Contains(got, []byte("README")) {
		t.Errorf(".gitattributes pins a file that is not a script:\n%s", got)
	}
}
