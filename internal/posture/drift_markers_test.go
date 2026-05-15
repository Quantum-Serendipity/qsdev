package posture

import (
	"path/filepath"
	"testing"
)

func TestDetectMarkerIntegrity_AllIntact(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"# CLAUDE.md\n"+
			"<!-- qsdev:semgrep -->\nSemgrep rules\n<!-- /qsdev:semgrep -->\n"+
			"<!-- qsdev:gitleaks -->\nGitleaks config\n<!-- /qsdev:gitleaks -->\n")

	enabledTools := map[string]bool{
		"semgrep":  true,
		"gitleaks": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when all markers intact, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectMarkerIntegrity_MissingClosingMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"# CLAUDE.md\n<!-- qsdev:semgrep -->\nSemgrep rules\n")

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	foundUnpaired := false
	for _, f := range cat.Findings {
		if f.Subject == "marker:semgrep" && f.Severity == DriftWarning {
			foundUnpaired = true
			break
		}
	}
	if !foundUnpaired {
		t.Errorf("expected warning about missing closing marker for semgrep, findings: %+v", cat.Findings)
	}
}

func TestDetectMarkerIntegrity_MissingOpeningMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"# CLAUDE.md\nSemgrep rules\n<!-- /qsdev:semgrep -->\n")

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	foundUnpaired := false
	for _, f := range cat.Findings {
		if f.Subject == "marker:semgrep" && f.Severity == DriftWarning {
			foundUnpaired = true
			break
		}
	}
	if !foundUnpaired {
		t.Errorf("expected warning about missing opening marker for semgrep, findings: %+v", cat.Findings)
	}
}

func TestDetectMarkerIntegrity_EntirelyMissingPair(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# CLAUDE.md\nNo markers here.\n")

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	foundMissing := false
	for _, f := range cat.Findings {
		if f.Subject == "marker:semgrep" && f.Severity == DriftWarning {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("expected warning about entirely missing marker pair, findings: %+v", cat.Findings)
	}
}

func TestDetectMarkerIntegrity_ClaudeMDNotFound(t *testing.T) {
	dir := t.TempDir()
	// No CLAUDE.md file.

	enabledTools := map[string]bool{
		"semgrep": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for missing CLAUDE.md, got %d: %+v", len(cat.Findings), cat.Findings)
	}

	f := cat.Findings[0]
	if f.Severity != DriftError {
		t.Errorf("expected severity %q, got %q", DriftError, f.Severity)
	}
	if f.Subject != "CLAUDE.md" {
		t.Errorf("expected subject %q, got %q", "CLAUDE.md", f.Subject)
	}
}

func TestDetectMarkerIntegrity_NoEnabledTools(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# CLAUDE.md\n")

	cat := detectMarkerIntegrity(dir, nil)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings with no enabled tools, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectMarkerIntegrity_MultiplePairs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"),
		"# CLAUDE.md\n"+
			"<!-- qsdev:semgrep -->\nSemgrep\n<!-- /qsdev:semgrep -->\n"+
			"<!-- qsdev:gitleaks -->\nGitleaks\n<!-- /qsdev:gitleaks -->\n"+
			"<!-- qsdev:attach-guard -->\nGuard\n<!-- /qsdev:attach-guard -->\n")

	enabledTools := map[string]bool{
		"semgrep":      true,
		"gitleaks":     true,
		"attach-guard": true,
	}

	cat := detectMarkerIntegrity(dir, enabledTools)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when all markers present, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}
