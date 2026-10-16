package drift

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestExpectedMarkersRegistryError: when the tool registry cannot be built
// the expected markers are unknown, so the error is returned rather than an
// empty set that would report every expected marker as present.
func TestExpectedMarkersRegistryError(t *testing.T) {
	t.Parallel()
	errBuild := errors.New("registry build failed")
	got, err := expectedClaudeMDMarkers(func() (*toolreg.Registry, error) { return nil, errBuild },
		map[string]bool{"semgrep": true})
	if !errors.Is(err, errBuild) {
		t.Fatalf("expectedClaudeMDMarkers() error = %v, want %v", err, errBuild)
	}
	if got != nil {
		t.Errorf("expectedClaudeMDMarkers() = %v, want nil on error", got)
	}
}

// TestDetectBrokenCatalogNoPanic: with a project defaults file the catalog
// rejects, drift detection (MCP status tier 2) reports the unknown expected
// markers as one Warning finding instead of panicking in the tool registry.
// The catalog and tool registry are process globals, so this test does not
// run in parallel.
func TestDetectBrokenCatalogNoPanic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "CLAUDE.md"), "# CLAUDE.md\n<!-- qsdev:semgrep -->\nx\n<!-- /qsdev:semgrep -->\n")
	writeFile(t, catalog.ProjectConfigPath(dir), "keep_vars: [AWS_SECRET_ACCESS_KEY]\n")
	catalog.ResetDefault()
	toolreg.ResetDefaultRegistry()
	t.Cleanup(func() {
		catalog.ResetDefault()
		toolreg.ResetDefaultRegistry()
	})
	if err := catalog.SetProjectRoot(dir); err != nil {
		t.Fatal(err)
	}

	report := Detect(dir, types.GeneratedState{}, map[string]bool{"semgrep": true})

	var markers *Category
	for i := range report.Categories {
		if report.Categories[i].Name == categoryMarkerIntegrity {
			markers = &report.Categories[i]
		}
	}
	if markers == nil {
		t.Fatalf("no %q category in report", categoryMarkerIntegrity)
	}
	if len(markers.Findings) != 1 {
		t.Fatalf("marker findings = %+v, want one warning", markers.Findings)
	}
	f := markers.Findings[0]
	if f.Severity != Warning || !strings.Contains(f.Description, "Unable to determine the expected section markers") {
		t.Errorf("finding = %+v, want a Warning naming the unknown expected markers", f)
	}
	if !strings.Contains(f.Description, "may only add or tighten") {
		t.Errorf("finding description %q does not carry the catalog error", f.Description)
	}
}

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
		if f.Subject == "marker:semgrep" && f.Severity == Warning {
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
		if f.Subject == "marker:semgrep" && f.Severity == Warning {
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
		if f.Subject == "marker:semgrep" && f.Severity == Warning {
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
	if f.Severity != Error {
		t.Errorf("expected severity %q, got %q", Error, f.Severity)
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
