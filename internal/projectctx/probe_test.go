package projectctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProbeBoundary(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	mkdir := func(parts ...string) string {
		t.Helper()
		dir := filepath.Join(append([]string{base}, parts...)...)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	home := mkdir("home")
	repo := mkdir("home", "repos", "app")
	mkdir("home", "repos", "app", ".git")
	sub := mkdir("home", "repos", "app", "web", "src")
	loose := mkdir("home", "scratch")
	dotHome := mkdir("dothome")
	mkdir("dothome", ".git")
	dotSub := mkdir("dothome", "notes")
	// A qsdev marker below the toplevel does not end the probe walk: version
	// probes are bounded by the repository, not by the project.
	mkdir("home", "repos", "app", "svc", ".devinit")
	markedSub := mkdir("home", "repos", "app", "svc", "cmd")

	tests := []struct {
		name, wd, home, want string
	}{
		{"repository toplevel", repo, home, repo},
		{"repository subdirectory resolves to toplevel", sub, home, repo},
		{"project marker below toplevel is passed over", markedSub, home, repo},
		{"outside a repository uses the working directory", loose, home, loose},
		{"home itself is no project", home, home, ""},
		{"above home is no project", base, home, ""},
		{"dotfiles repository at home is no project", dotSub, dotHome, ""},
		{"unknown home keeps the boundary", sub, "", repo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := probeBoundary(tt.wd, tt.home); got != tt.want {
				t.Errorf("probeBoundary(%q, %q) = %q, want %q", tt.wd, tt.home, got, tt.want)
			}
		})
	}
}

// TestProbeBoundary_StatCallsPerLevel bounds the probe's filesystem calls on
// every OS: one stat and one lstat per level of the repository walk, plus one
// stat per level of the home walk.
func TestProbeBoundary_StatCallsPerLevel(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "repo/.git", "repo/a/b/c", "home")
	wd := filepath.Join(root, "repo", "a", "b", "c")
	home := filepath.Join(root, "home")
	levels := 1
	for d := home; filepath.Dir(d) != d; d = filepath.Dir(d) {
		levels++
	}
	calls := countSeams(t)
	if got := probeBoundary(wd, home); got != filepath.Join(root, "repo") {
		t.Fatalf("probeBoundary = %q, want the repository", got)
	}
	// Repository walk: 4 levels (c, b, a, repo) at 2 calls each; home walk:
	// one stat of the boundary plus one per level from home up.
	if limit := 2*4 + 1 + levels; *calls > limit {
		t.Errorf("probeBoundary made %d stat/lstat calls, want <= %d", *calls, limit)
	}
}
