package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// dockerignorePatterns returns the non-comment, non-blank lines of the
// repository's .dockerignore.
func dockerignorePatterns(t *testing.T, root string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".dockerignore"))
	if err != nil {
		t.Fatalf("reading .dockerignore: %v", err)
	}
	var patterns []string
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

// TestDockerignoreIsAllowlist is the W193 regression test. The gateway image
// build does `COPY . .`, so the build context must be an allowlist: a denylist
// kept missing gitignored local material (the .go/ module cache, bin/, the
// root binary, mcp.db and .qsdev/mcp-secrets.env). The re-included trees must
// still cover every package and embedded file `go build ./cmd/qsdev` needs,
// and secret patterns must be excluded again after the re-includes.
func TestDockerignoreIsAllowlist(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..")
	patterns := dockerignorePatterns(t, root)

	if len(patterns) == 0 || patterns[0] != "*" {
		t.Fatalf(".dockerignore must start by excluding everything (\"*\"), got %q", patterns)
	}

	lastInclude := -1
	included := map[string]bool{}
	for i, p := range patterns {
		if rest, ok := strings.CutPrefix(p, "!"); ok {
			lastInclude = i
			included[strings.TrimSuffix(rest, "/")] = true
		}
	}
	for _, secret := range []string{"**/.qsdev/", "**/.env", "**/*.key", "**/*.pem"} {
		if i := slices.Index(patterns, secret); i < lastInclude {
			t.Errorf("secret pattern %q must be excluded after the last re-include", secret)
		}
	}
	for _, f := range []string{"go.mod", "go.sum", "vendor"} {
		if !included[f] {
			t.Errorf(".dockerignore does not re-include %s", f)
		}
	}

	// Every top-level directory holding a package or embedded file of the
	// binary must be re-included, or the image build fails.
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(goBin, "list", "-deps", "-f",
		`{{.Dir}}{{range .EmbedFiles}}{{"\n"}}{{$.Dir}}/{{.}}{{end}}`, "./cmd/qsdev")
	cmd.Dir = absRoot
	// Resolve packages the way the image build does (build/docker/Dockerfile
	// sets GOWORK=off and GOFLAGS=-mod=vendor). Otherwise a module cache that
	// happens to live inside the checkout (devenv's .devenv/state/go) shows up
	// as a build input the image never reads.
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=vendor")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		rel, err := filepath.Rel(absRoot, line)
		if line == "" || err != nil || strings.HasPrefix(rel, "..") || rel == "." {
			continue // outside the module (stdlib, workspace modules)
		}
		top := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
		if !included[top] {
			t.Errorf("%s is needed by the build but %s/ is not re-included in .dockerignore", rel, top)
		}
	}
}
