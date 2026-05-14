package posture

import (
	"errors"
	"testing"
)

func TestDetectToolAvailability_AllFound(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	defer func() { lookPath = origLookPath }()

	enabledTools := map[string]bool{
		"semgrep":  true,
		"gitleaks": true,
	}

	cat := detectToolAvailability(enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when all tools found, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectToolAvailability_OneMissing(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		if file == "semgrep" {
			return "", errors.New("not found")
		}
		return "/usr/bin/" + file, nil
	}
	defer func() { lookPath = origLookPath }()

	enabledTools := map[string]bool{
		"semgrep":  true,
		"gitleaks": true,
	}

	cat := detectToolAvailability(enabledTools)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for missing tool, got %d: %+v", len(cat.Findings), cat.Findings)
	}

	f := cat.Findings[0]
	if f.Severity != DriftWarning {
		t.Errorf("expected severity %q, got %q", DriftWarning, f.Severity)
	}
	if f.Subject != "semgrep" {
		t.Errorf("expected subject %q, got %q", "semgrep", f.Subject)
	}
	if f.Expected != "semgrep" {
		t.Errorf("expected Expected=%q, got %q", "semgrep", f.Expected)
	}
}

func TestDetectToolAvailability_NonBinaryToolsSkipped(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", errors.New("should not be called for MCP tools")
	}
	defer func() { lookPath = origLookPath }()

	// MCP servers and skills don't have binary requirements.
	enabledTools := map[string]bool{
		"context7":    true,
		"github-mcp":  true,
		"socket-dev-mcp": true,
		"agent-postmortem": true,
		"trail-of-bits-skills": true,
	}

	cat := detectToolAvailability(enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings for non-binary tools, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectToolAvailability_DisabledToolSkipped(t *testing.T) {
	origLookPath := lookPath
	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}
	defer func() { lookPath = origLookPath }()

	enabledTools := map[string]bool{
		"semgrep": false,
	}

	cat := detectToolAvailability(enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings for disabled tools, got %d", len(cat.Findings))
	}
}

func TestDetectToolAvailability_NilMap(t *testing.T) {
	cat := detectToolAvailability(nil)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings for nil map, got %d", len(cat.Findings))
	}
}
