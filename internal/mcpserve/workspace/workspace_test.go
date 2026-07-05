package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeWS writes content to root/rel, creating parent directories.
func writeWS(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Parser tests -----------------------------------------------------------

func TestParseNpmWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("array form with negation", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "package.json", `{"name":"root","workspaces":["packages/*","!packages/ignored"]}`)
		inc, exc, err := ParseNpmWorkspaces(root)
		if err != nil {
			t.Fatalf("ParseNpmWorkspaces: %v", err)
		}
		if !equalStrings(inc, []string{"packages/*"}) {
			t.Errorf("includes = %v, want [packages/*]", inc)
		}
		if !equalStrings(exc, []string{"packages/ignored"}) {
			t.Errorf("excludes = %v, want [packages/ignored]", exc)
		}
	})

	t.Run("yarn object form", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "package.json", `{"name":"root","workspaces":{"packages":["apps/*","libs/*"],"nohoist":["**/x"]}}`)
		inc, _, err := ParseNpmWorkspaces(root)
		if err != nil {
			t.Fatalf("ParseNpmWorkspaces: %v", err)
		}
		if !equalStrings(inc, []string{"apps/*", "libs/*"}) {
			t.Errorf("includes = %v, want [apps/* libs/*]", inc)
		}
	})
}

func TestParsePnpmWorkspaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWS(t, root, "pnpm-workspace.yaml", "packages:\n  - 'packages/*'\n  - '!packages/excluded'\n")
	inc, exc, err := ParsePnpmWorkspaces(root)
	if err != nil {
		t.Fatalf("ParsePnpmWorkspaces: %v", err)
	}
	if !equalStrings(inc, []string{"packages/*"}) {
		t.Errorf("includes = %v, want [packages/*]", inc)
	}
	if !equalStrings(exc, []string{"packages/excluded"}) {
		t.Errorf("excludes = %v, want [packages/excluded]", exc)
	}
}

func TestParseCargoWorkspaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWS(t, root, "Cargo.toml", "[workspace]\nmembers = [\"crates/*\", \"apps/cli\"]\nexclude = [\"crates/legacy\"]\n")
	inc, exc, err := ParseCargoWorkspaces(root)
	if err != nil {
		t.Fatalf("ParseCargoWorkspaces: %v", err)
	}
	if !equalStrings(inc, []string{"crates/*", "apps/cli"}) {
		t.Errorf("includes = %v, want [crates/* apps/cli]", inc)
	}
	if !equalStrings(exc, []string{"crates/legacy"}) {
		t.Errorf("excludes = %v, want [crates/legacy]", exc)
	}
}

func TestParseGoWorkspaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWS(t, root, "go.work", "go 1.21\n\nuse (\n\t./services/api\n\t./services/worker\n)\n")
	inc, exc, err := ParseGoWorkspaces(root)
	if err != nil {
		t.Fatalf("ParseGoWorkspaces: %v", err)
	}
	if !equalStrings(inc, []string{"./services/api", "./services/worker"}) {
		t.Errorf("includes = %v, want [./services/api ./services/worker]", inc)
	}
	if len(exc) != 0 {
		t.Errorf("excludes = %v, want empty", exc)
	}
}

func TestParseUvWorkspaces(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWS(t, root, "pyproject.toml", "[project]\nname = \"root\"\n\n[tool.uv.workspace]\nmembers = [\"packages/*\"]\nexclude = [\"packages/old\"]\n")
	inc, exc, err := ParseUvWorkspaces(root)
	if err != nil {
		t.Fatalf("ParseUvWorkspaces: %v", err)
	}
	if !equalStrings(inc, []string{"packages/*"}) {
		t.Errorf("includes = %v, want [packages/*]", inc)
	}
	if !equalStrings(exc, []string{"packages/old"}) {
		t.Errorf("excludes = %v, want [packages/old]", exc)
	}
}

// --- GlobResolver tests -----------------------------------------------------

