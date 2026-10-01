package claudecode_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestSelectableHookPresets_SingleSource(t *testing.T) {
	t.Parallel()
	selectable := claudecode.SelectableHookPresets()
	vocabulary := catalog.MustDefault().HookPresets()

	t.Run("subset of the catalog vocabulary", func(t *testing.T) {
		t.Parallel()
		for _, name := range selectable {
			if !slices.Contains(vocabulary, name) {
				t.Errorf("selectable preset %q is not a catalog hook preset", name)
			}
		}
	})
	t.Run("excludes presets no hook implements", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"auto-format", "pre-commit"} {
			if slices.Contains(selectable, name) {
				t.Errorf("SelectableHookPresets() = %v, must exclude %q", selectable, name)
			}
		}
		for _, name := range []string{"safety-block", "audit-log", "credential-scan"} {
			if !slices.Contains(selectable, name) {
				t.Errorf("SelectableHookPresets() = %v, must include %q", selectable, name)
			}
		}
	})
	t.Run("add-hook completion", func(t *testing.T) {
		t.Parallel()
		cmd, _, err := claudecode.ExportClaudeCmd().Find([]string{"add-hook"})
		if err != nil {
			t.Fatalf("finding add-hook: %v", err)
		}
		got, _ := cmd.ValidArgsFunction(cmd, nil, "")
		if !slices.Equal(got, selectable) {
			t.Errorf("add-hook completions = %v, want %v", got, selectable)
		}
	})
	t.Run("validate", func(t *testing.T) {
		t.Parallel()
		for _, name := range vocabulary {
			err := claudecode.ValidateHookPreset(name)
			if want := slices.Contains(selectable, name); want != (err == nil) {
				t.Errorf("ValidateHookPreset(%q) = %v, selectable = %v", name, err, want)
			}
			if err != nil && !errors.Is(err, claudecode.ErrUnselectableHookPreset) {
				t.Errorf("ValidateHookPreset(%q) error %v does not wrap ErrUnselectableHookPreset", name, err)
			}
		}
	})
}

func TestCatalogProfiles_HooksAreSelectable(t *testing.T) {
	t.Parallel()
	selectable := claudecode.SelectableHookPresets()
	for name, profile := range catalog.MustDefault().ProjectProfiles() {
		for _, hook := range profile.Hooks {
			if !slices.Contains(selectable, hook) {
				t.Errorf("profile %q lists hook %q, which is not selectable (%v)", name, hook, selectable)
			}
		}
	}
}

// TestValidateHookPreset_NoDeadGitHooksPointer verifies a catalog preset
// that registers no Claude Code hook names the compliance level's devenv git
// hooks instead of the unwired --git-hooks flag, while a misspelled one gets
// only the list of valid presets.
func TestValidateHookPreset_NoDeadGitHooksPointer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		wantHint bool
	}{
		{name: "pre-commit", wantHint: true},
		{name: "auto-format", wantHint: true},
		{name: "not-a-preset", wantHint: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := claudecode.ValidateHookPreset(tt.name)
			if err == nil {
				t.Fatalf("ValidateHookPreset(%q) = nil, want an error", tt.name)
			}
			if !errors.Is(err, claudecode.ErrUnselectableHookPreset) {
				t.Errorf("ValidateHookPreset(%q) = %v, want it to wrap ErrUnselectableHookPreset", tt.name, err)
			}
			if strings.Contains(err.Error(), "--git-hooks") {
				t.Errorf("error %q points at the unwired --git-hooks flag", err)
			}
			if got := strings.Contains(err.Error(), "required_pre_commit_hooks"); got != tt.wantHint {
				t.Errorf("error %q names required_pre_commit_hooks = %v, want %v", err, got, tt.wantHint)
			}
			if want := `unknown or unimplemented hook preset "` + tt.name + `"`; !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		})
	}
}

// TestValidateHookChoices verifies hook flags set directly (as an answers
// file does) are held to the selectable presets, while internal flags that
// are not catalog presets pass.
func TestValidateHookChoices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		hooks   types.HookChoices
		wantErr []string
	}{
		{name: "selectable presets", hooks: types.HookChoices{SafetyBlock: true, AuditLog: true, CredentialScan: true}},
		{name: "internal flag", hooks: types.HookChoices{SelfProtection: true}},
		{name: "auto-format", hooks: types.HookChoices{SafetyBlock: true, AutoFormat: true}, wantErr: []string{`"auto-format"`}},
		{name: "pre-commit", hooks: types.HookChoices{PreCommit: true}, wantErr: []string{`"pre-commit"`}},
		{name: "both", hooks: types.HookChoices{AutoFormat: true, PreCommit: true}, wantErr: []string{`"auto-format"`, `"pre-commit"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := claudecode.ValidateHookChoices(tt.hooks)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("ValidateHookChoices() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, claudecode.ErrUnselectableHookPreset) {
				t.Fatalf("ValidateHookChoices() = %v, want ErrUnselectableHookPreset", err)
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not name %s", err, want)
				}
			}
		})
	}
}
