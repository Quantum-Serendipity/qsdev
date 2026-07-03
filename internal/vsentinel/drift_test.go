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
			// A manifest with no lockfile is the most dangerous (fully
			// unpinned) state and must fail closed: report the manifest with
			// drift, not silently skip it.
			name: "missing lockfile is drift",
			files: map[string]string{
				"go.mod": `module example.com/test

go 1.22

require (
	golang.org/x/sys v0.20.0
)
`,
			},
			wantManifests: 1,
			wantDrift:     map[string]int{"go": 1},
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

// TestDetectDrift_MissingLockfileIsDrift asserts the fail-closed behaviour:
// a manifest present without its lockfile must be surfaced as drift for every
// covered ecosystem, not silently dropped from the report.
func TestDetectDrift_MissingLockfileIsDrift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		manifest string
		content  string
		eco      string
	}{
		{"go.mod without go.sum", "go.mod", "module example.com/x\n\ngo 1.22\n\nrequire golang.org/x/sys v0.20.0\n", "go"},
		{"package.json without lock", "package.json", `{"name":"x","dependencies":{"express":"^4.18.0"}}`, "javascript"},
		{"Cargo.toml without lock", "Cargo.toml", "[package]\nname = \"x\"\n\n[dependencies]\nserde = \"1.0\"\n", "rust"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFixtures(t, dir, map[string]string{tc.manifest: tc.content})

			report, err := DetectDrift(dir)
			if err != nil {
				t.Fatalf("DetectDrift() error = %v", err)
			}
			if len(report.Manifests) != 1 {
				t.Fatalf("manifest count = %d, want 1 (missing lockfile must fail closed)", len(report.Manifests))
			}
			m := report.Manifests[0]
			if m.Ecosystem != tc.eco {
				t.Errorf("ecosystem = %q, want %q", m.Ecosystem, tc.eco)
			}
			if m.DriftCount < 1 {
				t.Errorf("DriftCount = %d, want >= 1 for a missing lockfile", m.DriftCount)
			}
		})
	}
}

// TestDetectDrift_NoFalsePositives is the M8 regression: two states that are NOT
// drift must not be flagged as "missing lockfile" — a dependency-free manifest
// (legitimately has no lockfile) and a project locked with a non-primary but
// catalog-valid lockfile (pnpm/yarn/bun instead of package-lock.json).
func TestDetectDrift_NoFalsePositives(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string]string
		eco   string
	}{
		{
			name:  "dependency-free go.mod (no go.sum) is not drift",
			files: map[string]string{"go.mod": "module example.com/x\n\ngo 1.22\n"},
			eco:   "go",
		},
		{
			name: "pnpm-locked project (no package-lock.json) is not drift",
			files: map[string]string{
				"package.json":   `{"name":"x","dependencies":{"express":"^4.18.0"}}`,
				"pnpm-lock.yaml": "lockfileVersion: '9.0'\n",
			},
			eco: "javascript",
		},
		{
			name: "yarn-locked project (no package-lock.json) is not drift",
			files: map[string]string{
				"package.json": `{"name":"x","dependencies":{"express":"^4.18.0"}}`,
				"yarn.lock":    "# yarn lockfile v1\n",
			},
			eco: "javascript",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFixtures(t, dir, tc.files)

			report, err := DetectDrift(dir)
			if err != nil {
				t.Fatalf("DetectDrift() error = %v", err)
			}
			if len(report.Manifests) != 1 {
				t.Fatalf("manifest count = %d, want 1", len(report.Manifests))
			}
			m := report.Manifests[0]
			if m.Ecosystem != tc.eco {
				t.Errorf("ecosystem = %q, want %q", m.Ecosystem, tc.eco)
			}
			if m.DriftCount != 0 {
				t.Errorf("DriftCount = %d, want 0 (not a real drift): %+v", m.DriftCount, m.Drifted)
			}
		})
	}
}

// TestJSSemverSatisfies_DowngradeIsDrift asserts that a within-major downgrade
// below the declared floor is flagged (not treated as satisfied), plus the
// caret, tilde, and exact boundaries — including npm's 0.x caret cap
// (^0.2.3 means >=0.2.3 <0.3.0, not "any 0.x above the floor") — and the
// fail-closed handling of unparseable input.
func TestJSSemverSatisfies_DowngradeIsDrift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		constraint string
		locked     string
		want       bool // true == satisfies (no drift)
	}{
		{"^4.18.0", "4.0.0", false},        // downgrade below floor -> drift
		{"^4.18.0", "4.19.2", true},        // within major, above floor -> ok
		{"^4.18.0", "5.0.0", false},        // out of major -> drift
		{"^4.18.0", "4.18.0", true},        // exact floor -> ok
		{"~4.18.0", "4.18.5", true},        // within minor -> ok
		{"~4.18.0", "4.19.0", false},       // next minor -> drift
		{"~4.18.0", "4.17.9", false},       // below floor -> drift
		{"4.17.21", "4.17.21", true},       // exact pin match
		{"4.17.21", "4.17.20", false},      // exact pin mismatch (downgrade)
		{"^0.2.3", "0.2.5", true},          // 0.x caret: within minor -> ok
		{"^0.2.3", "0.4.0", false},         // 0.x caret capped at <0.3.0 -> drift
		{"^0.2.3", "0.2.2", false},         // 0.x caret: below floor -> drift
		{"not-a-range", "1.0.0", false},    // unparseable constraint -> fail closed
		{"^1.0.0", "not-a-version", false}, // unparseable locked version -> fail closed
	}

	for _, tc := range cases {
		got := jsSemverSatisfies(tc.constraint, tc.locked)
		if got != tc.want {
			t.Errorf("jsSemverSatisfies(%q, %q) = %v, want %v", tc.constraint, tc.locked, got, tc.want)
		}
	}
}

// TestCargoSemverSatisfies_BoundaryFalseNegative asserts the cargo comparison
// no longer accepts "10.0.0" for a declared "1" (the old string-prefix false
// negative), keeps legitimate caret matches, applies Cargo's default caret
// semantics to bare versions (including the 0.x cap), and passes explicit
// operators through unchanged.
func TestCargoSemverSatisfies_BoundaryFalseNegative(t *testing.T) {
	t.Parallel()

	cases := []struct {
		constraint string
		locked     string
		want       bool
	}{
		{"1", "10.0.0", false},          // boundary false negative must be rejected
		{"1", "1.5.0", true},            // caret major -> ok
		{"1.0", "1.0.203", true},        // caret from bare "1.0" -> ok
		{"1.0", "2.0.1", false},         // next major -> drift
		{"1.0", "0.9.0", false},         // below floor -> drift
		{"0.2.3", "0.2.5", true},        // bare 0.x caret: within minor -> ok
		{"0.2.3", "0.4.0", false},       // bare 0.x caret capped at <0.3.0 -> drift
		{"^0.2.3", "0.4.0", false},      // explicit caret, same 0.x cap -> drift
		{"~1.2.0", "1.2.9", true},       // explicit tilde passes through -> ok
		{"~1.2.0", "1.3.0", false},      // explicit tilde: next minor -> drift
		{"garbage", "1.0.0", false},     // unparseable requirement -> fail closed
		{"1.0", "not-a-version", false}, // unparseable locked version -> fail closed
	}

	for _, tc := range cases {
		got := cargoSemverSatisfies(tc.constraint, tc.locked)
		if got != tc.want {
			t.Errorf("cargoSemverSatisfies(%q, %q) = %v, want %v", tc.constraint, tc.locked, got, tc.want)
		}
	}
}
