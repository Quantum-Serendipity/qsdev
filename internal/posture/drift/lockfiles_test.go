package drift

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestDetectLockfileDrift_Valid(t *testing.T) {
	dir := t.TempDir()

	// Create manifest and lockfile with lockfile newer.
	manifestPath := filepath.Join(dir, "package.json")
	lockfilePath := filepath.Join(dir, "package-lock.json")

	writeFile(t, manifestPath, `{"name": "test"}`)
	writeFile(t, lockfilePath, `{"lockfileVersion": 3}`)

	// Ensure lockfile is newer.
	past := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(manifestPath, past, past); err != nil {
		t.Fatal(err)
	}

	cat := detectLockfileDrift(dir)

	// Filter findings to just this pair.
	for _, f := range cat.Findings {
		if f.Subject == "package-lock.json" {
			t.Errorf("expected no finding for valid lock file, got: %+v", f)
		}
	}
}

func TestDetectLockfileDrift_Stale(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "go.mod")
	lockfilePath := filepath.Join(dir, "go.sum")

	writeFile(t, lockfilePath, "some-lock-content")
	writeFile(t, manifestPath, "module test")

	// Make lockfile older than manifest.
	past := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(lockfilePath, past, past); err != nil {
		t.Fatal(err)
	}

	cat := detectLockfileDrift(dir)

	found := false
	for _, f := range cat.Findings {
		if f.Subject == "go.sum" {
			found = true
			if f.Severity != Warning {
				t.Errorf("expected severity %q, got %q", Warning, f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected warning about stale go.sum")
	}
}

func TestDetectLockfileDrift_MissingLockfile(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "Cargo.toml"), "[package]\nname = \"test\"")
	// No Cargo.lock.

	cat := detectLockfileDrift(dir)

	found := false
	for _, f := range cat.Findings {
		if f.Subject == "Cargo.lock" {
			found = true
			if f.Severity != Error {
				t.Errorf("expected severity %q, got %q", Error, f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected error about missing Cargo.lock")
	}
}

func TestDetectLockfileDrift_NoManifest(t *testing.T) {
	dir := t.TempDir()
	// Empty directory — no manifests at all.

	cat := detectLockfileDrift(dir)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when no manifests exist, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectLockfileDrift_MultipleManifests(t *testing.T) {
	dir := t.TempDir()

	// package.json with yarn.lock (fresh) and go.mod with missing go.sum.
	writeFile(t, filepath.Join(dir, "package.json"), `{}`)
	writeFile(t, filepath.Join(dir, "yarn.lock"), `yarn lockfile v1`)

	past := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(filepath.Join(dir, "package.json"), past, past); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "go.mod"), "module test")
	// No go.sum.

	cat := detectLockfileDrift(dir)

	// Should report the missing go.sum, but nothing for package.json: yarn.lock
	// is present and fresh, and it satisfies the manifest on its own.
	yarnFinding := false
	goSumFinding := false
	for _, f := range cat.Findings {
		if f.Subject == "yarn.lock" {
			yarnFinding = true
		}
		if f.Subject == "go.sum" {
			goSumFinding = true
		}
	}
	if yarnFinding {
		t.Error("yarn.lock is present and fresh; should not have a finding")
	}
	if !goSumFinding {
		t.Error("go.sum is missing; should have a finding")
	}
	if len(cat.Findings) != 1 {
		t.Errorf("expected exactly 1 finding (go.sum), got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

// TestDetectLockfileDrift_AlternativeLockfiles guards against reporting every
// unused package manager's lockfile as missing: one present alternative
// satisfies the manifest, and only an entirely unlocked manifest is an error.
func TestDetectLockfileDrift_AlternativeLockfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		files        []string
		wantSubjects []string
	}{
		{name: "pnpm project", files: []string{"package.json", "pnpm-lock.yaml"}},
		{name: "yarn project", files: []string{"package.json", "yarn.lock"}},
		{name: "bun project", files: []string{"package.json", "bun.lockb"}},
		{name: "uv project", files: []string{"pyproject.toml", "uv.lock"}},
		{name: "poetry project", files: []string{"pyproject.toml", "poetry.lock"}},
		{name: "unlocked package.json", files: []string{"package.json"}, wantSubjects: []string{"package.json"}},
		{name: "unlocked pyproject.toml", files: []string{"pyproject.toml"}, wantSubjects: []string{"pyproject.toml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			past := time.Now().Add(-10 * time.Second)
			for i, name := range tt.files {
				path := filepath.Join(dir, name)
				writeFile(t, path, "content")
				// The manifest (first file) is older than its lockfile.
				if i == 0 {
					if err := os.Chtimes(path, past, past); err != nil {
						t.Fatal(err)
					}
				}
			}

			cat := detectLockfileDrift(dir)

			var got []string
			for _, f := range cat.Findings {
				got = append(got, f.Subject)
				if f.Severity != Error {
					t.Errorf("unexpected %s finding: %+v", f.Severity, f)
				}
			}
			if !slices.Equal(got, tt.wantSubjects) {
				t.Errorf("finding subjects = %v, want %v", got, tt.wantSubjects)
			}
		})
	}
}

func TestDetectLockfileDrift_PythonEcosystem(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[build-system]")
	writeFile(t, filepath.Join(dir, "uv.lock"), "uv lock content")

	// Make lockfile older than manifest.
	past := time.Now().Add(-10 * time.Second)
	if err := os.Chtimes(filepath.Join(dir, "uv.lock"), past, past); err != nil {
		t.Fatal(err)
	}

	cat := detectLockfileDrift(dir)

	found := false
	for _, f := range cat.Findings {
		if f.Subject == "uv.lock" {
			found = true
			if f.Severity != Warning {
				t.Errorf("expected severity %q, got %q", Warning, f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected warning about stale uv.lock")
	}
}
