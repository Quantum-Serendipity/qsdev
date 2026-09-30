package generate_test

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

var errDiskFull = errors.New("disk full")

func TestWriteResultErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   generate.WriteResult
		wantNil  bool
		wantIs   []error
		wantText string
	}{
		{
			name: "none failed",
			result: generate.WriteResult{Created: 1, Kept: 1, Files: []generate.FileResult{
				{Path: "a.txt", Action: generate.ActionCreated},
				{Path: ".envrc", Action: generate.ActionKept},
			}},
			wantNil: true,
		},
		{
			name: "two failed",
			result: generate.WriteResult{Created: 1, Failed: 2, Files: []generate.FileResult{
				{Path: "a.txt", Action: generate.ActionCreated},
				{Path: ".claude/settings.json", Action: generate.ActionFailed, Error: fs.ErrPermission},
				{Path: "CLAUDE.md", Action: generate.ActionFailed, Error: errDiskFull},
			}},
			wantIs:   []error{fs.ErrPermission, errDiskFull},
			wantText: "  - .claude/settings.json: permission denied\n  - CLAUDE.md: disk full",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.result.Err()
			if tt.wantNil {
				if err != nil {
					t.Fatalf("Err() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Err() = nil, want an error")
			}
			for _, target := range tt.wantIs {
				if !errors.Is(err, target) {
					t.Errorf("errors.Is(Err(), %v) = false", target)
				}
			}
			if got := err.Error(); got != tt.wantText {
				t.Errorf("Err() = %q, want %q", got, tt.wantText)
			}
		})
	}
}

func TestSummaryListsFailedAndKept(t *testing.T) {
	t.Parallel()

	r := generate.WriteResult{
		Created: 2, Updated: 1, Unchanged: 3, Kept: 1, Sidecar: 1, Failed: 1,
		Files: []generate.FileResult{
			{Path: ".claude/settings.json", Action: generate.ActionFailed, Error: errDiskFull},
			{Path: ".envrc", Action: generate.ActionKept},
			{Path: "devenv.nix", Action: generate.ActionSidecar, SidecarPath: "devenv.nix.new"},
			{Path: "CLAUDE.md", Action: generate.ActionUpdated, Note: "overwritten (--force)"},
		},
	}
	got := r.Summary()
	lines := strings.Split(got, "\n")
	if want := "Created 2, updated 1, unchanged 3, kept 1, sidecar 1, failed 1"; lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
	for _, want := range []string{
		"  FAILED .claude/settings.json: disk full",
		"  kept existing .envrc (never overwritten); delete it to use the generated version",
		"  devenv.nix has local changes; merge devenv.nix.new into it manually",
		"  CLAUDE.md: overwritten (--force)",
	} {
		if !strings.Contains(got, want+"\n") && !strings.HasSuffix(got, want) {
			t.Errorf("Summary() missing line %q:\n%s", want, got)
		}
	}
}

func TestSuccessfulFilesExcludesKeptAndSidecar(t *testing.T) {
	t.Parallel()

	all := []types.GeneratedFile{
		{Path: "created"}, {Path: "updated"}, {Path: "unchanged"},
		{Path: "kept"}, {Path: "sidecar"}, {Path: "failed"},
	}
	r := generate.WriteResult{Files: []generate.FileResult{
		{Path: "created", Action: generate.ActionCreated},
		{Path: "updated", Action: generate.ActionUpdated},
		{Path: "unchanged", Action: generate.ActionUnchanged},
		{Path: "kept", Action: generate.ActionKept},
		{Path: "sidecar", Action: generate.ActionSidecar, SidecarPath: "sidecar.new"},
		{Path: "failed", Action: generate.ActionFailed, Error: errDiskFull},
	}}
	var got []string
	for _, f := range r.SuccessfulFiles(all) {
		got = append(got, f.Path)
	}
	if want := "created,updated,unchanged"; strings.Join(got, ",") != want {
		t.Errorf("SuccessfulFiles = %v, want %s", got, want)
	}
}

func TestPartialWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  generate.WriteResult
		wantNil bool
	}{
		{name: "no failures", result: generate.WriteResult{Created: 1}, wantNil: true},
		{
			name: "one failure",
			result: generate.WriteResult{Created: 3, Failed: 1, Files: []generate.FileResult{
				{Path: ".claude/settings.json", Action: generate.ActionFailed, Error: fs.ErrPermission},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := generate.PartialWriteError(tt.result, 3, "'qsdev claude init --force'")
			if tt.wantNil {
				if err != nil {
					t.Fatalf("PartialWriteError = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("PartialWriteError = nil, want an error")
			}
			if !errors.Is(err, fs.ErrPermission) {
				t.Errorf("errors.Is(err, fs.ErrPermission) = false for %v", err)
			}
			for _, want := range []string{
				"partial write: 1 files failed (state saved for 3 successful files)",
				"re-run 'qsdev claude init --force' to finish setup:\n  - .claude/settings.json: permission denied",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err, want)
				}
			}
		})
	}
}
