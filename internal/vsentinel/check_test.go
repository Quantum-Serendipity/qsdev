package vsentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCheckVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		files     map[string]string
		wantCount int
		wantEcos  []string
		wantDeps  map[string]int // ecosystem -> dep count
	}{
		{
			name: "go project with go.mod",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.20.0
)
`,
			},
			wantCount: 1,
			wantEcos:  []string{"go"},
			wantDeps:  map[string]int{"go": 2},
		},
		{
			name: "javascript project with package.json",
			files: map[string]string{
				"package.json": `{
  "name": "test",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "4.17.21"
  },
  "devDependencies": {
    "jest": "^29.7.0"
  }
}`,
			},
			wantCount: 1,
			wantEcos:  []string{"javascript"},
			wantDeps:  map[string]int{"javascript": 3},
		},
		{
			name:      "empty directory",
			files:     map[string]string{},
			wantCount: 0,
			wantEcos:  nil,
			wantDeps:  nil,
		},
		{
			name: "multiple manifests in same directory",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	golang.org/x/text v0.15.0
)
`,
				"package.json": `{
  "name": "frontend",
  "dependencies": {
    "react": "^18.2.0"
  }
}`,
				"requirements.txt": `requests>=2.31.0
flask==3.0.0
`,
			},
			wantCount: 3,
			wantEcos:  []string{"go", "javascript", "python"},
			wantDeps:  map[string]int{"go": 1, "javascript": 1, "python": 2},
		},
		{
			name: "cargo project with Cargo.toml",
			files: map[string]string{
				"Cargo.toml": `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = "1.37"

[dev-dependencies]
criterion = "0.5"
`,
			},
			wantCount: 1,
			wantEcos:  []string{"rust"},
			wantDeps:  map[string]int{"rust": 3},
		},
		{
			name: "python pyproject.toml",
			files: map[string]string{
				"pyproject.toml": `[project]
name = "myapp"

dependencies = [
    "requests>=2.31.0",
    "pydantic~=2.0",
]
`,
			},
			wantCount: 1,
			wantEcos:  []string{"python"},
			wantDeps:  map[string]int{"python": 2},
		},
		{
			name: "go.mod with indirect dependencies",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.20.0 // indirect
)
`,
			},
			wantCount: 1,
			wantEcos:  []string{"go"},
			wantDeps:  map[string]int{"go": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFixtures(t, dir, tt.files)

			report, err := CheckVersions(dir)
			if err != nil {
				t.Fatalf("CheckVersions() error = %v", err)
			}

			if got := len(report.Manifests); got != tt.wantCount {
				t.Errorf("manifest count = %d, want %d", got, tt.wantCount)
			}

			var gotEcos []string
			for _, m := range report.Manifests {
				gotEcos = append(gotEcos, m.Ecosystem)
			}
			sort.Strings(gotEcos)
			sort.Strings(tt.wantEcos)

			if len(gotEcos) != len(tt.wantEcos) {
				t.Errorf("ecosystems = %v, want %v", gotEcos, tt.wantEcos)
			} else {
				for i := range gotEcos {
					if gotEcos[i] != tt.wantEcos[i] {
						t.Errorf("ecosystem[%d] = %q, want %q", i, gotEcos[i], tt.wantEcos[i])
					}
				}
			}

			if tt.wantDeps != nil {
				for _, m := range report.Manifests {
					want, ok := tt.wantDeps[m.Ecosystem]
					if !ok {
						continue
					}
					if got := len(m.Dependencies); got != want {
						t.Errorf("deps for %s = %d, want %d", m.Ecosystem, got, want)
					}
				}
			}
		})
	}
}

func TestCheckVersionsDepFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFixtures(t, dir, map[string]string{
		"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
)
`,
	})

	report, err := CheckVersions(dir)
	if err != nil {
		t.Fatalf("CheckVersions() error = %v", err)
	}

	if len(report.Manifests) != 1 || len(report.Manifests[0].Dependencies) != 1 {
		t.Fatal("expected 1 manifest with 1 dependency")
	}

	dep := report.Manifests[0].Dependencies[0]
	if dep.Name != "github.com/stretchr/testify" {
		t.Errorf("dep name = %q, want %q", dep.Name, "github.com/stretchr/testify")
	}
	if dep.DeclaredVersion != "v1.9.0" {
		t.Errorf("dep version = %q, want %q", dep.DeclaredVersion, "v1.9.0")
	}
}

