package archtest

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

const fixtureRoot = "testdata/repo"

var (
	fixtureOnce sync.Once
	fixture     *Repo
	fixtureErr  error
)

// loadFixture parses the synthetic repository once; tests only read it.
func loadFixture(t *testing.T) *Repo {
	t.Helper()
	fixtureOnce.Do(func() { fixture, fixtureErr = Load(fixtureRoot) })
	if fixtureErr != nil {
		t.Fatalf("Load(%s): %v", fixtureRoot, fixtureErr)
	}
	return fixture
}

func TestLoadRepo_ScansAllBuildTagsAndSkipsVendor(t *testing.T) {
	t.Parallel()
	repo := loadFixture(t)
	if repo.Module != "example.com/m" {
		t.Errorf("Module = %q, want example.com/m", repo.Module)
	}
	byPath := make(map[string]*File, len(repo.Files))
	for _, f := range repo.Files {
		byPath[f.Path] = f
	}

	tests := []struct {
		path   string
		loaded bool
	}{
		{"pkg/types/types.go", true},
		{"pkg/types/types_windows.go", true},
		{"pkg/types/types_test.go", true},
		{"vendor/example.com/dep/dep.go", false},
		{".hidden/pkg/hidden.go", false},
		{"pkg/types/testdata/bad.go", false},
		{"pkg/nested/nested.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if _, ok := byPath[tt.path]; ok != tt.loaded {
				t.Errorf("loaded = %v, want %v", ok, tt.loaded)
			}
		})
	}

	t.Run("file metadata", func(t *testing.T) {
		t.Parallel()
		f := byPath["pkg/types/types.go"]
		if f == nil {
			t.Fatal("pkg/types/types.go not loaded")
		}
		if f.Pkg != "pkg/types" || f.IsTest || f.AST == nil {
			t.Errorf("got Pkg=%q IsTest=%v AST nil=%v", f.Pkg, f.IsTest, f.AST == nil)
		}
		if got := f.Imports["osx"]; got != "os" {
			t.Errorf(`Imports["osx"] = %q, want "os"`, got)
		}
		if got := f.Imports["enumtext"]; got != "example.com/m/internal/enumtext" {
			t.Errorf(`Imports["enumtext"] = %q`, got)
		}
		if tf := byPath["pkg/types/types_test.go"]; tf == nil || !tf.IsTest {
			t.Error("types_test.go must be loaded with IsTest")
		}
	})
}

