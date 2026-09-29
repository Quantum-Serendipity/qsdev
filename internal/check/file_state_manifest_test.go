package check

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const (
	manifestHookPath = ".claude/hooks/package-guard.py"
	manifestHookBody = "#!/usr/bin/env python3\nprint('guard')\n"
)

// manifestProject is a generated project as a test sees it: its root, the
// generation state (whether or not it is on disk) and the check context.
type manifestProject struct {
	dir   string
	st    types.GeneratedState
	ctx   CheckContext
	state string // absolute path of the (maybe absent) init state file
}

// newManifestProject writes a machine-owned hook and a human-edited CLAUDE.md,
// the committed config's stand-in and, per the flags, the local generation
// state and the committed manifest.
func newManifestProject(t *testing.T, withState, withManifest bool) manifestProject {
	t.Helper()
	dir := t.TempDir()
	files := []types.GeneratedFile{
		{Path: manifestHookPath, Content: []byte(manifestHookBody), Mode: 0o755, Strategy: types.Overwrite},
		{Path: "CLAUDE.md", Content: []byte("# Project\n"), Mode: 0o644, Strategy: types.SectionMarker},
	}
	for _, f := range files {
		abs := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, f.Content, f.Mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(abs, f.Mode); err != nil {
			t.Fatal(err)
		}
	}
	st := state.RecordFiles(files)
	stateFile := filepath.Join(dir, filepath.FromSlash(state.InitStateFile()))
	if withState {
		if err := state.SaveStateToFile(stateFile, st); err != nil {
			t.Fatal(err)
		}
	}
	if withManifest {
		if err := state.WriteManifest(dir, state.BuildManifest(st)); err != nil {
			t.Fatal(err)
		}
	}
	return manifestProject{
		dir:   dir,
		st:    st,
		state: stateFile,
		ctx: CheckContext{
			ProjectRoot:  dir,
			StateFile:    stateFile,
			ManifestFile: filepath.Join(dir, state.ManifestFile()),
			QsdevConfig:  &types.QsdevConfig{},
		},
	}
}

// lookupResult returns the result with the given name and whether it exists.
func lookupResult(results []CheckResult, name string) (CheckResult, bool) {
	if r := findResult(results, name); r != nil {
		return *r, true
	}
	return CheckResult{}, false
}

func failures(results []CheckResult) []string {
	var names []string
	for _, r := range results {
		if r.Status == StatusFail {
			names = append(names, r.Name)
		}
	}
	return names
}

// TestCheckGeneratedFiles_CICheckout covers F349: on a clean CI checkout the
// gitignored generation state is absent, so the committed manifest is what
// catches an edited or deleted machine-owned file.
func TestCheckGeneratedFiles_CICheckout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mutate       func(t *testing.T, dir string)
		wantFail     string
		wantSeverity CheckSeverity
	}{
		{name: "untouched"},
		{
			name: "hook edited to a no-op",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				writeTestFile(t, dir, manifestHookPath, "import sys\nsys.exit(0)\n")
			},
			wantFail:     "file_unmodified_" + manifestHookPath,
			wantSeverity: SeverityMedium,
		},
		{
			name: "hook deleted",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Remove(filepath.Join(dir, manifestHookPath)); err != nil {
					t.Fatal(err)
				}
			},
			wantFail:     "file_exists_" + manifestHookPath,
			wantSeverity: SeverityHigh,
		},
		{
			name: "human-edited file changed",
			mutate: func(t *testing.T, dir string) {
				t.Helper()
				writeTestFile(t, dir, "CLAUDE.md", "# Project\n\nNotes.\n")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := newManifestProject(t, false, true)
			if tt.mutate != nil {
				tt.mutate(t, p.dir)
			}

			results := checkGeneratedFiles(p.ctx)

			failed := failures(results)
			if tt.wantFail == "" {
				if len(failed) != 0 {
					t.Fatalf("unexpected failures %v: %+v", failed, results)
				}
				if r, ok := lookupResult(results, "generated_files"); !ok || r.Status != StatusPass {
					t.Errorf("want a passing generated_files result, got %+v", results)
				}
				return
			}
			r, ok := lookupResult(results, tt.wantFail)
			if !ok || r.Status != StatusFail || r.Severity != tt.wantSeverity {
				t.Fatalf("want %s to fail at %s, got %+v", tt.wantFail, tt.wantSeverity, results)
			}
			if !ShouldFail(results, AuditLevelMedium) {
				t.Error("ShouldFail(medium) = false, want CI to fail")
			}
		})
	}
}

// TestCheckGeneratedFiles_ManifestMissingOrBroken covers a configured project
// whose manifest is absent or unusable: CI cannot verify generated files, so
// the check fails, and it is auto-fixable only when a local state exists.
func TestCheckGeneratedFiles_ManifestMissingOrBroken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		withState   bool
		manifest    string // "" leaves the manifest absent
		noConfig    bool
		wantResult  bool
		wantFixable bool
	}{
		{name: "CI checkout without manifest", wantResult: true},
		{name: "local project without manifest", withState: true, wantResult: true, wantFixable: true},
		{name: "unparsable manifest", withState: true, manifest: "not a manifest\n", wantResult: true, wantFixable: true},
		{name: "traversal entry", manifest: strings.Repeat("0", 64) + "  ../outside\n", wantResult: true},
		{name: "empty manifest on a CI checkout", manifest: "# truncated\n", wantResult: true},
		{name: "empty manifest with local state", withState: true, manifest: "\n", wantResult: true, wantFixable: true},
		{name: "unconfigured project", noConfig: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := newManifestProject(t, tt.withState, false)
			if tt.manifest != "" {
				writeTestFile(t, p.dir, state.ManifestFile(), tt.manifest)
			}
			if tt.noConfig {
				p.ctx.QsdevConfig = nil
				p.ctx.ConfigErr = fs.ErrNotExist
			}

			results := checkGeneratedFiles(p.ctx)

			r, ok := lookupResult(results, manifestCheckName)
			if ok != tt.wantResult {
				t.Fatalf("%s result present = %v, want %v: %+v", manifestCheckName, ok, tt.wantResult, results)
			}
			if !ok {
				return
			}
			if r.Status != StatusFail || r.Severity != SeverityHigh {
				t.Errorf("status/severity = %s/%s, want fail/high", r.Status, r.Severity)
			}
			if r.AutoFixable != tt.wantFixable {
				t.Errorf("AutoFixable = %v, want %v", r.AutoFixable, tt.wantFixable)
			}
			if !strings.Contains(r.Remediation, state.ManifestFile()) {
				t.Errorf("remediation %q does not name %s", r.Remediation, state.ManifestFile())
			}
		})
	}
}

