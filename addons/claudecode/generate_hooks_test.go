package claudecode_test

import (
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGeneratePackageGuardHook_Enabled(t *testing.T) {
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true},
	}
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile when SafetyBlock is enabled")
	}
	if got.Path != ".claude/hooks/package-guard.py" {
		t.Errorf("Path = %q, want %q", got.Path, ".claude/hooks/package-guard.py")
	}
	if got.Mode != 0o755 {
		t.Errorf("Mode = %o, want %o", got.Mode, 0o755)
	}
	if got.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", got.Strategy)
	}
	if len(got.Content) == 0 {
		t.Error("Content is empty")
	}
}

func TestGeneratePackageGuardHook_Disabled(t *testing.T) {
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: false},
	}
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil when SafetyBlock is disabled, got %+v", got)
	}
}

func TestGeneratePackageGuardHook_ZeroValue(t *testing.T) {
	var answers types.WizardAnswers
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for zero-value WizardAnswers, got %+v", got)
	}
}

func TestGeneratePackageGuardHook_ContentIntegrity(t *testing.T) {
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true},
	}
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(got.Content)

	checks := []struct {
		needle string
		desc   string
	}{
		{"#!/usr/bin/env python3", "shebang line"},
		{"osv.dev", "OSV.dev vulnerability database reference"},
		{"PreToolUse", "PreToolUse hook event name"},
		{"updatedInput", "updatedInput for command rewriting"},
		{"PACKAGE_GUARD_FAIL_CLOSED", "FAIL_CLOSED env var configuration"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.needle) {
			t.Errorf("content does not contain %q (%s)", c.needle, c.desc)
		}
	}
}

func TestGeneratePackageGuardHook_NpmAgeFix(t *testing.T) {
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true},
	}
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(got.Content)

	// The fixed npm age check must use dist-tags to find the latest version.
	if !strings.Contains(content, "dist-tags") {
		t.Error("content does not contain 'dist-tags' (npm age fix missing)")
	}
	if !strings.Contains(content, "dist_tags") {
		t.Error("content does not contain 'dist_tags' variable (npm age fix missing)")
	}

	// The old pattern used time.modified without dist-tags lookup. Verify
	// the fix replaced it: the function should NOT contain the old pattern
	// of reading "modified" from time_map as the age source.
	// We check that "modified" does not appear between check_npm_age and
	// check_pypi_age, which would indicate the old unfixed logic.
	npmStart := strings.Index(content, "def check_npm_age")
	pypiStart := strings.Index(content, "def check_pypi_age")
	if npmStart < 0 {
		t.Fatal("could not find check_npm_age function")
	}
	if pypiStart < 0 {
		t.Fatal("could not find check_pypi_age function")
	}
	npmFunc := content[npmStart:pypiStart]
	if strings.Contains(npmFunc, `time_map.get("modified")`) {
		t.Error("check_npm_age still contains old pattern using time_map.get(\"modified\")")
	}
}

func TestGeneratePackageGuardHook_StdlibOnly(t *testing.T) {
	answers := types.WizardAnswers{
		Hooks: types.HookChoices{SafetyBlock: true},
	}
	got, err := claudecode.GeneratePackageGuardHook(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(got.Content)

	forbidden := []struct {
		needle string
		desc   string
	}{
		{"import requests", "third-party requests library"},
		{"import pip", "pip module import"},
		{"from setuptools", "setuptools dependency"},
	}
	for _, f := range forbidden {
		if strings.Contains(content, f.needle) {
			t.Errorf("content contains %q (%s) — hook must use only stdlib", f.needle, f.desc)
		}
	}
}
