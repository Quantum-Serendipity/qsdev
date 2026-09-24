package vsentinel

import (
	"path/filepath"
	"runtime"
	"testing"
)

// driftFor runs DetectDrift over files and returns the drift entries reported
// for the given ecosystem.
func driftFor(t *testing.T, files map[string]string, eco string) []DriftEntry {
	t.Helper()
	dir := t.TempDir()
	writeFixtures(t, dir, files)
	report, err := DetectDrift(dir)
	if err != nil {
		t.Fatalf("DetectDrift() error = %v", err)
	}
	for _, m := range report.Manifests {
		if m.Ecosystem == eco {
			if m.DriftCount != len(m.Drifted) {
				t.Errorf("DriftCount = %d but %d entries", m.DriftCount, len(m.Drifted))
			}
			return m.Drifted
		}
	}
	t.Fatalf("no %s manifest in report %+v", eco, report)
	return nil
}

// TestDetectDrift_GoSumMembership covers go.sum's real shape — several versions
// per module, older ones as /go.mod-only lines — and replace directives. Drift
// is a required mod@version that go.sum does not checksum, not a mismatch with
// whichever go.sum version comes first.
func TestDetectDrift_GoSumMembership(t *testing.T) {
	t.Parallel()
	const goMod = `module example.com/x

go 1.22

require (
	github.com/spf13/pflag v1.0.10
	golang.org/x/sys v0.46.0
	example.com/local v0.0.0-00010101000000-000000000000
	example.com/forked v1.2.0
)

replace example.com/local => ../local

replace example.com/forked v1.2.0 => example.com/fork v1.2.1
`
	const multiVersionSum = `example.com/fork v1.2.1 h1:a=
example.com/fork v1.2.1/go.mod h1:b=
github.com/spf13/pflag v1.0.9/go.mod h1:c=
github.com/spf13/pflag v1.0.10 h1:d=
github.com/spf13/pflag v1.0.10/go.mod h1:e=
golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e/go.mod h1:f=
golang.org/x/sys v0.46.0 h1:g=
golang.org/x/sys v0.46.0/go.mod h1:h=
`
	if got := driftFor(t, map[string]string{"go.mod": goMod, "go.sum": multiVersionSum}, "go"); len(got) != 0 {
		t.Errorf("tidy multi-version go.sum reported drift: %+v", got)
	}

	// A require added to go.mod without updating go.sum is drift.
	const staleSum = `example.com/fork v1.2.1 h1:a=
github.com/spf13/pflag v1.0.9 h1:c=
golang.org/x/sys v0.46.0 h1:g=
`
	got := driftFor(t, map[string]string{"go.mod": goMod, "go.sum": staleSum}, "go")
	if len(got) != 1 || got[0].Name != "github.com/spf13/pflag" || got[0].LockedVersion != "v1.0.9" {
		t.Errorf("drift = %+v, want only pflag locked at v1.0.9", got)
	}
}

// TestDetectDrift_RepoGoModule runs Go drift detection on this repository's own
// go.mod/go.sum, a real tidied module, which must report no drift.
func TestDetectDrift_RepoGoModule(t *testing.T) {
	t.Parallel()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	drifted, err := checkGoDrift(filepath.Join(root, "go.mod"), filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatalf("checkGoDrift: %v", err)
	}
	if len(drifted) != 0 {
		t.Errorf("repo go.mod reported drift: %+v", drifted)
	}
}

// TestDetectDrift_MissingFromLockfile proves a declared dependency the
// lockfile does not pin at all is drift (the lockfile is stale and the
// dependency would resolve fresh), not silently skipped.
func TestDetectDrift_MissingFromLockfile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		eco   string
		want  []DriftEntry
	}{
		{
			name: "npm dependency added without npm install",
			files: map[string]string{
				"package.json": `{"dependencies":{"express":"^4.18.0","left-pad":"^1.3.0","ws-pkg":"workspace:*"}}`,
				"package-lock.json": `{"lockfileVersion":3,"packages":{"":{},
  "node_modules/express":{"version":"4.19.2"}}}`,
			},
			eco:  "javascript",
			want: []DriftEntry{{Name: "left-pad", DeclaredVersion: "^1.3.0", LockedVersion: notInLockfile}},
		},
		{
			name: "cargo dependency added without cargo update",
			files: map[string]string{
				"Cargo.toml": "[package]\nname = \"x\"\nversion = \"0.1.0\"\n\n[dependencies]\nserde = \"1.0\"\nanyhow = \"1\"\nlocal = { path = \"../local\" }\n",
				"Cargo.lock": "[[package]]\nname = \"serde\"\nversion = \"1.0.203\"\n",
			},
			eco:  "rust",
			want: []DriftEntry{{Name: "anyhow", DeclaredVersion: "1", LockedVersion: notInLockfile}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := driftFor(t, tt.files, tt.eco)
			if len(got) != len(tt.want) {
				t.Fatalf("drift = %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("drift[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestDetectDrift_CargoTOMLShapes covers the Cargo.toml and Cargo.lock shapes
// the line-oriented parser mishandled: inline-table specs, comments, renamed
// and workspace-inherited dependencies, build/target/workspace tables, and a
// crate locked at two versions.
func TestDetectDrift_CargoTOMLShapes(t *testing.T) {
	t.Parallel()
	const cargoToml = `[package]
name = "x"
version = "0.1.0"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
# comment = "x"
rand = "0.7"
rng = { package = "rand", version = "0.8" }
shared.workspace = true

[build-dependencies]
cc = "1.0"

[target.'cfg(unix)'.dependencies]
libc = "0.2"

[workspace.dependencies]
shared = "2.1"
`
	const cargoLock = `version = 3

[[package]]
name = "cc"
version = "1.0.90"

[[package]]
name = "libc"
version = "0.2.155"

[[package]]
name = "rand"
version = "0.7.3"

[[package]]
name = "rand"
version = "0.8.5"

[[package]]
name = "serde"
version = "1.0.200"

[[package]]
name = "shared"
version = "2.1.4"
`
	if got := driftFor(t, map[string]string{"Cargo.toml": cargoToml, "Cargo.lock": cargoLock}, "rust"); len(got) != 0 {
		t.Errorf("valid Cargo manifest reported drift: %+v", got)
	}

	deps, err := parseCargoDeps(writeTemp(t, "Cargo.toml", cargoToml))
	if err != nil {
		t.Fatalf("parseCargoDeps: %v", err)
	}
	names := make(map[string]bool, len(deps))
	for _, d := range deps {
		names[d.Name] = true
	}
	for _, want := range []string{"serde", "rand", "rng", "shared", "cc", "libc"} {
		if !names[want] {
			t.Errorf("dependency %q not parsed; got %+v", want, deps)
		}
	}
	for _, bogus := range []string{"# comment", "shared.workspace"} {
		if names[bogus] {
			t.Errorf("bogus dependency %q parsed", bogus)
		}
	}

	// rand requires 0.9 but only 0.7.3 and 0.8.5 are locked: drift.
	drifted := driftFor(t, map[string]string{
		"Cargo.toml": "[package]\nname = \"x\"\nversion = \"0.1.0\"\n\n[dependencies]\nrand = \"0.9\"\n",
		"Cargo.lock": cargoLock,
	}, "rust")
	if len(drifted) != 1 || drifted[0].Name != "rand" || drifted[0].LockedVersion != "0.7.3, 0.8.5" {
		t.Errorf("drift = %+v, want rand locked at 0.7.3, 0.8.5", drifted)
	}
}

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	writeFixtures(t, dir, map[string]string{name: body})
	return filepath.Join(dir, name)
}
