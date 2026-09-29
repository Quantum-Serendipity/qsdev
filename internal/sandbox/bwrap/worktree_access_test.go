//go:build !windows

package bwrap

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestBuildArgs_WorktreeAccessOverride pins that the policy's worktreeAccess,
// carried in SandboxConfig.WorktreeAccess, decides how the project directory
// is bound, overriding the hook category's default in both directions.
func TestBuildArgs_WorktreeAccessOverride(t *testing.T) {
	t.Parallel()

	const project = "/home/user/project"
	tests := []struct {
		name         string
		category     sandbox.HookCategory
		access       string
		wantReadOnly bool
	}{
		{name: "test-runner restricted to read-only", category: sandbox.CategoryTestRunner, access: sandbox.WorktreeAccessReadOnly, wantReadOnly: true},
		{name: "linter widened to read-write", category: sandbox.CategoryLinter, access: sandbox.WorktreeAccessReadWrite, wantReadOnly: false},
		{name: "unset keeps test-runner default", category: sandbox.CategoryTestRunner, access: "", wantReadOnly: false},
		{name: "malformed value fails closed", category: sandbox.CategoryFormatter, access: "yes", wantReadOnly: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := sandbox.SandboxConfig{
				ProjectDir:     project,
				HookCategory:   tt.category,
				WorktreeAccess: tt.access,
				Network:        sandbox.NetworkPolicy{Mode: "deny"},
			}

			args, err := BuildArgs(&cfg, sandbox.TierFull)
			if err != nil {
				t.Fatalf("BuildArgs: %v", err)
			}

			gotRO := containsSequence(args, []string{"--ro-bind", project, project})
			gotRW := containsBindRW(args, project)
			if gotRO != tt.wantReadOnly || gotRW == tt.wantReadOnly {
				t.Errorf("project bound ro=%v rw=%v, want read-only=%v: %v", gotRO, gotRW, tt.wantReadOnly, args)
			}
		})
	}
}