func TestGlobResolver(t *testing.T) {
	t.Parallel()

	t.Run("single-star and directory filtering", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "packages/a/package.json", `{}`)
		writeWS(t, root, "packages/b/package.json", `{}`)
		writeWS(t, root, "packages/README.md", `notes`) // a file, must be ignored
		dirs, err := NewGlobResolver().ResolvePatterns(root, []string{"packages/*"}, nil)
		if err != nil {
			t.Fatalf("ResolvePatterns: %v", err)
		}
		if !equalStrings(dirs, []string{"packages/a", "packages/b"}) {
			t.Errorf("dirs = %v, want [packages/a packages/b]", dirs)
		}
	})

	t.Run("doublestar recursive **", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "packages/a/.keep", ``)
		writeWS(t, root, "packages/group/b/.keep", ``)
		dirs, err := NewGlobResolver().ResolvePatterns(root, []string{"packages/**"}, nil)
		if err != nil {
			t.Fatalf("ResolvePatterns: %v", err)
		}
		// ** matches packages itself and every descendant directory.
		for _, want := range []string{"packages", "packages/a", "packages/group", "packages/group/b"} {
			if !contains(dirs, want) {
				t.Errorf("recursive match missing %q in %v", want, dirs)
			}
		}
	})

	t.Run("brace expansion", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "apps/web/.keep", ``)
		writeWS(t, root, "libs/util/.keep", ``)
		writeWS(t, root, "other/x/.keep", ``)
		dirs, err := NewGlobResolver().ResolvePatterns(root, []string{"{apps,libs}/*"}, nil)
		if err != nil {
			t.Fatalf("ResolvePatterns: %v", err)
		}
		if !equalStrings(dirs, []string{"apps/web", "libs/util"}) {
			t.Errorf("dirs = %v, want [apps/web libs/util]", dirs)
		}
	})

	t.Run("pnpm bang negation in includes", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "packages/keep/.keep", ``)
		writeWS(t, root, "packages/drop/.keep", ``)
		dirs, err := NewGlobResolver().ResolvePatterns(root, []string{"packages/*", "!packages/drop"}, nil)
		if err != nil {
			t.Fatalf("ResolvePatterns: %v", err)
		}
		if !equalStrings(dirs, []string{"packages/keep"}) {
			t.Errorf("dirs = %v, want [packages/keep]", dirs)
		}
	})

	t.Run("cargo prefix-match exclusion", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeWS(t, root, "crates/a/.keep", ``)
		writeWS(t, root, "crates/legacy/.keep", ``)
		writeWS(t, root, "crates/legacy/sub/.keep", ``)
		// excluding "crates/legacy" must also drop its subtree.
		dirs, err := NewGlobResolver().ResolvePatterns(root,
			[]string{"crates/*", "crates/legacy/*"}, []string{"crates/legacy"})
		if err != nil {
			t.Fatalf("ResolvePatterns: %v", err)
		}
		if !equalStrings(dirs, []string{"crates/a"}) {
			t.Errorf("dirs = %v, want [crates/a] (legacy subtree excluded)", dirs)
		}
	})
}

// --- DetectWorkspaces -------------------------------------------------------

func TestDetectWorkspacesMultiEcosystem(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// npm side.
	writeWS(t, root, "package.json", `{"name":"root","workspaces":["packages/*"]}`)
	writeWS(t, root, "packages/web/package.json", `{"name":"@org/web","dependencies":{"react":"18"}}`)
	writeWS(t, root, "packages/empty/README.md", `no manifest here`) // resolved dir w/o manifest -> dropped

	// Go side.
	writeWS(t, root, "go.work", "go 1.21\n\nuse ./services/api\n")
	writeWS(t, root, "services/api/go.mod", "module example.com/api\n\ngo 1.21\n\nrequire example.com/lib v1.2.3\n")

	graph, err := DetectWorkspaces(root)
	if err != nil {
		t.Fatalf("DetectWorkspaces: %v", err)
	}
	if graph.Len() != 2 {
		t.Fatalf("graph has %d packages, want 2: %v", graph.Len(), graph.Packages())
	}

	web := graph.ByRelDir("packages/web")
	if web == nil {
		t.Fatal("packages/web not in graph")
	}
	if web.Ecosystem != ecoNpm || web.Name != "@org/web" {
		t.Errorf("web = {%s %s}, want {npm @org/web}", web.Ecosystem, web.Name)
	}
	if !contains(web.Dependencies, "react") {
		t.Errorf("web deps %v should include react", web.Dependencies)
	}
	if web.ManifestPath != "packages/web/package.json" {
		t.Errorf("web manifest = %q", web.ManifestPath)
	}

	api := graph.ByRelDir("services/api")
	if api == nil {
		t.Fatal("services/api not in graph")
	}
	if api.Ecosystem != ecoGo || api.Name != "example.com/api" {
		t.Errorf("api = {%s %s}, want {go example.com/api}", api.Ecosystem, api.Name)
	}
	if !contains(api.Dependencies, "example.com/lib") {
		t.Errorf("api deps %v should include example.com/lib", api.Dependencies)
	}
}

func TestDetectWorkspacesPnpmConfigSpelling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		configFile string
	}{
		{name: "canonical yaml", configFile: "pnpm-workspace.yaml"},
		{name: "legacy yml only", configFile: "pnpm-workspace.yml"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			// The workspace's only membership file uses the spelling under test;
			// the .yml-only case must still be detected as a pnpm workspace.
			writeWS(t, root, tc.configFile, "packages:\n  - 'packages/*'\n")
			writeWS(t, root, "packages/web/package.json", `{"name":"@org/web","dependencies":{"react":"18"}}`)

			graph, err := DetectWorkspaces(root)
			if err != nil {
				t.Fatalf("DetectWorkspaces: %v", err)
			}
			web := graph.ByRelDir("packages/web")
			if web == nil {
				t.Fatalf("packages/web not detected for %s-only workspace: %v", tc.configFile, graph.Packages())
			}
			if web.Ecosystem != ecoPnpm {
				t.Errorf("ecosystem = %q, want %q", web.Ecosystem, ecoPnpm)
			}
			if web.Name != "@org/web" {
				t.Errorf("name = %q, want @org/web", web.Name)
			}
			if !contains(web.Dependencies, "react") {
				t.Errorf("deps %v should include react", web.Dependencies)
			}
		})
	}
}

