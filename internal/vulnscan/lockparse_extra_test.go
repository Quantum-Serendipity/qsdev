package vulnscan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestParseGoSumSkipsGoModOnlyVersions proves only versions with a module-zip
// hash line are queried: older versions that appear only as "/go.mod" lines are
// graph-pruning metadata, never built, and would produce false positives.
func TestParseGoSumSkipsGoModOnlyVersions(t *testing.T) {
	t.Parallel()
	const body = `github.com/spf13/pflag v1.0.9/go.mod h1:a=
github.com/spf13/pflag v1.0.10 h1:b=
github.com/spf13/pflag v1.0.10/go.mod h1:c=
golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e/go.mod h1:d=
golang.org/x/sys v0.46.0 h1:e=
golang.org/x/sys v0.46.0/go.mod h1:f=
`
	_, path := writeFile(t, "go.sum", body)
	pkgs, err := parseGoSum(path, "Go")
	if err != nil {
		t.Fatalf("parseGoSum: %v", err)
	}
	want := []string{
		"github.com/spf13/pflag@1.0.10 (Go)",
		"golang.org/x/sys@0.46.0 (Go)",
	}
	if got := pkgKeys(pkgs); !equalKeys(got, want) {
		t.Errorf("packages = %v, want %v", got, want)
	}
}

// TestParseNPMLockNestedAndAliased covers the lock shapes that silently hid
// packages from OSV: lockfileVersion 1 transitive dependencies nested under
// their parent, and npm aliases whose real registry name differs from the
// folder name. Workspace links are not registry packages and are skipped.
func TestParseNPMLockNestedAndAliased(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "v1 nested dependencies",
			body: `{"lockfileVersion": 1, "dependencies": {
  "a": {"version": "1.0.0", "dependencies": {
    "lodash": {"version": "4.17.4", "dependencies": {"deep": {"version": "0.1.0"}}}
  }}
}}`,
			want: []string{"a@1.0.0 (npm)", "deep@0.1.0 (npm)", "lodash@4.17.4 (npm)"},
		},
		{
			name: "v1 alias",
			body: `{"lockfileVersion": 1, "dependencies": {
  "foo": {"version": "npm:lodash@4.17.4"},
  "bar": {"version": "npm:@scope/pkg@2.0.0"}
}}`,
			want: []string{"@scope/pkg@2.0.0 (npm)", "lodash@4.17.4 (npm)"},
		},
		{
			name: "v3 alias and link",
			body: `{"lockfileVersion": 3, "packages": {
  "": {"name": "root"},
  "node_modules/foo": {"name": "lodash", "version": "4.17.4"},
  "node_modules/left-pad": {"version": "1.3.0"},
  "node_modules/my-workspace": {"resolved": "packages/ws", "link": true},
  "packages/ws": {"name": "my-workspace", "version": "0.0.1"}
}}`,
			want: []string{"left-pad@1.3.0 (npm)", "lodash@4.17.4 (npm)", "my-workspace@0.0.1 (npm)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, path := writeFile(t, "package-lock.json", tt.body)
			pkgs, err := parseNPMLock(path, "npm")
			if err != nil {
				t.Fatalf("parseNPMLock: %v", err)
			}
			if got := pkgKeys(pkgs); !equalKeys(got, tt.want) {
				t.Errorf("packages = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseRequirementsTxt pins which requirement lines yield a queryable
// coordinate: exact "==" and "===" pins do; loose specifiers do not.
func TestParseRequirementsTxt(t *testing.T) {
	t.Parallel()
	const body = `requests>=2.0
flask
Django===3.2.0
urllib3 == 1.26.0
pkg[extra]==1.0 ; python_version < "3.12"
-r other.txt
# comment
`
	_, path := writeFile(t, "requirements.txt", body)
	pkgs, err := parseRequirementsTxt(path, "PyPI")
	if err != nil {
		t.Fatalf("parseRequirementsTxt: %v", err)
	}
	want := []string{"Django@3.2.0 (PyPI)", "pkg@1.0 (PyPI)", "urllib3@1.26.0 (PyPI)"}
	if got := pkgKeys(pkgs); !equalKeys(got, want) {
		t.Errorf("packages = %v, want %v", got, want)
	}
}

// TestLockFileForEcosystemPrefersDedicatedLock proves the per-ecosystem
// resolver posture uses picks a dedicated lock over a loose manifest of the same
// ecosystem, and never returns another ecosystem's lock file.
func TestLockFileForEcosystemPrefersDedicatedLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"requirements.txt", "poetry.lock", "go.sum"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	tests := []struct {
		eco      string
		wantName string
		wantOK   bool
	}{
		{ecosystem.NamePython, "poetry.lock", true},
		{ecosystem.NameGo, "go.sum", true},
		{ecosystem.NameRust, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.eco, func(t *testing.T) {
			t.Parallel()
			lf, path, ok := LockFileForEcosystem(dir, tt.eco)
			if ok != tt.wantOK {
				t.Fatalf("ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if lf.Name() != tt.wantName || path != filepath.Join(dir, tt.wantName) {
				t.Errorf("got %q at %q, want %q", lf.Name(), path, tt.wantName)
			}
		})
	}
}
