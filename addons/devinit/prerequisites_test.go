package devinit

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrerequisiteResult_HasMissing_AllFound(t *testing.T) {
	result := PrerequisiteResult{
		Tools: []PrerequisiteStatus{
			{Name: "git", Found: true, Required: true},
			{Name: "nix", Found: true, Required: true},
		},
	}
	if result.HasMissing() {
		t.Error("HasMissing() = true, want false when all required tools found")
	}
}

func TestPrerequisiteResult_HasMissing_OneMissing(t *testing.T) {
	result := PrerequisiteResult{
		Tools: []PrerequisiteStatus{
			{Name: "git", Found: true, Required: true},
			{Name: "nix", Found: false, Required: true, InstallHint: "Install Nix"},
		},
	}
	if !result.HasMissing() {
		t.Error("HasMissing() = false, want true when required tool is missing")
	}
}

func TestPrerequisiteResult_HasMissing_OptionalMissing(t *testing.T) {
	result := PrerequisiteResult{
		Tools: []PrerequisiteStatus{
			{Name: "git", Found: true, Required: true},
			{Name: "optional-tool", Found: false, Required: false},
		},
	}
	if result.HasMissing() {
		t.Error("HasMissing() = true, want false when only optional tools are missing")
	}
}

func TestPrerequisiteResult_HasMissing_Empty(t *testing.T) {
	result := PrerequisiteResult{}
	if result.HasMissing() {
		t.Error("HasMissing() = true, want false for empty result")
	}
}

func TestPrerequisiteResult_PrintReport(t *testing.T) {
	result := PrerequisiteResult{
		Tools: []PrerequisiteStatus{
			{Name: "git", Found: true, Required: true, Version: "git version 2.43.0", Path: "/usr/bin/git"},
			{Name: "nix", Found: false, Required: true, InstallHint: "Install Nix: https://nixos.org"},
		},
	}

	var buf bytes.Buffer
	result.PrintReport(&buf)
	output := buf.String()

	if !strings.Contains(output, "git") {
		t.Error("PrintReport should mention git")
	}
	if !strings.Contains(output, "OK") {
		t.Error("PrintReport should show OK for found tools")
	}
	if !strings.Contains(output, "MISSING") {
		t.Error("PrintReport should show MISSING for missing required tools")
	}
	if !strings.Contains(output, "Install Nix") {
		t.Error("PrintReport should show install hint for missing tools")
	}
}

func TestPrerequisiteResult_PrintReport_WithVersion(t *testing.T) {
	result := PrerequisiteResult{
		Tools: []PrerequisiteStatus{
			{Name: "git", Found: true, Required: true, Version: "git version 2.43.0"},
		},
	}

	var buf bytes.Buffer
	result.PrintReport(&buf)
	output := buf.String()

	if !strings.Contains(output, "git version 2.43.0") {
		t.Errorf("PrintReport should include version string, got: %s", output)
	}
}