func TestDetectWorkspacesFailureIsolation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A malformed npm config must not block the valid Go workspace.
	writeWS(t, root, "package.json", `{ this is not valid json `)
	writeWS(t, root, "go.work", "go 1.21\n\nuse ./svc\n")
	writeWS(t, root, "svc/go.mod", "module example.com/svc\n\ngo 1.21\n")

	graph, err := DetectWorkspaces(root)
	if err != nil {
		t.Fatalf("DetectWorkspaces: %v", err)
	}
	if graph.ByRelDir("svc") == nil {
		t.Errorf("valid go workspace dropped because npm parser failed: %v", graph.Packages())
	}
}

// --- Graph API & V1 compat --------------------------------------------------

func TestQualifiedNameCollision(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// A Go module and an npm package both named "utils".
	writeWS(t, root, "package.json", `{"name":"root","workspaces":["js/*"]}`)
	writeWS(t, root, "js/utils/package.json", `{"name":"utils"}`)
	writeWS(t, root, "go.work", "go 1.21\n\nuse ./go/utils\n")
	writeWS(t, root, "go/utils/go.mod", "module utils\n\ngo 1.21\n")

	graph, err := DetectWorkspaces(root)
	if err != nil {
		t.Fatalf("DetectWorkspaces: %v", err)
	}

	npmUtils := graph.ByQualifiedName("npm:utils")
	if npmUtils == nil || npmUtils.RelDir != "js/utils" {
		t.Errorf("npm:utils = %v, want js/utils", npmUtils)
	}
	goUtils := graph.ByQualifiedName("go:utils")
	if goUtils == nil || goUtils.RelDir != "go/utils" {
		t.Errorf("go:utils = %v, want go/utils", goUtils)
	}
	// A bare ambiguous name must not resolve.
	if amb := graph.ByQualifiedName("utils"); amb != nil {
		t.Errorf("bare ambiguous name resolved to %v, want nil", amb)
	}
}

// --- Resource rendering -----------------------------------------------------

func TestExtractPackageID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		uri    string
		want   string
		wantOK bool
	}{
		{"qsdev://project/packages/web/context", "packages/web", true},
		{"qsdev://project/npm:utils/context", "npm:utils", true},
		{"qsdev://project/npm:@org%2Futils/context", "npm:@org/utils", true},
		{"qsdev://project/detection", "", false},
		{"qsdev://project//context", "", false},
	}
	for _, tc := range cases {
		got, ok := ExtractPackageID(tc.uri)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("ExtractPackageID(%q) = %q,%v; want %q,%v", tc.uri, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestRenderPackageContext(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeWS(t, root, "package.json", `{"name":"root","workspaces":["packages/*"]}`)
	writeWS(t, root, "packages/web/package.json", `{"name":"@org/web","dependencies":{"react":"18"}}`)

	graph, err := DetectWorkspaces(root)
	if err != nil {
		t.Fatalf("DetectWorkspaces: %v", err)
	}

	res, ok := graph.RenderPackageContext("qsdev://project/packages/web/context")
	if !ok {
		t.Fatal("RenderPackageContext by relDir returned not-ok")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &payload); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if payload["name"] != "@org/web" || payload["ecosystem"] != "npm" {
		t.Errorf("payload identity = %v", payload)
	}
	if payload["qualified_name"] != "npm:@org/web" {
		t.Errorf("qualified_name = %v, want npm:@org/web", payload["qualified_name"])
	}
	if _, ok := payload["sub_detection"].(map[string]any); !ok {
		t.Errorf("sub_detection missing or wrong type: %v", payload["sub_detection"])
	}

	// Resolution by qualified name works too.
	if _, ok := graph.RenderPackageContext("qsdev://project/npm:@org%2Fweb/context"); !ok {
		t.Error("RenderPackageContext by qualified name returned not-ok")
	}
	// An unknown package degrades (not-ok).
	if _, ok := graph.RenderPackageContext("qsdev://project/packages/missing/context"); ok {
		t.Error("RenderPackageContext for unknown package should be not-ok")
	}
}

// --- Performance ------------------------------------------------------------

func TestDetectWorkspacesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in -short mode")
	}
	root := t.TempDir()
	writeWS(t, root, "package.json", `{"name":"root","workspaces":["packages/*"]}`)
	const n = 50
	for i := 0; i < n; i++ {
		dir := fmt.Sprintf("packages/p%02d", i)
		writeWS(t, root, dir+"/package.json", fmt.Sprintf(`{"name":"p%02d","dependencies":{"dep":"1"}}`, i))
	}

	start := time.Now()
	graph, err := DetectWorkspaces(root)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("DetectWorkspaces: %v", err)
	}
	if graph.Len() != n {
		t.Fatalf("graph has %d packages, want %d", graph.Len(), n)
	}
	t.Logf("DetectWorkspaces for %d packages took %s", n, elapsed)
	if elapsed > 100*time.Millisecond {
		t.Errorf("detection took %s, exceeds 100ms target for %d packages", elapsed, n)
	}
}
