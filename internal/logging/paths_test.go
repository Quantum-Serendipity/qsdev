package logging

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWalkUp exercises the shared project-root traversal primitive directly:
// it must match in an ancestor, in the start dir itself, and report
// ("", false) when no ancestor matches.
func TestWalkUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	marker := filepath.Join(root, "MARK")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	match := func(d string) bool {
		_, err := os.Stat(filepath.Join(d, "MARK"))
		return err == nil
	}

	t.Run("found in ancestor", func(t *testing.T) {
		t.Parallel()
		got, ok := WalkUp(nested, match)
		if !ok || got != root {
			t.Errorf("WalkUp = (%q,%v), want (%q,true)", got, ok, root)
		}
	})

	t.Run("found in start dir", func(t *testing.T) {
		t.Parallel()
		got, ok := WalkUp(root, match)
		if !ok || got != root {
			t.Errorf("WalkUp = (%q,%v), want (%q,true)", got, ok, root)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, ok := WalkUp(nested, func(string) bool { return false })
		if ok {
			t.Error("WalkUp reported a match where none exists")
		}
	})
}

// TestWalkUpMarkerSetsStayDistinct proves the shared traversal is parameterised
// by the caller's marker predicate: the logging marker set (a .qsdev/ directory
// counts as a root) and the resolve marker set (only a .qsdev.yaml *file* counts
// as the strong marker) resolve the SAME tree differently, exactly as before the
// traversal was unified. A directory holding only ".qsdev/" is a root for the
// logging predicate but not for the resolve file predicate, and vice-versa for a
// directory holding only "go.mod".
func TestWalkUpMarkerSetsStayDistinct(t *testing.T) {
	t.Parallel()

	// logging-style predicate: .qsdev/ directory present (mirrors DetectProjectRoot).
	loggingMatch := func(d string) bool {
		info, err := os.Stat(filepath.Join(d, ".qsdev"))
		return err == nil && info.IsDir()
	}
	// resolve-style strong predicate: .qsdev.yaml regular file present.
	resolveFileMatch := func(d string) bool {
		info, err := os.Stat(filepath.Join(d, ".qsdev.yaml"))
		return err == nil && !info.IsDir()
	}

	t.Run("dot-qsdev dir is a logging root but not a resolve root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "sub")
		if err := os.MkdirAll(filepath.Join(root, ".qsdev"), 0o755); err != nil {
			t.Fatalf("mkdir .qsdev: %v", err)
		}
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}

		if got, ok := WalkUp(nested, loggingMatch); !ok || got != root {
			t.Errorf("logging predicate: WalkUp = (%q,%v), want (%q,true)", got, ok, root)
		}
		if _, ok := WalkUp(nested, resolveFileMatch); ok {
			t.Error("resolve file predicate unexpectedly matched a .qsdev directory")
		}
	})

	t.Run("qsdev yaml file is a resolve root and also a logging root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "sub")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir nested: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("qsdev_version: 1\n"), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}

		if got, ok := WalkUp(nested, resolveFileMatch); !ok || got != root {
			t.Errorf("resolve file predicate: WalkUp = (%q,%v), want (%q,true)", got, ok, root)
		}
	})
}

// TestClassifyInvocation covers the log-scope classification: shell completion,
// help and log browsing must not create sessions (each TAB press used to create
// one and evict real command logs), hooks are automated, and classification is
// by command-path prefix so subcommands inherit their parent's class.
func TestClassifyInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want CommandClass
	}{
		{"shell completion request", []string{"__complete", "lo"}, ClassUnlogged},
		{"shell completion without descriptions", []string{"__completeNoDesc", "in"}, ClassUnlogged},
		{"completion script", []string{"completion", "bash"}, ClassUnlogged},
		{"help command", []string{"help", "init"}, ClassUnlogged},
		{"help flag", []string{"init", "--help"}, ClassUnlogged},
		{"short help flag", []string{"enable", "-h"}, ClassUnlogged},
		{"bare root", nil, ClassUnlogged},
		{"root version flag", []string{"--version"}, ClassUnlogged},
		{"logs subcommand", []string{"logs", "list"}, ClassUnlogged},
		{"report subcommand", []string{"report", "bug"}, ClassGlobal},
		{"self-update with version flag", []string{"self-update", "--version", "v1.2.3"}, ClassGlobal},
		{"version", []string{"version"}, ClassGlobal},
		{"selfprotect hook", []string{"selfprotect", "--hook"}, ClassAutomated},
		{"enforce hook", []string{"enforce", "--hook", "pre-tool-use"}, ClassAutomated},
		{"sandbox exec", []string{"sandbox", "exec", "--", "ls"}, ClassAutomated},
		{"sandbox status", []string{"sandbox", "status"}, ClassProject},
		{"help after terminator is an argument", []string{"sandbox", "exec", "--", "tool", "--help"}, ClassAutomated},
		{"init", []string{"init", "--mode", "join"}, ClassProject},
		{"unknown command", []string{"doctor"}, ClassProject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyInvocation(tt.args); got != tt.want {
				t.Errorf("ClassifyInvocation(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
