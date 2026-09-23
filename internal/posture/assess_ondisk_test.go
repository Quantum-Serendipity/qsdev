package posture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCIRepository pins that reports produced in GitHub Actions carry their
// source repository (used by team-report --create-issues) and that malformed
// or out-of-CI values are ignored.
func TestCIRepository(t *testing.T) {
	tests := []struct {
		name, actions, repo, want string
	}{
		{name: "github actions", actions: "true", repo: "acme/web-app", want: "acme/web-app"},
		{name: "outside github actions", actions: "", repo: "acme/web-app", want: ""},
		{name: "malformed slug", actions: "true", repo: "acme/web app; rm", want: ""},
		{name: "missing name", actions: "true", repo: "acme/", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_ACTIONS", tt.actions)
			t.Setenv("GITHUB_REPOSITORY", tt.repo)
			if got := ciRepository(); got != tt.want {
				t.Errorf("ciRepository() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAssess_StateListedFilesMustExistOnDisk is the F322 regression test: the
// state file still lists security files that were deleted from the working
// tree. Defense layers and baseline conformance must judge the files on disk,
// so the deletion fails baseline instead of reporting PASS and 100% defense.
func TestAssess_StateListedFilesMustExistOnDisk(t *testing.T) {
	t.Parallel()

	tracked := []string{
		"CLAUDE.md",
		".claude/settings.json",
		".claude/hooks/package-guard.py",
		".pre-commit-config.yaml",
	}

	tests := []struct {
		name        string
		writeFiles  bool
		wantPresent bool
	}{
		{name: "files deleted after generation", writeFiles: false, wantPresent: false},
		{name: "files present", writeFiles: true, wantPresent: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			st := types.GeneratedState{
				QsdevVersion: "1.0.0",
				Files:        map[string]types.FileState{},
				EnabledTools: map[string]bool{"attach-guard": true, "gitleaks": true, "socket-dev-mcp": true},
			}
			for _, rel := range tracked {
				st.Files[rel] = types.FileState{Hash: "stale"}
				if tt.writeFiles {
					abs := filepath.Join(root, filepath.FromSlash(rel))
					if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(abs, []byte("x\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			writeState(t, root, filepath.Join(".devinit", ".qsdev-init-state.yaml"), st)

			report, err := Assess(root, AssessOptions{})
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}

			for _, name := range []CheckName{CheckClaudeMDPresent, CheckSettingsJSONPresent, CheckPreCommitHooks} {
				var found bool
				for _, c := range report.Conformance.Baseline.Checks {
					if c.Name != name {
						continue
					}
					found = true
					if c.Pass != tt.wantPresent {
						t.Errorf("baseline %s pass = %v (%s), want %v", name, c.Pass, c.Reason, tt.wantPresent)
					}
				}
				if !found {
					t.Errorf("baseline check %s missing", name)
				}
			}

			pretool := FindLayerByName(report.Defense.Layers, "pretooluse-hooks")
			if pretool == nil {
				t.Fatal("pretooluse-hooks layer missing")
			}
			if gotEnabled := pretool.Status == LayerEnabled; gotEnabled != tt.wantPresent {
				t.Errorf("pretooluse-hooks status = %q (%s), want enabled=%v", pretool.Status, pretool.Reason, tt.wantPresent)
			}

			if !tt.wantPresent {
				if report.Conformance.Baseline.Pass {
					t.Error("baseline PASS although the tracked security files are gone")
				}
				if !ShouldExitNonZero(report, "high") {
					t.Error("default audit level exits 0 although the tracked security files are gone")
				}
			}
		})
	}
}
