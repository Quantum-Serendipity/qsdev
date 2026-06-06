package vsentinel

import (
	"testing"
)

func TestDetectDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		files         map[string]string
		wantManifests int
		wantDrift     map[string]int // ecosystem -> drift count
	}{
		{
			name: "go no drift",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.20.0
)
`,
				"go.sum": `github.com/stretchr/testify v1.9.0 h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=
github.com/stretchr/testify v1.9.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
golang.org/x/sys v0.20.0 h1:Od9JTbYCk261bKm4M/mw7AklTlFYIa0bIp9BgSm1S8Y=
golang.org/x/sys v0.20.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"go": 0},
		},
		{
			name: "go with drift",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
	golang.org/x/sys v0.20.0
)
`,
				"go.sum": `github.com/stretchr/testify v1.8.4 h1:CcVxjf3Q8PM0mHUKJCdn+eZZtm5yQksXRJi6+GOwDY=
github.com/stretchr/testify v1.8.4/go.mod h1:sz/lmYIOXD/1dqDmKjjqLyZ2RngseejIcXlSw2iwfAo=
golang.org/x/sys v0.20.0 h1:Od9JTbYCk261bKm4M/mw7AklTlFYIa0bIp9BgSm1S8Y=
golang.org/x/sys v0.20.0/go.mod h1:/VUhepiaJMQUp4+oa/7Zr1D23ma6VTLIYjOOTFZPUcA=
`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"go": 1},
		},
		{
			name: "javascript no drift",
			files: map[string]string{
				"package.json": `{
  "name": "test",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "4.17.21"
  }
}`,
				"package-lock.json": `{
  "name": "test",
  "lockfileVersion": 3,
  "packages": {
    "": {
      "name": "test",
      "dependencies": {
        "express": "^4.18.0",
        "lodash": "4.17.21"
      }
    },
    "node_modules/express": {
      "version": "4.19.2"
    },
    "node_modules/lodash": {
      "version": "4.17.21"
    }
  }
}`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"javascript": 0},
		},
		{
			name: "javascript with drift",
			files: map[string]string{
				"package.json": `{
  "name": "test",
  "dependencies": {
    "express": "~4.18.0",
    "lodash": "4.17.21"
  }
}`,
				"package-lock.json": `{
  "name": "test",
  "lockfileVersion": 3,
  "packages": {
    "": {
      "name": "test",
      "dependencies": {
        "express": "~4.18.0",
        "lodash": "4.17.21"
      }
    },
    "node_modules/express": {
      "version": "4.19.2"
    },
    "node_modules/lodash": {
      "version": "4.17.21"
    }
  }
}`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"javascript": 1},
		},
		{
			name: "missing lockfile",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	golang.org/x/sys v0.20.0
)
`,
			},
			wantManifests: 0,
			wantDrift:     nil,
		},
		{
			name: "cargo no drift",
			files: map[string]string{
				"Cargo.toml": `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
serde = "1.0"
`,
				"Cargo.lock": `[[package]]
name = "serde"
version = "1.0.203"
source = "registry+https://github.com/rust-lang/crates.io-index"
`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"rust": 0},
		},
		{
			name: "cargo with drift",
			files: map[string]string{
				"Cargo.toml": `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
serde = "1.0"
`,
				"Cargo.lock": `[[package]]
name = "serde"
version = "2.0.1"
source = "registry+https://github.com/rust-lang/crates.io-index"
`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"rust": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFixtures(t, dir, tt.files)

			report, err := DetectDrift(dir)
			if err != nil {
				t.Fatalf("DetectDrift() error = %v", err)
			}

			if got := len(report.Manifests); got != tt.wantManifests {
				t.Errorf("manifest count = %d, want %d", got, tt.wantManifests)
			}

			if tt.wantDrift != nil {
				for _, m := range report.Manifests {
					want, ok := tt.wantDrift[m.Ecosystem]
					if !ok {
						continue
					}
					if m.DriftCount != want {
						t.Errorf("drift count for %s = %d, want %d", m.Ecosystem, m.DriftCount, want)
					}
					if len(m.Drifted) != want {
						t.Errorf("drifted entries for %s = %d, want %d", m.Ecosystem, len(m.Drifted), want)
					}
				}
			}
		})
	}
}

func TestDetectDriftEntryFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFixtures(t, dir, map[string]string{
		"go.mod": `module example.com/test

go 1.22

require (
	github.com/stretchr/testify v1.9.0
)
`,
		"go.sum": `github.com/stretchr/testify v1.8.4 h1:CcVxjf3Q8PM0mHUKJCdn+eZZtm5yQksXRJi6+GOwDY=
github.com/stretchr/testify v1.8.4/go.mod h1:sz/lmYIOXD/1dqDmKjjqLyZ2RngseejIcXlSw2iwfAo=
`,
	})

	report, err := DetectDrift(dir)
	if err != nil {
		t.Fatalf("DetectDrift() error = %v", err)
	}

	if len(report.Manifests) != 1 || len(report.Manifests[0].Drifted) != 1 {
		t.Fatal("expected 1 manifest with 1 drifted entry")
	}

	entry := report.Manifests[0].Drifted[0]
	if entry.Name != "github.com/stretchr/testify" {
		t.Errorf("name = %q, want %q", entry.Name, "github.com/stretchr/testify")
	}
	if entry.DeclaredVersion != "v1.9.0" {
		t.Errorf("declared = %q, want %q", entry.DeclaredVersion, "v1.9.0")
	}
	if entry.LockedVersion != "v1.8.4" {
		t.Errorf("locked = %q, want %q", entry.LockedVersion, "v1.8.4")
	}
}
