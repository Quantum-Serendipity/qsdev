package toolreg

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// nixRelPathRe matches a relative Nix path literal (./foo, ../foo) that is not
// part of a longer token.
var nixRelPathRe = regexp.MustCompile(`(?:^|[\s(\[{=])(\.{1,2}/[A-Za-z0-9._+/-]+)`)

func TestNixRelPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		expr string
		want []string
	}{
		{`(pkgs.callPackage ./.opengrep/nix {})`, []string{".opengrep/nix"}},
		{`(import ./nix/foo.nix { inherit pkgs; })`, []string{"nix/foo.nix"}},
		{`pkgs.hello`, nil},
		{`(pkgs.callPackage ./a/ {}) ++ [ ./b ]`, []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			if got := nixRelPaths(tt.expr); !slices.Equal(got, tt.want) {
				t.Errorf("nixRelPaths(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

// nixRelPaths returns the project-relative paths a nix_expr references,
// cleaned (a trailing slash dropped, ./ removed).
func nixRelPaths(expr string) []string {
	var out []string
	for _, m := range nixRelPathRe.FindAllStringSubmatch(expr, -1) {
		out = append(out, path.Clean(m[1]))
	}
	return out
}

// TestNixExprProjectPathsAreGenerated guards against a catalog nix_expr that
// imports a project-relative path no generator writes: devenv.nix would then
// reference a file that exists only in the qsdev repository, and devenv
// evaluation fails in every downstream project that enables the tool. Each
// referenced path must be a file (or a directory's default.nix) produced by
// the tool's own GenerateFunc and exclusively owned by it, so enable writes it
// and disable removes it.
func TestNixExprProjectPathsAreGenerated(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	reg, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	for name, expr := range cat.ToolNixExprs() {
		paths := nixRelPaths(expr)
		if len(paths) == 0 {
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("tool %q not in registry", name)
			}
			if tool.GenerateFunc == nil {
				t.Fatalf("nix_expr %q references project paths %q but the tool generates no files", expr, paths)
			}
			files, err := tool.GenerateFunc(types.WizardAnswers{EnabledTools: map[string]bool{name: true}})
			if err != nil {
				t.Fatalf("GenerateFunc: %v", err)
			}
			for _, p := range paths {
				assertNixPathGenerated(t, tool, files, p)
			}
		})
	}
}

func assertNixPathGenerated(t *testing.T, tool *Tool, files []types.GeneratedFile, p string) {
	t.Helper()
	if strings.HasPrefix(p, "../") {
		t.Errorf("nix_expr path %q escapes the project directory", p)
		return
	}
	for _, f := range files {
		if f.Path != p && f.Path != path.Join(p, "default.nix") {
			continue
		}
		if len(f.Content) == 0 {
			t.Errorf("generated %s is empty", f.Path)
		}
		if !tool.OwnsExclusively(f.Path) {
			t.Errorf("generated %s is not an exclusive owned_files entry, so disable would leave it behind", f.Path)
		}
		return
	}
	t.Errorf("nix_expr path ./%s is not produced by the tool's GenerateFunc (neither %s nor %s/default.nix)", p, p, p)
}
