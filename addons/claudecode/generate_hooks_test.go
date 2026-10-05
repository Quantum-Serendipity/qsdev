package claudecode_test

import (
	"bytes"
	"os"
	"slices"
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
			// Isolate the consulting hooks from the always-on lsp-guard.
			LSP: types.LSPSettings{Enforcement: "off"},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Six hooks plus the shared Python hook library they load.
		if len(files) != 7 {
			t.Errorf("expected 7 files, got %d", len(files))
		}
		for _, f := range files {
			wantMode := os.FileMode(0o755)
			if f.Path == claudecode.HookLibPath {
				wantMode = 0o644 // loaded by path, never executed
			}
			if f.Mode != wantMode {
				t.Errorf("%s: mode = %o, want %o", f.Path, f.Mode, wantMode)
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
				AuditLog:          true,
				SafetyBlockOptOut: true,
			},
			LSP: types.LSPSettings{Enforcement: "off"},
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
		// Opt out of the safety block and disable LSP too so no hooks at all
		// are generated; package-guard and the lsp-guard are otherwise on by
		// default (enforcement defaults to "block").
		answers := types.WizardAnswers{
			Hooks: types.HookChoices{SafetyBlockOptOut: true},
			LSP:   types.LSPSettings{Enforcement: "off"},
		}
		files, err := claudecode.GenerateHookFiles(answers)
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
			LSP:   types.LSPSettings{Enforcement: "off"},
		}
		files, err := claudecode.GenerateHookFiles(answers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 || files[0].Path != claudecode.PackageGuardPath || files[1].Path != claudecode.HookLibPath {
			t.Fatalf("files = %v, want the package guard and the hook library", files)
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

// TestPackageGuardContent pins that the content posture judges the guard
// against is exactly what the generator writes for it.
func TestPackageGuardContent(t *testing.T) {
	t.Parallel()
	files, err := claudecode.GenerateHookFiles(types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true}})
	if err != nil {
		t.Fatal(err)
	}
	want := claudecode.PackageGuardContent()
	if len(want) == 0 {
		t.Fatal("PackageGuardContent is empty")
	}
	for _, f := range files {
		if f.Path == claudecode.PackageGuardPath {
			if !bytes.Equal(f.Content, want) {
				t.Error("generated package-guard.py differs from PackageGuardContent")
			}
			return
		}
	}
	t.Fatalf("no %s generated", claudecode.PackageGuardPath)
}

// TestHookScriptContents pins that every hook script the generator writes,
// whichever answers enable it, is listed with exactly the content written.
func TestHookScriptContents(t *testing.T) {
	t.Parallel()
	all := types.HookChoices{
		SafetyBlock: true, AuditLog: true, CredentialScan: true, DestructivePrevention: true,
		FileBoundary: true, ToolGates: true,
	}
	soc2 := all
	soc2.SOC2Audit = true
	contents := claudecode.HookScriptContents()
	if len(contents[claudecode.PackageGuardPath]) == 0 {
		t.Fatalf("HookScriptContents lacks %s", claudecode.PackageGuardPath)
	}
	for _, hooks := range []types.HookChoices{all, soc2} {
		files, err := claudecode.GenerateHookFiles(types.WizardAnswers{Tier: "full", Hooks: hooks, AgentTools: types.AgentToolsAnswers{SembleEnabled: true}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			want, ok := contents[f.Path]
			if !ok {
				t.Errorf("HookScriptContents lacks generated %s", f.Path)
				continue
			}
			if !bytes.Equal(f.Content, want) {
				t.Errorf("generated %s differs from HookScriptContents", f.Path)
			}
		}
	}
}

// TestGenerateHookFiles_EmitsHookLib pins that the shared Python hook library
// is written, with its template's exact content, exactly when a hook that
// loads it is: such a hook without it blocks every call, and a library
// nothing loads is clutter.
func TestGenerateHookFiles_EmitsHookLib(t *testing.T) {
	t.Parallel()
	template, err := os.ReadFile(hookLibTemplate)
	if err != nil {
		t.Fatal(err)
	}
	noLSP := types.LSPSettings{Enforcement: "off"}
	tests := []struct {
		name    string
		answers types.WizardAnswers
		want    bool
	}{
		{"package guard only", types.WizardAnswers{Hooks: types.HookChoices{SafetyBlock: true}, LSP: noLSP}, true},
		{"every hook", types.WizardAnswers{Hooks: types.HookChoices{
			SafetyBlock: true, CredentialScan: true, DestructivePrevention: true,
			FileBoundary: true, ToolGates: true, SOC2Audit: true,
		}}, true},
		// soc2-audit-log.py writes its own trail and does not load the library.
		{"soc2 audit only", types.WizardAnswers{Hooks: types.HookChoices{SafetyBlockOptOut: true, SOC2Audit: true}, LSP: noLSP}, false},
		{"shell hooks only", types.WizardAnswers{Hooks: types.HookChoices{SafetyBlockOptOut: true, AuditLog: true}}, false},
		{"no hooks", types.WizardAnswers{Hooks: types.HookChoices{SafetyBlockOptOut: true}, LSP: noLSP}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			files, err := claudecode.GenerateHookFiles(tc.answers)
			if err != nil {
				t.Fatal(err)
			}
			i := slices.IndexFunc(files, func(f types.GeneratedFile) bool { return f.Path == claudecode.HookLibPath })
			if got := i >= 0; got != tc.want {
				t.Fatalf("hook library emitted = %v, want %v (files %v)", got, tc.want, files)
			}
			if i < 0 {
				return
			}
			if lib := files[i]; !bytes.Equal(lib.Content, template) || lib.Mode != 0o644 || lib.Strategy != types.Overwrite {
				t.Errorf("hook library = mode %o strategy %v, content equal %v; want 644, Overwrite, the template",
					lib.Mode, lib.Strategy, bytes.Equal(lib.Content, template))
			}
		})
	}
}

// TestHookSupportPaths pins which hooks depend on the shared library, derived
// from whether their template loads it, so whatever writes a hook (init,
// update, enable) can write the library with it.
func TestHookSupportPaths(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want []string
	}{
		{claudecode.PackageGuardPath, []string{claudecode.HookLibPath}},
		{".claude/hooks/block-destructive.py", []string{claudecode.HookLibPath}},
		{".claude/hooks/scan-secrets.py", []string{claudecode.HookLibPath}},
		{".claude/hooks/file-boundary.py", []string{claudecode.HookLibPath}},
		{".claude/hooks/tool-gates.py", []string{claudecode.HookLibPath}},
		{".claude/hooks/soc2-audit-log.py", nil},
		{".claude/hooks/audit-log.sh", nil},
		{claudecode.HookLibPath, nil},
		{"CLAUDE.md", nil},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			if got := claudecode.HookSupportPaths(tc.path); !slices.Equal(got, tc.want) {
				t.Errorf("HookSupportPaths(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