// TestCheckVersionsJSONShape pins the check_versions MCP output: snake_case keys
// like the other version-sentinel payloads, and no staleness fields, since
// nothing computes staleness and an always-zero stale count reads as "nothing
// is stale".
func TestCheckVersionsJSONShape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFixtures(t, dir, map[string]string{
		"go.mod": "module example.com/test\n\ngo 1.22\n\nrequire github.com/stretchr/testify v1.9.0\n",
	})
	report, err := CheckVersions(dir)
	if err != nil {
		t.Fatalf("CheckVersions() error = %v", err)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(data)
	for _, want := range []string{`"manifests"`, `"last_check_time"`, `"ecosystem"`, `"dependencies"`, `"declared_version"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing key %s: %s", want, out)
		}
	}
	for _, unwanted := range []string{"Stale", "stale", "LatestKnown", "latest_known", "DriftDetected", "Manifests"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output contains %q: %s", unwanted, out)
		}
	}
}

// TestParsePythonDeps covers PEP 621/735 and Poetry pyproject declarations and
// PEP 508 specifier splitting: single-line arrays, multi-clause specifiers,
// extras, markers, comments and direct references.
func TestParsePythonDeps(t *testing.T) {
	t.Parallel()

	t.Run("pyproject", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		writeFixtures(t, dir, map[string]string{"pyproject.toml": `[project]
name = "x"
dependencies = ["requests>=2", "flask"]

[project.optional-dependencies]
dev = ["pytest>=7 ; python_version >= '3.8'"]

[dependency-groups]
lint = ["ruff==0.4.0", {include-group = "dev"}]

[tool.poetry.dependencies]
python = "^3.11"
django = "^5.0"
httpx = { version = "^0.27", extras = ["http2"] }
local = { path = "../local" }
`})
		deps, err := parsePyprojectToml(dir + "/pyproject.toml")
		if err != nil {
			t.Fatalf("parsePyprojectToml: %v", err)
		}
		got := make(map[string]string, len(deps))
		for _, d := range deps {
			got[d.Name] = d.DeclaredVersion
		}
		want := map[string]string{
			"requests": ">=2", "flask": "", "pytest": ">=7", "ruff": "==0.4.0",
			"django": "^5.0", "httpx": "^0.27", "local": "",
		}
		if len(got) != len(want) {
			t.Errorf("deps = %v, want %v", got, want)
		}
		for name, ver := range want {
			if gv, ok := got[name]; !ok || gv != ver {
				t.Errorf("dep %q = %q (present %t), want %q", name, gv, ok, ver)
			}
		}
	})

	tests := []struct {
		in, wantName, wantVer string
	}{
		{"pkg<2,>=1", "pkg", "<2,>=1"},
		{"foo==1.0  # pinned", "foo", "==1.0"},
		{"bar @ https://x/y.whl", "bar", "@ https://x/y.whl"},
		{"baz[extra1,extra2]>=3.0; sys_platform == 'linux'", "baz", ">=3.0"},
		{"Django===3.2.0", "Django", "===3.2.0"},
		{"legacy (>=1.0)", "legacy", ">=1.0"},
		{"plain", "plain", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			name, ver := splitPythonDep(tt.in)
			if name != tt.wantName || ver != tt.wantVer {
				t.Errorf("splitPythonDep(%q) = (%q, %q), want (%q, %q)", tt.in, name, ver, tt.wantName, tt.wantVer)
			}
		})
	}
}

func writeFixtures(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", name, err)
		}
	}
}
