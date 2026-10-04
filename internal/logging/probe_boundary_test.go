package logging

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

	tests := []struct {
		name, wd, home, want string
	}{
		{"repository toplevel", repo, home, repo},
		{"repository subdirectory resolves to toplevel", sub, home, repo},
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