func TestImportName(t *testing.T) {
	t.Parallel()
	tests := []struct{ path, want string }{
		{"os", "os"},
		{"os/exec", "exec"},
		{"gopkg.in/yaml.v3", "yaml"},
		{"github.com/fastcat/gdev/v2", "gdev"},
		{"example.com/m/internal/catalog", "catalog"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := defaultImportName(tt.path); got != tt.want {
				t.Errorf("defaultImportName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLayerRules(t *testing.T) {
	t.Parallel()
	got := Collect(loadFixture(t), layerRules())

	tests := []struct {
		name    string
		key     Key
		present bool
	}{
		{"direct pkg->internal", Key{"pkg-public-leaf", "pkg/types -> internal/enumtext"}, true},
		{"windows-only file edge", Key{"pkg-public-leaf", "pkg/types -> addons/claudecode"}, true},
		{"test file exempt", Key{"pkg-public-leaf", "pkg/types -> internal/shelltest"}, false},
		{"pkg->pkg allowed", Key{"pkg-public-leaf", "pkg/clean -> pkg/types"}, false},
		{"pkg->procexec allowed", Key{"pkg-public-leaf", "pkg/clean -> internal/procexec"}, false},
		{"skipped trees contribute nothing", Key{"pkg-public-leaf", "pkg/nested -> internal/catalog"}, false},
		{"transitive foundation->catalog", Key{"foundation-leaf", "internal/secrets -> internal/catalog"}, true},
		{"transitive foundation->net/http", Key{"foundation-leaf", "internal/secrets -> net/http"}, true},
		{"hookrt external prefix", Key{"hookrt-lean", "internal/hookrt -> github.com/charmbracelet/huh"}, true},
		{"internal->addons", Key{"internal-no-adapters", "internal/mcp -> addons/claudecode"}, true},
		{"app->instance", Key{"app-no-adapters", "internal/app/project -> instance"}, true},
		{"app not double-reported", Key{"internal-no-adapters", "internal/app/project -> instance"}, false},
		{"secretstest from production", Key{"secretstest-test-only", "internal/redact -> internal/secrets/secretstest"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n, ok := got[tt.key]
			if ok != tt.present {
				t.Fatalf("%v present = %v, want %v (all: %v)", tt.key, ok, tt.present, got)
			}
			if ok && n != 1 {
				t.Errorf("%v count = %d, want 1 (edges count once)", tt.key, n)
			}
		})
	}
	if len(got) != 8 {
		t.Errorf("got %d violations, want 8: %v", len(got), got)
	}
}

func TestCompareBaseline(t *testing.T) {
	t.Parallel()
	current := Set{
		{"r", "a -> b"}: 1,
		{"r", "pkg/x"}:  3,
	}
	tests := []struct {
		name     string
		baseline string
		want     []string // substrings, one per expected problem
	}{
		{"exact match", "r\ta -> b\t1\nr\tpkg/x\t3\n", nil},
		{"comments and blank lines", "# header\n\nr\ta -> b\t1\nr\tpkg/x\t3\n", nil},
		{"CRLF", "# header\r\nr\ta -> b\t1\r\nr\tpkg/x\t3\r\n", nil},
		{"new violation", "r\tpkg/x\t3\n", []string{"new violation [r] a -> b"}},
		{"raised count", "r\ta -> b\t1\nr\tpkg/x\t2\n", []string{"[r] pkg/x: count rose from 2 to 3"}},
		{"lowered count", "r\ta -> b\t1\nr\tpkg/x\t5\n", []string{"[r] pkg/x now occurs 3 times (baseline 5): lower it"}},
		{"stale entry", "r\ta -> b\t1\nr\tpkg/x\t3\nr\tgone\t2\n", []string{"[r] gone no longer occurs: delete it"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, err := ParseBaseline(strings.NewReader(tt.baseline))
			if err != nil {
				t.Fatalf("ParseBaseline: %v", err)
			}
			assertProblems(t, Compare(current, base), tt.want)
		})
	}
}

func TestParseBaseline_Rejects(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, in string }{
		{"two fields", "r\tsubject\n"},
		{"non-numeric count", "r\tsubject\tmany\n"},
		{"zero count", "r\tsubject\t0\n"},
		{"duplicate key", "r\ts\t1\nr\ts\t2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseBaseline(strings.NewReader(tt.in)); err == nil {
				t.Errorf("ParseBaseline(%q) succeeded, want error", tt.in)
			}
		})
	}
}

func TestFormatBaselineRoundTrip(t *testing.T) {
	t.Parallel()
	in := Set{{"b", "z"}: 2, {"a", "y -> x"}: 1, {"b", "a"}: 4}
	out := FormatBaseline(in)
	body := strings.Join(slices.DeleteFunc(strings.Split(string(out), "\n"), func(l string) bool {
		return l == "" || strings.HasPrefix(l, "#")
	}), "\n")
	if want := "a\ty -> x\t1\nb\ta\t4\nb\tz\t2"; body != want {
		t.Errorf("FormatBaseline body =\n%s\nwant\n%s", body, want)
	}
	back, err := ParseBaseline(strings.NewReader(string(out)))
	if err != nil {
		t.Fatalf("ParseBaseline: %v", err)
	}
	assertProblems(t, Compare(in, back), nil)
}

func TestCompareMonotone(t *testing.T) {
	t.Parallel()
	base := Set{{"r", "a"}: 2, {"r", "b"}: 1}
	tests := []struct {
		name    string
		current Set
		want    []string
	}{
		{"identical", Set{{"r", "a"}: 2, {"r", "b"}: 1}, nil},
		{"shrunk", Set{{"r", "a"}: 1}, nil},
		{"new key", Set{{"r", "a"}: 2, {"r", "b"}: 1, {"r", "c"}: 1}, []string{"gained entry [r] c"}},
		{"raised", Set{{"r", "a"}: 3, {"r", "b"}: 1}, []string{"raised [r] a from 2 to 3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertProblems(t, CompareMonotone(tt.current, base), tt.want)
		})
	}
}

func assertProblems(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d problems %q, want %d matching %q", len(got), got, len(want), want)
	}
	for i := range want {
		if !strings.Contains(got[i], want[i]) {
			t.Errorf("problem %d = %q, want it to contain %q", i, got[i], want[i])
		}
	}
}
