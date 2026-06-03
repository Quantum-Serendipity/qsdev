package claudecode_test

import (
	"strings"
	"testing"

	claudecode "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateHookFiles_AllEnabled(t *testing.T) {
	t.Parallel()

	t.Run("all consulting hooks enabled", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{
				SafetyBlock:           true,
				CredentialScan:        true,
				DestructivePrevention: true,
				FileBoundary:          true,
				ToolGates:             true,
				SOC2Audit:             true,
			},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 6 {
			t.Errorf("expected 6 files, got %d", len(files))
		}
		for _, f := range files {
			if f.Mode != 0o755 {
				t.Errorf("%s: mode = %o, want %o", f.Path, f.Mode, 0o755)
			}
			if len(f.Content) == 0 {
				t.Errorf("%s: content is empty", f.Path)
			}
		}
	})

	t.Run("audit-log without soc2", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{
				AuditLog: true,
			},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		if files[0].Path != ".claude/hooks/audit-log.sh" {
			t.Errorf("path = %q, want audit-log.sh", files[0].Path)
		}
	})

	t.Run("soc2 suppresses audit-log", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{
				AuditLog:  true,
				SOC2Audit: true,
			},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, f := range files {
			if f.Path == ".claude/hooks/audit-log.sh" {
				t.Error("audit-log.sh should be suppressed when SOC2Audit is enabled")
			}
		}
	})

	t.Run("none enabled", func(t *testing.T) {
		t.Parallel()
		files, err := claudecode.GenerateHookFiles(types.WizardAnswers{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("package-guard content integrity", func(t *testing.T) {
		t.Parallel()
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{SafetyBlock: true},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
		content := string(files[0].Content)
		checks := []string{
			"#!/usr/bin/env python3",
			"osv.dev",
			"PreToolUse",
			"FAIL_CLOSED = True",
		}
		for _, c := range checks {
			if !strings.Contains(content, c) {
				t.Errorf("content does not contain %q", c)
			}
		}
	})
}
