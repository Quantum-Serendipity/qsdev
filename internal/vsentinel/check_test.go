package vsentinel

import (
	"os"
	"path/filepath"
	"sort"
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
	if dep.LatestKnown != "" {
		t.Errorf("latest known should be empty, got %q", dep.LatestKnown)
	}
	if dep.StaleDays != 0 {
		t.Errorf("stale days should be 0, got %d", dep.StaleDays)
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