// TestCheckGeneratedFiles_ConfigParseFailureStillNeedsManifest: a config that
// fails to parse is still a configured project.
func TestCheckGeneratedFiles_ConfigParseFailureStillNeedsManifest(t *testing.T) {
	t.Parallel()
	p := newManifestProject(t, false, false)
	p.ctx.QsdevConfig = nil
	p.ctx.ConfigErr = errors.New("invalid YAML")

	if _, ok := lookupResult(checkGeneratedFiles(p.ctx), manifestCheckName); !ok {
		t.Error("want a generated_manifest failure when the config exists but does not parse")
	}
}

// TestCheckGeneratedFiles_ManifestNewerThanState: after pulling a teammate's
// regeneration the local state is older than the committed files; a file
// matching the committed manifest is not drift.
func TestCheckGeneratedFiles_ManifestNewerThanState(t *testing.T) {
	t.Parallel()
	p := newManifestProject(t, true, false)
	newer := "#!/usr/bin/env python3\nprint('guard v2')\n"
	writeTestFile(t, p.dir, manifestHookPath, newer)
	m := state.BuildManifest(p.st)
	m[manifestHookPath] = state.ComputeHash([]byte(newer))
	if err := state.WriteManifest(p.dir, m); err != nil {
		t.Fatal(err)
	}

	if failed := failures(checkGeneratedFiles(p.ctx)); len(failed) != 0 {
		t.Errorf("unexpected failures %v for a file matching the committed manifest", failed)
	}
}

// TestCheckGeneratedFiles_ManifestCoverage warns when the local state tracks
// a machine-owned file the committed manifest does not list.
func TestCheckGeneratedFiles_ManifestCoverage(t *testing.T) {
	t.Parallel()
	p := newManifestProject(t, true, false)
	const other = ".claude/rules/other.md"
	writeTestFile(t, p.dir, other, "rule\n")
	if err := state.WriteManifest(p.dir, state.Manifest{other: state.ComputeHash([]byte("rule\n"))}); err != nil {
		t.Fatal(err)
	}

	r, ok := lookupResult(checkGeneratedFiles(p.ctx), "generated_manifest_coverage")
	if !ok || r.Status != StatusWarn || !strings.Contains(r.Message, manifestHookPath) {
		t.Errorf("want a coverage warning naming %s, got %+v (found %v)", manifestHookPath, r, ok)
	}
}

// TestApplyAutoFixes_WritesManifestFromState: --auto-fix rebuilds a missing
// manifest from the local state, after which the check passes.
func TestApplyAutoFixes_WritesManifestFromState(t *testing.T) {
	t.Parallel()
	p := newManifestProject(t, true, false)

	fixed := ApplyAutoFixes(checkGeneratedFiles(p.ctx), p.dir, p.state, nil)
	r, ok := lookupResult(fixed, manifestCheckName)
	if !ok || r.Status != StatusPass || !strings.Contains(r.Message, "commit "+state.ManifestFile()) {
		t.Fatalf("manifest result after auto-fix = %+v (found %v)", r, ok)
	}
	m, err := state.LoadManifest(p.ctx.ManifestFile)
	if err != nil {
		t.Fatalf("loading written manifest: %v", err)
	}
	if !maps.Equal(m, state.BuildManifest(p.st)) {
		t.Errorf("manifest = %v, want %v", m, state.BuildManifest(p.st))
	}
	if failed := failures(checkGeneratedFiles(p.ctx)); len(failed) != 0 {
		t.Errorf("failures after auto-fix: %v", failed)
	}
}

// TestApplyAutoFixes_RestoredFileUpdatesManifest: restoring a deleted file
// records the restored content in the manifest too.
func TestApplyAutoFixes_RestoredFileUpdatesManifest(t *testing.T) {
	t.Parallel()
	p := newManifestProject(t, false, true)
	if err := os.Remove(filepath.Join(p.dir, manifestHookPath)); err != nil {
		t.Fatal(err)
	}
	restored := "#!/usr/bin/env python3\nprint('regenerated')\n"
	regen := func(string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{manifestHookPath: {
			Path: manifestHookPath, Content: []byte(restored), Mode: 0o755, Strategy: types.Overwrite,
		}}, nil
	}

	ApplyAutoFixes(checkGeneratedFiles(p.ctx), p.dir, p.state, regen)

	m, err := state.LoadManifest(p.ctx.ManifestFile)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m[manifestHookPath], state.ComputeHash([]byte(restored)); got != want {
		t.Errorf("manifest hash for restored file = %s, want %s", got, want)
	}
	if failed := failures(checkGeneratedFiles(p.ctx)); len(failed) != 0 {
		t.Errorf("failures after restoring: %v", failed)
	}
}
