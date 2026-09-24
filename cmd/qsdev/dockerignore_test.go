package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		rel = filepath.ToSlash(rel)
		top := strings.SplitN(rel, "/", 2)[0]
		if !included[top] {
			t.Errorf("%s is needed by the build but %s/ is not re-included in .dockerignore", rel, top)
		}
		// A later exclusion (e.g. **/tls/) can still drop a nested package.
		// Probe a source file inside each package dir, and embedded files as-is.
		probe := rel
		if info, err := os.Stat(line); err == nil && info.IsDir() {
			probe = rel + "/x.go"
		}
		if dockerignoreExcludes(t, patterns, probe) {
			t.Errorf("%s is needed by the build but .dockerignore excludes it", rel)
		}
	}
}

// dockerignoreExcludes reports whether path (slash-separated, relative to the
// context root) is excluded, using Docker's rules: patterns apply in order,
// the last match wins, "!" re-includes, and a pattern matching a parent
// directory matches everything beneath it.
func dockerignoreExcludes(t *testing.T, patterns []string, path string) bool {
	t.Helper()
	excluded := false
	for _, p := range patterns {
		neg := strings.HasPrefix(p, "!")
		re := dockerignoreRegexp(t, strings.TrimSuffix(strings.TrimPrefix(p, "!"), "/"))
		parts := strings.Split(path, "/")
		for i := len(parts); i > 0; i-- {
			if re.MatchString(strings.Join(parts[:i], "/")) {
				excluded = !neg
				break
			}
		}
	}
	return excluded
}

// dockerignoreRegexp translates a .dockerignore glob into an anchored regexp:
// "**" spans directories, "*" and "?" stay within one path segment.
func dockerignoreRegexp(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		switch c := pattern[i]; {
		case strings.HasPrefix(pattern[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i++
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		t.Fatalf("translating .dockerignore pattern %q: %v", pattern, err)
	}
	return re
}

func TestDockerignoreExcludes(t *testing.T) {
	t.Parallel()
	patterns := []string{"*", "!vendor/", "!internal/", "**/tls/", "!vendor/**/tls/", "**/*.key"}
	tests := []struct {
		path string
		want bool
	}{
		{"vendor/github.com/google/certificate-transparency-go/tls/tls.go", false},
		{"internal/tls/server.go", true},
		{"vendor/example.com/tls/server.key", true},
		{"internal/config/parse.go", false},
		{"bin/qsdev", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := dockerignoreExcludes(t, patterns, tt.path); got != tt.want {
				t.Errorf("dockerignoreExcludes(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
